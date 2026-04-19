import { useState, useEffect } from 'react';
import { api, AuthMeResponse, User } from '../lib/api';

export function useUser() {
  const [user, setUser] = useState<User | null>(null);
  const [isStravaConnected, setIsStravaConnected] = useState(false);
  const [isGarminConnected, setIsGarminConnected] = useState(false);
  const [isLoading, setIsLoading] = useState(true);
  const [error, setError] = useState<Error | null>(null);

  useEffect(() => {
    async function fetchUser() {
      try {
        const token = localStorage.getItem('pacer_token');
        if (!token) {
          setIsLoading(false);
          return;
        }

        const data = await api.get<AuthMeResponse>('/auth/me');
        setUser(data.user);
        setIsStravaConnected(data.strava_connected);
        setIsGarminConnected(data.garmin_connected);
      } catch (err: unknown) {
        setError(err instanceof Error ? err : new Error('Unknown error'));
        localStorage.removeItem('pacer_token');
      } finally {
        setIsLoading(false);
      }
    }

    fetchUser();
  }, []);

  return {
    user,
    isLoading,
    error,
    isStravaConnected,
    isGarminConnected,
    isConnected: isStravaConnected || isGarminConnected,
  };
}
