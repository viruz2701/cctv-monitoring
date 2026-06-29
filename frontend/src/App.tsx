import { useEffect, type ReactNode } from 'react';
import { BrowserRouter, Routes, Route, Navigate, useLocation } from 'react-router-dom';
import { QueryClient, QueryClientProvider } from '@tanstack/react-query';
import { Layout, PageSuspense } from './components/layout';

import { ThemeProvider } from './store';
import { AuthProvider, useAuth } from './hooks/useAuth';
import { ToastProvider } from './components/ui';

// ── Sentry (QA.4) ─────────────────────────────────────────────────────────
// Инициализация на уровне модуля — до монтирования React
import { initSentry, SentryErrorBoundary, setSentryUser } from './lib/sentry';
initSentry(import.meta.env.VITE_SENTRY_DSN, {
  environment: import.meta.env.MODE,
  tracesSampleRate: import.meta.env.PROD ? 0.2 : 0.0,
  replaysSessionSampleRate: import.meta.env.PROD ? 0.1 : 0.0,
  replaysOnErrorSampleRate: import.meta.env.PROD ? 1.0 : 0.0,
});

// ── Lazy-loaded pages via barrel ──────────────────────────────────────────
import * as Pages from './pages';

import { RoleProtectedRoute } from './components/auth/RoleProtectedRoute';

function ProtectedRoute({ children }: { children: ReactNode }) {
  const { user } = useAuth();
  const location = useLocation();

  if (!user) {
    return <Navigate to="/login" state={{ from: location }} replace />;
  }

  return children;
}

// ── Sentry User Context Sync ─────────────────────────────────────────────
function SentryUserSync({ children }: { children: ReactNode }) {
  const { user } = useAuth();

  useEffect(() => {
    if (user) {
      setSentryUser({
        id: user.id,
        email: user.email ?? '',
        role: user.role ?? 'unknown',
      });
    } else {
      setSentryUser(null);
    }
  }, [user]);

  return <>{children}</>;
}

// ── React Query Client (ARCH-02) ─────────────────────────────────────
// Единый QueryClient для server state.
// staleTime по умолчанию — 30s (переопределяется в каждом query).
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

function App() {
  return (
    <SentryErrorBoundary context={{ layer: 'app-root' }}>
    <QueryClientProvider client={queryClient}>
    <ThemeProvider>
      <ToastProvider>
        <AuthProvider>
          <SentryUserSync>
            <BrowserRouter>
              <Routes>
                {/* Public Routes */}
                <Route path="/login" element={<Pages.Login />} />
                <Route path="/forgot-password" element={<Pages.ForgotPassword />} />
                <Route path="/request" element={<Pages.WorkRequestPortal />} />
                <Route path="/security-advisories" element={<PageSuspense><Pages.SecurityAdvisories /></PageSuspense>} />

                {/* P0-CE.4: Setup Wizard — публичный, без Layout (On-Premise first-run) */}
                <Route path="/setup" element={
                  <PageSuspense><Pages.SetupWizard /></PageSuspense>
                } />

                {/* Protected Routes with Layout */}
                <Route element={
                  <ProtectedRoute>
                    <Layout />
                  </ProtectedRoute>
                }>
                  <Route path="/dashboard" element={<PageSuspense><Pages.DashboardHub /></PageSuspense>} />
                  <Route path="/sites" element={<PageSuspense><Pages.Sites /></PageSuspense>} />
                  <Route path="/sites/:siteId" element={<PageSuspense><Pages.SiteDetail /></PageSuspense>} />
                  <Route path="/sites/device/:deviceId" element={<PageSuspense><Pages.DeviceDetail /></PageSuspense>} />
                  <Route path="/devices" element={<PageSuspense><Pages.Devices /></PageSuspense>} />
                  <Route path="/devices/:deviceId" element={<PageSuspense><Pages.DeviceDetail /></PageSuspense>} />
                  <Route path="/tickets" element={<PageSuspense><Pages.Tickets /></PageSuspense>} />
                  <Route path="/tickets/:ticketId" element={<PageSuspense><Pages.TicketDetail /></PageSuspense>} />
                  <Route path="/alerts" element={<PageSuspense><Pages.Alerts /></PageSuspense>} />
                  <Route path="/notifications" element={<PageSuspense><Pages.Notifications /></PageSuspense>} />

                  <Route element={<RoleProtectedRoute allowedRoles={['admin', 'manager', 'technician']} />}>
                    <Route path="/reports" element={<PageSuspense><Pages.Reports /></PageSuspense>} />
                  </Route>

                  <Route element={<RoleProtectedRoute allowedRoles={['admin', 'support', 'owner']} />}>
                    <Route path="/analytics" element={<PageSuspense><Pages.Analytics /></PageSuspense>} />
                  </Route>

                  <Route element={<RoleProtectedRoute allowedRoles={['admin', 'support']} />}>
                    <Route path="/logs" element={<PageSuspense><Pages.Logs /></PageSuspense>} />
                    <Route path="/audit-log" element={<PageSuspense><Pages.AuditLog /></PageSuspense>} />
                    <Route path="/blackbox" element={<PageSuspense><Pages.BlackBox /></PageSuspense>} />
                    <Route path="/advanced-analytics" element={<PageSuspense><Pages.AdvancedAnalytics /></PageSuspense>} />
                    <Route path="/events" element={<PageSuspense><Pages.EventReplay /></PageSuspense>} />
                  </Route>

                  {/* Admin Only Routes */}
                  <Route element={<RoleProtectedRoute allowedRoles={['admin']} />}>
                    <Route path="/users" element={<PageSuspense><Pages.Users /></PageSuspense>} />
                    <Route path="/api-keys" element={<PageSuspense><Pages.APIKeys /></PageSuspense>} />
                    <Route path="/webhooks" element={<PageSuspense><Pages.Webhooks /></PageSuspense>} />
                    <Route path="/workload-analytics" element={<PageSuspense><Pages.WorkloadAnalytics /></PageSuspense>} />
                    <Route path="/executive-dashboard" element={<Navigate to="/dashboard" replace />} />
                  </Route>

                  {/* Admin Only Routes - Settings */}
                  <Route element={<RoleProtectedRoute allowedRoles={['admin']} />}>
                    <Route path="/settings/:tab?" element={<PageSuspense><Pages.Settings /></PageSuspense>} />
                  </Route>

                  {/* Profile Route - Accessible to all authenticated users */}
                  <Route path="/profile" element={<PageSuspense><Pages.Profile /></PageSuspense>} />

                  {/* CMMS Routes */}
                  <Route element={<RoleProtectedRoute allowedRoles={['admin', 'manager', 'technician']} />}>
                    <Route path="/maintenance" element={<PageSuspense><Pages.MaintenanceSchedules /></PageSuspense>} />
                    <Route path="/work-orders" element={<PageSuspense><Pages.WorkOrders /></PageSuspense>} />
                    <Route path="/work-orders/:id" element={<PageSuspense><Pages.WorkOrderDetail /></PageSuspense>} />
                    <Route path="/spare-parts" element={<PageSuspense><Pages.SpareParts /></PageSuspense>} />
                    <Route path="/technician-dashboard" element={<Navigate to="/dashboard" replace />} />
                    <Route path="/technician-week" element={<PageSuspense><Pages.TechnicianWeek /></PageSuspense>} />
                  </Route>

                  <Route element={<RoleProtectedRoute allowedRoles={['admin', 'manager']} />}>
                    <Route path="/manager-dashboard" element={<Navigate to="/dashboard" replace />} />
                    <Route path="/asset-overview" element={<PageSuspense><Pages.AssetOverview /></PageSuspense>} />
                    <Route path="/sla" element={<PageSuspense><Pages.SLADashboard /></PageSuspense>} />
                    <Route path="/maintenance-reports" element={<PageSuspense><Pages.MaintenanceReports /></PageSuspense>} />
                    <Route path="/meter-dashboard" element={<PageSuspense><Pages.MeterDashboard /></PageSuspense>} />
                    <Route path="/wo-aging" element={<PageSuspense><Pages.WOAging /></PageSuspense>} />
                    <Route path="/location-tree" element={<PageSuspense><Pages.LocationTree /></PageSuspense>} />
                    <Route path="/cost-dashboard" element={<PageSuspense><Pages.TotalCostDashboard /></PageSuspense>} />
                    <Route path="/vendor-performance" element={<PageSuspense><Pages.VendorPerformance /></PageSuspense>} />
                    <Route path="/on-call" element={<PageSuspense><Pages.OnCallSchedule /></PageSuspense>} />
                    <Route path="/compliance-shield" element={<PageSuspense><Pages.ComplianceShield /></PageSuspense>} />
                    <Route path="/predictive-maintenance" element={<PageSuspense><Pages.PredictiveMaintenance /></PageSuspense>} />
                  </Route>

                  {/* Tutorials — all roles */}
                  <Route path="/tutorials" element={<PageSuspense><Pages.Tutorials /></PageSuspense>} />
                  <Route path="/glossary" element={<PageSuspense><Pages.Glossary /></PageSuspense>} />
                </Route>

                {/* Default Redirect */}
                <Route path="/" element={<Navigate to="/dashboard" replace />} />
                <Route path="*" element={<Navigate to="/dashboard" replace />} />
              </Routes>
            </BrowserRouter>
          </SentryUserSync>
        </AuthProvider>
        </ToastProvider>
      </ThemeProvider>
    </QueryClientProvider>
    </SentryErrorBoundary>
  );
}

export default App;
