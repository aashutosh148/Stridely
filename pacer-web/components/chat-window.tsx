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
    <div className="flex h-[calc(100vh-9rem)] flex-col overflow-hidden rounded-lg border border-gray-800 bg-[#161b26]">
      {/* Header */}
      <div className="bg-blue-600 px-6 py-5 border-b border-gray-800">
        <div className="flex items-center gap-3">
          <div className="flex h-10 w-10 items-center justify-center rounded-lg bg-white/20">
            <MessageCircle className="h-5 w-5 text-white" />
          </div>
          <div>
            <h1 className="text-xl font-bold text-white">Coach Chat</h1>
            <p className="text-sm text-blue-100">Get personalized training insights and advice</p>
          </div>
        </div>
      </div>

      {/* Messages Area */}
      <div className="flex-1 space-y-4 overflow-y-auto bg-[#0b0f19] p-6">
        {showSuggestions ? (
          <div className="rounded-lg border-2 border-dashed border-gray-800 bg-[#161b26] p-6">
            <div className="mb-4 flex items-center gap-2">
              <Sparkles className="h-5 w-5 text-blue-500" />
              <p className="font-semibold text-gray-100">Get started with these questions:</p>
            </div>
            <div className="flex flex-wrap gap-3">
              {suggestedPrompts.map((prompt) => (
                <button
                  key={prompt}
                  type="button"
                  onClick={() => setInput(prompt)}
                  className="rounded-lg border border-gray-800 bg-[#1e2530] px-4 py-2 text-sm font-medium text-gray-300 transition hover:border-blue-600 hover:bg-[#1e2530]/80"
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
              className={`max-w-[85%] rounded-lg px-5 py-4 text-sm sm:max-w-[70%] ${
                message.role === 'user'
                  ? 'bg-blue-600 text-white'
                  : 'border border-gray-800 bg-[#161b26] text-gray-100'
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
          <div className="inline-flex items-center gap-2 rounded-lg bg-amber-900/50 border border-amber-800 px-4 py-2 text-xs font-semibold text-amber-400">
            <Sparkles className="h-4 w-4 animate-pulse" />
            <span className="animate-pulse">Analyzing your training data...</span>
          </div>
        ) : null}

        {error ? (
          <div className="rounded-lg bg-red-900/50 border border-red-800 p-4">
            <p className="text-sm font-medium text-red-400">{error}</p>
          </div>
        ) : null}
      </div>

      {/* Input Form */}
      <form onSubmit={handleSubmit} className="border-t border-gray-800 bg-[#161b26] p-5">
        <div className="flex items-center gap-3">
          <input
            type="text"
            value={input}
            onChange={(event) => setInput(event.target.value)}
            placeholder="Ask about your training, nutrition, or race strategy..."
            disabled={isStreaming}
            className="w-full rounded-lg border-2 border-gray-800 bg-[#1e2530] px-4 py-3 text-sm text-gray-100 placeholder-gray-500 outline-none transition focus:border-blue-600 focus:ring-2 focus:ring-blue-600/20 disabled:bg-[#1e2530]/50"
          />
          <button
            type="submit"
            disabled={isStreaming || !input.trim()}
            className="inline-flex items-center gap-2 rounded-lg bg-blue-600 px-6 py-3 text-sm font-bold text-white transition hover:bg-blue-500 disabled:cursor-not-allowed disabled:opacity-50"
          >
            <Send size={16} />
            Send
          </button>
        </div>
      </form>
    </div>
  );
}
