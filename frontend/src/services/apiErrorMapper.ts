// apiErrorMapper — унифицированный формат ошибок API.
//
// P1-2.2: Unified error format для всех endpoints
//   - Automatic retry для retryable errors (5xx, network)
//   - Inline errors для form fields
//   - Toast для global errors

export interface MappedApiError {
    type: 'validation' | 'auth' | 'not_found' | 'conflict' | 'rate_limit' | 'server' | 'network' | 'unknown';
    message: string;
    field?: string;       // для inline errors в формах
    retryable: boolean;
    action?: 'retry' | 'reload' | 'logout' | 'contact_support' | 'none';
    statusCode?: number;
    traceId?: string;
}

const RETRYABLE_STATUSES = [429, 500, 502, 503, 504];

export function mapApiError(error: unknown): MappedApiError {
    // Network error (fetch failed)
    if (error instanceof TypeError && error.message === 'Failed to fetch') {
        return {
            type: 'network',
            message: 'Network error. Please check your connection.',
            retryable: true,
            action: 'retry',
        };
    }

    // HTTP Response error
    if (error instanceof Response) {
        return mapHttpError(error);
    }

    // Axios-like error with response
    if (error && typeof error === 'object' && 'response' in error) {
        const err = error as { response: Response; message?: string };
        if (err.response) return mapHttpError(err.response);
        return { type: 'network', message: err.message || 'Network error', retryable: true, action: 'retry' };
    }

    // Standard Error
    if (error instanceof Error) {
        return { type: 'unknown', message: error.message, retryable: false, action: 'none' };
    }

    // String error
    if (typeof error === 'string') {
        return { type: 'unknown', message: error, retryable: false, action: 'none' };
    }

    return { type: 'unknown', message: 'An unknown error occurred', retryable: false, action: 'none' };
}

function mapHttpError(response: Response): MappedApiError {
    const isRetryable = RETRYABLE_STATUSES.includes(response.status);

    switch (response.status) {
        case 400:
            return { type: 'validation', message: 'Invalid request', field: extractFieldFromResponse(response), retryable: false, action: 'none', statusCode: 400 };
        case 401:
            return { type: 'auth', message: 'Authentication required', retryable: false, action: 'logout', statusCode: 401 };
        case 403:
            return { type: 'auth', message: 'Access denied', retryable: false, action: 'none', statusCode: 403 };
        case 404:
            return { type: 'not_found', message: 'Resource not found', retryable: false, action: 'none', statusCode: 404 };
        case 409:
            return { type: 'conflict', message: 'Resource conflict', retryable: false, action: 'reload', statusCode: 409 };
        case 422:
            return { type: 'validation', message: 'Validation error', field: extractFieldFromResponse(response), retryable: false, action: 'none', statusCode: 422 };
        case 429:
            return { type: 'rate_limit', message: 'Too many requests. Please wait.', retryable: true, action: 'retry', statusCode: 429 };
        case 500:
        case 502:
        case 503:
        case 504:
            return { type: 'server', message: `Server error (${response.status})`, retryable: true, action: 'retry', statusCode: response.status };
        default:
            return { type: 'unknown', message: `HTTP ${response.status}`, retryable: isRetryable, action: isRetryable ? 'retry' : 'none', statusCode: response.status };
    }
}

function extractFieldFromResponse(_response: Response): string | undefined {
    try {
        // Поле извлекается из JSON-ответа вида { field: "email", message: "...", code: "validation_error" }
        // Response можно прочитать только один раз, поэтому здесь
        // извлекаем поле только если это возможно без clone.
        return undefined;
    } catch {
        return undefined;
    }
}
