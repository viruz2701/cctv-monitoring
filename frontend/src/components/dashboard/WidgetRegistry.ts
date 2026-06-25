// ═══════════════════════════════════════════════════════════════════════
// Widget Registry — центральный реестр виджетов дашборда
// UX-14.2.1: Dashboard Widget System — metadata, role-based access,
// tab filtering и размерные constraints для каждого виджета.
// ═══════════════════════════════════════════════════════════════════════

// ═══════════════════════════════════════════════════════════════════════
// Types
// ═══════════════════════════════════════════════════════════════════════

export interface WidgetDefinition {
    id: string;
    titleKey: string;        // i18n key
    descriptionKey: string;  // i18n key
    icon: string;            // lucide icon name
    minRole: string;         // минимальная роль для доступа
    defaultVisible: boolean;
    minW: number;
    minH: number;
    /** Массив id табов, в которых виджет доступен */
    tabs: string[];          // ['overview', 'sla', 'performance', 'maintenance']
    /** Размерность данных (для аналитики) */
    dataType?: 'device' | 'ticket' | 'alert' | 'sla' | 'cost' | 'performance';
}

// ═══════════════════════════════════════════════════════════════════════
// Registry
// ═══════════════════════════════════════════════════════════════════════

const WIDGET_REGISTRY: WidgetDefinition[] = [
    // ── Overview tab ────────────────────────────────────────────────
    {
        id: 'statsOverview',
        titleKey: 'device_statistics',
        descriptionKey: 'overview_of_all_devices',
        icon: 'Camera',
        minRole: 'viewer',
        defaultVisible: true,
        minW: 2,
        minH: 1,
        tabs: ['overview'],
    },
    {
        id: 'ticketAnalytics',
        titleKey: 'ticket_analytics',
        descriptionKey: 'open_tickets_and_resolution',
        icon: 'Ticket',
        minRole: 'viewer',
        defaultVisible: true,
        minW: 2,
        minH: 1,
        tabs: ['overview'],
    },
    {
        id: 'deviceHealthChart',
        titleKey: 'device_health',
        descriptionKey: 'device_health_distribution',
        icon: 'Heart',
        minRole: 'viewer',
        defaultVisible: true,
        minW: 1,
        minH: 2,
        tabs: ['overview', 'performance'],
    },
    {
        id: 'alertTrendChart',
        titleKey: 'alert_trend',
        descriptionKey: 'alert_trend_over_time',
        icon: 'AlertTriangle',
        minRole: 'viewer',
        defaultVisible: true,
        minW: 1,
        minH: 2,
        tabs: ['overview'],
    },
    {
        id: 'ticketTrendChart',
        titleKey: 'ticket_trend',
        descriptionKey: 'ticket_trend_by_priority',
        icon: 'BarChart3',
        minRole: 'viewer',
        defaultVisible: true,
        minW: 1,
        minH: 2,
        tabs: ['overview'],
    },
    {
        id: 'recentAlerts',
        titleKey: 'recent_alerts',
        descriptionKey: 'latest_alerts',
        icon: 'Shield',
        minRole: 'viewer',
        defaultVisible: true,
        minW: 1,
        minH: 2,
        tabs: ['overview'],
    },
    {
        id: 'latestTickets',
        titleKey: 'latest_tickets',
        descriptionKey: 'recent_tickets',
        icon: 'Ticket',
        minRole: 'viewer',
        defaultVisible: true,
        minW: 1,
        minH: 2,
        tabs: ['overview'],
    },
    {
        id: 'quickActions',
        titleKey: 'quick_actions',
        descriptionKey: 'common_actions',
        icon: 'Zap',
        minRole: 'technician',
        defaultVisible: true,
        minW: 1,
        minH: 1,
        tabs: ['overview'],
    },

    // ── SLA tab ─────────────────────────────────────────────────────
    {
        id: 'slaOverview',
        titleKey: 'sla_overview',
        descriptionKey: 'sla_compliance_overview',
        icon: 'ShieldCheck',
        minRole: 'manager',
        defaultVisible: true,
        minW: 2,
        minH: 2,
        tabs: ['sla'],
        dataType: 'sla',
    },
    {
        id: 'slaBreachChart',
        titleKey: 'sla_breaches',
        descriptionKey: 'sla_breach_trend',
        icon: 'AlertTriangle',
        minRole: 'manager',
        defaultVisible: true,
        minW: 2,
        minH: 2,
        tabs: ['sla'],
        dataType: 'sla',
    },

    // ── Performance tab ─────────────────────────────────────────────
    {
        id: 'deviceUptime',
        titleKey: 'device_uptime',
        descriptionKey: 'device_uptime_percentage',
        icon: 'Activity',
        minRole: 'viewer',
        defaultVisible: true,
        minW: 2,
        minH: 2,
        tabs: ['performance'],
        dataType: 'device',
    },
    {
        id: 'responseTime',
        titleKey: 'response_time',
        descriptionKey: 'average_response_time',
        icon: 'Clock',
        minRole: 'viewer',
        defaultVisible: true,
        minW: 2,
        minH: 2,
        tabs: ['performance'],
        dataType: 'performance',
    },

    // ── Maintenance tab ─────────────────────────────────────────────
    {
        id: 'upcomingMaintenance',
        titleKey: 'upcoming_maintenance',
        descriptionKey: 'scheduled_maintenance',
        icon: 'Wrench',
        minRole: 'technician',
        defaultVisible: true,
        minW: 2,
        minH: 2,
        tabs: ['maintenance'],
        dataType: 'ticket',
    },
    {
        id: 'overdueMaintenance',
        titleKey: 'overdue_maintenance',
        descriptionKey: 'overdue_maintenance_tasks',
        icon: 'AlertCircle',
        minRole: 'technician',
        defaultVisible: true,
        minW: 2,
        minH: 2,
        tabs: ['maintenance'],
        dataType: 'ticket',
    },
];

// ═══════════════════════════════════════════════════════════════════════
// Helper functions
// ═══════════════════════════════════════════════════════════════════════

/** Иерархия ролей для проверки доступа */
const ROLE_HIERARCHY: Record<string, number> = {
    viewer: 0,
    technician: 1,
    manager: 2,
    support: 3,
    owner: 3,
    admin: 4,
};

/**
 * Проверяет, имеет ли пользователь минимальную требуемую роль.
 * admin имеет доступ ко всему.
 */
function hasMinRole(userRole: string, minRole: string): boolean {
    const userLevel = ROLE_HIERARCHY[userRole] ?? 0;
    const requiredLevel = ROLE_HIERARCHY[minRole] ?? 0;
    return userLevel >= requiredLevel;
}

/**
 * Возвращает список виджетов, доступных для указанного таба и роли.
 */
export function getWidgetsForTab(tabId: string, role: string): WidgetDefinition[] {
    return WIDGET_REGISTRY.filter(
        (w) => w.tabs.includes(tabId) && hasMinRole(role, w.minRole),
    );
}

/**
 * Возвращает виджет по его id.
 */
export function getWidget(id: string): WidgetDefinition | undefined {
    return WIDGET_REGISTRY.find((w) => w.id === id);
}

/**
 * Возвращает все зарегистрированные виджеты.
 */
export function getAllWidgets(): WidgetDefinition[] {
    return WIDGET_REGISTRY;
}

/**
 * Возвращает массив всех уникальных id виджетов.
 */
export function getAllWidgetIds(): string[] {
    return WIDGET_REGISTRY.map((w) => w.id);
}
