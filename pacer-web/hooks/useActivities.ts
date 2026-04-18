'use client';

import { useQuery } from '@tanstack/react-query';
import { api } from '@/lib/api';

export interface Activity {
  id: string;
  strava_id?: string;
  activity_date: string;
  workout_type: string;
  distance_m: number;
  duration_s: number;
  avg_pace_s?: number | null;
  tss?: number | null;
  zone_distribution?: {
    z1_pct?: number;
    z2_pct?: number;
    z3_pct?: number;
    z4_pct?: number;
    z5_pct?: number;
  } | null;
}

export interface ActivitiesResponse {
  activities: Activity[];
  total: number;
  limit: number;
  offset: number;
}

export interface RecentActivitiesResponse {
  activities: Activity[];
}

export interface UseActivitiesParams {
  limit?: number;
  offset?: number;
  after?: string;
  before?: string;
}

export function useActivities(params: UseActivitiesParams = {}) {
  const query = new URLSearchParams();
  query.set('limit', String(params.limit ?? 20));
  query.set('offset', String(params.offset ?? 0));
  if (params.after) query.set('after', params.after);
  if (params.before) query.set('before', params.before);

  return useQuery({
    queryKey: ['activities', params],
    queryFn: () => api.get<ActivitiesResponse>(`/activities?${query.toString()}`),
  });
}

export function useRecentActivities() {
  return useQuery({
    queryKey: ['activities', 'recent'],
    queryFn: () => api.get<RecentActivitiesResponse>('/activities/recent'),
  });
}
