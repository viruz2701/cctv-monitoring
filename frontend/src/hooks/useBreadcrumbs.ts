// ═══════════════════════════════════════════════════════════════════════
// useBreadcrumbs — Dynamic breadcrumb generation from route path
//
// UX-1.4: Breadcrumbs Enhancement
// - Dynamic generation из текущего route path
// - i18n поддержка через routeBreadcrumbLabels
// - Поддержка динамических параметров (:id, :siteId, etc.)
//
// Compliance:
//   - WCAG 2.4.1 (Bypass Blocks — breadcrumbs как навигационная помощь)
//   - WCAG 2.4.8 (Location — breadcrumbs показывают текущую позицию)
// ═══════════════════════════════════════════════════════════════════════

import { useMemo } from 'react';
import { useLocation } from 'react-router-dom';
import { useTranslation } from 'react-i18next';
import type { BreadcrumbItem } from '../components/ui/Breadcrumbs';
import {
  Home,
  type LucideIcon,
} from '../components/ui/Icons';

// ═══════════════════════════════════════════════════════════════════════
// Route → Breadcrumb mapping
// ═══════════════════════════════════════════════════════════════════════

interface BreadcrumbRouteConfig {
  /** i18n key for the label */
  labelKey: string;
  /** Optional icon */
  icon?: LucideIcon;
  /** Parent route for breadcrumb hierarchy (e.g., '/sites' is parent of '/sites/:siteId') */
  parentRoute?: string;
  /** If true, this segment is treated as a dynamic param and gets a generic label */
  isDynamic?: boolean;
}

/**
 * Route breadcrumb configuration.
 * More specific routes must come before less specific ones.
 */
const ROUTE_BREADCRUMBS: [string, BreadcrumbRouteConfig][] = [
  // ── Dashboard ────────────────────────────────────────────
  ['/dashboard', { labelKey: 'dashboard', icon: Home }],

  // ── Sites ────────────────────────────────────────────────
  ['/sites', { labelKey: 'sites', icon: Home }],
  ['/sites/:siteId', { labelKey: 'site_detail', parentRoute: '/sites' }],
  ['/sites/device/:deviceId', { labelKey: 'device_detail', parentRoute: '/sites/:siteId' }],

  // ── Devices ──────────────────────────────────────────────
  ['/devices', { labelKey: 'devices', icon: Home }],
  ['/devices/:deviceId', { labelKey: 'device_detail', parentRoute: '/devices' }],
  ['/assets/devices/:id', { labelKey: 'device_detail', parentRoute: '/devices' }],

  // ── Work Orders ──────────────────────────────────────────
  ['/work-orders', { labelKey: 'work_orders', icon: Home }],
  ['/work-orders/:id', { labelKey: 'work_order_detail', parentRoute: '/work-orders' }],

  // ── Tickets ──────────────────────────────────────────────
  ['/tickets', { labelKey: 'tickets', icon: Home }],
  ['/tickets/:ticketId', { labelKey: 'ticket_detail', parentRoute: '/tickets' }],

  // ── Hub ──────────────────────────────────────────────────
  ['/hub', { labelKey: 'work_hub', icon: Home }],

  // ── Agents ───────────────────────────────────────────────
  ['/agents', { labelKey: 'agents', icon: Home }],
  ['/agents/:id', { labelKey: 'agent_detail', parentRoute: '/agents' }],

  // ── Core pages ───────────────────────────────────────────
  ['/alerts', { labelKey: 'alerts', icon: Home }],
  ['/notifications', { labelKey: 'notifications', icon: Home }],
  ['/reports', { labelKey: 'reports', icon: Home }],
  ['/analytics', { labelKey: 'analytics', icon: Home }],
  ['/logs', { labelKey: 'logs', icon: Home }],
  ['/audit-log', { labelKey: 'audit_log', icon: Home }],
  ['/blackbox', { labelKey: 'blackbox', icon: Home }],
  ['/users', { labelKey: 'users', icon: Home }],
  ['/settings', { labelKey: 'settings', icon: Home }],
  ['/profile', { labelKey: 'profile', icon: Home }],
  ['/maintenance', { labelKey: 'maintenance', icon: Home }],
  ['/sla', { labelKey: 'sla', icon: Home }],
  ['/spare-parts', { labelKey: 'spare_parts', icon: Home }],
  ['/tutorials', { labelKey: 'tutorials', icon: Home }],
  ['/glossary', { labelKey: 'glossary', icon: Home }],

  // ── Sub-pages ────────────────────────────────────────────
  ['/technician-week', { labelKey: 'technician_week', icon: Home }],
  ['/on-call', { labelKey: 'on_call', icon: Home }],
  ['/location-tree', { labelKey: 'location_tree', icon: Home }],
  ['/asset-overview', { labelKey: 'asset_overview', icon: Home }],
  ['/meter-dashboard', { labelKey: 'meter_dashboard', icon: Home }],
  ['/cost-dashboard', { labelKey: 'cost_dashboard', icon: Home }],
  ['/compliance-shield', { labelKey: 'compliance_shield', icon: Home }],
  ['/predictive-maintenance', { labelKey: 'predictive_maintenance', icon: Home }],
  ['/vendor-performance', { labelKey: 'vendor_performance', icon: Home }],
  ['/workload-analytics', { labelKey: 'workload_analytics', icon: Home }],
  ['/wo-aging', { labelKey: 'wo_aging', icon: Home }],
  ['/advanced-analytics', { labelKey: 'advanced_analytics', icon: Home }],
  ['/maintenance-reports', { labelKey: 'maintenance_reports', icon: Home }],
  ['/webhooks', { labelKey: 'webhooks', icon: Home }],
  ['/api-keys', { labelKey: 'api_keys', icon: Home }],
  ['/events', { labelKey: 'events', icon: Home }],

  // ── Admin sub-pages ──────────────────────────────────────
  ['/admin/descriptors', { labelKey: 'protocol_descriptors', icon: Home }],
  ['/admin/descriptors/new', { labelKey: 'new_descriptor', parentRoute: '/admin/descriptors' }],
  ['/admin/descriptors/:vendor/edit', { labelKey: 'edit_descriptor', parentRoute: '/admin/descriptors' }],
  ['/bi-query', { labelKey: 'bi_query', icon: Home }],
  ['/executive-dashboard', { labelKey: 'executive_dashboard', icon: Home }],
  ['/manager-dashboard', { labelKey: 'manager_dashboard', icon: Home }],
];

/**
 * Match a pathname against a route pattern (e.g., '/sites/:siteId').
 * Returns params if matched, null otherwise.
 */
function matchRoute(pattern: string, pathname: string): Record<string, string> | null {
  const patternParts = pattern.split('/');
  const pathParts = pathname.split('/');

  if (patternParts.length !== pathParts.length) return null;

  const params: Record<string, string> = {};

  for (let i = 0; i < patternParts.length; i++) {
    if (patternParts[i].startsWith(':')) {
      const paramName = patternParts[i].slice(1);
      params[paramName] = decodeURIComponent(pathParts[i]);
    } else if (patternParts[i] !== pathParts[i]) {
      return null;
    }
  }

  return params;
}

/**
 * Build breadcrumb hierarchy from pathname.
 * Traverses parentRoute chain to build the full breadcrumb path.
 */
function buildBreadcrumbHierarchy(
  pathname: string,
  config: BreadcrumbRouteConfig,
  t: (key: string) => string,
): BreadcrumbItem[] {
  const crumbs: BreadcrumbItem[] = [];

  // Recursively build parent chain
  const buildChain = (route: string, cfg: BreadcrumbRouteConfig) => {
    if (cfg.parentRoute) {
      // Find parent config
      for (const [parentPattern, parentCfg] of ROUTE_BREADCRUMBS) {
        if (parentPattern === cfg.parentRoute) {
          buildChain(parentPattern, parentCfg);
          break;
        }
      }
    }

    // Build label with params if dynamic
    let label: string;
    const match = matchRoute(route, pathname);
    if (match && Object.keys(match).length > 0) {
      // Dynamic param — use param value as label
      const paramValue = Object.values(match)[0];
      label = paramValue;
    } else {
      label = t(cfg.labelKey);
    }

    crumbs.push({
      label,
      href: route === pathname ? undefined : route,
      icon: cfg.icon,
    });
  };

  buildChain(pathname, config);
  return crumbs;
}

/**
 * useBreadcrumbs — динамическая генерация breadcrumbs из текущего route.
 *
 * @returns Массив BreadcrumbItem[] для передачи в <Breadcrumbs items={...} />
 *
 * @example
 * ```tsx
 * const breadcrumbs = useBreadcrumbs();
 * return <Breadcrumbs items={breadcrumbs} />;
 * ```
 */
export function useBreadcrumbs(): BreadcrumbItem[] {
  const location = useLocation();
  const { t } = useTranslation();

  return useMemo(() => {
    const pathname = location.pathname;

    // Find matching route config (most specific first)
    for (const [pattern, config] of ROUTE_BREADCRUMBS) {
      const match = matchRoute(pattern, pathname);
      if (match) {
        return buildBreadcrumbHierarchy(pathname, config, t);
      }
    }

    // Fallback: generate from path segments
    const segments = pathname.split('/').filter(Boolean);
    return segments.map((segment, idx) => ({
      label: segment.charAt(0).toUpperCase() + segment.slice(1),
      href: idx < segments.length - 1 ? '/' + segments.slice(0, idx + 1).join('/') : undefined,
    }));
  }, [location.pathname, t]);
}
