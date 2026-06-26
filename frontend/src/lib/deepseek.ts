// ═══════════════════════════════════════════════════════════════════════
// DeepSeek AI Assistant API Client (P2-1.2)
//
// Клиент для общения с AI Assistant через backend proxy.
// Все запросы идут через /api/v1/ai/*, API key хранится только на сервере.
//
// Compliance:
//   - OWASP ASVS V3 (Session Management — JWT токен из заголовка)
//   - OWASP ASVS V5.1 (Input validation)
//   - IEC 62443 SR 7.1 (Timeout — 60s)
// ═══════════════════════════════════════════════════════════════════════

export interface AIChatMessage {
  role: 'user' | 'assistant';
  content: string;
}

export interface AIChatContext {
  current_page?: string;
  device_id?: string;
  wo_id?: string;
  site_id?: string;
}

export interface AIChatHistoryRequest {
  message: string;
  context: AIChatContext;
  history: AIChatMessage[];
}

export interface AIFeedbackRequest {
  message_id: string;
  score: 'like' | 'dislike';
}

export type AIStreamEvent =
  | { type: 'chunk'; content: string }
  | { type: 'done'; content: string }
  | { type: 'error'; content: string };

const API_BASE = '/api/v1/ai';

/**
 * Получает JWT токен из localStorage для авторизации.
 */
function getAuthToken(): string | null {
  return localStorage.getItem('token');
}

/**
 * Отправляет сообщение AI Assistant и получает ответ через SSE стриминг.
 *
 * @param message - Текст сообщения пользователя
 * @param context - Контекст текущей страницы (page, device_id, wo_id)
 * @param history - История предыдущих сообщений
 * @param onEvent - Колбэк для каждого SSE события (chunk/done/error)
 * @param signal - AbortSignal для отмены запроса
 * @returns Promise с полным ответом по завершении
 */
export async function sendChatMessage(
  message: string,
  context: AIChatContext,
  history: AIChatMessage[],
  onEvent: (event: AIStreamEvent) => void,
  signal?: AbortSignal
): Promise<string> {
  const token = getAuthToken();

  const response = await fetch(`${API_BASE}/chat`, {
    method: 'POST',
    headers: {
      'Content-Type': 'application/json',
      ...(token ? { Authorization: `Bearer ${token}` } : {}),
    },
    body: JSON.stringify({
      message,
      context,
      history,
    } satisfies AIChatHistoryRequest),
    signal,
  });

  if (!response.ok) {
    const body = await response.text().catch(() => '');
    let errorMessage = `AI request failed (${response.status})`;
    try {
      const parsed = JSON.parse(body);
      errorMessage = parsed?.error?.message || errorMessage;
    } catch {
      // ignore parse error
    }
    throw new Error(errorMessage);
  }

  const contentType = response.headers.get('content-type') || '';

  // SSE streaming
  if (contentType.includes('text/event-stream')) {
    return readSSEStream(response, onEvent, signal);
  }

  // Fallback для не-streaming ответа
  const data = await response.json();
  const content = data?.content || '';
  onEvent({ type: 'done', content });
  return content;
}

/**
 * Читает SSE поток из ответа.
 */
async function readSSEStream(
  response: Response,
  onEvent: (event: AIStreamEvent) => void,
  signal?: AbortSignal
): Promise<string> {
  const reader = response.body?.getReader();
  if (!reader) {
    throw new Error('Response body is not readable');
  }

  const decoder = new TextDecoder();
  let buffer = '';
  let fullResponse = '';

  try {
    while (true) {
      if (signal?.aborted) {
        reader.cancel();
        throw new DOMException('Aborted', 'AbortError');
      }

      const { done, value } = await reader.read();
      if (done) break;

      buffer += decoder.decode(value, { stream: true });

      // Разбиваем буфер на SSE сообщения (разделяются \n\n)
      const parts = buffer.split('\n\n');
      buffer = parts.pop() || ''; // последний может быть неполным

      for (const part of parts) {
        const event = parseSSEEvent(part);
        if (event) {
          onEvent(event);
          if (event.type === 'done') {
            fullResponse = event.content;
          }
          if (event.type === 'error') {
            throw new Error(event.content);
          }
        }
      }
    }

    // Обрабатываем остаток буфера
    if (buffer.trim()) {
      const event = parseSSEEvent(buffer);
      if (event) {
        onEvent(event);
        if (event.type === 'done') {
          fullResponse = event.content;
        }
      }
    }
  } finally {
    reader.releaseLock();
  }

  return fullResponse;
}

/**
 * Парсит одно SSE event сообщение.
 * Формат: "data: {...}"
 */
function parseSSEEvent(data: string): AIStreamEvent | null {
  const lines = data.trim().split('\n');
  for (const line of lines) {
    if (line.startsWith('data: ')) {
      const jsonStr = line.slice(6).trim();
      try {
        return JSON.parse(jsonStr) as AIStreamEvent;
      } catch {
        // skip malformed JSON
      }
    }
  }
  return null;
}

/**
 * Отправляет обратную связь (like/dislike) для ответа AI.
 */
export async function sendFeedback(
  messageId: string,
  score: 'like' | 'dislike'
): Promise<void> {
  const token = getAuthToken();

  const response = await fetch(`${API_BASE}/feedback`, {
    method: 'POST',
    headers: {
      'Content-Type': 'application/json',
      ...(token ? { Authorization: `Bearer ${token}` } : {}),
    },
    body: JSON.stringify({
      message_id: messageId,
      score,
    } satisfies AIFeedbackRequest),
  });

  if (!response.ok) {
    throw new Error(`Failed to send feedback (${response.status})`);
  }
}

/**
 * Извлекает контекст из текущего URL.
 */
export function extractContextFromURL(): AIChatContext {
  const path = window.location.pathname;
  const search = window.location.search;

  const context: AIChatContext = {
    current_page: path,
  };

  // Извлекаем device_id из пути: /devices/{id} или /devices/{id}/edit
  const deviceMatch = path.match(/\/devices\/([a-zA-Z0-9_-]+)/);
  if (deviceMatch) {
    context.device_id = deviceMatch[1];
  }

  // Извлекаем wo_id из пути: /work-orders/{id}
  const woMatch = path.match(/\/work-orders\/([a-zA-Z0-9_-]+)/);
  if (woMatch) {
    context.wo_id = woMatch[1];
  }

  // Из параметров URL
  const params = new URLSearchParams(search);
  if (params.get('device_id')) {
    context.device_id = params.get('device_id')!;
  }
  if (params.get('wo_id')) {
    context.wo_id = params.get('wo_id')!;
  }
  if (params.get('site_id')) {
    context.site_id = params.get('site_id')!;
  }

  return context;
}

/**
 * Генерирует уникальный ID для сообщения.
 */
export function generateMessageId(): string {
  return `msg_${Date.now()}_${Math.random().toString(36).slice(2, 9)}`;
}
