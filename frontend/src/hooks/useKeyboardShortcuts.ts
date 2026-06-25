// ═══════════════════════════════════════════════════════════════════════
// useKeyboardShortcuts — глобальный хук для регистрации шорткатов
// UX-14.1.8: Keyboard Shortcuts
//
// Особенности:
//   - Автоматически разрегистрирует listeners при unmount
//   - Не срабатывает внутри input/textarea/select/contenteditable
//   - Поддержка cross-platform: ⌘ (meta) + Ctrl
//   - Порядок регистрации = приоритет (первый подходящий выигрывает)
// ═══════════════════════════════════════════════════════════════════════

import { useEffect } from 'react';

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
  /** Обработчик */
  handler: () => void;
  /** Человекочитаемое описание */
  description: string;
  /** Группа для отображения в Cheatsheet */
  category: ShortcutCategory;
}

// ═══════════════════════════════════════════════════════════════════════
// Hook
// ═══════════════════════════════════════════════════════════════════════

/**
 * Регистрирует глобальные клавиатурные шорткаты.
 * Шорткаты не срабатывают внутри текстовых полей.
 * Автоматически очищает listeners при unmount.
 *
 * @example
 * ```tsx
 * useKeyboardShortcuts([
 *   { key: 'k', ctrl: true, meta: true, handler: togglePalette, description: 'Command Palette', category: 'actions' },
 * ]);
 * ```
 */
export function useKeyboardShortcuts(shortcuts: Shortcut[]): void {
  useEffect(() => {
    const handleKeyDown = (e: KeyboardEvent) => {
      // ── Игнорируем, если фокус внутри текстового поля ───────────
      const target = e.target as HTMLElement;
      const tagName = target.tagName.toLowerCase();

      if (
        tagName === 'input' ||
        tagName === 'textarea' ||
        tagName === 'select' ||
        target.isContentEditable
      ) {
        return;
      }

      // ── Ищем первый подходящий шорткат ──────────────────────────
      for (const s of shortcuts) {
        if (e.key.toLowerCase() !== s.key.toLowerCase()) continue;

        // Проверка Shift
        if (s.shift && !e.shiftKey) continue;

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
          // Без модификаторов — убеждаемся, что ни один не зажат
          if (e.metaKey || e.ctrlKey || e.shiftKey || e.altKey) continue;
        }

        // ── Все проверки пройдены — вызываем handler ──────────────
        e.preventDefault();
        e.stopPropagation();
        s.handler();
        return;
      }
    };

    document.addEventListener('keydown', handleKeyDown);
    return () => document.removeEventListener('keydown', handleKeyDown);
  }, [shortcuts]);
}
