'use client';

import { Pie, PieChart, ResponsiveContainer, Tooltip, Cell } from 'recharts';

interface ZoneShape {
  z1_pct?: number;
  z2_pct?: number;
  z3_pct?: number;
  z4_pct?: number;
  z5_pct?: number;
}

const COLORS = ['#38BDF8', '#60A5FA', '#FACC15', '#FB923C', '#F43F5E'];

function toData(source?: ZoneShape) {
  return [
    { name: 'Z1', value: source?.z1_pct ?? 0 },
    { name: 'Z2', value: source?.z2_pct ?? 0 },
    { name: 'Z3', value: source?.z3_pct ?? 0 },
    { name: 'Z4', value: source?.z4_pct ?? 0 },
    { name: 'Z5', value: source?.z5_pct ?? 0 },
  ];
}

export function ZoneDonut({ actual, prescribed, isLoading }: { actual?: ZoneShape; prescribed?: ZoneShape; isLoading?: boolean }) {
  if (isLoading) {
    return (
      <div className="rounded-xl border border-gray-200 bg-white p-5 shadow-sm">
        <div className="h-5 w-32 animate-pulse rounded bg-gray-200" />
        <div className="mt-4 h-56 animate-pulse rounded bg-gray-100" />
      </div>
    );
  }

  const actualData = toData(actual);
  const prescribedData = toData(prescribed);

  return (
    <div className="rounded-xl border border-gray-200 bg-white p-5 shadow-sm">
      <h2 className="text-sm font-semibold text-gray-900">Zone Distribution</h2>
      <div className="mt-4 h-56 w-full">
        <ResponsiveContainer width="100%" height="100%">
          <PieChart>
            <Pie data={actualData} dataKey="value" nameKey="name" innerRadius={46} outerRadius={72} paddingAngle={2}>
              {actualData.map((entry, idx) => (
                <Cell key={entry.name} fill={COLORS[idx]} />
              ))}
            </Pie>
            <Tooltip
              formatter={(value, name) => {
                const n = typeof value === 'number' ? value : Number(value ?? 0);
                return [`${n.toFixed(1)}%`, String(name)];
              }}
            />
          </PieChart>
        </ResponsiveContainer>
      </div>

      <div className="mt-3 grid grid-cols-5 gap-2 text-center text-xs">
        {actualData.map((z, idx) => (
          <div key={z.name} className="rounded-md bg-gray-50 p-2">
            <p className="font-semibold" style={{ color: COLORS[idx] }}>
              {z.name}
            </p>
            <p className="text-gray-700">{z.value.toFixed(0)}%</p>
            <p className="text-gray-400">/{prescribedData[idx].value.toFixed(0)}%</p>
          </div>
        ))}
      </div>
    </div>
  );
}
