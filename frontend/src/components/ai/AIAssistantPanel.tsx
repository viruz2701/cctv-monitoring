// ═══════════════════════════════════════════════════════════════════════
// AIAssistantPanel — боковая панель чата с DeepSeek AI (P2-1.2)
//
// Возможности:
//   - Чат с AI Assistant через backend proxy (SSE streaming)
//   - Context-aware ответы (текущая страница, device, WO)
//   - Markdown рендеринг
//   - RCA suggestions
//   - Обратная связь (like/dislike)
//   - Очистка истории
//
// Compliance:
//   - OWASP ASVS V7.1 (Error handling — no info leakage)
//   - WCAG 2.1 AA (Keyboard navigation)
//   - IEC 62443 SR 7.1 (Timeout controls)
// ═══════════════════════════════════════════════════════════════════════

import React, { useState, useRef, useEffect, useCallback } from 'react';
import ReactMarkdown from 'react-markdown';
import remarkGfm from 'remark-gfm';
import {
  Bot,
  Send,
  X,
  ThumbsUp,
  ThumbsDown,
  Trash2,
  Loader2,
  Sparkles,
  PanelRightOpen,
  PanelRightClose,
  RefreshCw,
} from '../ui/Icons';
import { useAIAssistant, type ChatMessage } from '../../hooks/useAIAssistant';

// ─── Props ────────────────────────────────────────────────────────────

interface AIAssistantPanelProps {
  /** Начальное состояние (открыт/закрыт) */
  defaultOpen?: boolean;
  /** Колбэк при смене состояния */
  onToggle?: (open: boolean) => void;
}

// ─── Component ────────────────────────────────────────────────────────

export function AIAssistantPanel({ defaultOpen = false, onToggle }: AIAssistantPanelProps) {
  const [isOpen, setIsOpen] = useState(defaultOpen);
  const [inputValue, setInputValue] = useState('');
  const inputRef = useRef<HTMLInputElement>(null);
  const messagesEndRef = useRef<HTMLDivElement>(null);

  const {
    messages,
    isLoading,
    error,
    context,
    sendMessage,
    sendFeedback,
    clearHistory,
    retry,
    cancel,
  } = useAIAssistant();

  // Auto-scroll to bottom on new messages
  const scrollToBottom = useCallback(() => {
    messagesEndRef.current?.scrollIntoView({ behavior: 'smooth' });
  }, []);

  useEffect(() => {
    scrollToBottom();
  }, [messages, scrollToBottom]);

  // Focus input when panel opens
  useEffect(() => {
    if (isOpen) {
      setTimeout(() => inputRef.current?.focus(), 100);
    }
  }, [isOpen]);

  const handleToggle = useCallback(() => {
    const next = !isOpen;
    setIsOpen(next);
    onToggle?.(next);
  }, [isOpen, onToggle]);

  const handleSubmit = useCallback(async (e: React.FormEvent) => {
    e.preventDefault();
    if (!inputValue.trim() || isLoading) return;
    const text = inputValue;
    setInputValue('');
    await sendMessage(text);
  }, [inputValue, isLoading, sendMessage]);

  const handleKeyDown = useCallback((e: React.KeyboardEvent) => {
    if (e.key === 'Enter' && !e.shiftKey) {
      e.preventDefault();
      handleSubmit(e);
    }
    if (e.key === 'Escape') {
      handleToggle();
    }
  }, [handleSubmit, handleToggle]);

  const handleFeedback = useCallback((messageId: string, score: 'like' | 'dislike') => {
    sendFeedback(messageId, score);
  }, [sendFeedback]);

  // ── Context badge ───────────────────────────────────────────────
  const contextLabel = getContextLabel(context);

  return (
    <>
      {/* Toggle button */}
      <button
        onClick={handleToggle}
        className="fixed bottom-4 right-4 z-50 flex h-12 w-12 items-center justify-center rounded-full
          bg-blue-600 text-white shadow-lg hover:bg-blue-700 focus:outline-none focus:ring-2
          focus:ring-blue-500 focus:ring-offset-2 transition-all duration-200
          dark:focus:ring-offset-slate-900"
        aria-label={isOpen ? 'Закрыть AI Assistant' : 'Открыть AI Assistant'}
        aria-expanded={isOpen}
        title="AI Assistant"
      >
        {isOpen ? <PanelRightClose className="h-5 w-5" /> : <Bot className="h-5 w-5" />}
      </button>

      {/* Panel overlay (mobile) */}
      {isOpen && (
        <div
          className="fixed inset-0 z-40 bg-black/30 lg:hidden"
          onClick={handleToggle}
          aria-hidden="true"
        />
      )}

      {/* Chat panel */}
      <aside
        className={`fixed bottom-0 right-0 z-50 flex h-[85vh] w-full flex-col bg-white shadow-xl
          transition-transform duration-300 dark:bg-slate-900
          sm:w-96 sm:bottom-4 sm:right-4 sm:h-[75vh] sm:rounded-2xl sm:border sm:border-slate-200
          sm:dark:border-slate-700
          ${isOpen ? 'translate-y-0' : 'translate-y-full sm:translate-y-[calc(100%+2rem)]'}`}
        role="complementary"
        aria-label="AI Assistant чат"
      >
        {/* ── Header ─────────────────────────────────────────────────── */}
        <div className="flex items-center justify-between border-b border-slate-200 px-4 py-3 dark:border-slate-700">
          <div className="flex items-center gap-2">
            <Bot className="h-5 w-5 text-blue-600 dark:text-blue-400" />
            <div>
              <h2 className="text-sm font-semibold text-slate-900 dark:text-slate-100">
                AI Assistant
              </h2>
              {contextLabel && (
                <p className="text-[10px] text-slate-500 dark:text-slate-400">
                  {contextLabel}
                </p>
              )}
            </div>
          </div>
          <div className="flex items-center gap-1">
            {messages.length > 0 && (
              <button
                onClick={clearHistory}
                className="rounded-lg p-1.5 text-slate-400 hover:bg-slate-100 hover:text-red-500
                  dark:hover:bg-slate-800 dark:hover:text-red-400 transition-colors"
                title="Очистить историю"
                aria-label="Очистить историю чата"
              >
                <Trash2 className="h-4 w-4" />
              </button>
            )}
            <button
              onClick={handleToggle}
              className="rounded-lg p-1.5 text-slate-400 hover:bg-slate-100 hover:text-slate-600
                dark:hover:bg-slate-800 dark:hover:text-slate-300 transition-colors"
              aria-label="Закрыть"
            >
              <X className="h-4 w-4" />
            </button>
          </div>
        </div>

        {/* ── Messages ────────────────────────────────────────────────── */}
        <div className="flex-1 overflow-y-auto px-4 py-3 space-y-4">
          {messages.length === 0 && !isLoading && (
            <div className="flex h-full flex-col items-center justify-center text-center">
              <Sparkles className="h-10 w-10 text-blue-400 mb-3" />
              <h3 className="text-sm font-medium text-slate-700 dark:text-slate-300">
                AI Assistant
              </h3>
              <p className="mt-1 text-xs text-slate-500 dark:text-slate-400 max-w-[240px]">
                Спрашивайте о диагностике устройств, Work Orders, RCA или compliance стандартах.
              </p>
              {contextLabel && (
                <div className="mt-3 rounded-full bg-blue-50 px-3 py-1 text-[10px] text-blue-600 dark:bg-blue-900/30 dark:text-blue-400">
                  {contextLabel}
                </div>
              )}
              {/* Quick prompts */}
              <div className="mt-4 flex flex-wrap justify-center gap-2 px-4">
                {QUICK_PROMPTS.map((prompt) => (
                  <button
                    key={prompt}
                    onClick={() => {
                      setInputValue(prompt);
                      inputRef.current?.focus();
                    }}
                    className="rounded-full border border-slate-200 px-3 py-1 text-[11px] text-slate-500
                      hover:border-blue-300 hover:text-blue-600 hover:bg-blue-50
                      dark:border-slate-600 dark:text-slate-400
                      dark:hover:border-blue-500 dark:hover:text-blue-400 dark:hover:bg-blue-900/20
                      transition-colors"
                  >
                    {prompt}
                  </button>
                ))}
              </div>
            </div>
          )}

          {messages.map((msg) => (
            <ChatBubble
              key={msg.id}
              message={msg}
              onFeedback={handleFeedback}
              onRetry={retry}
            />
          ))}

          {/* Loading indicator */}
          {isLoading && messages[messages.length - 1]?.status === 'streaming' && (
            <div className="flex items-center gap-2 text-xs text-slate-400 pl-2">
              <Loader2 className="h-3 w-3 animate-spin" />
              <span>AI печатает...</span>
              <button
                onClick={cancel}
                className="ml-2 text-red-400 hover:text-red-500 underline"
              >
                Отмена
              </button>
            </div>
          )}

          {/* Error state */}
          {error && !isLoading && (
            <div className="flex items-center gap-2 rounded-lg bg-red-50 p-3 text-xs text-red-600 dark:bg-red-900/20 dark:text-red-400">
              <span>{error}</span>
              <button
                onClick={retry}
                className="ml-auto flex items-center gap-1 font-medium text-red-600 hover:text-red-700
                  dark:text-red-400 dark:hover:text-red-300"
              >
                <RefreshCw className="h-3 w-3" />
                Повторить
              </button>
            </div>
          )}

          <div ref={messagesEndRef} />
        </div>

        {/* ── Input ──────────────────────────────────────────────────── */}
        <div className="border-t border-slate-200 p-3 dark:border-slate-700">
          <form onSubmit={handleSubmit} className="flex items-center gap-2">
            <input
              ref={inputRef}
              type="text"
              value={inputValue}
              onChange={(e) => setInputValue(e.target.value)}
              onKeyDown={handleKeyDown}
              placeholder="Спросить AI Assistant..."
              disabled={isLoading}
              maxLength={4096}
              className="flex-1 rounded-xl border border-slate-300 bg-slate-50 px-4 py-2 text-sm
                placeholder:text-slate-400 focus:border-blue-500 focus:outline-none focus:ring-1
                focus:ring-blue-500 disabled:opacity-50
                dark:border-slate-600 dark:bg-slate-800 dark:text-slate-100
                dark:placeholder:text-slate-500 dark:focus:border-blue-400"
              aria-label="Введите сообщение"
            />
            <button
              type="submit"
              disabled={!inputValue.trim() || isLoading}
              className="flex h-9 w-9 items-center justify-center rounded-xl bg-blue-600 text-white
                hover:bg-blue-700 disabled:opacity-40 disabled:cursor-not-allowed
                focus:outline-none focus:ring-2 focus:ring-blue-500 focus:ring-offset-2
                dark:focus:ring-offset-slate-900 transition-colors"
              aria-label="Отправить"
            >
              {isLoading ? (
                <Loader2 className="h-4 w-4 animate-spin" />
              ) : (
                <Send className="h-4 w-4" />
              )}
            </button>
          </form>
        </div>
      </aside>
    </>
  );
}

// ─── Chat Bubble ──────────────────────────────────────────────────────

interface ChatBubbleProps {
  message: ChatMessage;
  onFeedback: (messageId: string, score: 'like' | 'dislike') => void;
  onRetry: () => void;
}

const ChatBubble = React.memo(function ChatBubble({ message, onFeedback, onRetry }: ChatBubbleProps) {
  const isUser = message.role === 'user';
  const isAssistant = message.role === 'assistant';
  const showFeedback = isAssistant && message.status === 'done' && message.content;

  return (
    <div className={`flex ${isUser ? 'justify-end' : 'justify-start'}`}>
      <div
        className={`max-w-[85%] rounded-2xl px-3 py-2 ${
          isUser
            ? 'bg-blue-600 text-white'
            : 'bg-slate-100 text-slate-800 dark:bg-slate-800 dark:text-slate-200'
        }`}
      >
        {isUser ? (
          <p className="text-sm whitespace-pre-wrap">{message.content}</p>
        ) : message.status === 'streaming' && !message.content ? (
          <div className="flex items-center gap-1 py-1">
            <span className="h-2 w-2 animate-bounce rounded-full bg-slate-400" style={{ animationDelay: '0ms' }} />
            <span className="h-2 w-2 animate-bounce rounded-full bg-slate-400" style={{ animationDelay: '150ms' }} />
            <span className="h-2 w-2 animate-bounce rounded-full bg-slate-400" style={{ animationDelay: '300ms' }} />
          </div>
        ) : (
          <div className="prose prose-sm max-w-none dark:prose-invert prose-headings:text-sm
            prose-headings:font-semibold prose-a:text-blue-600 prose-code:bg-slate-200
            prose-code:px-1 prose-code:rounded dark:prose-code:bg-slate-700 prose-pre:bg-slate-800
            prose-pre:text-slate-100">
            <ReactMarkdown remarkPlugins={[remarkGfm]}>
              {message.content}
            </ReactMarkdown>
          </div>
        )}

        {/* Error state */}
        {message.status === 'error' && (
          <div className="mt-1">
            <p className="text-xs text-red-400">{message.error || 'Ошибка получения ответа'}</p>
            <button
              onClick={onRetry}
              className="mt-1 flex items-center gap-1 text-xs font-medium text-red-400 hover:text-red-300"
            >
              <RefreshCw className="h-3 w-3" />
              Повторить
            </button>
          </div>
        )}

        {/* Feedback buttons */}
        {showFeedback && (
          <div className="mt-2 flex items-center gap-2 border-t border-slate-200 pt-2 dark:border-slate-700">
            <span className="text-[10px] text-slate-400">Ответ полезен?</span>
            <button
              onClick={() => onFeedback(message.id, 'like')}
              className={`rounded p-1 transition-colors ${
                message.feedback === 'like'
                  ? 'bg-green-100 text-green-600 dark:bg-green-900/30 dark:text-green-400'
                  : 'text-slate-400 hover:text-green-500 hover:bg-green-50 dark:hover:bg-green-900/20'
              }`}
              aria-label="Полезно"
              title="Полезно"
            >
              <ThumbsUp className="h-3.5 w-3.5" />
            </button>
            <button
              onClick={() => onFeedback(message.id, 'dislike')}
              className={`rounded p-1 transition-colors ${
                message.feedback === 'dislike'
                  ? 'bg-red-100 text-red-600 dark:bg-red-900/30 dark:text-red-400'
                  : 'text-slate-400 hover:text-red-500 hover:bg-red-50 dark:hover:bg-red-900/20'
              }`}
              aria-label="Не полезно"
              title="Не полезно"
            >
              <ThumbsDown className="h-3.5 w-3.5" />
            </button>
          </div>
        )}
      </div>
    </div>
  );
});

// ─── Helpers ──────────────────────────────────────────────────────────

const QUICK_PROMPTS = [
  'Диагностика устройства',
  'Причины сбоя камеры',
  'Что такое SLA?',
  'Анализ инцидента',
];

/**
 * Возвращает читаемый контекст для отображения в заголовке.
 */
function getContextLabel(context: { current_page?: string; device_id?: string; wo_id?: string; site_id?: string }): string | null {
  const parts: string[] = [];

  if (context.device_id) {
    parts.push(`Устройство: ${context.device_id}`);
  }
  if (context.wo_id) {
    parts.push(`WO: ${context.wo_id}`);
  }
  if (context.site_id) {
    parts.push(`Объект: ${context.site_id}`);
  }

  if (parts.length === 0 && context.current_page) {
    const pageName = getPageName(context.current_page);
    if (pageName) {
      parts.push(pageName);
    }
  }

  return parts.length > 0 ? parts.join(' · ') : null;
}

/**
 * Возвращает человекочитаемое имя страницы из URL пути.
 */
function getPageName(path: string): string | null {
  const pageMap: Record<string, string> = {
    '/dashboard': 'Дашборд',
    '/devices': 'Устройства',
    '/work-orders': 'Work Orders',
    '/alarms': 'Тревоги',
    '/analytics': 'Аналитика',
    '/compliance': 'Compliance',
    '/settings': 'Настройки',
    '/reports': 'Отчёты',
  };

  // Exact match
  if (pageMap[path]) {
    return pageMap[path];
  }

  // Prefix match
  for (const [prefix, name] of Object.entries(pageMap)) {
    if (path.startsWith(prefix)) {
      return name;
    }
  }

  return null;
}
