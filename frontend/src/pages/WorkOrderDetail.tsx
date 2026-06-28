// ═══════════════════════════════════════════════════════════════════════
// WorkOrderDetail — barrel re-export
//
// После рефакторинга компонент разбит на части в pages/WorkOrderDetail/:
//   WorkOrderDetail/WorkOrderDetail.tsx  — основной компонент (~300 строк)
//   WorkOrderDetail/WOStatusCards.tsx    — статус-карточки
//   WorkOrderDetail/WOPartsSection.tsx   — секция запчастей
//   WorkOrderDetail/WOTimeTracking.tsx   — учёт времени
//   WorkOrderDetail/WOModals.tsx         — модальные окна
//
// Этот файл сохранён для обратной совместимости.
// ═══════════════════════════════════════════════════════════════════════

export { WorkOrderDetail } from './WorkOrderDetail/WorkOrderDetail';
