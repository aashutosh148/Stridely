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
    <div className="flex min-h-screen items-center justify-center bg-[#0b0f19] px-4">
      <div className="w-full max-w-md space-y-8 rounded-lg bg-[#161b26] border border-gray-800 p-8 shadow-2xl">
        <div className="text-center">
          <div className="mx-auto w-16 h-16 bg-blue-600 rounded-lg flex items-center justify-center mb-4">
            <span className="text-white font-bold text-2xl">P</span>
          </div>
          <h2 className="text-3xl font-bold tracking-tight text-gray-100">
            Welcome to Pacer
          </h2>
          <p className="mt-2 text-sm text-gray-400">
            Connect your devices to get started
          </p>
        </div>
        
        <div className="space-y-4">
          <button
            onClick={handleStravaConnect}
            className="flex w-full justify-center items-center gap-2 rounded-lg bg-[#fc4c02] px-4 py-3 text-sm font-semibold text-white shadow-sm hover:bg-[#e34402] transition-colors focus:outline-none focus:ring-2 focus:ring-[#fc4c02] focus:ring-offset-2 focus:ring-offset-[#161b26]"
          >
            <svg className="w-5 h-5" viewBox="0 0 24 24" fill="currentColor">
              <path d="M15.387 17.944l-2.089-4.116h-3.065L15.387 24l5.15-10.172h-3.066m-7.008-5.599l2.836 5.598h4.172L10.463 0l-7 13.828h4.169"/>
            </svg>
            Connect with Strava
          </button>

          <div className="relative">
            <div className="absolute inset-0 flex items-center">
              <div className="w-full border-t border-gray-800" />
            </div>
            <div className="relative flex justify-center text-sm">
              <span className="bg-[#161b26] px-2 text-gray-500">or</span>
            </div>
          </div>

          <form onSubmit={handleGarminConnect} className="space-y-4">
            <div>
              <label htmlFor="email" className="block text-sm font-medium text-gray-300 mb-1">
                Garmin Email
              </label>
              <input
                id="email"
                type="email"
                required
                className="block w-full rounded-lg border border-gray-700 bg-[#1e2530] px-3 py-2.5 text-gray-100 placeholder-gray-500 shadow-sm focus:border-blue-500 focus:outline-none focus:ring-1 focus:ring-blue-500 sm:text-sm"
                value={email}
                onChange={(e) => setEmail(e.target.value)}
                placeholder="you@example.com"
              />
            </div>
            <div>
              <label htmlFor="password" className="block text-sm font-medium text-gray-300 mb-1">
                Garmin Password
              </label>
              <input
                id="password"
                type="password"
                required
                className="block w-full rounded-lg border border-gray-700 bg-[#1e2530] px-3 py-2.5 text-gray-100 placeholder-gray-500 shadow-sm focus:border-blue-500 focus:outline-none focus:ring-1 focus:ring-blue-500 sm:text-sm"
                value={password}
                onChange={(e) => setPassword(e.target.value)}
                placeholder="••••••••"
              />
            </div>
            <button
              type="submit"
              disabled={isLoading}
              className="flex w-full justify-center rounded-lg bg-gray-700 px-4 py-3 text-sm font-semibold text-gray-100 shadow-sm hover:bg-gray-600 transition-colors focus:outline-none focus:ring-2 focus:ring-gray-500 focus:ring-offset-2 focus:ring-offset-[#161b26] disabled:opacity-50 disabled:cursor-not-allowed"
            >
              {isLoading ? 'Connecting...' : 'Connect with Garmin'}
            </button>
          </form>
        </div>
      </div>
    </div>
  );
}
