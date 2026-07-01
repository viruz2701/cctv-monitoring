// ═══════════════════════════════════════════════════════════════════════
// AppShell — Lazy-loaded application shell (P0-CR-06)
//
// Содержит Layout и все роуты, что позволяет Rolldown выделить их
// в отдельный chunk и уменьшить main bundle.
//
// Compliance:
//   - OWASP ASVS V2.1.1 (Input validation — через Zod в дочерних компонентах)
//   - IEC 62443 SR 3.1 (RBAC — через RoleProtectedRoute)
// ═══════════════════════════════════════════════════════════════════════

import { useEffect, type ReactNode, lazy } from 'react';
import { BrowserRouter, Routes, Route, Navigate, useLocation } from 'react-router-dom';
import { Layout, PageSuspense } from './layout';
import { useAuth } from '../hooks/useAuth';
import { setSentryUser } from '../lib/sentry';
import { isFeatureEnabled } from '../config/featureFlags';

// ── Lazy-loaded pages (P0-CR-06: Route-based code splitting) ────────────
// Каждая страница — отдельный chunk для минимизации main bundle
const Login = lazy(() => import('../pages/Login').then((m) => ({ default: m.Login })));
const ForgotPassword = lazy(() => import('../pages/ForgotPassword').then((m) => ({ default: m.ForgotPassword })));
const WorkRequestPortal = lazy(() => import('../pages/WorkRequestPortal').then((m) => ({ default: m.WorkRequestPortal })));
const SecurityAdvisories = lazy(() => import('../pages/SecurityAdvisories').then((m) => ({ default: m.SecurityAdvisories })));
const SetupWizard = lazy(() => import('../pages/SetupWizard').then((m) => ({ default: m.SetupWizard })));
const DashboardHub = lazy(() => import('../pages/DashboardHub').then((m) => ({ default: m.DashboardHub })));
const Sites = lazy(() => import('../pages/Sites').then((m) => ({ default: m.Sites })));
const SiteDetail = lazy(() => import('../pages/SiteDetail').then((m) => ({ default: m.SiteDetail })));
const DeviceDetail = lazy(() => import('../pages/DeviceDetail').then((m) => ({ default: m.DeviceDetail })));
const Devices = lazy(() => import('../pages/Devices').then((m) => ({ default: m.Devices })));
const AgentDashboard = lazy(() => import('../pages/AgentDashboard').then((m) => ({ default: m.AgentDashboard })));
const AgentDetail = lazy(() => import('../pages/AgentDetail').then((m) => ({ default: m.AgentDetail })));
const Tickets = lazy(() => import('../pages/Tickets').then((m) => ({ default: m.Tickets })));
const TicketDetail = lazy(() => import('../pages/TicketDetail').then((m) => ({ default: m.TicketDetail })));
const Alerts = lazy(() => import('../pages/Alerts').then((m) => ({ default: m.Alerts })));
const Notifications = lazy(() => import('../pages/Notifications').then((m) => ({ default: m.Notifications })));
const Reports = lazy(() => import('../pages/Reports').then((m) => ({ default: m.Reports })));
const Analytics = lazy(() => import('../pages/Analytics').then((m) => ({ default: m.Analytics })));
const Logs = lazy(() => import('../pages/Logs').then((m) => ({ default: m.Logs })));
const AuditLog = lazy(() => import('../pages/AuditLog').then((m) => ({ default: m.AuditLog })));
const BlackBox = lazy(() => import('../pages/BlackBox').then((m) => ({ default: m.BlackBox })));
const AdvancedAnalytics = lazy(() => import('../pages/AdvancedAnalytics').then((m) => ({ default: m.AdvancedAnalytics })));
const EventReplay = lazy(() => import('../pages/EventReplay').then((m) => ({ default: m.EventReplay })));
const Users = lazy(() => import('../pages/Users').then((m) => ({ default: m.Users })));
const APIKeys = lazy(() => import('../pages/APIKeys').then((m) => ({ default: m.APIKeys })));
const Webhooks = lazy(() => import('../pages/Webhooks').then((m) => ({ default: m.Webhooks })));
const WorkloadAnalytics = lazy(() => import('../pages/WorkloadAnalytics').then((m) => ({ default: m.WorkloadAnalytics })));
const APIVersioning = lazy(() => import('../pages/APIVersioning').then((m) => ({ default: m.APIVersioning })));
const DescriptorEditor = lazy(() => import('../pages/DescriptorEditor').then((m) => ({ default: m.DescriptorEditor })));
const BIQueryBuilder = lazy(() => import('../pages/BIQueryBuilder').then((m) => ({ default: m.BIQueryBuilder })));
const Settings = lazy(() => import('../pages/Settings').then((m) => ({ default: m.Settings })));
const Profile = lazy(() => import('../pages/Profile').then((m) => ({ default: m.Profile })));
const MaintenanceSchedules = lazy(() => import('../pages/MaintenanceSchedules').then((m) => ({ default: m.MaintenanceSchedules })));
const WorkOrders = lazy(() => import('../pages/WorkOrders').then((m) => ({ default: m.WorkOrders })));
const WorkOrderDetail = lazy(() => import('../pages/WorkOrderDetail/WorkOrderDetail').then((m) => ({ default: m.WorkOrderDetail })));
const UnifiedWorkHub = lazy(() => import('../pages/UnifiedWorkHub').then((m) => ({ default: m.UnifiedWorkHub })));
const SpareParts = lazy(() => import('../pages/SpareParts').then((m) => ({ default: m.SpareParts })));
const TechnicianWeek = lazy(() => import('../pages/TechnicianWeek').then((m) => ({ default: m.TechnicianWeek })));
const AssetOverview = lazy(() => import('../pages/AssetOverview').then((m) => ({ default: m.AssetOverview })));
const SLADashboard = lazy(() => import('../pages/SLADashboard').then((m) => ({ default: m.SLADashboard })));
const MaintenanceReports = lazy(() => import('../pages/MaintenanceReports').then((m) => ({ default: m.MaintenanceReports })));
const MeterDashboard = lazy(() => import('../pages/MeterDashboard').then((m) => ({ default: m.MeterDashboard })));
const WOAging = lazy(() => import('../pages/WOAging').then((m) => ({ default: m.WOAging })));
const LocationTree = lazy(() => import('../pages/LocationTree').then((m) => ({ default: m.LocationTree })));
const TotalCostDashboard = lazy(() => import('../pages/TotalCostDashboard').then((m) => ({ default: m.TotalCostDashboard })));
const VendorPerformance = lazy(() => import('../pages/VendorPerformance').then((m) => ({ default: m.VendorPerformance })));
const OnCallSchedule = lazy(() => import('../pages/OnCallSchedule').then((m) => ({ default: m.OnCallSchedule })));
const ComplianceShield = lazy(() => import('../pages/ComplianceShield').then((m) => ({ default: m.ComplianceShield })));
const PredictiveMaintenance = lazy(() => import('../pages/PredictiveMaintenance').then((m) => ({ default: m.PredictiveMaintenance })));
const PlaybookMarketplace = lazy(() => import('../pages/PlaybookMarketplace').then((m) => ({ default: m.PlaybookMarketplace })));
const Tutorials = lazy(() => import('../pages/Tutorials').then((m) => ({ default: m.Tutorials })));
const Glossary = lazy(() => import('../pages/Glossary').then((m) => ({ default: m.Glossary })));

import { RoleProtectedRoute } from './auth/RoleProtectedRoute';

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

function AppRoutes() {
  return (
    <Routes>
      {/* Public Routes */}
      <Route path="/login" element={<Login />} />
      <Route path="/forgot-password" element={<ForgotPassword />} />
      <Route path="/request" element={<WorkRequestPortal />} />
      <Route path="/security-advisories" element={<PageSuspense><SecurityAdvisories /></PageSuspense>} />

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

        {/* EDGE-11: Agent Monitoring Dashboard */}
        <Route element={<RoleProtectedRoute allowedRoles={['admin', 'support', 'manager']} />}>
          <Route path="/agents" element={<PageSuspense><AgentDashboard /></PageSuspense>} />
          <Route path="/agents/:id" element={<PageSuspense><AgentDetail /></PageSuspense>} />
        </Route>

        {/* UX-1.2: Unified Work Hub — conditional override */}
        {isFeatureEnabled('unified_work_hub_v2') ? (
          <Route path="/tickets" element={<Navigate to="/hub?tab=requests" replace />} />
        ) : (
          <Route path="/tickets" element={<PageSuspense><Tickets /></PageSuspense>} />
        )}
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
          <Route path="/events" element={<PageSuspense><EventReplay /></PageSuspense>} />
        </Route>

        {/* Admin Only Routes */}
        <Route element={<RoleProtectedRoute allowedRoles={['admin']} />}>
          <Route path="/users" element={<PageSuspense><Users /></PageSuspense>} />
          <Route path="/api-keys" element={<PageSuspense><APIKeys /></PageSuspense>} />
          <Route path="/webhooks" element={<PageSuspense><Webhooks /></PageSuspense>} />
          <Route path="/workload-analytics" element={<PageSuspense><WorkloadAnalytics /></PageSuspense>} />
          <Route path="/api-versioning" element={<PageSuspense><APIVersioning /></PageSuspense>} />
          <Route path="/executive-dashboard" element={<Navigate to="/dashboard" replace />} />

          {/* PROTO-06: Protocol Descriptor Editor */}
          <Route path="/admin/descriptors" element={<PageSuspense><DescriptorEditor /></PageSuspense>} />
          <Route path="/admin/descriptors/new" element={<PageSuspense><DescriptorEditor /></PageSuspense>} />
          <Route path="/admin/descriptors/:vendor/edit" element={<PageSuspense><DescriptorEditor /></PageSuspense>} />

          {/* P2-BI: Self-Service Analytics Query Builder */}
          <Route path="/bi-query" element={<PageSuspense><BIQueryBuilder /></PageSuspense>} />
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

          {/* UX-1.2: Unified Work Hub — conditional override */}
          {isFeatureEnabled('unified_work_hub_v2') ? (
            <>
              <Route path="/hub" element={<PageSuspense><UnifiedWorkHub /></PageSuspense>} />
              <Route path="/work-orders" element={<Navigate to="/hub?tab=tasks" replace />} />
            </>
          ) : (
            <Route path="/work-orders" element={<PageSuspense><WorkOrders /></PageSuspense>} />
          )}

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
          <Route path="/playbook-marketplace" element={<PageSuspense><PlaybookMarketplace /></PageSuspense>} />
        </Route>

        {/* Tutorials — all roles */}
        <Route path="/tutorials" element={<PageSuspense><Tutorials /></PageSuspense>} />
        <Route path="/glossary" element={<PageSuspense><Glossary /></PageSuspense>} />
      </Route>

      {/* Default Redirect */}
      <Route path="/" element={<Navigate to="/dashboard" replace />} />
      <Route path="*" element={<Navigate to="/dashboard" replace />} />
    </Routes>
  );
}

export default function AppShell() {
  return (
    <SentryUserSync>
      <BrowserRouter>
        <AppRoutes />
      </BrowserRouter>
    </SentryUserSync>
  );
}
