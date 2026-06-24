import React, { useEffect, useState, useMemo, useCallback } from 'react';
import { useParams, useNavigate } from 'react-router-dom';
import { useTranslation } from 'react-i18next';
import {
  ArrowLeft, Clock, MapPin, HardDrive, User, AlertTriangle,
  Wrench, CheckCircle, XCircle, Play, ClipboardList, Package,
  Camera, FileText, Send, Loader2, Calendar,
  Timer, Pause, Square, DollarSign, Plus, Trash2,
} from 'lucide-react';
import {
  Card, CardHeader, CardBody, Badge, Button, useToast, Modal,
  SLAProgress, Timeline, FileUpload, PartCard, StatsCard,
  LiveSLATimer,
} from '../components/ui';
import { workOrdersApi, WorkOrder, PartUsage, TimeEntry, LaborCost } from '../services/workOrdersApi';
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

  // ── Time Tracking state ─────────────────────────────────────────
  const [timeEntries, setTimeEntries] = useState<TimeEntry[]>([]);
  const [elapsed, setElapsed] = useState(0);
  const [timeLoading, setTimeLoading] = useState(false);
  const [timeSubmitting, setTimeSubmitting] = useState(false);

  // ── Labor Cost state ────────────────────────────────────────────
  const [laborCost, setLaborCost] = useState<LaborCost | null>(null);
  const [costLoading, setCostLoading] = useState(false);

  // ── Parts Consumption state ─────────────────────────────────────
  const [selectedPartId, setSelectedPartId] = useState('');
  const [partQuantity, setPartQuantity] = useState(1);
  const [partsCostLoading, setPartsCostLoading] = useState(false);

  useEffect(() => {
    if (!id) return;
    const fetch = async () => {
      setLoading(true);
      try {
        const [wo, parts, entries, cost] = await Promise.all([
          workOrdersApi.getWorkOrder(id),
          sparePartsApi.getSpareParts(),
          workOrdersApi.getTimeEntries(id),
          workOrdersApi.getLaborCost(id),
        ]);
        setWorkOrder(wo);
        setSpareParts(parts || []);
        setTimeEntries(entries || []);
        setLaborCost(cost);
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

  // ── Timer interval effect ────────────────────────────────────────
  useEffect(() => {
    const running = timeEntries.find(e => e.status === 'running');
    if (running) {
      const startedAt = new Date(running.started_at).getTime();
      const tick = () => {
        setElapsed(Math.max(0, Math.floor((Date.now() - startedAt) / 1000)));
      };
      tick();
      const interval = setInterval(tick, 1000);
      return () => clearInterval(interval);
    } else {
      const paused = timeEntries.find(e => e.status === 'paused');
      if (paused) {
        setElapsed(paused.total_seconds);
      } else {
        setElapsed(0);
      }
    }
  }, [timeEntries]);

  // ── Time Tracking handlers ──────────────────────────────────────

  const handleStartTimer = async () => {
    if (!id) return;
    setTimeSubmitting(true);
    try {
      const entry = await workOrdersApi.createTimeEntry(id, {
        hourly_rate: laborCost?.hourly_rate || 0,
      });
      setTimeEntries(prev => [...prev, entry]);
      toast.success('Таймер запущен');
    } catch {
      toast.error('Ошибка при запуске таймера');
    } finally {
      setTimeSubmitting(false);
    }
  };

  const handlePauseTimer = async (entryId: string) => {
    setTimeSubmitting(true);
    try {
      const updated = await workOrdersApi.pauseTimeEntry(entryId);
      setTimeEntries(prev => prev.map(e => e.id === entryId ? updated : e));
      toast.success('Таймер поставлен на паузу');
    } catch {
      toast.error('Ошибка при паузе');
    } finally {
      setTimeSubmitting(false);
    }
  };

  const handleResumeTimer = async (entryId: string) => {
    setTimeSubmitting(true);
    try {
      const updated = await workOrdersApi.resumeTimeEntry(entryId);
      setTimeEntries(prev => prev.map(e => e.id === entryId ? updated : e));
      toast.success('Таймер возобновлён');
    } catch {
      toast.error('Ошибка при возобновлении');
    } finally {
      setTimeSubmitting(false);
    }
  };

  const handleStopTimer = async (entryId: string) => {
    setTimeSubmitting(true);
    try {
      const updated = await workOrdersApi.stopTimeEntry(entryId);
      setTimeEntries(prev => prev.map(e => e.id === entryId ? updated : e));
      // Refresh labor cost after stopping
      if (id) {
        const cost = await workOrdersApi.getLaborCost(id);
        setLaborCost(cost);
      }
      toast.success('Таймер остановлен');
    } catch {
      toast.error('Ошибка при остановке');
    } finally {
      setTimeSubmitting(false);
    }
  };

  const handleDeleteTimeEntry = async (entryId: string) => {
    setTimeSubmitting(true);
    try {
      await workOrdersApi.deleteTimeEntry(entryId);
      setTimeEntries(prev => prev.filter(e => e.id !== entryId));
      toast.success('Запись времени удалена');
    } catch {
      toast.error('Ошибка при удалении');
    } finally {
      setTimeSubmitting(false);
    }
  };

  // ── Parts Consumption handler ────────────────────────────────────

  const handleAddPartWithCost = async () => {
    if (!id || !selectedPartId) return;
    setPartsCostLoading(true);
    try {
      await workOrdersApi.addPartWithCost(id, {
        part_id: selectedPartId,
        quantity: partQuantity,
      });
      toast.success('Запчасть добавлена со стоимостью');
      setSelectedPartId('');
      setPartQuantity(1);
      // Refresh work order to show updated parts
      const updated = await workOrdersApi.getWorkOrder(id);
      setWorkOrder(updated);
    } catch {
      toast.error('Ошибка при добавлении запчасти');
    } finally {
      setPartsCostLoading(false);
    }
  };

  const formatDuration = (seconds: number): string => {
    const h = Math.floor(seconds / 3600);
    const m = Math.floor((seconds % 3600) / 60);
    const s = seconds % 60;
    return [h, m, s].map(v => String(v).padStart(2, '0')).join(':');
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

        {/* Three-Column Layout (Atlas CMMS Pattern) */}
        {/* LEFT:   Status, SLA, Assignee, Priority, Type
         CENTER: Checklist, Notes, Photos, Timeline
         RIGHT:  Asset, Location, Parts, Actions */}
        <div className="grid grid-cols-1 lg:grid-cols-12 gap-6">
          {/* ═══ LEFT COLUMN (4/12): Status + SLA + Assignee ═══ */}
          <div className="lg:col-span-4 space-y-6">
            {/* Status Card */}
            <Card>
              <CardHeader className="flex items-center gap-2">
                <div className={`w-3 h-3 rounded-full ${
                  workOrder.status === 'completed' ? 'bg-emerald-500' :
                  workOrder.status === 'in_progress' ? 'bg-blue-500' :
                  workOrder.status === 'cancelled' ? 'bg-red-500' :
                  'bg-slate-400'
                }`} />
                <span>Статус</span>
              </CardHeader>
              <CardBody>
                <div className="flex flex-wrap gap-2 mb-4">
                  <Badge variant={statusVariant[workOrder.status] || 'neutral'} size="lg">
                    {statusLabel[workOrder.status] || workOrder.status}
                  </Badge>
                  <Badge variant={priorityVariant[workOrder.priority] || 'info'} size="lg">
                    {workOrder.priority}
                  </Badge>
                  <Badge variant="neutral" size="lg">
                    <span className="flex items-center gap-1">
                      {typeIcon[workOrder.type]}
                      {typeLabel[workOrder.type] || workOrder.type}
                    </span>
                  </Badge>
                </div>

                {/* Live SLA Timer (WO-4.3.2) */}
                {workOrder.sla_deadline && (
                  <LiveSLATimer
                    deadline={workOrder.sla_deadline}
                    createdAt={workOrder.created_at}
                    status={workOrder.sla_status as 'on_track' | 'at_risk' | 'breached' | 'completed' | undefined}
                  />
                )}
              </CardBody>
            </Card>

            {/* Assignee Card */}
            <Card>
              <CardHeader className="flex items-center gap-2">
                <User className="w-5 h-5 text-slate-600 dark:text-slate-400" />
                <span>Исполнитель</span>
              </CardHeader>
              <CardBody>
                {workOrder.assignee_name ? (
                  <div className="flex items-center gap-3">
                    <div className="w-10 h-10 rounded-full bg-blue-100 dark:bg-blue-900/50 flex items-center justify-center text-blue-600 dark:text-blue-400 font-semibold text-sm">
                      {workOrder.assignee_name.charAt(0).toUpperCase()}
                    </div>
                    <div>
                      <p className="font-medium text-slate-900 dark:text-white">{workOrder.assignee_name}</p>
                      <p className="text-xs text-slate-500 dark:text-slate-400">Техник</p>
                    </div>
                  </div>
                ) : (
                  <p className="text-sm text-slate-400 dark:text-slate-500 italic">Не назначен</p>
                )}
              </CardBody>
            </Card>

            {/* Timeline */}
            <Card>
              <CardHeader className="flex items-center gap-2">
                <Clock className="w-5 h-5 text-slate-600 dark:text-slate-400" />
                <span>История</span>
              </CardHeader>
              <CardBody>
                <Timeline events={timelineEvents} />
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

          {/* ═══ CENTER COLUMN (5/12): Checklist + Notes + Photos ═══ */}
          <div className="lg:col-span-5 space-y-6">

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

          {/* ═══ RIGHT COLUMN (3/12): Asset + Location + Parts + Actions ═══ */}
          <div className="lg:col-span-3 space-y-6">
            {/* Asset Card */}
            <Card>
              <CardHeader className="flex items-center gap-2">
                <HardDrive className="w-5 h-5 text-slate-600 dark:text-slate-400" />
                <span>Устройство</span>
              </CardHeader>
              <CardBody>
                <div className="space-y-3">
                  {workOrder.device_name && (
                    <div>
                      <p className="text-xs text-slate-500 dark:text-slate-400 mb-1">Название</p>
                      <p className="text-sm font-medium text-slate-900 dark:text-white">{workOrder.device_name}</p>
                    </div>
                  )}
                  <div>
                    <p className="text-xs text-slate-500 dark:text-slate-400 mb-1">ID устройства</p>
                    <code className="text-xs font-mono bg-slate-100 dark:bg-slate-800 px-2 py-1 rounded text-slate-700 dark:text-slate-300">
                      {workOrder.device_id}
                    </code>
                  </div>
                  {workOrder.schedule_id && (
                    <div>
                      <p className="text-xs text-slate-500 dark:text-slate-400 mb-1">Расписание</p>
                      <code className="text-xs font-mono bg-slate-100 dark:bg-slate-800 px-2 py-1 rounded text-slate-700 dark:text-slate-300">
                        {workOrder.schedule_id.slice(0, 8)}...
                      </code>
                    </div>
                  )}
                </div>
              </CardBody>
            </Card>

            {/* Location Card */}
            <Card>
              <CardHeader className="flex items-center gap-2">
                <MapPin className="w-5 h-5 text-slate-600 dark:text-slate-400" />
                <span>Локация</span>
              </CardHeader>
              <CardBody>
                <p className="text-sm text-slate-400 dark:text-slate-500 italic">
                  Информация о расположении устройства
                </p>
              </CardBody>
            </Card>

            {/* ═══ Time Tracking Card ═══ */}
            <Card>
              <CardHeader className="flex items-center gap-2">
                <Timer className="w-5 h-5 text-blue-600 dark:text-blue-400" />
                <span>Учёт времени</span>
              </CardHeader>
              <CardBody>
                {/* Active Timer Display */}
                {timeEntries.some(e => e.status === 'running' || e.status === 'paused') ? (
                  <div className="mb-4 text-center">
                    <div className="text-3xl font-mono font-bold text-slate-900 dark:text-white tracking-wider">
                      {formatDuration(elapsed)}
                    </div>
                    <p className="text-xs text-slate-500 dark:text-slate-400 mt-1">
                      {timeEntries.find(e => e.status === 'running')
                        ? 'Идёт запись...'
                        : 'На паузе'}
                    </p>
                    <div className="flex justify-center gap-2 mt-3">
                      {timeEntries.find(e => e.status === 'running') ? (
                        <>
                          <Button
                            size="sm"
                            variant="secondary"
                            icon={<Pause className="w-3.5 h-3.5" />}
                            onClick={() => {
                              const running = timeEntries.find(e => e.status === 'running');
                              if (running) handlePauseTimer(running.id);
                            }}
                            loading={timeSubmitting}
                          >
                            Пауза
                          </Button>
                          <Button
                            size="sm"
                            variant="danger"
                            icon={<Square className="w-3.5 h-3.5" />}
                            onClick={() => {
                              const running = timeEntries.find(e => e.status === 'running');
                              if (running) handleStopTimer(running.id);
                            }}
                            loading={timeSubmitting}
                          >
                            Стоп
                          </Button>
                        </>
                      ) : (
                        <>
                          <Button
                            size="sm"
                            variant="primary"
                            icon={<Play className="w-3.5 h-3.5" />}
                            onClick={() => {
                              const paused = timeEntries.find(e => e.status === 'paused');
                              if (paused) handleResumeTimer(paused.id);
                            }}
                            loading={timeSubmitting}
                          >
                            Продолжить
                          </Button>
                          <Button
                            size="sm"
                            variant="danger"
                            icon={<Square className="w-3.5 h-3.5" />}
                            onClick={() => {
                              const paused = timeEntries.find(e => e.status === 'paused');
                              if (paused) handleStopTimer(paused.id);
                            }}
                            loading={timeSubmitting}
                          >
                            Стоп
                          </Button>
                        </>
                      )}
                    </div>
                  </div>
                ) : (
                  <div className="mb-4">
                    <Button
                      fullWidth
                      icon={<Play className="w-4 h-4" />}
                      onClick={handleStartTimer}
                      loading={timeSubmitting}
                      disabled={workOrder.status === 'completed' || workOrder.status === 'cancelled'}
                    >
                      Начать учёт времени
                    </Button>
                  </div>
                )}

                {/* Time Entries List */}
                {timeEntries.length > 0 && (
                  <div className="space-y-2 max-h-40 overflow-y-auto">
                    {[...timeEntries]
                      .sort((a, b) => new Date(b.started_at).getTime() - new Date(a.started_at).getTime())
                      .map(entry => (
                        <div
                          key={entry.id}
                          className="flex items-center justify-between p-2 bg-slate-50 dark:bg-slate-800/50 rounded-lg text-sm"
                        >
                          <div className="flex items-center gap-2 min-w-0">
                            <div className={`w-2 h-2 rounded-full flex-shrink-0 ${
                              entry.status === 'running' ? 'bg-green-500 animate-pulse' :
                              entry.status === 'paused' ? 'bg-yellow-500' :
                              'bg-slate-400'
                            }`} />
                            <span className="text-slate-700 dark:text-slate-300 truncate">
                              {formatDuration(entry.total_seconds)}
                            </span>
                            {entry.status === 'stopped' && (
                              <button
                                onClick={() => handleDeleteTimeEntry(entry.id)}
                                className="p-1 rounded hover:bg-red-100 dark:hover:bg-red-900/30 text-slate-400 hover:text-red-500 transition-colors flex-shrink-0"
                              >
                                <Trash2 className="w-3.5 h-3.5" />
                              </button>
                            )}
                          </div>
                          <Badge variant={
                            entry.status === 'running' ? 'success' :
                            entry.status === 'paused' ? 'warning' : 'neutral'
                          } size="sm">
                            {entry.status === 'running' ? 'Активен' :
                             entry.status === 'paused' ? 'Пауза' : 'Остановлен'}
                          </Badge>
                        </div>
                      ))}
                  </div>
                )}
              </CardBody>
            </Card>

            {/* ═══ Labor Cost Card ═══ */}
            <Card>
              <CardHeader className="flex items-center gap-2">
                <DollarSign className="w-5 h-5 text-emerald-600 dark:text-emerald-400" />
                <span>Стоимость работ</span>
              </CardHeader>
              <CardBody>
                {laborCost ? (
                  <div className="space-y-3">
                    <div className="grid grid-cols-2 gap-3">
                      <div className="p-3 bg-slate-50 dark:bg-slate-800/50 rounded-lg text-center">
                        <p className="text-xs text-slate-500 dark:text-slate-400 mb-1">Часов</p>
                        <p className="text-lg font-bold text-slate-900 dark:text-white">
                          {laborCost.total_hours.toFixed(1)}
                        </p>
                      </div>
                      <div className="p-3 bg-slate-50 dark:bg-slate-800/50 rounded-lg text-center">
                        <p className="text-xs text-slate-500 dark:text-slate-400 mb-1">Ставка</p>
                        <p className="text-lg font-bold text-slate-900 dark:text-white">
                          {laborCost.currency}{laborCost.hourly_rate.toFixed(2)}
                        </p>
                      </div>
                    </div>
                    <div className="p-3 bg-emerald-50 dark:bg-emerald-900/20 rounded-lg text-center">
                      <p className="text-xs text-emerald-600 dark:text-emerald-400 mb-1">Всего</p>
                      <p className="text-xl font-bold text-emerald-700 dark:text-emerald-300">
                        {laborCost.currency}{laborCost.total_cost.toFixed(2)}
                      </p>
                    </div>
                  </div>
                ) : (
                  <p className="text-sm text-slate-400 dark:text-slate-500 italic text-center">
                    Нет данных о стоимости
                  </p>
                )}
              </CardBody>
            </Card>

            {/* ═══ Parts Consumption Card ═══ */}
            <Card>
              <CardHeader className="flex items-center gap-2">
                <Package className="w-5 h-5 text-violet-600 dark:text-violet-400" />
                <span>Добавить запчасть (со стоимостью)</span>
              </CardHeader>
              <CardBody>
                <div className="space-y-3">
                  {/* Part Selector */}
                  <div>
                    <label className="block text-xs font-medium text-slate-500 dark:text-slate-400 mb-1">
                      Запчасть
                    </label>
                    <select
                      value={selectedPartId}
                      onChange={(e) => setSelectedPartId(e.target.value)}
                      className="w-full px-3 py-2 border border-slate-300 dark:border-slate-600 rounded-lg bg-white dark:bg-slate-800 text-slate-900 dark:text-white text-sm focus:ring-2 focus:ring-violet-500 focus:border-transparent"
                      disabled={workOrder.status === 'completed' || workOrder.status === 'cancelled'}
                    >
                      <option value="">— Выберите запчасть —</option>
                      {spareParts.map(part => (
                        <option key={part.id} value={part.id}>
                          {part.name} (SKU: {part.sku}) — {part.cost}₽
                        </option>
                      ))}
                    </select>
                  </div>

                  {/* Quantity */}
                  <div>
                    <label className="block text-xs font-medium text-slate-500 dark:text-slate-400 mb-1">
                      Количество
                    </label>
                    <div className="flex items-center gap-2">
                      <button
                        onClick={() => setPartQuantity(Math.max(1, partQuantity - 1))}
                        className="w-8 h-8 rounded-lg bg-slate-200 dark:bg-slate-700 flex items-center justify-center text-slate-600 dark:text-slate-400 hover:bg-slate-300 dark:hover:bg-slate-600 text-lg font-medium"
                        disabled={workOrder.status === 'completed' || workOrder.status === 'cancelled'}
                      >
                        −
                      </button>
                      <input
                        type="number"
                        min={1}
                        value={partQuantity}
                        onChange={(e) => setPartQuantity(Math.max(1, parseInt(e.target.value) || 1))}
                        className="w-16 text-center px-2 py-1.5 border border-slate-300 dark:border-slate-600 rounded-lg bg-white dark:bg-slate-800 text-slate-900 dark:text-white text-sm focus:ring-2 focus:ring-violet-500 focus:border-transparent"
                        disabled={workOrder.status === 'completed' || workOrder.status === 'cancelled'}
                      />
                      <button
                        onClick={() => setPartQuantity(partQuantity + 1)}
                        className="w-8 h-8 rounded-lg bg-slate-200 dark:bg-slate-700 flex items-center justify-center text-slate-600 dark:text-slate-400 hover:bg-slate-300 dark:hover:bg-slate-600 text-lg font-medium"
                        disabled={workOrder.status === 'completed' || workOrder.status === 'cancelled'}
                      >
                        +
                      </button>
                    </div>
                  </div>

                  {/* Cost Preview */}
                  {selectedPartId && (() => {
                    const part = spareParts.find(p => p.id === selectedPartId);
                    return part ? (
                      <div className="p-2 bg-slate-50 dark:bg-slate-800/50 rounded-lg text-sm">
                        <span className="text-slate-500 dark:text-slate-400">Стоимость: </span>
                        <span className="font-medium text-slate-900 dark:text-white">
                          {part.cost}₽ × {partQuantity} = {part.cost * partQuantity}₽
                        </span>
                      </div>
                    ) : null;
                  })()}

                  <Button
                    fullWidth
                    variant="primary"
                    icon={<Plus className="w-4 h-4" />}
                    onClick={handleAddPartWithCost}
                    loading={partsCostLoading}
                    disabled={!selectedPartId || workOrder.status === 'completed' || workOrder.status === 'cancelled'}
                  >
                    Добавить со стоимостью
                  </Button>
                </div>
              </CardBody>
            </Card>

            {/* ═══ Total Cost Dashboard (WO-4.4.5) ═══ */}
            <Card>
              <CardHeader className="flex items-center gap-2">
                <DollarSign className="w-5 h-5 text-blue-600 dark:text-blue-400" />
                <span>Total Cost Dashboard</span>
              </CardHeader>
              <CardBody>
                <div className="grid grid-cols-3 gap-3">
                  <StatsCard
                    title="Total Cost"
                    value={workOrder.total_cost ? `${workOrder.total_cost.toFixed(2)} ₽` : '—'}
                    icon={DollarSign}
                    iconColor="text-blue-600"
                    iconBgColor="bg-blue-50"
                  />
                  <StatsCard
                    title="Labor Cost"
                    value={workOrder.total_labor_cost ? `${workOrder.total_labor_cost.toFixed(2)} ₽` : '—'}
                    icon={DollarSign}
                    iconColor="text-emerald-600"
                    iconBgColor="bg-emerald-50"
                  />
                  <StatsCard
                    title="Parts Cost"
                    value={workOrder.total_parts_cost ? `${workOrder.total_parts_cost.toFixed(2)} ₽` : '—'}
                    icon={DollarSign}
                    iconColor="text-amber-600"
                    iconBgColor="bg-amber-50"
                  />
                </div>
              </CardBody>
            </Card>

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
            <Card>
              <CardHeader className="flex items-center gap-2">
                <Package className="w-5 h-5 text-slate-600 dark:text-slate-400" />
                <span>Запчасти ({workOrder.parts_used?.length || 0})</span>
              </CardHeader>
              <CardBody>
                {workOrder.parts_used?.length > 0 ? (
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
                ) : (
                  <p className="text-sm text-slate-400 dark:text-slate-500 italic">
                    Запчасти не использованы
                  </p>
                )}
              </CardBody>
            </Card>
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