// ═══════════════════════════════════════════════════════════════════════
// DashboardHub — Unified Dashboard с tabs.
//
// P1-1.1: Unified Dashboard Hub
//   - Одна страница вместо 5 разрозненных дашбордов
//   - Tabs: Overview, SLA & Compliance, Performance, Maintenance
//   - Lazy-load widgets per tab
//   - URL sync: /dashboard?view=sla
//
// P1-1.2: Role-Based Default Views
//   - Technician → "My Work" (Overview)
//   - Manager → "Overview"
//   - Admin → "System Health" (Overview)
// ═══════════════════════════════════════════════════════════════════════

import React, { Suspense, lazy, useCallback, useMemo } from 'react';
import { useSearchParams } from 'react-router-dom';
import { useTranslation } from 'react-i18next';
import { useAuth } from '../hooks/useAuth';
import {
    LayoutDashboard,
    Shield,
    Activity,
    Wrench,
    Loader2,
} from 'lucide-react';

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
}

const TABS: TabConfig[] = [
    { id: 'overview', labelKey: 'overview', icon: LayoutDashboard, component: OverviewTab, roles: ['admin', 'manager', 'technician', 'viewer', 'owner', 'support'] },
    { id: 'sla', labelKey: 'sla_compliance', icon: Shield, component: SLAComplianceTab, roles: ['admin', 'manager'] },
    { id: 'performance', labelKey: 'performance', icon: Activity, component: PerformanceTab, roles: ['admin', 'manager', 'technician'] },
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

// ═══ DashboardHub Component ═════════════════════════════════════════

export function DashboardHub() {
    const { t } = useTranslation();
    const { user } = useAuth();
    const role = user?.role ?? 'viewer';
    const [searchParams, setSearchParams] = useSearchParams();

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
    }, [setSearchParams]);

    // Если текущий tab недоступен для роли — переключаем на первый доступный
    const safeTab = availableTabs.find(t => t.id === activeTab)?.id ?? availableTabs[0]?.id ?? 'overview';

    const ActiveComponent = useMemo(() => {
        const tab = TABS.find(t => t.id === safeTab);
        return tab?.component ?? OverviewTab;
    }, [safeTab]);

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

            {/* P1-1.1: Tab Content (lazy-loaded) */}
            <div
                role="tabpanel"
                id={`panel-${safeTab}`}
                aria-labelledby={`tab-${safeTab}`}
                className="min-h-[400px]"
            >
                <Suspense fallback={<TabSkeleton />}>
                    <ActiveComponent />
                </Suspense>
            </div>
        </div>
    );
}

export default DashboardHub;
