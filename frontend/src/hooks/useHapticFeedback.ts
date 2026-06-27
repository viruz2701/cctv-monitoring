// ═══════════════════════════════════════════════════════════════════════
// useHapticFeedback — Haptic feedback hook (P3-UI.2)
//
// Использует Vibration API для тактильной обратной связи на мобильных.
// На десктопе — no-op (graceful degradation).
//
// Типы обратной связи:
//   - light:  короткий импульс (клик, тап)
//   - medium: средний импульс (долгое нажатие, переключение)
//   - heavy:  сильный импульс (предупреждение, confirm)
//   - selection:  импульс выбора (переключение между элементами)
//   - success:  короткий паттерн "успех"
//   - error:    тройной паттерн "ошибка"
//
// Соответствие стандартам:
//   - WCAG 2.1 SC 2.3.3 — не блокирует при prefers-reduced-motion
//   - IEC 62443 SR 7.1 — graceful degradation при отсутствии API
// ═══════════════════════════════════════════════════════════════════════

import { useCallback } from 'react';
import { useReducedMotion } from './useReducedMotion';

type HapticType = 'light' | 'medium' | 'heavy' | 'selection' | 'success' | 'error';

const HAPTIC_PATTERNS: Record<HapticType, number[]> = {
  light: [10],
  medium: [20],
  heavy: [40],
  selection: [5, 5, 5],
  success: [10, 20, 10],
  error: [30, 10, 30, 10, 30],
};

/**
 * Хук для тактильной обратной связи.
 *
 * Пример:
 * ```tsx
 * const haptics = useHapticFeedback();
 * <button onClick={() => { haptics.light(); handleClick(); }}>
 *   Нажми меня
 * </button>
 * ```
 */
export function useHapticFeedback() {
  const prefersReduced = useReducedMotion();

  const vibrate = useCallback(
    (pattern: number[]) => {
      // Не вибрируем при prefers-reduced-motion
      if (prefersReduced) return;
      if (typeof navigator !== 'undefined' && 'vibrate' in navigator) {
        navigator.vibrate(pattern);
      }
    },
    [prefersReduced],
  );

  const light = useCallback(() => vibrate(HAPTIC_PATTERNS.light), [vibrate]);
  const medium = useCallback(() => vibrate(HAPTIC_PATTERNS.medium), [vibrate]);
  const heavy = useCallback(() => vibrate(HAPTIC_PATTERNS.heavy), [vibrate]);
  const selection = useCallback(() => vibrate(HAPTIC_PATTERNS.selection), [vibrate]);
  const success = useCallback(() => vibrate(HAPTIC_PATTERNS.success), [vibrate]);
  const error = useCallback(() => vibrate(HAPTIC_PATTERNS.error), [vibrate]);

  return { light, medium, heavy, selection, success, error };
}
