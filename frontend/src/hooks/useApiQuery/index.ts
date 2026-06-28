// ═══════════════════════════════════════════════════════════════════════
// useApiQuery — barrel export
// Все хуки React Query, сгруппированные по доменам.
//
// Поддерживает обратную совместимость: все экспорты доступны
// через единый импорт `from '../../hooks/useApiQuery'`
// ═══════════════════════════════════════════════════════════════════════

export { queryKeys, CACHE } from './shared';
export type {
  Device, Site, Ticket, Alarm, AppNotification,
  User, DashboardStats, Report, AuditLogEntry,
  ServicesSettings, WorkOrder,
  MaintenanceSchedule, CreateScheduleRequest,
  SparePart, CreateSparePartRequest, SparePartCategory,
  Prediction,
} from './shared';

export {
  useDevices, useDevice,
  useCreateDevice, useUpdateDevice, useDeleteDevice,
  useSites, useSite,
  useCreateSite, useUpdateSite, useDeleteSite,
  useTickets, useTicket,
  useCreateTicket, useUpdateTicket, useDeleteTicket,
  useAlarms, useAcknowledgeAlarm, useResolveAlarm,
  prefetchDevice,
} from './devices';

export {
  useWorkOrders,
  useCreateWorkOrder, useUpdateWorkOrder, useDeleteWorkOrder,
  useDashboardStats,
  useReports,
  useMaintenanceSchedules,
  useCreateMaintenanceSchedule, useUpdateMaintenanceSchedule,
  useDeleteMaintenanceSchedule, useCompleteMaintenanceSchedule,
  useSpareParts, useLowStockParts,
  useSparePartCategories,
  useCreateSparePart, useUpdateSparePart, useDeleteSparePart,
  useAdjustStock,
  useCreateSparePartCategory, useUpdateSparePartCategory, useDeleteSparePartCategory,
  usePredictions,
  prefetchWorkOrder,
} from './workOrders';

export {
  useUsers, useCurrentUser,
  useCreateUser, useUpdateUser, useDeleteUser,
  useNotifications,
  useMarkNotificationRead, useMarkAllNotificationsRead, useDeleteNotification,
  useServicesSettings, useServicesStatus, useUpdateServicesSettings,
  useAuditLog,
} from './users';
