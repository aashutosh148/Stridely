'use client';

import { useMemo, useState } from 'react';
import { Calendar, Target, Zap } from 'lucide-react';
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

function formatDateLong(dateStr: string): string {
  const date = new Date(dateStr);
  return date.toLocaleDateString('en-US', { month: 'long', day: 'numeric', year: 'numeric' });
}

function getDaysUntilRace(raceDate: string): number {
  const today = new Date();
  const race = new Date(raceDate);
  const diff = race.getTime() - today.getTime();
  return Math.ceil(diff / (1000 * 60 * 60 * 24));
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

  const daysUntilRace = useMemo(() => {
    return effectiveRaceDate ? getDaysUntilRace(effectiveRaceDate) : null;
  }, [effectiveRaceDate]);

  async function onGeneratePlan() {
    if (!raceDate) return;
    await generatePlan.mutateAsync({ race_date: raceDate });
    setRaceDate('');
  }

  return (
    <div className="space-y-6">
      {/* Dark Hero Header */}
      <div className="relative overflow-hidden rounded-lg bg-[#161b26] p-8 border border-gray-800">
        <div className="relative z-10">
          <div className="flex items-center gap-3">
            <div className="flex h-12 w-12 items-center justify-center rounded-lg bg-blue-600">
              <Calendar className="h-6 w-6 text-white" />
            </div>
            <div>
              <h1 className="text-3xl font-bold text-gray-100">Training Plan</h1>
              <p className="text-sm text-gray-400">Your personalized path to success</p>
            </div>
          </div>

          {effectiveRaceDate && daysUntilRace !== null && (
            <div className="mt-6 flex flex-wrap gap-4">
              <div className="rounded-lg bg-[#1e2530] px-4 py-3 border border-gray-800">
                <div className="flex items-center gap-2">
                  <Target className="h-5 w-5 text-gray-400" />
                  <div>
                    <p className="text-xs font-medium text-gray-500">Race Day</p>
                    <p className="text-lg font-bold text-gray-100">{formatDateLong(effectiveRaceDate)}</p>
                  </div>
                </div>
              </div>
              <div className="rounded-lg bg-[#1e2530] px-4 py-3 border border-gray-800">
                <div className="flex items-center gap-2">
                  <Zap className="h-5 w-5 text-gray-400" />
                  <div>
                    <p className="text-xs font-medium text-gray-500">Days Until Race</p>
                    <p className="text-lg font-bold text-gray-100">{daysUntilRace} days</p>
                  </div>
                </div>
              </div>
            </div>
          )}
        </div>
      </div>

      {!hasActivePlan && !activeLoading && (
        <div className="overflow-hidden rounded-lg border border-gray-800 bg-[#161b26]">
          <div className="bg-[#1e2530] p-6 border-b border-gray-800">
            <h2 className="text-xl font-bold text-gray-100">Generate Your Training Plan</h2>
            <p className="mt-1 text-sm text-gray-400">
              No active training plan found. Enter your race date to generate a personalized training schedule.
            </p>
          </div>
          <div className="p-6">
            <div className="flex flex-col gap-4 sm:flex-row sm:items-end">
              <div className="flex-1">
                <label htmlFor="race_date" className="block text-sm font-medium text-gray-300">
                  Target Race Date
                </label>
                <input
                  id="race_date"
                  type="date"
                  value={raceDate}
                  onChange={(e) => setRaceDate(e.target.value)}
                  className="mt-1 w-full rounded-lg border border-gray-800 bg-[#1e2530] px-4 py-2 text-sm text-gray-100 transition focus:border-blue-600 focus:ring-2 focus:ring-blue-600/20 focus:outline-none"
                />
              </div>
              <button
                type="button"
                onClick={onGeneratePlan}
                disabled={!raceDate || generatePlan.isPending}
                className="rounded-lg bg-blue-600 px-6 py-2 text-sm font-semibold text-white transition hover:bg-blue-500 disabled:opacity-50"
              >
                {generatePlan.isPending ? 'Generating...' : 'Generate Plan'}
              </button>
            </div>
            {generatePlan.error && (
              <div className="mt-4 rounded-lg bg-red-900/50 border border-red-800 p-4">
                <p className="text-sm text-red-400">Failed to generate plan. Please retry.</p>
              </div>
            )}
          </div>
        </div>
      )}

      <PlanCalendar workouts={weekPlan?.workouts ?? []} weekStart={weekPlan?.week_start} isLoading={weekLoading} />
    </div>
  );
}
