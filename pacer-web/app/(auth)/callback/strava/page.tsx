'use client';

import { useEffect, Suspense } from 'react';
import { useRouter, useSearchParams } from 'next/navigation';

function CallbackLogic() {
  const router = useRouter();
  const searchParams = useSearchParams();

  useEffect(() => {
    const token = searchParams.get('token');
    
    if (token) {
      localStorage.setItem('pacer_token', token);
      router.replace('/dashboard');
    } else {
      router.replace('/login?error=auth_failed');
    }
  }, [router, searchParams]);

  return null;
}

export default function StravaCallbackPage() {
  return (
    <div className="flex min-h-screen items-center justify-center bg-gray-50">
      <div className="text-center">
        <h2 className="text-xl font-semibold text-gray-900">Connecting to Strava...</h2>
        <p className="mt-2 text-sm text-gray-500">Please wait while we set up your account.</p>
        <Suspense fallback={<div className="mt-4">Loading...</div>}>
          <CallbackLogic />
        </Suspense>
      </div>
    </div>
  );
}
