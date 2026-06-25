import React, { useState, useEffect, useCallback } from 'react';
import { Outlet, useLocation } from 'react-router-dom';
import { Sidebar } from './Sidebar';
import { Header } from './Header';
import { useAlarmWebSocket } from '../../services/websocket';
import { CommandPalette } from '../ui/CommandPalette';
import { useCommandPaletteStore } from '../../store/commandPaletteStore';

export function Layout() {
    const [sidebarCollapsed, setSidebarCollapsed] = useState(false);
    const [mobileMenuOpen, setMobileMenuOpen] = useState(false);
    const location = useLocation();
    const toggleCommandPalette = useCommandPaletteStore((s) => s.toggle);

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

    // Global keyboard shortcut: Cmd+K / Ctrl+K to open Command Palette
    const handleKeyDown = useCallback(
        (e: KeyboardEvent) => {
            if ((e.metaKey || e.ctrlKey) && e.key === 'k') {
                e.preventDefault();
                e.stopPropagation();
                toggleCommandPalette();
            }
        },
        [toggleCommandPalette]
    );

    useEffect(() => {
        document.addEventListener('keydown', handleKeyDown);
        return () => document.removeEventListener('keydown', handleKeyDown);
    }, [handleKeyDown]);

    return (
        <div className="min-h-screen bg-slate-50 dark:bg-slate-950">
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
                />
            )}

            {/* Header */}
            <Header
                sidebarCollapsed={sidebarCollapsed}
                onMobileMenuToggle={() => setMobileMenuOpen(!mobileMenuOpen)}
            />

            {/* Command Palette (⌘K) — UX-14.1.5 */}
            <CommandPalette />

            {/* Main Content */}
            <main
                className={`transition-all duration-300 pt-16 min-h-screen ${sidebarCollapsed ? 'lg:ml-20' : 'lg:ml-64'
                    }`}
            >
                <div className="p-4 md:p-6 lg:p-8">
                    <Outlet />
                </div>
            </main>
        </div>
    );
}
