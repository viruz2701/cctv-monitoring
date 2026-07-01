// ═══════════════════════════════════════════════════════════════════════
// useAccessibility — WCAG 2.1 AA compliance hooks (UX-14.2.7, UX-14.2.8)
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
// useRestoreFocus — сохраняет и восстанавливает фокус при открытии/закрытии
// UX-14.2.8: Focus Management — restore focus on close
// ───────────────────────────────────────────────────────────────────────
export function useRestoreFocus(isActive: boolean) {
    const previousFocusRef = useRef<HTMLElement | null>(null);

    useEffect(() => {
        if (isActive) {
            previousFocusRef.current = document.activeElement as HTMLElement;
        } else if (previousFocusRef.current) {
            const element = previousFocusRef.current;
            previousFocusRef.current = null;
            // Restore focus after a microtask to let DOM settle
            requestAnimationFrame(() => {
                element.focus({ preventScroll: true });
            });
        }

        return () => {
            // Cleanup: restore focus if component unmounts while active
            if (!isActive && previousFocusRef.current) {
                const element = previousFocusRef.current;
                previousFocusRef.current = null;
                requestAnimationFrame(() => {
                    element.focus({ preventScroll: true });
                });
            }
        };
    }, [isActive]);

    return previousFocusRef;
}

// ───────────────────────────────────────────────────────────────────────
// FocusTrapStack — глобальный стек для вложенных focus trap'ов (P2-MED-14)
//
// Позволяет открывать модалку поверх модалки; только верхний trap
// перехватывает Tab/Shift+Tab. При закрытии верхнего — активируется
// предыдущий.
//
// UX-14.2.8: Focus Management — nested modals support
// WCAG 2.4.3: Focus Order — predictable focus cycling
// ───────────────────────────────────────────────────────────────────────

let trapCounter = 0;

interface TrapEntry {
    id: number;
    containerRef: React.RefObject<HTMLDivElement | null>;
}

const focusTrapStack: TrapEntry[] = [];

function getTopTrap(): TrapEntry | null {
    return focusTrapStack.length > 0 ? focusTrapStack[focusTrapStack.length - 1] : null;
}

function isTopTrap(id: number): boolean {
    return getTopTrap()?.id === id;
}

/**
 * Получить фокусируемые элементы внутри контейнера.
 */
function getFocusableElements(container: HTMLElement): HTMLElement[] {
    return Array.from(
        container.querySelectorAll<HTMLElement>(
            'a[href], button:not([disabled]), textarea:not([disabled]), input:not([disabled]), select:not([disabled]), [tabindex]:not([tabindex="-1"])'
        )
    );
}

// ───────────────────────────────────────────────────────────────────────
// useFocusTrap — фокус-труппинг для модалок и панелей
// Tab/Shift+Tab цикл внутри контейнера
// UX-14.2.8: Focus Management — focus trap with restore + nested support
// ───────────────────────────────────────────────────────────────────────
export function useFocusTrap(isActive: boolean, options?: { restoreFocus?: boolean }) {
    const containerRef = useRef<HTMLDivElement>(null);
    const restoreFocus = options?.restoreFocus ?? true;
    const trapIdRef = useRef<number>(0);
    useRestoreFocus(restoreFocus ? isActive : false);

    // Регистрация/дерегистрация в глобальном стеке
    useEffect(() => {
        if (!isActive) return;

        const id = ++trapCounter;
        trapIdRef.current = id;
        focusTrapStack.push({ id, containerRef });

        return () => {
            const idx = focusTrapStack.findIndex((t) => t.id === id);
            if (idx !== -1) {
                focusTrapStack.splice(idx, 1);
            }
        };
    }, [isActive]);

    const focusFirstElement = useCallback(() => {
        const container = containerRef.current;
        if (!container) return;

        const focusable = getFocusableElements(container);

        if (focusable.length > 0) {
            focusable[0].focus({ preventScroll: true });
        } else {
            container.setAttribute('tabindex', '-1');
            container.focus({ preventScroll: true });
        }
    }, []);

    // Auto-focus first element when trap activates
    useEffect(() => {
        if (isActive) {
            // Small delay to ensure DOM is ready
            requestAnimationFrame(focusFirstElement);
        }
    }, [isActive, focusFirstElement]);

    const handleKeyDown = useCallback(
        (e: React.KeyboardEvent) => {
            if (!isActive || e.key !== 'Tab') return;

            // P2-MED-14: Игнорируем событие, если этот trap не на вершине стека
            if (!isTopTrap(trapIdRef.current)) return;

            const container = containerRef.current;
            if (!container) return;

            const focusableElements = getFocusableElements(container);

            if (focusableElements.length === 0) {
                e.preventDefault();
                return;
            }

            const firstElement = focusableElements[0];
            const lastElement = focusableElements[focusableElements.length - 1];

            if (e.shiftKey) {
                if (document.activeElement === firstElement) {
                    e.preventDefault();
                    lastElement.focus({ preventScroll: true });
                }
            } else {
                if (document.activeElement === lastElement) {
                    e.preventDefault();
                    firstElement.focus({ preventScroll: true });
                }
            }
        },
        [isActive]
    );

    return { containerRef, handleKeyDown, focusFirstElement };
}

// ───────────────────────────────────────────────────────────────────────
// useTabIndex — управление tabIndex для элементов вне модалки
// UX-14.2.8: Focus Management — disable tabIndex when modal is open
// ───────────────────────────────────────────────────────────────────────
export function useTabIndex(
    isActive: boolean,
    selector = 'a[href], button, input, textarea, select, [tabindex]:not([tabindex="-1"])'
) {
    const previousValuesRef = useRef<Map<Element, string | null>>(new Map());

    useEffect(() => {
        if (isActive) {
            const elements = document.querySelectorAll<HTMLElement>(selector);
            previousValuesRef.current = new Map();

            elements.forEach((el) => {
                // Save current tabindex
                previousValuesRef.current.set(el, el.getAttribute('tabindex'));
                el.setAttribute('tabindex', '-1');
            });

            return () => {
                // Restore tabindex values
                previousValuesRef.current.forEach((savedValue, el) => {
                    if (savedValue === null) {
                        el.removeAttribute('tabindex');
                    } else {
                        el.setAttribute('tabindex', savedValue);
                    }
                });
                previousValuesRef.current.clear();
            };
        }
    }, [isActive, selector]);
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
