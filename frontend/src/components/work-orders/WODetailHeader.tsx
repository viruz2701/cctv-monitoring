import React from 'react';
import { useTranslation } from 'react-i18next';
import {
  ArrowLeft, Play, CheckCircle, XCircle, Loader2,
  AlertTriangle, Wrench, HardDrive, User,
} from 'lucide-react';
import { Badge, Button, LiveSLATimer } from '../ui';
import { WorkOrder } from '../../services/workOrdersApi';

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

const typeIcon: Record<string, React.ReactNode> = {
  preventive: <Wrench className="w-4 h-4" />,
  corrective: <AlertTriangle className="w-4 h-4" />,
  emergency: <AlertTriangle className="w-4 h-4 text-red-500" />,
};

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

interface WODetailHeaderProps {
  workOrder: WorkOrder;
  submitting: boolean;
  onStart: () => void;
  onComplete: () => void;
  onCancel: () => void;
  onBack: () => void;
}

export const WODetailHeader: React.FC<WODetailHeaderProps> = ({
  workOrder,
  submitting,
  onStart,
  onComplete,
  onCancel,
  onBack,
}) => {
  const { t } = useTranslation();

  return (
    <div className="sticky top-0 z-20 bg-white dark:bg-slate-900 pb-4 border-b border-slate-200 dark:border-slate-700">
      {/* Top row: back + title + badges + actions */}
      <div className="flex items-start gap-4 mb-4">
        <button
          onClick={onBack}
          className="p-2 rounded-lg hover:bg-slate-100 dark:hover:bg-slate-800 transition-colors mt-1"
        >
          <ArrowLeft className="w-5 h-5 text-slate-600 dark:text-slate-400" />
        </button>

        <div className="flex-1 min-w-0">
          <div className="flex items-center gap-3 mb-1 flex-wrap">
            <h1 className="text-2xl font-bold text-slate-900 dark:text-white">
              {t('workOrder.title') || `Наряд-заказ #${workOrder.id.slice(0, 8)}`}
            </h1>
            <Badge variant={statusVariant[workOrder.status] || 'neutral'}>
              {statusLabel[workOrder.status] || workOrder.status}
            </Badge>
            <Badge variant={priorityVariant[workOrder.priority] || 'info'}>
              {workOrder.priority}
            </Badge>
          </div>

          <div className="flex items-center gap-4 text-sm text-slate-500 dark:text-slate-400 flex-wrap">
            <span className="flex items-center gap-1">
              {typeIcon[workOrder.type]}
              {typeLabel[workOrder.type] || workOrder.type}
            </span>
            {workOrder.device_name && (
              <span className="flex items-center gap-1">
                <HardDrive className="w-4 h-4" />
                {workOrder.device_name}
              </span>
            )}
            {workOrder.assignee_name && (
              <span className="flex items-center gap-1">
                <User className="w-4 h-4" />
                {workOrder.assignee_name}
              </span>
            )}
          </div>
        </div>

        {/* Action buttons */}
        <div className="flex items-center gap-2 flex-shrink-0">
          {workOrder.status === 'open' && (
            <Button
              icon={<Play className="w-4 h-4" />}
              onClick={onStart}
              loading={submitting}
            >
              {t('workOrder.start') || 'Начать работу'}
            </Button>
          )}

          {(workOrder.status === 'open' || workOrder.status === 'in_progress') && (
            <Button
              variant="primary"
              icon={<CheckCircle className="w-4 h-4" />}
              onClick={onComplete}
            >
              {t('workOrder.complete') || 'Завершить'}
            </Button>
          )}

          {workOrder.status !== 'cancelled' && workOrder.status !== 'completed' && (
            <Button
              variant="danger"
              icon={<XCircle className="w-4 h-4" />}
              onClick={onCancel}
            >
              {t('workOrder.cancel') || 'Отменить'}
            </Button>
          )}
        </div>
      </div>

      {/* SLA Timer */}
      {workOrder.sla_deadline && workOrder.sla_status && workOrder.sla_status !== 'no_sla' && (
        <LiveSLATimer
          deadline={workOrder.sla_deadline}
          createdAt={workOrder.created_at}
          status={workOrder.sla_status as 'on_track' | 'at_risk' | 'breached' | 'completed' | undefined}
        />
      )}
    </div>
  );
};
