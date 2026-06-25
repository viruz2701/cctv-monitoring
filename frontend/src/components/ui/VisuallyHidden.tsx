// ═══════════════════════════════════════════════════════════════════════
// VisuallyHidden — Screen-reader only text (WCAG 2.1 AA, UX-14.2.7)
//
// Use when you need text that's accessible to screen readers but
// visually hidden. Wraps Tailwind's `sr-only` utility.
// ═══════════════════════════════════════════════════════════════════════

import React from 'react';

interface VisuallyHiddenProps {
    children: React.ReactNode;
    as?: 'span' | 'div' | 'p';
    className?: string;
}

export function VisuallyHidden({
    children,
    as: Tag = 'span',
    className = '',
}: VisuallyHiddenProps) {
    return (
        <Tag className={`sr-only ${className}`}>
            {children}
        </Tag>
    );
}
