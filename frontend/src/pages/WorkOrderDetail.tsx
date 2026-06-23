import React, { useEffect, useState, useMemo, useCallback } from 'react';
import { useParams, useNavigate } from 'react-router-dom';
import { useTranslation } from 'react-i18next';
import {
  ArrowLeft, Clock, MapPin, HardDrive, User, AlertTriangle,
  Wrench, CheckCircle, XCircle, Play, ClipboardList, Package,
  Camera, FileText, Send, Loader2, Calendar,
} from 'lucide-react';
import {
  Card, CardHeader, CardBody, Badge, Button, useToast, Modal,
  SLAProgress, Timeline, FileUpload, PartCard,
} from '../components/ui';
import { workOrdersApi, WorkOrder, PartUsage } from '../services/workOrdersApi';
import { sparePartsApi, SparePart } from '../services/sparePartsApi';
import { useWorkOrders } from '../context/WorkOrdersContext';
import { PermissionGuard } from '../components/auth/PermissionGuard';
import { DragDropContext, Droppable, Draggable, DropResult } from '@hello-pangea/dnd';
import { PhotoAnnotation } from '../components/work-orders/PhotoAnnotation';
import { BeforeAfterSlider } from '../components/work-orders/BeforeAfterSlider';

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

export const WorkOrderDetail: React.FC = () => {
  const { t } = useTranslation();
  const { id } = useParams<{ id: string }>();
  const navigate = useNavigate();
  const toast = useToast();
  const { fetchWorkOrders } = useWorkOrders();

  const [workOrder, setWorkOrder] = useState<WorkOrder | null>(null);
  const [loading, setLoading] = useState(true);
  const [spareParts, setSpareParts] = useState<SparePart[]>([]);
  const [completeModal, setCompleteModal] = useState(false);
  const [completeNotes, setCompleteNotes] = useState('');
  const [completePhotos, setCompletePhotos] = useState<string[]>([]);
  const [completeParts, setCompleteParts] = useState<PartUsage[]>([]);
  const [cancelModal, setCancelModal] = useState(false);
  const [cancelReason, setCancelReason] = useState('');
  const [submitting, setSubmitting] = useState(false);

  useEffect(() => {
    if (!id) return;
    const fetch = async () => {
      setLoading(true);
      try {
        const [wo, parts] = await Promise.all([
          workOrdersApi.getWorkOrder(id),
          sparePartsApi.getSpareParts(),
        ]);
        setWorkOrder(wo);
        setSpareParts(parts || []);
      } catch (err) {
        console.error(err);
        toast.error('Не удалось загрузить наряд-заказ');
      } finally {
        setLoading(false);
      }
    };
    fetch();
  }, [id]);

  const handleStart = async () => {
    if (!id) return;
    setSubmitting(true);
    try {
      await workOrdersApi.startWorkOrder(id);
      const updated = await workOrdersApi.getWorkOrder(id);
      setWorkOrder(updated);
      toast.success('Наряд-заказ начат');
    } catch {
      toast.error('Ошибка при старте');
    } finally {
      setSubmitting(false);
    }
  };

  const handleComplete = async () => {
    if (!id) return;
    setSubmitting(true);
    try {
      await workOrdersApi.completeWorkOrder(id, completeNotes, completePhotos, completeParts);
      const updated = await workOrdersApi.getWorkOrder(id);
      setWorkOrder(updated);
      setCompleteModal(false);
      fetchWorkOrders();
      toast.success('Наряд-заказ завершён');
    } catch {
      toast.error('Ошибка при завершении');
    } finally {
      setSubmitting(false);
    }
  };

  const handleCancel = async () => {
    if (!id) return;
    setSubmitting(true);
    try {
      await workOrdersApi.cancelWorkOrder(id, cancelReason);
      const updated = await workOrdersApi.getWorkOrder(id);
      setWorkOrder(updated);
      setCancelModal(false);
      fetchWorkOrders();
      toast.success('Наряд-заказ отменён');
    } catch {
      toast.error('Ошибка при отмене');
    } finally {
      setSubmitting(false);
    }
  };

  const handleFileUpload = async (files: File[]) => {
    if (!id) return;
    try {
      const result = await workOrdersApi.uploadPhotos(id, files);
      setCompletePhotos(prev => [...prev, ...(result.photos || [])]);
      toast.success(`${files.length} файлов загружено`);
    } catch {
      toast.error('Ошибка загрузки');
    }
  };

  const togglePartUsage = (partId: string, quantity: number) => {
    setCompleteParts(prev => {
      const existing = prev.find(p => p.part_id === partId);
      if (existing) {
        return quantity === 0
          ? prev.filter(p => p.part_id !== partId)
          : prev.map(p => p.part_id === partId ? { ...p, quantity } : p);
      }
      return [...prev, { part_id: partId, quantity }];
    });
  };

  const timelineEvents = useMemo(() => {
    if (!workOrder) return [];
    const events = [];
    events.push({
      id: 'created',
      timestamp: workOrder.created_at,
      type: 'system' as const,
      title: 'Наряд-заказ создан',
      description: `Тип: ${typeLabel[workOrder.type] || workOrder.type}, Приоритет: ${workOrder.priority}`,
      user: workOrder.created_by,
    });
    if (workOrder.started_at) {
      events.push({
        id: 'started',
        timestamp: workOrder.started_at,
        type: 'status_change' as const,
        title: 'Работа начата',
        user: workOrder.assignee_name,
      });
    }
    if (workOrder.completed_at) {
      events.push({
        id: 'completed',
        timestamp: workOrder.completed_at,
        type: 'status_change' as const,
        title: 'Работа завершена',
        user: workOrder.assignee_name,
      });
    }
    if (workOrder.parts_used?.length > 0) {
      workOrder.parts_used.forEach((p, i) => {
        events.push({
          id: `part-${i}`,
          timestamp: workOrder.completed_at || workOrder.updated_at,
          type: 'part' as const,
          title: `Запчасть использована: ${p.part_id}`,
          description: `Количество: ${p.quantity}`,
        });
      });
    }
    return events.sort((a, b) => new Date(b.timestamp).getTime() - new Date(a.timestamp).getTime());
  }, [workOrder]);

  if (loading) {
    return (
      <div className="p-6 flex items-center justify-center h-64">
        <Loader2 className="w-8 h-8 animate-spin text-blue-600" />
      </div>
    );
  }

  if (!workOrder) {
    return (
      <div className="p-6 text-center">
        <p className="text-slate-500 dark:text-slate-400">Наряд-заказ не найден</p>
        <Button variant="outline" className="mt-4" onClick={() => navigate('/work-orders')}>
          Вернуться к списку
        </Button>
      </div>
    );
  }

  return (
    <PermissionGuard requiredRole={['admin', 'manager', 'technician']}>
      <div className="p-6">
        {/* Header */}
        <div className="flex items-center gap-4 mb-6">
          <button
            onClick={() => navigate('/work-orders')}
            className="p-2 rounded-lg hover:bg-slate-100 dark:hover:bg-slate-800 transition-colors"
          >
            <ArrowLeft className="w-5 h-5 text-slate-600 dark:text-slate-400" />
          </button>
          <div className="flex-1">
            <div className="flex items-center gap-3 mb-1">
              <h1 className="text-2xl font-bold text-slate-900 dark:text-white">
                Наряд-заказ #{workOrder.id.slice(0, 8)}
              </h1>
              <Badge variant={statusVariant[workOrder.status] || 'neutral'}>
                {statusLabel[workOrder.status] || workOrder.status}
              </Badge>
              <Badge variant={priorityVariant[workOrder.priority] || 'info'}>
                {workOrder.priority}
              </Badge>
            </div>
            <div className="flex items-center gap-4 text-sm text-slate-500 dark:text-slate-400">
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
        </div>

        {/* SLA Bar */}
        {workOrder.sla_deadline && workOrder.sla_status && workOrder.sla_status !== 'no_sla' && (
          <div className="mb-6">
            <SLAProgress
              deadline={workOrder.sla_deadline}
              createdAt={workOrder.created_at}
              status={workOrder.sla_status}
            />
          </div>
        )}

        {/* Three-Column Layout */}
        <div className="grid grid-cols-1 lg:grid-cols-3 gap-6">
          {/* LEFT: Details & Checklist */}
          <div className="space-y-6">
            <Card>
              <CardHeader className="flex items-center gap-2">
                <FileText className="w-5 h-5 text-slate-600 dark:text-slate-400" />
                <span>Детали наряда</span>
              </CardHeader>
              <CardBody>
                <dl className="space-y-3">
                  <div className="flex justify-between text-sm">
                    <dt className="text-slate-500 dark:text-slate-400">ID устройства</dt>
                    <dd className="font-mono text-slate-900 dark:text-white">{workOrder.device_id}</dd>
                  </div>
                  {workOrder.schedule_id && (
                    <div className="flex justify-between text-sm">
                      <dt className="text-slate-500 dark:text-slate-400">Расписание</dt>
                      <dd className="font-mono text-slate-900 dark:text-white">{workOrder.schedule_id.slice(0, 8)}</dd>
                    </div>
                  )}
                  <div className="flex justify-between text-sm">
                    <dt className="text-slate-500 dark:text-slate-400">Создан</dt>
                    <dd className="text-slate-900 dark:text-white">
                      {new Date(workOrder.created_at).toLocaleString()}
                    </dd>
                  </div>
                  {workOrder.started_at && (
                    <div className="flex justify-between text-sm">
                      <dt className="text-slate-500 dark:text-slate-400">Начат</dt>
                      <dd className="text-slate-900 dark:text-white">
                        {new Date(workOrder.started_at).toLocaleString()}
                      </dd>
                    </div>
                  )}
                  {workOrder.completed_at && (
                    <div className="flex justify-between text-sm">
                      <dt className="text-slate-500 dark:text-slate-400">Завершён</dt>
                      <dd className="text-slate-900 dark:text-white">
                        {new Date(workOrder.completed_at).toLocaleString()}
                      </dd>
                    </div>
                  )}
                  {workOrder.sla_deadline && (
                    <div className="flex justify-between text-sm">
                      <dt className="text-slate-500 dark:text-slate-400">SLA дедлайн</dt>
                      <dd className="flex items-center gap-1 text-slate-900 dark:text-white">
                        <Clock className="w-3.5 h-3.5" />
                        {new Date(workOrder.sla_deadline).toLocaleString()}
                      </dd>
                    </div>
                  )}
                </dl>
              </CardBody>
            </Card>

            {/* Checklist — Drag-and-Drop (@hello-pangea/dnd) */}
            {workOrder.checklist?.length > 0 && (
              <Card>
                <CardHeader className="flex items-center gap-2">
                  <ClipboardList className="w-5 h-5 text-blue-600 dark:text-blue-400" />
                  <span>Чеклист</span>
                </CardHeader>
                <CardBody>
                  <DragDropContext
                    onDragEnd={(result: DropResult) => {
                      if (!result.destination || !workOrder) return;
                      const items = Array.from(workOrder.checklist);
                      const [reordered] = items.splice(result.source.index, 1);
                      items.splice(result.destination.index, 0, reordered);
                      setWorkOrder({ ...workOrder, checklist: items });
                    }}
                  >
                    <Droppable droppableId="checklist">
                      {(provided) => (
                        <div
                          {...provided.droppableProps}
                          ref={provided.innerRef}
                          className="space-y-2"
                        >
                          {workOrder.checklist.map((item, i) => (
                            <Draggable key={`checklist-${i}`} draggableId={`checklist-${i}`} index={i}>
                              {(provided, snapshot) => (
                                <div
                                  ref={provided.innerRef}
                                  {...provided.draggableProps}
                                  {...provided.dragHandleProps}
                                  className={`flex items-center gap-3 p-3 rounded-lg text-sm transition-all ${
                                    snapshot.isDragging
                                      ? 'bg-blue-50 dark:bg-blue-900/30 shadow-lg ring-2 ring-blue-400'
                                      : item.completed
                                        ? 'bg-emerald-50 dark:bg-emerald-900/20 text-emerald-700 dark:text-emerald-300'
                                        : 'bg-slate-50 dark:bg-slate-800/50 text-slate-700 dark:text-slate-300'
                                  } ${
                                    snapshot.isDragging ? 'rotate-2' : ''
                                  }`}
                                  onClick={() => {
                                    if (workOrder.status === 'in_progress' || workOrder.status === 'open') {
                                      const updated = { ...workOrder };
                                      updated.checklist = updated.checklist.map((ci, idx) =>
                                        idx === i ? { ...ci, completed: !ci.completed } : ci
                                      );
                                      setWorkOrder(updated);
                                    }
                                  }}
                                >
                                  {item.completed ? (
                                    <CheckCircle className="w-5 h-5 text-emerald-500 flex-shrink-0" />
                                  ) : (
                                    <div className="w-5 h-5 rounded-full border-2 border-slate-300 dark:border-slate-600 flex-shrink-0 flex items-center justify-center">
                                      <div className="w-2 h-2 rounded-full bg-slate-300 dark:bg-slate-600" />
                                    </div>
                                  )}
                                  <span className="flex-1">{item.task}</span>
                                  <span className="text-xs text-slate-400 dark:text-slate-500 font-mono">☰</span>
                                </div>
                              )}
                            </Draggable>
                          ))}
                          {provided.placeholder}
                        </div>
                      )}
                    </Droppable>
                  </DragDropContext>
                </CardBody>
              </Card>
            )}

            {/* Notes */}
            {workOrder.notes && (
              <Card>
                <CardHeader className="flex items-center gap-2">
                  <FileText className="w-5 h-5 text-slate-600 dark:text-slate-400" />
                  <span>Заметки</span>
                </CardHeader>
                <CardBody>
                  <p className="text-sm text-slate-700 dark:text-slate-300 whitespace-pre-wrap">
                    {workOrder.notes}
                  </p>
                </CardBody>
              </Card>
            )}
          </div>

          {/* CENTER: Timeline & Photos */}
          <div className="space-y-6">
            <Card>
              <CardHeader className="flex items-center gap-2">
                <Clock className="w-5 h-5 text-slate-600 dark:text-slate-400" />
                <span>История</span>
              </CardHeader>
              <CardBody>
                <Timeline events={timelineEvents} />
              </CardBody>
            </Card>

            {/* Photos with Annotation & Before/After */}
            {workOrder.photos?.length > 0 && (
              <Card>
                <CardHeader className="flex items-center gap-2">
                  <Camera className="w-5 h-5 text-slate-600 dark:text-slate-400" />
                  <span>Фотографии ({workOrder.photos.length})</span>
                </CardHeader>
                <CardBody>
                  <div className="space-y-4">
                    {/* Photo Annotation for first photo */}
                    <div>
                      <p className="text-xs font-medium text-slate-500 dark:text-slate-400 mb-2">
                        Аннотация на фото
                      </p>
                      <PhotoAnnotation
                        imageUrl={workOrder.photos[0]}
                        readOnly={workOrder.status === 'completed' || workOrder.status === 'cancelled'}
                      />
                    </div>

                    {/* Before/After slider if 2+ photos */}
                    {workOrder.photos.length >= 2 && (
                      <div>
                        <p className="text-xs font-medium text-slate-500 dark:text-slate-400 mb-2">
                          Сравнение «До/После»
                        </p>
                        <BeforeAfterSlider
                          beforeImage={workOrder.photos[0]}
                          afterImage={workOrder.photos[workOrder.photos.length - 1]}
                          beforeLabel="До"
                          afterLabel="После"
                        />
                      </div>
                    )}

                    {/* All photos grid */}
                    <div className="grid grid-cols-2 gap-2">
                      {workOrder.photos.map((photo, i) => (
                        <a
                          key={i}
                          href={photo}
                          target="_blank"
                          rel="noopener noreferrer"
                          className="block aspect-video rounded-lg overflow-hidden bg-slate-100 dark:bg-slate-800 hover:ring-2 hover:ring-blue-500 transition-all"
                        >
                          <img
                            src={photo}
                            alt={`Фото ${i + 1}`}
                            className="w-full h-full object-cover"
                          />
                        </a>
                      ))}
                    </div>
                  </div>
                </CardBody>
              </Card>
            )}
          </div>

          {/* RIGHT: Actions */}
          <div className="space-y-6">
            {/* Actions Card */}
            <Card>
              <CardHeader className="flex items-center gap-2">
                <Wrench className="w-5 h-5 text-slate-600 dark:text-slate-400" />
                <span>Действия</span>
              </CardHeader>
              <CardBody>
                <div className="space-y-3">
                  {workOrder.status === 'open' && (
                    <Button
                      fullWidth
                      icon={<Play className="w-4 h-4" />}
                      onClick={handleStart}
                      loading={submitting}
                    >
                      Начать работу
                    </Button>
                  )}

                  {(workOrder.status === 'open' || workOrder.status === 'in_progress') && (
                    <Button
                      fullWidth
                      variant="primary"
                      icon={<CheckCircle className="w-4 h-4" />}
                      onClick={() => setCompleteModal(true)}
                    >
                      Завершить
                    </Button>
                  )}

                  {workOrder.status !== 'cancelled' && workOrder.status !== 'completed' && (
                    <Button
                      fullWidth
                      variant="danger"
                      icon={<XCircle className="w-4 h-4" />}
                      onClick={() => setCancelModal(true)}
                    >
                      Отменить
                    </Button>
                  )}
                </div>
              </CardBody>
            </Card>

            {/* Parts Used */}
            {workOrder.parts_used?.length > 0 && (
              <Card>
                <CardHeader className="flex items-center gap-2">
                  <Package className="w-5 h-5 text-slate-600 dark:text-slate-400" />
                  <span>Использованные запчасти</span>
                </CardHeader>
                <CardBody>
                  <div className="space-y-2">
                    {workOrder.parts_used.map((p, i) => {
                      const part = spareParts.find(sp => sp.id === p.part_id);
                      return (
                        <div
                          key={i}
                          className="flex justify-between items-center p-2 bg-slate-50 dark:bg-slate-800/50 rounded-lg text-sm"
                        >
                          <span className="text-slate-700 dark:text-slate-300">
                            {part?.name || p.part_id}
                          </span>
                          <Badge variant="info">{p.quantity} шт</Badge>
                        </div>
                      );
                    })}
                  </div>
                </CardBody>
              </Card>
            )}
          </div>
        </div>

        {/* Complete Modal */}
        <Modal
          isOpen={completeModal}
          onClose={() => setCompleteModal(false)}
          title="Завершить наряд-заказ"
          size="lg"
        >
          <div className="space-y-6">
            {/* Notes */}
            <div>
              <label className="block text-sm font-medium text-slate-700 dark:text-slate-200 mb-2">
                Заметки о выполнении
              </label>
              <textarea
                value={completeNotes}
                onChange={(e) => setCompleteNotes(e.target.value)}
                rows={3}
                className="w-full px-3 py-2 border border-slate-300 dark:border-slate-600 rounded-lg bg-white dark:bg-slate-800 text-slate-900 dark:text-white focus:ring-2 focus:ring-blue-500 focus:border-transparent"
                placeholder="Опишите выполненные работы..."
              />
            </div>

            {/* File Upload */}
            <div>
              <label className="block text-sm font-medium text-slate-700 dark:text-slate-200 mb-2">
                Фотографии
              </label>
              <FileUpload
                onUpload={handleFileUpload}
                accept="image/*"
                maxFiles={10}
                maxSizeMB={20}
                label="Перетащите фото или нажмите для выбора"
              />
              {completePhotos.length > 0 && (
                <div className="grid grid-cols-3 gap-2 mt-2">
                  {completePhotos.map((p, i) => (
                    <div key={i} className="relative aspect-video rounded-lg overflow-hidden bg-slate-100 dark:bg-slate-800">
                      <img src={p} alt="" className="w-full h-full object-cover" />
                      <button
                        onClick={() => setCompletePhotos(prev => prev.filter((_, j) => j !== i))}
                        className="absolute top-1 right-1 p-0.5 bg-red-500 text-white rounded-full"
                      >
                        <XCircle className="w-3.5 h-3.5" />
                      </button>
                    </div>
                  ))}
                </div>
              )}
            </div>

            {/* Parts Selection */}
            <div>
              <label className="block text-sm font-medium text-slate-700 dark:text-slate-200 mb-2">
                Использованные запчасти
              </label>
              <div className="grid grid-cols-1 sm:grid-cols-2 gap-2 max-h-48 overflow-y-auto">
                {spareParts.map(part => {
                  const used = completeParts.find(p => p.part_id === part.id);
                  return (
                    <div
                      key={part.id}
                      className="flex items-center justify-between p-2 bg-slate-50 dark:bg-slate-800/50 rounded-lg text-sm"
                    >
                      <span className="text-slate-700 dark:text-slate-300 truncate flex-1 mr-2">
                        {part.name}
                      </span>
                      <div className="flex items-center gap-1">
                        <button
                          onClick={() => togglePartUsage(part.id, Math.max(0, (used?.quantity || 0) - 1))}
                          className="w-6 h-6 rounded bg-slate-200 dark:bg-slate-700 flex items-center justify-center text-slate-600 dark:text-slate-400 hover:bg-slate-300 dark:hover:bg-slate-600"
                        >
                          −
                        </button>
                        <span className="w-6 text-center font-mono text-sm">
                          {used?.quantity || 0}
                        </span>
                        <button
                          onClick={() => togglePartUsage(part.id, (used?.quantity || 0) + 1)}
                          className="w-6 h-6 rounded bg-slate-200 dark:bg-slate-700 flex items-center justify-center text-slate-600 dark:text-slate-400 hover:bg-slate-300 dark:hover:bg-slate-600"
                        >
                          +
                        </button>
                      </div>
                    </div>
                  );
                })}
              </div>
            </div>

            <div className="flex justify-end gap-3 pt-4 border-t border-slate-200 dark:border-slate-700">
              <Button variant="outline" onClick={() => setCompleteModal(false)}>
                Отмена
              </Button>
              <Button
                icon={<CheckCircle className="w-4 h-4" />}
                onClick={handleComplete}
                loading={submitting}
              >
                Завершить
              </Button>
            </div>
          </div>
        </Modal>

        {/* Cancel Modal */}
        <Modal
          isOpen={cancelModal}
          onClose={() => setCancelModal(false)}
          title="Отменить наряд-заказ"
        >
          <div className="space-y-4">
            <div>
              <label className="block text-sm font-medium text-slate-700 dark:text-slate-200 mb-2">
                Причина отмены
              </label>
              <textarea
                value={cancelReason}
                onChange={(e) => setCancelReason(e.target.value)}
                rows={3}
                className="w-full px-3 py-2 border border-slate-300 dark:border-slate-600 rounded-lg bg-white dark:bg-slate-800 text-slate-900 dark:text-white focus:ring-2 focus:ring-red-500 focus:border-transparent"
                placeholder="Укажите причину отмены..."
              />
            </div>
            <div className="flex justify-end gap-3 pt-4 border-t border-slate-200 dark:border-slate-700">
              <Button variant="outline" onClick={() => setCancelModal(false)}>
                Назад
              </Button>
              <Button
                variant="danger"
                icon={<XCircle className="w-4 h-4" />}
                onClick={handleCancel}
                loading={submitting}
                disabled={!cancelReason.trim()}
              >
                Отменить наряд
              </Button>
            </div>
          </div>
        </Modal>
      </div>
    </PermissionGuard>
  );
};