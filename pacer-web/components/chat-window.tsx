'use client';

import { FormEvent, useMemo, useState } from 'react';
import { Bot, Send, Sparkles, User } from 'lucide-react';
import type { ChatMessage } from '@/hooks/useChat';

interface ChatWindowProps {
  messages: ChatMessage[];
  isStreaming: boolean;
  toolsActive: string[];
  error: string | null;
  onSend: (text: string) => Promise<void>;
}

const suggestedPrompts = [
  'How was my run today?',
  'Am I on track for my goal?',
  'What should I eat tomorrow?',
];

export function ChatWindow({ messages, isStreaming, toolsActive, error, onSend }: ChatWindowProps) {
  const [input, setInput] = useState('');

  const showSuggestions = useMemo(() => messages.length === 0, [messages.length]);

  async function handleSubmit(event: FormEvent<HTMLFormElement>) {
    event.preventDefault();
    const value = input.trim();
    if (!value || isStreaming) return;
    setInput('');
    await onSend(value);
  }

  return (
    <div className="flex h-[calc(100vh-9rem)] flex-col overflow-hidden rounded-2xl border border-slate-200 bg-white shadow-sm">
      <div className="border-b border-slate-100 bg-slate-50 px-5 py-4">
        <h1 className="text-lg font-semibold text-slate-900">Coach Chat</h1>
        <p className="text-sm text-slate-600">Ask Pacer about readiness, pacing, and race-week decisions.</p>
      </div>

      <div className="flex-1 space-y-4 overflow-y-auto bg-gradient-to-b from-white to-slate-50/60 p-4">
        {showSuggestions ? (
          <div className="rounded-xl border border-dashed border-slate-300 bg-white p-4">
            <p className="mb-3 text-sm font-medium text-slate-700">Try one of these:</p>
            <div className="flex flex-wrap gap-2">
              {suggestedPrompts.map((prompt) => (
                <button
                  key={prompt}
                  type="button"
                  onClick={() => setInput(prompt)}
                  className="rounded-full border border-slate-200 bg-slate-100 px-3 py-1 text-sm text-slate-700 transition hover:bg-slate-200"
                >
                  {prompt}
                </button>
              ))}
            </div>
          </div>
        ) : null}

        {messages.map((message) => (
          <div key={message.id} className={`flex ${message.role === 'user' ? 'justify-end' : 'justify-start'}`}>
            <div
              className={`max-w-[85%] rounded-2xl px-4 py-3 text-sm shadow-sm sm:max-w-[70%] ${
                message.role === 'user'
                  ? 'bg-sky-600 text-white'
                  : 'border border-slate-200 bg-white text-slate-800'
              }`}
            >
              <div className="mb-1 flex items-center gap-1 text-xs opacity-80">
                {message.role === 'user' ? <User size={12} /> : <Bot size={12} />}
                <span>{message.role === 'user' ? 'You' : 'Pacer'}</span>
              </div>
              <p className="whitespace-pre-wrap leading-relaxed">{message.content || (isStreaming ? '...' : '')}</p>
            </div>
          </div>
        ))}

        {isStreaming && toolsActive.length > 0 ? (
          <div className="inline-flex items-center gap-2 rounded-full bg-amber-100 px-3 py-1 text-xs font-medium text-amber-800">
            <Sparkles className="h-3 w-3 animate-pulse" />
            <span className="animate-pulse">Analysing your long run...</span>
          </div>
        ) : null}

        {error ? <p className="text-sm text-rose-600">{error}</p> : null}
      </div>

      <form onSubmit={handleSubmit} className="border-t border-slate-200 bg-white p-4">
        <div className="flex items-center gap-2">
          <input
            type="text"
            value={input}
            onChange={(event) => setInput(event.target.value)}
            placeholder="Ask about your training..."
            disabled={isStreaming}
            className="w-full rounded-xl border border-slate-300 px-4 py-2 text-sm text-slate-900 outline-none transition focus:border-sky-500 focus:ring-2 focus:ring-sky-100 disabled:bg-slate-100"
          />
          <button
            type="submit"
            disabled={isStreaming || !input.trim()}
            className="inline-flex items-center gap-1 rounded-xl bg-sky-600 px-4 py-2 text-sm font-semibold text-white transition hover:bg-sky-700 disabled:cursor-not-allowed disabled:opacity-50"
          >
            <Send size={14} />
            Send
          </button>
        </div>
      </form>
    </div>
  );
}
