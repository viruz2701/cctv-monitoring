import React, { useId, useState, useCallback, useRef, useEffect } from 'react';

// ═══════════════════════════════════════════════════════════════════════
// Tooltip Component
// CSS-only tooltip with keyboard accessibility.
// WCAG AA: focusable trigger, dismissible via Escape.
// Supports dark mode and 4 positions.
// ═══════════════════════════════════════════════════════════════════════

type TooltipPosition = 'top' | 'bottom' | 'left' | 'right';

interface TooltipProps {
  /** Tooltip text content */
  content: string;
  /** Position relative to children. Default: 'top' */
  position?: TooltipPosition;
  /** Show delay in ms. Default: 300 */
  delay?: number;
  /** Hide delay in ms. Default: 150 */
  hideDelay?: number;
  /** Additional CSS classes for the wrapper */
  className?: string;
  children: React.ReactNode;
}

const positionStyles: Record<TooltipPosition, string> = {
  top: 'bottom-full left-1/2 -translate-x-1/2 mb-1.5',
  bottom: 'top-full left-1/2 -translate-x-1/2 mt-1.5',
  left: 'right-full top-1/2 -translate-y-1/2 mr-1.5',
  right: 'left-full top-1/2 -translate-y-1/2 ml-1.5',
};

const arrowStyles: Record<TooltipPosition, string> = {
  top: 'top-full left-1/2 -translate-x-1/2 border-l-4 border-r-4 border-t-4 border-l-transparent border-r-transparent border-t-slate-900 dark:border-t-slate-700',
  bottom: 'bottom-full left-1/2 -translate-x-1/2 border-l-4 border-r-4 border-b-4 border-l-transparent border-r-transparent border-b-slate-900 dark:border-b-slate-700',
  left: 'left-full top-1/2 -translate-y-1/2 border-t-4 border-b-4 border-l-4 border-t-transparent border-b-transparent border-l-slate-900 dark:border-l-slate-700',
  right: 'right-full top-1/2 -translate-y-1/2 border-t-4 border-b-4 border-r-4 border-t-transparent border-b-transparent border-r-slate-900 dark:border-r-slate-700',
};

export function Tooltip({
  content,
  position = 'top',
  delay = 300,
  hideDelay = 150,
  className = '',
  children,
}: TooltipProps) {
  const tooltipId = useId();
  const [visible, setVisible] = useState(false);
  const showTimerRef = useRef<ReturnType<typeof setTimeout> | null>(null);
  const hideTimerRef = useRef<ReturnType<typeof setTimeout> | null>(null);

  const clearTimers = useCallback(() => {
    if (showTimerRef.current) {
      clearTimeout(showTimerRef.current);
      showTimerRef.current = null;
    }
    if (hideTimerRef.current) {
      clearTimeout(hideTimerRef.current);
      hideTimerRef.current = null;
    }
  }, []);

  const handleMouseEnter = useCallback(() => {
    if (hideTimerRef.current) {
      clearTimeout(hideTimerRef.current);
      hideTimerRef.current = null;
    }
    showTimerRef.current = setTimeout(() => setVisible(true), delay);
  }, [delay]);

  const handleMouseLeave = useCallback(() => {
    if (showTimerRef.current) {
      clearTimeout(showTimerRef.current);
      showTimerRef.current = null;
    }
    hideTimerRef.current = setTimeout(() => setVisible(false), hideDelay);
  }, [hideDelay]);

  const handleFocus = useCallback(() => {
    clearTimers();
    setVisible(true);
  }, [clearTimers]);

  const handleBlur = useCallback(() => {
    setVisible(false);
  }, []);

  const handleKeyDown = useCallback((e: React.KeyboardEvent) => {
    if (e.key === 'Escape') {
      setVisible(false);
    }
  }, []);

  // Cleanup on unmount
  useEffect(() => {
    return () => clearTimers();
  }, [clearTimers]);

  return (
    <div
      className={`relative inline-flex ${className}`}
      onMouseEnter={handleMouseEnter}
      onMouseLeave={handleMouseLeave}
      onFocus={handleFocus}
      onBlur={handleBlur}
      onKeyDown={handleKeyDown}
    >
      {children}
      <div
        id={tooltipId}
        role="tooltip"
        aria-hidden={!visible}
        className={`
          absolute z-50 pointer-events-none
          px-2 py-1 text-xs font-medium text-white
          bg-slate-900 dark:bg-slate-700
          rounded-md shadow-lg
          whitespace-nowrap
          transition-opacity duration-150
          ${positionStyles[position]}
          ${visible ? 'opacity-100' : 'opacity-0'}
        `}
      >
        {content}
        <span className={`absolute w-0 h-0 ${arrowStyles[position]}`} />
      </div>
    </div>
  );
}
