'use client';

import { useEffect, useMemo, useState } from 'react';
import {
  Bar,
  BarChart,
  CartesianGrid,
  Legend,
  Line,
  LineChart,
  ResponsiveContainer,
  Tooltip,
  XAxis,
  YAxis,
} from 'recharts';
import { TrendingUp, AlertTriangle, Heart, Activity, Calendar } from 'lucide-react';

const API_BASE_URL = process.env.NEXT_PUBLIC_API_URL || 'http://localhost:3001/api/v1';

interface StatsResponse {
  injury_risk_score: number;
  zone_compliance: Array<{
    week_start: string;
    actual: { z1: number; z2: number; z3: number; z4: number; z5: number };
    target: { z1: number; z2: number; z3: number; z4: number; z5: number };
  }>;
  monthly_mileage: Array<{ month: string; mileage_km: number }>;
  running_economy: Array<{ week_start: string; cadence: number; gct: number }>;
  semantic_facts: Array<{ fact_key: string; notes: string; confidence: number }>;
}

function riskTone(score: number) {
  if (score < 35) return {
    color: '#10b981',
    badge: 'bg-green-900/50 text-green-400 border border-green-800',
    label: 'Low Risk'
  };
  if (score < 65) return {
    color: '#f59e0b',
    badge: 'bg-amber-900/50 text-amber-400 border border-amber-800',
    label: 'Moderate Risk'
  };
  return {
    color: '#ef4444',
    badge: 'bg-red-900/50 text-red-400 border border-red-800',
    label: 'High Risk'
  };
}

export default function StatsPage() {
  const [data, setData] = useState<StatsResponse | null>(null);
  const [error, setError] = useState<string | null>(null);
  const [loading, setLoading] = useState(true);

  useEffect(() => {
    let cancelled = false;
    async function fetchStats() {
      try {
        setLoading(true);
        const token = localStorage.getItem('pacer_token');
        if (!token) {
          setError('Missing auth token.');
          return;
        }

        const response = await fetch(`${API_BASE_URL}/stats/overview`, {
          headers: { Authorization: `Bearer ${token}` },
        });
        if (!response.ok) throw new Error('Failed to fetch analytics');

        const payload = (await response.json()) as StatsResponse;
        if (!cancelled) {
          setData(payload);
        }
      } catch (statsError) {
        if (!cancelled) {
          setError(statsError instanceof Error ? statsError.message : 'Failed to load stats');
        }
      } finally {
        if (!cancelled) {
          setLoading(false);
        }
      }
    }
    void fetchStats();
    return () => {
      cancelled = true;
    };
  }, []);

  const zoneRows = useMemo(
    () =>
      (data?.zone_compliance ?? []).map((row) => ({
        week: new Date(row.week_start).toLocaleDateString(undefined, { month: 'short', day: 'numeric' }),
        actualZ2: row.actual.z2,
        targetZ2: row.target.z2,
      })),
    [data?.zone_compliance],
  );

  const risk = riskTone(data?.injury_risk_score ?? 0);

  return (
    <div className="space-y-8 max-w-7xl mx-auto">
      {/* Header */}
      <div className="relative overflow-hidden bg-[#161b26] rounded-lg p-8 border border-gray-800">
        <div className="relative z-10">
          <div className="flex items-center gap-3 mb-2">
            <Activity className="w-8 h-8 text-blue-500" />
            <h1 className="text-4xl font-bold tracking-tight text-gray-100">Performance Analytics</h1>
          </div>
          <p className="text-gray-400 text-lg">Deep insights into your training data</p>
        </div>
      </div>

      {error && (
        <div className="bg-red-900/50 border border-red-800 rounded-lg p-4 flex items-center gap-3">
          <AlertTriangle className="w-5 h-5 text-red-400" />
          <p className="text-sm text-red-400">{error}</p>
        </div>
      )}

      {loading ? (
        <div className="grid grid-cols-1 gap-6 lg:grid-cols-3">
          {Array.from({ length: 6 }).map((_, i) => (
            <div key={i} className="h-64 bg-[#1e2530] animate-pulse rounded-lg" />
          ))}
        </div>
      ) : (
        <>
          {/* Top Stats Row */}
          <div className="grid grid-cols-1 gap-6 lg:grid-cols-3">
            {/* Injury Risk Score */}
            <div className="group relative overflow-hidden bg-[#161b26] rounded-lg border border-gray-800">
              <div className="relative p-6">
                <div className="flex items-center gap-3 mb-4">
                  <div className="p-3 rounded-lg" style={{ backgroundColor: risk.color + '20' }}>
                    <AlertTriangle className="w-6 h-6" style={{ color: risk.color }} />
                  </div>
                  <div>
                    <h2 className="text-sm font-medium text-gray-400">Injury Risk</h2>
                    <p className="text-xs text-gray-500">Based on ACWR & Fatigue</p>
                  </div>
                </div>
                <div className="flex items-baseline gap-2 mb-3">
                  <div className="text-5xl font-bold" style={{ color: risk.color }}>
                    {(data?.injury_risk_score ?? 0).toFixed(0)}
                  </div>
                  <span className="text-2xl text-gray-500">/100</span>
                </div>
                <div className={`inline-flex items-center gap-1 px-3 py-1 rounded text-xs font-semibold ${risk.badge}`}>
                  {risk.label}
                </div>
                <div className="mt-4 w-full bg-gray-800 rounded-full h-2 overflow-hidden">
                  <div 
                    className="h-full transition-all duration-1000 ease-out rounded-full"
                    style={{ width: `${data?.injury_risk_score || 0}%`, backgroundColor: risk.color }}
                  />
                </div>
              </div>
            </div>

            {/* Zone Compliance Chart */}
            <div className="lg:col-span-2 bg-[#161b26] rounded-lg border border-gray-800 p-6">
              <div className="flex items-center gap-3 mb-4">
                <div className="p-3 bg-blue-900/50 rounded-lg border border-blue-800">
                  <Heart className="w-6 h-6 text-blue-500" />
                </div>
                <div>
                  <h2 className="text-lg font-bold text-gray-100">Zone 2 Compliance</h2>
                  <p className="text-xs text-gray-500">Weekly actual vs. target</p>
                </div>
              </div>
              <div className="h-56">
                <ResponsiveContainer width="100%" height="100%">
                  <BarChart data={zoneRows}>
                    <CartesianGrid strokeDasharray="3 3" stroke="#374151" vertical={false} />
                    <XAxis dataKey="week" tick={{ fontSize: 12, fill: '#9ca3af' }} stroke="#6b7280" />
                    <YAxis tick={{ fontSize: 12, fill: '#9ca3af' }} stroke="#6b7280" />
                    <Tooltip 
                      contentStyle={{ 
                        backgroundColor: '#161b26', 
                        border: '1px solid #374151', 
                        borderRadius: '8px',
                        color: '#e5e7eb'
                      }} 
                    />
                    <Legend wrapperStyle={{ fontSize: '12px', color: '#9ca3af' }} />
                    <Bar dataKey="actualZ2" fill="#3b82f6" name="Actual" radius={[4, 4, 0, 0]} />
                    <Bar dataKey="targetZ2" fill="#6b7280" name="Target" radius={[4, 4, 0, 0]} />
                  </BarChart>
                </ResponsiveContainer>
              </div>
            </div>
          </div>

          {/* Monthly Mileage & Running Economy */}
          <div className="grid grid-cols-1 gap-6 lg:grid-cols-2">
            {/* Monthly Mileage */}
            <div className="bg-[#161b26] rounded-lg border border-gray-800 p-6">
              <div className="flex items-center gap-3 mb-4">
                <div className="p-3 bg-blue-900/50 rounded-lg border border-blue-800">
                  <Calendar className="w-6 h-6 text-blue-500" />
                </div>
                <div>
                  <h2 className="text-lg font-bold text-gray-100">Monthly Mileage</h2>
                  <p className="text-xs text-gray-500">Total distance per month</p>
                </div>
              </div>
              <div className="h-56">
                <ResponsiveContainer width="100%" height="100%">
                  <BarChart data={data?.monthly_mileage ?? []}>
                    <CartesianGrid strokeDasharray="3 3" stroke="#374151" vertical={false} />
                    <XAxis dataKey="month" tick={{ fontSize: 12, fill: '#9ca3af' }} stroke="#6b7280" />
                    <YAxis tick={{ fontSize: 12, fill: '#9ca3af' }} stroke="#6b7280" />
                    <Tooltip 
                      contentStyle={{ 
                        backgroundColor: '#161b26', 
                        border: '1px solid #374151', 
                        borderRadius: '8px',
                        color: '#e5e7eb'
                      }} 
                    />
                    <Bar dataKey="mileage_km" fill="#3b82f6" radius={[4, 4, 0, 0]} />
                  </BarChart>
                </ResponsiveContainer>
              </div>
            </div>

            {/* Running Economy */}
            <div className="bg-[#161b26] rounded-lg border border-gray-800 p-6">
              <div className="flex items-center gap-3 mb-4">
                <div className="p-3 bg-green-900/50 rounded-lg border border-green-800">
                  <TrendingUp className="w-6 h-6 text-green-500" />
                </div>
                <div>
                  <h2 className="text-lg font-bold text-gray-100">Running Economy</h2>
                  <p className="text-xs text-gray-500">Last 8 weeks</p>
                </div>
              </div>
              <div className="h-56">
                <ResponsiveContainer width="100%" height="100%">
                  <LineChart data={data?.running_economy ?? []}>
                    <CartesianGrid strokeDasharray="3 3" stroke="#374151" vertical={false} />
                    <XAxis 
                      dataKey="week_start" 
                      tickFormatter={(value) => new Date(value).toLocaleDateString(undefined, { month: 'short', day: 'numeric' })} 
                      tick={{ fontSize: 12, fill: '#9ca3af' }}
                      stroke="#6b7280"
                    />
                    <YAxis yAxisId="left" tick={{ fontSize: 12, fill: '#9ca3af' }} stroke="#6b7280" />
                    <YAxis yAxisId="right" orientation="right" tick={{ fontSize: 12, fill: '#9ca3af' }} stroke="#6b7280" />
                    <Tooltip 
                      contentStyle={{ 
                        backgroundColor: '#161b26', 
                        border: '1px solid #374151', 
                        borderRadius: '8px',
                        color: '#e5e7eb'
                      }} 
                    />
                    <Legend wrapperStyle={{ fontSize: '12px', color: '#9ca3af' }} />
                    <Line yAxisId="left" type="monotone" dataKey="cadence" stroke="#10b981" strokeWidth={3} dot={{ fill: '#10b981', r: 4 }} name="Cadence" />
                    <Line yAxisId="right" type="monotone" dataKey="gct" stroke="#f59e0b" strokeWidth={3} dot={{ fill: '#f59e0b', r: 4 }} name="GCT" />
                  </LineChart>
                </ResponsiveContainer>
              </div>
            </div>
          </div>

          {/* Semantic Facts */}
          <div className="bg-[#161b26] rounded-lg border border-gray-800 p-8">
            <div className="mb-6">
              <h2 className="text-2xl font-bold text-gray-100 mb-2">Training Insights</h2>
              <p className="text-sm text-gray-500">What Pacer knows about you</p>
            </div>
            <div className="space-y-4">
              {(data?.semantic_facts ?? []).map((fact, idx) => (
                <div 
                  key={fact.fact_key}
                  className="group p-4 bg-[#1e2530] rounded-lg hover:bg-[#1e2530]/80 transition-all duration-300 border border-gray-800"
                  style={{ animationDelay: `${idx * 50}ms` }}
                >
                  <div className="flex items-center justify-between mb-2">
                    <p className="font-medium text-gray-100">{fact.notes}</p>
                    <div className="flex items-center gap-2">
                      <div className="text-xs font-semibold text-gray-500">{Math.round(fact.confidence * 100)}%</div>
                      <div className={`px-2 py-1 rounded text-xs font-semibold ${
                        fact.confidence >= 0.8 ? 'bg-green-900/50 text-green-400 border border-green-800' :
                        fact.confidence >= 0.6 ? 'bg-blue-900/50 text-blue-400 border border-blue-800' :
                        'bg-amber-900/50 text-amber-400 border border-amber-800'
                      }`}>
                        {fact.confidence >= 0.8 ? 'High' : fact.confidence >= 0.6 ? 'Medium' : 'Low'}
                      </div>
                    </div>
                  </div>
                  <div className="h-2 w-full rounded-full bg-gray-800 overflow-hidden">
                    <div 
                      className="h-2 rounded-full bg-blue-600 transition-all duration-1000 ease-out" 
                      style={{ width: `${Math.max(4, fact.confidence * 100)}%` }} 
                    />
                  </div>
                </div>
              ))}
              {!data?.semantic_facts?.length && (
                <div className="text-center py-12">
                  <div className="inline-flex items-center justify-center w-16 h-16 rounded-full bg-[#1e2530] mb-4">
                    <Activity className="w-8 h-8 text-gray-500" />
                  </div>
                  <p className="text-gray-400 font-medium">No insights available yet</p>
                  <p className="text-sm text-gray-500 mt-1">Keep training to generate insights</p>
                </div>
              )}
            </div>
          </div>
        </>
      )}
    </div>
  );
}
