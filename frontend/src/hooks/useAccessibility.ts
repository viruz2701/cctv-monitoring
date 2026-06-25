// ═══════════════════════════════════════════════════════════════════════
// useAccessibility — WCAG 2.1 AA compliance hooks (UX-14.2.7)
// ═══════════════════════════════════════════════════════════════════════

import { useEffect, useRef, useCallback } from 'react';

// ───────────────────────────────────────────────────────────────────────
// useSkipLink — добавляет skip-to-content ссылку для keyboard users
// ───────────────────────────────────────────────────────────────────────
export function useSkipLink(targetId = 'main-content') {
    const skipLinkRef = useRef<HTMLAnchorElement>(null);

    useEffect(() => {
        const handleFirstTab = (e: KeyboardEvent) => {
            if (e.key === 'Tab') {
                document.body.classList.add('user-is-tabbing');
                window.removeEventListener('keydown', handleFirstTab);
            }
        };
        window.addEventListener('keydown', handleFirstTab);
        return () => window.removeEventListener('keydown', handleFirstTab);
    }, []);

    const handleSkip = useCallback(
        (e: React.MouseEvent | React.KeyboardEvent) => {
            e.preventDefault();
            const target = document.getElementById(targetId);
            if (target) {
                target.setAttribute('tabindex', '-1');
                target.focus();
                // Remove tabindex after focus so it doesn't break tab flow
                target.addEventListener(
                    'blur',
                    () => target.removeAttribute('tabindex'),
                    { once: true }
                );
            }
        },
        [targetId]
    );

    return { skipLinkRef, handleSkip, targetId };
}

// ───────────────────────────────────────────────────────────────────────
// announce — программное объявление для screen readers (aria-live)
// ───────────────────────────────────────────────────────────────────────
let liveRegion: HTMLDivElement | null = null;

function getLiveRegion(): HTMLDivElement {
    if (!liveRegion) {
        liveRegion = document.createElement('div');
        liveRegion.setAttribute('role', 'status');
        liveRegion.setAttribute('aria-live', 'polite');
        liveRegion.setAttribute('aria-atomic', 'true');
        liveRegion.className = 'sr-only';
        document.body.appendChild(liveRegion);
    }
    return liveRegion;
}

/**
 * announce — отправляет сообщение screen reader'у через aria-live region.
 * Использование: announce('Загрузка завершена');
 */
export function announce(message: string, priority: 'polite' | 'assertive' = 'polite') {
    const region = getLiveRegion();
    region.setAttribute('aria-live', priority);

    // Clear and reset to ensure announcement repeats for same text
    region.textContent = '';
    requestAnimationFrame(() => {
        region.textContent = message;
    });
}

// ───────────────────────────────────────────────────────────────────────
// useFocusTrap — фокус-труппинг для модалок и панелей
// ───────────────────────────────────────────────────────────────────────
export function useFocusTrap(isActive: boolean) {
    const containerRef = useRef<HTMLDivElement>(null);
    const previousFocusRef = useRef<HTMLElement | null>(null);

    // Save and restore focus
    useEffect(() => {
        if (isActive) {
            previousFocusRef.current = document.activeElement as HTMLElement;
        } else if (previousFocusRef.current) {
            // Restore focus after a microtask to let DOM settle
            requestAnimationFrame(() => {
                previousFocusRef.current?.focus();
                previousFocusRef.current = null;
            });
        }
    }, [isActive]);

    const handleKeyDown = useCallback(
        (e: React.KeyboardEvent) => {
            if (!isActive || e.key !== 'Tab') return;

            const container = containerRef.current;
            if (!container) return;

            const focusableElements = container.querySelectorAll<HTMLElement>(
                'a[href], button:not([disabled]), textarea:not([disabled]), input:not([disabled]), select:not([disabled]), [tabindex]:not([tabindex="-1"])'
            );

            if (focusableElements.length === 0) {
                e.preventDefault();
                return;
            }

            const firstElement = focusableElements[0];
            const lastElement = focusableElements[focusableElements.length - 1];

            if (e.shiftKey) {
                if (document.activeElement === firstElement) {
                    e.preventDefault();
                    lastElement.focus();
                }
            } else {
                if (document.activeElement === lastElement) {
                    e.preventDefault();
                    firstElement.focus();
                }
            }
        },
        [isActive]
    );

    return { containerRef, handleKeyDown };
}

// ───────────────────────────────────────────────────────────────────────
// useAnnouncer — хук для отправки объявлений из компонента
// ───────────────────────────────────────────────────────────────────────
export function useAnnouncer() {
    const announcePolite = useCallback((message: string) => {
        announce(message, 'polite');
    }, []);

    const announceAssertive = useCallback((message: string) => {
        announce(message, 'assertive');
    }, []);

    return { announcePolite, announceAssertive };
}
