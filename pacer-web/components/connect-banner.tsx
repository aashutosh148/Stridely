'use client';

import { useUser } from '../hooks/useUser';

export function ConnectBanner() {
  const { isConnected, isStravaConnected, isGarminConnected, isLoading } = useUser();

  if (isLoading || isConnected) {
    return null;
  }

  return (
    <div className="bg-yellow-50 border-l-4 border-yellow-400 p-4 mb-4">
      <div className="flex">
        <div className="ml-3">
          <p className="text-sm text-yellow-700">
            You haven&apos;t connected a device yet. Please connect your Strava or Garmin account to get personalized training data.
          </p>
          <div className="mt-4">
            <div className="-mx-2 -my-1.5 flex flex-wrap">
              {!isStravaConnected && (
                <button
                  onClick={() => window.location.href = `${process.env.NEXT_PUBLIC_API_URL || 'http://localhost:3001'}/api/v1/auth/strava`}
                  className="rounded-md bg-yellow-100 px-2 py-1.5 text-sm font-medium text-yellow-800 hover:bg-yellow-200 focus:outline-none focus:ring-2 focus:ring-yellow-600 focus:ring-offset-2 focus:ring-offset-yellow-50 mr-3"
                >
                  Connect Strava
                </button>
              )}
              {!isGarminConnected && (
                <button
                  onClick={() => window.location.href = '/login'}
                  className="rounded-md bg-yellow-100 px-2 py-1.5 text-sm font-medium text-yellow-800 hover:bg-yellow-200 focus:outline-none focus:ring-2 focus:ring-yellow-600 focus:ring-offset-2 focus:ring-offset-yellow-50"
                >
                  Connect Garmin
                </button>
              )}
            </div>
          </div>
        </div>
      </div>
    </div>
  );
}
