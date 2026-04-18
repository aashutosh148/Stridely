'use client';

import { useEffect } from 'react';

export default function GlobalError({
  error,
  unstable_retry,
}: {
  error: Error & { digest?: string };
  unstable_retry: () => void;
}) {
  useEffect(() => {
    console.error(error);
  }, [error]);

  return (
    <html lang="en">
      <body className="min-h-screen bg-gray-100 p-6">
        <main className="mx-auto mt-20 w-full max-w-lg rounded-xl border border-gray-200 bg-white p-6 text-center shadow-sm">
          <h1 className="text-xl font-semibold text-gray-900">Something went wrong</h1>
          <p className="mt-2 text-sm text-gray-600">An unexpected error occurred while loading the app.</p>
          <button
            type="button"
            onClick={() => unstable_retry()}
            className="mt-5 rounded-lg bg-indigo-600 px-4 py-2 text-sm font-semibold text-white hover:bg-indigo-700"
          >
            Try again
          </button>
        </main>
      </body>
    </html>
  );
}
