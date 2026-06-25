// ═══════════════════════════════════════════════════════════════════════
// useReducedMotion — detects user preference for reduced motion
// WCAG 2.1 SC 2.3.3 (Animation from Interactions)
//
// Usage:
//   const prefersReduced = useReducedMotion();
//   if (prefersReduced) { /* skip animation */ }
// ═══════════════════════════════════════════════════════════════════════

import { useState, useEffect } from 'react';

/**
 * Returns `true` if the user prefers reduced motion via
 * `prefers-reduced-motion: reduce` OS/browser setting.
 *
 * Automatically re-renders on preference change.
 */
export function useReducedMotion(): boolean {
  const [prefersReduced, setPrefersReduced] = useState<boolean>(() => {
    if (typeof window === 'undefined') return false;
    return window.matchMedia('(prefers-reduced-motion: reduce)').matches;
  });

  useEffect(() => {
    const mq = window.matchMedia('(prefers-reduced-motion: reduce)');

    const handler = (event: MediaQueryListEvent) => {
      setPrefersReduced(event.matches);
    };

    // Modern browsers
    mq.addEventListener('change', handler);
    return () => mq.removeEventListener('change', handler);
  }, []);

  return prefersReduced;
}
