'use client';

import {
  CartesianGrid,
  Line,
  LineChart,
  ReferenceLine,
  ResponsiveContainer,
  Tooltip,
  XAxis,
  YAxis,
} from 'recharts';
import type { FitnessPoint } from '@/hooks/useFitness';

function formatDate(dateStr: string) {
  const date = new Date(dateStr);
  return `${date.getMonth() + 1}/${date.getDate()}`;
}

export function FitnessChart({
  data,
  raceDate,
  isLoading,
}: {
  data?: FitnessPoint[];
  raceDate?: string | null;
  isLoading?: boolean;
}) {
  if (isLoading) {
    return (
      <div className="rounded-xl border border-gray-200 bg-white p-5 shadow-sm">
        <div className="h-5 w-40 animate-pulse rounded bg-gray-200" />
        <div className="mt-4 h-72 animate-pulse rounded bg-gray-100" />
      </div>
    );
  }

  const chartData = (data ?? []).slice(-90);
  const raceDay = raceDate ? new Date(raceDate).toISOString().slice(0, 10) : null;

  if (!chartData.length) {
    return (
      <div className="rounded-xl border border-gray-200 bg-white p-5 shadow-sm">
        <h2 className="text-sm font-semibold text-gray-900">Fitness Trends</h2>
        <p className="mt-3 text-sm text-gray-500">No CTL/ATL/TSB data yet.</p>
      </div>
    );
  }

  return (
    <div className="rounded-xl border border-gray-200 bg-white p-5 shadow-sm">
      <h2 className="text-sm font-semibold text-gray-900">Fitness Trends (90 Days)</h2>
      <div className="mt-4 h-72 w-full">
        <ResponsiveContainer width="100%" height="100%">
          <LineChart data={chartData} margin={{ top: 8, right: 12, left: -8, bottom: 0 }}>
            <CartesianGrid strokeDasharray="3 3" stroke="#E5E7EB" />
            <XAxis dataKey="date" tickFormatter={formatDate} stroke="#6B7280" tick={{ fontSize: 12 }} minTickGap={24} />
            <YAxis stroke="#6B7280" tick={{ fontSize: 12 }} />
            <Tooltip
              labelFormatter={(value) => `Date: ${value}`}
              formatter={(value, name) => {
                const n = typeof value === 'number' ? value : Number(value ?? 0);
                return [n.toFixed(1), String(name).toUpperCase()];
              }}
              contentStyle={{ borderRadius: 10, borderColor: '#E5E7EB' }}
            />
            {raceDay ? <ReferenceLine x={raceDay} stroke="#EF4444" strokeDasharray="4 4" label="Race Day" /> : null}
            <Line type="monotone" dataKey="ctl" stroke="#2563EB" strokeWidth={2} dot={false} />
            <Line type="monotone" dataKey="atl" stroke="#F97316" strokeWidth={2} dot={false} />
            <Line type="monotone" dataKey="tsb" stroke="#16A34A" strokeWidth={2} dot={false} />
          </LineChart>
        </ResponsiveContainer>
      </div>
    </div>
  );
}
