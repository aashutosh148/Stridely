'use client';

import { useEffect, useMemo, useRef, useState } from 'react';

const API_BASE_URL = process.env.NEXT_PUBLIC_API_URL || 'http://localhost:3001/api/v1';

export interface NotificationToast {
  id: string;
  title: string;
  body: string;
  createdAt: number;
}

interface SSEEnvelope {
  id: string;
  type: string;
  payload?: Record<string, unknown>;
  timestamp?: string;
}

export function useSSENotifications() {
  const [toasts, setToasts] = useState<NotificationToast[]>([]);
  const abortRef = useRef<AbortController | null>(null);

  useEffect(() => {
    let active = true;

    async function connect() {
      while (active) {
        const token = localStorage.getItem('pacer_token');
        if (!token) {
          await new Promise((resolve) => setTimeout(resolve, 1500));
          continue;
        }

        const controller = new AbortController();
        abortRef.current = controller;

        try {
          const response = await fetch(`${API_BASE_URL}/events/stream`, {
            headers: {
              Accept: 'text/event-stream',
              Authorization: `Bearer ${token}`,
            },
            signal: controller.signal,
          });

          if (!response.ok || !response.body) {
            throw new Error('SSE connection failed');
          }

          const reader = response.body.getReader();
          const decoder = new TextDecoder();
          let buffer = '';

          while (active) {
            const { done, value } = await reader.read();
            if (done) break;

            buffer += decoder.decode(value, { stream: true });
            const chunks = buffer.split('\n\n');
            buffer = chunks.pop() ?? '';

            for (const chunk of chunks) {
              const line = chunk.split('\n').find((candidate) => candidate.startsWith('data: '));
              if (!line) continue;

              let event: SSEEnvelope;
              try {
                event = JSON.parse(line.slice(6)) as SSEEnvelope;
              } catch {
                continue;
              }

              if (event.type === 'post_workout') {
                const preview =
                  typeof event.payload?.debrief_preview === 'string'
                    ? event.payload.debrief_preview
                    : 'Your post-workout debrief is ready.';

                setToasts((prev) => [
                  ...prev,
                  {
                    id: event.id || crypto.randomUUID(),
                    title: 'Workout Synced',
                    body: preview,
                    createdAt: Date.now(),
                  },
                ]);
              }

              if (event.type === 'readiness.updated' && window.location.pathname.startsWith('/dashboard')) {
                window.dispatchEvent(new CustomEvent('pacer:readiness-flash'));
              }
            }
          }
        } catch {
          await new Promise((resolve) => setTimeout(resolve, 1500));
        }
      }
    }

    void connect();

    return () => {
      active = false;
      abortRef.current?.abort();
    };
  }, []);

  useEffect(() => {
    if (!toasts.length) return;
    const timer = window.setInterval(() => {
      setToasts((prev) => prev.filter((t) => Date.now() - t.createdAt < 6000));
    }, 500);
    return () => window.clearInterval(timer);
  }, [toasts.length]);

  const dismissToast = (id: string) => {
    setToasts((prev) => prev.filter((t) => t.id !== id));
  };

  return useMemo(
    () => ({
      toasts,
      dismissToast,
    }),
    [toasts],
  );
}
