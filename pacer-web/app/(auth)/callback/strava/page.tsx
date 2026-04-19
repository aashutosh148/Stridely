'use client';

import { useEffect, Suspense } from 'react';
import { useRouter, useSearchParams } from 'next/navigation';

function CallbackLogic() {
  const router = useRouter();
  const searchParams = useSearchParams();

  useEffect(() => {
    console.log('🔐 Strava callback page loaded');
    console.log('📋 Search params:', searchParams.toString());
    
    const token = searchParams.get('token');
    console.log('🎫 Token from URL:', token ? `${token.substring(0, 20)}...` : 'NULL');
    
    if (token) {
      console.log('✅ Storing token in localStorage');
      localStorage.setItem('pacer_token', token);
      console.log('✅ Token stored, redirecting to dashboard');
      router.replace('/dashboard');
    } else {
      console.error('❌ No token found in URL, redirecting to login');
      router.replace('/login?error=auth_failed');
    }
  }, [router, searchParams]);

  return null;
}

export default function StravaCallbackPage() {
  console.log('🎨 Rendering Strava callback page');
  
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
