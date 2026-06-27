// ═══════════════════════════════════════════════════════════════════════
// API Client — base request function + auth helpers
// ARCH.2: Выделен из monolithic api.ts.
// ═══════════════════════════════════════════════════════════════════════

import { mapApiError, type MappedApiError } from '../apiErrorMapper';

export const API_BASE = '/api/v1';

// P1-SEC.1: JWT теперь в HttpOnly cookie, а не в localStorage.
// authToken используется только как fallback для API клиентов без cookies.
let authToken: string | null = null;

export function setAuthToken(token: string | null) {
  authToken = token;
}

export function getAuthToken(): string | null {
  return authToken;
}

// Получаем CSRF токен из cookie для state-changing запросов
function getCSRFToken(): string | null {
  if (typeof document === 'undefined') return null;
  const match = document.cookie.match(/(?:^|;\s*)csrf_token=([^;]*)/);
  return match ? match[1] : null;
}

export async function request<T>(
  path: string,
  options: RequestInit = {},
): Promise<T> {
  const headers: Record<string, string> = {
    'Content-Type': 'application/json',
  };

  if (options.headers) {
    Object.assign(headers, options.headers as Record<string, string>);
  }

  // P1-SEC.1: Добавляем CSRF токен для state-changing методов
  if (options.method && options.method !== 'GET' && options.method !== 'HEAD') {
    const csrfToken = getCSRFToken();
    if (csrfToken) {
      headers['X-CSRF-Token'] = csrfToken;
    }
  }

  // P1-SEC.1: JWT теперь в HttpOnly cookie, Authorization header — fallback
  if (authToken) {
    headers['Authorization'] = `Bearer ${authToken}`;
  }

  const response = await fetch(`${API_BASE}${path}`, {
    ...options,
    headers,
    credentials: 'include',
  });

  if (!response.ok) {
    if (response.status === 401) {
      setAuthToken(null);
      if (typeof window !== 'undefined' && !window.location.pathname.includes('/login')) {
        window.location.href = '/login';
        throw new Error('Session expired. Please log in again.');
      }
    }
    if (response.status === 403) {
      throw new Error('Access denied. Insufficient permissions.');
    }

    const body = await response.text();
    try {
      const parsed = JSON.parse(body);
      const msg = parsed?.error?.message;
      if (msg && typeof msg === 'string') {
        throw new Error(msg);
      }
    } catch (e) {
      if (e instanceof Error && e.message !== 'Failed to parse error response') {
        throw e;
      }
    }
    throw new Error(body || `Request failed with status ${response.status}`);
  }

  if (response.status === 204) {
    return null as T;
  }

  const contentType = response.headers.get('content-type');
  if (contentType && contentType.includes('application/json')) {
    return response.json();
  }

  return null as T;
}

export async function requestBlob(path: string): Promise<Blob> {
  const headers: Record<string, string> = {};
  if (authToken) {
    headers['Authorization'] = `Bearer ${authToken}`;
  }
  const response = await fetch(`${API_BASE}${path}`, { headers });
  if (!response.ok) throw new Error('Failed to download file');
  return response.blob();
}

export function handleApiError(error: unknown): MappedApiError {
  const mapped = mapApiError(error);
  if (mapped.retryable) {
    console.warn('[API] Retryable error:', mapped);
  }
  return mapped;
}
