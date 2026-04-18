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
  if (score < 35) return 'bg-emerald-100 text-emerald-800';
  if (score < 65) return 'bg-amber-100 text-amber-800';
  return 'bg-rose-100 text-rose-800';
}

export default function StatsPage() {
  const [data, setData] = useState<StatsResponse | null>(null);
  const [error, setError] = useState<string | null>(null);

  useEffect(() => {
    let cancelled = false;
    async function fetchStats() {
      try {
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

  return (
    <div className="space-y-6">
      <h1 className="text-2xl font-semibold text-gray-900">Performance Analytics</h1>

      {error ? <p className="text-sm text-rose-600">{error}</p> : null}

      <div className="grid grid-cols-1 gap-6 lg:grid-cols-3">
        <div className="rounded-xl border border-gray-200 bg-white p-5 shadow-sm lg:col-span-1">
          <h2 className="text-sm font-semibold text-gray-900">Injury Risk Score</h2>
          <div className={`mt-3 inline-flex rounded-full px-3 py-1 text-sm font-semibold ${riskTone(data?.injury_risk_score ?? 0)}`}>
            {(data?.injury_risk_score ?? 0).toFixed(0)} / 100
          </div>
          <p className="mt-2 text-sm text-gray-600">Derived from ACWR and freshness trend.</p>
        </div>

        <div className="rounded-xl border border-gray-200 bg-white p-5 shadow-sm lg:col-span-2">
          <h2 className="text-sm font-semibold text-gray-900">Weekly Zone Compliance (Z2)</h2>
          <div className="mt-3 h-56">
            <ResponsiveContainer width="100%" height="100%">
              <BarChart data={zoneRows}>
                <CartesianGrid strokeDasharray="3 3" stroke="#E5E7EB" />
                <XAxis dataKey="week" />
                <YAxis />
                <Tooltip />
                <Legend />
                <Bar dataKey="actualZ2" fill="#0284c7" name="Actual" radius={[4, 4, 0, 0]} />
                <Bar dataKey="targetZ2" fill="#94a3b8" name="Target" radius={[4, 4, 0, 0]} />
              </BarChart>
            </ResponsiveContainer>
          </div>
        </div>
      </div>

      <div className="grid grid-cols-1 gap-6 lg:grid-cols-2">
        <div className="rounded-xl border border-gray-200 bg-white p-5 shadow-sm">
          <h2 className="text-sm font-semibold text-gray-900">Monthly Mileage</h2>
          <div className="mt-3 h-56">
            <ResponsiveContainer width="100%" height="100%">
              <BarChart data={data?.monthly_mileage ?? []}>
                <CartesianGrid strokeDasharray="3 3" stroke="#E5E7EB" />
                <XAxis dataKey="month" />
                <YAxis />
                <Tooltip />
                <Bar dataKey="mileage_km" fill="#2563eb" radius={[6, 6, 0, 0]} />
              </BarChart>
            </ResponsiveContainer>
          </div>
        </div>

        <div className="rounded-xl border border-gray-200 bg-white p-5 shadow-sm">
          <h2 className="text-sm font-semibold text-gray-900">Running Economy (8 Weeks)</h2>
          <div className="mt-3 h-56">
            <ResponsiveContainer width="100%" height="100%">
              <LineChart data={data?.running_economy ?? []}>
                <CartesianGrid strokeDasharray="3 3" stroke="#E5E7EB" />
                <XAxis dataKey="week_start" tickFormatter={(value) => new Date(value).toLocaleDateString(undefined, { month: 'short', day: 'numeric' })} />
                <YAxis yAxisId="left" />
                <YAxis yAxisId="right" orientation="right" />
                <Tooltip />
                <Legend />
                <Line yAxisId="left" type="monotone" dataKey="cadence" stroke="#16a34a" strokeWidth={2} dot={false} />
                <Line yAxisId="right" type="monotone" dataKey="gct" stroke="#f97316" strokeWidth={2} dot={false} />
              </LineChart>
            </ResponsiveContainer>
          </div>
        </div>
      </div>

      <div className="rounded-xl border border-gray-200 bg-white p-5 shadow-sm">
        <h2 className="text-sm font-semibold text-gray-900">What Pacer knows about you</h2>
        <div className="mt-4 space-y-3">
          {(data?.semantic_facts ?? []).map((fact) => (
            <div key={fact.fact_key}>
              <div className="flex items-center justify-between text-sm">
                <p className="font-medium text-gray-800">{fact.notes}</p>
                <p className="text-xs text-gray-500">{Math.round(fact.confidence * 100)}%</p>
              </div>
              <div className="mt-1 h-2 w-full rounded-full bg-gray-100">
                <div className="h-2 rounded-full bg-indigo-500" style={{ width: `${Math.max(4, fact.confidence * 100)}%` }} />
              </div>
            </div>
          ))}
          {!data?.semantic_facts?.length ? <p className="text-sm text-gray-500">No semantic facts available yet.</p> : null}
        </div>
      </div>
    </div>
  );
}
