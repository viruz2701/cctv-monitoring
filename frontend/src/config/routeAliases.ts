// ═══════════════════════════════════════════════════════════════════════
// routeAliases.ts — Route aliasing configuration
//
// UX-1.3: Route aliasing middleware support
// Старые URL → новые URL с HTTP 301 redirect
//
// Compliance:
//   - WCAG 2.4.1 (Bypass Blocks — redirects сохраняют navigation landmarks)
// ═══════════════════════════════════════════════════════════════════════

export interface RouteAlias {
  /** Source path pattern (React Router path syntax) */
  from: string;
  /** Target path (может содержать :params из from) */
  to: string;
  /** Description for Sentry logging */
  description: string;
}

/**
 * Route aliases: старые URL → новые.
 * Используются в RouteAliasingMiddleware для 301 redirect.
 */
export const ROUTE_ALIASES: RouteAlias[] = [
  {
    from: '/work-orders',
    to: '/hub?tab=tasks',
    description: 'UX-1.3: /work-orders → /hub?tab=tasks',
  },
  {
    from: '/tickets',
    to: '/hub?tab=requests',
    description: 'UX-1.3: /tickets → /hub?tab=requests',
  },
  {
    from: '/devices/:id',
    to: '/assets/devices/:id',
    description: 'UX-1.3: /devices/:id → /assets/devices/:id',
  },
];

/**
 * Build redirect path from matched route alias and URL params.
 */
export function buildRedirectPath(
  alias: RouteAlias,
  params: Record<string, string>,
): string {
  let target = alias.to;
  for (const [key, value] of Object.entries(params)) {
    target = target.replace(`:${key}`, encodeURIComponent(value));
  }
  return target;
}
