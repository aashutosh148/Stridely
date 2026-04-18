'use client';

import { useCallback, useEffect, useMemo, useState } from 'react';

const API_BASE_URL = process.env.NEXT_PUBLIC_API_URL || 'http://localhost:3001/api/v1';

export interface ChatMessage {
  id: string;
  role: 'user' | 'assistant';
  content: string;
  created_at?: string;
}

interface ChatHistoryResponse {
  messages: Array<{
    id: string;
    role: 'user' | 'assistant';
    content: string;
    created_at: string;
  }>;
}

type SSEEvent =
  | { type: 'token'; text: string }
  | { type: 'tool_call'; tool: string }
  | { type: 'tool_result'; tool: string }
  | { type: 'done'; session_id?: string }
  | { type: 'error'; msg?: string };

export function useChat(sessionId: string) {
  const [messages, setMessages] = useState<ChatMessage[]>([]);
  const [isStreaming, setIsStreaming] = useState(false);
  const [toolsActive, setToolsActive] = useState<string[]>([]);
  const [error, setError] = useState<string | null>(null);

  useEffect(() => {
    let cancelled = false;

    async function loadHistory() {
      if (!sessionId) return;
      const token = localStorage.getItem('pacer_token');
      if (!token) return;

      try {
        const response = await fetch(`${API_BASE_URL}/chat/history?session_id=${encodeURIComponent(sessionId)}`, {
          headers: {
            Authorization: `Bearer ${token}`,
          },
        });

        if (!response.ok) return;
        const data = (await response.json()) as ChatHistoryResponse;
        if (cancelled) return;

        setMessages(
          (data.messages ?? []).map((m) => ({
            id: m.id,
            role: m.role,
            content: m.content,
            created_at: m.created_at,
          })),
        );
      } catch {
        if (!cancelled) {
          setMessages([]);
        }
      }
    }

    void loadHistory();

    return () => {
      cancelled = true;
    };
  }, [sessionId]);

  const sendMessage = useCallback(
    async (text: string) => {
      if (!text.trim() || !sessionId || isStreaming) return;

      const token = localStorage.getItem('pacer_token');
      if (!token) {
        setError('Missing auth token. Please log in again.');
        return;
      }

      setError(null);

      const userMessage: ChatMessage = {
        id: crypto.randomUUID(),
        role: 'user',
        content: text,
      };

      const assistantMessageId = crypto.randomUUID();
      setMessages((prev) => [...prev, userMessage, { id: assistantMessageId, role: 'assistant', content: '' }]);
      setIsStreaming(true);

      try {
        const response = await fetch(`${API_BASE_URL}/chat`, {
          method: 'POST',
          headers: {
            'Content-Type': 'application/json',
            Accept: 'text/event-stream',
            Authorization: `Bearer ${token}`,
          },
          body: JSON.stringify({ message: text, session_id: sessionId }),
        });

        if (!response.ok || !response.body) {
          throw new Error('Unable to start chat stream');
        }

        const reader = response.body.getReader();
        const decoder = new TextDecoder();
        let buffer = '';
        let assistantText = '';

        while (true) {
          const { done, value } = await reader.read();
          if (done) break;

          buffer += decoder.decode(value, { stream: true });
          const chunks = buffer.split('\n\n');
          buffer = chunks.pop() ?? '';

          for (const chunk of chunks) {
            const line = chunk
              .split('\n')
              .find((candidate) => candidate.startsWith('data: '));
            if (!line) continue;

            let event: SSEEvent;
            try {
              event = JSON.parse(line.slice(6)) as SSEEvent;
            } catch {
              continue;
            }

            if (event.type === 'token') {
              assistantText += event.text;
              setMessages((prev) =>
                prev.map((m) => (m.id === assistantMessageId ? { ...m, content: assistantText } : m)),
              );
            } else if (event.type === 'tool_call') {
              setToolsActive((prev) => (prev.includes(event.tool) ? prev : [...prev, event.tool]));
            } else if (event.type === 'tool_result') {
              setToolsActive((prev) => prev.filter((tool) => tool !== event.tool));
            } else if (event.type === 'error') {
              setError(event.msg ?? 'Chat stream failed');
            } else if (event.type === 'done') {
              setToolsActive([]);
              setIsStreaming(false);
            }
          }
        }
      } catch (streamError) {
        setError(streamError instanceof Error ? streamError.message : 'Failed to send message');
      } finally {
        setIsStreaming(false);
        setToolsActive([]);
      }
    },
    [isStreaming, sessionId],
  );

  return useMemo(
    () => ({
      messages,
      isStreaming,
      toolsActive,
      error,
      sendMessage,
    }),
    [messages, isStreaming, toolsActive, error, sendMessage],
  );
}
