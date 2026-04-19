'use client';

import { FormEvent, useMemo, useState } from 'react';
import { Bot, Send, Sparkles, User, MessageCircle } from 'lucide-react';
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
    <div className="flex h-[calc(100vh-9rem)] flex-col overflow-hidden rounded-2xl border border-gray-200 bg-white shadow-xl">
      {/* Header */}
      <div className="bg-gradient-to-br from-indigo-500 via-purple-500 to-pink-500 px-6 py-5">
        <div className="flex items-center gap-3">
          <div className="flex h-10 w-10 items-center justify-center rounded-xl bg-white/20 backdrop-blur-sm">
            <MessageCircle className="h-5 w-5 text-white" />
          </div>
          <div>
            <h1 className="text-xl font-bold text-white">Coach Chat</h1>
            <p className="text-sm text-white/80">Get personalized training insights and advice</p>
          </div>
        </div>
      </div>

      {/* Messages Area */}
      <div className="flex-1 space-y-4 overflow-y-auto bg-gradient-to-b from-gray-50 to-white p-6">
        {showSuggestions ? (
          <div className="rounded-xl border-2 border-dashed border-indigo-200 bg-gradient-to-br from-indigo-50 to-purple-50 p-6">
            <div className="mb-4 flex items-center gap-2">
              <Sparkles className="h-5 w-5 text-indigo-600" />
              <p className="font-semibold text-indigo-900">Get started with these questions:</p>
            </div>
            <div className="flex flex-wrap gap-3">
              {suggestedPrompts.map((prompt) => (
                <button
                  key={prompt}
                  type="button"
                  onClick={() => setInput(prompt)}
                  className="rounded-xl border border-indigo-200 bg-white px-4 py-2 text-sm font-medium text-indigo-700 shadow-sm transition hover:border-indigo-300 hover:bg-indigo-50 hover:shadow-md"
                >
                  {prompt}
                </button>
              ))}
            </div>
          </div>
        ) : null}

        {messages.map((message, idx) => (
          <div
            key={message.id}
            className={`flex ${message.role === 'user' ? 'justify-end' : 'justify-start'}`}
            style={{ animationDelay: `${idx * 50}ms` }}
          >
            <div
              className={`max-w-[85%] rounded-2xl px-5 py-4 text-sm shadow-md sm:max-w-[70%] ${
                message.role === 'user'
                  ? 'bg-gradient-to-br from-indigo-600 to-purple-600 text-white'
                  : 'border border-gray-200 bg-white text-gray-800'
              }`}
            >
              <div className="mb-2 flex items-center gap-2 text-xs font-semibold opacity-80">
                {message.role === 'user' ? (
                  <>
                    <User size={14} />
                    <span>You</span>
                  </>
                ) : (
                  <>
                    <Bot size={14} />
                    <span>Pacer Coach</span>
                  </>
                )}
              </div>
              <p className="whitespace-pre-wrap leading-relaxed">{message.content || (isStreaming ? '...' : '')}</p>
            </div>
          </div>
        ))}

        {isStreaming && toolsActive.length > 0 ? (
          <div className="inline-flex items-center gap-2 rounded-xl bg-gradient-to-r from-amber-100 to-orange-100 px-4 py-2 text-xs font-semibold text-amber-800 shadow-sm">
            <Sparkles className="h-4 w-4 animate-pulse" />
            <span className="animate-pulse">Analyzing your training data...</span>
          </div>
        ) : null}

        {error ? (
          <div className="rounded-xl bg-rose-50 p-4">
            <p className="text-sm font-medium text-rose-600">{error}</p>
          </div>
        ) : null}
      </div>

      {/* Input Form */}
      <form onSubmit={handleSubmit} className="border-t border-gray-200 bg-gradient-to-r from-gray-50 to-white p-5">
        <div className="flex items-center gap-3">
          <input
            type="text"
            value={input}
            onChange={(event) => setInput(event.target.value)}
            placeholder="Ask about your training, nutrition, or race strategy..."
            disabled={isStreaming}
            className="w-full rounded-xl border-2 border-gray-300 bg-white px-4 py-3 text-sm text-gray-900 outline-none transition focus:border-indigo-500 focus:ring-2 focus:ring-indigo-200 disabled:bg-gray-100"
          />
          <button
            type="submit"
            disabled={isStreaming || !input.trim()}
            className="inline-flex items-center gap-2 rounded-xl bg-gradient-to-r from-indigo-600 to-purple-600 px-6 py-3 text-sm font-bold text-white shadow-md transition hover:shadow-lg disabled:cursor-not-allowed disabled:opacity-50"
          >
            <Send size={16} />
            Send
          </button>
        </div>
      </form>
    </div>
  );
}
