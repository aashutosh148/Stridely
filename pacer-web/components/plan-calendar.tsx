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
      <div className="overflow-hidden rounded-2xl border border-gray-200 bg-white shadow-lg">
        <div className="bg-gradient-to-r from-indigo-50 to-purple-50 p-5">
          <div className="h-6 w-32 animate-pulse rounded bg-gray-300" />
        </div>
        <div className="p-5">
          <div className="grid grid-cols-1 gap-3 sm:grid-cols-2 lg:grid-cols-7">
            {Array.from({ length: 7 }).map((_, idx) => (
              <div key={idx} className="h-32 animate-pulse rounded-xl bg-gray-100" />
            ))}
          </div>
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
      <div className="overflow-hidden rounded-2xl border border-gray-200 bg-white shadow-lg">
        <div className="bg-gradient-to-r from-indigo-50 to-purple-50 p-5">
          <h2 className="text-lg font-bold text-gray-900">This Week's Training</h2>
          <p className="text-sm text-gray-600">Click a workout to view details and mark completion</p>
        </div>
        <div className="p-5">
          <div className="grid grid-cols-1 gap-3 sm:grid-cols-2 lg:grid-cols-7">
            {days.map((day, idx) => {
              const workout = workoutByDay[idx];
              const isToday = new Date().toDateString() === day.toDateString();
              return (
                <button
                  key={day.toISOString()}
                  type="button"
                  onClick={() => workout && setSelected(workout)}
                  className={`group rounded-xl border-2 p-4 text-left transition-all ${
                    isToday
                      ? 'border-indigo-400 bg-gradient-to-br from-indigo-50 to-purple-50 shadow-md'
                      : 'border-gray-200 bg-gray-50 hover:border-indigo-300 hover:shadow-md'
                  } ${workout ? 'cursor-pointer' : 'cursor-default'}`}
                  disabled={!workout}
                >
                  <div className="flex items-center justify-between">
                    <p className="text-xs font-bold uppercase tracking-wide text-gray-500">{WEEK_DAYS[idx]}</p>
                    {isToday && <span className="h-2 w-2 rounded-full bg-indigo-500"></span>}
                  </div>
                  <p className="mt-1 text-lg font-bold text-gray-900">{day.getDate()}</p>
                  {workout ? (
                    <div className="mt-3 space-y-2">
                      <span
                        className={`inline-flex rounded-lg px-2.5 py-1 text-xs font-bold shadow-sm ${workoutTypeClass(
                          workout.workout_type,
                        )}`}
                      >
                        {workout.workout_type}
                      </span>
                      <p className="text-sm font-semibold text-gray-700">
                        {workout.distance_km ? `${workout.distance_km.toFixed(1)} km` : '-'}
                      </p>
                      <span
                        className={`inline-flex rounded-lg px-2 py-0.5 text-[11px] font-bold ${statusClass(
                          workout.status,
                        )}`}
                      >
                        {workout.status}
                      </span>
                    </div>
                  ) : (
                    <div className="mt-4 flex items-center gap-1">
                      <div className="h-1 w-1 rounded-full bg-gray-300"></div>
                      <p className="text-xs font-medium text-gray-400">Rest Day</p>
                    </div>
                  )}
                </button>
              );
            })}
          </div>
        </div>
      </div>

      {selected ? (
        <div className="fixed inset-0 z-50">
          <button
            type="button"
            aria-label="Close drawer"
            className="absolute inset-0 bg-black/40 backdrop-blur-sm"
            onClick={() => setSelected(null)}
          />
          <aside className="absolute right-0 top-0 h-full w-full max-w-md overflow-hidden border-l border-gray-200 bg-white shadow-2xl">
            <div className="bg-gradient-to-br from-indigo-500 via-purple-500 to-pink-500 p-6">
              <div className="flex items-center justify-between">
                <div>
                  <h3 className="text-xl font-bold text-white">Workout Details</h3>
                  <p className="mt-1 text-sm text-white/80">Review and mark completion</p>
                </div>
                <button
                  type="button"
                  onClick={() => setSelected(null)}
                  className="rounded-lg bg-white/20 p-2 text-white backdrop-blur-sm transition hover:bg-white/30"
                >
                  <X className="h-5 w-5" />
                </button>
              </div>
            </div>

            <div className="p-6">
              <div className="space-y-4">
                <div className="rounded-xl bg-gradient-to-br from-gray-50 to-gray-100 p-4">
                  <p className="text-xs font-medium uppercase tracking-wide text-gray-500">Workout Type</p>
                  <span
                    className={`mt-2 inline-flex rounded-lg px-3 py-1.5 text-sm font-bold shadow-sm ${workoutTypeClass(
                      selected.workout_type,
                    )}`}
                  >
                    {selected.workout_type}
                  </span>
                </div>

                <div className="rounded-xl bg-gradient-to-br from-gray-50 to-gray-100 p-4">
                  <p className="text-xs font-medium uppercase tracking-wide text-gray-500">Distance</p>
                  <p className="mt-1 text-2xl font-bold text-gray-900">
                    {selected.distance_km ? `${selected.distance_km.toFixed(1)} km` : 'N/A'}
                  </p>
                </div>

                {selected.description ? (
                  <div className="rounded-xl bg-gradient-to-br from-blue-50 to-indigo-50 p-4">
                    <p className="text-xs font-medium uppercase tracking-wide text-indigo-600">Description</p>
                    <p className="mt-1 text-sm leading-relaxed text-gray-700">{selected.description}</p>
                  </div>
                ) : null}

                {selected.purpose ? (
                  <div className="rounded-xl bg-gradient-to-br from-purple-50 to-pink-50 p-4">
                    <p className="text-xs font-medium uppercase tracking-wide text-purple-600">Purpose</p>
                    <p className="mt-1 text-sm leading-relaxed text-gray-700">{selected.purpose}</p>
                  </div>
                ) : null}

                <div className="rounded-xl bg-gradient-to-br from-gray-50 to-gray-100 p-4">
                  <p className="text-xs font-medium uppercase tracking-wide text-gray-500">Current Status</p>
                  <span
                    className={`mt-2 inline-flex rounded-lg px-3 py-1.5 text-sm font-bold ${statusClass(
                      selected.status,
                    )}`}
                  >
                    {selected.status}
                  </span>
                </div>
              </div>

              <div className="mt-6 flex gap-3">
                <button
                  type="button"
                  onClick={() => setStatus(selected, 'completed')}
                  disabled={updateWorkout.isPending}
                  className="flex-1 rounded-xl bg-gradient-to-r from-emerald-600 to-green-600 px-4 py-3 text-sm font-bold text-white shadow-md transition hover:shadow-lg disabled:opacity-50"
                >
                  Mark Completed
                </button>
                <button
                  type="button"
                  onClick={() => setStatus(selected, 'skipped')}
                  disabled={updateWorkout.isPending}
                  className="flex-1 rounded-xl bg-gradient-to-r from-rose-600 to-red-600 px-4 py-3 text-sm font-bold text-white shadow-md transition hover:shadow-lg disabled:opacity-50"
                >
                  Mark Skipped
                </button>
              </div>
            </div>
          </aside>
        </div>
      ) : null}
    </>
  );
}
