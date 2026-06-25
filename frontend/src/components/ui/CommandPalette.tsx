// ═══════════════════════════════════════════════════════════════════════
// Command Palette (⌘K)
// UX-14.1.5: Linear-style command palette для быстрой навигации
//
// Features:
//   - ⌘K / Ctrl+K для открытия
//   - Fuzzy search по страницам и действиям
//   - Группировка по категориям с иконками
//   - Навигация стрелками + Enter
//   - Escape для закрытия
//   - Подсветка совпадений в результатах
// ═══════════════════════════════════════════════════════════════════════

import React, { useState, useEffect, useRef, useCallback, useMemo } from 'react';
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
} from 'lucide-react';
import type { LucideIcon } from 'lucide-react';
import { useCommandPaletteStore } from '../../store/commandPaletteStore';

// ═══════════════════════════════════════════════════════════════════════
// Types
// ═══════════════════════════════════════════════════════════════════════

interface CommandItem {
  id: string;
  label: string;
  description?: string;
  path: string;
  category: CommandCategory;
  icon: LucideIcon;
  keywords: string[];
}

type CommandCategory =
  | 'dashboard'
  | 'inventory'
  | 'maintenance'
  | 'monitoring'
  | 'reports'
  | 'administration'
  | 'analytics';

interface CategoryGroup {
  key: CommandCategory;
  label: string;
}

// ═══════════════════════════════════════════════════════════════════════
// Constants
// ═══════════════════════════════════════════════════════════════════════

const CATEGORIES: CategoryGroup[] = [
  { key: 'dashboard', label: 'Dashboard' },
  { key: 'inventory', label: 'Inventory & Assets' },
  { key: 'maintenance', label: 'Maintenance & CMMS' },
  { key: 'monitoring', label: 'Monitoring & Alerts' },
  { key: 'reports', label: 'Reports & Analytics' },
  { key: 'administration', label: 'Administration' },
  { key: 'analytics', label: 'Advanced Analytics' },
];

const ALL_COMMANDS: CommandItem[] = [
  // ── Dashboard ─────────────────────────────────────────────────────
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
    id: 'manager-dashboard',
    label: 'Manager Dashboard',
    description: 'Management overview',
    path: '/manager-dashboard',
    category: 'dashboard',
    icon: LayoutDashboard,
    keywords: ['manager', 'management', 'kpi'],
  },
  {
    id: 'executive-dashboard',
    label: 'Executive Dashboard',
    description: 'Executive-level metrics',
    path: '/executive-dashboard',
    category: 'dashboard',
    icon: BarChart3,
    keywords: ['executive', 'exec', 'metrics', 'business'],
  },
  {
    id: 'technician-dashboard',
    label: 'Technician Dashboard',
    description: 'Technician work overview',
    path: '/technician-dashboard',
    category: 'dashboard',
    icon: Activity,
    keywords: ['technician', 'tech', 'field', 'work'],
  },

  // ── Inventory & Assets ────────────────────────────────────────────
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

  // ── Maintenance & CMMS ────────────────────────────────────────────
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

  // ── Monitoring & Alerts ───────────────────────────────────────────
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
    id: 'sla',
    label: 'SLA Dashboard',
    description: 'Service level agreements',
    path: '/sla',
    category: 'monitoring',
    icon: TrendingUp,
    keywords: ['sla', 'uptime', 'agreement', 'compliance'],
  },

  // ── Reports & Analytics ────────────────────────────────────────────
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

  // ── Administration ─────────────────────────────────────────────────
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
];

// ═══════════════════════════════════════════════════════════════════════
// Fuzzy match
// ═══════════════════════════════════════════════════════════════════════

function fuzzyMatch(text: string, query: string): boolean {
  const lower = text.toLowerCase();
  const q = query.toLowerCase();
  let qi = 0;
  for (let i = 0; i < lower.length && qi < q.length; i++) {
    if (lower[i] === q[qi]) qi++;
  }
  return qi === q.length;
}

// ═══════════════════════════════════════════════════════════════════════
// Highlight matching text
// ═══════════════════════════════════════════════════════════════════════

function HighlightMatch({ text, query }: { text: string; query: string }) {
  if (!query.trim()) return <>{text}</>;

  const lower = text.toLowerCase();
  const q = query.toLowerCase();
  const parts: { char: string; match: boolean }[] = [];
  let qi = 0;

  for (let i = 0; i < text.length; i++) {
    const isMatch = qi < q.length && lower[i] === q[qi];
    if (isMatch) qi++;
    parts.push({ char: text[i], match: isMatch });
  }

  return (
    <>
      {parts.map((p, i) =>
        p.match ? (
          <span key={i} className="text-blue-600 dark:text-blue-400 font-semibold underline decoration-blue-300/50 decoration-2 underline-offset-2">
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

export function CommandPalette() {
  const { isOpen, close } = useCommandPaletteStore();
  const navigate = useNavigate();

  const [query, setQuery] = useState('');
  const [selectedIndex, setSelectedIndex] = useState(0);
  const inputRef = useRef<HTMLInputElement>(null);
  const listRef = useRef<HTMLDivElement>(null);

  // Filter commands based on fuzzy match
  const filtered = useMemo(() => {
    if (!query.trim()) {
      // When empty, show all grouped by category
      return ALL_COMMANDS;
    }
    return ALL_COMMANDS.filter(
      (cmd) =>
        fuzzyMatch(cmd.label, query) ||
        fuzzyMatch(cmd.description ?? '', query) ||
        cmd.keywords.some((kw) => fuzzyMatch(kw, query))
    );
  }, [query]);

  // Group filtered results by category
  const grouped = useMemo(() => {
    const groups = new Map<CommandCategory, CommandItem[]>();
    for (const cmd of filtered) {
      const existing = groups.get(cmd.category) ?? [];
      existing.push(cmd);
      groups.set(cmd.category, existing);
    }
    return CATEGORIES.filter((cat) => (groups.get(cat.key)?.length ?? 0) > 0).map(
      (cat) => ({
        ...cat,
        items: groups.get(cat.key) ?? [],
      })
    );
  }, [filtered]);

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

  // Reset selection when results change
  useEffect(() => {
    setSelectedIndex(0);
  }, [query]);

  // Auto-focus input
  useEffect(() => {
    if (isOpen) {
      // Small delay to ensure DOM is ready
      requestAnimationFrame(() => {
        inputRef.current?.focus();
      });
      setQuery('');
      setSelectedIndex(0);
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
      close();
      navigate(item.path);
    },
    [close, navigate]
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

  return createPortal(
    <div
      className="fixed inset-0 z-[100] flex items-start justify-center pt-[15vh]"
      onKeyDown={handleKeyDown}
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
            placeholder="Search pages, actions, settings..."
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
        <div
          ref={listRef}
          className="max-h-[50vh] overflow-y-auto overscroll-contain py-2"
        >
          {!hasResults && (
            <div className="px-6 py-12 text-center">
              <Search className="w-8 h-8 mx-auto mb-3 text-slate-300 dark:text-slate-600" />
              <p className="text-sm font-medium text-slate-500 dark:text-slate-400">
                No results for "<span className="text-slate-700 dark:text-slate-300">{query}</span>"
              </p>
              <p className="text-xs text-slate-400 dark:text-slate-500 mt-1">
                Try a different search term
              </p>
            </div>
          )}

          {grouped.map((group) => (
            <div key={group.key}>
              <div className="px-5 py-1.5 text-xs font-semibold text-slate-400 dark:text-slate-500 uppercase tracking-wider">
                {group.label}
              </div>
              {group.items.map((item) => {
                const flatIdx = flatItems.findIndex(
                  (f) => f.item.id === item.id
                );
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
          ))}
        </div>

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
  );
}
