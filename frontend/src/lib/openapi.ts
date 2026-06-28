// ═══════════════════════════════════════════════════════════════════════
// OpenAPI Type-Safe Client
// P1-ARCH.3: Type-safe API клиент на основе сгенерированных openapi-typescript типов.
//
// Использование:
//   import { apiClient } from '../lib/openapi';
//
//   // GET с типизированным ответом
//   const devices = await apiClient.GET('/devices', { params: { query: { status: 'online' } } });
//
//   // POST с типизированным телом и ответом
//   const created = await apiClient.POST('/users', { body: { username, password, email, role } });
//
//   // PUT с path параметрами
//   await apiClient.PUT('/devices/{id}', { params: { path: { id } }, body: { ... } });
//
//   // DELETE
//   await apiClient.DELETE('/devices/{id}', { params: { path: { id } } });
// ═══════════════════════════════════════════════════════════════════════

import type {
  paths,
  operations,
  components,
} from '../types/api';

// ─── Type Utilities ───────────────────────────────────────────────────

/**
 * Извлекает тип ответа 2xx из operation.
 * Пробует 200 → 201 → fallback void.
 */
type ResponseData<Op extends keyof operations> =
  operations[Op]['responses'] extends {
    200: { content: { "application/json": infer T } };
  } ? T
  : operations[Op]['responses'] extends {
    201: { content: { "application/json": infer T } };
  } ? T
  : void;

/**
 * Извлекает тип тела запроса из operation (если есть).
 */
type RequestBodyType<Op extends keyof operations> =
  operations[Op] extends { requestBody: { content: { "application/json": infer T } } }
    ? T
    : undefined;

/**
 * Извлекает query-параметры из operation (если есть).
 */
type QueryParams<Op extends keyof operations> =
  operations[Op] extends { parameters: { query: infer Q } } ? Q
  : undefined;

/**
 * Извлекает path-параметры из operation (если есть).
 */
type PathParams<Op extends keyof operations> =
  operations[Op] extends { parameters: { path: infer P } } ? P
  : undefined;

// ─── Route Configuration ─────────────────────────────────────────────

/**
 * Маппинг URL paths → methods → operationId.
 * Используется для type-safe роутинга.
 */
type RouteMap = {
  [Path in keyof paths]: {
    [Method in keyof paths[Path]]: paths[Path][Method] extends {
      [K in keyof operations]: operations[K];
    } ? paths[Path][Method]
    : never;
  };
};

/**
 * Извлекает operationId по path и method.
 */
type OperationFor<
  Path extends keyof paths,
  Method extends keyof paths[Path]
> = paths[Path][Method] extends keyof operations
  ? paths[Path][Method]
  : paths[Path][Method] extends { [K in keyof operations]: operations[K] }
    ? paths[Path][Method] extends keyof operations ? paths[Path][Method] : keyof paths[Path][Method] extends keyof operations ? keyof paths[Path][Method] : never
    : never;

/**
 * Тип тела запроса для конкретного path + method.
 */
type RequestBodyFor<
  Path extends keyof paths,
  Method extends keyof paths[Path]
> =
  paths[Path][Method] extends keyof operations
    ? RequestBodyType<paths[Path][Method]>
    : undefined;

/**
 * Тип ответа для конкретного path + method.
 */
type ResponseFor<
  Path extends keyof paths,
  Method extends keyof paths[Path]
> =
  paths[Path][Method] extends keyof operations
    ? ResponseData<paths[Path][Method]>
    : never;

/**
 * Тип query-параметров для конкретного path + method.
 */
type QueryFor<
  Path extends keyof paths,
  Method extends keyof paths[Path]
> =
  paths[Path][Method] extends keyof operations
    ? QueryParams<paths[Path][Method]>
    : undefined;

/**
 * Тип path-параметров для конкретного path + method.
 */
type PathFor<
  Path extends keyof paths,
  Method extends keyof paths[Path]
> =
  paths[Path][Method] extends keyof operations
    ? PathParams<paths[Path][Method]>
    : undefined;

// ─── Request Options ──────────────────────────────────────────────────

interface RequestOptionsBase {
  /** Additional headers */
  headers?: Record<string, string>;
  /** Signal for abort */
  signal?: AbortSignal;
}

interface RequestOptionsWithBody<B> extends RequestOptionsBase {
  body: B;
}

interface RequestOptionsWithParams<P, Q> extends RequestOptionsBase {
  params?: {
    path?: P;
    query?: Q;
  };
}

// ─── URL Builder ──────────────────────────────────────────────────────

/**
 * Заменяет path-параметры {id} на фактические значения.
 */
function buildPath(path: string, pathParams?: Record<string, string>): string {
  if (!pathParams) return path;
  let result = path;
  for (const [key, value] of Object.entries(pathParams)) {
    result = result.replace(`{${key}}`, encodeURIComponent(String(value)));
  }
  return result;
}

/**
 * Строит query-строку.
 */
function buildQuery(queryParams?: Record<string, unknown>): string {
  if (!queryParams) return '';
  const searchParams = new URLSearchParams();
  for (const [key, value] of Object.entries(queryParams)) {
    if (value !== undefined && value !== null) {
      searchParams.append(key, String(value));
    }
  }
  const qs = searchParams.toString();
  return qs ? `?${qs}` : '';
}

// ─── API Client ───────────────────────────────────────────────────────

const API_BASE = '/api/v1';

async function apiRequest<T>(
  method: string,
  path: string,
  options?: {
    body?: unknown;
    headers?: Record<string, string>;
    signal?: AbortSignal;
  },
): Promise<T> {
  const headers: Record<string, string> = {
    'Content-Type': 'application/json',
    ...options?.headers,
  };

  // CSRF для state-changing методов
  if (method !== 'GET' && method !== 'HEAD') {
    if (typeof document !== 'undefined') {
      const match = document.cookie.match(/(?:^|;\s*)csrf_token=([^;]*)/);
      if (match) {
        headers['X-CSRF-Token'] = match[1];
      }
    }
  }

  const response = await fetch(`${API_BASE}${path}`, {
    method,
    headers,
    credentials: 'include',
    body: options?.body ? JSON.stringify(options.body) : undefined,
    signal: options?.signal,
  });

  if (!response.ok) {
    if (response.status === 401) {
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

// ─── Public API ───────────────────────────────────────────────────────

export const apiClient = {
  /**
   * GET запрос с type-safe ответом.
   *
   * @example
   *   const devices = await apiClient.GET('/devices');
   *   const filtered = await apiClient.GET('/devices', {
   *     params: { query: { status: 'online', site_id: '...' } }
   *   });
   */
  GET<Path extends keyof paths, Method extends 'get'>(
    path: Path,
    options?: RequestOptionsWithParams<
      PathFor<Path, Method>,
      QueryFor<Path, Method>
    >,
  ): Promise<ResponseFor<Path, Method>> {
    const fullPath = buildPath(
      path as string,
      options?.params?.path as Record<string, string> | undefined,
    ) + buildQuery(options?.params?.query as Record<string, unknown> | undefined);

    return apiRequest<ResponseFor<Path, Method>>('GET', fullPath, {
      headers: options?.headers,
      signal: options?.signal,
    });
  },

  /**
   * POST запрос с type-safe телом и ответом.
   *
   * @example
   *   const user = await apiClient.POST('/users', {
   *     body: { username: 'john', password: '...', email: '...', role: 'technician' }
   *   });
   */
  POST<Path extends keyof paths, Method extends 'post'>(
    path: Path,
    options: RequestOptionsWithBody<RequestBodyFor<Path, Method>> &
      RequestOptionsWithParams<
        PathFor<Path, Method>,
        QueryFor<Path, Method>
      >,
  ): Promise<ResponseFor<Path, Method>> {
    const fullPath = buildPath(
      path as string,
      options?.params?.path as Record<string, string> | undefined,
    ) + buildQuery(options?.params?.query as Record<string, unknown> | undefined);

    return apiRequest<ResponseFor<Path, Method>>('POST', fullPath, {
      body: options.body,
      headers: options.headers,
      signal: options.signal,
    });
  },

  /**
   * PUT запрос с type-safe телом и ответом.
   */
  PUT<Path extends keyof paths, Method extends 'put'>(
    path: Path,
    options: RequestOptionsWithBody<RequestBodyFor<Path, Method>> &
      RequestOptionsWithParams<
        PathFor<Path, Method>,
        QueryFor<Path, Method>
      >,
  ): Promise<ResponseFor<Path, Method>> {
    const fullPath = buildPath(
      path as string,
      options?.params?.path as Record<string, string> | undefined,
    ) + buildQuery(options?.params?.query as Record<string, unknown> | undefined);

    return apiRequest<ResponseFor<Path, Method>>('PUT', fullPath, {
      body: options.body,
      headers: options.headers,
      signal: options.signal,
    });
  },

  /**
   * DELETE запрос.
   */
  DELETE<Path extends keyof paths, Method extends 'delete'>(
    path: Path,
    options?: RequestOptionsWithParams<
      PathFor<Path, Method>,
      QueryFor<Path, Method>
    >,
  ): Promise<ResponseFor<Path, Method>> {
    const fullPath = buildPath(
      path as string,
      options?.params?.path as Record<string, string> | undefined,
    ) + buildQuery(options?.params?.query as Record<string, unknown> | undefined);

    return apiRequest<ResponseFor<Path, Method>>('DELETE', fullPath, {
      headers: options?.headers,
      signal: options?.signal,
    });
  },
};

// ─── Typed Schema Helpers ────────────────────────────────────────────

/**
 * Type-safe доступ к схемам из OpenAPI спецификации.
 * Позволяет использовать сгенерированные типы в runtime-валидации.
 *
 * @example
 *   import type { schemas } from '../lib/openapi';
 *   const user: schemas['User'] = { id: '...', username: 'john', ... };
 */
export type schemas = components['schemas'];

// ─── Re-export generated types for convenience ───────────────────────

export type {
  components,
  operations,
  paths,
} from '../types/api';
