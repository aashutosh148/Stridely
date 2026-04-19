'use client';

import { useEffect, useState } from 'react';
import { X, Calendar, Clock, Footprints, Heart, TrendingUp, Zap, MapPin } from 'lucide-react';

const API_BASE_URL = process.env.NEXT_PUBLIC_API_URL || 'http://localhost:3001/api/v1';

interface Split {
  km: number;
  pace_s: number;
  hr: number;
  elevation_m: number;
}

interface ZoneDistribution {
  z1_pct: number;
  z2_pct: number;
  z3_pct: number;
  z4_pct: number;
  z5_pct: number;
}

interface ActivityDetail {
  id: string;
  activity_date: string;
  workout_type: string;
  distance_m: number;
  duration_s: number;
  avg_pace_s: number | null;
  avg_hr: number;
  max_hr: number;
  tss: number;
  elevation_gain_m: number;
  intensity_factor: number;
  cardiac_decoupling_pct: number;
  zone_distribution: ZoneDistribution | null;
  splits_km: Split[];
}

interface ActivityDetailModalProps {
  activityId: string | null;
  isOpen: boolean;
  onClose: () => void;
}

export function ActivityDetailModal({ activityId, isOpen, onClose }: ActivityDetailModalProps) {
  const [activity, setActivity] = useState<ActivityDetail | null>(null);
  const [loading, setLoading] = useState(false);

  useEffect(() => {
    if (isOpen && activityId) {
      fetchActivityDetail();
    }
  }, [isOpen, activityId]);

  const fetchActivityDetail = async () => {
    try {
      setLoading(true);
      const token = localStorage.getItem('pacer_token');
      if (!token) return;

      const response = await fetch(`${API_BASE_URL}/activities/${activityId}`, {
        headers: { Authorization: `Bearer ${token}` },
      });

      if (!response.ok) throw new Error('Failed to fetch activity');

      const data = await response.json();
      setActivity(data);
    } catch (error) {
      console.error('Error fetching activity:', error);
    } finally {
      setLoading(false);
    }
  };

  const formatDuration = (seconds: number) => {
    const hrs = Math.floor(seconds / 3600);
    const mins = Math.floor((seconds % 3600) / 60);
    const secs = seconds % 60;
    if (hrs > 0) return `${hrs}:${mins.toString().padStart(2, '0')}:${secs.toString().padStart(2, '0')}`;
    return `${mins}:${secs.toString().padStart(2, '0')}`;
  };

  const formatPace = (paceSeconds: number | null | undefined) => {
    if (!paceSeconds || isNaN(paceSeconds)) return '-';
    const mins = Math.floor(paceSeconds / 60);
    const secs = Math.floor(paceSeconds % 60);
    return `${mins}:${secs.toString().padStart(2, '0')}/km`;
  };

  const formatDistance = (meters: number) => {
    return (meters / 1000).toFixed(2);
  };

  if (!isOpen) return null;

  return (
    <div className="fixed inset-0 z-50 overflow-y-auto">
      <div className="flex min-h-screen items-center justify-center p-4">
        {/* Backdrop */}
        <div 
          className="fixed inset-0 bg-black/80 backdrop-blur-sm transition-opacity"
          onClick={onClose}
        />
        
        {/* Modal */}
        <div className="relative w-full max-w-4xl bg-[#161b26] rounded-lg shadow-2xl transform transition-all border border-gray-800">
          {/* Header */}
          <div className="border-b border-gray-800 p-6">
            <div className="flex items-center justify-between">
              <div>
                <h2 className="text-2xl font-bold text-gray-100 capitalize">
                  {activity?.workout_type.replace('_', ' ') || 'Activity Details'}
                </h2>
                <p className="text-gray-400 mt-1 text-sm">
                  {activity && new Date(activity.activity_date).toLocaleDateString('en-US', {
                    weekday: 'long',
                    year: 'numeric',
                    month: 'long',
                    day: 'numeric'
                  })}
                </p>
              </div>
              <button
                onClick={onClose}
                className="p-2 hover:bg-gray-800 rounded-lg transition-colors text-gray-400 hover:text-gray-200"
              >
                <X className="w-6 h-6" />
              </button>
            </div>
          </div>

          {/* Content */}
          <div className="p-6 max-h-[70vh] overflow-y-auto">
            {loading ? (
              <div className="space-y-4">
                {Array.from({ length: 5 }).map((_, i) => (
                  <div key={i} className="h-20 bg-gray-800/30 animate-pulse rounded-lg" />
                ))}
              </div>
            ) : activity ? (
              <div className="space-y-6">
                {/* Main Metrics */}
                <div className="grid grid-cols-2 md:grid-cols-4 gap-4">
                  <div className="bg-blue-500/10 rounded-lg p-4 border border-blue-500/30">
                    <div className="flex items-center gap-2 mb-2">
                      <Footprints className="w-5 h-5 text-blue-400" />
                      <span className="text-sm text-gray-400">Distance</span>
                    </div>
                    <div className="text-2xl font-bold text-gray-100">{formatDistance(activity.distance_m)} km</div>
                  </div>

                  <div className="bg-purple-500/10 rounded-lg p-4 border border-purple-500/30">
                    <div className="flex items-center gap-2 mb-2">
                      <Clock className="w-5 h-5 text-purple-400" />
                      <span className="text-sm text-gray-400">Duration</span>
                    </div>
                    <div className="text-2xl font-bold text-gray-100 font-mono">{formatDuration(activity.duration_s)}</div>
                  </div>

                  <div className="bg-emerald-500/10 rounded-lg p-4 border border-emerald-500/30">
                    <div className="flex items-center gap-2 mb-2">
                      <TrendingUp className="w-5 h-5 text-emerald-400" />
                      <span className="text-sm text-gray-400">Avg Pace</span>
                    </div>
                    <div className="text-2xl font-bold text-gray-100 font-mono">{formatPace(activity.avg_pace_s)}</div>
                  </div>

                  <div className="bg-red-500/10 rounded-lg p-4 border border-red-500/30">
                    <div className="flex items-center gap-2 mb-2">
                      <Heart className="w-5 h-5 text-red-400" />
                      <span className="text-sm text-gray-400">Avg HR</span>
                    </div>
                    <div className="text-2xl font-bold text-gray-100">{activity.avg_hr || '-'} bpm</div>
                  </div>
                </div>

                {/* Secondary Metrics */}
                <div className="grid grid-cols-2 md:grid-cols-5 gap-4">
                  {activity.max_hr > 0 && (
                    <div className="text-center p-3 bg-[#1e2530] rounded-lg border border-gray-800">
                      <div className="text-xs text-gray-500 mb-1">Max HR</div>
                      <div className="text-lg font-semibold text-gray-200">{activity.max_hr} bpm</div>
                    </div>
                  )}
                  {activity.elevation_gain_m > 0 && (
                    <div className="text-center p-3 bg-[#1e2530] rounded-lg border border-gray-800">
                      <div className="text-xs text-gray-500 mb-1">Elevation</div>
                      <div className="text-lg font-semibold text-gray-200">{Math.round(activity.elevation_gain_m)} m</div>
                    </div>
                  )}
                  {activity.tss > 0 && (
                    <div className="text-center p-3 bg-blue-500/10 rounded-lg border border-blue-500/30">
                      <div className="text-xs text-blue-400 mb-1">TSS</div>
                      <div className="text-lg font-bold text-blue-300">{activity.tss.toFixed(0)}</div>
                    </div>
                  )}
                  {activity.intensity_factor > 0 && (
                    <div className="text-center p-3 bg-[#1e2530] rounded-lg border border-gray-800">
                      <div className="text-xs text-gray-500 mb-1">IF</div>
                      <div className="text-lg font-semibold text-gray-200">{activity.intensity_factor.toFixed(2)}</div>
                    </div>
                  )}
                  {activity.cardiac_decoupling_pct > 0 && (
                    <div className="text-center p-3 bg-[#1e2530] rounded-lg border border-gray-800">
                      <div className="text-xs text-gray-500 mb-1">Decoupling</div>
                      <div className="text-lg font-semibold text-gray-200">{activity.cardiac_decoupling_pct.toFixed(1)}%</div>
                    </div>
                  )}
                </div>

                {/* HR Zones */}
                {activity.zone_distribution && (
                  <div className="bg-[#1e2530] rounded-lg border border-gray-800 p-6">
                    <h3 className="text-lg font-bold text-gray-100 mb-4">Heart Rate Zones</h3>
                    <div className="space-y-3">
                      {[
                        { zone: 'Zone 1', pct: activity.zone_distribution.z1_pct, color: 'bg-gray-500' },
                        { zone: 'Zone 2', pct: activity.zone_distribution.z2_pct, color: 'bg-blue-500' },
                        { zone: 'Zone 3', pct: activity.zone_distribution.z3_pct, color: 'bg-yellow-500' },
                        { zone: 'Zone 4', pct: activity.zone_distribution.z4_pct, color: 'bg-orange-500' },
                        { zone: 'Zone 5', pct: activity.zone_distribution.z5_pct, color: 'bg-red-500' },
                      ].map(({ zone, pct, color }) => (
                        pct > 0 && (
                          <div key={zone} className="flex items-center gap-3">
                            <div className="w-20 text-sm text-gray-400">{zone}</div>
                            <div className="flex-1 bg-gray-800 rounded-full h-6 overflow-hidden">
                              <div 
                                className={`h-full ${color} flex items-center justify-end pr-2 transition-all duration-1000`}
                                style={{ width: `${pct}%` }}
                              >
                                <span className="text-xs font-semibold text-white">{pct.toFixed(1)}%</span>
                              </div>
                            </div>
                          </div>
                        )
                      ))}
                    </div>
                  </div>
                )}

                {/* Splits */}
                {activity.splits_km && activity.splits_km.length > 0 && (
                  <div className="bg-[#1e2530] rounded-lg border border-gray-800 p-6">
                    <h3 className="text-lg font-bold text-gray-100 mb-4">Splits</h3>
                    <div className="grid grid-cols-1 md:grid-cols-2 lg:grid-cols-3 gap-3 max-h-60 overflow-y-auto">
                      {activity.splits_km.map((split, idx) => (
                        <div key={idx} className="p-3 bg-[#161b26] rounded-lg border border-gray-800">
                          <div className="flex items-center justify-between mb-1">
                            <span className="text-sm font-semibold text-gray-300">KM {split.km}</span>
                            <span className="text-xs text-gray-500">{split.hr > 0 ? `${split.hr} bpm` : ''}</span>
                          </div>
                          <div className="text-lg font-bold text-gray-100 font-mono">{formatPace(split.pace_s)}</div>
                        </div>
                      ))}
                    </div>
                  </div>
                )}
              </div>
            ) : (
              <div className="text-center py-12">
                <p className="text-gray-500">Failed to load activity details</p>
              </div>
            )}
          </div>
        </div>
      </div>
    </div>
  );
}
