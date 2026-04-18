import { useState, useEffect } from 'react';
import { api, AuthMeResponse, User } from '../lib/api';

export function useUser() {
  const [user, setUser] = useState<User | null>(null);
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
    isStravaConnected: user?.isStravaConnected || false,
    isGarminConnected: user?.isGarminConnected || false,
    isConnected: user?.isStravaConnected || user?.isGarminConnected || false,
  };
}
