// ═══════════════════════════════════════════════════════════════════════
// KeyboardShortcutsHelp — модалка со списком горячих клавиш.
// P3-3.2: Power User Keyboard Shortcuts
//
// Особенности:
//   - Открывается по ?
//   - Закрывается по клику вне или на кнопку X
//   - Адаптивная тёмная/светлая тема
// ═══════════════════════════════════════════════════════════════════════

import React from 'react';
import { useTranslation } from 'react-i18next';
import { X, Keyboard } from '../ui/Icons';
import type { ShortcutDef } from '../../hooks/useKeyboardShortcuts';

// ═══════════════════════════════════════════════════════════════════════
// Types
// ═══════════════════════════════════════════════════════════════════════

interface KeyboardShortcutsHelpProps {
  isOpen: boolean;
  onClose: () => void;
  shortcuts: ShortcutDef[];
}

// ═══════════════════════════════════════════════════════════════════════
// Helpers
// ═══════════════════════════════════════════════════════════════════════

/**
 * Форматирует комбинацию клавиш в читаемый kbd-вид.
 * Примеры: ⌘K, Ctrl+N, ?, Alt+1
 */
function formatKeyLabel(raw: string): string {
  const isMac = navigator.platform.toLowerCase().includes('mac');

  const parts = raw.split('+').map((part) => {
    const p = part.trim();

    // Символьные модификаторы
    if (p === 'Meta') return isMac ? '⌘' : 'Meta';
    if (p === 'Ctrl') return isMac ? '⌃' : 'Ctrl';
    if (p === 'Alt') return isMac ? '⌥' : 'Alt';
    if (p === 'Shift') return isMac ? '⇧' : 'Shift';

    // Спецклавиши
    const keyMap: Record<string, string> = {
      escape: 'Esc',
      enter: '↵',
      arrowup: '↑',
      arrowdown: '↓',
      arrowleft: '←',
      arrowright: '→',
      ' ': 'Space',
      '/': '/',
      '?': '?',
    };

    return keyMap[p.toLowerCase()] ?? p.toUpperCase();
  });

  return parts.join(isMac ? '' : '+');
}

// ═══════════════════════════════════════════════════════════════════════
// Component
// ═══════════════════════════════════════════════════════════════════════

export function KeyboardShortcutsHelp({ isOpen, onClose, shortcuts }: KeyboardShortcutsHelpProps) {
  const { t } = useTranslation();

  if (!isOpen) return null;

  return (
    <div
      className="fixed inset-0 z-50 flex items-center justify-center bg-black/50"
      onClick={onClose}
      role="dialog"
      aria-modal="true"
      aria-label={t('keyboard_shortcuts') || 'Keyboard Shortcuts'}
    >
      <div
        className="bg-white dark:bg-slate-800 rounded-2xl shadow-xl w-full max-w-md mx-4 max-h-[80vh] flex flex-col"
        onClick={(e) => e.stopPropagation()}
      >
        {/* Header */}
        <div className="flex items-center justify-between p-4 border-b border-slate-200 dark:border-slate-700 shrink-0">
          <div className="flex items-center gap-2">
            <Keyboard className="w-5 h-5 text-slate-500 dark:text-slate-400" />
            <h2 className="text-lg font-bold text-slate-900 dark:text-white">
              {t('keyboard_shortcuts') || 'Keyboard Shortcuts'}
            </h2>
          </div>
          <button
            onClick={onClose}
            className="p-1 hover:bg-slate-100 dark:hover:bg-slate-700 rounded-lg transition-colors"
            aria-label={t('close') || 'Close'}
          >
            <X className="w-5 h-5 text-slate-400" />
          </button>
        </div>

        {/* Shortcuts list */}
        <div className="p-4 space-y-2 overflow-y-auto">
          {shortcuts.length === 0 ? (
            <p className="text-sm text-slate-400 dark:text-slate-500 text-center py-8">
              {t('no_shortcuts') || 'No shortcuts configured'}
            </p>
          ) : (
            shortcuts.map((s, i) => (
              <div
                key={i}
                className="flex items-center justify-between py-2 px-1 rounded-lg hover:bg-slate-50 dark:hover:bg-slate-700/50 transition-colors"
              >
                <span className="text-sm text-slate-600 dark:text-slate-400">
                  {s.description}
                </span>
                <kbd className="px-2.5 py-1 text-xs font-mono font-medium text-slate-700 dark:text-slate-300 bg-slate-100 dark:bg-slate-700 rounded-md border border-slate-200 dark:border-slate-600 shadow-sm whitespace-nowrap">
                  {formatKeyLabel(s.key)}
                </kbd>
              </div>
            ))
          )}
        </div>
      </div>
    </div>
  );
}
