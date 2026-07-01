// ═══════════════════════════════════════════════════════════════════════
// AppProviders — Lazy-loaded provider tree (P0-CR-06)
//
// Все провайдеры и AppShell в одном динамическом chunk, чтобы main bundle
// содержал только минимальный код для инициализации.
// ═══════════════════════════════════════════════════════════════════════

import { Suspense, lazy } from 'react';
import { QueryClient, QueryClientProvider } from '@tanstack/react-query';
import { ThemeProvider } from '../store';
import { AuthProvider } from '../hooks/useAuth';
import { ToastProvider } from '../components/ui';
import { initSentry, SentryErrorBoundary } from '../lib/sentry';

// Sentry init — на уровне модуля, до монтирования React
initSentry(import.meta.env.VITE_SENTRY_DSN, {
  environment: import.meta.env.MODE,
  tracesSampleRate: import.meta.env.PROD ? 0.2 : 0.0,
  replaysSessionSampleRate: import.meta.env.PROD ? 0.1 : 0.0,
  replaysOnErrorSampleRate: import.meta.env.PROD ? 1.0 : 0.0,
});

// AppShell содержит Layout + BrowserRouter + все роуты
const AppShell = lazy(() => import('./AppShell'));

// ── React Query Client (ARCH-02) ─────────────────────────────────────
const queryClient = new QueryClient({
  defaultOptions: {
    queries: {
      staleTime: 30_000,
      retry: 2,
      refetchOnWindowFocus: true,
      refetchOnReconnect: true,
    },
  },
});

export default function AppProviders() {
  return (
    <SentryErrorBoundary context={{ layer: 'app-root' }}>
    <QueryClientProvider client={queryClient}>
    <ThemeProvider>
      <ToastProvider>
        <AuthProvider>
          <Suspense fallback={null}>
            <AppShell />
          </Suspense>
        </AuthProvider>
        </ToastProvider>
      </ThemeProvider>
    </QueryClientProvider>
    </SentryErrorBoundary>
  );
}
