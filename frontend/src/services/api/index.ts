// ═══════════════════════════════════════════════════════════════════════
// API Services barrel export
// ARCH.2: Модульная структура вместо monolithic api.ts.
//
// Каждый модуль можно импортировать отдельно:
//   import { devicesApi } from '../services/api/devices';
//   import { request } from '../services/api/client';
//
// Для backward compat — комбинированный объект `api`.
// ═══════════════════════════════════════════════════════════════════════

// Client
export { request, requestBlob, setAuthToken, getAuthToken, handleApiError, API_BASE } from './client';
export type { MappedApiError } from '../apiErrorMapper';

// Devices
export { devicesApi } from './devices';
export type {
  Device,
  DeviceDetectionResult,
  CapacityParams,
  CapacityResult,
  DashboardStats,
} from './devices';

// Alarms
export { alarmsApi } from './alarms';
export type { Alarm } from './alarms';

// Users & Auth
export { authApi, usersApi } from './users';
export type { User, TechnicianSiteAssignment } from './users';

// Sites & Tickets
export { sitesApi, ticketsApi } from './sites';
export type { Site, Ticket, TicketComment } from './sites';

// Reports, Notifications & Audit Log
export { reportsApi, notificationsApi, auditLogApi } from './reports';
export type { Report, Notification, AuditLogEntry } from './reports';

// Integrations (Webhooks, P2P, Atlas, Camera Models)
export { webhooksApi, p2pApi, atlasApi, cameraModelsApi } from './integrations';
export type { WebhookEndpoint, CameraSpec, CameraBrand, CameraModelSummary } from './integrations';

// Analytics
export { predictionsApi, costApi, reliabilityApi, slaApi, logsApi, biApi } from './analytics';
export type {
  Prediction, CostData, CostTrend, TopExpensiveDevice,
  VendorReliability, SLAMetrics, ReliabilityData, ParsedLog,
  QueryTemplate, QueryParams, QueryResult, Field, FilterCondition,
} from './analytics';

// P2-AI.4: Anomaly Detection
export { anomaliesApi } from './anomalies';
export type {
  AnomalyResult, AnomalyListResponse, AnomalyStats,
  FeedMetricRequest, FeedMetricResponse,
} from './anomalies';

// Services Settings
export { servicesApi } from './services';
export type {
  ServicesSettings, SyslogSettings, FTPSettings, SNMPSettings,
  GB28181Settings, P2PGatewaySettings,
} from './services';

// RCA Graph
export { rcaApi } from './rca';
export type { RCAGraphResponse, RCAGraphNode, RCAGraphEdge } from './rca';

// Work Orders
export { workOrdersApi } from './workOrders';
export type {
  WorkOrder, ChecklistItem, PartUsage, CreateWorkOrderRequest, TimeEntry,
  AnnotationSaveRequest, AnnotationResponse,
} from './workOrders';

// Workflows
export {
  getWorkflows,
  getWorkflow,
  createWorkflow,
  updateWorkflow,
  deleteWorkflow,
} from './workflows';
export type { WorkflowDefinition, WorkflowNode, WorkflowEdge, WorkflowListResponse } from './workflows';

// P1-MARKET: Playbook Marketplace
export { playbookMarketplaceApi } from './playbookMarketplace';
export type { MarketplacePlaybook, MarketplaceListResponse, MarketplaceFilter } from './playbookMarketplace';

// P2-FIELDS: Custom Fields
export { customFieldsApi } from './customFields';

// P2-API: API Versioning
export { versionsApi } from './versions';
export type {
  VersionInfo,
  ChangelogEntry,
  VersionListResponse,
  ChangelogResponse,
} from './versions';
export type {
  FieldDefinition, FieldGroup, FieldDefinitionWithValue,
  FieldType, EntityType, ValidationRule, FieldCondition,
  CreateFieldDefinitionRequest, UpdateFieldDefinitionRequest,
  CreateGroupRequest, BulkUpdateValuesRequest,
} from './customFields';

// Agents — EDGE-11
export { agentsApi } from './agents';

// PROTO-06: Protocol Descriptors
export { descriptorsApi } from './descriptors';

// PROTO-07: Community Protocol Registry
export { communityRegistryApi } from './communityRegistry';
export type {
  CommunityDescriptorSummary,
  CommunityDescriptor,
  CommunityDescriptorListResponse,
  CommunityDescriptorFilter,
  PublishDescriptorRequest,
} from './communityRegistry';

// UX-2.4: Secure Tunnel
export { tunnelApi } from './tunnel';
export type {
  TunnelTokenResponse,
  TunnelStatus,
  TunnelLogEntry,
  TunnelProtocol,
} from './tunnel';

// ═══════════════════════════════════════════════════════════════════════
// Комбинированный объект `api` для backward compat
// ═══════════════════════════════════════════════════════════════════════

import { devicesApi } from './devices';
import { versionsApi } from './versions';
import { alarmsApi } from './alarms';
import { authApi, usersApi } from './users';
import { sitesApi, ticketsApi } from './sites';
import { reportsApi, notificationsApi, auditLogApi } from './reports';
import { webhooksApi, p2pApi, atlasApi, cameraModelsApi } from './integrations';
import { predictionsApi, costApi, reliabilityApi, slaApi, logsApi, biApi } from './analytics';
import { anomaliesApi } from './anomalies';
import { customFieldsApi } from './customFields';
import { servicesApi } from './services';
import { rcaApi } from './rca';
import { agentsApi } from './agents';

export const api = {
  // Auth
  login: authApi.login.bind(authApi),
  getCurrentUser: authApi.getCurrentUser.bind(authApi),
  logout: authApi.logout.bind(authApi),
  login2FA: authApi.login2FA.bind(authApi),

  // Devices
  getDevices: devicesApi.getDevices.bind(devicesApi),
  getDevice: devicesApi.getDevice.bind(devicesApi),
  getDeviceStatus: devicesApi.getDeviceStatus.bind(devicesApi),
  createDevice: devicesApi.createDevice.bind(devicesApi),
  updateDevice: devicesApi.updateDevice.bind(devicesApi),
  deleteDevice: devicesApi.deleteDevice.bind(devicesApi),
  getDeviceImages: devicesApi.getDeviceImages.bind(devicesApi),
  detectDevice: devicesApi.detectDevice.bind(devicesApi),
  calculateDeviceCapacity: devicesApi.calculateDeviceCapacity.bind(devicesApi),

  // Dashboard
  getDashboardStats: devicesApi.getDashboardStats.bind(devicesApi),

  // Alarms
  getAlarms: alarmsApi.getAlarms.bind(alarmsApi),
  acknowledgeAlarm: alarmsApi.acknowledgeAlarm.bind(alarmsApi),
  resolveAlarm: alarmsApi.resolveAlarm.bind(alarmsApi),
  deleteAlarm: alarmsApi.deleteAlarm.bind(alarmsApi),

  // Analytics / Predictions
  getPredictions: predictionsApi.getPredictions.bind(predictionsApi),
  triggerPredictionRun: predictionsApi.triggerRun.bind(predictionsApi),

  // Cost Analysis
  getCostData: costApi.getCostData.bind(costApi),
  getCostTrend: costApi.getCostTrend.bind(costApi),
  getTopExpensiveDevices: costApi.getTopExpensiveDevices.bind(costApi),

  // Reliability
  getReliabilityData: reliabilityApi.getData.bind(reliabilityApi),

  // SLA
  getSLAMetrics: slaApi.getMetrics.bind(slaApi),

  // Logs
  searchLogs: logsApi.search.bind(logsApi),

  // Sites
  getSites: sitesApi.getSites.bind(sitesApi),
  getSite: sitesApi.getSite.bind(sitesApi),
  createSite: sitesApi.createSite.bind(sitesApi),
  updateSite: sitesApi.updateSite.bind(sitesApi),
  deleteSite: sitesApi.deleteSite.bind(sitesApi),

  // Tickets
  getTickets: ticketsApi.getTickets.bind(ticketsApi),
  getTicket: ticketsApi.getTicket.bind(ticketsApi),
  createTicket: ticketsApi.createTicket.bind(ticketsApi),
  updateTicket: ticketsApi.updateTicket.bind(ticketsApi),
  deleteTicket: ticketsApi.deleteTicket.bind(ticketsApi),
  addTicketComment: ticketsApi.addTicketComment.bind(ticketsApi),

  // Users
  getUsers: usersApi.getUsers.bind(usersApi),
  getUser: usersApi.getUser.bind(usersApi),
  createUser: usersApi.createUser.bind(usersApi),
  updateUser: usersApi.updateUser.bind(usersApi),
  deleteUser: usersApi.deleteUser.bind(usersApi),
  changePassword: usersApi.changePassword.bind(usersApi),
  resetUserPassword: usersApi.resetUserPassword.bind(usersApi),

  // Sessions
  getSessions: usersApi.getSessions.bind(usersApi),
  revokeSession: usersApi.revokeSession.bind(usersApi),
  revokeAllOtherSessions: usersApi.revokeAllOtherSessions.bind(usersApi),

  // 2FA
  setup2FA: usersApi.setup2FA.bind(usersApi),
  verify2FA: usersApi.verify2FA.bind(usersApi),
  disable2FA: usersApi.disable2FA.bind(usersApi),

  // Notifications
  getNotifications: notificationsApi.getNotifications.bind(notificationsApi),
  markNotificationRead: notificationsApi.markNotificationRead.bind(notificationsApi),
  markAllNotificationsRead: notificationsApi.markAllNotificationsRead.bind(notificationsApi),
  deleteNotification: notificationsApi.deleteNotification.bind(notificationsApi),
  deleteNotifications: notificationsApi.deleteNotifications.bind(notificationsApi),

  // Reports
  getReports: reportsApi.getReports.bind(reportsApi),
  generateReport: reportsApi.generateReport.bind(reportsApi),
  getReportFile: reportsApi.getReportFile.bind(reportsApi),
  deleteReport: reportsApi.deleteReport.bind(reportsApi),

  // Audit Log
  getAuditLog: auditLogApi.getAuditLog.bind(auditLogApi),

  // Services Settings
  getServicesSettings: servicesApi.getSettings.bind(servicesApi),
  updateServicesSettings: servicesApi.updateSettings.bind(servicesApi),
  getServicesStatus: servicesApi.getStatus.bind(servicesApi),

  // P2P
  listP2PDevices: p2pApi.listP2PDevices.bind(p2pApi),
  registerP2PDevice: p2pApi.registerP2PDevice.bind(p2pApi),
  getP2PDeviceStatus: p2pApi.getP2PDeviceStatus.bind(p2pApi),
  sendP2PCommand: p2pApi.sendP2PCommand.bind(p2pApi),
  getP2PSnapshot: p2pApi.getP2PSnapshot.bind(p2pApi),

  // External Alarms
  sendExternalAlarm: alarmsApi.sendExternalAlarm.bind(alarmsApi),

  // API Keys
  getAPIKeys: usersApi.getAPIKeys.bind(usersApi),
  createAPIKey: usersApi.createAPIKey.bind(usersApi),
  revokeAPIKey: usersApi.revokeAPIKey.bind(usersApi),

  // Telegram
  generateTelegramLink: usersApi.generateTelegramLink.bind(usersApi),
  getTelegramStatus: usersApi.getTelegramStatus.bind(usersApi),
  updateTelegramSettings: usersApi.updateTelegramSettings.bind(usersApi),
  requestTelegramLoginCode: usersApi.requestTelegramLoginCode.bind(usersApi),
  verifyTelegramLogin: usersApi.verifyTelegramLogin.bind(usersApi),

  // Technician Site Assignments
  getTechnicianSiteAssignments: usersApi.getTechnicianSiteAssignments.bind(usersApi),
  createTechnicianSiteAssignment: usersApi.createTechnicianSiteAssignment.bind(usersApi),
  updateTechnicianSiteAssignment: usersApi.updateTechnicianSiteAssignment.bind(usersApi),
  deleteTechnicianSiteAssignment: usersApi.deleteTechnicianSiteAssignment.bind(usersApi),

  // Atlas CMMS
  atlasHealthCheck: atlasApi.healthCheck.bind(atlasApi),
  atlasFallbackStatus: atlasApi.fallbackStatus.bind(atlasApi),
  atlasRetryFallback: atlasApi.retryFallback.bind(atlasApi),
  atlasSyncAsset: atlasApi.syncAsset.bind(atlasApi),

  // Webhooks
  getWebhooks: webhooksApi.getWebhooks.bind(webhooksApi),
  createWebhook: webhooksApi.createWebhook.bind(webhooksApi),
  updateWebhook: webhooksApi.updateWebhook.bind(webhooksApi),
  deleteWebhook: webhooksApi.deleteWebhook.bind(webhooksApi),
  testWebhook: webhooksApi.testWebhook.bind(webhooksApi),

  // RCA Graph
  getRCAGraph: rcaApi.getGraph.bind(rcaApi),

  // Camera Models
  listCameraBrands: cameraModelsApi.listBrands.bind(cameraModelsApi),
  listCameraModels: cameraModelsApi.listModels.bind(cameraModelsApi),
  searchCameraModels: cameraModelsApi.searchModels.bind(cameraModelsApi),
  getCameraSpecs: cameraModelsApi.getSpecs.bind(cameraModelsApi),
  importCameraSpecs: cameraModelsApi.importSpecs.bind(cameraModelsApi),
  seedCameraSpecs: cameraModelsApi.seedSpecs.bind(cameraModelsApi),

  // P2-AI.4: Anomaly Detection
  getAnomalies: anomaliesApi.getAnomalies.bind(anomaliesApi),
  feedMetric: anomaliesApi.feedMetric.bind(anomaliesApi),
  acknowledgeAnomaly: anomaliesApi.acknowledgeAnomaly.bind(anomaliesApi),
  resolveAnomaly: anomaliesApi.resolveAnomaly.bind(anomaliesApi),
  getAnomalyStats: anomaliesApi.getStats.bind(anomaliesApi),

  // P2-API: API Versioning
  listVersions: versionsApi.listVersions.bind(versionsApi),
  createVersion: versionsApi.createVersion.bind(versionsApi),
  updateVersion: versionsApi.updateVersion.bind(versionsApi),
  getChangelog: versionsApi.getChangelog.bind(versionsApi),
 
  // P2-FIELDS: Custom Fields
  getCustomFieldDefinitions: customFieldsApi.getDefinitions.bind(customFieldsApi),
  getCustomFieldDefinition: customFieldsApi.getDefinition.bind(customFieldsApi),
  createCustomFieldDefinition: customFieldsApi.createDefinition.bind(customFieldsApi),
  updateCustomFieldDefinition: customFieldsApi.updateDefinition.bind(customFieldsApi),
  deleteCustomFieldDefinition: customFieldsApi.deleteDefinition.bind(customFieldsApi),
  getCustomFieldGroups: customFieldsApi.getGroups.bind(customFieldsApi),
  getCustomFieldGroup: customFieldsApi.getGroup.bind(customFieldsApi),
  createCustomFieldGroup: customFieldsApi.createGroup.bind(customFieldsApi),
  updateCustomFieldGroup: customFieldsApi.updateGroup.bind(customFieldsApi),
  deleteCustomFieldGroup: customFieldsApi.deleteGroup.bind(customFieldsApi),
  getCustomFieldValues: customFieldsApi.getFieldValues.bind(customFieldsApi),
  bulkUpdateCustomFieldValues: customFieldsApi.bulkUpdateValues.bind(customFieldsApi),

  // Agents — EDGE-11
  getAgents: agentsApi.getAgents.bind(agentsApi),
  getAgent: agentsApi.getAgent.bind(agentsApi),
  sendAgentCommand: agentsApi.sendCommand.bind(agentsApi),
  deleteAgent: agentsApi.deleteAgent.bind(agentsApi),

  // P2-BI: Self-Service Analytics
  getBITemplates: biApi.getTemplates.bind(biApi),
  executeBIQuery: biApi.executeQuery.bind(biApi),
};
