// ═══════════════════════════════════════════════════════════════════════
// sidebarGroups.ts — Domain groups for sidebar progressive disclosure
//
// UX-1.1: 5 доменов → Operations, Assets, Analytics, Governance, Admin
// 14 видимых пунктов + 5 групп-аккордеонов с collapse state в localStorage
//
// Compliance:
//   - IEC 62443 SR 3.1 (RBAC — роли через roles[])
//   - ISO 27001 A.9.2.1 (Role-based access control)
// ═══════════════════════════════════════════════════════════════════════

import type { LucideIcon } from '../components/ui/Icons';
import {
  Activity,
  Archive,
  BarChart3,
  Bell,
  BookOpen,
  Building2,
  Calendar,
  Clock,
  Database,
  FileText,
  HardDrive,
  History,
  Key,
  LayoutDashboard,
  MapPin,
  Phone,
  Server,
  Settings,
  Shield,
  Ticket,
  TrendingUp,
  Truck,
  Users,
  Video,
  Webhook,
  Wrench,
} from '../components/ui/Icons';

// ═══════════════════════════════════════════════════════════════════════
// Types
// ═══════════════════════════════════════════════════════════════════════

export interface SidebarItem {
  /** Route path */
  path: string;
  /** i18n key for label */
  label: string;
  /** Icon component */
  icon: LucideIcon;
  /** Allowed roles */
  roles: string[];
  /** Visible in quick-access bar */
  quickAccessible?: boolean;
  /** Always visible outside accordion (progressive disclosure) */
  alwaysVisible?: boolean;
}

export interface SidebarGroup {
  /** Group identifier */
  id: string;
  /** Display label (i18n key) */
  label: string;
  /** Group icon */
  icon: LucideIcon;
  /** Minimum role to see this group */
  minRole: string;
  /** Items in this group */
  items: SidebarItem[];
  /** Default expanded state */
  defaultExpanded?: boolean;
}

// ═══════════════════════════════════════════════════════════════════════
// Navigation Items (source of truth)
// ═══════════════════════════════════════════════════════════════════════

/** 14 always-visible items + items grouped into 5 domains */
export const ALWAYS_VISIBLE_ITEMS: SidebarItem[] = [
  { path: '/dashboard', label: 'dashboard', icon: LayoutDashboard, roles: ['admin', 'manager', 'technician', 'viewer', 'owner', 'support'], quickAccessible: true, alwaysVisible: true },
  { path: '/sites', label: 'sites', icon: MapPin, roles: ['admin', 'manager', 'technician', 'viewer', 'owner', 'support'], quickAccessible: true, alwaysVisible: true },
  { path: '/devices', label: 'devices', icon: HardDrive, roles: ['admin', 'manager', 'technician', 'viewer', 'owner', 'support'], quickAccessible: true, alwaysVisible: true },
  { path: '/work-orders', label: 'work_orders', icon: Ticket, roles: ['admin', 'manager', 'technician'], quickAccessible: true, alwaysVisible: true },
  { path: '/tickets', label: 'tickets', icon: Ticket, roles: ['admin', 'manager', 'technician', 'viewer', 'owner', 'support'], alwaysVisible: true },
  { path: '/alerts', label: 'alerts', icon: Bell, roles: ['admin', 'manager', 'technician', 'viewer', 'owner', 'support'], quickAccessible: true, alwaysVisible: true },
  { path: '/reports', label: 'reports', icon: FileText, roles: ['admin', 'manager', 'technician', 'viewer', 'owner', 'support'], alwaysVisible: true },
  { path: '/analytics', label: 'analytics', icon: TrendingUp, roles: ['admin', 'support', 'owner'], alwaysVisible: true },
  { path: '/maintenance', label: 'maintenance', icon: Wrench, roles: ['admin', 'manager', 'technician'], alwaysVisible: true },
  { path: '/sla', label: 'sla', icon: Activity, roles: ['admin', 'manager'], alwaysVisible: true },
  { path: '/users', label: 'users', icon: Users, roles: ['admin'], alwaysVisible: true },
  { path: '/settings', label: 'settings', icon: Settings, roles: ['admin'], alwaysVisible: true },
  { path: '/audit-log', label: 'audit_log', icon: History, roles: ['admin', 'support'], alwaysVisible: true },
  { path: '/tutorials', label: 'tutorials', icon: Video, roles: ['admin', 'manager', 'technician', 'viewer', 'owner', 'support'], alwaysVisible: true },
];

/**
 * 5 domain groups with accordion behavior.
 * Items not in ALWAYS_VISIBLE_ITEMS live inside groups.
 */
export const SIDEBAR_GROUPS: SidebarGroup[] = [
  // ── Operations ─────────────────────────────────────────────
  {
    id: 'operations',
    label: 'operations_group',
    icon: Ticket,
    minRole: 'technician',
    defaultExpanded: false,
    items: [
      { path: '/on-call', label: 'on_call', icon: Phone, roles: ['admin', 'manager'] },
      { path: '/maintenance-reports', label: 'maintenance_reports', icon: FileText, roles: ['admin', 'manager'] },
      { path: '/hub', label: 'work_hub', icon: Calendar, roles: ['admin', 'manager', 'technician'] },
      { path: '/technician-week', label: 'technician_week', icon: Calendar, roles: ['admin', 'manager', 'technician'] },
    ],
  },
  // ── Assets ─────────────────────────────────────────────────
  {
    id: 'assets',
    label: 'assets_group',
    icon: Server,
    minRole: 'viewer',
    defaultExpanded: false,
    items: [
      { path: '/location-tree', label: 'location_tree', icon: Building2, roles: ['admin', 'manager', 'technician'] },
      { path: '/asset-overview', label: 'asset_overview', icon: Database, roles: ['admin', 'manager', 'technician'] },
      { path: '/spare-parts', label: 'spare_parts', icon: HardDrive, roles: ['admin', 'manager', 'technician'] },
      { path: '/meter-dashboard', label: 'meter_dashboard', icon: Activity, roles: ['admin', 'manager'] },
    ],
  },
  // ── Analytics ──────────────────────────────────────────────
  {
    id: 'analytics',
    label: 'analytics_group',
    icon: BarChart3,
    minRole: 'viewer',
    defaultExpanded: false,
    items: [
      { path: '/cost-dashboard', label: 'cost_dashboard', icon: TrendingUp, roles: ['admin', 'manager'] },
      { path: '/predictive-maintenance', label: 'predictive_maintenance', icon: TrendingUp, roles: ['admin', 'manager', 'technician'] },
      { path: '/vendor-performance', label: 'vendor_performance', icon: Truck, roles: ['admin', 'manager'] },
      { path: '/workload-analytics', label: 'workload_analytics', icon: BarChart3, roles: ['admin', 'manager'] },
      { path: '/wo-aging', label: 'wo_aging', icon: Clock, roles: ['admin', 'manager'] },
      { path: '/advanced-analytics', label: 'advanced_analytics', icon: BarChart3, roles: ['admin', 'support'] },
      { path: '/executive-dashboard', label: 'executive_dashboard', icon: LayoutDashboard, roles: ['admin', 'manager'] },
    ],
  },
  // ── Governance ─────────────────────────────────────────────
  {
    id: 'governance',
    label: 'governance_group',
    icon: Shield,
    minRole: 'support',
    defaultExpanded: false,
    items: [
      { path: '/compliance-shield', label: 'compliance_shield', icon: Shield, roles: ['admin', 'manager'] },
      { path: '/logs', label: 'logs', icon: FileText, roles: ['admin', 'support'] },
      { path: '/blackbox', label: 'blackbox', icon: Archive, roles: ['admin', 'support'] },
      { path: '/manager-dashboard', label: 'manager_dashboard', icon: LayoutDashboard, roles: ['admin', 'manager'] },
    ],
  },
  // ── Admin ──────────────────────────────────────────────────
  {
    id: 'admin',
    label: 'admin_group',
    icon: Settings,
    minRole: 'admin',
    defaultExpanded: false,
    items: [
      { path: '/webhooks', label: 'webhooks', icon: Webhook, roles: ['admin'] },
      { path: '/api-keys', label: 'api_keys', icon: Key, roles: ['admin'] },
      { path: '/notifications', label: 'notifications', icon: Bell, roles: ['admin', 'manager', 'technician', 'viewer', 'owner', 'support'] },
      { path: '/glossary', label: 'glossary', icon: BookOpen, roles: ['admin', 'manager', 'technician', 'viewer', 'owner', 'support'] },
    ],
  },
];

// ── Quick access defaults (from always-visible items) ────────────────
export const DEFAULT_QUICK_ACCESS: string[] = [
  '/dashboard',
  '/devices',
  '/work-orders',
  '/alerts',
];
