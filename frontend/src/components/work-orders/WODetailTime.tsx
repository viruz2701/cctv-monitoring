import React from 'react';
import { useTranslation } from 'react-i18next';
import {
  Timer, Play, Pause, Square, Trash2, DollarSign,
} from '../ui/Icons';
import { Card, CardHeader, CardBody, Badge, Button } from '../ui';
import { TimeEntry, LaborCost } from '../../services/workOrdersApi';

interface WODetailTimeProps {
  timeEntries: TimeEntry[];
  laborCost: LaborCost | null;
  elapsed: number;
  timeSubmitting: boolean;
  workOrderStatus: string;
  totalCost?: number;
  totalLaborCost?: number;
  totalPartsCost?: number;
  onStartTimer: () => void;
  onPauseTimer: (entryId: string) => void;
  onResumeTimer: (entryId: string) => void;
  onStopTimer: (entryId: string) => void;
  onDeleteTimeEntry: (entryId: string) => void;
  formatDuration: (seconds: number) => string;
}

export const WODetailTime: React.FC<WODetailTimeProps> = ({
  timeEntries,
  laborCost,
  elapsed,
  timeSubmitting,
  workOrderStatus,
  totalCost,
  totalLaborCost,
  totalPartsCost,
  onStartTimer,
  onPauseTimer,
  onResumeTimer,
  onStopTimer,
  onDeleteTimeEntry,
  formatDuration,
}) => {
  const { t } = useTranslation();
  const isReadOnly = workOrderStatus === 'completed' || workOrderStatus === 'cancelled';
  const runningEntry = timeEntries.find(e => e.status === 'running');
  const pausedEntry = timeEntries.find(e => e.status === 'paused');

  return (
    <div className="space-y-6">
      {/* Active Timer */}
      <Card>
        <CardHeader className="flex items-center gap-2">
          <Timer className="w-5 h-5 text-blue-600 dark:text-blue-400" />
          <span>{t('workOrder.timeTracking') || 'Учёт времени'}</span>
        </CardHeader>
        <CardBody>
          {runningEntry || pausedEntry ? (
            <div className="text-center">
              <div className="text-4xl font-mono font-bold text-slate-900 dark:text-white tracking-wider">
                {formatDuration(elapsed)}
              </div>
              <p className="text-xs text-slate-500 dark:text-slate-400 mt-2">
                {runningEntry
                  ? (t('workOrder.recording') || 'Идёт запись...')
                  : (t('workOrder.paused') || 'На паузе')}
              </p>
              <div className="flex justify-center gap-3 mt-4">
                {runningEntry ? (
                  <>
                    <Button
                      size="sm"
                      variant="secondary"
                      icon={<Pause className="w-3.5 h-3.5" />}
                      onClick={() => onPauseTimer(runningEntry.id)}
                      loading={timeSubmitting}
                    >
                      {t('workOrder.pause') || 'Пауза'}
                    </Button>
                    <Button
                      size="sm"
                      variant="danger"
                      icon={<Square className="w-3.5 h-3.5" />}
                      onClick={() => onStopTimer(runningEntry.id)}
                      loading={timeSubmitting}
                    >
                      {t('workOrder.stop') || 'Стоп'}
                    </Button>
                  </>
                ) : pausedEntry ? (
                  <>
                    <Button
                      size="sm"
                      variant="primary"
                      icon={<Play className="w-3.5 h-3.5" />}
                      onClick={() => onResumeTimer(pausedEntry.id)}
                      loading={timeSubmitting}
                    >
                      {t('workOrder.resume') || 'Продолжить'}
                    </Button>
                    <Button
                      size="sm"
                      variant="danger"
                      icon={<Square className="w-3.5 h-3.5" />}
                      onClick={() => onStopTimer(pausedEntry.id)}
                      loading={timeSubmitting}
                    >
                      {t('workOrder.stop') || 'Стоп'}
                    </Button>
                  </>
                ) : null}
              </div>
            </div>
          ) : (
            <div>
              <Button
                fullWidth
                icon={<Play className="w-4 h-4" />}
                onClick={onStartTimer}
                loading={timeSubmitting}
                disabled={isReadOnly}
              >
                {t('workOrder.startTimer') || 'Начать учёт времени'}
              </Button>
            </div>
          )}
        </CardBody>
      </Card>

      {/* Time Entries List */}
      {timeEntries.length > 0 && (
        <Card>
          <CardHeader className="flex items-center gap-2">
            <Timer className="w-5 h-5 text-slate-600 dark:text-slate-400" />
            <span>{t('workOrder.timeEntries') || 'Записи времени'}</span>
          </CardHeader>
          <CardBody>
            <div className="space-y-2 max-h-60 overflow-y-auto">
              {[...timeEntries]
                .sort((a, b) => new Date(b.started_at).getTime() - new Date(a.started_at).getTime())
                .map(entry => (
                  <div
                    key={entry.id}
                    className="flex items-center justify-between p-3 bg-slate-50 dark:bg-slate-800/50 rounded-lg text-sm"
                  >
                    <div className="flex items-center gap-3 min-w-0">
                      <div className={`w-2.5 h-2.5 rounded-full flex-shrink-0 ${
                        entry.status === 'running' ? 'bg-green-500 animate-pulse' :
                        entry.status === 'paused' ? 'bg-yellow-500' :
                        'bg-slate-400'
                      }`} />
                      <div className="min-w-0">
                        <span className="text-slate-700 dark:text-slate-300 font-mono font-medium">
                          {formatDuration(entry.total_seconds)}
                        </span>
                        <p className="text-xs text-slate-400 dark:text-slate-500 mt-0.5">
                          {new Date(entry.started_at).toLocaleString()}
                        </p>
                      </div>
                    </div>
                    <div className="flex items-center gap-2 flex-shrink-0">
                      <Badge variant={
                        entry.status === 'running' ? 'success' :
                        entry.status === 'paused' ? 'warning' : 'neutral'
                      } size="sm">
                        {entry.status === 'running'
                          ? (t('workOrder.active') || 'Активен')
                          : entry.status === 'paused'
                            ? (t('workOrder.paused') || 'Пауза')
                            : (t('workOrder.stopped') || 'Остановлен')}
                      </Badge>
                      {entry.status === 'stopped' && (
                        <button
                          onClick={() => onDeleteTimeEntry(entry.id)}
                          className="p-1.5 rounded hover:bg-red-100 dark:hover:bg-red-900/30 text-slate-400 hover:text-red-500 transition-colors"
                        >
                          <Trash2 className="w-3.5 h-3.5" />
                        </button>
                      )}
                    </div>
                  </div>
                ))}
            </div>
          </CardBody>
        </Card>
      )}

      {/* Labor Cost */}
      <Card>
        <CardHeader className="flex items-center gap-2">
          <DollarSign className="w-5 h-5 text-emerald-600 dark:text-emerald-400" />
          <span>{t('workOrder.laborCost') || 'Стоимость работ'}</span>
        </CardHeader>
        <CardBody>
          {laborCost ? (
            <div className="space-y-3">
              <div className="grid grid-cols-2 gap-4">
                <div className="p-3 bg-slate-50 dark:bg-slate-800/50 rounded-lg text-center">
                  <p className="text-xs text-slate-500 dark:text-slate-400 mb-1">
                    {t('workOrder.hours') || 'Часов'}
                  </p>
                  <p className="text-lg font-bold text-slate-900 dark:text-white">
                    {laborCost.total_hours.toFixed(1)}
                  </p>
                </div>
                <div className="p-3 bg-slate-50 dark:bg-slate-800/50 rounded-lg text-center">
                  <p className="text-xs text-slate-500 dark:text-slate-400 mb-1">
                    {t('workOrder.rate') || 'Ставка'}
                  </p>
                  <p className="text-lg font-bold text-slate-900 dark:text-white">
                    {laborCost.currency}{laborCost.hourly_rate.toFixed(2)}
                  </p>
                </div>
              </div>
              <div className="p-4 bg-emerald-50 dark:bg-emerald-900/20 rounded-lg text-center">
                <p className="text-xs text-emerald-600 dark:text-emerald-400 mb-1">
                  {t('workOrder.total') || 'Всего'}
                </p>
                <p className="text-2xl font-bold text-emerald-700 dark:text-emerald-300">
                  {laborCost.currency}{laborCost.total_cost.toFixed(2)}
                </p>
              </div>
            </div>
          ) : (
            <p className="text-sm text-slate-400 dark:text-slate-500 italic text-center py-4">
              {t('workOrder.noLaborCost') || 'Нет данных о стоимости'}
            </p>
          )}
        </CardBody>
      </Card>

      {/* Total Cost Dashboard */}
      {(totalCost != null || totalLaborCost != null || totalPartsCost != null) && (
        <Card>
          <CardHeader className="flex items-center gap-2">
            <DollarSign className="w-5 h-5 text-blue-600 dark:text-blue-400" />
            <span>{t('workOrder.totalCostDashboard') || 'Total Cost Dashboard'}</span>
          </CardHeader>
          <CardBody>
            <div className="grid grid-cols-3 gap-3">
              <div className="text-center p-3 bg-blue-50 dark:bg-blue-900/20 rounded-lg">
                <p className="text-xs text-blue-600 dark:text-blue-400 mb-1">
                  {t('workOrder.totalCost') || 'Total Cost'}
                </p>
                <p className="text-lg font-bold text-blue-700 dark:text-blue-300">
                  {totalCost != null ? `${totalCost.toFixed(2)} ₽` : '—'}
                </p>
              </div>
              <div className="text-center p-3 bg-emerald-50 dark:bg-emerald-900/20 rounded-lg">
                <p className="text-xs text-emerald-600 dark:text-emerald-400 mb-1">
                  {t('workOrder.laborCost') || 'Labor Cost'}
                </p>
                <p className="text-lg font-bold text-emerald-700 dark:text-emerald-300">
                  {totalLaborCost != null ? `${totalLaborCost.toFixed(2)} ₽` : '—'}
                </p>
              </div>
              <div className="text-center p-3 bg-amber-50 dark:bg-amber-900/20 rounded-lg">
                <p className="text-xs text-amber-600 dark:text-amber-400 mb-1">
                  {t('workOrder.partsCost') || 'Parts Cost'}
                </p>
                <p className="text-lg font-bold text-amber-700 dark:text-amber-300">
                  {totalPartsCost != null ? `${totalPartsCost.toFixed(2)} ₽` : '—'}
                </p>
              </div>
            </div>
          </CardBody>
        </Card>
      )}
    </div>
  );
};
