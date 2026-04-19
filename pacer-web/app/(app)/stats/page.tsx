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
    bg: 'from-emerald-500 to-green-600',
    text: 'text-white',
    badge: 'bg-emerald-100 text-emerald-700',
    label: 'Low Risk'
  };
  if (score < 65) return {
    bg: 'from-amber-500 to-orange-600',
    text: 'text-white',
    badge: 'bg-amber-100 text-amber-700',
    label: 'Moderate Risk'
  };
  return {
    bg: 'from-rose-500 to-red-600',
    text: 'text-white',
    badge: 'bg-rose-100 text-rose-700',
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
      <div className="relative overflow-hidden bg-gradient-to-br from-indigo-500 via-purple-500 to-pink-500 rounded-2xl p-8 text-white shadow-xl">
        <div className="absolute inset-0 bg-black/10"></div>
        <div className="relative z-10">
          <div className="flex items-center gap-3 mb-2">
            <Activity className="w-8 h-8" />
            <h1 className="text-4xl font-bold tracking-tight">Performance Analytics</h1>
          </div>
          <p className="text-indigo-100 text-lg">Deep insights into your training data</p>
        </div>
      </div>

      {error && (
        <div className="bg-rose-50 border border-rose-200 rounded-2xl p-4 flex items-center gap-3">
          <AlertTriangle className="w-5 h-5 text-rose-600" />
          <p className="text-sm text-rose-700">{error}</p>
        </div>
      )}

      {loading ? (
        <div className="grid grid-cols-1 gap-6 lg:grid-cols-3">
          {Array.from({ length: 6 }).map((_, i) => (
            <div key={i} className="h-64 bg-gradient-to-r from-gray-100 to-gray-50 animate-pulse rounded-2xl" />
          ))}
        </div>
      ) : (
        <>
          {/* Top Stats Row */}
          <div className="grid grid-cols-1 gap-6 lg:grid-cols-3">
            {/* Injury Risk Score */}
            <div className="group relative overflow-hidden bg-white rounded-2xl shadow-lg hover:shadow-xl transition-all duration-300 border border-gray-100">
              <div className={`absolute top-0 right-0 w-32 h-32 bg-gradient-to-br ${risk.bg} rounded-full -mr-16 -mt-16 opacity-20`}></div>
              <div className="relative p-6">
                <div className="flex items-center gap-3 mb-4">
                  <div className={`p-3 bg-gradient-to-br ${risk.bg} rounded-xl shadow-lg`}>
                    <AlertTriangle className="w-6 h-6 text-white" />
                  </div>
                  <div>
                    <h2 className="text-sm font-medium text-gray-500">Injury Risk</h2>
                    <p className="text-xs text-gray-400">Based on ACWR & Fatigue</p>
                  </div>
                </div>
                <div className="flex items-baseline gap-2 mb-3">
                  <div className={`text-5xl font-bold bg-gradient-to-br ${risk.bg} bg-clip-text text-transparent`}>
                    {(data?.injury_risk_score ?? 0).toFixed(0)}
                  </div>
                  <span className="text-2xl text-gray-400">/100</span>
                </div>
                <div className={`inline-flex items-center gap-1 px-3 py-1 rounded-full text-xs font-semibold ${risk.badge}`}>
                  {risk.label}
                </div>
                <div className="mt-4 w-full bg-gray-100 rounded-full h-2 overflow-hidden">
                  <div 
                    className={`h-full bg-gradient-to-r ${risk.bg} transition-all duration-1000 ease-out rounded-full`}
                    style={{ width: `${data?.injury_risk_score || 0}%` }}
                  />
                </div>
              </div>
            </div>

            {/* Zone Compliance Chart */}
            <div className="lg:col-span-2 bg-white rounded-2xl shadow-lg hover:shadow-xl transition-all duration-300 p-6 border border-gray-100">
              <div className="flex items-center gap-3 mb-4">
                <div className="p-3 bg-gradient-to-br from-blue-500 to-cyan-600 rounded-xl shadow-lg">
                  <Heart className="w-6 h-6 text-white" />
                </div>
                <div>
                  <h2 className="text-lg font-bold text-gray-900">Zone 2 Compliance</h2>
                  <p className="text-xs text-gray-500">Weekly actual vs. target</p>
                </div>
              </div>
              <div className="h-56">
                <ResponsiveContainer width="100%" height="100%">
                  <BarChart data={zoneRows}>
                    <defs>
                      <linearGradient id="actualGradient" x1="0" y1="0" x2="0" y2="1">
                        <stop offset="0%" stopColor="#0ea5e9" stopOpacity={0.9} />
                        <stop offset="100%" stopColor="#06b6d4" stopOpacity={0.6} />
                      </linearGradient>
                      <linearGradient id="targetGradient" x1="0" y1="0" x2="0" y2="1">
                        <stop offset="0%" stopColor="#94a3b8" stopOpacity={0.7} />
                        <stop offset="100%" stopColor="#cbd5e1" stopOpacity={0.5} />
                      </linearGradient>
                    </defs>
                    <CartesianGrid strokeDasharray="3 3" stroke="#e5e7eb" vertical={false} />
                    <XAxis dataKey="week" tick={{ fontSize: 12 }} stroke="#9ca3af" />
                    <YAxis tick={{ fontSize: 12 }} stroke="#9ca3af" />
                    <Tooltip 
                      contentStyle={{ 
                        backgroundColor: 'white', 
                        border: '1px solid #e5e7eb', 
                        borderRadius: '12px',
                        boxShadow: '0 4px 6px -1px rgb(0 0 0 / 0.1)'
                      }} 
                    />
                    <Legend wrapperStyle={{ fontSize: '12px' }} />
                    <Bar dataKey="actualZ2" fill="url(#actualGradient)" name="Actual" radius={[8, 8, 0, 0]} />
                    <Bar dataKey="targetZ2" fill="url(#targetGradient)" name="Target" radius={[8, 8, 0, 0]} />
                  </BarChart>
                </ResponsiveContainer>
              </div>
            </div>
          </div>

          {/* Monthly Mileage & Running Economy */}
          <div className="grid grid-cols-1 gap-6 lg:grid-cols-2">
            {/* Monthly Mileage */}
            <div className="bg-white rounded-2xl shadow-lg hover:shadow-xl transition-all duration-300 p-6 border border-gray-100">
              <div className="flex items-center gap-3 mb-4">
                <div className="p-3 bg-gradient-to-br from-purple-500 to-pink-600 rounded-xl shadow-lg">
                  <Calendar className="w-6 h-6 text-white" />
                </div>
                <div>
                  <h2 className="text-lg font-bold text-gray-900">Monthly Mileage</h2>
                  <p className="text-xs text-gray-500">Total distance per month</p>
                </div>
              </div>
              <div className="h-56">
                <ResponsiveContainer width="100%" height="100%">
                  <BarChart data={data?.monthly_mileage ?? []}>
                    <defs>
                      <linearGradient id="mileageGradient" x1="0" y1="0" x2="0" y2="1">
                        <stop offset="0%" stopColor="#a855f7" stopOpacity={0.9} />
                        <stop offset="100%" stopColor="#ec4899" stopOpacity={0.7} />
                      </linearGradient>
                    </defs>
                    <CartesianGrid strokeDasharray="3 3" stroke="#e5e7eb" vertical={false} />
                    <XAxis dataKey="month" tick={{ fontSize: 12 }} stroke="#9ca3af" />
                    <YAxis tick={{ fontSize: 12 }} stroke="#9ca3af" />
                    <Tooltip 
                      contentStyle={{ 
                        backgroundColor: 'white', 
                        border: '1px solid #e5e7eb', 
                        borderRadius: '12px',
                        boxShadow: '0 4px 6px -1px rgb(0 0 0 / 0.1)'
                      }} 
                    />
                    <Bar dataKey="mileage_km" fill="url(#mileageGradient)" radius={[8, 8, 0, 0]} />
                  </BarChart>
                </ResponsiveContainer>
              </div>
            </div>

            {/* Running Economy */}
            <div className="bg-white rounded-2xl shadow-lg hover:shadow-xl transition-all duration-300 p-6 border border-gray-100">
              <div className="flex items-center gap-3 mb-4">
                <div className="p-3 bg-gradient-to-br from-emerald-500 to-green-600 rounded-xl shadow-lg">
                  <TrendingUp className="w-6 h-6 text-white" />
                </div>
                <div>
                  <h2 className="text-lg font-bold text-gray-900">Running Economy</h2>
                  <p className="text-xs text-gray-500">Last 8 weeks</p>
                </div>
              </div>
              <div className="h-56">
                <ResponsiveContainer width="100%" height="100%">
                  <LineChart data={data?.running_economy ?? []}>
                    <CartesianGrid strokeDasharray="3 3" stroke="#e5e7eb" vertical={false} />
                    <XAxis 
                      dataKey="week_start" 
                      tickFormatter={(value) => new Date(value).toLocaleDateString(undefined, { month: 'short', day: 'numeric' })} 
                      tick={{ fontSize: 12 }}
                      stroke="#9ca3af"
                    />
                    <YAxis yAxisId="left" tick={{ fontSize: 12 }} stroke="#9ca3af" />
                    <YAxis yAxisId="right" orientation="right" tick={{ fontSize: 12 }} stroke="#9ca3af" />
                    <Tooltip 
                      contentStyle={{ 
                        backgroundColor: 'white', 
                        border: '1px solid #e5e7eb', 
                        borderRadius: '12px',
                        boxShadow: '0 4px 6px -1px rgb(0 0 0 / 0.1)'
                      }} 
                    />
                    <Legend wrapperStyle={{ fontSize: '12px' }} />
                    <Line yAxisId="left" type="monotone" dataKey="cadence" stroke="#10b981" strokeWidth={3} dot={{ fill: '#10b981', r: 4 }} name="Cadence" />
                    <Line yAxisId="right" type="monotone" dataKey="gct" stroke="#f97316" strokeWidth={3} dot={{ fill: '#f97316', r: 4 }} name="GCT" />
                  </LineChart>
                </ResponsiveContainer>
              </div>
            </div>
          </div>

          {/* Semantic Facts */}
          <div className="bg-white rounded-2xl shadow-lg p-8 border border-gray-100">
            <div className="mb-6">
              <h2 className="text-2xl font-bold text-gray-900 mb-2">Training Insights</h2>
              <p className="text-sm text-gray-500">What Pacer knows about you</p>
            </div>
            <div className="space-y-4">
              {(data?.semantic_facts ?? []).map((fact, idx) => (
                <div 
                  key={fact.fact_key}
                  className="group p-4 bg-gradient-to-r from-gray-50 to-white rounded-xl hover:from-indigo-50 hover:to-purple-50 transition-all duration-300 border border-gray-100 hover:border-indigo-200"
                  style={{ animationDelay: `${idx * 50}ms` }}
                >
                  <div className="flex items-center justify-between mb-2">
                    <p className="font-medium text-gray-900">{fact.notes}</p>
                    <div className="flex items-center gap-2">
                      <div className="text-xs font-semibold text-gray-500">{Math.round(fact.confidence * 100)}%</div>
                      <div className={`px-2 py-1 rounded-full text-xs font-semibold ${
                        fact.confidence >= 0.8 ? 'bg-emerald-100 text-emerald-700' :
                        fact.confidence >= 0.6 ? 'bg-blue-100 text-blue-700' :
                        'bg-amber-100 text-amber-700'
                      }`}>
                        {fact.confidence >= 0.8 ? 'High' : fact.confidence >= 0.6 ? 'Medium' : 'Low'}
                      </div>
                    </div>
                  </div>
                  <div className="h-2 w-full rounded-full bg-gray-100 overflow-hidden">
                    <div 
                      className="h-2 rounded-full bg-gradient-to-r from-indigo-500 to-purple-600 transition-all duration-1000 ease-out" 
                      style={{ width: `${Math.max(4, fact.confidence * 100)}%` }} 
                    />
                  </div>
                </div>
              ))}
              {!data?.semantic_facts?.length && (
                <div className="text-center py-12">
                  <div className="inline-flex items-center justify-center w-16 h-16 rounded-full bg-gradient-to-br from-gray-100 to-gray-50 mb-4">
                    <Activity className="w-8 h-8 text-gray-400" />
                  </div>
                  <p className="text-gray-500 font-medium">No insights available yet</p>
                  <p className="text-sm text-gray-400 mt-1">Keep training to generate insights</p>
                </div>
              )}
            </div>
          </div>
        </>
      )}
    </div>
  );
}
