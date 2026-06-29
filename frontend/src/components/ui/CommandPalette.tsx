// ═══════════════════════════════════════════════════════════════════════
// Command Palette (⌘K) v2
// UX-14.2.6: Global Search (⌘K) — weighted fuzzy search + recent + categories
//
// Features:
//   - ⌘K / Ctrl+K для открытия
//   - Weighted fuzzy search (label > keywords > description)
//   - Recent commands (последние 5, localStorage)
//   - Группировка по категориям с иконками на группу
//   - Action commands (New WO, New Ticket, Toggle Dark Mode, Shortcuts)
//   - Навигация стрелками + Enter + Tab trap
//   - Подсветка совпадений через getCharMatches
//   - Keyboard Shortshots как команда
// ═══════════════════════════════════════════════════════════════════════

import React, { useState, useEffect, useRef, useCallback, useMemo } from 'react';
import { useFocusTrap } from '../../hooks/useAccessibility';
import { useNavigate } from 'react-router-dom';
import { createPortal } from 'react-dom';
import {
  LayoutDashboard,
  MapPin,
  HardDrive,
  Ticket,
  FileText,
  Users,
  Settings,
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
  Search,
  Camera,
  Command,
  PlusCircle,
  SunMoon,
  Keyboard,
  History,
  Sparkles,
  Video,
  Archive,
  Server,
  Wrench,
  Package,
  User,
  Loader2,
} from './Icons';
import type { LucideIcon } from './Icons';
import { useCommandPaletteStore } from '../../store/commandPaletteStore';
import { useThemeStore, type Theme } from '../../store/themeStore';
import { weightedFuzzySearch, getCharMatches } from '../../lib/fuzzySearch';
import type { SearchableItem } from '../../lib/fuzzySearch';
import { useSearchEntities } from '../../hooks/useSearchEntities';
import type { EntityResult } from '../../hooks/useSearchEntities';

// ═══════════════════════════════════════════════════════════════════════
// Types
// ═══════════════════════════════════════════════════════════════════════

type CommandCategory =
  | 'dashboard'
  | 'inventory'
  | 'maintenance'
  | 'monitoring'
  | 'reports'
  | 'administration'
  | 'analytics'
  | 'actions';

interface CommandItem extends SearchableItem {
  id: string;
  label: string;
  description?: string;
  path?: string;
  category: CommandCategory;
  icon: LucideIcon;
  keywords: string[];
  /** If set, this is an action command (no navigation) */
  action?: () => void;
}

interface CategoryGroup {
  key: CommandCategory;
  label: string;
  icon: LucideIcon;
}

// ═══════════════════════════════════════════════════════════════════════
// Constants
// ═══════════════════════════════════════════════════════════════════════

const CATEGORIES: CategoryGroup[] = [
  { key: 'dashboard', label: 'Dashboard', icon: LayoutDashboard },
  { key: 'inventory', label: 'Inventory & Assets', icon: Camera },
  { key: 'maintenance', label: 'Maintenance & CMMS', icon: Ticket },
  { key: 'monitoring', label: 'Monitoring & Alerts', icon: Activity },
  { key: 'reports', label: 'Reports & Analytics', icon: FileText },
  { key: 'administration', label: 'Administration', icon: Settings },
  { key: 'analytics', label: 'Advanced Analytics', icon: TrendingUp },
  { key: 'actions', label: 'Actions', icon: Sparkles },
];

const CATEGORY_ICONS: Record<CommandCategory, LucideIcon> = Object.fromEntries(
  CATEGORIES.map((c) => [c.key, c.icon])
) as Record<CommandCategory, LucideIcon>;

// ── Navigation commands ─────────────────────────────────────────────

const NAV_COMMANDS: CommandItem[] = [
  // Dashboard
  {
    id: 'dashboard',
    label: 'Dashboard',
    description: 'Main health overview',
    path: '/dashboard',
    category: 'dashboard',
    icon: LayoutDashboard,
    keywords: ['home', 'overview', 'main', 'health'],
  },
  {
    id: 'go-dashboard',
    label: 'Go to Dashboard',
    description: 'Navigate to main dashboard',
    path: '/dashboard',
    category: 'dashboard',
    icon: LayoutDashboard,
    keywords: ['go', 'navigate', 'open'],
  },
  // Inventory & Assets
  {
    id: 'sites',
    label: 'Sites',
    description: 'Manage locations and sites',
    path: '/sites',
    category: 'inventory',
    icon: MapPin,
    keywords: ['location', 'site', 'address', 'building'],
  },
  {
    id: 'go-devices',
    label: 'Go to Devices',
    description: 'All CCTV devices',
    path: '/devices',
    category: 'inventory',
    icon: HardDrive,
    keywords: ['go', 'navigate', 'camera', 'nvr'],
  },
  {
    id: 'devices',
    label: 'Devices',
    description: 'All CCTV devices',
    path: '/devices',
    category: 'inventory',
    icon: HardDrive,
    keywords: ['camera', 'nvr', 'dvr', 'hardware', 'equipment'],
  },
  {
    id: 'spare-parts',
    label: 'Spare Parts',
    description: 'Inventory of spare components',
    path: '/spare-parts',
    category: 'inventory',
    icon: HardDrive,
    keywords: ['parts', 'spares', 'components', 'stock', 'warehouse'],
  },
  {
    id: 'asset-overview',
    label: 'Asset Overview',
    description: 'Full asset inventory',
    path: '/asset-overview',
    category: 'inventory',
    icon: Camera,
    keywords: ['asset', 'inventory', 'equipment', 'register'],
  },
  {
    id: 'location-tree',
    label: 'Location Tree',
    description: 'Hierarchical location view',
    path: '/location-tree',
    category: 'inventory',
    icon: Building2,
    keywords: ['hierarchy', 'tree', 'organization', 'structure'],
  },

  // Maintenance & CMMS
  {
    id: 'work-orders',
    label: 'Work Orders',
    description: 'Manage maintenance tasks',
    path: '/work-orders',
    category: 'maintenance',
    icon: Ticket,
    keywords: ['wo', 'task', 'job', 'repair', 'service'],
  },
  {
    id: 'maintenance',
    label: 'Maintenance Schedules',
    description: 'Planned maintenance calendar',
    path: '/maintenance',
    category: 'maintenance',
    icon: Clock,
    keywords: ['schedule', 'calendar', 'planned', 'preventive', 'pm'],
  },
  {
    id: 'maintenance-reports',
    label: 'Maintenance Reports',
    description: 'CMMS performance reports',
    path: '/maintenance-reports',
    category: 'maintenance',
    icon: FileText,
    keywords: ['cmms', 'history', 'performance', 'metrics'],
  },
  {
    id: 'workload-analytics',
    label: 'Workload Analytics',
    description: 'Team workload distribution',
    path: '/workload-analytics',
    category: 'maintenance',
    icon: BarChart3,
    keywords: ['workload', 'capacity', 'distribution', 'team'],
  },
  {
    id: 'on-call',
    label: 'On-Call Schedule',
    description: 'Technician on-call rotation',
    path: '/on-call',
    category: 'maintenance',
    icon: Phone,
    keywords: ['oncall', 'rotation', 'duty', 'support'],
  },
  {
    id: 'wo-aging',
    label: 'WO Aging',
    description: 'Work order aging analysis',
    path: '/wo-aging',
    category: 'maintenance',
    icon: Clock,
    keywords: ['aging', 'overdue', 'pending', 'backlog', 'wo'],
  },

  // Monitoring & Alerts
  {
    id: 'alerts',
    label: 'Alerts',
    description: 'Active alerts and alarms',
    path: '/alerts',
    category: 'monitoring',
    icon: Shield,
    keywords: ['alarm', 'warning', 'critical', 'notification'],
  },
  {
    id: 'notifications',
    label: 'Notifications',
    description: 'All system notifications',
    path: '/notifications',
    category: 'monitoring',
    icon: Activity,
    keywords: ['notify', 'bell', 'updates', 'activity'],
  },
  {
    id: 'meter-dashboard',
    label: 'Meter Dashboard',
    description: 'Meter readings and monitoring',
    path: '/meter-dashboard',
    category: 'monitoring',
    icon: Activity,
    keywords: ['meter', 'reading', 'sensor', 'gauge'],
  },
  {
    id: 'blackbox',
    label: 'Black Box',
    description: 'Incident evidence collection and reports',
    path: '/blackbox',
    category: 'monitoring',
    icon: Archive,
    keywords: ['blackbox', 'incident', 'evidence', 'report', 'snapshot', 'forensic', 'investigation'],
  },
  {
    id: 'sla',
    label: 'SLA Dashboard',
    description: 'Service level agreements',
    path: '/sla',
    category: 'monitoring',
    icon: TrendingUp,
    keywords: ['sla', 'uptime', 'agreement', 'compliance'],
  },

  // Reports & Analytics
  {
    id: 'reports',
    label: 'Reports',
    description: 'Generate and view reports',
    path: '/reports',
    category: 'reports',
    icon: FileText,
    keywords: ['export', 'pdf', 'download', 'generate'],
  },
  {
    id: 'cost-dashboard',
    label: 'Cost Dashboard',
    description: 'TCO per device analysis',
    path: '/cost-dashboard',
    category: 'reports',
    icon: TrendingUp,
    keywords: ['cost', 'tco', 'budget', 'financial', 'spending'],
  },
  {
    id: 'vendor-performance',
    label: 'Vendor Performance',
    description: 'Vendor SLA compliance',
    path: '/vendor-performance',
    category: 'reports',
    icon: Truck,
    keywords: ['vendor', 'supplier', 'contractor', 'performance'],
  },
  {
    id: 'analytics',
    label: 'Analytics',
    description: 'Deep analytics dashboard',
    path: '/analytics',
    category: 'analytics',
    icon: TrendingUp,
    keywords: ['insights', 'analysis', 'data', 'metrics', 'statistics'],
  },
  {
    id: 'advanced-analytics',
    label: 'Advanced Analytics',
    description: 'Predictive maintenance, cost analysis & vendor performance',
    path: '/advanced-analytics',
    category: 'analytics',
    icon: BarChart3,
    keywords: ['advanced', 'predictive', 'cost', 'vendor', 'tco', 'mtbf', 'mttr'],
  },
  {
    id: 'predictive-maintenance',
    label: 'Predictive Maintenance',
    description: 'ML-powered failure prediction dashboard',
    path: '/predictive-maintenance',
    category: 'analytics',
    icon: TrendingUp,
    keywords: ['predictive', 'failure', 'prediction', 'ml', 'ai', 'machine learning', 'forecast'],
  },
  {
    id: 'compliance-shield',
    label: 'Compliance Shield',
    description: 'Compliance & Fines risk assessment',
    path: '/compliance-shield',
    category: 'analytics',
    icon: Shield,
    keywords: ['compliance', 'fines', 'risk', 'shield', 'exposure', 'downtime', 'financial'],
  },

  // Administration
  {
    id: 'users',
    label: 'Users',
    description: 'User management and roles',
    path: '/users',
    category: 'administration',
    icon: Users,
    keywords: ['team', 'people', 'accounts', 'permissions', 'roles'],
  },
  {
    id: 'settings',
    label: 'Settings',
    description: 'System configuration',
    path: '/settings',
    category: 'administration',
    icon: Settings,
    keywords: ['config', 'preferences', 'options', 'system'],
  },
  {
    id: 'profile',
    label: 'Profile',
    description: 'Your account settings',
    path: '/profile',
    category: 'administration',
    icon: Users,
    keywords: ['account', 'me', 'personal', 'avatar'],
  },
  {
    id: 'api-keys',
    label: 'API Keys',
    description: 'Manage API access tokens',
    path: '/api-keys',
    category: 'administration',
    icon: Key,
    keywords: ['api', 'token', 'auth', 'integration', 'developer'],
  },
  {
    id: 'webhooks',
    label: 'Webhooks',
    description: 'Outgoing webhook integrations',
    path: '/webhooks',
    category: 'administration',
    icon: Webhook,
    keywords: ['webhook', 'integration', 'callback', 'event'],
  },
  {
    id: 'audit-log',
    label: 'Audit Log',
    description: 'Security audit trail',
    path: '/audit-log',
    category: 'administration',
    icon: Shield,
    keywords: ['audit', 'log', 'security', 'trail', 'compliance'],
  },
  {
    id: 'logs',
    label: 'System Logs',
    description: 'Application error logs',
    path: '/logs',
    category: 'administration',
    icon: FileText,
    keywords: ['error', 'debug', 'system', 'application'],
  },
  {
    id: 'tickets',
    label: 'Support Tickets',
    description: 'Support ticket system',
    path: '/tickets',
    category: 'administration',
    icon: Ticket,
    keywords: ['support', 'help', 'issue', 'request'],
  },
  {
    id: 'tutorials',
    label: 'Tutorials',
    description: 'Video tutorials and onboarding guides',
    path: '/tutorials',
    category: 'administration',
    icon: Video,
    keywords: ['help', 'guide', 'video', 'learn', 'onboarding', 'training'],
  },
];

// ── Action commands ─────────────────────────────────────────────────

function createActionCommands(
  navigate: ReturnType<typeof useNavigate>,
  isDark: boolean,
  setTheme: (theme: Theme) => void,
  onOpenShortcuts: () => void
): CommandItem[] {
  return [
    {
      id: 'new-work-order',
      label: 'New Work Order',
      description: 'Create a new work order',
      path: '/work-orders',
      category: 'actions',
      icon: PlusCircle,
      keywords: ['create', 'new', 'wo', 'add', 'maintenance', 'job'],
      action: () => navigate('/work-orders'),
    },
    {
      id: 'new-ticket',
      label: 'New Ticket',
      description: 'Create a new support ticket',
      path: '/tickets',
      category: 'actions',
      icon: PlusCircle,
      keywords: ['create', 'new', 'support', 'issue', 'add'],
      action: () => navigate('/tickets'),
    },
    {
      id: 'toggle-dark-mode',
      label: 'Toggle Dark Mode',
      description: 'Switch between light and dark theme',
      category: 'actions',
      icon: SunMoon,
      keywords: ['theme', 'dark', 'light', 'mode', 'appearance', 'toggle'],
      action: () => {
        const next: Theme = isDark ? 'light' : 'dark';
        setTheme(next);
      },
    },
    {
      id: 'keyboard-shortcuts',
      label: 'Keyboard Shortcuts',
      description: 'View all keyboard shortcuts (⌘/)',
      category: 'actions',
      icon: Keyboard,
      keywords: ['shortcuts', 'keys', 'hotkeys', 'help', 'cheatsheet'],
      action: onOpenShortcuts,
    },
  ];
}

// ═══════════════════════════════════════════════════════════════════════
// Highlight matching text (v2 — uses getCharMatches)
// ═══════════════════════════════════════════════════════════════════════

function HighlightMatch({ text, query }: { text: string; query: string }) {
  const chars = useMemo(() => getCharMatches(text, query), [text, query]);

  if (!query.trim()) return <>{text}</>;

  return (
    <>
      {chars.map((p, i) =>
        p.matched ? (
          <span
            key={i}
            className="text-blue-600 dark:text-blue-400 font-semibold underline decoration-blue-300/50 decoration-2 underline-offset-2"
          >
            {p.char}
          </span>
        ) : (
          <span key={i}>{p.char}</span>
        )
      )}
    </>
  );
}

// ═══════════════════════════════════════════════════════════════════════
// Component
// ═══════════════════════════════════════════════════════════════════════

// ── Entity result icons ──────────────────────────────────────────────

const ENTITY_TYPE_ICONS: Record<string, LucideIcon> = {
  site: MapPin,
  device: HardDrive,
  'work-order': Ticket,
  'spare-part': Package,
  user: User,
};

const ENTITY_TYPE_LABELS: Record<string, string> = {
  site: 'Site',
  device: 'Device',
  'work-order': 'Work Order',
  'spare-part': 'Spare Part',
  user: 'User',
};

// ── Component ────────────────────────────────────────────────────────

export function CommandPalette() {
  const { isOpen, close, recentCommands, addRecent, clearRecent } = useCommandPaletteStore();
  const themeStore = useThemeStore();
  const navigate = useNavigate();

  const [query, setQuery] = useState('');
  const [selectedIndex, setSelectedIndex] = useState(0);
  const [showShortcuts, setShowShortcuts] = useState(false);
  const inputRef = useRef<HTMLInputElement>(null);
  const listRef = useRef<HTMLDivElement>(null);

  // P2-4: Entity search via API (debounced 300ms)
  const entitySearch = useSearchEntities(query, isOpen);

  // UX-14.2.8: Focus trap when palette is open
  const { containerRef: paletteRef, handleKeyDown: handleTrapKeyDown } = useFocusTrap(isOpen, {
    restoreFocus: true,
  });

  // Build action commands (needs hooks inside component)
  const actionCommands = useMemo(
    () => createActionCommands(navigate, themeStore.isDark, themeStore.setTheme, () => setShowShortcuts(true)),
    [navigate, themeStore.isDark, themeStore.setTheme]
  );

  // All commands (memoized static reference)
  const allCommands = useMemo<CommandItem[]>(() => [...NAV_COMMANDS, ...actionCommands], [actionCommands]);

  // ── Weighted fuzzy search ──────────────────────────────────────────
  const searchResults = useMemo(() => {
    const trimmed = query.trim();
    if (!trimmed) return null; // null means "show all grouped"
    return weightedFuzzySearch(allCommands, query, { threshold: 10 });
  }, [query, allCommands]);

  // Filtered commands (for non-empty query)
  const filteredCommands = useMemo(() => {
    if (!searchResults) return allCommands;
    return searchResults.map((r) => r.item);
  }, [searchResults, allCommands]);

  // Group filtered results by category
  const grouped = useMemo(() => {
    const groups = new Map<CommandCategory, CommandItem[]>();
    const items = searchResults ? filteredCommands : allCommands;

    for (const cmd of items) {
      const existing = groups.get(cmd.category) ?? [];
      existing.push(cmd);
      groups.set(cmd.category, existing);
    }

    return CATEGORIES.filter((cat) => (groups.get(cat.key)?.length ?? 0) > 0).map((cat) => ({
      ...cat,
      items: groups.get(cat.key) ?? [],
    }));
  }, [searchResults, filteredCommands, allCommands]);

  // Flatten for keyboard index
  const flatItems = useMemo(() => {
    const items: { category: CommandCategory; item: CommandItem }[] = [];
    for (const group of grouped) {
      for (const item of group.items) {
        items.push({ category: group.key, item });
      }
    }
    return items;
  }, [grouped]);

  // Build recent command items (only when query is empty)
  const recentItems = useMemo(() => {
    if (query.trim()) return [];
    return recentCommands
      .map((id) => allCommands.find((c) => c.id === id))
      .filter((c): c is CommandItem => c !== undefined);
  }, [recentCommands, allCommands, query]);

  // Reset selection when results change
  useEffect(() => {
    setSelectedIndex(0);
  }, [query]);

  // Auto-focus input
  useEffect(() => {
    if (isOpen) {
      requestAnimationFrame(() => {
        inputRef.current?.focus();
      });
      setQuery('');
      setSelectedIndex(0);
      setShowShortcuts(false);
    }
  }, [isOpen]);

  // Scroll selected item into view
  useEffect(() => {
    if (!listRef.current) return;
    const selected = listRef.current.querySelector('[data-selected="true"]');
    if (selected) {
      selected.scrollIntoView({ block: 'nearest' });
    }
  }, [selectedIndex]);

  const executeCommand = useCallback(
    (item: CommandItem) => {
      addRecent(item.id);
      close();

      if (item.action) {
        item.action();
      } else if (item.path) {
        navigate(item.path);
      }
    },
    [addRecent, close, navigate]
  );

  const handleKeyDown = useCallback(
    (e: React.KeyboardEvent) => {
      switch (e.key) {
        case 'ArrowDown':
          e.preventDefault();
          setSelectedIndex((prev) => Math.min(prev + 1, flatItems.length - 1));
          break;
        case 'ArrowUp':
          e.preventDefault();
          setSelectedIndex((prev) => Math.max(prev - 1, 0));
          break;
        case 'Enter':
          e.preventDefault();
          if (flatItems[selectedIndex]) {
            executeCommand(flatItems[selectedIndex].item);
          }
          break;
        case 'Escape':
          e.preventDefault();
          close();
          break;
      }
    },
    [flatItems, selectedIndex, executeCommand, close]
  );

  if (!isOpen) return null;

  const hasResults = flatItems.length > 0;
  const hasRecent = recentItems.length > 0 && !query.trim();

  // Combine palette keyboard handler with focus trap handler
  const combinedKeyDown = useCallback(
    (e: React.KeyboardEvent) => {
      if (e.key === 'Tab') {
        handleTrapKeyDown(e);
        return;
      }
      handleKeyDown(e);
    },
    [handleTrapKeyDown, handleKeyDown]
  );

  const hasEntityResults = entitySearch.results.length > 0 && query.trim().length >= 2;

  const flatIdxOffset = (hasRecent ? recentItems.length + 1 : 0) + (hasEntityResults ? entitySearch.results.length + 1 : 0);

  const resultsNode = (
    <div ref={listRef} className="max-h-[50vh] overflow-y-auto overscroll-contain py-2">
      {/* Recent commands section (only when empty query) */}
      {hasRecent && (
        <>
          <div className="px-5 py-1.5 text-xs font-semibold text-slate-400 dark:text-slate-500 uppercase tracking-wider flex items-center gap-1.5">
            <History className="w-3 h-3" />
            <span>Recent</span>
            <button
              onClick={(e) => {
                e.stopPropagation();
                clearRecent();
              }}
              className="ml-auto text-[10px] font-normal text-slate-400 hover:text-slate-600 dark:hover:text-slate-300 transition-colors"
              aria-label="Clear recent searches"
            >
              Clear
            </button>
          </div>
          {recentItems.map((item) => {
            const flatIdx = flatItems.findIndex((f) => f.item.id === item.id);
            const isSelected = flatIdx === selectedIndex;
            const Icon = item.icon;

            return (
              <button
                key={item.id}
                data-selected={isSelected ? 'true' : undefined}
                onClick={() => executeCommand(item)}
                onMouseEnter={() => setSelectedIndex(flatIdx >= 0 ? flatIdx : 0)}
                className={`w-full flex items-center gap-3 px-5 py-2.5 text-left transition-colors ${
                  isSelected
                    ? 'bg-purple-50 dark:bg-purple-900/20 text-slate-900 dark:text-white'
                    : 'text-slate-700 dark:text-slate-300 hover:bg-slate-50 dark:hover:bg-slate-800/50'
                }`}
              >
                <div
                  className={`flex items-center justify-center w-8 h-8 rounded-lg flex-shrink-0 ${
                    isSelected
                      ? 'bg-purple-100 dark:bg-purple-900/30 text-purple-600 dark:text-purple-400'
                      : 'bg-slate-100 dark:bg-slate-800 text-slate-500 dark:text-slate-400'
                  }`}
                >
                  <Icon className="w-4 h-4" />
                </div>
                <div className="flex-1 min-w-0">
                  <div className="text-sm font-medium truncate">{item.label}</div>
                  {item.description && (
                    <div className="text-xs text-slate-400 dark:text-slate-500 truncate mt-0.5">
                      {item.description}
                    </div>
                  )}
                </div>
                <div className="flex-shrink-0 text-xs text-slate-400 dark:text-slate-500 font-mono hidden sm:block">
                  {item.path}
                </div>
              </button>
            );
          })}

          {/* Separator */}
          <div className="mx-5 my-1 border-t border-slate-200 dark:border-slate-700/50" />
        </>
      )}

      {/* P2-4: Entity search results (API-driven, debounced) */}
      {hasEntityResults && (
        <>
          <div className="px-5 py-1.5 text-xs font-semibold text-slate-400 dark:text-slate-500 uppercase tracking-wider flex items-center gap-1.5">
            <Search className="w-3 h-3" />
            <span>Search results ({entitySearch.total})</span>
            {entitySearch.isFetching && (
              <Loader2 className="w-3 h-3 ml-1 animate-spin text-blue-500" />
            )}
          </div>
          {entitySearch.results.map((entity, idx) => {
            const flatIdx = idx;
            const isSelected = flatIdx === selectedIndex;
            const EntityIcon = ENTITY_TYPE_ICONS[entity.type] || Search;

            return (
              <button
                key={`entity-${entity.type}-${entity.id}`}
                data-selected={isSelected ? 'true' : undefined}
                onClick={() => {
                  addRecent(`entity-${entity.type}-${entity.id}`);
                  close();
                  navigate(entity.path);
                }}
                onMouseEnter={() => setSelectedIndex(flatIdx)}
                className={`w-full flex items-center gap-3 px-5 py-2.5 text-left transition-colors ${
                  isSelected
                    ? 'bg-amber-50 dark:bg-amber-900/20 text-slate-900 dark:text-white'
                    : 'text-slate-700 dark:text-slate-300 hover:bg-slate-50 dark:hover:bg-slate-800/50'
                }`}
              >
                <div
                  className={`flex items-center justify-center w-8 h-8 rounded-lg flex-shrink-0 ${
                    isSelected
                      ? 'bg-amber-100 dark:bg-amber-900/30 text-amber-600 dark:text-amber-400'
                      : 'bg-slate-100 dark:bg-slate-800 text-slate-500 dark:text-slate-400'
                  }`}
                >
                  <EntityIcon className="w-4 h-4" />
                </div>
                <div className="flex-1 min-w-0">
                  <div className="text-sm font-medium truncate flex items-center gap-2">
                    <HighlightMatch text={entity.label} query={query} />
                    <span className="text-[10px] font-normal text-slate-400 bg-slate-100 dark:bg-slate-800 px-1.5 py-0.5 rounded uppercase tracking-wider">
                      {ENTITY_TYPE_LABELS[entity.type] || entity.type}
                    </span>
                  </div>
                  {entity.description && (
                    <div className="text-xs text-slate-400 dark:text-slate-500 truncate mt-0.5">
                      {entity.description}
                    </div>
                  )}
                </div>
                <div className="flex-shrink-0 text-xs text-slate-400 dark:text-slate-500 font-mono hidden sm:block">
                  {entity.path}
                </div>
              </button>
            );
          })}

          {/* Separator */}
          <div className="mx-5 my-1 border-t border-slate-200 dark:border-slate-700/50" />
        </>
      )}

      {/* No results */}
      {!hasResults && !hasEntityResults && (
        <div className="px-6 py-12 text-center">
          <Search className="w-8 h-8 mx-auto mb-3 text-slate-300 dark:text-slate-600" />
          <p className="text-sm font-medium text-slate-500 dark:text-slate-400">
            {query.trim().length >= 2 ? (
              <>No results for "<span className="text-slate-700 dark:text-slate-300">{query}</span>"</>
            ) : (
              <>Type to search pages, commands, or entities</>
            )}
          </p>
          <p className="text-xs text-slate-400 dark:text-slate-500 mt-1">
            {query.trim().length >= 2 ? 'Try a different search term' : 'Search across sites, devices, work orders, and more'}
          </p>
        </div>
      )}

      {/* Grouped results */}
      {grouped.map((group) => {
        const GroupIcon = group.icon;
        return (
          <div key={group.key}>
            <div className="px-5 py-1.5 text-xs font-semibold text-slate-400 dark:text-slate-500 uppercase tracking-wider flex items-center gap-1.5">
              <GroupIcon className="w-3 h-3" />
              <span>{group.label}</span>
            </div>
            {group.items.map((item) => {
              const flatIdx = flatItems.findIndex((f) => f.item.id === item.id);
              const isSelected = flatIdx === selectedIndex;
              const Icon = item.icon;

              return (
                <button
                  key={item.id}
                  data-selected={isSelected ? 'true' : undefined}
                  onClick={() => executeCommand(item)}
                  onMouseEnter={() => setSelectedIndex(flatIdx)}
                  className={`w-full flex items-center gap-3 px-5 py-2.5 text-left transition-colors ${
                    isSelected
                      ? 'bg-blue-50 dark:bg-blue-900/20 text-slate-900 dark:text-white'
                      : 'text-slate-700 dark:text-slate-300 hover:bg-slate-50 dark:hover:bg-slate-800/50'
                  }`}
                >
                  <div
                    className={`flex items-center justify-center w-8 h-8 rounded-lg flex-shrink-0 ${
                      isSelected
                        ? 'bg-blue-100 dark:bg-blue-900/30 text-blue-600 dark:text-blue-400'
                        : 'bg-slate-100 dark:bg-slate-800 text-slate-500 dark:text-slate-400'
                    }`}
                  >
                    <Icon className="w-4 h-4" />
                  </div>
                  <div className="flex-1 min-w-0">
                    <div className="text-sm font-medium truncate">
                      <HighlightMatch text={item.label} query={query} />
                    </div>
                    {item.description && (
                      <div className="text-xs text-slate-400 dark:text-slate-500 truncate mt-0.5">
                        <HighlightMatch text={item.description} query={query} />
                      </div>
                    )}
                  </div>
                  <div className="flex-shrink-0 text-xs text-slate-400 dark:text-slate-500 font-mono hidden sm:block">
                    {item.path}
                  </div>
                </button>
              );
            })}
          </div>
        );
      })}
    </div>
  );

  return (
    <>
      {createPortal(
        <div
          ref={paletteRef}
          tabIndex={-1}
          className="fixed inset-0 z-[100] flex items-start justify-center pt-[15vh]"
          onKeyDown={combinedKeyDown}
        >
          {/* Backdrop */}
          <div
            className="absolute inset-0 bg-slate-900/60 backdrop-blur-sm"
            onClick={close}
          />

          {/* Palette */}
          <div
            className="relative w-full max-w-2xl mx-4 bg-white dark:bg-slate-900 rounded-2xl shadow-2xl border border-slate-200 dark:border-slate-700 overflow-hidden animate-in fade-in zoom-in-95 slide-in-from-bottom-8 duration-200"
            role="dialog"
            aria-modal="true"
            aria-label="Command palette"
          >
            {/* Search Input */}
            <div className="flex items-center gap-3 px-5 py-4 border-b border-slate-200 dark:border-slate-800">
              <Search className="w-5 h-5 text-slate-400 flex-shrink-0" />
              <input
                ref={inputRef}
                type="text"
                value={query}
                onChange={(e) => setQuery(e.target.value)}
                placeholder="Search pages, entities, actions... (try ⌘N for new WO)"
                className="flex-1 text-base bg-transparent text-slate-900 dark:text-white placeholder:text-slate-400 dark:placeholder:text-slate-500 border-0 outline-none focus:outline-none focus:ring-0"
                autoComplete="off"
                spellCheck={false}
              />
              <kbd className="hidden sm:inline-flex items-center gap-1 px-2 py-1 text-xs font-medium text-slate-400 dark:text-slate-500 bg-slate-100 dark:bg-slate-800 rounded-md border border-slate-200 dark:border-slate-700">
                <Command className="w-3 h-3" />
                <span>K</span>
              </kbd>
              {query && (
                <button
                  onClick={() => setQuery('')}
                  className="p-1 text-slate-400 hover:text-slate-600 dark:hover:text-slate-300 rounded-md hover:bg-slate-100 dark:hover:bg-slate-800 transition-colors"
                  aria-label="Clear search"
                >
                  <svg className="w-4 h-4" fill="none" viewBox="0 0 24 24" stroke="currentColor" strokeWidth={2}>
                    <path strokeLinecap="round" strokeLinejoin="round" d="M6 18L18 6M6 6l12 12" />
                  </svg>
                </button>
              )}
            </div>

            {/* Results */}
            {resultsNode}

            {/* Footer */}
            <div className="px-5 py-2.5 border-t border-slate-200 dark:border-slate-800 bg-slate-50 dark:bg-slate-900/50 flex items-center gap-4 text-xs text-slate-400 dark:text-slate-500">
              <span className="flex items-center gap-1">
                <kbd className="px-1.5 py-0.5 bg-slate-200 dark:bg-slate-800 rounded text-[10px] font-medium text-slate-500 dark:text-slate-400">↑</kbd>
                <kbd className="px-1.5 py-0.5 bg-slate-200 dark:bg-slate-800 rounded text-[10px] font-medium text-slate-500 dark:text-slate-400">↓</kbd>
                <span>navigate</span>
              </span>
              <span className="flex items-center gap-1">
                <kbd className="px-1.5 py-0.5 bg-slate-200 dark:bg-slate-800 rounded text-[10px] font-medium text-slate-500 dark:text-slate-400">↵</kbd>
                <span>open</span>
              </span>
              <span className="flex items-center gap-1">
                <kbd className="px-1.5 py-0.5 bg-slate-200 dark:bg-slate-800 rounded text-[10px] font-medium text-slate-500 dark:text-slate-400">⎋</kbd>
                <span>close</span>
              </span>
            </div>
          </div>
        </div>,
        document.body
      )}

      {/* Shortcuts Cheatsheet modal — rendered inline for state sharing */}
      {showShortcuts && (
        <ShortcutsCheatsheetModal onClose={() => setShowShortcuts(false)} />
      )}
    </>
  );
}

// ═══════════════════════════════════════════════════════════════════════
// Inline Shortcuts Cheatsheet (lazy import pattern)
// ═══════════════════════════════════════════════════════════════════════

function ShortcutsCheatsheetModal({ onClose }: { onClose: () => void }) {
  const { shortcuts } = useKeyboardShortcutsInline();

  return (
    <div className="fixed inset-0 z-[110] flex items-center justify-center">
      <div className="absolute inset-0 bg-slate-900/40 backdrop-blur-sm" onClick={onClose} />
      <div className="relative bg-white dark:bg-slate-900 rounded-2xl shadow-2xl border border-slate-200 dark:border-slate-700 max-w-lg w-full mx-4 max-h-[70vh] overflow-y-auto p-6 animate-in fade-in zoom-in-95 duration-200">
        <div className="flex items-center justify-between mb-6">
          <h2 className="text-lg font-semibold text-slate-900 dark:text-white flex items-center gap-2">
            <Keyboard className="w-5 h-5 text-slate-500" />
            Keyboard Shortcuts
          </h2>
          <button
            onClick={onClose}
            className="p-1.5 text-slate-400 hover:text-slate-600 dark:hover:text-slate-300 rounded-md hover:bg-slate-100 dark:hover:bg-slate-800 transition-colors"
            aria-label="Close shortcuts"
          >
            <svg className="w-4 h-4" fill="none" viewBox="0 0 24 24" stroke="currentColor" strokeWidth={2}>
              <path strokeLinecap="round" strokeLinejoin="round" d="M6 18L18 6M6 6l12 12" />
            </svg>
          </button>
        </div>
        <div className="space-y-4">
          {shortcuts.map((section, idx) => (
            <div key={idx}>
              <h3 className="text-xs font-semibold text-slate-500 dark:text-slate-400 uppercase tracking-wider mb-2">
                {section.category}
              </h3>
              <div className="divide-y divide-slate-100 dark:divide-slate-700/50 border border-slate-200 dark:border-slate-700 rounded-xl overflow-hidden">
                {section.items.map((item, iidx) => (
                  <div
                    key={iidx}
                    className="flex items-center justify-between px-4 py-2.5 bg-white dark:bg-slate-800/50"
                  >
                    <span className="text-sm text-slate-700 dark:text-slate-200">{item.label}</span>
                    <kbd className="inline-flex items-center gap-0.5 px-2.5 py-1 text-xs font-mono font-medium text-slate-600 dark:text-slate-300 bg-slate-100 dark:bg-slate-800 rounded-md border border-slate-200 dark:border-slate-700 shadow-sm whitespace-nowrap">
                      {item.keys}
                    </kbd>
                  </div>
                ))}
              </div>
            </div>
          ))}
        </div>
      </div>
    </div>
  );
}

/** Inline shortcut definitions (no external dependency) */
interface ShortcutSection {
  category: string;
  items: { label: string; keys: string }[];
}

function useKeyboardShortcutsInline(): { shortcuts: ShortcutSection[] } {
  const isMac = typeof navigator !== 'undefined' && navigator.platform.toLowerCase().includes('mac');
  const mod = isMac ? '⌘' : 'Ctrl';

  const shortcuts: ShortcutSection[] = [
    {
      category: 'Navigation',
      items: [
        { label: 'Open Command Palette', keys: `${mod}K` },
        { label: 'Go to Dashboard', keys: `${mod}D` },
        { label: 'Go to Devices', keys: `${mod}E` },
        { label: 'Go to Work Orders', keys: `${mod}W` },
        { label: 'Go to Alerts', keys: `${mod}A` },
      ],
    },
    {
      category: 'Actions',
      items: [
        { label: 'New Work Order', keys: `${mod}N` },
        { label: 'Refresh current view', keys: `${mod}R` },
        { label: 'Fullscreen', keys: 'F11' },
      ],
    },
    {
      category: 'Modals',
      items: [
        { label: 'Open Keyboard Shortcuts', keys: `${mod}/` },
        { label: 'Close modal / palette', keys: 'Esc' },
      ],
    },
  ];

  return { shortcuts };
}
