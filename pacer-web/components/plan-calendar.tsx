'use client';

import { useMemo, useState } from 'react';
import { X } from 'lucide-react';
import type { PlanWorkout } from '@/hooks/usePlan';
import { useUpdateWorkout } from '@/hooks/usePlan';

const WEEK_DAYS = ['Mon', 'Tue', 'Wed', 'Thu', 'Fri', 'Sat', 'Sun'];

function toMonday(date: Date) {
  const copy = new Date(date);
  const day = copy.getDay();
  const diff = day === 0 ? -6 : 1 - day;
  copy.setDate(copy.getDate() + diff);
  copy.setHours(0, 0, 0, 0);
  return copy;
}

function dayKeyFromDate(date: Date) {
  const y = date.getFullYear();
  const m = String(date.getMonth() + 1).padStart(2, '0');
  const d = String(date.getDate()).padStart(2, '0');
  return `${y}-${m}-${d}`;
}

function dayKeyFromValue(value: string) {
  return value.slice(0, 10);
}

function statusClass(status: string) {
  if (status === 'completed') return 'bg-emerald-100 text-emerald-700';
  if (status === 'skipped') return 'bg-rose-100 text-rose-700';
  if (status === 'modified') return 'bg-amber-100 text-amber-700';
  return 'bg-slate-100 text-slate-700';
}

function workoutTypeClass(type: string) {
  const map: Record<string, string> = {
    easy: 'bg-sky-100 text-sky-800',
    long: 'bg-indigo-100 text-indigo-800',
    tempo: 'bg-orange-100 text-orange-800',
    interval: 'bg-fuchsia-100 text-fuchsia-800',
    recovery: 'bg-emerald-100 text-emerald-800',
    race: 'bg-rose-100 text-rose-800',
  };
  return map[type] ?? 'bg-gray-100 text-gray-700';
}

export function PlanCalendar({ workouts, weekStart, isLoading }: { workouts: PlanWorkout[]; weekStart?: string; isLoading?: boolean }) {
  if (isLoading) {
    return (
      <div className="rounded-xl border border-gray-200 bg-white p-5 shadow-sm">
        <div className="h-5 w-24 animate-pulse rounded bg-gray-200" />
        <div className="mt-4 grid grid-cols-1 gap-3 sm:grid-cols-2 lg:grid-cols-7">
          {Array.from({ length: 7 }).map((_, idx) => (
            <div key={idx} className="h-24 animate-pulse rounded-lg bg-gray-100" />
          ))}
        </div>
      </div>
    );
  }

  const [selected, setSelected] = useState<PlanWorkout | null>(null);
  const updateWorkout = useUpdateWorkout();

  const monday = useMemo(() => {
    if (weekStart) {
      return toMonday(new Date(weekStart));
    }
    return toMonday(new Date());
  }, [weekStart]);

  const days = useMemo(() => {
    return Array.from({ length: 7 }).map((_, i) => {
      const d = new Date(monday);
      d.setDate(monday.getDate() + i);
      return d;
    });
  }, [monday]);

  const workoutByDay = useMemo(() => {
    return days.map((day) => {
      const dayKey = dayKeyFromDate(day);
      const found = workouts.find((w) => dayKeyFromValue(w.scheduled_date) === dayKey);
      return found ?? null;
    });
  }, [days, workouts]);

  async function setStatus(workout: PlanWorkout, status: 'completed' | 'skipped') {
    await updateWorkout.mutateAsync({ workoutId: workout.id, status });
    setSelected({ ...workout, status });
  }

  return (
    <>
      <div className="rounded-xl border border-gray-200 bg-white p-5 shadow-sm">
        <h2 className="text-sm font-semibold text-gray-900">This Week</h2>
        <div className="mt-4 grid grid-cols-1 gap-3 sm:grid-cols-2 lg:grid-cols-7">
          {days.map((day, idx) => {
            const workout = workoutByDay[idx];
            return (
              <button
                key={day.toISOString()}
                type="button"
                onClick={() => workout && setSelected(workout)}
                className="rounded-lg border border-gray-200 bg-gray-50 p-3 text-left transition hover:border-gray-300 hover:bg-gray-100 disabled:cursor-default"
                disabled={!workout}
              >
                <p className="text-xs font-semibold uppercase tracking-wide text-gray-500">{WEEK_DAYS[idx]}</p>
                <p className="mt-1 text-sm font-medium text-gray-900">{day.getDate()}</p>
                {workout ? (
                  <>
                    <span
                      className={`mt-2 inline-flex rounded-full px-2 py-0.5 text-xs font-semibold ${workoutTypeClass(
                        workout.workout_type,
                      )}`}
                    >
                      {workout.workout_type}
                    </span>
                    <p className="mt-2 text-xs text-gray-600">{workout.distance_km ? `${workout.distance_km.toFixed(1)} km` : '-'}</p>
                    <span
                      className={`mt-2 inline-flex rounded-full px-2 py-0.5 text-[11px] font-semibold ${statusClass(
                        workout.status,
                      )}`}
                    >
                      {workout.status}
                    </span>
                  </>
                ) : (
                  <p className="mt-3 text-xs text-gray-400">Rest</p>
                )}
              </button>
            );
          })}
        </div>
      </div>

      {selected ? (
        <div className="fixed inset-0 z-50">
          <button
            type="button"
            aria-label="Close drawer"
            className="absolute inset-0 bg-black/30"
            onClick={() => setSelected(null)}
          />
          <aside className="absolute right-0 top-0 h-full w-full max-w-md border-l border-gray-200 bg-white p-5 shadow-xl">
            <div className="flex items-center justify-between">
              <h3 className="text-lg font-semibold text-gray-900">Workout details</h3>
              <button type="button" onClick={() => setSelected(null)} className="rounded p-1 text-gray-500 hover:bg-gray-100">
                <X className="h-5 w-5" />
              </button>
            </div>

            <div className="mt-4 space-y-3 text-sm">
              <p>
                <span className="font-semibold text-gray-900">Type:</span> <span className="text-gray-700">{selected.workout_type}</span>
              </p>
              <p>
                <span className="font-semibold text-gray-900">Distance:</span>{' '}
                <span className="text-gray-700">{selected.distance_km ? `${selected.distance_km.toFixed(1)} km` : 'N/A'}</span>
              </p>
              {selected.description ? (
                <p>
                  <span className="font-semibold text-gray-900">Description:</span>{' '}
                  <span className="text-gray-700">{selected.description}</span>
                </p>
              ) : null}
              {selected.purpose ? (
                <p>
                  <span className="font-semibold text-gray-900">Purpose:</span> <span className="text-gray-700">{selected.purpose}</span>
                </p>
              ) : null}
              <p>
                <span className="font-semibold text-gray-900">Status:</span> <span className="text-gray-700">{selected.status}</span>
              </p>
            </div>

            <div className="mt-6 flex gap-3">
              <button
                type="button"
                onClick={() => setStatus(selected, 'completed')}
                disabled={updateWorkout.isPending}
                className="rounded-lg bg-emerald-600 px-4 py-2 text-sm font-semibold text-white hover:bg-emerald-700 disabled:opacity-50"
              >
                Mark completed
              </button>
              <button
                type="button"
                onClick={() => setStatus(selected, 'skipped')}
                disabled={updateWorkout.isPending}
                className="rounded-lg bg-rose-600 px-4 py-2 text-sm font-semibold text-white hover:bg-rose-700 disabled:opacity-50"
              >
                Mark skipped
              </button>
            </div>
          </aside>
        </div>
      ) : null}
    </>
  );
}
