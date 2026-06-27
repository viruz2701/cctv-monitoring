import React, { useEffect, useState, useMemo, useCallback } from 'react';
import { useParams, useNavigate } from 'react-router-dom';
import { useTranslation } from 'react-i18next';
import { Breadcrumbs } from '../components/ui/Breadcrumbs';
import {
  Info, Package, Timer, Camera, Clock,
  CheckCircle, XCircle, ArrowLeft, Shield,
  User, HardDrive, MapPin, Calendar, FileText,
  AlertTriangle, Wrench, DollarSign, ClipboardList,
  Play, Square, Users,
} from 'lucide-react';
import {
  Button, useToast, Modal,
  FileUpload, Card, CardHeader, CardBody, Badge, SLAProgress,
} from '../components/ui';
import { workOrdersApi, WorkOrder, PartUsage, TimeEntry, LaborCost } from '../services/workOrdersApi';
import { sparePartsApi, SparePart } from '../services/sparePartsApi';
import { queryKeys } from '../hooks/useApiQuery';
import { useQueryClient } from '@tanstack/react-query';
import { PermissionGuard } from '../components/auth/PermissionGuard';
import { DragDropContext, Droppable, Draggable, DropResult } from '@hello-pangea/dnd';
import { ThreeColumnTemplate, SkeletonDetailPage } from '../components/layout';
import { SLATimer } from '../components/work-orders/SLATimer';
import { WODetailHeader } from '../components/work-orders/WODetailHeader';
import { WODetailInfo } from '../components/work-orders/WODetailInfo';
import { WODetailParts } from '../components/work-orders/WODetailParts';
import { WODetailTime } from '../components/work-orders/WODetailTime';
import { WODetailPhotos } from '../components/work-orders/WODetailPhotos';
import { WODetailTimeline } from '../components/work-orders/WODetailTimeline';
import { WOAuditLog } from '../components/work-orders/WOAuditLog';

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

export const WorkOrderDetail: React.FC = () => {
  const { t } = useTranslation();
  const { id } = useParams<{ id: string }>();
  const navigate = useNavigate();
  const toast = useToast();
  const queryClient = useQueryClient();

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

  // ── P3-3.3: Real-time Collaboration — presence ──────────────────
  const [activeUsers, setActiveUsers] = useState<{id: string; name: string; action: string}[]>([]);

  useEffect(() => {
    // Заглушка — в будущем WebSocket подключение
    // const ws = new WebSocket(`ws://localhost:8080/ws/work-order/${id}`);
    // ws.onmessage = (e) => setActiveUsers(JSON.parse(e.data));
    return () => {};
  }, [id]);

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
      queryClient.invalidateQueries({ queryKey: queryKeys.workOrders.all });
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
      queryClient.invalidateQueries({ queryKey: queryKeys.workOrders.all });
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

  const isEditable = workOrder?.status === 'open' || workOrder?.status === 'in_progress';

  // ── Loading state ──────────────────────────────────────────────────

  if (loading) {
    return <SkeletonDetailPage />;
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

  // ── Left Column: Metadata, SLA, Timeline ─────────────────────────

  const leftColumn = (
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

      {/* Timeline */}
      {timelineEvents.length > 0 && (
        <WODetailTimeline events={timelineEvents} />
      )}
    </>
  );

  // ── Center Column: Checklist, Notes, Photos, Audit ───────────────

  const centerColumn = (
    <>
      {/* Checklist with DragDrop */}
      {workOrder.checklist?.length > 0 && (
        <Card>
          <CardHeader className="flex items-center gap-2">
            <ClipboardList className="w-5 h-5 text-blue-600 dark:text-blue-400" />
            <span>
              Чеклист ({workOrder.checklist.filter(c => c.completed).length}/{workOrder.checklist.length})
            </span>
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
                              if (isEditable) {
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

      {/* Notes / Description */}
      {workOrder.notes && (
        <Card>
          <CardHeader className="flex items-center gap-2">
            <FileText className="w-5 h-5 text-slate-600 dark:text-slate-400" />
            <span className="text-sm font-medium">Заметки</span>
          </CardHeader>
          <CardBody>
            <p className="text-sm text-slate-700 dark:text-slate-300 whitespace-pre-wrap">
              {workOrder.notes}
            </p>
          </CardBody>
        </Card>
      )}

      {/* Photos */}
      <WODetailPhotos workOrder={workOrder} />

      {/* Audit Log */}
      {id && <WOAuditLog workOrderId={id} />}
    </>
  );

  // ── Right Column: Device Info, Parts, Labor Cost ────────────────

  const rightColumn = (
    <>
      {/* Device Info */}
      <Card>
        <CardHeader className="flex items-center gap-2">
          <HardDrive className="w-5 h-5 text-slate-600 dark:text-slate-400" />
          <span className="text-sm font-medium">Устройство</span>
        </CardHeader>
        <CardBody>
          <div className="space-y-3">
            {workOrder.device_name && (
              <div>
                <p className="text-xs text-slate-500 dark:text-slate-400 mb-0.5">Название</p>
                <p className="text-sm font-medium text-slate-900 dark:text-white">{workOrder.device_name}</p>
              </div>
            )}
            <div>
              <p className="text-xs text-slate-500 dark:text-slate-400 mb-0.5">ID устройства</p>
              <code className="text-xs font-mono bg-slate-100 dark:bg-slate-800 px-2 py-1 rounded text-slate-700 dark:text-slate-300">
                {workOrder.device_id}
              </code>
            </div>
            {workOrder.schedule_id && (
              <div>
                <p className="text-xs text-slate-500 dark:text-slate-400 mb-0.5">Расписание</p>
                <code className="text-xs font-mono bg-slate-100 dark:bg-slate-800 px-2 py-1 rounded text-slate-700 dark:text-slate-300">
                  {workOrder.schedule_id.slice(0, 8)}...
                </code>
              </div>
            )}
          </div>
        </CardBody>
      </Card>

      {/* Dates */}
      <Card>
        <CardHeader className="flex items-center gap-2">
          <Calendar className="w-5 h-5 text-slate-600 dark:text-slate-400" />
          <span className="text-sm font-medium">Даты</span>
        </CardHeader>
        <CardBody>
          <div className="space-y-2 text-sm">
            <div className="flex justify-between">
              <span className="text-slate-500 dark:text-slate-400">Создан</span>
              <span className="text-slate-900 dark:text-white">{new Date(workOrder.created_at).toLocaleString()}</span>
            </div>
            {workOrder.started_at && (
              <div className="flex justify-between">
                <span className="text-slate-500 dark:text-slate-400">Начат</span>
                <span className="text-slate-900 dark:text-white">{new Date(workOrder.started_at).toLocaleString()}</span>
              </div>
            )}
            {workOrder.completed_at && (
              <div className="flex justify-between">
                <span className="text-slate-500 dark:text-slate-400">Завершён</span>
                <span className="text-slate-900 dark:text-white">{new Date(workOrder.completed_at).toLocaleString()}</span>
              </div>
            )}
          </div>
        </CardBody>
      </Card>

      {/* Parts Used (compact) */}
      <Card>
        <CardHeader className="flex items-center gap-2">
          <Package className="w-5 h-5 text-slate-600 dark:text-slate-400" />
          <span className="text-sm font-medium">
            Запчасти ({workOrder.parts_used?.length || 0})
          </span>
        </CardHeader>
        <CardBody>
          {workOrder.parts_used?.length > 0 ? (
            <div className="space-y-2">
              {workOrder.parts_used.map((p, i) => {
                const part = spareParts.find(sp => sp.id === p.part_id);
                return (
                  <div key={i} className="flex justify-between items-center p-2 bg-slate-50 dark:bg-slate-800/50 rounded-lg text-sm">
                    <span className="text-slate-700 dark:text-slate-300 truncate mr-2">
                      {part?.name || p.part_id}
                    </span>
                    <div className="flex items-center gap-2 flex-shrink-0">
                      <Badge variant="info" size="sm">{p.quantity} шт</Badge>
                      {part?.cost != null && (
                        <span className="text-xs text-slate-500">{part.cost * p.quantity}₽</span>
                      )}
                    </div>
                  </div>
                );
              })}
            </div>
          ) : (
            <p className="text-sm text-slate-400 dark:text-slate-500 italic text-center py-2">Нет запчастей</p>
          )}
          {workOrder.total_parts_cost != null && (
            <div className="mt-3 pt-3 border-t border-slate-200 dark:border-slate-700 flex justify-between text-sm">
              <span className="text-slate-500 dark:text-slate-400">Стоимость запчастей</span>
              <span className="font-medium text-slate-900 dark:text-white">{workOrder.total_parts_cost.toFixed(2)} ₽</span>
            </div>
          )}
        </CardBody>
      </Card>

      {/* Labor Cost (compact) */}
      <Card>
        <CardHeader className="flex items-center gap-2">
          <DollarSign className="w-5 h-5 text-emerald-600 dark:text-emerald-400" />
          <span className="text-sm font-medium">Стоимость работ</span>
        </CardHeader>
        <CardBody>
          {laborCost ? (
            <div className="space-y-2">
              <div className="flex justify-between text-sm">
                <span className="text-slate-500 dark:text-slate-400">Часов</span>
                <span className="text-slate-900 dark:text-white font-medium">{laborCost.total_hours.toFixed(1)}</span>
              </div>
              <div className="flex justify-between text-sm">
                <span className="text-slate-500 dark:text-slate-400">Ставка</span>
                <span className="text-slate-900 dark:text-white font-medium">{laborCost.currency}{laborCost.hourly_rate.toFixed(2)}</span>
              </div>
              <div className="pt-2 border-t border-slate-200 dark:border-slate-700 flex justify-between text-sm font-semibold">
                <span className="text-emerald-600 dark:text-emerald-400">Всего</span>
                <span className="text-emerald-700 dark:text-emerald-300">{laborCost.currency}{laborCost.total_cost.toFixed(2)}</span>
              </div>
            </div>
          ) : (
            <p className="text-sm text-slate-400 dark:text-slate-500 italic text-center py-2">Нет данных</p>
          )}
        </CardBody>
      </Card>

      {/* Time Tracking (compact) */}
      <Card>
        <CardHeader className="flex items-center gap-2">
          <Timer className="w-5 h-5 text-blue-600 dark:text-blue-400" />
          <span className="text-sm font-medium">Учёт времени</span>
        </CardHeader>
        <CardBody>
          {timeEntries.find(e => e.status === 'running' || e.status === 'paused') ? (
            <div className="text-center">
              <div className="text-2xl font-mono font-bold text-slate-900 dark:text-white tracking-wider">
                {formatDuration(elapsed)}
              </div>
              <div className="flex justify-center gap-2 mt-3">
                {timeEntries.find(e => e.status === 'running') ? (
                  <>
                    <Button
                      size="sm"
                      variant="secondary"
                      icon={<Timer className="w-3.5 h-3.5" />}
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
                      icon={<XCircle className="w-3.5 h-3.5" />}
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
            <Button
              fullWidth
              size="sm"
              icon={<Play className="w-4 h-4" />}
              onClick={handleStartTimer}
              loading={timeSubmitting}
              disabled={!isEditable}
            >
              Начать учёт времени
            </Button>
          )}

          {/* Recent entries */}
          {timeEntries.length > 0 && (
            <div className="mt-3 space-y-1.5">
              {[...timeEntries]
                .sort((a, b) => new Date(b.started_at).getTime() - new Date(a.started_at).getTime())
                .slice(0, 3)
                .map(entry => (
                  <div key={entry.id} className="flex items-center justify-between p-2 bg-slate-50 dark:bg-slate-800/50 rounded text-xs">
                    <span className="font-mono text-slate-600 dark:text-slate-400">
                      {formatDuration(entry.total_seconds)}
                    </span>
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
    </>
  );

  // ── Breadcrumb items ────────────────────────────────────────────────
  const breadcrumbItems = useMemo(() => {
    if (!workOrder) return [{ label: 'nav_work_orders', href: '/work-orders' }];
    return [
      { label: 'nav_work_orders', href: '/work-orders' },
      { label: `WO-${workOrder.id.slice(0, 8)}`, href: undefined },
    ];
  }, [workOrder]);

  // ── Render ────────────────────────────────────────────────────────

  return (
    <PermissionGuard requiredRole={['admin', 'manager', 'technician']}>
      <Breadcrumbs items={breadcrumbItems} className="mb-4 px-6 pt-4" />
      <ThreeColumnTemplate
        header={
          <>
            <WODetailHeader
              workOrder={workOrder}
              submitting={submitting}
              onStart={handleStart}
              onComplete={() => setCompleteModal(true)}
              onCancel={() => setCancelModal(true)}
              onBack={() => navigate('/work-orders')}
            />
            {activeUsers.length > 0 && (
              <div className="flex items-center gap-2 text-xs text-slate-500 bg-slate-100 dark:bg-slate-800 rounded-lg px-3 py-1.5">
                <Users className="w-3.5 h-3.5" />
                {activeUsers.map(u => u.action === 'editing' ? (
                  <span key={u.id} className="text-amber-600 font-medium">{u.name} editing...</span>
                ) : (
                  <span key={u.id}>{u.name}</span>
                ))}
              </div>
            )}
          </>
        }
        left={leftColumn}
        center={centerColumn}
        right={rightColumn}
        leftHeader="Метаданные"
        centerHeader="Основное"
        rightHeader="Детали"
      />

      {/* ═══ MODALS ═══ */}

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
    </PermissionGuard>
  );
};
