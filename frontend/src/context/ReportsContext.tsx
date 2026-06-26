// ═══════════════════════════════════════════════════════════════════════
// ReportsContext — Bridge (backward compat)
// ARCH-02: Новый код ДОЛЖЕН импортировать useReportsStore из store/.
//
// Миграция:
//   Было:  import { useReportsStore } from '../context/ReportsContext'
//   Стало: import { useReportsStore } from '../store/reportsStore'
//
// После полной миграции: удалить этот файл.
// ═══════════════════════════════════════════════════════════════════════

export {
  useReportsStore,
  startReportExpirationSweep,
  stopReportExpirationSweep,
} from '../store/reportsStore';

export type { ReportHistoryItem } from '../store/reportsStore';
