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
      <div className="h-72 animate-pulse rounded bg-gray-800/30" />
    );
  }

  const chartData = (data ?? []).slice(-90);
  const raceDay = raceDate ? new Date(raceDate).toISOString().slice(0, 10) : null;

  if (!chartData.length) {
    return (
      <div className="h-72 flex items-center justify-center">
        <p className="text-sm text-gray-500">No CTL/ATL/TSB data yet.</p>
      </div>
    );
  }

  return (
    <div className="h-72 w-full">
      <ResponsiveContainer width="100%" height="100%">
        <LineChart data={chartData} margin={{ top: 8, right: 12, left: -8, bottom: 0 }}>
          <CartesianGrid strokeDasharray="3 3" stroke="#1e2530" />
          <XAxis 
            dataKey="date" 
            tickFormatter={formatDate} 
            stroke="#6b7280" 
            tick={{ fontSize: 12, fill: '#9ca3af' }} 
            minTickGap={24} 
          />
          <YAxis 
            stroke="#6b7280" 
            tick={{ fontSize: 12, fill: '#9ca3af' }} 
          />
          <Tooltip
            labelFormatter={(value) => `Date: ${value}`}
            formatter={(value, name) => {
              const n = typeof value === 'number' ? value : Number(value ?? 0);
              return [n.toFixed(1), String(name).toUpperCase()];
            }}
            contentStyle={{ 
              backgroundColor: '#161b26', 
              border: '1px solid #374151',
              borderRadius: 8,
              color: '#e5e7eb'
            }}
            labelStyle={{ color: '#9ca3af' }}
          />
          {raceDay ? <ReferenceLine x={raceDay} stroke="#ef4444" strokeDasharray="4 4" label="Race Day" /> : null}
          <Line type="monotone" dataKey="ctl" stroke="#3b82f6" strokeWidth={2} dot={false} />
          <Line type="monotone" dataKey="atl" stroke="#ef4444" strokeWidth={2} dot={false} />
          <Line type="monotone" dataKey="tsb" stroke="#10b981" strokeWidth={2} dot={false} />
        </LineChart>
      </ResponsiveContainer>
    </div>
  );
}
