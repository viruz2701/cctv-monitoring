// ═══════════════════════════════════════════════════════════════════════
// useRipple — Ripple эффект для кнопок (P3-UI.2)
//
// Создаёт кастомный ripple-эффект при клике на элемент.
// Использует CSS анимацию через ripple-span класс.
//
// Возвращает:
//   - createRipple: (event: React.MouseEvent) => void — обработчик
//   - ripples: React.ReactNode — ripple-элементы для рендера
//
// Пример:
// ```tsx
// const { createRipple, ripples } = useRipple();
// <button className="ripple-container" onClick={createRipple}>
//   {children}
//   {ripples}
// </button>
// ```
//
// Соответствие:
//   - OWASP ASVS V7 (graceful degradation)
//   - WCAG 2.1 SC 2.3.3 (prefers-reduced-motion)
// ═══════════════════════════════════════════════════════════════════════

import { useState, useCallback, useRef } from 'react';
import { useReducedMotion } from './useReducedMotion';

interface Ripple {
  id: number;
  x: number;
  y: number;
  size: number;
}

let rippleIdCounter = 0;

export function useRipple() {
  const [ripples, setRipples] = useState<Ripple[]>([]);
  const timerRef = useRef<ReturnType<typeof setTimeout> | null>(null);
  const prefersReduced = useReducedMotion();

  const createRipple = useCallback(
    (event: React.MouseEvent<HTMLElement>) => {
      if (prefersReduced) return;

      const container = event.currentTarget;
      const rect = container.getBoundingClientRect();
      const size = Math.max(rect.width, rect.height);
      const x = event.clientX - rect.left - size / 2;
      const y = event.clientY - rect.top - size / 2;

      const id = ++rippleIdCounter;
      setRipples((prev) => [...prev, { id, x, y, size }]);

      if (timerRef.current) clearTimeout(timerRef.current);
      timerRef.current = setTimeout(() => {
        setRipples((prev) => prev.filter((r) => r.id !== id));
      }, 600);
    },
    [prefersReduced],
  );

  const ripplesNode = ripples.map((ripple) => (
    <span
      key={ripple.id}
      className="ripple-span"
      style={{
        left: ripple.x,
        top: ripple.y,
        width: ripple.size,
        height: ripple.size,
      }}
      aria-hidden="true"
    />
  ));

  return { createRipple, ripples: ripples.length > 0 ? ripplesNode : null };
}
