// ═══════════════════════════════════════════════════════════════════════
// useAIAssistant Hook (P2-1.2)
//
// Управляет состоянием AI Assistant чата:
//   - Список сообщений (user + assistant)
//   - SSE стриминг ответов через backend proxy
//   - Контекст текущей страницы (device_id, wo_id, page)
//   - Отправка обратной связи (like/dislike)
//
// Compliance:
//   - OWASP ASVS V7.1 (Error handling — no info leakage)
//   - IEC 62443 SR 7.1 (Timeout — abort controller)
// ═══════════════════════════════════════════════════════════════════════

import { useState, useRef, useCallback, useEffect } from 'react';
import {
  sendChatMessage,
  sendFeedback,
  extractContextFromURL,
  generateMessageId,
  type AIChatMessage,
  type AIChatContext,
  type AIStreamEvent,
} from '../lib/deepseek';

// ─── Types ────────────────────────────────────────────────────────────

export interface ChatMessage {
  id: string;
  role: 'user' | 'assistant';
  content: string;
  timestamp: Date;
  /** ID для обратной связи (только assistant) */
  feedbackId?: string;
  /** Статус: idle | streaming | done | error */
  status: 'idle' | 'streaming' | 'done' | 'error';
  /** Ошибка если статус error */
  error?: string;
  /** Обратная связь */
  feedback?: 'like' | 'dislike' | null;
}

export interface AIAssistantState {
  /** Все сообщения чата */
  messages: ChatMessage[];
  /** Идёт ли загрузка ответа */
  isLoading: boolean;
  /** Ошибка последнего запроса */
  error: string | null;
  /** Контекст текущей страницы */
  context: AIChatContext;
}

export interface AIAssistantActions {
  /** Отправить сообщение */
  sendMessage: (text: string) => Promise<void>;
  /** Отправить обратную связь */
  sendFeedback: (messageId: string, score: 'like' | 'dislike') => Promise<void>;
  /** Очистить историю чата */
  clearHistory: () => void;
  /** Обновить контекст (при смене страницы) */
  refreshContext: () => void;
  /** Повторить последний запрос */
  retry: () => Promise<void>;
  /** Отменить текущий запрос */
  cancel: () => void;
}

export type UseAIAssistantResult = AIAssistantState & AIAssistantActions;

// ─── Hook ─────────────────────────────────────────────────────────────

const MAX_HISTORY = 20;
const STORAGE_KEY = 'ai_assistant_messages';

/**
 * Hook для управления AI Assistant чатом.
 *
 * Автоматически:
 * - Извлекает контекст из URL (device_id, wo_id, page)
 * - Сохраняет историю в sessionStorage
 * - Стримит SSE ответы в реальном времени
 */
export function useAIAssistant(): UseAIAssistantResult {
  const [messages, setMessages] = useState<ChatMessage[]>(() => {
    try {
      const saved = sessionStorage.getItem(STORAGE_KEY);
      if (saved) {
        const parsed = JSON.parse(saved);
        return parsed.map((m: Record<string, unknown>) => ({
          ...m,
          timestamp: new Date(m.timestamp as string),
        }));
      }
    } catch {
      // ignore
    }
    return [];
  });

  const [isLoading, setIsLoading] = useState(false);
  const [error, setError] = useState<string | null>(null);
  const [context, setContext] = useState<AIChatContext>(() => extractContextFromURL());

  const abortRef = useRef<AbortController | null>(null);
  const lastRequestRef = useRef<{ message: string } | null>(null);
  const messagesRef = useRef<ChatMessage[]>(messages);
  messagesRef.current = messages;

  // Сохраняем историю в sessionStorage
  useEffect(() => {
    try {
      sessionStorage.setItem(STORAGE_KEY, JSON.stringify(messages.slice(-50)));
    } catch {
      // ignore QUOTA_EXCEEDED
    }
  }, [messages]);

  // Обновляем контекст при фокусе страницы (пользователь мог перейти на другую страницу)
  useEffect(() => {
    const handleFocus = () => {
      setContext(extractContextFromURL());
    };
    window.addEventListener('focus', handleFocus);
    return () => window.removeEventListener('focus', handleFocus);
  }, []);

  // Очищаем контекст при демонтаже
  useEffect(() => {
    return () => {
      abortRef.current?.abort();
    };
  }, []);

  /**
   * Отправляет сообщение и стримит ответ.
   */
  const sendMessage = useCallback(async (text: string) => {
    const trimmed = text.trim();
    if (!trimmed || isLoading) return;

    // Отменяем предыдущий запрос если был
    abortRef.current?.abort();

    const userMessage: ChatMessage = {
      id: generateMessageId(),
      role: 'user',
      content: trimmed,
      timestamp: new Date(),
      status: 'done',
    };

    const assistantMessage: ChatMessage = {
      id: generateMessageId(),
      role: 'assistant',
      content: '',
      timestamp: new Date(),
      status: 'streaming',
      feedback: null,
    };

    setMessages(prev => [...prev, userMessage, assistantMessage]);
    setIsLoading(true);
    setError(null);

    const abortController = new AbortController();
    abortRef.current = abortController;

    lastRequestRef.current = { message: trimmed };

    // Берём последние MAX_HISTORY сообщений для контекста
    const currentMessages = messagesRef.current;
    const recentMessages = currentMessages
      .slice(-MAX_HISTORY)
      .filter(m => m.status === 'done' && m.content)
      .map(m => ({
        role: m.role as 'user' | 'assistant',
        content: m.content,
      } satisfies AIChatMessage));

    try {
      let fullContent = '';

      await sendChatMessage(
        trimmed,
        context,
        recentMessages,
        (event: AIStreamEvent) => {
          if (event.type === 'chunk') {
            fullContent += event.content;
            setMessages(prev =>
              prev.map(m =>
                m.id === assistantMessage.id
                  ? { ...m, content: fullContent }
                  : m
              )
            );
          }
        },
        abortController.signal
      );

      // Отмечаем сообщение как завершённое
      setMessages(prev =>
        prev.map(m =>
          m.id === assistantMessage.id
            ? { ...m, status: 'done', feedbackId: assistantMessage.id }
            : m
        )
      );
    } catch (err) {
      if (err instanceof DOMException && err.name === 'AbortError') {
        // Отменено пользователем — удаляем сообщение assistant
        setMessages(prev =>
          prev.filter(m => m.id !== assistantMessage.id)
        );
        return;
      }

      const errorMessage = err instanceof Error ? err.message : 'Failed to get AI response';
      setMessages(prev =>
        prev.map(m =>
          m.id === assistantMessage.id
            ? { ...m, status: 'error' as const, content: '', error: errorMessage }
            : m
        ).filter(m => m.id !== assistantMessage.id || m.content || m.error)
      );
      setError(errorMessage);
    } finally {
      setIsLoading(false);
      abortRef.current = null;
    }
  }, [isLoading, context]);

  /**
   * Отправляет обратную связь.
   */
  const handleSendFeedback = useCallback(async (messageId: string, score: 'like' | 'dislike') => {
    // Оптимистичное обновление UI
    setMessages(prev =>
      prev.map(m =>
        m.id === messageId ? { ...m, feedback: score } : m
      )
    );

    try {
      await sendFeedback(messageId, score);
    } catch {
      // Откатываем при ошибке
      setMessages(prev =>
        prev.map(m =>
          m.id === messageId ? { ...m, feedback: null } : m
        )
      );
    }
  }, []);

  /**
   * Очищает историю чата.
   */
  const clearHistory = useCallback(() => {
    abortRef.current?.abort();
    setMessages([]);
    setError(null);
    setIsLoading(false);
    try {
      sessionStorage.removeItem(STORAGE_KEY);
    } catch {
      // ignore
    }
  }, []);

  /**
   * Обновляет контекст текущей страницы.
   */
  const refreshContext = useCallback(() => {
    setContext(extractContextFromURL());
  }, []);

  /**
   * Повторяет последний запрос.
   */
  const retry = useCallback(async () => {
    const last = lastRequestRef.current;
    if (last) {
      await sendMessage(last.message);
    }
  }, [sendMessage]);

  /**
   * Отменяет текущий запрос.
   */
  const cancel = useCallback(() => {
    abortRef.current?.abort();
    abortRef.current = null;
  }, []);

  return {
    messages,
    isLoading,
    error,
    context,
    sendMessage,
    sendFeedback: handleSendFeedback,
    clearHistory,
    refreshContext,
    retry,
    cancel,
  };
}
