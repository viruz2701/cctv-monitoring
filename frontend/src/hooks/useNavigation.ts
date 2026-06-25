// useNavigation — централизованный хук навигации.
//
// P0-2.1: Grouped navigation (5 parents: Dashboard, Assets, Operations, Insights, Administration)
// P0-2.2: Role-based filtering
// P0-2.3: Quick access bar
//
// Compliance:
//   - WCAG 2.1 AA (keyboard navigation via Arrow keys)
//   - ISO 27001 A.9.2.1 (Role-based access control)

import { useMemo, useState } from 'react';
import { useAuth } from './useAuth';
import type { LucideIcon } from 'lucide-react';
import {
    LayoutDashboard,
    MapPin,
    HardDrive,
    Ticket,
    FileText,
    Users,
    Settings,
    Camera,
    Shield,
    Activity,
    Truck,
    BarChart3,
    TrendingUp,
    Clock,
    Building2,
    Key,
    Webhook,
    Phone,
    Video,
    Archive,
} from 'lucide-react';
import { useTranslation } from 'react-i18next';

// ═══════════════════════════════════════════════════════════════════
// Types
// ═══════════════════════════════════════════════════════════════════

export interface NavItem {
    path: string;
    label: string;
    icon: LucideIcon;
    roles: string[];
    /** Для P0-2.3: можно ли добавить в quick access */
    quickAccessible?: boolean;
}

export interface NavGroup {
    id: string;
    label: string;
    icon: LucideIcon;
    children: NavItem[];
    /** Минимальная роль для видимости группы */
    minRole?: string;
}

export interface NavigationState {
    /** Сгруппированные пункты для sidebar */
    groups: NavGroup[];
    /** Quick access пункты (3-4 самых важных) */
    quickAccess: NavItem[];
    /** Все плоские пункты для поиска */
    flatItems: NavItem[];
    /** Роль пользователя */
    role: string;
}

// ═══════════════════════════════════════════════════════════════════
// All navigation items (source of truth)
// ═══════════════════════════════════════════════════════════════════

// Используем i18n ключи как label — они будут переведены через useTranslation
const NAV_ITEMS: NavItem[] = [
    // ── Dashboard ──────────────────────────────────────────────
    { path: '/dashboard', label: 'dashboard', icon: LayoutDashboard, roles: ['admin', 'manager', 'technician', 'viewer', 'owner', 'support'], quickAccessible: true },
    { path: '/manager-dashboard', label: 'manager_dashboard', icon: LayoutDashboard, roles: ['admin', 'manager'] },
    { path: '/executive-dashboard', label: 'executive_dashboard', icon: BarChart3, roles: ['admin', 'manager'] },
    { path: '/cost-dashboard', label: 'cost_dashboard', icon: TrendingUp, roles: ['admin', 'manager'] },
    { path: '/compliance-shield', label: 'compliance_shield', icon: Shield, roles: ['admin', 'manager'] },

    // ── Assets ─────────────────────────────────────────────────
    { path: '/sites', label: 'sites', icon: MapPin, roles: ['admin', 'manager', 'technician', 'viewer', 'owner', 'support'], quickAccessible: true },
    { path: '/devices', label: 'devices', icon: HardDrive, roles: ['admin', 'manager', 'technician', 'viewer', 'owner', 'support'], quickAccessible: true },
    { path: '/location-tree', label: 'location_tree', icon: Building2, roles: ['admin', 'manager', 'technician'] },
    { path: '/asset-overview', label: 'asset_overview', icon: HardDrive, roles: ['admin', 'manager', 'technician'] },
    { path: '/spare-parts', label: 'spare_parts', icon: HardDrive, roles: ['admin', 'manager', 'technician'] },
    { path: '/meter-dashboard', label: 'meter_dashboard', icon: Activity, roles: ['admin', 'manager'] },

    // ── Operations ─────────────────────────────────────────────
    { path: '/work-orders', label: 'work_orders', icon: Ticket, roles: ['admin', 'manager', 'technician'], quickAccessible: true },
    { path: '/tickets', label: 'tickets', icon: Ticket, roles: ['admin', 'manager', 'technician', 'viewer', 'owner', 'support'] },
    { path: '/alerts', label: 'alerts', icon: Shield, roles: ['admin', 'manager', 'technician', 'viewer', 'owner', 'support'] },
    { path: '/maintenance', label: 'maintenance', icon: FileText, roles: ['admin', 'manager', 'technician'] },
    { path: '/on-call', label: 'on_call', icon: Phone, roles: ['admin', 'manager'] },
    { path: '/maintenance-reports', label: 'maintenance_reports', icon: FileText, roles: ['admin', 'manager'] },

    // ── Insights ───────────────────────────────────────────────
    { path: '/reports', label: 'reports', icon: FileText, roles: ['admin', 'manager', 'technician', 'viewer', 'owner', 'support'] },
    { path: '/sla', label: 'sla', icon: TrendingUp, roles: ['admin', 'manager'] },
    { path: '/predictive-maintenance', label: 'predictive_maintenance', icon: TrendingUp, roles: ['admin', 'manager', 'technician'] },
    { path: '/vendor-performance', label: 'vendor_performance', icon: Truck, roles: ['admin', 'manager'] },
    { path: '/workload-analytics', label: 'workload_analytics', icon: BarChart3, roles: ['admin', 'manager'] },
    { path: '/wo-aging', label: 'wo_aging', icon: Clock, roles: ['admin', 'manager'] },
    { path: '/advanced-analytics', label: 'advanced_analytics', icon: BarChart3, roles: ['admin', 'support'] },
    { path: '/analytics', label: 'analytics', icon: TrendingUp, roles: ['admin', 'support', 'owner'] },

    // ── Administration ─────────────────────────────────────────
    { path: '/users', label: 'users', icon: Users, roles: ['admin'] },
    { path: '/settings', label: 'settings', icon: Settings, roles: ['admin'] },
    { path: '/webhooks', label: 'webhooks', icon: Webhook, roles: ['admin'] },
    { path: '/api-keys', label: 'api_keys', icon: Key, roles: ['admin'] },
    { path: '/audit-log', label: 'audit_log', icon: Shield, roles: ['admin', 'support'] },
    { path: '/logs', label: 'logs', icon: FileText, roles: ['admin', 'support'] },
    { path: '/blackbox', label: 'blackbox', icon: Archive, roles: ['admin', 'support'] },

    // ── Help ───────────────────────────────────────────────────
    { path: '/tutorials', label: 'tutorials', icon: Video, roles: ['admin', 'manager', 'technician', 'viewer', 'owner', 'support'] },
];

// ═══════════════════════════════════════════════════════════════════
// Группировка
// ═══════════════════════════════════════════════════════════════════

interface NavGroupDef {
    id: string;
    label: string;
    icon: LucideIcon;
    minRole: string;
}

const NAV_GROUPS: NavGroupDef[] = [
    { id: 'dashboard', label: 'dashboard_group', icon: LayoutDashboard, minRole: 'viewer' },
    { id: 'assets', label: 'assets_group', icon: MapPin, minRole: 'viewer' },
    { id: 'operations', label: 'operations_group', icon: Ticket, minRole: 'technician' },
    { id: 'insights', label: 'insights_group', icon: BarChart3, minRole: 'viewer' },
    { id: 'administration', label: 'administration_group', icon: Settings, minRole: 'support' },
];

// Маппинг групп по пути
const GROUP_BY_PATH: Record<string, string> = {
    // Dashboard
    '/dashboard': 'dashboard',
    '/manager-dashboard': 'dashboard',
    '/executive-dashboard': 'dashboard',
    '/cost-dashboard': 'dashboard',
    '/compliance-shield': 'dashboard',
    // Assets
    '/sites': 'assets',
    '/devices': 'assets',
    '/location-tree': 'assets',
    '/asset-overview': 'assets',
    '/spare-parts': 'assets',
    '/meter-dashboard': 'assets',
    // Operations
    '/work-orders': 'operations',
    '/tickets': 'operations',
    '/alerts': 'operations',
    '/maintenance': 'operations',
    '/on-call': 'operations',
    '/maintenance-reports': 'operations',
    // Insights
    '/reports': 'insights',
    '/sla': 'insights',
    '/predictive-maintenance': 'insights',
    '/vendor-performance': 'insights',
    '/workload-analytics': 'insights',
    '/wo-aging': 'insights',
    '/advanced-analytics': 'insights',
    '/analytics': 'insights',
    // Administration
    '/users': 'administration',
    '/settings': 'administration',
    '/webhooks': 'administration',
    '/api-keys': 'administration',
    '/audit-log': 'administration',
    '/logs': 'administration',
    '/blackbox': 'administration',
    // Help (без группы)
    '/tutorials': '',
};

// ═══════════════════════════════════════════════════════════════════
// Quick access defaults
// ═══════════════════════════════════════════════════════════════════

const DEFAULT_QUICK_ACCESS: string[] = [
    '/dashboard',
    '/devices',
    '/work-orders',
    '/alerts',
];

const STORAGE_KEY = 'sidebar_expanded_groups';
const QUICK_ACCESS_KEY = 'sidebar_quick_access';

/** Загрузка expanded групп из localStorage */
function loadExpandedGroups(): Record<string, boolean> {
    try {
        const saved = localStorage.getItem(STORAGE_KEY);
        if (saved) return JSON.parse(saved);
    } catch { /* ignore */ }
    return { dashboard: true, assets: true, operations: true, insights: true, administration: true };
}

/** Сохранение expanded групп в localStorage */
function saveExpandedGroups(groups: Record<string, boolean>): void {
    try {
        localStorage.setItem(STORAGE_KEY, JSON.stringify(groups));
    } catch { /* ignore */ }
}

/** Загрузка quick access из localStorage */
function loadQuickAccess(): string[] {
    try {
        const saved = localStorage.getItem(QUICK_ACCESS_KEY);
        if (saved) return JSON.parse(saved);
    } catch { /* ignore */ }
    return DEFAULT_QUICK_ACCESS;
}

export interface UseNavigationResult extends NavigationState {
    /** Проверить, раскрыта ли группа */
    isGroupExpanded: (groupId: string) => boolean;
    /** Переключить раскрытие группы */
    toggleGroup: (groupId: string) => void;
    /** Обновить quick access */
    setQuickAccess: (paths: string[]) => void;
}

/**
 * useNavigation — централизованный хук навигации.
 *
 * P0-2.1: Возвращает сгруппированные NavGroup
 * P0-2.2: Фильтрует по роли пользователя
 * P0-2.3: Возвращает quick access пункты
 */
export function useNavigation(): UseNavigationResult {
    const { user } = useAuth();
    const { t } = useTranslation();
    const role = user?.role ?? 'viewer';

    const [expandedGroups, setExpandedGroups] = useState<Record<string, boolean>>(loadExpandedGroups);
    const [quickAccessPaths, setQuickAccessPaths] = useState<string[]>(loadQuickAccess);

    // Фильтрация по роли
    const filteredItems = useMemo(() =>
        NAV_ITEMS.filter(item => item.roles.includes(role)),
        [role]
    );

    // Группировка
    const groups = useMemo<NavGroup[]>(() => {
        return NAV_GROUPS
            .filter(group => {
                const children = filteredItems.filter(item => GROUP_BY_PATH[item.path] === group.id);
                return children.length > 0;
            })
            .map(group => ({
                id: group.id,
                label: t(group.label),
                icon: group.icon,
                children: filteredItems
                    .filter(item => GROUP_BY_PATH[item.path] === group.id)
                    .map(item => ({ ...item, label: t(item.label) })),
            }));
    }, [filteredItems, t]);

    // Плоский список для поиска (с переводом)
    const flatItems = useMemo(() =>
        filteredItems.map(item => ({ ...item, label: t(item.label) })),
        [filteredItems, t]
    );

    // Quick access
    const quickAccess = useMemo(() => {
        const items: NavItem[] = [];
        for (const path of quickAccessPaths) {
            const item = flatItems.find(i => i.path === path);
            if (item) items.push(item);
            if (items.length >= 4) break;
        }
        if (items.length === 0) {
            return flatItems.filter(i => i.quickAccessible).slice(0, 4);
        }
        return items;
    }, [quickAccessPaths, flatItems]);

    const isGroupExpanded = (groupId: string): boolean => {
        return expandedGroups[groupId] ?? true;
    };

    const toggleGroup = (groupId: string): void => {
        setExpandedGroups(prev => {
            const next = { ...prev, [groupId]: !prev[groupId] };
            saveExpandedGroups(next);
            return next;
        });
    };

    const setQuickAccess = (paths: string[]): void => {
        setQuickAccessPaths(paths);
        try {
            localStorage.setItem(QUICK_ACCESS_KEY, JSON.stringify(paths));
        } catch { /* ignore */ }
    };

    return {
        groups,
        quickAccess,
        flatItems,
        role,
        isGroupExpanded,
        toggleGroup,
        setQuickAccess,
    };
}
