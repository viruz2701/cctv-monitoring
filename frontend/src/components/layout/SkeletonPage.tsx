// ═══════════════════════════════════════════════════════════════════════
// SkeletonPage — переиспользуемые скелетоны для страниц
// P1-2.4: Skeleton на всех страницах
//
// Особенности:
//   - 5 вариантов скелетонов под разные типы страниц
//   - Shimmer animation (CSS @keyframes)
//   - aria-busy="true" для accessibility
//   - Использует существующие Skeleton-компоненты из ui/Skeleton
// ═══════════════════════════════════════════════════════════════════════

import {
  SkeletonStatsCard,
  SkeletonChart,
  SkeletonTable,
  SkeletonCard,
  SkeletonFilterBar,
  SkeletonPage as UISkeletonPage,
} from '../ui/Skeleton';

// ── Shimmer animation (CSS-in-JS via style tag) ───────────────────────
const shimmerStyle = `
@keyframes skeleton-shimmer {
  0% { background-position: -200% 0; }
  100% { background-position: 200% 0; }
}
.skeleton-shimmer {
  background: linear-gradient(
    90deg,
    transparent 0%,
    rgba(255,255,255,0.08) 50%,
    transparent 100%
  );
  background-size: 200% 100%;
  animation: skeleton-shimmer 1.5s ease-in-out infinite;
}
`;

// ── Inline <style> injection ─────────────────────────────────────────
function ShimmerStyles() {
  return <style>{shimmerStyle}</style>;
}

// ═══════════════════════════════════════════════════════════════════════
// SkeletonAdvancedAnalytics — скелет для AdvancedAnalytics
// ═══════════════════════════════════════════════════════════════════════

export function SkeletonAdvancedAnalytics() {
  return (
    <div className="space-y-6" aria-label="Loading Advanced Analytics" aria-busy="true">
      <ShimmerStyles />
      <UISkeletonPage title subtitle>
        {/* KPI Cards */}
        <div className="grid grid-cols-1 md:grid-cols-4 gap-4">
          {[1, 2, 3, 4].map((i) => (
            <div key={i} className="skeleton-shimmer relative overflow-hidden rounded-xl bg-slate-100 dark:bg-slate-800 p-5">
              <div className="flex items-center gap-3 mb-3">
                <div className="w-10 h-10 rounded-lg bg-slate-200 dark:bg-slate-700" />
                <div className="space-y-1.5 flex-1">
                  <div className="h-3 w-16 bg-slate-200 dark:bg-slate-700 rounded" />
                  <div className="h-6 w-12 bg-slate-200 dark:bg-slate-700 rounded" />
                </div>
              </div>
            </div>
          ))}
        </div>

        {/* Predictive Maintenance Section */}
        <div className="rounded-xl bg-white dark:bg-slate-800 border border-slate-200 dark:border-slate-700 p-5">
          <div className="flex items-center gap-2 mb-4">
            <div className="h-5 w-5 rounded bg-slate-200 dark:bg-slate-700" />
            <div className="h-5 w-48 bg-slate-200 dark:bg-slate-700 rounded" />
          </div>
          <div className="grid grid-cols-1 sm:grid-cols-2 lg:grid-cols-4 gap-4">
            {[1, 2, 3, 4].map((i) => (
              <div key={i} className="rounded-lg border border-slate-200 dark:border-slate-700 p-4">
                <div className="h-3 w-3/4 bg-slate-200 dark:bg-slate-700 rounded mb-3" />
                <div className="h-8 w-1/2 bg-slate-200 dark:bg-slate-700 rounded mb-3" />
                <div className="h-3 w-full bg-slate-200 dark:bg-slate-700 rounded mb-2" />
                <div className="h-3 w-2/3 bg-slate-200 dark:bg-slate-700 rounded" />
              </div>
            ))}
          </div>
        </div>

        {/* Cost Analysis + Vendor Performance */}
        <div className="grid grid-cols-1 lg:grid-cols-3 gap-6">
          <div className="lg:col-span-2 rounded-xl bg-white dark:bg-slate-800 border border-slate-200 dark:border-slate-700 p-5">
            <div className="h-5 w-36 bg-slate-200 dark:bg-slate-700 rounded mb-4" />
            <div className="grid grid-cols-1 lg:grid-cols-2 gap-4">
              <div className="h-[200px] bg-slate-100 dark:bg-slate-700/50 rounded-xl" />
              <div className="h-[200px] bg-slate-100 dark:bg-slate-700/50 rounded-xl" />
            </div>
          </div>
          <div className="rounded-xl bg-white dark:bg-slate-800 border border-slate-200 dark:border-slate-700 p-5">
            <div className="h-5 w-28 bg-slate-200 dark:bg-slate-700 rounded mb-4" />
            <div className="h-[200px] bg-slate-100 dark:bg-slate-700/50 rounded-xl" />
          </div>
        </div>

        {/* Vendor Table */}
        <div className="rounded-xl bg-white dark:bg-slate-800 border border-slate-200 dark:border-slate-700 p-5">
          <div className="h-5 w-40 bg-slate-200 dark:bg-slate-700 rounded mb-4" />
          <SkeletonTable rows={4} columns={5} />
        </div>
      </UISkeletonPage>
    </div>
  );
}

// ═══════════════════════════════════════════════════════════════════════
// SkeletonDashboard — скелет для дашбордов (Dashboard, ExecutiveDashboard)
// ═══════════════════════════════════════════════════════════════════════

export function SkeletonDashboard() {
  return (
    <div className="space-y-6" aria-label="Loading dashboard" aria-busy="true">
      <ShimmerStyles />
      <div className="space-y-2">
        <div className="h-7 w-40 skeleton-shimmer bg-slate-200 dark:bg-slate-700 rounded" />
        <div className="h-4 w-64 skeleton-shimmer bg-slate-200 dark:bg-slate-700 rounded" />
      </div>

      {/* Stats Cards */}
      <div className="grid grid-cols-1 sm:grid-cols-2 lg:grid-cols-5 gap-3">
        {[1, 2, 3, 4, 5].map((i) => (
          <SkeletonStatsCard key={i} />
        ))}
      </div>

      {/* Charts */}
      <div className="grid grid-cols-1 lg:grid-cols-2 gap-4">
        <SkeletonChart />
        <SkeletonChart />
      </div>

      {/* Table */}
      <SkeletonTable rows={6} columns={5} />
    </div>
  );
}

// ═══════════════════════════════════════════════════════════════════════
// SkeletonAnalytics — скелет для Analytics
// ═══════════════════════════════════════════════════════════════════════

export function SkeletonAnalytics() {
  return (
    <div className="space-y-6" aria-label="Loading analytics" aria-busy="true">
      <ShimmerStyles />
      <div className="space-y-2">
        <div className="h-7 w-36 skeleton-shimmer bg-slate-200 dark:bg-slate-700 rounded" />
        <div className="h-4 w-56 skeleton-shimmer bg-slate-200 dark:bg-slate-700 rounded" />
      </div>

      <div className="grid grid-cols-1 md:grid-cols-4 gap-4">
        <SkeletonStatsCard count={4} withTrend />
      </div>

      <div className="grid grid-cols-1 lg:grid-cols-2 gap-6">
        <SkeletonChart />
        <SkeletonChart />
      </div>

      <SkeletonTable rows={5} columns={4} />
    </div>
  );
}

// ═══════════════════════════════════════════════════════════════════════
// SkeletonFormPage — скелет для страниц с формами (Settings, Profile)
// ═══════════════════════════════════════════════════════════════════════

export function SkeletonFormPage() {
  return (
    <div className="space-y-6" aria-label="Loading form page" aria-busy="true">
      <ShimmerStyles />
      <div className="space-y-2">
        <div className="h-7 w-32 skeleton-shimmer bg-slate-200 dark:bg-slate-700 rounded" />
        <div className="h-4 w-48 skeleton-shimmer bg-slate-200 dark:bg-slate-700 rounded" />
      </div>

      <div className="space-y-4">
        {[1, 2, 3].map((i) => (
          <SkeletonCard key={i} headerLines={1} bodyLines={3} />
        ))}
      </div>
    </div>
  );
}

// ═══════════════════════════════════════════════════════════════════════
// SkeletonListPage — скелет для страниц со списками (Devices, Tickets)
// ═══════════════════════════════════════════════════════════════════════

export function SkeletonListPage() {
  return (
    <div className="space-y-6" aria-label="Loading list page" aria-busy="true">
      <ShimmerStyles />
      <div className="space-y-2">
        <div className="h-7 w-40 skeleton-shimmer bg-slate-200 dark:bg-slate-700 rounded" />
        <div className="h-4 w-60 skeleton-shimmer bg-slate-200 dark:bg-slate-700 rounded" />
      </div>

      <SkeletonFilterBar />

      <div className="grid grid-cols-2 md:grid-cols-4 gap-4">
        <SkeletonStatsCard />
        <SkeletonStatsCard />
        <SkeletonStatsCard />
        <SkeletonStatsCard />
      </div>

      <SkeletonTable rows={8} columns={6} />
    </div>
  );
}

// ═══════════════════════════════════════════════════════════════════════
// SkeletonComplianceShield — скелет для ComplianceShield
// ═══════════════════════════════════════════════════════════════════════

export function SkeletonComplianceShield() {
  return (
    <div className="space-y-6" aria-label="Loading compliance shield" aria-busy="true">
      <ShimmerStyles />
      <div className="grid grid-cols-1 sm:grid-cols-2 lg:grid-cols-4 gap-4">
        {[1, 2, 3, 4].map((i) => (
          <SkeletonStatsCard key={i} />
        ))}
      </div>

      <div className="grid grid-cols-1 lg:grid-cols-3 gap-6">
        <SkeletonChart />
        <SkeletonChart />
        <SkeletonChart />
      </div>

      <SkeletonTable rows={5} columns={5} />
    </div>
  );
}

// ═══════════════════════════════════════════════════════════════════════
// SkeletonDetailPage — скелет для страниц деталей (DeviceDetail, TicketDetail)
// ═══════════════════════════════════════════════════════════════════════

export function SkeletonDetailPage() {
  return (
    <div className="space-y-6" aria-label="Loading detail page" aria-busy="true">
      <ShimmerStyles />
      <div className="flex items-center gap-3">
        <div className="h-8 w-8 skeleton-shimmer bg-slate-200 dark:bg-slate-700 rounded-lg" />
        <div className="space-y-2 flex-1">
          <div className="h-6 w-48 skeleton-shimmer bg-slate-200 dark:bg-slate-700 rounded" />
          <div className="h-4 w-32 skeleton-shimmer bg-slate-200 dark:bg-slate-700 rounded" />
        </div>
      </div>

      <div className="grid grid-cols-1 lg:grid-cols-3 gap-6">
        <div className="lg:col-span-2 space-y-4">
          <SkeletonCard headerLines={1} bodyLines={6} />
          <SkeletonCard headerLines={1} bodyLines={4} />
        </div>
        <div className="space-y-4">
          <SkeletonCard headerLines={1} bodyLines={3} />
          <SkeletonCard headerLines={1} bodyLines={3} />
        </div>
      </div>
    </div>
  );
}
