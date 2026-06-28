import React, { useState, useEffect, useRef, lazy, Suspense } from 'react';
import { Outlet, useLocation, useNavigate } from 'react-router-dom';
import { Sidebar } from './Sidebar';
import { Header } from './Header';
import { ErrorBoundaryLite } from '../ErrorBoundaryLite';
import { useAlarmWebSocket } from '../../services/websocket';
import { CommandPalette } from '../ui/CommandPalette';
import { ShortcutsCheatsheet } from '../ui/ShortcutsCheatsheet';
import { useCommandPaletteStore } from '../../store/commandPaletteStore';
import { useSkipLink } from '../../hooks/useAccessibility';
import { useKeyboardShortcuts } from '../../hooks/useKeyboardShortcuts';
import type { Shortcut } from '../../hooks/useKeyboardShortcuts';
import { VisuallyHidden } from '../ui/VisuallyHidden';

// P1-PERF-BUNDLE: Lazy-load OnboardingTour (react-joyride ~200KB) — только для первой сессии
const OnboardingTour = lazy(() => import('../ui/OnboardingTour'));

export function Layout() {
    const [sidebarCollapsed, setSidebarCollapsed] = useState(false);
    const [mobileMenuOpen, setMobileMenuOpen] = useState(false);
    const [cheatsheetOpen, setCheatsheetOpen] = useState(false);
    const location = useLocation();
    const navigate = useNavigate();
    const toggleCommandPalette = useCommandPaletteStore((s) => s.toggle);
    const openCommandPalette = useCommandPaletteStore((s) => s.open);
    const liveRegionRef = useRef<HTMLDivElement>(null);

    // WCAG 2.1 AA: Skip-to-content link (UX-14.2.7)
    const { handleSkip, targetId } = useSkipLink('main-content');

    // Initialize WebSocket for real-time alarms (only connects if user is authenticated)
    useAlarmWebSocket();

    // Close mobile menu on route change
    useEffect(() => {
        setMobileMenuOpen(false);
    }, [location.pathname]);

    // Close mobile menu on window resize to desktop
    useEffect(() => {
        const handleResize = () => {
            if (window.innerWidth >= 1024) {
                setMobileMenuOpen(false);
            }
        };
        window.addEventListener('resize', handleResize);
        return () => window.removeEventListener('resize', handleResize);
    }, []);

    // UX-14.1.8: Глобальные клавиатурные шорткаты
    const shortcuts: Shortcut[] = [
        // ── Navigation ────────────────────────────────────────────────
        {
            key: 'n',
            ctrl: true,
            meta: true,
            handler: () => navigate('/work-orders'),
            description: 'Создать новый Work Order',
            category: 'navigation',
        },
        {
            key: 'd',
            ctrl: true,
            meta: true,
            handler: () => navigate('/dashboard'),
            description: 'Перейти на Dashboard',
            category: 'navigation',
        },
        {
            key: ',',
            ctrl: true,
            meta: true,
            handler: () => navigate('/settings'),
            description: 'Открыть Settings',
            category: 'navigation',
        },
        // ── Actions ──────────────────────────────────────────────────
        {
            key: 'k',
            ctrl: true,
            meta: true,
            handler: () => openCommandPalette(),
            description: 'Открыть Command Palette',
            category: 'actions',
        },
        // P1-UX.8: / (без модификаторов, вне input) → Command Palette
        {
            key: '/',
            handler: () => openCommandPalette(),
            description: 'Открыть поиск (Command Palette)',
            category: 'actions',
        },
        {
            key: '/',
            ctrl: true,
            meta: true,
            handler: () => setCheatsheetOpen((prev) => !prev),
            description: 'Показать список шорткатов',
            category: 'actions',
        },
        {
            key: '?',
            handler: () => setCheatsheetOpen((prev) => !prev),
            description: 'Показать список шорткатов',
            category: 'actions',
        },
    ];

    useKeyboardShortcuts(shortcuts);

    return (
        <div className="min-h-screen bg-slate-50 dark:bg-slate-950">
            {/* WCAG 2.1 AA: Skip Link — первый фокусируемый элемент (UX-14.2.7) */}
            <a
                href={`#${targetId}`}
                onClick={handleSkip}
                className="sr-only focus:not-sr-only focus:fixed focus:top-4 focus:left-4 focus:z-[100] focus:px-4 focus:py-2 focus:bg-blue-600 focus:text-white focus:rounded-lg focus:shadow-lg focus:outline-none"
            >
                Перейти к основному содержанию
            </a>

            {/* WCAG 2.1 AA: aria-live region для динамических обновлений (UX-14.2.7) */}
            <div
                ref={liveRegionRef}
                role="status"
                aria-live="polite"
                aria-atomic="true"
                className="sr-only"
            />

            {/* Sidebar */}
            <Sidebar
                collapsed={sidebarCollapsed}
                onToggle={() => setSidebarCollapsed(!sidebarCollapsed)}
                mobileOpen={mobileMenuOpen}
                onMobileClose={() => setMobileMenuOpen(false)}
            />

            {/* Mobile overlay */}
            {mobileMenuOpen && (
                <div
                    className="fixed inset-0 z-30 bg-slate-900/50 lg:hidden"
                    onClick={() => setMobileMenuOpen(false)}
                    aria-hidden="true"
                />
            )}

            {/* Header */}
            <Header
                sidebarCollapsed={sidebarCollapsed}
                onMobileMenuToggle={() => setMobileMenuOpen(!mobileMenuOpen)}
            />

            {/* Command Palette (⌘K) — UX-14.1.5 */}
            <CommandPalette />

            {/* Onboarding Tour — UX-14.1.6 (lazy-loaded: react-joyride ~200KB) */}
            <Suspense fallback={null}>
                <OnboardingTour />
            </Suspense>

            {/* Keyboard Shortcuts Cheatsheet (⌘/) — UX-14.1.8 */}
            <ShortcutsCheatsheet
                isOpen={cheatsheetOpen}
                onClose={() => setCheatsheetOpen(false)}
                shortcuts={shortcuts}
            />

            {/* WCAG 2.1 AA: Main content landmark (UX-14.2.7) */}
            <main
                id={targetId}
                role="main"
                tabIndex={-1}
                className={`transition-all duration-300 pt-16 min-h-screen ${sidebarCollapsed ? 'lg:ml-20' : 'lg:ml-64'
                    }`}
            >
                <div className="p-4 md:p-6 lg:p-8">
                    <ErrorBoundaryLite>
                        <Outlet />
                    </ErrorBoundaryLite>
                </div>
            </main>
        </div>
    );
}
