// ═══════════════════════════════════════════════════════════════════════
// useAnnotationKeyboard.ts — Keyboard shortcuts for PhotoAnnotation
// P1-PHOTO: WCAG 2.1 AA compliance — Ctrl+Z, Ctrl+Shift+Z, Escape, Delete
// ═══════════════════════════════════════════════════════════════════════

import { useEffect } from 'react';

interface KeyboardHandlers {
  onUndo: () => void;
  onRedo: () => void;
  onClear: () => void;
  onEscape: () => void;
}

export function useAnnotationKeyboard({
  onUndo,
  onRedo,
  onClear,
  onEscape,
}: KeyboardHandlers): void {
  useEffect(() => {
    const handler = (e: KeyboardEvent) => {
      // Don't trigger when typing in an input
      if (
        e.target instanceof HTMLInputElement ||
        e.target instanceof HTMLTextAreaElement
      ) {
        if (e.key === 'Escape') {
          onEscape();
        }
        return;
      }

      const isCtrl = e.ctrlKey || e.metaKey;

      if (isCtrl && e.shiftKey && e.key === 'z') {
        e.preventDefault();
        onRedo();
        return;
      }

      if (isCtrl && e.key === 'z') {
        e.preventDefault();
        onUndo();
        return;
      }

      if (e.key === 'Delete' || e.key === 'Backspace') {
        onClear();
        return;
      }

      if (e.key === 'Escape') {
        onEscape();
        return;
      }
    };

    window.addEventListener('keydown', handler);
    return () => window.removeEventListener('keydown', handler);
  }, [onUndo, onRedo, onClear, onEscape]);
}
