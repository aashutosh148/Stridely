'use client';

import { X } from 'lucide-react';
import { useSSENotifications } from '@/hooks/useSSENotifications';

export function NotificationToaster() {
  const { toasts, dismissToast } = useSSENotifications();

  if (toasts.length === 0) return null;

  return (
    <div className="pointer-events-none fixed bottom-4 right-4 z-[70] space-y-2">
      {toasts.map((toast) => (
        <div
          key={toast.id}
          className="pointer-events-auto w-80 rounded-xl border border-slate-200 bg-white p-3 shadow-lg"
        >
          <div className="flex items-start justify-between gap-2">
            <div>
              <p className="text-sm font-semibold text-slate-900">{toast.title}</p>
              <p className="mt-1 text-sm text-slate-600">{toast.body}</p>
            </div>
            <button
              type="button"
              onClick={() => dismissToast(toast.id)}
              className="rounded p-1 text-slate-400 transition hover:bg-slate-100 hover:text-slate-700"
            >
              <X size={14} />
            </button>
          </div>
        </div>
      ))}
    </div>
  );
}
