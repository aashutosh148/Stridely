'use client';

import { useMemo } from 'react';
import { useUser } from '../../../hooks/useUser';
import { useReadiness } from '../../../hooks/useReadiness';
import { useActivePlan, usePlanWeek } from '../../../hooks/usePlan';
import { useFitnessMetrics, usePrediction } from '../../../hooks/useFitness';
import { useRecentActivities } from '../../../hooks/useActivities';
import { ReadinessCard } from '../../../components/readiness-card';
import { FitnessChart } from '../../../components/fitness-chart';
import { PredictionBadge } from '../../../components/prediction-badge';
import { ZoneDonut } from '../../../components/zone-donut';

function formatDuration(seconds: number) {
  const hrs = Math.floor(seconds / 3600);
  const mins = Math.floor((seconds % 3600) / 60);
  if (hrs > 0) {
    return `${hrs}h ${mins}m`;
  }
  return `${mins}m`;
}

function formatKm(meters: number) {
  return (meters / 1000).toFixed(1);
}

export default function DashboardPage() {
  const { user } = useUser();
  const { data: readiness, isLoading: readinessLoading } = useReadiness();
  const { data: planWeek } = usePlanWeek(0);
  const { data: activePlan } = useActivePlan();
  const { data: fitness, isLoading: fitnessLoading } = useFitnessMetrics();
  const { data: prediction, isLoading: predictionLoading } = usePrediction();
  const { data: recentActivities, isLoading: recentActivitiesLoading } = useRecentActivities();

  const raceDate = useMemo(() => {
    const target = activePlan?.target_race;
    if (typeof target === 'string') {
      return target.slice(0, 10);
    }
    if (target && typeof target === 'object' && target.Valid && target.Time) {
      return target.Time.slice(0, 10);
    }
    return null;
  }, [activePlan?.target_race]);

  return (
    <div className="space-y-6">
      <h1 className="text-2xl font-semibold leading-tight text-gray-900">
        Welcome back, {user?.name || 'Runner'}
      </h1>

      <div className="grid grid-cols-1 gap-6 lg:grid-cols-3">
        <ReadinessCard readiness={readiness} isLoading={readinessLoading} />
        <PredictionBadge prediction={prediction} isLoading={predictionLoading} />

        <div className="rounded-xl border border-gray-200 bg-white p-5 shadow-sm">
          <h2 className="text-sm font-semibold text-gray-900">Today&apos;s Plan</h2>
          {planWeek?.workouts?.[0] ? (
            <>
              <p className="mt-3 text-lg font-semibold capitalize text-gray-900">{planWeek.workouts[0].workout_type}</p>
              <p className="text-sm text-gray-600">
                {planWeek.workouts[0].distance_km ? `${planWeek.workouts[0].distance_km.toFixed(1)} km` : 'Distance TBD'}
              </p>
              {planWeek.workouts[0].description ? (
                <p className="mt-2 text-sm text-gray-600">{planWeek.workouts[0].description}</p>
              ) : null}
            </>
          ) : (
            <p className="mt-3 text-sm text-gray-500">No workout scheduled today.</p>
          )}
        </div>
      </div>

      <FitnessChart data={fitness?.history} raceDate={raceDate} isLoading={fitnessLoading} />

      <div className="grid grid-cols-1 gap-6 lg:grid-cols-2">
        <ZoneDonut
          actual={fitness?.zone_distribution_this_week}
          prescribed={fitness?.zone_distribution_prescribed}
          isLoading={fitnessLoading}
        />

        <div className="rounded-xl border border-gray-200 bg-white p-5 shadow-sm">
          <h2 className="text-sm font-semibold text-gray-900">Recent Activities</h2>
          <div className="mt-3 space-y-2">
            {recentActivitiesLoading
              ? Array.from({ length: 4 }).map((_, idx) => <div key={idx} className="h-14 animate-pulse rounded-md bg-gray-100" />)
              : (recentActivities?.activities ?? []).slice(0, 6).map((activity) => (
              <div key={activity.id} className="flex items-center justify-between rounded-md bg-gray-50 px-3 py-2 text-sm">
                <div>
                  <p className="font-medium capitalize text-gray-900">{activity.workout_type}</p>
                  <p className="text-xs text-gray-500">{new Date(activity.activity_date).toLocaleDateString()}</p>
                </div>
                <div className="text-right">
                  <p className="font-medium text-gray-900">{formatKm(activity.distance_m)} km</p>
                  <p className="text-xs text-gray-500">{formatDuration(activity.duration_s)}</p>
                </div>
              </div>
            ))}

            {!recentActivities?.activities?.length ? (
              <p className="text-sm text-gray-500">No activities yet. Connect your device to sync runs.</p>
            ) : null}
          </div>
        </div>
      </div>
    </div>
  );
}
