'use client';

import { ChangeEvent, FormEvent, useState } from 'react';
import { Trophy, Upload, Zap, Cloud, Droplets } from 'lucide-react';

const API_BASE_URL = process.env.NEXT_PUBLIC_API_URL || 'http://localhost:3001/api/v1';

interface Split {
  km: number;
  pace_s: number;
  pace_label: string;
  elevation_adj: string;
}

interface RaceStrategyResponse {
  weather?: {
    temp_c: number;
    condition: string;
    wind_kph: number;
    humidity: number;
  };
  fueling_timeline?: Array<{ km: number; action: string }>;
  splits?: Split[];
}

export default function RacePage() {
  const [raceName, setRaceName] = useState('');
  const [raceDate, setRaceDate] = useState('');
  const [goalTime, setGoalTime] = useState('03:45:00');
  const [gpxData, setGpxData] = useState('');
  const [isLoading, setIsLoading] = useState(false);
  const [error, setError] = useState<string | null>(null);
  const [result, setResult] = useState<RaceStrategyResponse | null>(null);

  async function handleUpload(event: ChangeEvent<HTMLInputElement>) {
    const file = event.target.files?.[0];
    if (!file) return;
    const text = await file.text();
    setGpxData(text);
  }

  function parseGoalTimeToSeconds(input: string) {
    const [h = '0', m = '0', s = '0'] = input.split(':');
    return Number(h) * 3600 + Number(m) * 60 + Number(s);
  }

  async function handleGenerate(event: FormEvent<HTMLFormElement>) {
    event.preventDefault();
    setError(null);
    setIsLoading(true);

    try {
      const token = localStorage.getItem('pacer_token');
      if (!token) throw new Error('Missing auth token. Please log in.');

      const response = await fetch(`${API_BASE_URL}/race/strategy`, {
        method: 'POST',
        headers: {
          'Content-Type': 'application/json',
          Authorization: `Bearer ${token}`,
        },
        body: JSON.stringify({
          race_name: raceName,
          race_date: raceDate,
          goal_time_s: parseGoalTimeToSeconds(goalTime),
          gpx_data: gpxData,
        }),
      });

      if (!response.ok) {
        throw new Error('Failed to generate race strategy');
      }

      const data = (await response.json()) as RaceStrategyResponse;
      setResult(data);
    } catch (requestError) {
      setError(requestError instanceof Error ? requestError.message : 'Unknown error');
    } finally {
      setIsLoading(false);
    }
  }

  return (
    <div className="space-y-6">
      {/* Gradient Hero Header */}
      <div className="relative overflow-hidden rounded-2xl bg-gradient-to-br from-amber-500 via-orange-500 to-red-500 p-8 shadow-lg">
        <div className="absolute inset-0 bg-black/5"></div>
        <div className="relative z-10">
          <div className="flex items-center gap-3">
            <div className="flex h-12 w-12 items-center justify-center rounded-xl bg-white/20 backdrop-blur-sm">
              <Trophy className="h-6 w-6 text-white" />
            </div>
            <div>
              <h1 className="text-3xl font-bold text-white">Race Strategy Planner</h1>
              <p className="text-sm text-white/80">Optimize your pacing for race day success</p>
            </div>
          </div>
        </div>
      </div>

      {/* Form */}
      <form onSubmit={handleGenerate} className="overflow-hidden rounded-2xl border border-gray-200 bg-white shadow-lg">
        <div className="bg-gradient-to-r from-orange-50 to-red-50 p-6">
          <h2 className="text-lg font-bold text-gray-900">Race Details</h2>
          <p className="text-sm text-gray-600">Enter your race information and upload the course GPX</p>
        </div>
        <div className="p-6">
          <div className="grid grid-cols-1 gap-4 md:grid-cols-3">
            <div>
              <label className="block text-sm font-medium text-gray-700">Race Name</label>
              <input
                type="text"
                value={raceName}
                onChange={(e) => setRaceName(e.target.value)}
                className="mt-1 w-full rounded-lg border border-gray-300 px-3 py-2 text-sm transition focus:border-orange-500 focus:ring-2 focus:ring-orange-200"
                placeholder="Berlin Marathon"
              />
            </div>
            <div>
              <label className="block text-sm font-medium text-gray-700">Race Date</label>
              <input
                type="date"
                value={raceDate}
                onChange={(e) => setRaceDate(e.target.value)}
                className="mt-1 w-full rounded-lg border border-gray-300 px-3 py-2 text-sm transition focus:border-orange-500 focus:ring-2 focus:ring-orange-200"
              />
            </div>
            <div>
              <label className="block text-sm font-medium text-gray-700">Goal Time (HH:MM:SS)</label>
              <input
                type="text"
                value={goalTime}
                onChange={(e) => setGoalTime(e.target.value)}
                className="mt-1 w-full rounded-lg border border-gray-300 px-3 py-2 text-sm transition focus:border-orange-500 focus:ring-2 focus:ring-orange-200"
              />
            </div>
          </div>

          <div className="mt-4">
            <label className="block text-sm font-medium text-gray-700">GPX Course File</label>
            <div className="mt-2 flex items-center gap-3">
              <label className="flex cursor-pointer items-center gap-2 rounded-lg border-2 border-dashed border-gray-300 bg-gray-50 px-4 py-2 text-sm text-gray-600 transition hover:border-orange-400 hover:bg-orange-50">
                <Upload className="h-4 w-4" />
                Choose File
                <input type="file" accept=".gpx,application/gpx+xml,text/xml" onChange={handleUpload} className="hidden" />
              </label>
              {gpxData ? (
                <span className="flex items-center gap-2 rounded-lg bg-emerald-50 px-3 py-1 text-xs font-medium text-emerald-700">
                  <span className="h-2 w-2 rounded-full bg-emerald-500"></span>
                  GPX loaded successfully
                </span>
              ) : null}
            </div>
          </div>

          <button
            type="submit"
            disabled={!gpxData || isLoading}
            className="mt-6 flex items-center gap-2 rounded-xl bg-gradient-to-r from-orange-600 to-red-600 px-6 py-3 text-sm font-bold text-white shadow-md transition hover:shadow-lg disabled:opacity-50"
          >
            <Zap className="h-4 w-4" />
            {isLoading ? 'Generating Strategy...' : 'Generate Race Strategy'}
          </button>

          {error ? (
            <div className="mt-4 rounded-lg bg-rose-50 p-4">
              <p className="text-sm text-rose-600">{error}</p>
            </div>
          ) : null}
        </div>
      </form>

      {/* Results */}
      {result ? (
        <div className="grid grid-cols-1 gap-6 lg:grid-cols-3">
          {/* Splits Table */}
          <div className="overflow-hidden rounded-2xl border border-gray-200 bg-white shadow-lg lg:col-span-2">
            <div className="bg-gradient-to-r from-indigo-50 to-purple-50 p-5">
              <h2 className="text-lg font-bold text-gray-900">Kilometer-by-Kilometer Splits</h2>
              <p className="text-sm text-gray-600">Your optimized pacing strategy</p>
            </div>
            <div className="p-5">
              <div className="overflow-x-auto">
                <table className="min-w-full text-sm">
                  <thead>
                    <tr className="border-b-2 border-gray-200 text-left text-xs font-bold uppercase tracking-wide text-gray-500">
                      <th className="px-4 py-3">KM</th>
                      <th className="px-4 py-3">Target Pace</th>
                      <th className="px-4 py-3">Elevation Adj</th>
                    </tr>
                  </thead>
                  <tbody className="divide-y divide-gray-100">
                    {(result.splits ?? []).map((split, idx) => (
                      <tr
                        key={split.km}
                        className="transition hover:bg-gray-50"
                        style={{ animationDelay: `${idx * 20}ms` }}
                      >
                        <td className="px-4 py-3 font-bold text-gray-800">{split.km}</td>
                        <td className="px-4 py-3 font-medium text-gray-700">{split.pace_label}</td>
                        <td className="px-4 py-3 text-gray-600">{split.elevation_adj}</td>
                      </tr>
                    ))}
                  </tbody>
                </table>
              </div>
            </div>
          </div>

          {/* Sidebar Cards */}
          <div className="space-y-4">
            {/* Weather Card */}
            <div className="overflow-hidden rounded-2xl border border-gray-200 bg-white shadow-lg">
              <div className="bg-gradient-to-br from-blue-500 to-cyan-500 p-5">
                <div className="flex items-center gap-2">
                  <Cloud className="h-5 w-5 text-white" />
                  <h2 className="text-sm font-bold text-white">Weather Forecast</h2>
                </div>
              </div>
              <div className="p-5">
                <p className="text-lg font-bold text-gray-900">{result.weather?.condition ?? 'Unknown'}</p>
                <div className="mt-4 space-y-2 text-sm">
                  <div className="flex items-center justify-between rounded-lg bg-gray-50 px-3 py-2">
                    <span className="text-gray-600">Temperature</span>
                    <span className="font-bold text-gray-900">{result.weather?.temp_c ?? 0}°C</span>
                  </div>
                  <div className="flex items-center justify-between rounded-lg bg-gray-50 px-3 py-2">
                    <span className="text-gray-600">Wind Speed</span>
                    <span className="font-bold text-gray-900">{result.weather?.wind_kph ?? 0} kph</span>
                  </div>
                  <div className="flex items-center justify-between rounded-lg bg-gray-50 px-3 py-2">
                    <span className="text-gray-600">Humidity</span>
                    <span className="font-bold text-gray-900">{result.weather?.humidity ?? 0}%</span>
                  </div>
                </div>
              </div>
            </div>

            {/* Fueling Card */}
            <div className="overflow-hidden rounded-2xl border border-gray-200 bg-white shadow-lg">
              <div className="bg-gradient-to-br from-emerald-500 to-green-500 p-5">
                <div className="flex items-center gap-2">
                  <Droplets className="h-5 w-5 text-white" />
                  <h2 className="text-sm font-bold text-white">Fueling Timeline</h2>
                </div>
              </div>
              <div className="p-5">
                <ul className="space-y-2 text-sm">
                  {(result.fueling_timeline ?? []).map((item, idx) => (
                    <li
                      key={`${item.km}-${item.action}`}
                      className="flex items-start gap-3 rounded-lg bg-gradient-to-r from-emerald-50 to-green-50 p-3"
                      style={{ animationDelay: `${idx * 50}ms` }}
                    >
                      <span className="flex h-6 w-6 flex-shrink-0 items-center justify-center rounded-full bg-emerald-500 text-xs font-bold text-white">
                        {item.km}
                      </span>
                      <span className="text-gray-700">{item.action}</span>
                    </li>
                  ))}
                </ul>
              </div>
            </div>
          </div>
        </div>
      ) : null}
    </div>
  );
}
