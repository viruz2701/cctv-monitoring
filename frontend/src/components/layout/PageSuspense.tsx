import { Suspense, type ReactNode } from 'react';
import { ErrorBoundaryLite } from '../ErrorBoundaryLite';
import { SkeletonPage } from '../ui/Skeleton';

// ─────────────────────────────────────────────────────────────────────────────
// PageSuspense — компонент-обёртка для per-route загрузки
// ─────────────────────────────────────────────────────────────────────────────
// Использование:
//   <Route path="/dashboard" element={<PageSuspense><Dashboard /></PageSuspense>} />
//
// Внутри:
//   1. ErrorBoundaryLite — ловит ошибки в секции (per-route)
//   2. Suspense с SkeletonPage — показывается пока ленивый компонент грузится
// ─────────────────────────────────────────────────────────────────────────────

interface PageSuspenseProps {
  children: ReactNode;
}

export function PageSuspense({ children }: PageSuspenseProps) {
  return (
    <ErrorBoundaryLite>
      <Suspense fallback={<SkeletonPage><div /></SkeletonPage>}>
        {children}
      </Suspense>
    </ErrorBoundaryLite>
  );
}
