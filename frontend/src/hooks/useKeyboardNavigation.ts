// ═══════════════════════════════════════════════════════════════════════
// useKeyboardNavigation — Keyboard Navigation Audit Hook (UX-8.2)
//
// UX-8.2: Keyboard Navigation Audit
//   - Focusable elements check
//   - Tab order logical
//   - Escape closes modals
//   - WCAG 2.1 AA compliance (2.1.1, 2.1.2, 2.4.3)
// ═══════════════════════════════════════════════════════════════════════

import { useEffect, useCallback, useRef } from 'react';

// ── Types ─────────────────────────────────────────────────────────────

export interface FocusableElement {
  element: HTMLElement;
  tagName: string;
  tabIndex: number;
  accessibleName: string;
  isVisible: boolean;
  isDisabled: boolean;
  rect: DOMRect | null;
}

export interface KeyboardAuditResult {
  /** Все фокусируемые элементы на странице */
  focusableElements: FocusableElement[];
  /** Элементы с некорректным tabIndex (>0) */
  nonStandardTabIndex: FocusableElement[];
  /** Элементы, скрытые от keyboard (tabindex=-1) */
  hiddenFromKeyboard: FocusableElement[];
  /** Элементы без доступного имени */
  missingAccessibleName: FocusableElement[];
  /** Нарушения tab order (визуальный порядок != DOM порядок) */
  tabOrderIssues: Array<{ element: FocusableElement; expectedPosition: number; actualPosition: number }>;
  /** Общее количество проблем */
  totalIssues: number;
}

// ── Helpers ───────────────────────────────────────────────────────────

const FOCUSABLE_SELECTOR = [
  'a[href]',
  'button:not([disabled])',
  'textarea:not([disabled])',
  'input:not([type="hidden"]):not([disabled])',
  'select:not([disabled])',
  '[tabindex]:not([tabindex="-1"]):not([disabled])',
  '[contenteditable="true"]',
].join(', ');

function getAccessibleName(element: HTMLElement): string {
  const ariaLabel = element.getAttribute('aria-label');
  if (ariaLabel) return ariaLabel;

  const labelledBy = element.getAttribute('aria-labelledby');
  if (labelledBy) {
    const labelElement = document.getElementById(labelledBy);
    if (labelElement) return labelElement.textContent?.trim() || '';
  }

  const title = element.getAttribute('title');
  if (title) return title;

  const text = element.textContent?.trim();
  if (text && text.length < 100) return text;

  // For inputs: check associated label
  if (element instanceof HTMLInputElement || element instanceof HTMLTextAreaElement || element instanceof HTMLSelectElement) {
    const id = element.id;
    if (id) {
      const label = document.querySelector(`label[for="${id}"]`);
      if (label) return label.textContent?.trim() || '';
    }
    // Check parent label
    const parentLabel = element.closest('label');
    if (parentLabel) return parentLabel.textContent?.trim() || '';
  }

  // Check aria-describedby
  const describedBy = element.getAttribute('aria-describedby');
  if (describedBy) {
    const desc = document.getElementById(describedBy);
    if (desc) return desc.textContent?.trim() || '';
  }

  return '';
}

function isElementVisible(element: HTMLElement): boolean {
  const style = window.getComputedStyle(element);
  if (style.display === 'none' || style.visibility === 'hidden') return false;
  if (element.offsetParent === null && element.tagName !== 'BODY') return false;
  const rect = element.getBoundingClientRect();
  return rect.width > 0 && rect.height > 0;
}

function getVisualPosition(element: HTMLElement): { top: number; left: number } {
  const rect = element.getBoundingClientRect();
  return { top: rect.top, left: rect.left };
}

// ── useKeyboardAudit Hook ────────────────────────────────────────────

/**
 * useKeyboardAudit — проводит аудит клавиатурной навигации на странице.
 *
 * UX-8.2: Keyboard Navigation Audit
 *   - Возвращает все фокусируемые элементы с их свойствами
 *   - Выявляет проблемы с tabIndex, доступными именами, порядком
 *
 * @example
 * ```tsx
 * function MyComponent() {
 *   const { auditResult, runAudit } = useKeyboardAudit();
 *
 *   useEffect(() => {
 *     const result = runAudit();
 *     console.log(`Found ${result.totalIssues} keyboard navigation issues`);
 *   }, [runAudit]);
 * }
 * ```
 */
export function useKeyboardAudit() {
  const resultRef = useRef<KeyboardAuditResult | null>(null);

  const runAudit = useCallback((): KeyboardAuditResult => {
    const elements = document.querySelectorAll<HTMLElement>(FOCUSABLE_SELECTOR);
    const focusableElements: FocusableElement[] = [];
    const nonStandardTabIndex: FocusableElement[] = [];
    const hiddenFromKeyboard: FocusableElement[] = [];
    const missingAccessibleName: FocusableElement[] = [];

    elements.forEach((el) => {
      const tabIndex = parseInt(el.getAttribute('tabindex') || '0', 10);
      const isVisible = isElementVisible(el);
      const accessibleName = getAccessibleName(el);
      const rect = el.getBoundingClientRect();

      const entry: FocusableElement = {
        element: el,
        tagName: el.tagName.toLowerCase(),
        tabIndex,
        accessibleName,
        isVisible,
        isDisabled: el.hasAttribute('disabled'),
        rect: isVisible ? rect : null,
      };

      focusableElements.push(entry);

      // Non-standard tabIndex (not -1, 0)
      if (tabIndex > 0) {
        nonStandardTabIndex.push(entry);
      }

      // Hidden from keyboard (tabindex=-1) and not an interactive container
      if (tabIndex === -1 && !el.closest('[role="dialog"]')) {
        hiddenFromKeyboard.push(entry);
      }

      // Missing accessible name for interactive elements
      if (!accessibleName && tabIndex >= 0) {
        const interactiveTags = ['a', 'button', 'input', 'textarea', 'select'];
        if (interactiveTags.includes(el.tagName.toLowerCase()) || el.getAttribute('role')) {
          missingAccessibleName.push(entry);
        }
      }
    });

    // Check tab order issues
    const tabOrderIssues: KeyboardAuditResult['tabOrderIssues'] = [];
    const sortedByDOM = [...focusableElements].filter((e) => e.tabIndex >= 0 && e.isVisible);
    const sortedByPosition = [...sortedByDOM].sort((a, b) => {
      if (!a.rect || !b.rect) return 0;
      if (Math.abs(a.rect.top - b.rect.top) < 10) {
        return a.rect.left - b.rect.left;
      }
      return a.rect.top - b.rect.top;
    });

    sortedByDOM.forEach((el, actualIndex) => {
      const visualIndex = sortedByPosition.indexOf(el);
      if (visualIndex !== -1 && Math.abs(visualIndex - actualIndex) > 2) {
        tabOrderIssues.push({
          element: el,
          expectedPosition: visualIndex + 1,
          actualPosition: actualIndex + 1,
        });
      }
    });

    const result: KeyboardAuditResult = {
      focusableElements,
      nonStandardTabIndex,
      hiddenFromKeyboard,
      missingAccessibleName,
      tabOrderIssues,
      totalIssues:
        nonStandardTabIndex.length +
        missingAccessibleName.length +
        tabOrderIssues.length,
    };

    resultRef.current = result;
    return result;
  }, []);

  /**
   * Тиражировать результаты аудита в console.table (для development)
   */
  const printAuditReport = useCallback(() => {
    const result = runAudit();

    console.group('%c Keyboard Navigation Audit (UX-8.2)', 'font-weight: bold; font-size: 14px;');
    console.log(`Total focusable elements: ${result.focusableElements.length}`);
    console.log(`Issues found: ${result.totalIssues}`);

    if (result.nonStandardTabIndex.length > 0) {
      console.warn('Non-standard tabIndex (>0):');
      console.table(result.nonStandardTabIndex.map((e) => ({
        tag: e.tagName,
        tabIndex: e.tabIndex,
        name: e.accessibleName,
      })));
    }

    if (result.missingAccessibleName.length > 0) {
      console.warn('Missing accessible names:');
      console.table(result.missingAccessibleName.map((e) => ({
        tag: e.tagName,
        text: e.element.textContent?.trim().slice(0, 50),
      })));
    }

    if (result.tabOrderIssues.length > 0) {
      console.warn('Tab order issues:');
      console.table(result.tabOrderIssues.map((e) => ({
        tag: e.element.tagName.toLowerCase(),
        name: e.element.accessibleName,
        expected: e.expectedPosition,
        actual: e.actualPosition,
      })));
    }

    console.groupEnd();

    return result;
  }, [runAudit]);

  return {
    auditResult: resultRef.current,
    runAudit,
    printAuditReport,
  };
}

// ── useEscapeListener Hook ────────────────────────────────────────────

/**
 * useEscapeListener — глобальный listener для Escape (UX-8.2: WCAG 2.1.2)
 *
 * @example
 * ```tsx
 * useEscapeListener(() => {
 *   if (isModalOpen) closeModal();
 * });
 * ```
 */
export function useEscapeListener(onEscape: () => void, isActive = true) {
  useEffect(() => {
    if (!isActive) return;

    const handleKeyDown = (e: KeyboardEvent) => {
      if (e.key === 'Escape') {
        e.stopPropagation();
        onEscape();
      }
    };

    document.addEventListener('keydown', handleKeyDown, true);
    return () => document.removeEventListener('keydown', handleKeyDown, true);
  }, [onEscape, isActive]);
}

// ── useFocusableElements Hook ─────────────────────────────────────────

/**
 * useFocusableElements — проверяет количество фокусируемых элементов в контейнере.
 * Полезно для определения, есть ли у модалки фокусируемые элементы для trap.
 *
 * @example
 * ```tsx
 * const { focusableCount, hasFocusableElements } = useFocusableElements(containerRef);
 * ```
 */
export function useFocusableElements(containerRef: React.RefObject<HTMLElement | null>) {
  const getFocusableCount = useCallback((): number => {
    if (!containerRef.current) return 0;
    return containerRef.current.querySelectorAll<HTMLElement>(FOCUSABLE_SELECTOR).length;
  }, [containerRef]);

  return {
    focusableCount: containerRef.current
      ? containerRef.current.querySelectorAll<HTMLElement>(FOCUSABLE_SELECTOR).length
      : 0,
    hasFocusableElements: containerRef.current
      ? containerRef.current.querySelectorAll<HTMLElement>(FOCUSABLE_SELECTOR).length > 0
      : false,
    getFocusableCount,
  };
}

// ── useTabOrderCheck Hook ─────────────────────────────────────────────

/**
 * useTabOrderCheck — проверяет логический порядок табов в контейнере.
 * Возвращает true если есть проблемы с tab order.
 */
export function useTabOrderCheck(containerRef: React.RefObject<HTMLElement | null>) {
  const checkTabOrder = useCallback((): boolean => {
    if (!containerRef.current) return false;

    const elements = containerRef.current.querySelectorAll<HTMLElement>(FOCUSABLE_SELECTOR);
    let hasIssue = false;

    elements.forEach((el, i) => {
      const tabIndex = parseInt(el.getAttribute('tabindex') || '0', 10);
      if (tabIndex > 0) {
        console.warn(
          `[useTabOrderCheck] Element #${i} has non-standard tabIndex=${tabIndex}:`,
          el
        );
        hasIssue = true;
      }
      if (!getAccessibleName(el) && ['button', 'a', 'input'].includes(el.tagName.toLowerCase())) {
        console.warn(
          `[useTabOrderCheck] Element #${i} missing accessible name:`,
          el
        );
        hasIssue = true;
      }
    });

    return hasIssue;
  }, [containerRef]);

  return { checkTabOrder };
}
