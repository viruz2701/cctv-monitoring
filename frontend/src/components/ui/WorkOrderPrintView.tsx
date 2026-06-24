import React, { useCallback } from 'react';
import { Printer, X, FileText, FileSpreadsheet, Receipt } from 'lucide-react';
import { format } from 'date-fns';
import { useTranslation } from 'react-i18next';
import type { WorkOrder, LaborCost, ChecklistItem, PartUsage } from '../../services/workOrdersApi';

// ── Types ─────────────────────────────────────────────────────────

export type PrintTemplate = 'standard' | 'detailed' | 'invoice';

export interface PartWithCost {
  part_id: string;
  name: string;
  sku?: string;
  quantity: number;
  unit_price: number;
  total_price: number;
}

export interface WorkOrderPrintViewProps {
  workOrder: WorkOrder;
  laborCost?: LaborCost;
  partsDetail?: PartWithCost[];
  onClose?: () => void;
  template?: PrintTemplate; // WO-4.5.2: Standard | Detailed | Invoice
}

// ── Style helpers ──────────────────────────────────────────────────

const statusLabel: Record<string, string> = {
  open: 'workOrder.open',
  in_progress: 'workOrder.inProgress',
  completed: 'workOrder.completed',
  cancelled: 'workOrder.cancelled',
};

const priorityLabel: Record<string, string> = {
  critical: 'workOrder.critical',
  high: 'workOrder.high',
  medium: 'workOrder.medium',
  low: 'workOrder.low',
};

const typeLabel: Record<string, string> = {
  preventive: 'workOrder.preventive',
  corrective: 'workOrder.corrective',
  emergency: 'workOrder.emergency',
};

const slaStatusLabel: Record<string, string> = {
  on_track: 'workOrder.slaOnTrack',
  at_risk: 'workOrder.slaAtRisk',
  breached: 'workOrder.slaBreached',
  completed: 'workOrder.slaCompleted',
  no_sla: 'workOrder.noSla',
};

const slaStatusVariant: Record<string, string> = {
  on_track: 'text-emerald-700 bg-emerald-50',
  at_risk: 'text-amber-700 bg-amber-50',
  breached: 'text-red-700 bg-red-50',
  completed: 'text-blue-700 bg-blue-50',
  no_sla: 'text-slate-500 bg-slate-100',
};

const priorityVariant: Record<string, string> = {
  critical: 'text-red-700 bg-red-50',
  high: 'text-amber-700 bg-amber-50',
  medium: 'text-cyan-700 bg-cyan-50',
  low: 'text-slate-700 bg-slate-100',
};

const statusVariant: Record<string, string> = {
  open: 'text-red-700 bg-red-50',
  in_progress: 'text-blue-700 bg-blue-50',
  completed: 'text-emerald-700 bg-emerald-50',
  cancelled: 'text-slate-500 bg-slate-100',
};

const typeVariant: Record<string, string> = {
  preventive: 'text-blue-700 bg-blue-50',
  corrective: 'text-amber-700 bg-amber-50',
  emergency: 'text-red-700 bg-red-50',
};

// ── Utilities ──────────────────────────────────────────────────────

function formatCurrency(amount: number, currency: string = 'USD'): string {
  return new Intl.NumberFormat('en-US', { style: 'currency', currency }).format(amount);
}

function formatDuration(seconds: number): string {
  const h = Math.floor(seconds / 3600);
  const m = Math.floor((seconds % 3600) / 60);
  if (h > 0) return `${h}h ${m}m`;
  return `${m}m`;
}

function formatDateTime(dateStr?: string): string {
  if (!dateStr) return '—';
  try {
    return format(new Date(dateStr), 'dd.MM.yyyy HH:mm');
  } catch {
    return dateStr;
  }
}

function formatDate(dateStr?: string): string {
  if (!dateStr) return '—';
  try {
    return format(new Date(dateStr), 'dd.MM.yyyy');
  } catch {
    return dateStr;
  }
}

// ── Sub-components ─────────────────────────────────────────────────

interface TemplateMeta { icon: React.FC<{ className?: string }>; labelKey: string; }
const TEMPLATE_META: Record<PrintTemplate, TemplateMeta> = {
  standard: { icon: FileText, labelKey: 'workOrder.templateStandard' },
  detailed: { icon: FileSpreadsheet, labelKey: 'workOrder.templateDetailed' },
  invoice:  { icon: Receipt, labelKey: 'workOrder.templateInvoice' },
};

function InfoRow({ label, value }: { label: string; value: React.ReactNode }) {
  return (
    <div className="flex items-baseline gap-2 py-1.5 border-b border-dashed border-slate-200 last:border-0">
      <span className="text-xs font-medium text-slate-500 uppercase tracking-wider min-w-[140px] shrink-0">{label}</span>
      <span className="text-sm text-slate-900">{value}</span>
    </div>
  );
}

const Section: React.FC<{ title: string; children: React.ReactNode; compact?: boolean }> = ({ title, children, compact }) => (
  <div className={compact ? 'mb-3 print:mb-2' : 'mb-6 print:mb-4'}>
    <h3 className={`font-semibold text-slate-800 uppercase tracking-wider border-b-2 border-slate-300 pb-1.5 mb-3 ${compact ? 'text-[10px]' : 'text-sm'}`}>
      {title}
    </h3>
    {children}
  </div>
);

// ── Template Renderers ─────────────────────────────────────────────

function StandardTemplate({ wo, lc, parts, t, fmtCurrency, fmtDT, fmtD }: {
  wo: WorkOrder; lc: LaborCost | undefined; parts: PartWithCost[] | undefined;
  t: (key: string) => string; fmtCurrency: typeof formatCurrency; fmtDT: typeof formatDateTime; fmtD: typeof formatDate;
}) {
  const totalPartsCost = parts?.reduce((sum, p) => sum + p.total_price, 0) ?? 0;
  const hasParts = wo.parts_used.length > 0;
  const completedChecklist = wo.checklist.filter((c) => c.completed).length;
  const totalChecklist = wo.checklist.length;
  const ts = lc ? { totalHours: lc.total_hours, hourlyRate: lc.hourly_rate, totalCost: lc.total_cost, currency: lc.currency } : null;

  return (
    <>
      {/* Device Info */}
      <Section title={t('workOrder.deviceInfo') || 'Информация об устройстве'}>
        <div className="grid grid-cols-2 gap-x-6">
          <InfoRow label={t('workOrder.deviceName') || 'Устройство'} value={wo.device_name || '—'} />
          <InfoRow label={t('workOrder.deviceId') || 'ID устройства'} value={<code className="text-xs font-mono bg-slate-100 px-1.5 py-0.5 rounded">{wo.device_id}</code>} />
          {wo.assigned_to && <InfoRow label={t('workOrder.assignedTo') || 'Назначено'} value={wo.assignee_name || wo.assigned_to} />}
          {wo.schedule_id && <InfoRow label={t('workOrder.scheduleId') || 'ID расписания'} value={wo.schedule_id} />}
        </div>
      </Section>

      {/* SLA */}
      {wo.sla_deadline && (
        <Section title={t('workOrder.slaInfo') || 'SLA'}>
          <div className="grid grid-cols-2 gap-x-6">
            <InfoRow label={t('workOrder.slaDeadline') || 'SLA дедлайн'} value={fmtDT(wo.sla_deadline)} />
            <InfoRow label={t('workOrder.slaStatus') || 'SLA статус'} value={<span className={`inline-block px-2 py-0.5 rounded text-xs font-semibold ${slaStatusVariant[wo.sla_status || 'no_sla']}`}>{t(slaStatusLabel[wo.sla_status || 'no_sla']) || wo.sla_status || '—'}</span>} />
            {wo.started_at && <InfoRow label={t('workOrder.startedAt') || 'Начало работ'} value={fmtDT(wo.started_at)} />}
            {wo.completed_at && <InfoRow label={t('workOrder.completedAt') || 'Завершено'} value={fmtDT(wo.completed_at)} />}
          </div>
        </Section>
      )}

      {/* Checklist */}
      {wo.checklist.length > 0 && (
        <Section title={`${t('workOrder.checklist') || 'Чеклист'} (${completedChecklist}/${totalChecklist})`}>
          <table className="w-full text-sm">
            <thead><tr className="border-b border-slate-200"><th className="text-left py-1.5 pr-2 font-medium text-slate-600 text-xs uppercase tracking-wider w-10">{t('workOrder.done') || 'Вып.'}</th><th className="text-left py-1.5 font-medium text-slate-600 text-xs uppercase tracking-wider">{t('workOrder.task') || 'Задача'}</th></tr></thead>
            <tbody>{wo.checklist.map((item, idx) => (
              <tr key={idx} className="border-b border-slate-100">
                <td className="py-1.5 pr-2 align-middle"><span className={`inline-flex items-center justify-center w-5 h-5 rounded-full text-xs font-bold ${item.completed ? 'bg-emerald-100 text-emerald-700' : 'bg-slate-100 text-slate-400'}`}>{item.completed ? '✓' : '○'}</span></td>
                <td className={`py-1.5 align-middle ${item.completed ? 'text-slate-500 line-through' : 'text-slate-900'}`}>{item.task}</td>
              </tr>
            ))}</tbody>
          </table>
        </Section>
      )}

      {/* Notes */}
      {wo.notes && (
        <Section title={t('workOrder.notes') || 'Заметки / описание работ'}>
          <div className="bg-slate-50 border border-slate-200 rounded-lg p-3 text-sm text-slate-700 whitespace-pre-wrap">{wo.notes}</div>
        </Section>
      )}

      {/* Parts Used */}
      {hasParts && (
        <Section title={t('workOrder.partsUsed') || 'Запчасти'}>
          <table className="w-full text-sm">
            <thead><tr className="border-b border-slate-200"><th className="text-left py-1.5 font-medium text-slate-600 text-xs uppercase tracking-wider">{t('workOrder.partName') || 'Наименование'}</th><th className="text-left py-1.5 font-medium text-slate-600 text-xs uppercase tracking-wider w-16">{t('workOrder.sku') || 'Артикул'}</th><th className="text-right py-1.5 font-medium text-slate-600 text-xs uppercase tracking-wider w-16">{t('workOrder.qty') || 'Кол-во'}</th><th className="text-right py-1.5 font-medium text-slate-600 text-xs uppercase tracking-wider w-24">{t('workOrder.unitPrice') || 'Цена'}</th><th className="text-right py-1.5 font-medium text-slate-600 text-xs uppercase tracking-wider w-24">{t('workOrder.total') || 'Сумма'}</th></tr></thead>
            <tbody>{(parts && parts.length > 0
              ? parts.map(p => ({ id: p.part_id, name: p.name, sku: p.sku, qty: p.quantity, up: p.unit_price, tp: p.total_price }))
              : wo.parts_used.map(pu => ({ id: pu.part_id, name: pu.part_id, sku: undefined as string | undefined, qty: pu.quantity, up: 0, tp: 0 }))
            ).map((part, idx) => (
              <tr key={part.id || idx} className="border-b border-slate-100">
                <td className="py-1.5 text-slate-900">{part.name}</td>
                <td className="py-1.5 text-slate-500 text-xs">{part.sku || '—'}</td>
                <td className="py-1.5 text-right text-slate-900">{part.qty}</td>
                <td className="py-1.5 text-right text-slate-700">{part.up > 0 ? fmtCurrency(part.up) : '—'}</td>
                <td className="py-1.5 text-right text-slate-900 font-medium">{part.tp > 0 ? fmtCurrency(part.tp) : '—'}</td>
              </tr>
            ))}</tbody>
            {parts && parts.length > 0 && (
              <tfoot><tr className="border-t-2 border-slate-300 font-semibold"><td colSpan={4} className="py-1.5 text-right text-slate-800 text-sm">{t('workOrder.totalParts') || 'Итого по запчастям'}:</td><td className="py-1.5 text-right text-slate-900 text-sm">{fmtCurrency(totalPartsCost, lc?.currency)}</td></tr></tfoot>
            )}
          </table>
        </Section>
      )}

      {/* Time Tracking */}
      {ts && (
        <Section title={t('workOrder.timeTracking') || 'Временные затраты'}>
          <div className="grid grid-cols-3 gap-4">
            <div className="bg-slate-50 border border-slate-200 rounded-lg p-3 text-center"><div className="text-2xl font-bold text-slate-900">{ts.totalHours.toFixed(1)}</div><div className="text-xs text-slate-500 mt-1">{t('workOrder.totalHours') || 'Всего часов'}</div></div>
            <div className="bg-slate-50 border border-slate-200 rounded-lg p-3 text-center"><div className="text-lg font-semibold text-slate-900">{fmtCurrency(ts.hourlyRate, ts.currency)}</div><div className="text-xs text-slate-500 mt-1">{t('workOrder.hourlyRate') || 'Ставка / час'}</div></div>
            <div className="bg-slate-50 border border-slate-200 rounded-lg p-3 text-center"><div className="text-2xl font-bold text-slate-900">{fmtCurrency(ts.totalCost, ts.currency)}</div><div className="text-xs text-slate-500 mt-1">{t('workOrder.totalLaborCost') || 'Стоимость работ'}</div></div>
          </div>
        </Section>
      )}

      {/* Signature */}
      <Section title={t('workOrder.signature') || 'Подпись'}>
        <div className="grid grid-cols-2 gap-8 mt-2">
          <div><div className="text-xs text-slate-500 mb-1">{t('workOrder.technicianSignature') || 'Подпись техника'}</div><div className="border-b border-slate-300 h-8 mb-1" /><div className="text-xs text-slate-400">{t('workOrder.signatureDate') || 'Дата'}: _______________</div></div>
          <div><div className="text-xs text-slate-500 mb-1">{t('workOrder.customerSignature') || 'Подпись заказчика'}</div><div className="border-b border-slate-300 h-8 mb-1" /><div className="text-xs text-slate-400">{t('workOrder.signatureDate') || 'Дата'}: _______________</div></div>
        </div>
      </Section>
    </>
  );
}

function DetailedTemplate({ wo, lc, parts, t, fmtCurrency, fmtDT, fmtD }: {
  wo: WorkOrder; lc: LaborCost | undefined; parts: PartWithCost[] | undefined;
  t: (key: string) => string; fmtCurrency: typeof formatCurrency; fmtDT: typeof formatDateTime; fmtD: typeof formatDate;
}) {
  const totalPartsCost = parts?.reduce((sum, p) => sum + p.total_price, 0) ?? 0;
  const completedChecklist = wo.checklist.filter((c) => c.completed).length;
  const totalChecklist = wo.checklist.length;
  const ts = lc ? { totalHours: lc.total_hours, hourlyRate: lc.hourly_rate, totalCost: lc.total_cost, currency: lc.currency } : null;

  return (
    <>
      {/* Full Device Info + Timelines */}
      <Section title={t('workOrder.deviceInfo') || 'Информация об устройстве'}>
        <div className="grid grid-cols-2 gap-x-6">
          <InfoRow label={t('workOrder.deviceName') || 'Устройство'} value={wo.device_name || '—'} />
          <InfoRow label={t('workOrder.deviceId') || 'ID устройства'} value={<code className="text-xs font-mono bg-slate-100 px-1.5 py-0.5 rounded">{wo.device_id}</code>} />
          <InfoRow label={t('workOrder.type') || 'Тип'} value={t(typeLabel[wo.type] || wo.type) || wo.type} />
          <InfoRow label={t('workOrder.priority') || 'Приоритет'} value={<span className={`inline-block px-2 py-0.5 rounded text-xs font-semibold ${priorityVariant[wo.priority] || ''}`}>{t(priorityLabel[wo.priority] || wo.priority) || wo.priority}</span>} />
          {wo.assigned_to && <InfoRow label={t('workOrder.assignedTo') || 'Назначено'} value={wo.assignee_name || wo.assigned_to} />}
          {wo.schedule_id && <InfoRow label={t('workOrder.scheduleId') || 'ID расписания'} value={wo.schedule_id} />}
          <InfoRow label={t('workOrder.createdAt') || 'Создан'} value={fmtDT(wo.created_at)} />
          {wo.started_at && <InfoRow label={t('workOrder.startedAt') || 'Начало работ'} value={fmtDT(wo.started_at)} />}
          {wo.completed_at && <InfoRow label={t('workOrder.completedAt') || 'Завершено'} value={fmtDT(wo.completed_at)} />}
        </div>
      </Section>

      {/* SLA Detailed */}
      {wo.sla_deadline && (
        <Section title={t('workOrder.slaInfo') || 'SLA Информация'}>
          <div className="grid grid-cols-3 gap-x-6">
            <InfoRow label={t('workOrder.slaDeadline') || 'SLA дедлайн'} value={fmtDT(wo.sla_deadline)} />
            <InfoRow label={t('workOrder.slaStatus') || 'SLA статус'} value={<span className={`inline-block px-2 py-0.5 rounded text-xs font-semibold ${slaStatusVariant[wo.sla_status || 'no_sla']}`}>{t(slaStatusLabel[wo.sla_status || 'no_sla']) || wo.sla_status || '—'}</span>} />
            <InfoRow label="Work Order ID" value={<code className="text-xs font-mono bg-slate-100 px-1.5 py-0.5 rounded">{wo.id}</code>} />
          </div>
        </Section>
      )}

      {/* Checklist */}
      {wo.checklist.length > 0 && (
        <Section title={`${t('workOrder.checklist') || 'Чеклист'} (${completedChecklist}/${totalChecklist})`}>
          <table className="w-full text-sm">
            <thead><tr className="border-b border-slate-200"><th className="text-left py-1.5 font-medium text-slate-600 text-xs uppercase tracking-wider w-8">#</th><th className="text-left py-1.5 font-medium text-slate-600 text-xs uppercase tracking-wider">{t('workOrder.task') || 'Задача'}</th><th className="text-center py-1.5 font-medium text-slate-600 text-xs uppercase tracking-wider w-16">{t('workOrder.status') || 'Статус'}</th></tr></thead>
            <tbody>{wo.checklist.map((item, idx) => (
              <tr key={idx} className="border-b border-slate-100">
                <td className="py-1.5 text-slate-400 text-xs">{idx + 1}</td>
                <td className={`py-1.5 ${item.completed ? 'text-slate-500 line-through' : 'text-slate-900'}`}>{item.task}</td>
                <td className="py-1.5 text-center"><span className={`inline-flex items-center justify-center w-5 h-5 rounded-full text-xs font-bold ${item.completed ? 'bg-emerald-100 text-emerald-700' : 'bg-slate-100 text-slate-400'}`}>{item.completed ? '✓' : '○'}</span></td>
              </tr>
            ))}</tbody>
          </table>
        </Section>
      )}

      {/* Notes */}
      {wo.notes && (
        <Section title={t('workOrder.notes') || 'Заметки / описание работ'}>
          <div className="bg-slate-50 border border-slate-200 rounded-lg p-3 text-sm text-slate-700 whitespace-pre-wrap">{wo.notes}</div>
        </Section>
      )}

      {/* Parts with Cost Details */}
      {parts && parts.length > 0 && (
        <Section title={t('workOrder.partsUsed') || 'Запчасти'}>
          <table className="w-full text-sm">
            <thead><tr className="border-b border-slate-200"><th className="text-left py-1.5 font-medium text-slate-600 text-xs uppercase tracking-wider">{t('workOrder.partName') || 'Наименование'}</th><th className="text-left py-1.5 font-medium text-slate-600 text-xs uppercase tracking-wider w-16">{t('workOrder.sku') || 'Артикул'}</th><th className="text-right py-1.5 font-medium text-slate-600 text-xs uppercase tracking-wider w-16">{t('workOrder.qty') || 'Кол-во'}</th><th className="text-right py-1.5 font-medium text-slate-600 text-xs uppercase tracking-wider w-24">{t('workOrder.unitPrice') || 'Цена'}</th><th className="text-right py-1.5 font-medium text-slate-600 text-xs uppercase tracking-wider w-24">{t('workOrder.total') || 'Сумма'}</th></tr></thead>
            <tbody>{parts.map(part => (
              <tr key={part.part_id} className="border-b border-slate-100">
                <td className="py-1.5 text-slate-900">{part.name}</td><td className="py-1.5 text-slate-500 text-xs">{part.sku || '—'}</td>
                <td className="py-1.5 text-right text-slate-900">{part.quantity}</td>
                <td className="py-1.5 text-right text-slate-700">{fmtCurrency(part.unit_price)}</td>
                <td className="py-1.5 text-right text-slate-900 font-medium">{fmtCurrency(part.total_price)}</td>
              </tr>
            ))}</tbody>
            <tfoot><tr className="border-t-2 border-slate-300 font-semibold"><td colSpan={4} className="py-1.5 text-right text-slate-800 text-sm">{t('workOrder.totalParts') || 'Итого по запчастям'}:</td><td className="py-1.5 text-right text-slate-900 text-sm">{fmtCurrency(totalPartsCost, lc?.currency)}</td></tr></tfoot>
          </table>
        </Section>
      )}

      {/* Time Tracking with Cost Breakdown */}
      {ts && (
        <Section title={t('workOrder.timeTracking') || 'Временные затраты'}>
          <div className="grid grid-cols-4 gap-4">
            <div className="bg-slate-50 border border-slate-200 rounded-lg p-3 text-center"><div className="text-2xl font-bold text-slate-900">{ts.totalHours.toFixed(1)}</div><div className="text-xs text-slate-500 mt-1">{t('workOrder.totalHours') || 'Всего часов'}</div></div>
            <div className="bg-slate-50 border border-slate-200 rounded-lg p-3 text-center"><div className="text-lg font-semibold text-slate-900">{fmtCurrency(ts.hourlyRate, ts.currency)}</div><div className="text-xs text-slate-500 mt-1">{t('workOrder.hourlyRate') || 'Ставка / час'}</div></div>
            <div className="bg-slate-50 border border-slate-200 rounded-lg p-3 text-center"><div className="text-2xl font-bold text-slate-900">{fmtCurrency(ts.totalCost, ts.currency)}</div><div className="text-xs text-slate-500 mt-1">{t('workOrder.laborCost') || 'Стоимость работ'}</div></div>
            <div className="bg-slate-50 border border-slate-200 rounded-lg p-3 text-center"><div className="text-lg font-semibold text-slate-900">{fmtCurrency(ts.totalCost / Math.max(ts.totalHours, 0.1), ts.currency)}</div><div className="text-xs text-slate-500 mt-1">{t('workOrder.avgHourly') || 'Средняя ставка'}</div></div>
          </div>
        </Section>
      )}

      {/* Total Cost Summary */}
      <Section title={t('workOrder.totalSummary') || 'Итоговая стоимость'}>
        <div className="space-y-1.5">
          {ts && <div className="flex justify-between text-sm text-slate-700"><span>{t('workOrder.laborCost') || 'Работы'}</span><span>{fmtCurrency(ts.totalCost, ts.currency)}</span></div>}
          {totalPartsCost > 0 && <div className="flex justify-between text-sm text-slate-700"><span>{t('workOrder.partsCost') || 'Запчасти'}</span><span>{fmtCurrency(totalPartsCost, lc?.currency)}</span></div>}
          <div className="flex justify-between text-base font-bold text-slate-900 pt-2 border-t-2 border-slate-300"><span>{t('workOrder.total') || 'Итого'}</span><span>{fmtCurrency((ts?.totalCost ?? 0) + totalPartsCost, lc?.currency)}</span></div>
        </div>
      </Section>

      {/* Timeline */}
      <Section title={t('workOrder.timeline') || 'Хронология'}>
        <div className="space-y-1 text-xs text-slate-600">
          <p>{t('workOrder.createdAt') || 'Создан'}: {fmtDT(wo.created_at)}</p>
          {wo.started_at && <p>{t('workOrder.startedAt') || 'Начало работ'}: {fmtDT(wo.started_at)}</p>}
          {wo.completed_at && <p>{t('workOrder.completedAt') || 'Завершено'}: {fmtDT(wo.completed_at)}</p>}
        </div>
      </Section>

      {/* Signature */}
      <Section title={t('workOrder.signature') || 'Подпись'}>
        <div className="grid grid-cols-2 gap-8 mt-2">
          <div><div className="text-xs text-slate-500 mb-1">{t('workOrder.technicianSignature') || 'Подпись техника'}</div><div className="border-b border-slate-300 h-8 mb-1" /><div className="text-xs text-slate-400">{t('workOrder.signatureDate') || 'Дата'}: _______________</div></div>
          <div><div className="text-xs text-slate-500 mb-1">{t('workOrder.customerSignature') || 'Подпись заказчика'}</div><div className="border-b border-slate-300 h-8 mb-1" /><div className="text-xs text-slate-400">{t('workOrder.signatureDate') || 'Дата'}: _______________</div></div>
        </div>
      </Section>
    </>
  );
}

function InvoiceTemplate({ wo, lc, parts, t, fmtCurrency, fmtDT }: {
  wo: WorkOrder; lc: LaborCost | undefined; parts: PartWithCost[] | undefined;
  t: (key: string) => string; fmtCurrency: typeof formatCurrency; fmtDT: typeof formatDateTime;
}) {
  const totalPartsCost = parts?.reduce((sum, p) => sum + p.total_price, 0) ?? 0;
  const ts = lc ? { totalHours: lc.total_hours, hourlyRate: lc.hourly_rate, totalCost: lc.total_cost, currency: lc.currency } : null;
  const totalCost = (ts?.totalCost ?? 0) + totalPartsCost;

  return (
    <>
      {/* Invoice Header */}
      <div className="text-center mb-6">
        <h2 className="text-lg font-bold text-slate-900 uppercase tracking-wider">{t('workOrder.invoice') || 'СЧЁТ'}</h2>
        <p className="text-xs text-slate-500 mt-1">
          #{wo.id.slice(0, 8).toUpperCase()} | {t('workOrder.date') || 'Дата'}: {fmtDT(wo.created_at)}
        </p>
      </div>

      {/* Bill To */}
      <Section title={t('workOrder.billTo') || 'Плательщик'} compact>
        <div className="text-sm text-slate-700 space-y-0.5">
          <p className="font-medium">{wo.device_name || wo.device_id}</p>
          <p className="text-slate-500">ID: {wo.device_id}</p>
          {wo.assignee_name && <p className="text-slate-500">{t('workOrder.technician') || 'Техник'}: {wo.assignee_name}</p>}
        </div>
      </Section>

      {/* Line Items */}
      <Section title={t('workOrder.lineItems') || 'Позиции'} compact>
        <table className="w-full text-xs">
          <thead>
            <tr className="border-b border-slate-300">
              <th className="text-left py-1 pr-2 font-semibold text-slate-600 uppercase">#</th>
              <th className="text-left py-1 pr-2 font-semibold text-slate-600 uppercase">{t('workOrder.description') || 'Описание'}</th>
              <th className="text-right py-1 pr-2 font-semibold text-slate-600 uppercase w-16">{t('workOrder.qty') || 'Кол-во'}</th>
              <th className="text-right py-1 pr-2 font-semibold text-slate-600 uppercase w-20">{t('workOrder.unitPrice') || 'Цена'}</th>
              <th className="text-right py-1 font-semibold text-slate-600 uppercase w-20">{t('workOrder.amount') || 'Сумма'}</th>
            </tr>
          </thead>
          <tbody>
            {ts && ts.totalHours > 0 && (
              <tr className="border-b border-slate-100">
                <td className="py-1 pr-2 text-slate-400">1</td>
                <td className="py-1 pr-2 text-slate-900">{t('workOrder.laborService') || 'Работы по обслуживанию'} ({t(wo.type)})</td>
                <td className="py-1 pr-2 text-right text-slate-900">{ts.totalHours.toFixed(1)} ч</td>
                <td className="py-1 pr-2 text-right text-slate-700">{fmtCurrency(ts.hourlyRate, ts.currency)}</td>
                <td className="py-1 pr-2 text-right text-slate-900 font-medium">{fmtCurrency(ts.totalCost, ts.currency)}</td>
              </tr>
            )}
            {parts?.map((part, idx) => (
              <tr key={part.part_id} className="border-b border-slate-100">
                <td className="py-1 pr-2 text-slate-400">{idx + (ts ? 2 : 1)}</td>
                <td className="py-1 pr-2 text-slate-900">{part.name}{part.sku ? ` (${part.sku})` : ''}</td>
                <td className="py-1 pr-2 text-right text-slate-900">{part.quantity}</td>
                <td className="py-1 pr-2 text-right text-slate-700">{fmtCurrency(part.unit_price)}</td>
                <td className="py-1 pr-2 text-right text-slate-900 font-medium">{fmtCurrency(part.total_price)}</td>
              </tr>
            ))}
          </tbody>
          <tfoot>
            <tr className="border-t-2 border-slate-300">
              <td colSpan={3} />
              <td className="py-1.5 pr-2 text-right text-xs font-semibold text-slate-600 uppercase">{t('workOrder.subtotal') || 'Подытог'}</td>
              <td className="py-1.5 text-right text-sm font-semibold text-slate-900">{fmtCurrency(totalCost, lc?.currency)}</td>
            </tr>
            <tr>
              <td colSpan={3} />
              <td className="py-1 pr-2 text-right text-xs text-slate-500">{t('workOrder.tax') || 'Налог'}: 0%</td>
              <td className="py-1 text-right text-slate-500">{fmtCurrency(0, lc?.currency)}</td>
            </tr>
            <tr className="border-t-2 border-slate-900">
              <td colSpan={3} />
              <td className="py-1.5 pr-2 text-right text-sm font-bold text-slate-900 uppercase">{t('workOrder.total') || 'ИТОГО'}</td>
              <td className="py-1.5 text-right text-base font-bold text-slate-900">{fmtCurrency(totalCost, lc?.currency)}</td>
            </tr>
          </tfoot>
        </table>
      </Section>

      {/* Payment Info */}
      <Section title={t('workOrder.paymentInfo') || 'Платёжные реквизиты'} compact>
        <div className="text-xs text-slate-500 space-y-0.5">
          <p>{t('workOrder.paymentTerms') || 'Условия оплаты'}: {t('workOrder.net30') || 'Net 30'}</p>
          <p>{t('workOrder.currency') || 'Валюта'}: {lc?.currency || 'USD'}</p>
          <p className="mt-2 text-slate-400 italic">{t('workOrder.invoiceNote') || 'Счёт действителен в течение 30 дней с даты выставления.'}</p>
        </div>
      </Section>
    </>
  );
}

// ── Main Component ─────────────────────────────────────────────────

export function WorkOrderPrintView({
  workOrder,
  laborCost,
  partsDetail,
  onClose,
  template = 'standard',
}: WorkOrderPrintViewProps) {
  const { t } = useTranslation();
  const meta = TEMPLATE_META[template];
  const TemplateIcon = meta.icon;

  const handlePrint = useCallback(() => window.print(), []);

  // Common data
  const totalPartsCost = partsDetail?.reduce((sum, p) => sum + p.total_price, 0) ?? 0;
  const ts = laborCost ? { totalCost: laborCost.total_cost, currency: laborCost.currency } : null;

  return (
    <>
      {/* ── Toolbar ── */}
      <div className="print-hidden flex items-center justify-between px-4 py-3 bg-white border-b border-slate-200 sticky top-0 z-50">
        <div className="flex items-center gap-2">
          <h2 className="text-lg font-semibold text-slate-900">
            {t('workOrder.printView') || 'Печатная форма наряда'}
          </h2>
          <span className="inline-flex items-center gap-1 px-2 py-0.5 rounded-full text-xs font-medium bg-blue-50 text-blue-700">
            <TemplateIcon className="w-3 h-3" />
            {t(meta.labelKey) || template}
          </span>
        </div>
        <div className="flex items-center gap-2">
          <button
            onClick={handlePrint}
            className="inline-flex items-center gap-2 px-4 py-2 text-sm font-medium text-white bg-blue-600 rounded-lg hover:bg-blue-700 transition-colors"
          >
            <Printer className="w-4 h-4" />
            {t('workOrder.print') || 'Печать'}
          </button>
          {onClose && (
            <button
              onClick={onClose}
              className="inline-flex items-center justify-center p-2 text-slate-500 hover:text-slate-700 rounded-lg hover:bg-slate-100 transition-colors"
              aria-label={t('close') || 'Закрыть'}
            >
              <X className="w-5 h-5" />
            </button>
          )}
        </div>
      </div>

      {/* ── Printable Content ── */}
      <div className="print-content max-w-4xl mx-auto p-6 print:p-0">
        {/* Header */}
        <div className="flex items-start justify-between mb-6 print:mb-4 pb-4 border-b-2 border-slate-400">
          <div>
            <h1 className="text-xl font-bold text-slate-900">
              {template === 'invoice'
                ? (t('workOrder.invoice') || 'СЧЁТ')
                : (t('workOrder.workOrder') || 'Наряд-заказ')}{' '}
              #{workOrder.id.slice(0, 8).toUpperCase()}
            </h1>
            <p className="text-xs text-slate-500 mt-1">
              {t('workOrder.createdAt') || 'Создан'}: {formatDateTime(workOrder.created_at)}
            </p>
          </div>
          <div className="flex flex-wrap gap-1.5 justify-end">
            {template !== 'invoice' && (
              <>
                <span className={`inline-block px-2.5 py-1 rounded text-xs font-semibold ${typeVariant[workOrder.type] || ''}`}>
                  {t(typeLabel[workOrder.type] || workOrder.type) || workOrder.type}
                </span>
                <span className={`inline-block px-2.5 py-1 rounded text-xs font-semibold ${statusVariant[workOrder.status] || ''}`}>
                  {t(statusLabel[workOrder.status] || workOrder.status) || workOrder.status}
                </span>
                <span className={`inline-block px-2.5 py-1 rounded text-xs font-semibold ${priorityVariant[workOrder.priority] || ''}`}>
                  {t(priorityLabel[workOrder.priority] || workOrder.priority) || workOrder.priority}
                </span>
              </>
            )}
          </div>
        </div>

        {/* ── Template Content ── */}
        {template === 'detailed' ? (
          <DetailedTemplate wo={workOrder} lc={laborCost} parts={partsDetail} t={t} fmtCurrency={formatCurrency} fmtDT={formatDateTime} fmtD={formatDate} />
        ) : template === 'invoice' ? (
          <InvoiceTemplate wo={workOrder} lc={laborCost} parts={partsDetail} t={t} fmtCurrency={formatCurrency} fmtDT={formatDateTime} />
        ) : (
          <StandardTemplate wo={workOrder} lc={laborCost} parts={partsDetail} t={t} fmtCurrency={formatCurrency} fmtDT={formatDateTime} fmtD={formatDate} />
        )}

        {/* Footer */}
        {template !== 'invoice' && (
          <div className="mt-8 pt-3 border-t border-slate-200 text-center text-xs text-slate-400">
            {t('workOrder.printFooter') || 'CCTV Health Monitor — Печатная форма наряда-заказа'}
            {' | '}
            {t('workOrder.generatedAt') || 'Сгенерировано'}: {formatDateTime(new Date().toISOString())}
          </div>
        )}
      </div>

      {/* ── Print Styles ── */}
      <style>{`
        @media print {
          .print-hidden { display: none !important; }
          body { margin: 0; padding: 0; -webkit-print-color-adjust: exact; print-color-adjust: exact; }
          .print-content { padding: 0.5in !important; max-width: 100% !important; }
          .bg-emerald-50, .bg-blue-50, .bg-amber-50, .bg-red-50, .bg-slate-50, .bg-slate-100, .bg-cyan-50 {
            -webkit-print-color-adjust: exact; print-color-adjust: exact;
          }
          thead { display: table-header-group; }
          .print-content > div > div { page-break-inside: avoid; }
          tr { page-break-inside: avoid; }
          .border-b { border-bottom-width: 1px !important; }
          a { text-decoration: none; }
          @page { margin: 0.2in; }
        }
      `}</style>
    </>
  );
}

export default WorkOrderPrintView;
