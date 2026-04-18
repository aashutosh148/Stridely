'use client';

import { motion, AnimatePresence } from 'framer-motion';
import { useEffect, useState } from 'react';
import type { ReadinessResponse } from '@/hooks/useReadiness';

function levelClasses(level?: string) {
  if (level === 'green') {
    return 'bg-emerald-50 text-emerald-700 ring-emerald-200';
  }
  if (level === 'amber') {
    return 'bg-amber-50 text-amber-700 ring-amber-200';
  }
  return 'bg-rose-50 text-rose-700 ring-rose-200';
}

function formatWorkoutLabel(type?: string) {
  if (!type) return 'Rest day';
  return type.charAt(0).toUpperCase() + type.slice(1);
}

export function ReadinessCard({
  readiness,
  isLoading,
}: {
  readiness?: ReadinessResponse;
  isLoading?: boolean;
}) {
  const [flash, setFlash] = useState(false);

  useEffect(() => {
    const onFlash = () => {
      setFlash(true);
      window.setTimeout(() => setFlash(false), 1500);
    };
    window.addEventListener('pacer:readiness-flash', onFlash);
    return () => window.removeEventListener('pacer:readiness-flash', onFlash);
  }, []);

  if (isLoading) {
    return (
      <div className="rounded-xl border border-gray-200 bg-white p-5 shadow-sm">
        <div className="h-5 w-32 animate-pulse rounded bg-gray-200" />
        <div className="mt-4 h-10 w-16 animate-pulse rounded bg-gray-200" />
        <div className="mt-4 h-4 w-full animate-pulse rounded bg-gray-100" />
      </div>
    );
  }

  const score = readiness?.score ?? 0;
  const planned = readiness?.adjusted_workout ?? readiness?.planned_workout ?? null;

  return (
    <div className={`rounded-xl border p-5 shadow-sm transition ${flash ? 'border-amber-300 bg-amber-50/70' : 'border-gray-200 bg-white'}`}>
      <div className="flex items-start justify-between">
        <h2 className="text-sm font-semibold text-gray-900">Readiness</h2>
        <span
          className={`inline-flex items-center rounded-full px-2.5 py-1 text-xs font-semibold ring-1 ring-inset ${levelClasses(
            readiness?.level,
          )}`}
        >
          {(readiness?.level ?? 'red').toUpperCase()}
        </span>
      </div>

      <div className="mt-4 flex items-end gap-3">
        <AnimatePresence mode="popLayout">
          <motion.span
            key={score}
            initial={{ y: 8, opacity: 0 }}
            animate={{ y: 0, opacity: 1 }}
            exit={{ y: -8, opacity: 0 }}
            transition={{ duration: 0.25 }}
            className="text-4xl font-bold tracking-tight text-gray-900"
          >
            {score}
          </motion.span>
        </AnimatePresence>
        <span className="pb-1 text-sm text-gray-500">/ 10 today</span>
      </div>

      <div className="mt-4 rounded-lg bg-gray-50 p-3">
        <p className="text-xs font-semibold uppercase tracking-wide text-gray-500">Today&apos;s workout</p>
        <p className="mt-1 text-sm font-medium text-gray-900">{formatWorkoutLabel(planned?.workout_type)}</p>
        {planned?.distance_km ? <p className="text-sm text-gray-600">{planned.distance_km.toFixed(1)} km</p> : null}
        {readiness?.adjusted_workout ? (
          <p className="mt-1 text-xs font-medium text-amber-700">Adjusted from original plan based on readiness.</p>
        ) : null}
      </div>

      <div className="mt-4 flex flex-wrap gap-2">
        <span className="inline-flex items-center rounded-full bg-slate-100 px-2.5 py-1 text-xs font-medium text-slate-700">
          HRV: {readiness?.factors?.hrv_status ?? 'n/a'}
        </span>
        <span className="inline-flex items-center rounded-full bg-slate-100 px-2.5 py-1 text-xs font-medium text-slate-700">
          Sleep: {readiness?.factors?.sleep_hours ? `${readiness.factors.sleep_hours.toFixed(1)}h` : 'n/a'}
        </span>
      </div>

      {readiness?.note ? <p className="mt-4 text-sm text-gray-600">{readiness.note}</p> : null}
    </div>
  );
}
