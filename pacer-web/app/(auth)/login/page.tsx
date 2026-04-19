'use client';

import { useRouter } from 'next/navigation';
import { useState } from 'react';

export default function LoginPage() {
  const router = useRouter();
  const [email, setEmail] = useState('');
  const [password, setPassword] = useState('');
  const [isLoading, setIsLoading] = useState(false);

  const handleStravaConnect = () => {
    window.location.href = `${process.env.NEXT_PUBLIC_API_URL || 'http://localhost:3001/api/v1'}/auth/strava`;
  };

  const handleGarminConnect = async (e: React.FormEvent) => {
    e.preventDefault();
    setIsLoading(true);
    // Interim Garmin form handler
    try {
      const res = await fetch(`${process.env.NEXT_PUBLIC_API_URL || 'http://localhost:3001/api/v1'}/auth/garmin`, {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify({ email, password }),
      });
      if (!res.ok) throw new Error('Garmin connect failed');
      const data = await res.json();
      if (data.token) {
        localStorage.setItem('pacer_token', data.token);
        router.push('/dashboard');
      }
    } catch (err) {
      console.error(err);
      alert('Failed to connect Garmin');
    } finally {
      setIsLoading(false);
    }
  };

  return (
    <div className="flex min-h-screen items-center justify-center bg-gray-50 px-4">
      <div className="w-full max-w-md space-y-8 rounded-lg bg-white p-6 shadow-md">
        <div>
          <h2 className="text-center text-3xl font-bold tracking-tight text-gray-900">
            Connect your devices
          </h2>
        </div>
        <div className="space-y-4">
          <button
            onClick={handleStravaConnect}
            className="flex w-full justify-center rounded-md bg-[#fc4c02] px-3 py-2 text-sm font-semibold text-white shadow-sm hover:bg-[#e34402] focus-visible:outline focus-visible:outline-2 focus-visible:outline-offset-2 focus-visible:outline-[#fc4c02]"
          >
            Connect Strava
          </button>

          <div className="relative">
            <div className="absolute inset-0 flex items-center">
              <div className="w-full border-t border-gray-300" />
            </div>
            <div className="relative flex justify-center text-sm">
              <span className="bg-white px-2 text-gray-500">or</span>
            </div>
          </div>

          <form onSubmit={handleGarminConnect} className="space-y-4">
            <div>
              <label htmlFor="email" className="block text-sm font-medium text-gray-700">
                Garmin Email
              </label>
              <input
                id="email"
                type="email"
                required
                className="mt-1 block w-full rounded-md border border-gray-300 px-3 py-2 shadow-sm focus:border-indigo-500 focus:outline-none focus:ring-indigo-500 sm:text-sm"
                value={email}
                onChange={(e) => setEmail(e.target.value)}
              />
            </div>
            <div>
              <label htmlFor="password" className="block text-sm font-medium text-gray-700">
                Garmin Password
              </label>
              <input
                id="password"
                type="password"
                required
                className="mt-1 block w-full rounded-md border border-gray-300 px-3 py-2 shadow-sm focus:border-indigo-500 focus:outline-none focus:ring-indigo-500 sm:text-sm"
                value={password}
                onChange={(e) => setPassword(e.target.value)}
              />
            </div>
            <button
              type="submit"
              disabled={isLoading}
              className="flex w-full justify-center rounded-md bg-black px-3 py-2 text-sm font-semibold text-white shadow-sm hover:bg-gray-800 focus-visible:outline focus-visible:outline-2 focus-visible:outline-offset-2 focus-visible:outline-black disabled:opacity-50"
            >
              {isLoading ? 'Connecting...' : 'Connect Garmin'}
            </button>
          </form>
        </div>
      </div>
    </div>
  );
}
