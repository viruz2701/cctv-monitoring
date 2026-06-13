import { ReactNode } from 'react';
import { BrowserRouter, Routes, Route, Navigate, useLocation } from 'react-router-dom';
import { Layout } from './components/layout';
import { Login, Dashboard, Sites, Devices, DeviceDetail, Tickets, TicketDetail, Reports, Users, Settings, Alerts, Profile, Notifications } from './pages';

import { ThemeProvider } from './context/ThemeContext';
import { AuthProvider } from './hooks/useAuth';
import { SettingsProvider } from './context/SettingsContext';
import { UsersProvider } from './context/UsersContext';
import { DevicesSitesProvider } from './context/DevicesSitesContext';
import { TicketsProvider } from './context/TicketsContext';
import { AlertsProvider } from './context/AlertsContext';
import { NotificationsProvider } from './context/NotificationsContext';
import { ReportsProvider } from './context/ReportsContext';
import { ToastProvider } from './components/ui';
import { Analytics } from './pages/Analytics';
import { Logs } from './pages/Logs';

import { useAuth } from './hooks/useAuth';

function ProtectedRoute({ children }: { children: ReactNode }) {
  const { user } = useAuth();
  const location = useLocation();

  if (!user) {
    return <Navigate to="/login" state={{ from: location }} replace />;
  }

  return children;
}

import { RoleProtectedRoute } from './components/auth/RoleProtectedRoute';

function App() {
  return (
    <ThemeProvider>
      <AuthProvider>
        <SettingsProvider>
          <UsersProvider>
            <DevicesSitesProvider>
              <TicketsProvider>
                <AlertsProvider>
                  <NotificationsProvider>
                    <ReportsProvider>
                      <BrowserRouter>
                        <ToastProvider>
                          <Routes>
                            {/* Public Route */}
                            <Route path="/login" element={<Login />} />

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
                              </Route>

                              {/* Admin & Manager Routes */}
                              <Route element={<RoleProtectedRoute allowedRoles={['admin', 'manager']} />}>
                                <Route path="/settings" element={<Settings />} />
                              </Route>

                              {/* Profile Route - Accessible to all authenticated users */}
                              <Route path="/profile" element={<Profile />} />
                            </Route>

                            {/* Default Redirect */}
                            <Route path="/" element={<Navigate to="/dashboard" replace />} />
                            <Route path="*" element={<Navigate to="/dashboard" replace />} />
                          </Routes>
                        </ToastProvider>
                      </BrowserRouter>
                    </ReportsProvider>
                  </NotificationsProvider>
                </AlertsProvider>
              </TicketsProvider>
            </DevicesSitesProvider>
          </UsersProvider>
        </SettingsProvider>
      </AuthProvider>
    </ThemeProvider>
  );
}

export default App;
