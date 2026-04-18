'use client';

import { useQuery } from '@tanstack/react-query';
import { api } from '@/lib/api';

export interface FitnessPoint {
  date: string;
  ctl: number;
  atl: number;
  tsb: number;
}

export interface ZoneDistribution {
  z1_pct?: number;
  z2_pct?: number;
  z3_pct?: number;
  z4_pct?: number;
  z5_pct?: number;
}

export interface FitnessMetrics {
  ctl: number;
  atl: number;
  tsb: number;
  trend?: string;
  history?: FitnessPoint[];
  zone_distribution_this_week?: ZoneDistribution;
  zone_distribution_prescribed?: ZoneDistribution;
  injury_risk_score?: number;
}

export interface PredictionResponse {
  predicted_finish_time_s: number;
  confidence_band_min: number;
  confidence_band_max: number;
  delta_vs_last_week_s?: number;
}

export function useFitnessMetrics() {
  return useQuery({
    queryKey: ['fitness', 'metrics'],
    queryFn: () => api.get<FitnessMetrics>('/fitness/metrics'),
    staleTime: 60 * 60 * 1000,
  });
}

export function usePrediction() {
  return useQuery({
    queryKey: ['fitness', 'prediction'],
    queryFn: () => api.get<PredictionResponse>('/fitness/prediction'),
    staleTime: 60 * 60 * 1000,
  });
}
