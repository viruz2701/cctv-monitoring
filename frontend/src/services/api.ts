// ═══════════════════════════════════════════════════════════════════════
// [DEPRECATED] Backward-compat re-export barrel
// ARCH.2: Импортируйте из 'services/api/index' вместо этого файла.
//
// Миграция:
//   Было:  import { api } from '../services/api'
//          import { Device } from '../services/api'
//   Стало: import { api, devicesApi, type Device } from '../services/api'
//          import { useDevices } from '../hooks/useApiQuery'
// ═══════════════════════════════════════════════════════════════════════

export { api, setAuthToken, request, handleApiError, type MappedApiError } from './api/index';

// Re-export types for backward compat
export type { User } from './api/users';
export type { Device, DeviceDetectionResult, CapacityParams, CapacityResult, DashboardStats } from './api/devices';
export type { Alarm } from './api/alarms';
export type { Site, Ticket, TicketComment } from './api/sites';
export type { Report, Notification, AuditLogEntry } from './api/reports';
export type { WebhookEndpoint } from './api/integrations';
export type { CameraSpec, CameraBrand, CameraModelSummary } from './api/integrations';
export type { Prediction, ParsedLog, CostData, CostTrend, TopExpensiveDevice, VendorReliability, SLAMetrics, ReliabilityData } from './api/analytics';
export type { ServicesSettings, SyslogSettings, FTPSettings, SNMPSettings, GB28181Settings, P2PGatewaySettings } from './api/services';
export type { TechnicianSiteAssignment } from './api/users';
