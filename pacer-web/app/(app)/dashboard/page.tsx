'use client';

import { useMemo, useState } from 'react';
import { useUser } from '../../../hooks/useUser';
import { useReadiness } from '../../../hooks/useReadiness';
import { useFitnessMetrics } from '../../../hooks/useFitness';
import { useRecentActivities } from '../../../hooks/useActivities';
import { ConnectBanner } from '../../../components/connect-banner';
import { FitnessChart } from '../../../components/fitness-chart';
import { ZoneDonut } from '../../../components/zone-donut';
import { ActivityDetailModal } from '../../../components/activity-detail-modal';
import { Activity, TrendingUp, Zap, Target, Clock, Footprints, ArrowUp, ArrowDown, Minus } from 'lucide-react';

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

function formatPace(paceSeconds: number | null | undefined) {
  if (!paceSeconds || isNaN(paceSeconds)) return '-';
  const mins = Math.floor(paceSeconds / 60);
  const secs = Math.floor(paceSeconds % 60);
  return `${mins}:${secs.toString().padStart(2, '0')}/km`;
}

export default function DashboardPage() {
  const { user, isConnected } = useUser();
  const { data: readiness, isLoading: readinessLoading } = useReadiness();
  const { data: fitness, isLoading: fitnessLoading } = useFitnessMetrics();
  const { data: recentActivities, isLoading: recentActivitiesLoading } = useRecentActivities();
  
  const [selectedActivityId, setSelectedActivityId] = useState<string | null>(null);
  const [isModalOpen, setIsModalOpen] = useState(false);

  const handleActivityClick = (activityId: string) => {
    setSelectedActivityId(activityId);
    setIsModalOpen(true);
  };

  const handleCloseModal = () => {
    setIsModalOpen(false);
    setSelectedActivityId(null);
  };

  // Calculate weekly stats
  const weeklyStats = useMemo(() => {
    if (!recentActivities?.activities) return { runs: 0, distance: 0, time: 0 };
    
    const oneWeekAgo = new Date();
    oneWeekAgo.setDate(oneWeekAgo.getDate() - 7);
    
    const weekActivities = recentActivities.activities.filter(
      a => new Date(a.activity_date) >= oneWeekAgo
    );
    
    return {
      runs: weekActivities.length,
      distance: weekActivities.reduce((sum, a) => sum + a.distance_m, 0) / 1000,
      time: weekActivities.reduce((sum, a) => sum + a.duration_s, 0) / 3600,
    };
  }, [recentActivities]);

  const getReadinessColor = (score: number) => {
    if (score >= 80) return { bg: 'bg-emerald-500/10', border: 'border-emerald-500/30', text: 'text-emerald-400', icon: 'text-emerald-500' };
    if (score >= 60) return { bg: 'bg-blue-500/10', border: 'border-blue-500/30', text: 'text-blue-400', icon: 'text-blue-500' };
    if (score >= 40) return { bg: 'bg-amber-500/10', border: 'border-amber-500/30', text: 'text-amber-400', icon: 'text-amber-500' };
    return { bg: 'bg-red-500/10', border: 'border-red-500/30', text: 'text-red-400', icon: 'text-red-500' };
  };

  const getFormColor = (tsb: number) => {
    if (tsb > 5) return { bg: 'bg-emerald-500/10', border: 'border-emerald-500/30', text: 'text-emerald-400', icon: 'text-emerald-500' };
    if (tsb < -10) return { bg: 'bg-red-500/10', border: 'border-red-500/30', text: 'text-red-400', icon: 'text-red-500' };
    return { bg: 'bg-blue-500/10', border: 'border-blue-500/30', text: 'text-blue-400', icon: 'text-blue-500' };
  };

  const readinessColors = getReadinessColor(readiness?.readiness_score || 0);
  const formColors = getFormColor(fitness?.tsb || 0);

  return (
    <div className="min-h-screen bg-[#0b0f19] text-gray-100 p-4 md:p-6">
      <div className="max-w-7xl mx-auto space-y-4">
        {/* Header */}
        <div className="bg-[#161b26] border border-gray-800 rounded-lg p-6">
          <div className="flex items-center justify-between">
            <div className="flex items-center gap-4">
              {user?.profile_picture_url && (
                <img 
                  src={user.profile_picture_url} 
                  alt="Profile" 
                  className="w-12 h-12 rounded-lg border border-gray-700 object-cover"
                />
              )}
              <div>
                <h1 className="text-2xl font-semibold text-gray-100">
                  {user?.first_name ? `Welcome back, ${user.first_name}` : (user?.email ? `Welcome back, ${user.email.split('@')[0]}` : 'Welcome to Pacer')}
                </h1>
                <p className="text-sm text-gray-400 mt-0.5">
                  {new Date().toLocaleDateString('en-US', { weekday: 'long', month: 'short', day: 'numeric', year: 'numeric' })}
                </p>
              </div>
            </div>
            <div className="flex items-center gap-3">
              <div className="bg-[#1e2530] border border-gray-800 rounded px-4 py-2">
                <div className="text-xs text-gray-400">This Week</div>
                <div className="text-2xl font-bold text-gray-100 mt-0.5">{weeklyStats.runs}</div>
                <div className="text-xs text-gray-500">runs</div>
              </div>
              <div className="bg-[#1e2530] border border-gray-800 rounded px-4 py-2">
                <div className="text-xs text-gray-400">Distance</div>
                <div className="text-2xl font-bold text-gray-100 mt-0.5">{weeklyStats.distance.toFixed(0)}</div>
                <div className="text-xs text-gray-500">km</div>
              </div>
              <div className="bg-[#1e2530] border border-gray-800 rounded px-4 py-2">
                <div className="text-xs text-gray-400">Time</div>
                <div className="text-2xl font-bold text-gray-100 mt-0.5">{weeklyStats.time.toFixed(1)}</div>
                <div className="text-xs text-gray-500">hrs</div>
              </div>
            </div>
          </div>
        </div>

        {/* Connection Banner */}
        {!isConnected && <ConnectBanner />}

        {/* Key Metrics */}
        <div className="grid grid-cols-1 md:grid-cols-3 gap-4">
          {/* Readiness */}
          <div className="bg-[#161b26] border border-gray-800 rounded-lg p-6">
            <div className="flex items-center justify-between mb-4">
              <div className="flex items-center gap-2">
                <Zap className={`w-5 h-5 ${readinessColors.icon}`} />
                <span className="text-sm font-medium text-gray-300">Readiness</span>
              </div>
              <div className={`px-2 py-1 rounded text-xs font-medium ${readinessColors.bg} ${readinessColors.text} border ${readinessColors.border}`}>
                {readiness?.status || 'No data'}
              </div>
            </div>
            {readinessLoading ? (
              <div className="h-16 bg-gray-800/30 animate-pulse rounded" />
            ) : (
              <>
                <div className="flex items-baseline gap-2">
                  <div className={`text-5xl font-bold ${readinessColors.text}`}>
                    {readiness?.readiness_score?.toFixed(0) ?? '-'}
                  </div>
                  <span className="text-xl text-gray-500">%</span>
                </div>
                <div className="mt-4 h-1.5 bg-gray-800 rounded-full overflow-hidden">
                  <div 
                    className={`h-full ${readinessColors.icon} bg-current transition-all duration-500`}
                    style={{ width: `${readiness?.readiness_score || 0}%` }}
                  />
                </div>
              </>
            )}
          </div>

          {/* Fitness (CTL) */}
          <div className="bg-[#161b26] border border-gray-800 rounded-lg p-6">
            <div className="flex items-center justify-between mb-4">
              <div className="flex items-center gap-2">
                <TrendingUp className="w-5 h-5 text-blue-500" />
                <span className="text-sm font-medium text-gray-300">Fitness (CTL)</span>
              </div>
              <div className="flex items-center gap-1 text-xs text-gray-500">
                {(fitness?.ctl || 0) > 50 ? <ArrowUp className="w-3 h-3 text-emerald-500" /> : <Minus className="w-3 h-3" />}
                <span>{(fitness?.ctl || 0) > 50 ? 'Building' : 'Base'}</span>
              </div>
            </div>
            {fitnessLoading ? (
              <div className="h-16 bg-gray-800/30 animate-pulse rounded" />
            ) : (
              <>
                <div className="flex items-baseline gap-2">
                  <div className="text-5xl font-bold text-blue-400">
                    {fitness?.ctl?.toFixed(0) ?? '0'}
                  </div>
                  <span className="text-sm text-gray-500">TSS</span>
                </div>
                <div className="mt-4 text-xs text-gray-400">
                  Chronic Training Load
                </div>
              </>
            )}
          </div>

          {/* Form (TSB) */}
          <div className="bg-[#161b26] border border-gray-800 rounded-lg p-6">
            <div className="flex items-center justify-between mb-4">
              <div className="flex items-center gap-2">
                <Target className={`w-5 h-5 ${formColors.icon}`} />
                <span className="text-sm font-medium text-gray-300">Form (TSB)</span>
              </div>
              <div className={`px-2 py-1 rounded text-xs font-medium ${formColors.bg} ${formColors.text} border ${formColors.border}`}>
                {(fitness?.tsb ?? 0) > 5 ? 'Fresh' : (fitness?.tsb ?? 0) < -10 ? 'Fatigued' : 'Balanced'}
              </div>
            </div>
            {fitnessLoading ? (
              <div className="h-16 bg-gray-800/30 animate-pulse rounded" />
            ) : (
              <>
                <div className="flex items-baseline gap-2">
                  <div className={`text-5xl font-bold ${formColors.text}`}>
                    {fitness?.tsb >= 0 ? '+' : ''}{fitness?.tsb?.toFixed(0) ?? '0'}
                  </div>
                </div>
                <div className="mt-4 text-xs text-gray-400">
                  Training Stress Balance
                </div>
              </>
            )}
          </div>
        </div>

        {/* Fitness Chart */}
        <div className="bg-[#161b26] border border-gray-800 rounded-lg p-6">
          <div className="flex items-center justify-between mb-6">
            <div>
              <h2 className="text-lg font-semibold text-gray-100">Fitness Progression</h2>
              <p className="text-sm text-gray-400 mt-0.5">Training load over time</p>
            </div>
            <div className="flex items-center gap-4">
              <div className="flex items-center gap-2">
                <div className="w-3 h-3 rounded-sm bg-blue-500"></div>
                <span className="text-xs text-gray-400">CTL</span>
              </div>
              <div className="flex items-center gap-2">
                <div className="w-3 h-3 rounded-sm bg-red-500"></div>
                <span className="text-xs text-gray-400">ATL</span>
              </div>
              <div className="flex items-center gap-2">
                <div className="w-3 h-3 rounded-sm bg-emerald-500"></div>
                <span className="text-xs text-gray-400">TSB</span>
              </div>
            </div>
          </div>
          <FitnessChart data={fitness?.history} raceDate={null} isLoading={fitnessLoading} />
        </div>

        {/* Bottom Grid */}
        <div className="grid grid-cols-1 lg:grid-cols-2 gap-4">
          {/* Zone Distribution */}
          <div className="bg-[#161b26] border border-gray-800 rounded-lg p-6">
            <div className="mb-6">
              <h2 className="text-lg font-semibold text-gray-100">Training Zones</h2>
              <p className="text-sm text-gray-400 mt-0.5">Heart rate distribution this week</p>
            </div>
            <ZoneDonut
              actual={fitness?.zone_distribution_this_week}
              prescribed={fitness?.zone_distribution_prescribed}
              isLoading={fitnessLoading}
            />
          </div>

          {/* Recent Activities */}
          <div className="bg-[#161b26] border border-gray-800 rounded-lg p-6">
            <div className="flex items-center justify-between mb-6">
              <div>
                <h2 className="text-lg font-semibold text-gray-100">Recent Activities</h2>
                <p className="text-sm text-gray-400 mt-0.5">Your latest workouts</p>
              </div>
              <Activity className="w-5 h-5 text-gray-500" />
            </div>
            <div className="space-y-2 max-h-[400px] overflow-y-auto pr-2">
              {recentActivitiesLoading ? (
                Array.from({ length: 5 }).map((_, idx) => (
                  <div key={idx} className="h-16 bg-gray-800/30 animate-pulse rounded" />
                ))
              ) : recentActivities?.activities?.length ? (
                recentActivities.activities.slice(0, 8).map((activity) => (
                  <div
                    key={activity.id}
                    onClick={() => handleActivityClick(activity.id)}
                    className="group bg-[#1e2530] border border-gray-800 hover:border-gray-700 rounded p-3 cursor-pointer transition-all"
                  >
                    <div className="flex items-center justify-between">
                      <div className="flex-1">
                        <div className="flex items-center gap-3 mb-2">
                          <div className="p-1.5 bg-blue-500/10 border border-blue-500/30 rounded">
                            <Footprints className="w-3.5 h-3.5 text-blue-400" />
                          </div>
                          <div className="flex-1">
                            <div className="flex items-center gap-2">
                              <span className="font-medium text-gray-200 capitalize text-sm">
                                {activity.workout_type}
                              </span>
                              <span className="text-xs text-gray-500">
                                {new Date(activity.activity_date).toLocaleDateString('en-US', {
                                  month: 'short',
                                  day: 'numeric'
                                })}
                              </span>
                            </div>
                            <div className="flex items-center gap-4 mt-1">
                              <div className="flex items-center gap-1.5 text-xs text-gray-400">
                                <Footprints className="w-3 h-3" />
                                <span>{formatKm(activity.distance_m)} km</span>
                              </div>
                              <div className="flex items-center gap-1.5 text-xs text-gray-400">
                                <Clock className="w-3 h-3" />
                                <span>{formatDuration(activity.duration_s)}</span>
                              </div>
                              <div className="text-xs text-gray-400 font-mono">
                                {formatPace(activity.avg_pace_s)}
                              </div>
                            </div>
                          </div>
                        </div>
                      </div>
                      {activity.tss && (
                        <div className="ml-4">
                          <div className="px-2.5 py-1 bg-blue-500/10 border border-blue-500/30 rounded text-center">
                            <div className="text-sm font-bold text-blue-400">
                              {activity.tss.toFixed(0)}
                            </div>
                            <div className="text-xs text-gray-500">TSS</div>
                          </div>
                        </div>
                      )}
                    </div>
                  </div>
                ))
              ) : (
                <div className="text-center py-12">
                  <div className="inline-flex items-center justify-center w-12 h-12 rounded-lg bg-gray-800/50 mb-3">
                    <Activity className="w-6 h-6 text-gray-600" />
                  </div>
                  <p className="text-gray-400 font-medium text-sm">No activities yet</p>
                  <p className="text-xs text-gray-500 mt-1">Connect Strava to sync your runs</p>
                </div>
              )}
            </div>
          </div>
        </div>
      </div>

      <ActivityDetailModal
        activityId={selectedActivityId}
        isOpen={isModalOpen}
        onClose={handleCloseModal}
      />
    </div>
  );
}
