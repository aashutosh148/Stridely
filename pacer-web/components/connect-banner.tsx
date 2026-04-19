'use client';

import { useUser } from '../hooks/useUser';
import { AlertCircle } from 'lucide-react';

export function ConnectBanner() {
  const { isConnected, isStravaConnected, isGarminConnected, isLoading } = useUser();

  if (isLoading || isConnected) {
    return null;
  }

  return (
    <div className="bg-amber-500/10 border border-amber-500/30 rounded-lg p-4 mb-4">
      <div className="flex items-start gap-3">
        <AlertCircle className="w-5 h-5 text-amber-400 flex-shrink-0 mt-0.5" />
        <div className="flex-1">
          <p className="text-sm text-gray-300">
            You haven&apos;t connected a device yet. Please connect your Strava or Garmin account to get personalized training data.
          </p>
          <div className="mt-3 flex flex-wrap gap-2">
            {!isStravaConnected && (
              <button
                onClick={() => window.location.href = `${process.env.NEXT_PUBLIC_API_URL || 'http://localhost:3001/api/v1'}/auth/strava`}
                className="rounded bg-blue-600 px-3 py-1.5 text-sm font-medium text-white hover:bg-blue-700 transition-colors"
              >
                Connect Strava
              </button>
            )}
            {!isGarminConnected && (
              <button
                onClick={() => window.location.href = '/login'}
                className="rounded bg-gray-700 px-3 py-1.5 text-sm font-medium text-gray-200 hover:bg-gray-600 transition-colors"
              >
                Connect Garmin
              </button>
            )}
          </div>
        </div>
      </div>
    </div>
  );
}
