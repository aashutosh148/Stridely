'use client';

import { useMemo } from 'react';
import { ChatWindow } from '@/components/chat-window';
import { useChat } from '@/hooks/useChat';

const SESSION_STORAGE_KEY = 'pacer_chat_session_id';

function getSessionId() {
  if (typeof window === 'undefined') return '';
  const existing = localStorage.getItem(SESSION_STORAGE_KEY);
  if (existing) return existing;
  const generated = crypto.randomUUID();
  localStorage.setItem(SESSION_STORAGE_KEY, generated);
  return generated;
}

export default function ChatPage() {
  const sessionId = useMemo(() => getSessionId(), []);
  const { messages, isStreaming, toolsActive, error, sendMessage } = useChat(sessionId);

  return (
    <div className="mx-auto max-w-4xl">
      <ChatWindow
        messages={messages}
        isStreaming={isStreaming}
        toolsActive={toolsActive}
        error={error}
        onSend={sendMessage}
      />
    </div>
  );
}
