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
import { Activity, TrendingUp, Zap, Target, Calendar, Clock, Footprints, ArrowUpRight, ArrowDownRight } from 'lucide-react';

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
    if (score >= 80) return 'from-emerald-500 to-green-600';
    if (score >= 60) return 'from-blue-500 to-cyan-600';
    if (score >= 40) return 'from-amber-500 to-orange-600';
    return 'from-rose-500 to-red-600';
  };

  const getFormColor = (tsb: number) => {
    if (tsb > 5) return 'from-emerald-500 to-green-600';
    if (tsb < -10) return 'from-rose-500 to-red-600';
    return 'from-blue-500 to-cyan-600';
  };

  return (
    <div className="space-y-8 max-w-7xl mx-auto">
      {/* Header Section */}
      <div className="relative overflow-hidden bg-gradient-to-br from-indigo-500 via-purple-500 to-pink-500 rounded-2xl p-8 text-white shadow-xl">
        <div className="absolute inset-0 bg-black/10"></div>
        <div className="relative z-10">
          <div className="flex items-center justify-between">
            <div>
              <h1 className="text-4xl font-bold tracking-tight">
                {user?.email ? `Welcome back, ${user.email.split('@')[0]}!` : 'Welcome to Pacer'}
              </h1>
              <p className="text-indigo-100 mt-2 text-lg">
                {new Date().toLocaleDateString('en-US', { weekday: 'long', year: 'numeric', month: 'long', day: 'numeric' })}
              </p>
            </div>
            <div className="hidden md:flex items-center gap-3">
              <div className="bg-white/20 backdrop-blur-sm rounded-xl px-6 py-3 border border-white/30">
                <div className="text-sm text-indigo-100">This Week</div>
                <div className="text-3xl font-bold mt-1">{weeklyStats.runs}</div>
                <div className="text-xs text-indigo-200">Runs</div>
              </div>
              <div className="bg-white/20 backdrop-blur-sm rounded-xl px-6 py-3 border border-white/30">
                <div className="text-sm text-indigo-100">Distance</div>
                <div className="text-3xl font-bold mt-1">{weeklyStats.distance.toFixed(0)}</div>
                <div className="text-xs text-indigo-200">km</div>
              </div>
              <div className="bg-white/20 backdrop-blur-sm rounded-xl px-6 py-3 border border-white/30">
                <div className="text-sm text-indigo-100">Time</div>
                <div className="text-3xl font-bold mt-1">{weeklyStats.time.toFixed(1)}</div>
                <div className="text-xs text-indigo-200">hours</div>
              </div>
            </div>
          </div>
        </div>
      </div>

      {/* Connection Banner */}
      {!isConnected && <ConnectBanner />}

      {/* Key Metrics Cards - Redesigned */}
      <div className="grid grid-cols-1 md:grid-cols-3 gap-6">
        {/* Readiness Card */}
        <div className="group relative overflow-hidden bg-white rounded-2xl shadow-lg hover:shadow-xl transition-all duration-300 border border-gray-100">
          <div className="absolute top-0 right-0 w-32 h-32 bg-gradient-to-br from-emerald-50 to-green-50 rounded-full -mr-16 -mt-16 opacity-50"></div>
          <div className="relative p-6">
            <div className="flex items-center justify-between mb-4">
              <div className="flex items-center gap-3">
                <div className="p-3 bg-gradient-to-br from-emerald-500 to-green-600 rounded-xl shadow-lg">
                  <Zap className="w-6 h-6 text-white" />
                </div>
                <div>
                  <h3 className="text-sm font-medium text-gray-500">Readiness</h3>
                  <p className="text-xs text-gray-400">Daily Recovery Score</p>
                </div>
              </div>
            </div>
            {readinessLoading ? (
              <div className="h-20 bg-gradient-to-r from-gray-100 to-gray-50 animate-pulse rounded-xl" />
            ) : (
              <>
                <div className="flex items-baseline gap-2 mb-3">
                  <div className={`text-5xl font-bold bg-gradient-to-br ${getReadinessColor(readiness?.readiness_score || 0)} bg-clip-text text-transparent`}>
                    {readiness?.readiness_score?.toFixed(0) ?? '-'}
                  </div>
                  <span className="text-2xl text-gray-400 font-medium">%</span>
                </div>
                <div className="flex items-center gap-2">
                  <div className={`inline-flex items-center gap-1 px-3 py-1 rounded-full text-xs font-semibold ${
                    (readiness?.readiness_score || 0) >= 80 ? 'bg-emerald-100 text-emerald-700' :
                    (readiness?.readiness_score || 0) >= 60 ? 'bg-blue-100 text-blue-700' :
                    (readiness?.readiness_score || 0) >= 40 ? 'bg-amber-100 text-amber-700' :
                    'bg-rose-100 text-rose-700'
                  }`}>
                    {readiness?.status || 'No data'}
                  </div>
                </div>
                <div className="mt-4 w-full bg-gray-100 rounded-full h-2 overflow-hidden">
                  <div 
                    className={`h-full bg-gradient-to-r ${getReadinessColor(readiness?.readiness_score || 0)} transition-all duration-1000 ease-out rounded-full`}
                    style={{ width: `${readiness?.readiness_score || 0}%` }}
                  />
                </div>
              </>
            )}
          </div>
        </div>

        {/* Fitness (CTL) Card */}
        <div className="group relative overflow-hidden bg-white rounded-2xl shadow-lg hover:shadow-xl transition-all duration-300 border border-gray-100">
          <div className="absolute top-0 right-0 w-32 h-32 bg-gradient-to-br from-blue-50 to-indigo-50 rounded-full -mr-16 -mt-16 opacity-50"></div>
          <div className="relative p-6">
            <div className="flex items-center justify-between mb-4">
              <div className="flex items-center gap-3">
                <div className="p-3 bg-gradient-to-br from-blue-500 to-indigo-600 rounded-xl shadow-lg">
                  <TrendingUp className="w-6 h-6 text-white" />
                </div>
                <div>
                  <h3 className="text-sm font-medium text-gray-500">Fitness</h3>
                  <p className="text-xs text-gray-400">Chronic Training Load</p>
                </div>
              </div>
            </div>
            {fitnessLoading ? (
              <div className="h-20 bg-gradient-to-r from-gray-100 to-gray-50 animate-pulse rounded-xl" />
            ) : (
              <>
                <div className="flex items-baseline gap-2 mb-3">
                  <div className="text-5xl font-bold bg-gradient-to-br from-blue-500 to-indigo-600 bg-clip-text text-transparent">
                    {fitness?.ctl?.toFixed(0) ?? '0'}
                  </div>
                  <span className="text-sm text-gray-400">TSS</span>
                </div>
                <div className="flex items-center gap-2 text-sm">
                  {(fitness?.ctl || 0) > 50 ? (
                    <><ArrowUpRight className="w-4 h-4 text-emerald-500" /> <span className="text-emerald-600 font-medium">Building</span></>
                  ) : (
                    <span className="text-gray-600">Base Phase</span>
                  )}
                </div>
              </>
            )}
          </div>
        </div>

        {/* Form (TSB) Card */}
        <div className="group relative overflow-hidden bg-white rounded-2xl shadow-lg hover:shadow-xl transition-all duration-300 border border-gray-100">
          <div className="absolute top-0 right-0 w-32 h-32 bg-gradient-to-br from-purple-50 to-pink-50 rounded-full -mr-16 -mt-16 opacity-50"></div>
          <div className="relative p-6">
            <div className="flex items-center justify-between mb-4">
              <div className="flex items-center gap-3">
                <div className={`p-3 bg-gradient-to-br ${getFormColor(fitness?.tsb || 0)} rounded-xl shadow-lg`}>
                  <Target className="w-6 h-6 text-white" />
                </div>
                <div>
                  <h3 className="text-sm font-medium text-gray-500">Form</h3>
                  <p className="text-xs text-gray-400">Training Stress Balance</p>
                </div>
              </div>
            </div>
            {fitnessLoading ? (
              <div className="h-20 bg-gradient-to-r from-gray-100 to-gray-50 animate-pulse rounded-xl" />
            ) : (
              <>
                <div className="flex items-baseline gap-2 mb-3">
                  <div className={`text-5xl font-bold bg-gradient-to-br ${getFormColor(fitness?.tsb || 0)} bg-clip-text text-transparent`}>
                    {fitness?.tsb >= 0 ? '+' : ''}{fitness?.tsb?.toFixed(0) ?? '0'}
                  </div>
                </div>
                <div className="flex items-center gap-2">
                  <div className={`inline-flex items-center gap-1 px-3 py-1 rounded-full text-xs font-semibold ${
                    (fitness?.tsb ?? 0) > 5 ? 'bg-emerald-100 text-emerald-700' :
                    (fitness?.tsb ?? 0) < -10 ? 'bg-rose-100 text-rose-700' :
                    'bg-blue-100 text-blue-700'
                  }`}>
                    {(fitness?.tsb ?? 0) > 5 ? 'Fresh' : (fitness?.tsb ?? 0) < -10 ? 'Fatigued' : 'Balanced'}
                  </div>
                  {(fitness?.tsb ?? 0) > 5 && (
                    <span className="text-xs text-gray-500">Race Ready!</span>
                  )}
                </div>
              </>
            )}
          </div>
        </div>
      </div>

      {/* Fitness Chart - Enhanced */}
      <div className="bg-white rounded-2xl shadow-lg p-8 border border-gray-100">
        <div className="flex items-center justify-between mb-6">
          <div>
            <h2 className="text-2xl font-bold text-gray-900">Fitness Progression</h2>
            <p className="text-sm text-gray-500 mt-1">Track your training load over time</p>
          </div>
          <div className="flex items-center gap-3">
            <div className="flex items-center gap-2">
              <div className="w-3 h-3 rounded-full bg-blue-500"></div>
              <span className="text-xs text-gray-600">CTL</span>
            </div>
            <div className="flex items-center gap-2">
              <div className="w-3 h-3 rounded-full bg-rose-500"></div>
              <span className="text-xs text-gray-600">ATL</span>
            </div>
            <div className="flex items-center gap-2">
              <div className="w-3 h-3 rounded-full bg-emerald-500"></div>
              <span className="text-xs text-gray-600">TSB</span>
            </div>
          </div>
        </div>
        <FitnessChart data={fitness?.history} raceDate={null} isLoading={fitnessLoading} />
      </div>

      {/* Bottom Grid - Zone Distribution & Recent Activities */}
      <div className="grid grid-cols-1 lg:grid-cols-2 gap-6">
        {/* Zone Distribution */}
        <div className="bg-white rounded-2xl shadow-lg p-8 border border-gray-100">
          <div className="mb-6">
            <h2 className="text-2xl font-bold text-gray-900">Training Zones</h2>
            <p className="text-sm text-gray-500 mt-1">Heart rate distribution this week</p>
          </div>
          <ZoneDonut
            actual={fitness?.zone_distribution_this_week}
            prescribed={fitness?.zone_distribution_prescribed}
            isLoading={fitnessLoading}
          />
        </div>

        {/* Recent Activities */}
        <div className="bg-white rounded-2xl shadow-lg p-8 border border-gray-100">
          <div className="flex items-center justify-between mb-6">
            <div>
              <h2 className="text-2xl font-bold text-gray-900">Recent Activities</h2>
              <p className="text-sm text-gray-500 mt-1">Your latest workouts</p>
            </div>
            <Activity className="w-6 h-6 text-gray-400" />
          </div>
          <div className="space-y-3 max-h-[400px] overflow-y-auto pr-2">
            {recentActivitiesLoading ? (
              Array.from({ length: 5 }).map((_, idx) => (
                <div key={idx} className="h-20 bg-gradient-to-r from-gray-100 to-gray-50 animate-pulse rounded-xl" />
              ))
            ) : recentActivities?.activities?.length ? (
              recentActivities.activities.slice(0, 8).map((activity, idx) => (
                <div
                  key={activity.id}
                  onClick={() => handleActivityClick(activity.id)}
                  className="group relative overflow-hidden p-4 bg-gradient-to-r from-gray-50 to-white rounded-xl hover:from-indigo-50 hover:to-purple-50 transition-all duration-300 border border-gray-100 hover:border-indigo-200 hover:shadow-md cursor-pointer"
                  style={{ animationDelay: `${idx * 50}ms` }}
                >
                  <div className="flex items-center justify-between">
                    <div className="flex-1">
                      <div className="flex items-center gap-3 mb-2">
                        <div className="p-2 bg-gradient-to-br from-indigo-500 to-purple-600 rounded-lg">
                          <Footprints className="w-4 h-4 text-white" />
                        </div>
                        <div className="flex-1">
                          <div className="flex items-center gap-2">
                            <span className="font-semibold text-gray-900 capitalize">
                              {activity.workout_type}
                            </span>
                            <span className="text-xs text-gray-400 font-medium">
                              {new Date(activity.activity_date).toLocaleDateString('en-US', {
                                month: 'short',
                                day: 'numeric'
                              })}
                            </span>
                          </div>
                          <div className="flex items-center gap-4 mt-1">
                            <div className="flex items-center gap-1.5 text-sm text-gray-600">
                              <Footprints className="w-3.5 h-3.5 text-gray-400" />
                              <span className="font-medium">{formatKm(activity.distance_m)} km</span>
                            </div>
                            <div className="flex items-center gap-1.5 text-sm text-gray-600">
                              <Clock className="w-3.5 h-3.5 text-gray-400" />
                              <span>{formatDuration(activity.duration_s)}</span>
                            </div>
                            <div className="text-sm text-gray-600">
                              <span className="font-mono">{formatPace(activity.avg_pace_s)}</span>
                            </div>
                          </div>
                        </div>
                      </div>
                    </div>
                    {activity.tss && (
                      <div className="text-right ml-4">
                        <div className="px-3 py-1 bg-gradient-to-br from-indigo-500 to-purple-600 rounded-lg">
                          <div className="text-lg font-bold text-white">
                            {activity.tss.toFixed(0)}
                          </div>
                          <div className="text-xs text-indigo-100">TSS</div>
                        </div>
                      </div>
                    )}
                  </div>
                </div>
              ))
            ) : (
              <div className="text-center py-12">
                <div className="inline-flex items-center justify-center w-16 h-16 rounded-full bg-gradient-to-br from-gray-100 to-gray-50 mb-4">
                  <Activity className="w-8 h-8 text-gray-400" />
                </div>
                <p className="text-gray-500 font-medium">No activities yet</p>
                <p className="text-sm text-gray-400 mt-1">Connect Strava to sync your runs</p>
              </div>
            )}
          </div>
        </div>
      </div>

      {/* Activity Detail Modal */}
      {selectedActivityId && (
        <ActivityDetailModal
          activityId={selectedActivityId}
          isOpen={isModalOpen}
          onClose={handleCloseModal}
        />
      )}
    </div>
  );
}
