'use client';

import { useQuery } from '@tanstack/react-query';
import { api } from '@/lib/api';

export interface ReadinessFactors {
  hrv_status?: string;
  hrv_vs_baseline?: number;
  sleep_hours?: number;
  sleep_score?: number;
  atl_yesterday?: number;
  tsb_today?: number;
}

export interface ReadinessWorkout {
  id: string;
  workout_type: string;
  distance_km?: number | null;
  duration_min?: number | null;
  description?: string | null;
  purpose?: string | null;
  status?: string;
}

export interface ReadinessResponse {
  date: string;
  score: number;
  level: 'green' | 'amber' | 'red' | string;
  decision?: string;
  note?: string;
  planned_workout?: ReadinessWorkout | null;
  adjusted_workout?: ReadinessWorkout | null;
  factors?: ReadinessFactors;
}

export function useReadiness() {
  return useQuery({
    queryKey: ['readiness', 'today'],
    queryFn: () => api.get<ReadinessResponse>('/readiness/today'),
    staleTime: 5 * 60 * 1000,
    refetchInterval: 15 * 60 * 1000,
  });
}
