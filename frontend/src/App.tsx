import { lazy, useEffect, type ReactNode } from 'react';
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

// ── Lazy-loaded pages ─────────────────────────────────────────────────────
const DashboardHub = lazy(() => import('./pages/DashboardHub').then((m) => ({ default: m.DashboardHub })));
const SetupWizard = lazy(() => import('./pages/SetupWizard').then((m) => ({ default: m.SetupWizard })));
const Sites = lazy(() => import('./pages/Sites').then((m) => ({ default: m.Sites })));
const Devices = lazy(() => import('./pages/Devices').then((m) => ({ default: m.Devices })));
const Tickets = lazy(() => import('./pages/Tickets').then((m) => ({ default: m.Tickets })));
const Alerts = lazy(() => import('./pages/Alerts').then((m) => ({ default: m.Alerts })));
const WorkOrders = lazy(() => import('./pages/WorkOrders').then((m) => ({ default: m.WorkOrders })));
const Analytics = lazy(() => import('./pages/Analytics').then((m) => ({ default: m.Analytics })));
const Logs = lazy(() => import('./pages/Logs').then((m) => ({ default: m.Logs })));
const AuditLog = lazy(() => import('./pages/AuditLog').then((m) => ({ default: m.AuditLog })));
const MeterDashboard = lazy(() => import('./pages/MeterDashboard').then((m) => ({ default: m.MeterDashboard })));
const WOAging = lazy(() => import('./pages/WOAging').then((m) => ({ default: m.WOAging })));
const LocationTree = lazy(() => import('./pages/LocationTree').then((m) => ({ default: m.LocationTree })));
const APIKeys = lazy(() => import('./pages/APIKeys').then((m) => ({ default: m.APIKeys })));
const Webhooks = lazy(() => import('./pages/Webhooks').then((m) => ({ default: m.Webhooks })));
const WorkloadAnalytics = lazy(() => import('./pages/WorkloadAnalytics').then((m) => ({ default: m.WorkloadAnalytics })));
const WorkRequestPortal = lazy(() => import('./pages/WorkRequestPortal').then((m) => ({ default: m.WorkRequestPortal })));
const VendorPerformance = lazy(() => import('./pages/VendorPerformance').then((m) => ({ default: m.VendorPerformance })));
const OnCallSchedule = lazy(() => import('./pages/OnCallSchedule').then((m) => ({ default: m.OnCallSchedule })));
const MaintenanceSchedules = lazy(() => import('./pages/MaintenanceSchedules').then((m) => ({ default: m.MaintenanceSchedules })));
const SpareParts = lazy(() => import('./pages/SpareParts').then((m) => ({ default: m.SpareParts })));
const WorkOrderDetail = lazy(() => import('./pages/WorkOrderDetail').then((m) => ({ default: m.WorkOrderDetail })));
const TechnicianWeek = lazy(() => import('./pages/TechnicianWeek').then((m) => ({ default: m.TechnicianWeek })));
const SLADashboard = lazy(() => import('./pages/SLADashboard').then((m) => ({ default: m.SLADashboard })));
const MaintenanceReports = lazy(() => import('./pages/MaintenanceReports').then((m) => ({ default: m.MaintenanceReports })));
const ForgotPassword = lazy(() => import('./pages/ForgotPassword').then((m) => ({ default: m.ForgotPassword })));
const ComplianceShield = lazy(() => import('./pages/ComplianceShield').then((m) => ({ default: m.ComplianceShield })));
const PredictiveMaintenance = lazy(() => import('./pages/PredictiveMaintenance').then((m) => ({ default: m.PredictiveMaintenance })));
const Tutorials = lazy(() => import('./pages/Tutorials').then((m) => ({ default: m.Tutorials })));
const BlackBox = lazy(() => import('./pages/BlackBox').then((m) => ({ default: m.BlackBox })));
const Login = lazy(() => import('./pages/Login').then((m) => ({ default: m.Login })));
const DeviceDetail = lazy(() => import('./pages/DeviceDetail').then((m) => ({ default: m.DeviceDetail })));
const TicketDetail = lazy(() => import('./pages/TicketDetail').then((m) => ({ default: m.TicketDetail })));
const Reports = lazy(() => import('./pages/Reports').then((m) => ({ default: m.Reports })));
const Users = lazy(() => import('./pages/Users').then((m) => ({ default: m.Users })));
const Settings = lazy(() => import('./pages/Settings').then((m) => ({ default: m.Settings })));
const Profile = lazy(() => import('./pages/Profile').then((m) => ({ default: m.Profile })));
const Notifications = lazy(() => import('./pages/Notifications').then((m) => ({ default: m.Notifications })));
const TotalCostDashboard = lazy(() => import('./pages/TotalCostDashboard').then((m) => ({ default: m.TotalCostDashboard })));
const AssetOverview = lazy(() => import('./pages/AssetOverview').then((m) => ({ default: m.AssetOverview })));
const AdvancedAnalytics = lazy(() => import('./pages/AdvancedAnalytics').then((m) => ({ default: m.AdvancedAnalytics })));
const Glossary = lazy(() => import('./pages/Glossary').then((m) => ({ default: m.Glossary })));
const SiteDetail = lazy(() => import('./pages/SiteDetail').then((m) => ({ default: m.SiteDetail })));

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
                <Route path="/login" element={<Login />} />
                <Route path="/forgot-password" element={<ForgotPassword />} />
                <Route path="/request" element={<WorkRequestPortal />} />

                {/* P0-CE.4: Setup Wizard — публичный, без Layout (On-Premise first-run) */}
                <Route path="/setup" element={
                  <PageSuspense><SetupWizard /></PageSuspense>
                } />

                {/* Protected Routes with Layout */}
                <Route element={
                  <ProtectedRoute>
                    <Layout />
                  </ProtectedRoute>
                }>
                  <Route path="/dashboard" element={<PageSuspense><DashboardHub /></PageSuspense>} />
                  <Route path="/sites" element={<PageSuspense><Sites /></PageSuspense>} />
                  <Route path="/sites/:siteId" element={<PageSuspense><SiteDetail /></PageSuspense>} />
                  <Route path="/sites/device/:deviceId" element={<PageSuspense><DeviceDetail /></PageSuspense>} />
                  <Route path="/devices" element={<PageSuspense><Devices /></PageSuspense>} />
                  <Route path="/devices/:deviceId" element={<PageSuspense><DeviceDetail /></PageSuspense>} />
                  <Route path="/tickets" element={<PageSuspense><Tickets /></PageSuspense>} />
                  <Route path="/tickets/:ticketId" element={<PageSuspense><TicketDetail /></PageSuspense>} />
                  <Route path="/alerts" element={<PageSuspense><Alerts /></PageSuspense>} />
                  <Route path="/notifications" element={<PageSuspense><Notifications /></PageSuspense>} />

                  <Route element={<RoleProtectedRoute allowedRoles={['admin', 'manager', 'technician']} />}>
                    <Route path="/reports" element={<PageSuspense><Reports /></PageSuspense>} />
                  </Route>

                  <Route element={<RoleProtectedRoute allowedRoles={['admin', 'support', 'owner']} />}>
                    <Route path="/analytics" element={<PageSuspense><Analytics /></PageSuspense>} />
                  </Route>

                  <Route element={<RoleProtectedRoute allowedRoles={['admin', 'support']} />}>
                    <Route path="/logs" element={<PageSuspense><Logs /></PageSuspense>} />
                    <Route path="/audit-log" element={<PageSuspense><AuditLog /></PageSuspense>} />
                    <Route path="/blackbox" element={<PageSuspense><BlackBox /></PageSuspense>} />
                    <Route path="/advanced-analytics" element={<PageSuspense><AdvancedAnalytics /></PageSuspense>} />
                  </Route>

                  {/* Admin Only Routes */}
                  <Route element={<RoleProtectedRoute allowedRoles={['admin']} />}>
                    <Route path="/users" element={<PageSuspense><Users /></PageSuspense>} />
                    <Route path="/api-keys" element={<PageSuspense><APIKeys /></PageSuspense>} />
                    <Route path="/webhooks" element={<PageSuspense><Webhooks /></PageSuspense>} />
                    <Route path="/workload-analytics" element={<PageSuspense><WorkloadAnalytics /></PageSuspense>} />
                    <Route path="/executive-dashboard" element={<Navigate to="/dashboard" replace />} />
                  </Route>

                  {/* Admin Only Routes - Settings */}
                  <Route element={<RoleProtectedRoute allowedRoles={['admin']} />}>
                    <Route path="/settings/:tab?" element={<PageSuspense><Settings /></PageSuspense>} />
                  </Route>

                  {/* Profile Route - Accessible to all authenticated users */}
                  <Route path="/profile" element={<PageSuspense><Profile /></PageSuspense>} />

                  {/* CMMS Routes */}
                  <Route element={<RoleProtectedRoute allowedRoles={['admin', 'manager', 'technician']} />}>
                    <Route path="/maintenance" element={<PageSuspense><MaintenanceSchedules /></PageSuspense>} />
                    <Route path="/work-orders" element={<PageSuspense><WorkOrders /></PageSuspense>} />
                    <Route path="/work-orders/:id" element={<PageSuspense><WorkOrderDetail /></PageSuspense>} />
                    <Route path="/spare-parts" element={<PageSuspense><SpareParts /></PageSuspense>} />
                    <Route path="/technician-dashboard" element={<Navigate to="/dashboard" replace />} />
                    <Route path="/technician-week" element={<PageSuspense><TechnicianWeek /></PageSuspense>} />
                  </Route>

                  <Route element={<RoleProtectedRoute allowedRoles={['admin', 'manager']} />}>
                    <Route path="/manager-dashboard" element={<Navigate to="/dashboard" replace />} />
                    <Route path="/asset-overview" element={<PageSuspense><AssetOverview /></PageSuspense>} />
                    <Route path="/sla" element={<PageSuspense><SLADashboard /></PageSuspense>} />
                    <Route path="/maintenance-reports" element={<PageSuspense><MaintenanceReports /></PageSuspense>} />
                    <Route path="/meter-dashboard" element={<PageSuspense><MeterDashboard /></PageSuspense>} />
                    <Route path="/wo-aging" element={<PageSuspense><WOAging /></PageSuspense>} />
                    <Route path="/location-tree" element={<PageSuspense><LocationTree /></PageSuspense>} />
                    <Route path="/cost-dashboard" element={<PageSuspense><TotalCostDashboard /></PageSuspense>} />
                    <Route path="/vendor-performance" element={<PageSuspense><VendorPerformance /></PageSuspense>} />
                    <Route path="/on-call" element={<PageSuspense><OnCallSchedule /></PageSuspense>} />
                    <Route path="/compliance-shield" element={<PageSuspense><ComplianceShield /></PageSuspense>} />
                    <Route path="/predictive-maintenance" element={<PageSuspense><PredictiveMaintenance /></PageSuspense>} />
                  </Route>

                  {/* Tutorials — all roles */}
                  <Route path="/tutorials" element={<PageSuspense><Tutorials /></PageSuspense>} />
                  <Route path="/glossary" element={<PageSuspense><Glossary /></PageSuspense>} />
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
