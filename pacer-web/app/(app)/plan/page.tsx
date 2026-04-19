'use client';

import { useMemo, useState } from 'react';
import { useActivePlan, useGeneratePlan, usePlanWeek } from '@/hooks/usePlan';
import { PlanCalendar } from '@/components/plan-calendar';

function getTargetRaceDate(value: unknown): string | null {
  if (!value) return null;
  if (typeof value === 'string') {
    return value.slice(0, 10);
  }
  if (typeof value === 'object' && value !== null) {
    const maybe = value as { Time?: string; Valid?: boolean };
    if (maybe.Valid && maybe.Time) {
      return maybe.Time.slice(0, 10);
    }
  }
  return null;
}

export default function PlanPage() {
  const [raceDate, setRaceDate] = useState('');
  const { data: activePlan, isLoading: activeLoading, isError: noActivePlan } = useActivePlan();
  const { data: weekPlan, isLoading: weekLoading } = usePlanWeek(0);
  const generatePlan = useGeneratePlan();

  const hasActivePlan = !!activePlan && !noActivePlan;

  const effectiveRaceDate = useMemo(() => {
    return getTargetRaceDate(activePlan?.target_race) ?? '';
  }, [activePlan?.target_race]);

  async function onGeneratePlan() {
    if (!raceDate) return;
    await generatePlan.mutateAsync({ race_date: raceDate });
    setRaceDate('');
  }

  return (
    <div className="space-y-6">
      <div className="flex items-center justify-between">
        <h1 className="text-2xl font-semibold text-gray-900">Training Plan</h1>
        {effectiveRaceDate && (
          <span className="rounded-full bg-indigo-50 px-3 py-1 text-sm font-medium text-indigo-700">Race: {effectiveRaceDate}</span>
        )}
      </div>

      {!hasActivePlan && !activeLoading && (
        <div className="rounded-xl border border-gray-200 bg-white p-5 shadow-sm">
          <h2 className="text-lg font-semibold text-gray-900">Generate Plan</h2>
          <p className="mt-1 text-sm text-gray-600">No active training plan found. Add your race date to generate one.</p>
          <div className="mt-4 flex flex-col gap-3 sm:flex-row sm:items-end">
            <div>
              <label htmlFor="race_date" className="block text-sm font-medium text-gray-700">
                Race date
              </label>
              <input
                id="race_date"
                type="date"
                value={raceDate}
                onChange={(e) => setRaceDate(e.target.value)}
                className="mt-1 rounded-lg border border-gray-300 px-3 py-2 text-sm text-gray-900"
              />
            </div>
            <button
              type="button"
              onClick={onGeneratePlan}
              disabled={!raceDate || generatePlan.isPending}
              className="rounded-lg bg-indigo-600 px-4 py-2 text-sm font-semibold text-white hover:bg-indigo-700 disabled:opacity-50"
            >
              {generatePlan.isPending ? 'Generating...' : 'Generate Plan'}
            </button>
          </div>
          {generatePlan.error && <p className="mt-3 text-sm text-rose-600">Failed to generate plan. Please retry.</p>}
        </div>
      )}

      <PlanCalendar workouts={weekPlan?.workouts ?? []} weekStart={weekPlan?.week_start} isLoading={weekLoading} />
    </div>
  );
}
