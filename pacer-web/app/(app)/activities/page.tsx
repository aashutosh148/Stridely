'use client';

import { useState, useEffect, useMemo } from 'react';
import { 
  Calendar, 
  Clock, 
  Footprints, 
  Heart, 
  TrendingUp, 
  Filter,
  Search,
  ChevronLeft,
  ChevronRight,
  X
} from 'lucide-react';
import { ActivityDetailModal } from '../../../components/activity-detail-modal';

const API_BASE_URL = process.env.NEXT_PUBLIC_API_URL || 'http://localhost:3001/api/v1';

interface Activity {
  id: string;
  activity_date: string;
  workout_type: string;
  distance_m: number;
  duration_s: number;
  avg_pace_s: number;
  avg_hr: number;
  max_hr: number;
  tss: number;
  elevation_gain_m: number;
}

interface ActivitiesResponse {
  activities: Activity[];
  total: number;
  page: number;
  limit: number;
  total_pages: number;
}

export default function ActivitiesPage() {
  const [activities, setActivities] = useState<Activity[]>([]);
  const [loading, setLoading] = useState(true);
  const [page, setPage] = useState(1);
  const [totalPages, setTotalPages] = useState(1);
  const [total, setTotal] = useState(0);
  const limit = 15;

  // Filters
  const [workoutTypeFilter, setWorkoutTypeFilter] = useState<string>('all');
  const [searchQuery, setSearchQuery] = useState('');
  const [dateFilter, setDateFilter] = useState<string>('all'); // all, week, month, 3months

  // Modal state
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

  useEffect(() => {
    fetchActivities();
  }, [page, workoutTypeFilter, dateFilter]);

  const fetchActivities = async () => {
    try {
      setLoading(true);
      const token = localStorage.getItem('pacer_token');
      if (!token) return;

      // Build query params
      const params = new URLSearchParams({
        page: page.toString(),
        limit: limit.toString(),
      });

      if (workoutTypeFilter && workoutTypeFilter !== 'all') {
        params.append('workout_type', workoutTypeFilter);
      }

      if (dateFilter && dateFilter !== 'all') {
        const now = new Date();
        let daysAgo = 0;
        switch (dateFilter) {
          case 'week': daysAgo = 7; break;
          case 'month': daysAgo = 30; break;
          case '3months': daysAgo = 90; break;
        }
        if (daysAgo > 0) {
          const startDate = new Date(now.getTime() - daysAgo * 24 * 60 * 60 * 1000);
          params.append('start_date', startDate.toISOString().split('T')[0]);
        }
      }

      const response = await fetch(`${API_BASE_URL}/activities?${params.toString()}`, {
        headers: { Authorization: `Bearer ${token}` },
      });

      if (!response.ok) throw new Error('Failed to fetch activities');

      const data: ActivitiesResponse = await response.json();
      setActivities(data.activities || []);
      setTotal(data.total || 0);
      setTotalPages(data.total_pages || 1);
    } catch (error) {
      console.error('Error fetching activities:', error);
    } finally {
      setLoading(false);
    }
  };

  // Local search filter
  const filteredActivities = useMemo(() => {
    if (!searchQuery) return activities;
    const query = searchQuery.toLowerCase();
    return activities.filter(activity => 
      activity.workout_type.toLowerCase().includes(query) ||
      new Date(activity.activity_date).toLocaleDateString().includes(query)
    );
  }, [activities, searchQuery]);

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

  const workoutTypes = ['all', 'run', 'race', 'long_run', 'easy', 'tempo', 'interval'];

  return (
    <div className="space-y-6 max-w-7xl mx-auto">
      {/* Header */}
      <div className="flex items-center justify-between">
        <div>
          <h1 className="text-3xl font-bold text-gray-100">
            Activities
          </h1>
          <p className="text-gray-400 mt-1">
            {total} total {total === 1 ? 'activity' : 'activities'}
          </p>
        </div>
      </div>

      {/* Filters */}
      <div className="bg-[#161b26] rounded-lg shadow-lg p-6 border border-gray-800">
        <div className="flex items-center gap-2 mb-4">
          <Filter className="w-5 h-5 text-gray-400" />
          <h2 className="font-semibold text-gray-100">Filters</h2>
        </div>
        
        <div className="grid grid-cols-1 md:grid-cols-3 gap-4">
          {/* Search */}
          <div className="relative">
            <Search className="absolute left-3 top-1/2 -translate-y-1/2 w-4 h-4 text-gray-500" />
            <input
              type="text"
              placeholder="Search activities..."
              value={searchQuery}
              onChange={(e) => setSearchQuery(e.target.value)}
              className="w-full pl-10 pr-4 py-2.5 bg-[#1e2530] border border-gray-800 rounded-lg text-gray-100 placeholder-gray-500 focus:outline-none focus:ring-2 focus:ring-blue-600 focus:border-transparent transition-all"
            />
            {searchQuery && (
              <button
                onClick={() => setSearchQuery('')}
                className="absolute right-3 top-1/2 -translate-y-1/2 text-gray-500 hover:text-gray-300"
              >
                <X className="w-4 h-4" />
              </button>
            )}
          </div>

          {/* Workout Type Filter */}
          <select
            value={workoutTypeFilter}
            onChange={(e) => { setWorkoutTypeFilter(e.target.value); setPage(1); }}
            className="px-4 py-2.5 bg-[#1e2530] border border-gray-800 rounded-lg text-gray-100 focus:outline-none focus:ring-2 focus:ring-blue-600 focus:border-transparent transition-all capitalize"
          >
            {workoutTypes.map(type => (
              <option key={type} value={type} className="capitalize bg-[#1e2530]">
                {type === 'all' ? 'All Types' : type.replace('_', ' ')}
              </option>
            ))}
          </select>

          {/* Date Range Filter */}
          <select
            value={dateFilter}
            onChange={(e) => { setDateFilter(e.target.value); setPage(1); }}
            className="px-4 py-2.5 bg-[#1e2530] border border-gray-800 rounded-lg text-gray-100 focus:outline-none focus:ring-2 focus:ring-blue-600 focus:border-transparent transition-all"
          >
            <option value="all" className="bg-[#1e2530]">All Time</option>
            <option value="week" className="bg-[#1e2530]">Last 7 Days</option>
            <option value="month" className="bg-[#1e2530]">Last 30 Days</option>
            <option value="3months" className="bg-[#1e2530]">Last 3 Months</option>
          </select>
        </div>
      </div>

      {/* Activities List */}
      <div className="bg-[#161b26] rounded-lg shadow-lg border border-gray-800 overflow-hidden">
        {loading ? (
          <div className="p-8 space-y-4">
            {Array.from({ length: 5 }).map((_, i) => (
              <div key={i} className="h-24 bg-[#1e2530] animate-pulse rounded-lg" />
            ))}
          </div>
        ) : filteredActivities.length === 0 ? (
          <div className="text-center py-16">
            <div className="inline-flex items-center justify-center w-16 h-16 rounded-full bg-[#1e2530] mb-4">
              <Footprints className="w-8 h-8 text-gray-500" />
            </div>
            <h3 className="text-lg font-semibold text-gray-100 mb-2">No activities found</h3>
            <p className="text-gray-400">Try adjusting your filters or connect Strava to sync activities</p>
          </div>
        ) : (
          <div className="divide-y divide-gray-800">
            {filteredActivities.map((activity, idx) => (
              <div
                key={activity.id}
                onClick={() => handleActivityClick(activity.id)}
                className="p-6 hover:bg-[#1e2530] transition-all duration-200 cursor-pointer group"
                style={{ animationDelay: `${idx * 30}ms` }}
              >
                <div className="flex items-center justify-between">
                  <div className="flex-1">
                    <div className="flex items-center gap-4 mb-3">
                      <div className="p-3 bg-blue-600 rounded-lg group-hover:bg-blue-500 transition-colors">
                        <Footprints className="w-5 h-5 text-white" />
                      </div>
                      <div>
                        <h3 className="font-semibold text-lg text-gray-100 capitalize flex items-center gap-2">
                          {activity.workout_type.replace('_', ' ')}
                          {activity.tss >= 100 && (
                            <span className="px-2 py-0.5 bg-red-900/50 text-red-400 text-xs font-semibold rounded border border-red-800">
                              High Load
                            </span>
                          )}
                        </h3>
                        <p className="text-sm text-gray-400 flex items-center gap-2">
                          <Calendar className="w-3.5 h-3.5" />
                          {new Date(activity.activity_date).toLocaleDateString('en-US', {
                            weekday: 'long',
                            year: 'numeric',
                            month: 'long',
                            day: 'numeric'
                          })}
                        </p>
                      </div>
                    </div>

                    <div className="grid grid-cols-2 md:grid-cols-6 gap-4 pl-16">
                      <div className="flex items-center gap-2">
                        <Footprints className="w-4 h-4 text-gray-500" />
                        <div>
                          <div className="text-sm font-semibold text-gray-100">{formatDistance(activity.distance_m)} km</div>
                          <div className="text-xs text-gray-500">Distance</div>
                        </div>
                      </div>

                      <div className="flex items-center gap-2">
                        <Clock className="w-4 h-4 text-gray-500" />
                        <div>
                          <div className="text-sm font-semibold text-gray-100 font-mono">{formatDuration(activity.duration_s)}</div>
                          <div className="text-xs text-gray-500">Duration</div>
                        </div>
                      </div>

                      <div className="flex items-center gap-2">
                        <TrendingUp className="w-4 h-4 text-gray-500" />
                        <div>
                          <div className="text-sm font-semibold text-gray-100 font-mono">{formatPace(activity.avg_pace_s)}</div>
                          <div className="text-xs text-gray-500">Avg Pace</div>
                        </div>
                      </div>

                      {activity.avg_hr > 0 && (
                        <div className="flex items-center gap-2">
                          <Heart className="w-4 h-4 text-red-500" />
                          <div>
                            <div className="text-sm font-semibold text-gray-100">{activity.avg_hr} bpm</div>
                            <div className="text-xs text-gray-500">Avg HR</div>
                          </div>
                        </div>
                      )}

                      {activity.elevation_gain_m > 0 && (
                        <div className="flex items-center gap-2">
                          <TrendingUp className="w-4 h-4 text-green-500" />
                          <div>
                            <div className="text-sm font-semibold text-gray-100">{Math.round(activity.elevation_gain_m)} m</div>
                            <div className="text-xs text-gray-500">Elevation</div>
                          </div>
                        </div>
                      )}

                      {activity.tss > 0 && (
                        <div className="flex items-center gap-2">
                          <div className="w-4 h-4 flex items-center justify-center">
                            <div className="w-2 h-2 bg-blue-500 rounded-full"></div>
                          </div>
                          <div>
                            <div className="text-sm font-semibold text-gray-100">{activity.tss.toFixed(0)}</div>
                            <div className="text-xs text-gray-500">TSS</div>
                          </div>
                        </div>
                      )}
                    </div>
                  </div>
                </div>
              </div>
            ))}
          </div>
        )}
      </div>

      {/* Pagination */}
      {totalPages > 1 && (
        <div className="flex items-center justify-between bg-[#161b26] rounded-lg shadow-lg p-4 border border-gray-800">
          <div className="text-sm text-gray-400">
            Page {page} of {totalPages}
          </div>
          <div className="flex items-center gap-2">
            <button
              onClick={() => setPage(p => Math.max(1, p - 1))}
              disabled={page === 1}
              className="p-2 rounded-lg border border-gray-800 hover:bg-[#1e2530] disabled:opacity-50 disabled:cursor-not-allowed transition-all text-gray-400"
            >
              <ChevronLeft className="w-5 h-5" />
            </button>
            
            <div className="flex items-center gap-1">
              {Array.from({ length: Math.min(5, totalPages) }, (_, i) => {
                let pageNum;
                if (totalPages <= 5) {
                  pageNum = i + 1;
                } else if (page <= 3) {
                  pageNum = i + 1;
                } else if (page >= totalPages - 2) {
                  pageNum = totalPages - 4 + i;
                } else {
                  pageNum = page - 2 + i;
                }
                
                return (
                  <button
                    key={i}
                    onClick={() => setPage(pageNum)}
                    className={`w-10 h-10 rounded-lg font-medium transition-all ${
                      page === pageNum
                        ? 'bg-blue-600 text-white'
                        : 'hover:bg-[#1e2530] text-gray-400'
                    }`}
                  >
                    {pageNum}
                  </button>
                );
              })}
            </div>

            <button
              onClick={() => setPage(p => Math.min(totalPages, p + 1))}
              disabled={page === totalPages}
              className="p-2 rounded-lg border border-gray-800 hover:bg-[#1e2530] disabled:opacity-50 disabled:cursor-not-allowed transition-all text-gray-400"
            >
              <ChevronRight className="w-5 h-5" />
            </button>
          </div>
        </div>
      )}

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
