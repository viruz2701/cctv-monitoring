import { lazy, type ReactNode } from 'react';
import { BrowserRouter, Routes, Route, Navigate, useLocation } from 'react-router-dom';
import { QueryClient, QueryClientProvider } from '@tanstack/react-query';
import { Layout, PageSuspense } from './components/layout';
import { Login, DeviceDetail, TicketDetail, Reports, Users, Settings, Profile, Notifications, TotalCostDashboard, ManagerDashboard, AssetOverview } from './pages';

import { ThemeProvider } from './context/ThemeContext';
import { AuthProvider } from './hooks/useAuth';
import { SettingsProvider } from './context/SettingsContext';
import { UsersProvider } from './context/UsersContext';
import { DevicesSitesProvider } from './context/DevicesSitesContext';
import { TicketsProvider } from './context/TicketsContext';
import { AlertsProvider } from './context/AlertsContext';
import { NotificationsProvider } from './context/NotificationsContext';
import { ReportsProvider } from './context/ReportsContext';
import { MaintenanceProvider } from './context/MaintenanceContext';
import { WorkOrdersProvider } from './context/WorkOrdersContext';
import { SparePartsProvider } from './context/SparePartsContext';
import { ToastProvider } from './components/ui';

// ── Lazy-loaded pages ─────────────────────────────────────────────────────
const Dashboard = lazy(() => import('./pages/Dashboard').then((m) => ({ default: m.Dashboard })));
const Sites = lazy(() => import('./pages/Sites').then((m) => ({ default: m.Sites })));
const Devices = lazy(() => import('./pages/Devices').then((m) => ({ default: m.Devices })));
const Tickets = lazy(() => import('./pages/Tickets').then((m) => ({ default: m.Tickets })));
const Alerts = lazy(() => import('./pages/Alerts').then((m) => ({ default: m.Alerts })));
const WorkOrders = lazy(() => import('./pages/WorkOrders').then((m) => ({ default: m.WorkOrders })));
const Analytics = lazy(() => import('./pages/Analytics').then((m) => ({ default: m.Analytics })));

// ── Static page imports ───────────────────────────────────────────────────
import { Logs } from './pages/Logs';
import { AuditLog } from './pages/AuditLog';
import { MeterDashboard } from './pages/MeterDashboard';
import { WOAging } from './pages/WOAging';
import { LocationTree } from './pages/LocationTree';
import { APIKeys } from './pages/APIKeys';
import { Webhooks } from './pages/Webhooks';
import { WorkloadAnalytics } from './pages/WorkloadAnalytics';
import { WorkRequestPortal } from './pages/WorkRequestPortal';
import { VendorPerformance } from './pages/VendorPerformance';
import { OnCallSchedule } from './pages/OnCallSchedule';
import { ExecutiveDashboard } from './pages/ExecutiveDashboard';
import { MaintenanceSchedules } from './pages/MaintenanceSchedules';
import { SpareParts } from './pages/SpareParts';
import { WorkOrderDetail } from './pages/WorkOrderDetail';
import { TechnicianDashboard } from './pages/TechnicianDashboard';
import { SLADashboard } from './pages/SLADashboard';
import { MaintenanceReports } from './pages/MaintenanceReports';
import { ForgotPassword } from './pages/ForgotPassword';
import { ComplianceShield } from './pages/ComplianceShield';
import { PredictiveMaintenance } from './pages/PredictiveMaintenance';
import { Tutorials } from './pages/Tutorials';
import { BlackBox } from './pages/BlackBox';

import { useAuth } from './hooks/useAuth';
import { RoleProtectedRoute } from './components/auth/RoleProtectedRoute';

function ProtectedRoute({ children }: { children: ReactNode }) {
  const { user } = useAuth();
  const location = useLocation();

  if (!user) {
    return <Navigate to="/login" state={{ from: location }} replace />;
  }

  return children;
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
    <QueryClientProvider client={queryClient}>
    <ThemeProvider>
      <ToastProvider>
        {/*
            ToastProvider вынесен на самый верх
            Все контексты ниже (UsersProvider, SettingsProvider и т.д.)
            теперь могут использовать useToast()
        */}
        <AuthProvider>
          <SettingsProvider>
            <UsersProvider>
              <DevicesSitesProvider>
                <TicketsProvider>
                  <AlertsProvider>
                    <NotificationsProvider>
                      <ReportsProvider>
                        <MaintenanceProvider>
                          <WorkOrdersProvider>
                            <SparePartsProvider>
                              <BrowserRouter>
                          <Routes>
                            {/* Public Route */}
                            <Route path="/login" element={<Login />} />
                            <Route path="/forgot-password" element={<ForgotPassword />} />
                            <Route path="/request" element={<WorkRequestPortal />} />

                            {/* Protected Routes with Layout */}
                            <Route element={
                              <ProtectedRoute>
                                <Layout />
                              </ProtectedRoute>
                            }>
                              <Route path="/dashboard" element={<PageSuspense><Dashboard /></PageSuspense>} />
                              <Route path="/sites" element={<PageSuspense><Sites /></PageSuspense>} />
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
                              </Route>

                              {/* Admin Only Routes */}
                              <Route element={<RoleProtectedRoute allowedRoles={['admin']} />}>
                                <Route path="/users" element={<PageSuspense><Users /></PageSuspense>} />
                                <Route path="/api-keys" element={<PageSuspense><APIKeys /></PageSuspense>} />
                                <Route path="/webhooks" element={<PageSuspense><Webhooks /></PageSuspense>} />
                                <Route path="/workload-analytics" element={<PageSuspense><WorkloadAnalytics /></PageSuspense>} />
                                <Route path="/executive-dashboard" element={<PageSuspense><ExecutiveDashboard /></PageSuspense>} />
                              </Route>

                              {/* Admin Only Routes - Settings */}
                              <Route element={<RoleProtectedRoute allowedRoles={['admin']} />}>
                                <Route path="/settings" element={<PageSuspense><Settings /></PageSuspense>} />
                              </Route>

                              {/* Profile Route - Accessible to all authenticated users */}
                              <Route path="/profile" element={<PageSuspense><Profile /></PageSuspense>} />

                              {/* CMMS Routes */}
                              <Route element={<RoleProtectedRoute allowedRoles={['admin', 'manager', 'technician']} />}>
                                <Route path="/maintenance" element={<PageSuspense><MaintenanceSchedules /></PageSuspense>} />
                                <Route path="/work-orders" element={<PageSuspense><WorkOrders /></PageSuspense>} />
                                <Route path="/work-orders/:id" element={<PageSuspense><WorkOrderDetail /></PageSuspense>} />
                                <Route path="/spare-parts" element={<PageSuspense><SpareParts /></PageSuspense>} />
                                <Route path="/technician-dashboard" element={<PageSuspense><TechnicianDashboard /></PageSuspense>} />
                              </Route>

                              <Route element={<RoleProtectedRoute allowedRoles={['admin', 'manager']} />}>
                                <Route path="/manager-dashboard" element={<PageSuspense><ManagerDashboard /></PageSuspense>} />
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
                            </Route>

                            {/* Default Redirect */}
                            <Route path="/" element={<Navigate to="/dashboard" replace />} />
                            <Route path="*" element={<Navigate to="/dashboard" replace />} />
                          </Routes>
                              </BrowserRouter>
                            </SparePartsProvider>
                          </WorkOrdersProvider>
                        </MaintenanceProvider>
                      </ReportsProvider>
                    </NotificationsProvider>
                  </AlertsProvider>
                </TicketsProvider>
              </DevicesSitesProvider>
            </UsersProvider>
          </SettingsProvider>
        </AuthProvider>
        </ToastProvider>
      </ThemeProvider>
    </QueryClientProvider>
  );
}

export default App;
