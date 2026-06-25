// ═══════════════════════════════════════════════════════════════════════
// ShortcutsCheatsheet — модальное окно со списком всех шорткатов
// UX-14.1.8: Keyboard Shortcuts
//
// Особенности:
//   - Группировка: Navigation, Actions, Modals
//   - Таблица с key комбинациями и описаниями
//   - Открывается по ⌘/ / Ctrl+/ или ?
// ═══════════════════════════════════════════════════════════════════════

import React, { useMemo } from 'react';
import { Modal } from './Modal';
import { Command } from 'lucide-react';
import { useTranslation } from 'react-i18next';
import type { Shortcut, ShortcutCategory } from '../../hooks/useKeyboardShortcuts';

// ═══════════════════════════════════════════════════════════════════════
// Types
// ═══════════════════════════════════════════════════════════════════════

interface ShortcutsCheatsheetProps {
  isOpen: boolean;
  onClose: () => void;
  shortcuts: Shortcut[];
}

interface CategoryMeta {
  key: ShortcutCategory;
  labelKey: string;
}

// ═══════════════════════════════════════════════════════════════════════
// Constants
// ═══════════════════════════════════════════════════════════════════════

const CATEGORIES: CategoryMeta[] = [
  { key: 'navigation', labelKey: 'shortcuts_navigation' },
  { key: 'actions', labelKey: 'shortcuts_actions' },
  { key: 'modals', labelKey: 'shortcuts_modals' },
];

const FALLBACK_LABELS: Record<ShortcutCategory, string> = {
  navigation: 'Navigation',
  actions: 'Actions',
  modals: 'Modals',
};

// ═══════════════════════════════════════════════════════════════════════
// Helpers
// ═══════════════════════════════════════════════════════════════════════

/**
 * Форматирует комбинацию клавиш в читаемый вид.
 * Примеры: ⌘K, Ctrl+N, Esc, ?
 */
function formatKeyCombo(shortcut: Shortcut): string {
  const isMac = navigator.platform.toLowerCase().includes('mac');
  const parts: string[] = [];

  // Модификаторы
  if (shortcut.shift) {
    parts.push(isMac ? '⇧' : 'Shift');
  }

  if (shortcut.meta && shortcut.ctrl) {
    // Cross-platform: ⌘ на Mac, Ctrl на Windows/Linux
    parts.push(isMac ? '⌘' : 'Ctrl');
  } else if (shortcut.meta) {
    parts.push(isMac ? '⌘' : 'Meta');
  } else if (shortcut.ctrl) {
    parts.push('Ctrl');
  }

  // Клавиша
  const keyMap: Record<string, string> = {
    escape: 'Esc',
    enter: '↵',
    arrowup: '↑',
    arrowdown: '↓',
    arrowleft: '←',
    arrowright: '→',
    ' ': 'Space',
    '/': '/',
    ',': ',',
  };

  const displayKey = keyMap[shortcut.key.toLowerCase()] ?? shortcut.key.toUpperCase();
  parts.push(displayKey);

  return parts.join(isMac ? '' : '+');
}

// ═══════════════════════════════════════════════════════════════════════
// Component
// ═══════════════════════════════════════════════════════════════════════

export function ShortcutsCheatsheet({ isOpen, onClose, shortcuts }: ShortcutsCheatsheetProps) {
  const { t } = useTranslation();

  // Группируем шорткаты по категориям
  const grouped = useMemo(() => {
    const groups = new Map<ShortcutCategory, Shortcut[]>();

    for (const s of shortcuts) {
      const list = groups.get(s.category) ?? [];
      list.push(s);
      groups.set(s.category, list);
    }

    return CATEGORIES
      .filter((cat) => (groups.get(cat.key)?.length ?? 0) > 0)
      .map((cat) => ({
        ...cat,
        items: groups.get(cat.key)!,
      }));
  }, [shortcuts]);

  return (
    <Modal isOpen={isOpen} onClose={onClose} title="Keyboard Shortcuts" size="lg">
      <div className="space-y-6">
        {grouped.map((group) => (
          <section key={group.key}>
            <h3 className="text-sm font-semibold text-slate-500 dark:text-slate-400 uppercase tracking-wider mb-3">
              {t(group.labelKey, FALLBACK_LABELS[group.key])}
            </h3>

            <div className="divide-y divide-slate-100 dark:divide-slate-700/50 border border-slate-200 dark:border-slate-700 rounded-xl overflow-hidden">
              {group.items.map((shortcut, idx) => (
                <div
                  key={`${shortcut.key}-${idx}`}
                  className="flex items-center justify-between px-4 py-2.5 bg-white dark:bg-slate-800/50"
                >
                  <span className="text-sm text-slate-700 dark:text-slate-200">
                    {shortcut.description}
                  </span>

                  <kbd className="inline-flex items-center gap-0.5 px-2.5 py-1 text-xs font-mono font-medium text-slate-600 dark:text-slate-300 bg-slate-100 dark:bg-slate-800 rounded-md border border-slate-200 dark:border-slate-700 shadow-sm whitespace-nowrap">
                    {/* ⌘ icon for Meta shortcuts */}
                    {shortcut.meta && (
                      <Command className="w-3 h-3 inline-block -ml-0.5" />
                    )}
                    {formatKeyCombo(shortcut)}
                  </kbd>
                </div>
              ))}
            </div>
          </section>
        ))}
      </div>
    </Modal>
  );
}
