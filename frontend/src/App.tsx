import { ReactNode } from 'react';
import { BrowserRouter, Routes, Route, Navigate, useLocation } from 'react-router-dom';
import { Layout } from './components/layout';
import { Login, Dashboard, Sites, Devices, DeviceDetail, Tickets, TicketDetail, Reports, Users, Settings, Alerts, Profile, Notifications, TotalCostDashboard, ManagerDashboard, AssetOverview } from './pages';

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
import { Analytics } from './pages/Analytics';
import { Logs } from './pages/Logs';
import { MaintenanceSchedules } from './pages/MaintenanceSchedules';
import { WorkOrders } from './pages/WorkOrders';
import { SpareParts } from './pages/SpareParts';
import { WorkOrderDetail } from './pages/WorkOrderDetail';
import { TechnicianDashboard } from './pages/TechnicianDashboard';
import { SLADashboard } from './pages/SLADashboard';
import { MaintenanceReports } from './pages/MaintenanceReports';
import { ForgotPassword } from './pages/ForgotPassword';

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

function App() {
  return (
    <ThemeProvider>
      {/* ═══ ToastProvider вынесен на самый верх ═══ 
          Все контексты ниже (UsersProvider, SettingsProvider и т.д.) 
          теперь могут использовать useToast() */}
      <ToastProvider>
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

                            {/* Protected Routes with Layout */}
                            <Route element={
                              <ProtectedRoute>
                                <Layout />
                              </ProtectedRoute>
                            }>
                              <Route path="/dashboard" element={<Dashboard />} />
                              <Route path="/sites" element={<Sites />} />
                              <Route path="/sites/device/:deviceId" element={<DeviceDetail />} />
                              <Route path="/devices" element={<Devices />} />
                              <Route path="/devices/:deviceId" element={<DeviceDetail />} />
                              <Route path="/tickets" element={<Tickets />} />
                              <Route path="/tickets/:ticketId" element={<TicketDetail />} />
                              <Route path="/alerts" element={<Alerts />} />
                              <Route path="/notifications" element={<Notifications />} />
                              
                              <Route element={<RoleProtectedRoute allowedRoles={['admin', 'manager', 'technician']} />}>
                                <Route path="/reports" element={<Reports />} />
                              </Route>
                              
                              <Route element={<RoleProtectedRoute allowedRoles={['admin', 'support', 'owner']} />}>
                                <Route path="/analytics" element={<Analytics />} />
                              </Route>
                              
                              <Route element={<RoleProtectedRoute allowedRoles={['admin', 'support']} />}>
                                <Route path="/logs" element={<Logs />} />
                              </Route>

                              {/* Admin Only Routes */}
                              <Route element={<RoleProtectedRoute allowedRoles={['admin']} />}>
                                <Route path="/users" element={<Users />} />
                                <Route path="/api-keys" element={<Navigate to="/settings" replace />} />
                              </Route>

                              {/* Admin Only Routes - Settings */}
                              <Route element={<RoleProtectedRoute allowedRoles={['admin']} />}>
                                <Route path="/settings" element={<Settings />} />
                              </Route>

                              {/* Profile Route - Accessible to all authenticated users */}
                              <Route path="/profile" element={<Profile />} />

                              {/* CMMS Routes */}
                              <Route element={<RoleProtectedRoute allowedRoles={['admin', 'manager', 'technician']} />}>
                                <Route path="/maintenance" element={<MaintenanceSchedules />} />
                                <Route path="/work-orders" element={<WorkOrders />} />
                                <Route path="/work-orders/:id" element={<WorkOrderDetail />} />
                                <Route path="/spare-parts" element={<SpareParts />} />
                                <Route path="/technician-dashboard" element={<TechnicianDashboard />} />
                              </Route>

                              <Route element={<RoleProtectedRoute allowedRoles={['admin', 'manager']} />}>
                                <Route path="/manager-dashboard" element={<ManagerDashboard />} />
                                <Route path="/asset-overview" element={<AssetOverview />} />
                                <Route path="/sla" element={<SLADashboard />} />
                                <Route path="/maintenance-reports" element={<MaintenanceReports />} />
                                <Route path="/cost-dashboard" element={<TotalCostDashboard />} />
                              </Route>
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
  );
}

export default App;