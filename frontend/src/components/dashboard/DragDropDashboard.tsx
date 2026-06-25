// ═══════════════════════════════════════════════════════════════════════
// DragDropDashboard — Responsive draggable grid wrapper (UX-14.2.2)
//
// Оборачивает react-grid-layout v2 (ResponsiveGridLayout) с:
//   - localStorage persistence (layout + visibility)
//   - Responsive breakpoints (1/2/3/4 колонки)
//   - Drag-handle зона захвата
//   - Анимация при перетаскивании
//   - Кнопка "Reset Layout" в режиме кастомизации
// ═══════════════════════════════════════════════════════════════════════

import React from 'react';
import { ResponsiveGridLayout, useContainerWidth } from 'react-grid-layout';
import type { LayoutItem, ResponsiveLayouts } from 'react-grid-layout';
import { GripVertical, RotateCcw } from 'lucide-react';
import 'react-grid-layout/css/styles.css';
import 'react-resizable/css/styles.css';

// ── Types ────────────────────────────────────────────────────────────

export interface DashboardWidget {
    id: string;
    content: React.ReactNode;
    minW?: number;
    minH?: number;
}

export interface DragDropDashboardProps {
    widgets: DashboardWidget[];
    customizeMode?: boolean;
    visibleWidgets?: string[];
    onToggleWidget?: (id: string, visible: boolean) => void;
    onResetLayout?: () => void;
    storageKey?: string;
    className?: string;
}

/** Mutable layout item for localStorage serialization */
type MutableLayout = LayoutItem[];

// ── Constants ────────────────────────────────────────────────────────

const STORAGE_KEY_DEFAULT = 'dashboard:layout';
const STORAGE_KEY_VISIBILITY = 'dashboard:widgetVisibility';

const BREAKPOINTS = { lg: 1280, md: 1024, sm: 768, xs: 480, xxs: 0 };
const COLS: Record<string, number> = { lg: 4, md: 3, sm: 2, xs: 1, xxs: 1 };
const ROW_HEIGHT = 150;

// ── Default Layouts ──────────────────────────────────────────────────

function createDefaultLayouts(): Record<string, MutableLayout> {
    const base: MutableLayout = [
        { i: 'statsOverview', x: 0, y: 0, w: 4, h: 1, minH: 1, minW: 2 },
        { i: 'ticketAnalytics', x: 0, y: 1, w: 4, h: 1, minH: 1, minW: 2 },
        { i: 'deviceHealthChart', x: 0, y: 2, w: 2, h: 2, minH: 1, minW: 1 },
        { i: 'alertTrendChart', x: 2, y: 2, w: 2, h: 2, minH: 1, minW: 1 },
        { i: 'ticketTrendChart', x: 0, y: 4, w: 2, h: 2, minH: 1, minW: 1 },
        { i: 'recentAlerts', x: 2, y: 4, w: 2, h: 2, minH: 1, minW: 1 },
        { i: 'latestTickets', x: 0, y: 6, w: 2, h: 2, minH: 1, minW: 1 },
        { i: 'quickActions', x: 2, y: 6, w: 2, h: 1, minH: 1, minW: 1 },
    ];

    const layouts: Record<string, MutableLayout> = { lg: base };

    // Generate scaled layouts for smaller breakpoints
    const lgCols = COLS.lg ?? 4;
    for (const [bp, cols] of Object.entries(COLS)) {
        if (bp === 'lg') continue;
        const colRatio = cols / lgCols;
        layouts[bp] = base.map((item) => ({
            ...item,
            w: Math.max(1, Math.round((item.w ?? 1) * colRatio)),
            x: Math.min(Math.round((item.x ?? 0) * colRatio), cols - 1),
        }));
    }

    return layouts;
}

// ── Storage helpers ──────────────────────────────────────────────────

function loadLayouts(key: string): Record<string, MutableLayout> | null {
    try {
        const raw = localStorage.getItem(key);
        if (!raw) return null;
        const parsed = JSON.parse(raw) as Record<string, MutableLayout>;
        if (parsed.lg && Array.isArray(parsed.lg) && parsed.lg.length > 0) {
            return parsed;
        }
    } catch {
        // Corrupted data — fall through to default
    }
    return null;
}

function saveLayouts(key: string, layouts: Record<string, MutableLayout>): void {
    try {
        localStorage.setItem(key, JSON.stringify(layouts));
    } catch {
        // localStorage full or unavailable — silently fail
    }
}

function loadVisibility(key: string): string[] | null {
    try {
        const raw = localStorage.getItem(key);
        if (!raw) return null;
        const parsed = JSON.parse(raw);
        if (Array.isArray(parsed) && parsed.length > 0) return parsed;
    } catch {
        // ignore
    }
    return null;
}

function saveVisibility(key: string, ids: string[]): void {
    try {
        localStorage.setItem(key, JSON.stringify(ids));
    } catch {
        // ignore
    }
}

export const ALL_WIDGET_IDS = [
    'statsOverview',
    'ticketAnalytics',
    'deviceHealthChart',
    'alertTrendChart',
    'ticketTrendChart',
    'recentAlerts',
    'latestTickets',
    'quickActions',
] as const;

// ── Sub-component: Visibility toggle badge ───────────────────────────

function WidgetVisibilityToggle({
    id,
    visible,
    onToggle,
}: {
    id: string;
    visible: boolean;
    onToggle: (id: string) => void;
}) {
    return (
        <button
            onClick={(e) => {
                e.stopPropagation();
                onToggle(id);
            }}
            className={[
                'absolute top-2 right-2 z-20',
                'p-1.5 rounded-full',
                'transition-all duration-200 shadow-sm',
                visible
                    ? 'bg-blue-500 text-white hover:bg-blue-600'
                    : 'bg-slate-200 dark:bg-slate-600 text-slate-500 dark:text-slate-300 hover:bg-slate-300 dark:hover:bg-slate-500',
            ].join(' ')}
            aria-label={visible ? `Hide ${id}` : `Show ${id}`}
            title={visible ? 'Hide widget' : 'Show widget'}
        >
            <svg
                xmlns="http://www.w3.org/2000/svg"
                width="14"
                height="14"
                viewBox="0 0 24 24"
                fill="none"
                stroke="currentColor"
                strokeWidth="2"
                strokeLinecap="round"
                strokeLinejoin="round"
            >
                {visible ? (
                    <>
                        <path d="M1 12s4-8 11-8 11 8 11 8-4 8-11 8-11-8-11-8z" />
                        <circle cx="12" cy="12" r="3" />
                    </>
                ) : (
                    <>
                        <path d="M17.94 17.94A10.07 10.07 0 0 1 12 20c-7 0-11-8-11-8a18.45 18.45 0 0 1 5.06-5.94" />
                        <path d="M9.9 4.24A9.12 9.12 0 0 1 12 4c7 0 11 8 11 8a18.5 18.5 0 0 1-2.16 3.19" />
                        <line x1="1" y1="1" x2="23" y2="23" />
                    </>
                )}
            </svg>
        </button>
    );
}

// ── CSS-in-JS: animation styles (injected once) ──────────────────────

const ANIMATION_STYLES_ID = 'drag-drop-dashboard-styles';

function injectAnimationStyles() {
    if (typeof document === 'undefined') return;
    if (document.getElementById(ANIMATION_STYLES_ID)) return;

    const style = document.createElement('style');
    style.id = ANIMATION_STYLES_ID;
    style.textContent = [
        '.react-grid-item {',
        '    transition: all 200ms ease !important;',
        '    transition-property: left, top, width, height !important;',
        '}',
        '.react-grid-item.cssTransforms {',
        '    transition-property: transform, width, height !important;',
        '}',
        '.react-grid-item.react-draggable-dragging {',
        '    z-index: 100;',
        '    transform: scale(1.02) !important;',
        '    box-shadow: 0 8px 32px rgba(0, 0, 0, 0.12), 0 2px 8px rgba(0, 0, 0, 0.06);',
        '    transition: box-shadow 200ms ease !important;',
        '    border-radius: 12px;',
        '}',
        '.react-grid-item.react-grid-placeholder {',
        '    background: #3b82f6;',
        '    border-radius: 12px;',
        '    opacity: 0.12;',
        '    transition: all 200ms ease !important;',
        '}',
        '.dark .react-grid-item.react-draggable-dragging {',
        '    box-shadow: 0 8px 32px rgba(0, 0, 0, 0.4), 0 2px 8px rgba(0, 0, 0, 0.2);',
        '}',
        '.react-grid-layout {',
        '    user-select: none;',
        '    -webkit-user-select: none;',
        '}',
        '.drag-handle {',
        '    touch-action: none;',
        '}',
    ].join('\n');
    document.head.appendChild(style);
}

// Inject styles once on first load
if (typeof window !== 'undefined') {
    if (document.readyState === 'loading') {
        document.addEventListener('DOMContentLoaded', injectAnimationStyles);
    } else {
        injectAnimationStyles();
    }
}

// ── Main Component ───────────────────────────────────────────────────

export function DragDropDashboard({
    widgets,
    customizeMode = false,
    visibleWidgets: externalVisible,
    onToggleWidget,
    onResetLayout,
    storageKey = STORAGE_KEY_DEFAULT,
    className = '',
}: DragDropDashboardProps) {
    const defaultLayoutsRef = React.useRef<Record<string, MutableLayout>>(
        createDefaultLayouts(),
    );

    // Measure container width for responsive grid
    const { width, containerRef } = useContainerWidth({
        measureBeforeMount: true,
    });

    // ── Layout state (localStorage-backed) ───────────────────────────
    const [layouts, setLayouts] = React.useState<Record<string, MutableLayout>>(
        () => loadLayouts(storageKey) ?? defaultLayoutsRef.current,
    );

    // ── Visibility state (localStorage-backed) ───────────────────────
    const [internalVisible, setInternalVisible] = React.useState<string[]>(
        () => loadVisibility(STORAGE_KEY_VISIBILITY) ?? [...ALL_WIDGET_IDS],
    );

    const visibleIds = externalVisible ?? internalVisible;

    // Persist layout on every change
    const handleLayoutChange = React.useCallback(
        (
            _currentLayout: readonly LayoutItem[],
            allLayouts: ResponsiveLayouts<string>,
        ) => {
            const mutable: Record<string, MutableLayout> = {};
            for (const bp of Object.keys(allLayouts)) {
                const layout = allLayouts[bp];
                if (layout) {
                    mutable[bp] = layout.map((item) => ({ ...item }));
                }
            }
            setLayouts(mutable);
            saveLayouts(storageKey, mutable);
        },
        [storageKey],
    );

    // Toggle visibility
    const handleToggle = React.useCallback(
        (id: string) => {
            const next = visibleIds.includes(id)
                ? visibleIds.filter((v) => v !== id)
                : [...visibleIds, id];

            if (onToggleWidget) {
                onToggleWidget(id, next.includes(id));
            }
            setInternalVisible(next);
            saveVisibility(STORAGE_KEY_VISIBILITY, next);
        },
        [visibleIds, onToggleWidget],
    );

    // Reset layout and visibility to defaults
    const handleReset = React.useCallback(() => {
        const defaults = createDefaultLayouts();
        setLayouts(defaults);
        saveLayouts(storageKey, defaults);
        setInternalVisible([...ALL_WIDGET_IDS]);
        saveVisibility(STORAGE_KEY_VISIBILITY, [...ALL_WIDGET_IDS]);
        onResetLayout?.();
    }, [storageKey, onResetLayout]);

    // Filter visible widgets
    const visibleWidgets = widgets.filter((w) => visibleIds.includes(w.id));

    // Build grid items with drag handles
    const gridItems = visibleWidgets.map((widget) => (
        <div
            key={widget.id}
            className="relative group"
            style={{ height: '100%' }}
        >
            {/* Drag handle — always visible in customize mode, hover otherwise */}
            <div
                className={[
                    'drag-handle',
                    'absolute top-2 left-2 z-10',
                    'cursor-grab active:cursor-grabbing',
                    'p-1 rounded',
                    'bg-white/80 dark:bg-slate-700/80',
                    'shadow-sm border border-slate-200 dark:border-slate-600',
                    'transition-all duration-200',
                    customizeMode
                        ? 'opacity-100 scale-100'
                        : 'opacity-0 group-hover:opacity-100',
                ].join(' ')}
                aria-label="Drag to reorder"
                title="Drag to reorder"
            >
                <GripVertical className="w-3.5 h-3.5 text-slate-400" />
            </div>

            {/* Customize-mode eye toggle */}
            {customizeMode && (
                <WidgetVisibilityToggle
                    id={widget.id}
                    visible={visibleIds.includes(widget.id)}
                    onToggle={handleToggle}
                />
            )}

            {/* Widget content */}
            {widget.content}
        </div>
    ));

    return (
        <div className={`relative ${className}`} ref={containerRef}>
            {/* Reset button — shown only in customize mode */}
            {customizeMode && gridItems.length > 0 && (
                <div className="flex justify-end mb-3">
                    <button
                        onClick={handleReset}
                        className={[
                            'inline-flex items-center gap-1.5 px-3 py-1.5',
                            'text-xs font-medium',
                            'text-slate-600 dark:text-slate-300',
                            'bg-white dark:bg-slate-800',
                            'border border-slate-200 dark:border-slate-700',
                            'rounded-lg hover:bg-slate-50 dark:hover:bg-slate-700',
                            'transition-colors shadow-sm',
                        ].join(' ')}
                    >
                        <RotateCcw className="w-3.5 h-3.5" />
                        Reset Layout
                    </button>
                </div>
            )}

            {/* Responsive Grid */}
            <div className="drag-drop-grid-wrapper">
                <ResponsiveGridLayout
                    className="layout"
                    width={width}
                    layouts={layouts as unknown as ResponsiveLayouts<string>}
                    breakpoints={BREAKPOINTS}
                    cols={COLS}
                    rowHeight={ROW_HEIGHT}
                    margin={[12, 12]}
                    containerPadding={[0, 0]}
                    onLayoutChange={handleLayoutChange}
                    dragConfig={{
                        handle: '.drag-handle',
                        enabled: customizeMode,
                    }}
                    resizeConfig={{
                        enabled: false,
                    }}
                    autoSize={true}
                >
                    {gridItems}
                </ResponsiveGridLayout>
            </div>

            {/* Empty state when all widgets are hidden */}
            {gridItems.length === 0 && (
                <div className="flex flex-col items-center justify-center py-16 text-slate-400 dark:text-slate-500">
                    <p className="text-sm font-medium">
                        All widgets are hidden
                    </p>
                    <p className="text-xs mt-1">
                        Open customize mode to show widgets again
                    </p>
                </div>
            )}
        </div>
    );
}
