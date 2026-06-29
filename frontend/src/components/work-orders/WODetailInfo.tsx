import React from 'react';
import { useTranslation } from 'react-i18next';
import {
  HardDrive, MapPin, User, Calendar, FileText,
  ClipboardList, CheckCircle, Clock,
} from '../ui/Icons';
import { Card, CardHeader, CardBody, Badge } from '../ui';
import { WorkOrder } from '../../services/workOrdersApi';

interface WODetailInfoProps {
  workOrder: WorkOrder;
}

const typeLabel: Record<string, string> = {
  preventive: 'Плановое',
  corrective: 'Корректирующее',
  emergency: 'Аварийное',
};

export const WODetailInfo: React.FC<WODetailInfoProps> = ({ workOrder }) => {
  const { t } = useTranslation();

  return (
    <div className="space-y-6">
      {/* Device & Assignment Info */}
      <Card>
        <CardHeader className="flex items-center gap-2">
          <HardDrive className="w-5 h-5 text-slate-600 dark:text-slate-400" />
          <span>{t('workOrder.device') || 'Устройство'}</span>
        </CardHeader>
        <CardBody>
          <div className="grid grid-cols-1 md:grid-cols-2 gap-4">
            {workOrder.device_name && (
              <div>
                <p className="text-xs text-slate-500 dark:text-slate-400 mb-1">
                  {t('workOrder.deviceName') || 'Название'}
                </p>
                <p className="text-sm font-medium text-slate-900 dark:text-white">
                  {workOrder.device_name}
                </p>
              </div>
            )}
            <div>
              <p className="text-xs text-slate-500 dark:text-slate-400 mb-1">
                {t('workOrder.deviceId') || 'ID устройства'}
              </p>
              <code className="text-xs font-mono bg-slate-100 dark:bg-slate-800 px-2 py-1 rounded text-slate-700 dark:text-slate-300">
                {workOrder.device_id}
              </code>
            </div>
            {workOrder.schedule_id && (
              <div>
                <p className="text-xs text-slate-500 dark:text-slate-400 mb-1">
                  {t('workOrder.schedule') || 'Расписание'}
                </p>
                <code className="text-xs font-mono bg-slate-100 dark:bg-slate-800 px-2 py-1 rounded text-slate-700 dark:text-slate-300">
                  {workOrder.schedule_id.slice(0, 8)}...
                </code>
              </div>
            )}
            <div>
              <p className="text-xs text-slate-500 dark:text-slate-400 mb-1">
                {t('workOrder.type') || 'Тип'}
              </p>
              <Badge variant="neutral">
                {typeLabel[workOrder.type] || workOrder.type}
              </Badge>
            </div>
          </div>
        </CardBody>
      </Card>

      {/* Location */}
      <Card>
        <CardHeader className="flex items-center gap-2">
          <MapPin className="w-5 h-5 text-slate-600 dark:text-slate-400" />
          <span>{t('workOrder.location') || 'Локация'}</span>
        </CardHeader>
        <CardBody>
          <p className="text-sm text-slate-400 dark:text-slate-500 italic">
            {t('workOrder.locationInfo') || 'Информация о расположении устройства'}
          </p>
        </CardBody>
      </Card>

      {/* Assignee */}
      <Card>
        <CardHeader className="flex items-center gap-2">
          <User className="w-5 h-5 text-slate-600 dark:text-slate-400" />
          <span>{t('workOrder.assignee') || 'Исполнитель'}</span>
        </CardHeader>
        <CardBody>
          {workOrder.assignee_name ? (
            <div className="flex items-center gap-3">
              <div className="w-10 h-10 rounded-full bg-blue-100 dark:bg-blue-900/50 flex items-center justify-center text-blue-600 dark:text-blue-400 font-semibold text-sm">
                {workOrder.assignee_name.charAt(0).toUpperCase()}
              </div>
              <div>
                <p className="font-medium text-slate-900 dark:text-white">{workOrder.assignee_name}</p>
                <p className="text-xs text-slate-500 dark:text-slate-400">
                  {t('workOrder.technician') || 'Техник'}
                </p>
              </div>
            </div>
          ) : (
            <p className="text-sm text-slate-400 dark:text-slate-500 italic">
              {t('workOrder.notAssigned') || 'Не назначен'}
            </p>
          )}
        </CardBody>
      </Card>

      {/* Dates */}
      <Card>
        <CardHeader className="flex items-center gap-2">
          <Calendar className="w-5 h-5 text-slate-600 dark:text-slate-400" />
          <span>{t('workOrder.dates') || 'Даты'}</span>
        </CardHeader>
        <CardBody>
          <div className="grid grid-cols-1 md:grid-cols-2 gap-3">
            <div>
              <p className="text-xs text-slate-500 dark:text-slate-400 mb-1">
                {t('workOrder.created') || 'Создан'}
              </p>
              <p className="text-sm text-slate-900 dark:text-white">
                {new Date(workOrder.created_at).toLocaleString()}
              </p>
            </div>
            {workOrder.started_at && (
              <div>
                <p className="text-xs text-slate-500 dark:text-slate-400 mb-1">
                  {t('workOrder.started') || 'Начат'}
                </p>
                <p className="text-sm text-slate-900 dark:text-white">
                  {new Date(workOrder.started_at).toLocaleString()}
                </p>
              </div>
            )}
            {workOrder.completed_at && (
              <div>
                <p className="text-xs text-slate-500 dark:text-slate-400 mb-1">
                  {t('workOrder.completed') || 'Завершён'}
                </p>
                <p className="text-sm text-slate-900 dark:text-white">
                  {new Date(workOrder.completed_at).toLocaleString()}
                </p>
              </div>
            )}
            <div>
              <p className="text-xs text-slate-500 dark:text-slate-400 mb-1">
                {t('workOrder.updated') || 'Обновлён'}
              </p>
              <p className="text-sm text-slate-900 dark:text-white">
                {new Date(workOrder.updated_at).toLocaleString()}
              </p>
            </div>
          </div>
        </CardBody>
      </Card>

      {/* Description / Notes */}
      {workOrder.notes && (
        <Card>
          <CardHeader className="flex items-center gap-2">
            <FileText className="w-5 h-5 text-slate-600 dark:text-slate-400" />
            <span>{t('workOrder.notes') || 'Заметки'}</span>
          </CardHeader>
          <CardBody>
            <p className="text-sm text-slate-700 dark:text-slate-300 whitespace-pre-wrap">
              {workOrder.notes}
            </p>
          </CardBody>
        </Card>
      )}

      {/* Checklist */}
      {workOrder.checklist?.length > 0 && (
        <Card>
          <CardHeader className="flex items-center gap-2">
            <ClipboardList className="w-5 h-5 text-blue-600 dark:text-blue-400" />
            <span>{t('workOrder.checklist') || 'Чеклист'} ({workOrder.checklist.filter(c => c.completed).length}/{workOrder.checklist.length})</span>
          </CardHeader>
          <CardBody>
            <div className="space-y-2">
              {workOrder.checklist.map((item, i) => (
                <div
                  key={i}
                  className={`flex items-center gap-3 p-3 rounded-lg text-sm transition-all ${
                    item.completed
                      ? 'bg-emerald-50 dark:bg-emerald-900/20 text-emerald-700 dark:text-emerald-300'
                      : 'bg-slate-50 dark:bg-slate-800/50 text-slate-700 dark:text-slate-300'
                  }`}
                >
                  {item.completed ? (
                    <CheckCircle className="w-5 h-5 text-emerald-500 flex-shrink-0" />
                  ) : (
                    <div className="w-5 h-5 rounded-full border-2 border-slate-300 dark:border-slate-600 flex-shrink-0 flex items-center justify-center">
                      <div className="w-2 h-2 rounded-full bg-slate-300 dark:bg-slate-600" />
                    </div>
                  )}
                  <span className="flex-1">{item.task}</span>
                </div>
              ))}
            </div>
          </CardBody>
        </Card>
      )}

      {/* SLA Info */}
      {workOrder.sla_deadline && (
        <Card>
          <CardHeader className="flex items-center gap-2">
            <Clock className="w-5 h-5 text-slate-600 dark:text-slate-400" />
            <span>{t('workOrder.sla') || 'SLA'}</span>
          </CardHeader>
          <CardBody>
            <div className="flex items-center justify-between">
              <span className="text-sm text-slate-600 dark:text-slate-400">
                {t('workOrder.slaDeadline') || 'Крайний срок SLA'}
              </span>
              <span className="text-sm font-medium text-slate-900 dark:text-white">
                {new Date(workOrder.sla_deadline).toLocaleString()}
              </span>
            </div>
          </CardBody>
        </Card>
      )}
    </div>
  );
};
