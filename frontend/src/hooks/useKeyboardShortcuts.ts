// ═══════════════════════════════════════════════════════════════════════
// useKeyboardShortcuts — глобальный хук для регистрации шорткатов
// UX-14.1.8: Keyboard Shortcuts
// P3-3.2: Power User Keyboard Shortcuts
//
// Особенности:
//   - Автоматически разрегистрирует listeners при unmount
//   - Не срабатывает внутри input/textarea/select/contenteditable
//   - Поддержка cross-platform: ⌘ (meta) + Ctrl
//   - Порядок регистрации = приоритет (первый подходящий выигрывает)
//   - ? — переключение help-модалки
//   - Escape — сброс фокуса из текстовых полей
//   - Alt+1..9 — быстрый доступ к последним Work Orders
//   - / — фокус на поиск
// ═══════════════════════════════════════════════════════════════════════

import { useEffect, useCallback, useState } from 'react';

// ═══════════════════════════════════════════════════════════════════════
// Types
// ═══════════════════════════════════════════════════════════════════════

export type ShortcutCategory = 'navigation' | 'actions' | 'modals';

export interface Shortcut {
  /** Клавиша (регистронезависимая) — 'k', 'n', 'Escape', '?' и т.д. */
  key: string;
  /** Требуется ли Ctrl */
  ctrl?: boolean;
  /** Требуется ли Meta (⌘ на Mac) */
  meta?: boolean;
  /** Требуется ли Shift */
  shift?: boolean;
  /** Требуется ли Alt (для Alt+1..9) */
  alt?: boolean;
  /** Обработчик */
  handler: () => void;
  /** Человекочитаемое описание */
  description: string;
  /** Группа для отображения в Cheatsheet */
  category: ShortcutCategory;
}

// ═══════════════════════════════════════════════════════════════════════
// Types для упрощённого help-компонента
// ═══════════════════════════════════════════════════════════════════════

export interface ShortcutDef {
  key: string;
  description: string;
}

// ═══════════════════════════════════════════════════════════════════════
// Hook
// ═══════════════════════════════════════════════════════════════════════

/**
 * Регистрирует глобальные клавиатурные шорткаты.
 * Шорткаты не срабатывают внутри текстовых полей.
 * Автоматически очищает listeners при unmount.
 * Возвращает { showHelp, setShowHelp } для управления help-модалкой.
 *
 * Встроенные шорткаты:
 *   - ? — переключение help
 *   - Escape (в поле ввода) — сброс фокуса
 *
 * @example
 * ```tsx
 * const { showHelp, setShowHelp } = useKeyboardShortcuts([
 *   { key: 'k', ctrl: true, meta: true, handler: togglePalette, description: 'Command Palette', category: 'actions' },
 *   { key: '1', alt: true, handler: () => navigate('/work-orders/1'), description: 'Work Order #1', category: 'navigation' },
 * ]);
 * ```
 */
export function useKeyboardShortcuts(shortcuts: Shortcut[]): { showHelp: boolean; setShowHelp: (v: boolean) => void } {
  const [showHelp, setShowHelp] = useState(false);

  const handleKeyDown = useCallback((e: KeyboardEvent) => {
    // ── Игнорируем, если фокус внутри текстового поля ───────────
    const target = e.target as HTMLElement;
    const tagName = target.tagName.toLowerCase();

    if (
      tagName === 'input' ||
      tagName === 'textarea' ||
      tagName === 'select' ||
      target.isContentEditable
    ) {
      // Escape — сброс фокуса из поля
      if (e.key === 'Escape') {
        (target as HTMLInputElement).blur();
      }
      return;
    }

    // ── Встроенные глобальные шорткаты ───────────────────────────

    // ? — переключение help-модалки (без модификаторов)
    if (e.key === '?' && !e.metaKey && !e.ctrlKey && !e.altKey && !e.shiftKey) {
      e.preventDefault();
      e.stopPropagation();
      setShowHelp((prev) => !prev);
      return;
    }

    // ── Ищем первый подходящий шорткат из пользовательских ───────
    for (const s of shortcuts) {
      if (e.key.toLowerCase() !== s.key.toLowerCase()) continue;

      // Проверка Shift
      if (s.shift && !e.shiftKey) continue;
      if (!s.shift && e.shiftKey) continue;

      // Проверка Alt
      if (s.alt && !e.altKey) continue;
      if (!s.alt && e.altKey) continue;

      // Проверка cross-platform модификаторов (⌘ / Ctrl)
      const needsMod = s.ctrl || s.meta;

      if (needsMod) {
        // Если нужен модификатор — проверяем metaKey OR ctrlKey
        if (!e.metaKey && !e.ctrlKey) continue;

        // Если нужен ТОЛЬКО ctrl (meta не указан) — meta быть не должен
        if (s.ctrl && !s.meta && e.metaKey && !e.ctrlKey) continue;

        // Если нужен ТОЛЬКО meta (ctrl не указан) — ctrl быть не должен
        if (s.meta && !s.ctrl && e.ctrlKey && !e.metaKey) continue;
      } else {
        // Без ctrl/meta — убеждаемся, что они не зажаты
        if (e.metaKey || e.ctrlKey) continue;
      }

      // ── Все проверки пройдены — вызываем handler ──────────────
      e.preventDefault();
      e.stopPropagation();
      s.handler();
      return;
    }
  }, [shortcuts]);

  useEffect(() => {
    document.addEventListener('keydown', handleKeyDown);
    return () => document.removeEventListener('keydown', handleKeyDown);
  }, [handleKeyDown]);

  return { showHelp, setShowHelp };
}
