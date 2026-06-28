// ═══════════════════════════════════════════════════════════════════════
// WOStatusCards — статус-карточки левой колонки WorkOrderDetail
// ═══════════════════════════════════════════════════════════════════════

import React from 'react';
import {
  AlertTriangle, Wrench, User,
} from 'lucide-react';
import { Card, CardHeader, CardBody, Badge } from '../../components/ui';
import { SLATimer } from '../../components/work-orders/SLATimer';
import type { WorkOrder } from '../../hooks/useApiQuery';

interface WOStatusCardsProps {
  workOrder: WorkOrder;
}

const typeLabel: Record<string, string> = {
  preventive: 'Плановое',
  corrective: 'Корректирующее',
  emergency: 'Аварийное',
};

const statusLabel: Record<string, string> = {
  open: 'Открыт',
  in_progress: 'В работе',
  completed: 'Завершён',
  cancelled: 'Отменён',
};

const priorityVariant: Record<string, 'danger' | 'warning' | 'info' | 'success'> = {
  critical: 'danger',
  high: 'warning',
  medium: 'info',
  low: 'success',
};

const statusVariant: Record<string, 'neutral' | 'primary' | 'warning' | 'success' | 'danger'> = {
  open: 'primary',
  in_progress: 'warning',
  completed: 'success',
  cancelled: 'danger',
};

export const WOStatusCards: React.FC<WOStatusCardsProps> = ({ workOrder }) => {
  return (
    <>
      {/* Status & Priority */}
      <Card>
        <CardBody>
          <div className="flex items-center justify-between mb-3">
            <Badge variant={statusVariant[workOrder.status] || 'neutral'} size="sm">
              {statusLabel[workOrder.status] || workOrder.status}
            </Badge>
            <Badge variant={priorityVariant[workOrder.priority] || 'info'} size="sm">
              {workOrder.priority}
            </Badge>
          </div>
          <div className="flex items-center gap-2 text-sm text-slate-500 dark:text-slate-400">
            {workOrder.type === 'emergency' ? (
              <AlertTriangle className="w-4 h-4 text-red-500" />
            ) : workOrder.type === 'preventive' ? (
              <Wrench className="w-4 h-4" />
            ) : (
              <AlertTriangle className="w-4 h-4" />
            )}
            <span>{typeLabel[workOrder.type] || workOrder.type}</span>
          </div>
        </CardBody>
      </Card>

      {/* SLA Timer */}
      {workOrder.sla_deadline && workOrder.sla_status && workOrder.sla_status !== 'no_sla' && (
        <SLATimer
          deadline={workOrder.sla_deadline}
          createdAt={workOrder.created_at}
          status={workOrder.sla_status as 'on_track' | 'at_risk' | 'breached' | 'completed'}
        />
      )}

      {/* Assigned Technician */}
      <Card>
        <CardHeader className="flex items-center gap-2">
          <User className="w-5 h-5 text-slate-600 dark:text-slate-400" />
          <span className="text-sm font-medium">Исполнитель</span>
        </CardHeader>
        <CardBody>
          {workOrder.assignee_name ? (
            <div className="flex items-center gap-3">
              <div className="w-9 h-9 rounded-full bg-blue-100 dark:bg-blue-900/50 flex items-center justify-center text-blue-600 dark:text-blue-400 font-semibold text-sm">
                {workOrder.assignee_name.charAt(0).toUpperCase()}
              </div>
              <div>
                <p className="text-sm font-medium text-slate-900 dark:text-white">{workOrder.assignee_name}</p>
                <p className="text-xs text-slate-500 dark:text-slate-400">Техник</p>
              </div>
            </div>
          ) : (
            <p className="text-sm text-slate-400 dark:text-slate-500 italic">Не назначен</p>
          )}
        </CardBody>
      </Card>
    </>
  );
};
