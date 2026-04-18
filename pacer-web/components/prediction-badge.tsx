'use client';

import type { PredictionResponse } from '@/hooks/useFitness';

function formatTime(seconds: number) {
  const safe = Math.max(0, Math.round(seconds));
  const hours = Math.floor(safe / 3600);
  const minutes = Math.floor((safe % 3600) / 60);
  return `${hours}:${String(minutes).padStart(2, '0')}`;
}

export function PredictionBadge({ prediction, isLoading }: { prediction?: PredictionResponse; isLoading?: boolean }) {
  if (isLoading) {
    return (
      <div className="rounded-xl border border-gray-200 bg-white p-5 shadow-sm">
        <div className="h-5 w-28 animate-pulse rounded bg-gray-200" />
        <div className="mt-4 h-8 w-36 animate-pulse rounded bg-gray-200" />
        <div className="mt-3 h-5 w-32 animate-pulse rounded bg-gray-100" />
      </div>
    );
  }

  if (!prediction) {
    return (
      <div className="rounded-xl border border-gray-200 bg-white p-5 shadow-sm">
        <h2 className="text-sm font-semibold text-gray-900">Prediction</h2>
        <p className="mt-3 text-sm text-gray-500">No prediction yet.</p>
      </div>
    );
  }

  const delta = prediction.delta_vs_last_week_s ?? 0;
  const improving = delta < 0;
  const sign = delta < 0 ? '↓' : '↑';
  const deltaMin = Math.round(Math.abs(delta) / 60);
  const confidenceMin = Math.round((prediction.confidence_band_max - prediction.confidence_band_min) / 60 / 2);

  return (
    <div className="rounded-xl border border-gray-200 bg-white p-5 shadow-sm">
      <h2 className="text-sm font-semibold text-gray-900">Race Prediction</h2>
      <p className="mt-3 text-3xl font-bold tracking-tight text-gray-900">
        {formatTime(prediction.predicted_finish_time_s)}
        <span className="ml-2 text-sm font-medium text-gray-500">+/- {confidenceMin}min</span>
      </p>
      <span
        className={`mt-3 inline-flex items-center rounded-full px-2.5 py-1 text-xs font-semibold ${
          improving ? 'bg-emerald-100 text-emerald-700' : 'bg-rose-100 text-rose-700'
        }`}
      >
        {sign}
        {deltaMin}min vs last week
      </span>
    </div>
  );
}
