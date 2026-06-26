// ═══════════════════════════════════════════════════════════════════════
// InfoTooltip — Info icon with tooltip for complex/compliance terms
//
// P1-1.7: Contextual Tooltips
//   - Info icon (ℹ) that shows tooltip on hover/focus
//   - Links to Glossary page for detailed explanation
//   - WCAG AA accessible with keyboard support
// ═══════════════════════════════════════════════════════════════════════

import React, { useId, useState, useCallback, useRef, useEffect } from 'react';
import { useTranslation } from 'react-i18next';
import { Info } from 'lucide-react';
import { Link } from 'react-router-dom';

type TooltipPosition = 'top' | 'bottom' | 'left' | 'right';

interface InfoTooltipProps {
    /** Short explanation text */
    text: string;
    /** Optional link to glossary term */
    glossaryTerm?: string;
    /** Tooltip position relative to icon. Default: 'top' */
    position?: TooltipPosition;
    /** Show delay in ms. Default: 200 */
    delay?: number;
    /** Icon size. Default: 14 */
    iconSize?: number;
    /** Additional CSS classes */
    className?: string;
}

const positionStyles: Record<TooltipPosition, string> = {
    top: 'bottom-full left-1/2 -translate-x-1/2 mb-2',
    bottom: 'top-full left-1/2 -translate-x-1/2 mt-2',
    left: 'right-full top-1/2 -translate-y-1/2 mr-2',
    right: 'left-full top-1/2 -translate-y-1/2 ml-2',
};

export function InfoTooltip({
    text,
    glossaryTerm,
    position = 'top',
    delay = 200,
    iconSize = 14,
    className = '',
}: InfoTooltipProps) {
    const { t } = useTranslation();
    const tooltipId = useId();
    const [visible, setVisible] = useState(false);
    const timerRef = useRef<ReturnType<typeof setTimeout> | null>(null);

    const show = useCallback(() => {
        if (timerRef.current) clearTimeout(timerRef.current);
        timerRef.current = setTimeout(() => setVisible(true), delay);
    }, [delay]);

    const hide = useCallback(() => {
        if (timerRef.current) clearTimeout(timerRef.current);
        setVisible(false);
    }, []);

    useEffect(() => {
        return () => {
            if (timerRef.current) clearTimeout(timerRef.current);
        };
    }, []);

    return (
        <span
            className={`inline-flex items-center relative ${className}`}
            onMouseEnter={show}
            onMouseLeave={hide}
            onFocus={show}
            onBlur={hide}
        >
            <button
                type="button"
                className="inline-flex items-center justify-center text-slate-400 hover:text-blue-500 dark:text-slate-500 dark:hover:text-blue-400 transition-colors cursor-help focus:outline-none focus:ring-2 focus:ring-blue-500 rounded-full"
                aria-label={t('more_info') || 'More info'}
                aria-describedby={tooltipId}
                tabIndex={0}
                onKeyDown={(e) => {
                    if (e.key === 'Escape') hide();
                }}
            >
                <Info size={iconSize} />
            </button>

            {/* Tooltip */}
            <div
                id={tooltipId}
                role="tooltip"
                aria-hidden={!visible}
                className={`
                    absolute z-50 pointer-events-none
                    px-3 py-2 text-xs leading-relaxed
                    text-white bg-slate-900 dark:bg-slate-700
                    rounded-lg shadow-lg
                    max-w-[260px]
                    transition-opacity duration-150
                    ${positionStyles[position]}
                    ${visible ? 'opacity-100' : 'opacity-0'}
                `}
            >
                <p className="text-white/90">{text}</p>
                {glossaryTerm && (
                    <Link
                        to={`/glossary#${glossaryTerm.toLowerCase().replace(/\s+/g, '-')}`}
                        className="block mt-1.5 text-[10px] font-medium text-blue-300 hover:text-blue-200 underline pointer-events-auto"
                        onClick={(e) => {
                            e.stopPropagation();
                            hide();
                        }}
                    >
                        {t('learn_more_in_glossary') || 'Learn more in Glossary →'}
                    </Link>
                )}
                {/* Arrow */}
                <span
                    className={`absolute w-2 h-2 bg-slate-900 dark:bg-slate-700 rotate-45 ${
                        position === 'top'
                            ? 'top-full left-1/2 -translate-x-1/2 -mt-1'
                            : position === 'bottom'
                                ? 'bottom-full left-1/2 -translate-x-1/2 -mb-1'
                                : position === 'left'
                                    ? 'left-full top-1/2 -translate-y-1/2 -ml-1'
                                    : 'right-full top-1/2 -translate-y-1/2 -mr-1'
                    }`}
                />
            </div>
        </span>
    );
}