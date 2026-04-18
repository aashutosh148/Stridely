// LLD Section 10.2: Base API Client

const API_BASE_URL = process.env.NEXT_PUBLIC_API_URL || 'http://localhost:3001/api/v1';

export interface User {
  id: string;
  email: string | null;
  name: string | null;
  tier: 'FREE' | 'PRO' | 'ELITE';
  isStravaConnected: boolean;
  isGarminConnected: boolean;
  targetRaceId: string | null;
  createdAt: string;
  updatedAt: string;
}

export interface AuthMeResponse {
  user: User;
}

export class ApiError extends Error {
  constructor(public status: number, public message: string, public data?: unknown) {
    super(message);
    this.name = 'ApiError';
  }
}

async function fetchWithAuth<T>(endpoint: string, options: RequestInit = {}): Promise<T> {
  const token = typeof window !== 'undefined' ? localStorage.getItem('pacer_token') : null;
  
  const headers: HeadersInit = {
    'Content-Type': 'application/json',
    ...(token ? { Authorization: `Bearer ${token}` } : {}),
    ...options.headers,
  };

  const url = `${API_BASE_URL}${endpoint}`;
  
  let response: Response;
  let lastError: unknown;

  for (let attempt = 0; attempt < 3; attempt++) {
    try {
      response = await fetch(url, {
        ...options,
        headers,
      });

      // Retry 5xx responses with exponential backoff
      if (response.status >= 500 && response.status < 600 && attempt < 2) {
        const delay = 250 * Math.pow(2, attempt);
        await new Promise((resolve) => setTimeout(resolve, delay));
        continue;
      }

      break;
    } catch (error) {
      lastError = error;
      if (attempt < 2) {
        const delay = 250 * Math.pow(2, attempt);
        await new Promise((resolve) => setTimeout(resolve, delay));
        continue;
      }
      throw error;
    }
  }

  if (!response!) {
    throw new ApiError(500, `Network request failed: ${String(lastError)}`);
  }

  if (!response.ok) {
    let errorMessage = 'An error occurred';
    let errorData = null;
    try {
      const data = await response.json();
      errorMessage = data.message || errorMessage;
      errorData = data;
    } catch {
      // Not JSON
    }
    throw new ApiError(response.status, errorMessage, errorData);
  }

  // Handle 204 No Content
  if (response.status === 204) {
    return {} as T;
  }

  return response.json();
}

export const api = {
  get: <T>(endpoint: string, options?: RequestInit) => fetchWithAuth<T>(endpoint, { ...options, method: 'GET' }),
  post: <T>(endpoint: string, body?: unknown, options?: RequestInit) => fetchWithAuth<T>(endpoint, { ...options, method: 'POST', body: JSON.stringify(body) }),
  put: <T>(endpoint: string, body?: unknown, options?: RequestInit) => fetchWithAuth<T>(endpoint, { ...options, method: 'PUT', body: JSON.stringify(body) }),
  patch: <T>(endpoint: string, body?: unknown, options?: RequestInit) => fetchWithAuth<T>(endpoint, { ...options, method: 'PATCH', body: JSON.stringify(body) }),
  delete: <T>(endpoint: string, options?: RequestInit) => fetchWithAuth<T>(endpoint, { ...options, method: 'DELETE' }),
};
