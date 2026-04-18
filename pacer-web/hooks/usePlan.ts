'use client';

import { useMutation, useQuery, useQueryClient } from '@tanstack/react-query';
import { api } from '@/lib/api';

export type WorkoutStatus = 'planned' | 'completed' | 'skipped' | 'modified' | string;

export interface PlanWorkout {
  id: string;
  block_id?: string;
  user_id?: string;
  scheduled_date: string;
  workout_type: string;
  distance_km?: number | null;
  duration_min?: number | null;
  pace_target_min?: number | null;
  pace_target_max?: number | null;
  hr_zone?: number | null;
  rpe_target?: number | null;
  description?: string | null;
  purpose?: string | null;
  status: WorkoutStatus;
  completed_activity_id?: string | null;
}

export interface PlanWeekResponse {
  week_number?: number;
  phase?: string;
  week_start: string;
  week_end?: string;
  total_km_planned?: number;
  total_km_done?: number;
  quality_sessions_planned?: number;
  quality_sessions_done?: number;
  workouts: PlanWorkout[];
  count?: number;
}

export interface ActivePlanResponse {
  id: string;
  user_id: string;
  phase: string;
  block_start: string;
  block_end: string;
  target_race?: string | { Time?: string; Valid?: boolean } | null;
  goal_time_s?: number | null;
  peak_ctl?: number | null;
  is_active: boolean;
  created_at: string;
}

export interface GeneratePlanInput {
  race_date: string;
  goal_time_s?: number;
  available_days?: number;
}

export interface GeneratePlanResponse {
  block_id: string;
  total_weeks: number;
  start_date: string;
  race_date: string;
  peak_ctl?: number;
}

export interface UpdateWorkoutInput {
  workoutId: string;
  status?: WorkoutStatus;
  description?: string;
  distance_km?: number;
}

export function usePlanWeek(weekOffset = 0) {
  return useQuery({
    queryKey: ['plan', 'week', weekOffset],
    queryFn: () => api.get<PlanWeekResponse>(`/plan/week?offset=${weekOffset}`),
  });
}

export function useActivePlan() {
  return useQuery({
    queryKey: ['plan', 'active'],
    queryFn: () => api.get<ActivePlanResponse>('/plan/active'),
    retry: false,
  });
}

export function useGeneratePlan() {
  const queryClient = useQueryClient();

  return useMutation({
    mutationFn: (params: GeneratePlanInput) =>
      api.post<GeneratePlanResponse>('/plan/generate', {
        race_date: params.race_date,
        goal_time_s: params.goal_time_s ?? 14400,
        available_days: params.available_days ?? 5,
      }),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ['plan'] });
      queryClient.invalidateQueries({ queryKey: ['readiness'] });
    },
  });
}

export function useUpdateWorkout() {
  const queryClient = useQueryClient();

  return useMutation({
    mutationFn: ({ workoutId, ...payload }: UpdateWorkoutInput) =>
      api.put<{ status: string; message: string }>(`/plan/workout/${workoutId}`, payload),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ['plan'] });
      queryClient.invalidateQueries({ queryKey: ['readiness'] });
    },
  });
}
