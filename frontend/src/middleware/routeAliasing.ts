// ═══════════════════════════════════════════════════════════════════════
// RouteAliasingMiddleware — Redirect old routes to new ones
//
// UX-1.3: Route aliasing middleware
// - /work-orders → /hub?tab=tasks
// - /tickets → /hub?tab=requests
// - /devices/:id → /assets/devices/:id
// - HTTP 301 redirect (replace)
// - Sentry logging
//
// Используется в Layout.tsx — запускается на каждый route change.
//
// Compliance:
//   - OWASP ASVS V1.8 (Unvalidated redirects — только whitelisted aliases)
//   - ISO 27001 A.12.4.1 (Event logging — via Sentry)
// ═══════════════════════════════════════════════════════════════════════

import { useEffect } from 'react';
import { useLocation, useNavigate } from 'react-router-dom';
import { ROUTE_ALIASES, buildRedirectPath } from '../config/routeAliases';
import * as Sentry from '@sentry/react';

/**
 * Match a pathname against a route pattern (e.g., '/devices/:id').
 * Returns extracted params if matched, null otherwise.
 */
function matchAlias(pattern: string, pathname: string): Record<string, string> | null {
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
 * RouteAliasingMiddleware — проверяет текущий URL на совпадение
 * с алисами и выполняет 301 redirect при необходимости.
 */
export function RouteAliasingMiddleware() {
  const location = useLocation();
  const navigate = useNavigate();

  useEffect(() => {
    const pathname = location.pathname;

    for (const alias of ROUTE_ALIASES) {
      const params = matchAlias(alias.from, pathname);
      if (params) {
        const target = buildRedirectPath(alias, params);

        // Sentry logging
        Sentry.addBreadcrumb({
          category: 'navigation',
          message: `Route alias: ${pathname} → ${target}`,
          level: 'info',
          data: {
            from: pathname,
            to: target,
            alias: alias.description,
          },
        });

        // HTTP 301 redirect via React Router replace
        navigate(target, { replace: true });
        return;
      }
    }
  }, [location.pathname, navigate]);

  return null;
}
