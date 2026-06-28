// ═══════════════════════════════════════════════════════════════════════
// useApiQuery — barrel re-export
//
// После рефакторинга все хуки распределены по доменным файлам:
//   hooks/useApiQuery/shared.ts     — queryKeys, CACHE, типы
//   hooks/useApiQuery/devices.ts    — devices, sites, tickets, alarms
//   hooks/useApiQuery/workOrders.ts — workOrders, maintenance, spareParts, dashboard, reports, predictions
//   hooks/useApiQuery/users.ts      — users, notifications, services, auditLog
//   hooks/useApiQuery/index.ts      — barrel export
//
// Этот файл сохранён для обратной совместимости.
// Новый код должен импортировать напрямую из 'hooks/useApiQuery'.
// ═══════════════════════════════════════════════════════════════════════

export * from './useApiQuery/index';
