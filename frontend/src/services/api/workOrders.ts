// ═══════════════════════════════════════════════════════════════════════
// Work Orders API
// ARCH.2: Re-export из services/workOrdersApi для единообразия.
// ═══════════════════════════════════════════════════════════════════════

export { workOrdersApi } from '../workOrdersApi';
export type {
  WorkOrder,
  ChecklistItem,
  PartUsage,
  CreateWorkOrderRequest,
  TimeEntry,
} from '../workOrdersApi';
