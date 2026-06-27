// ═══════════════════════════════════════════════════════════════════════
// DashboardHub — Unified Dashboard с role-based widgets и drag-and-drop.
//
// P1-1.1: Unified Dashboard Hub
//   - Одна страница вместо 5 разрозненных дашбордов
//   - Tabs: Overview, SLA & Compliance, Performance, Maintenance
//   - Role-based widget visibility per role
//   - Drag-and-drop customization with saved layouts
//   - URL sync: /dashboard?view=sla
//
// P1-1.2: Role-Based Default Views
//   - Technician → "My Work" (Overview)
//   - Manager → "Overview"
//   - Admin → "System Health" (Overview)
// ═══════════════════════════════════════════════════════════════════════

import React, { Suspense, lazy, useCallback, useMemo, useState } from 'react';
import { useSearchParams } from 'react-router-dom';
import { useTranslation } from 'react-i18next';
import { useAuth } from '../hooks/useAuth';
import {
    LayoutDashboard,
    Shield,
    Activity,
    Wrench,
    Loader2,
    Pencil,
    Settings2,
} from 'lucide-react';
import { DragDropDashboard } from '../components/dashboard/DragDropDashboard';
import { getWidgetsForTab } from '../components/dashboard/WidgetRegistry';
import type { DashboardWidget } from '../components/dashboard/DragDropDashboard';
import { Button } from '../components/ui';

// ═══ Lazy-loaded tab components ══════════════════════════════════════

const OverviewTab = lazy(() => import('../components/dashboard/tabs/OverviewTab'));
const SLAComplianceTab = lazy(() => import('../components/dashboard/tabs/SLAComplianceTab'));
const PerformanceTab = lazy(() => import('../components/dashboard/tabs/PerformanceTab'));
const MaintenanceTab = lazy(() => import('../components/dashboard/tabs/MaintenanceTab'));

// ═══ Tab configuration ═══════════════════════════════════════════════

interface TabConfig {
    id: string;
    labelKey: string;
    icon: React.ElementType;
    component: React.LazyExoticComponent<React.ComponentType>;
    roles: string[];
    /** Использовать DragDropDashboard вместо lazy tab */
    useWidgets?: boolean;
}

const TABS: TabConfig[] = [
    { id: 'overview', labelKey: 'overview', icon: LayoutDashboard, component: OverviewTab, roles: ['admin', 'manager', 'technician', 'viewer', 'owner', 'support'], useWidgets: true },
    { id: 'sla', labelKey: 'sla_compliance', icon: Shield, component: SLAComplianceTab, roles: ['admin', 'manager'] },
    { id: 'performance', labelKey: 'performance', icon: Activity, component: PerformanceTab, roles: ['admin', 'manager'] },
    { id: 'maintenance', labelKey: 'maintenance_schedule', icon: Wrench, component: MaintenanceTab, roles: ['admin', 'manager', 'technician'] },
];

/** Определение tab по умолчанию на основе роли (P1-1.2) */
function getDefaultTab(role: string): string {
    switch (role) {
        case 'technician':
            return 'overview'; // "My Work"
        case 'manager':
            return 'overview';
        case 'admin':
        case 'support':
            return 'overview'; // "System Health"
        default:
            return 'overview';
    }
}

// ═══ Loading Skeleton ════════════════════════════════════════════════

function TabSkeleton() {
    return (
        <div className="flex items-center justify-center h-64">
            <Loader2 className="w-8 h-8 text-blue-500 animate-spin" />
        </div>
    );
}

// ═══ Widget content components ══════════════════════════════════════

function WidgetPlaceholder({ title }: { title: string }) {
    return (
        <div className="flex items-center justify-center h-full bg-white dark:bg-slate-800 rounded-xl border border-slate-200 dark:border-slate-700 p-4">
            <p className="text-sm text-slate-500">{title}</p>
        </div>
    );
}

// ═══ DashboardHub Component ═════════════════════════════════════════

export function DashboardHub() {
    const { t } = useTranslation();
    const { user } = useAuth();
    const role = user?.role ?? 'viewer';
    const [searchParams, setSearchParams] = useSearchParams();
    const [customizeMode, setCustomizeMode] = useState(false);

    // P1-1.2: Default tab based on role
    const defaultTab = useMemo(() => getDefaultTab(role), [role]);

    // P1-1.1: URL sync
    const activeTab = searchParams.get('view') || defaultTab;

    const availableTabs = useMemo(() =>
        TABS.filter(tab => tab.roles.includes(role)),
        [role]
    );

    const handleTabChange = useCallback((tabId: string) => {
        setSearchParams({ view: tabId }, { replace: true });
        setCustomizeMode(false);
    }, [setSearchParams]);

    // Если текущий tab недоступен для роли — переключаем на первый доступный
    const safeTab = availableTabs.find(t => t.id === activeTab)?.id ?? availableTabs[0]?.id ?? 'overview';
    const currentTab = TABS.find(t => t.id === safeTab);

    const ActiveComponent = useMemo(() => {
        const tab = TABS.find(t => t.id === safeTab);
        return tab?.component ?? OverviewTab;
    }, [safeTab]);

    // P1-1.1: Role-based widgets via WidgetRegistry
    const widgetDefs = useMemo(() => getWidgetsForTab(safeTab, role), [safeTab, role]);

    const dashboardWidgets = useMemo<DashboardWidget[]>(() => {
        return widgetDefs.map(def => ({
            id: def.id,
            minW: def.minW,
            minH: def.minH,
            content: <WidgetPlaceholder title={t(def.titleKey)} />,
        }));
    }, [widgetDefs, t]);

    const showWidgets = currentTab?.useWidgets && widgetDefs.length > 0;

    return (
        <div className="p-4 md:p-6 space-y-4">
            {/* Header */}
            <div className="flex items-center justify-between">
                <div>
                    <h1 className="text-2xl font-bold text-slate-900 dark:text-white">
                        {t('dashboard')}
                    </h1>
                    <p className="text-sm text-slate-500 dark:text-slate-400 mt-1">
                        {t('dashboard_hub_description') || 'Monitor and manage your system'}
                    </p>
                </div>
                {showWidgets && (
                    <Button
                        variant={customizeMode ? 'primary' : 'outline'}
                        onClick={() => setCustomizeMode(prev => !prev)}
                        icon={customizeMode ? <Settings2 className="w-4 h-4" /> : <Pencil className="w-4 h-4" />}
                    >
                        {customizeMode
                            ? (t('done_customizing') || 'Done')
                            : (t('customize') || 'Customize')
                        }
                    </Button>
                )}
            </div>

            {/* P1-1.1: Tab Navigation */}
            <div className="border-b border-slate-200 dark:border-slate-700" role="tablist" aria-label="Dashboard tabs">
                <nav className="flex space-x-1 overflow-x-auto">
                    {availableTabs.map((tab) => {
                        const Icon = tab.icon;
                        const isActive = safeTab === tab.id;
                        return (
                            <button
                                key={tab.id}
                                onClick={() => handleTabChange(tab.id)}
                                role="tab"
                                aria-selected={isActive}
                                aria-controls={`panel-${tab.id}`}
                                className={`flex items-center gap-2 px-4 py-3 text-sm font-medium border-b-2 transition-colors whitespace-nowrap ${
                                    isActive
                                        ? 'border-blue-600 text-blue-600 dark:text-blue-400 dark:border-blue-400'
                                        : 'border-transparent text-slate-500 hover:text-slate-700 dark:text-slate-400 dark:hover:text-slate-300'
                                }`}
                            >
                                <Icon className="w-4 h-4" />
                                {t(tab.labelKey)}
                            </button>
                        );
                    })}
                </nav>
            </div>

            {/* P1-1.1: Tab Content — widgets or lazy-loaded tab */}
            <div
                role="tabpanel"
                id={`panel-${safeTab}`}
                aria-labelledby={`tab-${safeTab}`}
                className="min-h-[400px]"
            >
                {showWidgets ? (
                    <DragDropDashboard
                        widgets={dashboardWidgets}
                        customizeMode={customizeMode}
                        storageKey={`dashboard:layout:${safeTab}`}
                    />
                ) : (
                    <Suspense fallback={<TabSkeleton />}>
                        <ActiveComponent />
                    </Suspense>
                )}
            </div>
        </div>
    );
}

export default DashboardHub;
