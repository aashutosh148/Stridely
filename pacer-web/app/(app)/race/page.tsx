'use client';

import { ChangeEvent, FormEvent, useState } from 'react';

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
      <h1 className="text-2xl font-semibold text-gray-900">Race Strategy Planner</h1>

      <form onSubmit={handleGenerate} className="rounded-xl border border-gray-200 bg-white p-5 shadow-sm">
        <div className="grid grid-cols-1 gap-4 md:grid-cols-3">
          <div>
            <label className="block text-sm font-medium text-gray-700">Race Name</label>
            <input
              type="text"
              value={raceName}
              onChange={(e) => setRaceName(e.target.value)}
              className="mt-1 w-full rounded-lg border border-gray-300 px-3 py-2 text-sm"
              placeholder="Berlin Marathon"
            />
          </div>
          <div>
            <label className="block text-sm font-medium text-gray-700">Race Date</label>
            <input
              type="date"
              value={raceDate}
              onChange={(e) => setRaceDate(e.target.value)}
              className="mt-1 w-full rounded-lg border border-gray-300 px-3 py-2 text-sm"
            />
          </div>
          <div>
            <label className="block text-sm font-medium text-gray-700">Goal Time (HH:MM:SS)</label>
            <input
              type="text"
              value={goalTime}
              onChange={(e) => setGoalTime(e.target.value)}
              className="mt-1 w-full rounded-lg border border-gray-300 px-3 py-2 text-sm"
            />
          </div>
        </div>

        <div className="mt-4">
          <label className="block text-sm font-medium text-gray-700">GPX Upload</label>
          <input type="file" accept=".gpx,application/gpx+xml,text/xml" onChange={handleUpload} className="mt-2 text-sm" />
          {gpxData ? <p className="mt-2 text-xs text-emerald-700">GPX loaded successfully.</p> : null}
        </div>

        <button
          type="submit"
          disabled={!gpxData || isLoading}
          className="mt-5 rounded-lg bg-indigo-600 px-4 py-2 text-sm font-semibold text-white hover:bg-indigo-700 disabled:opacity-50"
        >
          {isLoading ? 'Generating...' : 'Generate Strategy'}
        </button>

        {error ? <p className="mt-3 text-sm text-rose-600">{error}</p> : null}
      </form>

      {result ? (
        <div className="grid grid-cols-1 gap-6 lg:grid-cols-3">
          <div className="rounded-xl border border-gray-200 bg-white p-5 shadow-sm lg:col-span-2">
            <h2 className="text-lg font-semibold text-gray-900">Km-by-km Splits</h2>
            <div className="mt-3 overflow-x-auto">
              <table className="min-w-full divide-y divide-gray-200 text-sm">
                <thead>
                  <tr className="text-left text-xs uppercase tracking-wide text-gray-500">
                    <th className="px-2 py-2">KM</th>
                    <th className="px-2 py-2">Target Pace</th>
                    <th className="px-2 py-2">Elevation Adj</th>
                  </tr>
                </thead>
                <tbody className="divide-y divide-gray-100">
                  {(result.splits ?? []).map((split) => (
                    <tr key={split.km}>
                      <td className="px-2 py-2 font-medium text-gray-800">{split.km}</td>
                      <td className="px-2 py-2 text-gray-700">{split.pace_label}</td>
                      <td className="px-2 py-2 text-gray-700">{split.elevation_adj}</td>
                    </tr>
                  ))}
                </tbody>
              </table>
            </div>
          </div>

          <div className="space-y-4">
            <div className="rounded-xl border border-gray-200 bg-white p-5 shadow-sm">
              <h2 className="text-sm font-semibold text-gray-900">Weather Conditions</h2>
              <p className="mt-2 text-sm text-gray-700">{result.weather?.condition ?? 'Unknown'}</p>
              <p className="text-sm text-gray-600">Temp: {result.weather?.temp_c ?? 0} C</p>
              <p className="text-sm text-gray-600">Wind: {result.weather?.wind_kph ?? 0} kph</p>
              <p className="text-sm text-gray-600">Humidity: {result.weather?.humidity ?? 0}%</p>
            </div>

            <div className="rounded-xl border border-gray-200 bg-white p-5 shadow-sm">
              <h2 className="text-sm font-semibold text-gray-900">Fueling Timeline</h2>
              <ul className="mt-2 space-y-1 text-sm text-gray-700">
                {(result.fueling_timeline ?? []).map((item) => (
                  <li key={`${item.km}-${item.action}`}>
                    KM {item.km}: {item.action}
                  </li>
                ))}
              </ul>
            </div>
          </div>
        </div>
      ) : null}
    </div>
  );
}
