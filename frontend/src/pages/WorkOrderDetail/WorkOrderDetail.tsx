// ═══════════════════════════════════════════════════════════════════════
// WorkOrderDetail — основной компонент детального просмотра наряд-заказа
// ≈300 строк (было 985). Вспомогательные компоненты вынесены в:
//   WOStatusCards, WOPartsSection, WOTimeTracking, WOModals
// ═══════════════════════════════════════════════════════════════════════

import React, { useEffect, useState, useMemo, useCallback } from 'react';
import { useParams, useNavigate } from 'react-router-dom';
import { useUnsavedChanges } from '../../hooks/useUnsavedChanges';
import { useTranslation } from 'react-i18next';
import { useQueryClient } from '@tanstack/react-query';
import {
  Users, FileText, ClipboardList, HardDrive, Calendar,
  CheckCircle, ArrowLeft,
} from '../../components/ui/Icons';
import { Breadcrumbs } from '../../components/ui/Breadcrumbs';
import {
  Button, useToast, Card, CardHeader, CardBody,
} from '../../components/ui';
import { workOrdersApi, type WorkOrder, type PartUsage, type TimeEntry, type LaborCost } from '../../services/workOrdersApi';
import { sparePartsApi, type SparePart } from '../../services/sparePartsApi';
import { queryKeys } from '../../hooks/useApiQuery';
import { PermissionGuard } from '../../components/auth/PermissionGuard';
import { DragDropContext, Droppable, Draggable, type DropResult } from '@hello-pangea/dnd';
import { ThreeColumnTemplate, SkeletonDetailPage } from '../../components/layout';
import { WODetailTimeline } from '../../components/work-orders/WODetailTimeline';
import { WODetailHeader } from '../../components/work-orders/WODetailHeader';
import { WODetailPhotos } from '../../components/work-orders/WODetailPhotos';
import { WOAuditLog } from '../../components/work-orders/WOAuditLog';
import { WOStatusCards } from './WOStatusCards';
import { WOPartsSection } from './WOPartsSection';
import { WOTimeTracking } from './WOTimeTracking';
import { WOModals } from './WOModals';

const typeLabel: Record<string, string> = {
  preventive: 'Плановое',
  corrective: 'Корректирующее',
  emergency: 'Аварийное',
};

const formatDuration = (seconds: number): string => {
  const h = Math.floor(seconds / 3600);
  const m = Math.floor((seconds % 3600) / 60);
  const s = seconds % 60;
  return [h, m, s].map(v => String(v).padStart(2, '0')).join(':');
};

export const WorkOrderDetail: React.FC = () => {
  const { t } = useTranslation();
  const { id } = useParams<{ id: string }>();
  const navigate = useNavigate();
  const toast = useToast();
  const queryClient = useQueryClient();

  // ── State ──────────────────────────────────────────────────────────
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
  const [timeEntries, setTimeEntries] = useState<TimeEntry[]>([]);
  const [elapsed, setElapsed] = useState(0);
  const [timeSubmitting, setTimeSubmitting] = useState(false);
  const [laborCost, setLaborCost] = useState<LaborCost | null>(null);
  const [activeUsers] = useState<{id: string; name: string; action: string}[]>([]);
 
  // ── Unsaved changes guard (P0-CR-12) ─────────────────────────────
  const isFormDirty = useMemo(() => {
  	return completeNotes !== '' ||
  		completePhotos.length > 0 ||
  		completeParts.length > 0 ||
  		cancelReason !== '';
  }, [completeNotes, completePhotos, completeParts, cancelReason]);
 
  useUnsavedChanges(isFormDirty);

  // ── Data Fetching ──────────────────────────────────────────────────
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
      } catch {
        toast.error('Не удалось загрузить наряд-заказ');
      } finally {
        setLoading(false);
      }
    };
    fetch();
  }, [id]);

  // ── Timer interval effect ─────────────────────────────────────────
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
      setElapsed(paused ? paused.total_seconds : 0);
    }
  }, [timeEntries]);

  // ── Handlers ───────────────────────────────────────────────────────
  const handleStart = useCallback(async () => {
    if (!id) return;
    setSubmitting(true);
    try {
      await workOrdersApi.startWorkOrder(id);
      setWorkOrder(await workOrdersApi.getWorkOrder(id));
      toast.success('Наряд-заказ начат');
    } catch {
      toast.error('Ошибка при старте');
    } finally {
      setSubmitting(false);
    }
  }, [id]);

  const handleComplete = useCallback(async () => {
    if (!id) return;
    setSubmitting(true);
    try {
      await workOrdersApi.completeWorkOrder(id, completeNotes, completePhotos, completeParts);
      setWorkOrder(await workOrdersApi.getWorkOrder(id));
      setCompleteModal(false);
      queryClient.invalidateQueries({ queryKey: queryKeys.workOrders.all });
      toast.success('Наряд-заказ завершён');
    } catch {
      toast.error('Ошибка при завершении');
    } finally {
      setSubmitting(false);
    }
  }, [id, completeNotes, completePhotos, completeParts]);

  const handleCancel = useCallback(async () => {
    if (!id) return;
    setSubmitting(true);
    try {
      await workOrdersApi.cancelWorkOrder(id, cancelReason);
      setWorkOrder(await workOrdersApi.getWorkOrder(id));
      setCancelModal(false);
      queryClient.invalidateQueries({ queryKey: queryKeys.workOrders.all });
      toast.success('Наряд-заказ отменён');
    } catch {
      toast.error('Ошибка при отмене');
    } finally {
      setSubmitting(false);
    }
  }, [id, cancelReason]);

  const handleStartTimer = useCallback(async () => {
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
  }, [id, laborCost]);

  const handlePauseTimer = useCallback(async (entryId: string) => {
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
  }, []);

  const handleResumeTimer = useCallback(async (entryId: string) => {
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
  }, []);

  const handleStopTimer = useCallback(async (entryId: string) => {
    setTimeSubmitting(true);
    try {
      const updated = await workOrdersApi.stopTimeEntry(entryId);
      setTimeEntries(prev => prev.map(e => e.id === entryId ? updated : e));
      if (id) {
        setLaborCost(await workOrdersApi.getLaborCost(id));
      }
      toast.success('Таймер остановлен');
    } catch {
      toast.error('Ошибка при остановке');
    } finally {
      setTimeSubmitting(false);
    }
  }, [id]);

  const handleFileUpload = useCallback(async (files: File[]): Promise<void> => {
    if (!id) return;
    try {
      const result = await workOrdersApi.uploadPhotos(id, files);
      setCompletePhotos(prev => [...prev, ...(result.photos || [])]);
      toast.success(`${files.length} файлов загружено`);
    } catch {
      toast.error('Ошибка загрузки');
    }
  }, [id]);

  const togglePartUsage = useCallback((partId: string, quantity: number) => {
    setCompleteParts(prev => {
      const existing = prev.find(p => p.part_id === partId);
      if (existing) {
        return quantity === 0
          ? prev.filter(p => p.part_id !== partId)
          : prev.map(p => p.part_id === partId ? { ...p, quantity } : p);
      }
      return [...prev, { part_id: partId, quantity }];
    });
  }, []);

  // ── Derived Data ──────────────────────────────────────────────────
  const timelineEvents = useMemo(() => {
    if (!workOrder) return [];
    const events: Array<{
      id: string; timestamp: string; type: 'system' | 'status_change' | 'part';
      title: string; description?: string; user?: string;
    }> = [];
    events.push({
      id: 'created', timestamp: workOrder.created_at,
      type: 'system', title: 'Наряд-заказ создан',
      description: `Тип: ${typeLabel[workOrder.type] || workOrder.type}, Приоритет: ${workOrder.priority}`,
      user: workOrder.created_by,
    });
    if (workOrder.started_at) {
      events.push({ id: 'started', timestamp: workOrder.started_at, type: 'status_change', title: 'Работа начата', user: workOrder.assignee_name });
    }
    if (workOrder.completed_at) {
      events.push({ id: 'completed', timestamp: workOrder.completed_at, type: 'status_change', title: 'Работа завершена', user: workOrder.assignee_name });
    }
    if (workOrder.parts_used?.length > 0) {
      workOrder.parts_used.forEach((p, i) => {
        events.push({ id: `part-${i}`, timestamp: workOrder.completed_at || workOrder.updated_at, type: 'part', title: `Запчасть использована: ${p.part_id}`, description: `Количество: ${p.quantity}` });
      });
    }
    return events.sort((a, b) => new Date(b.timestamp).getTime() - new Date(a.timestamp).getTime());
  }, [workOrder]);

  const isEditable = workOrder?.status === 'open' || workOrder?.status === 'in_progress';

  // ── Loading / Error states ────────────────────────────────────────
  if (loading) return <SkeletonDetailPage />;

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

  // ── Column Content ────────────────────────────────────────────────
  const leftColumn = (
    <>
      <WOStatusCards workOrder={workOrder} />
      {timelineEvents.length > 0 && <WODetailTimeline events={timelineEvents} />}
    </>
  );

  const centerColumn = (
    <>
      {/* Checklist with DragDrop */}
      {workOrder.checklist?.length > 0 && (
        <Card>
          <CardHeader className="flex items-center gap-2">
            <ClipboardList className="w-5 h-5 text-blue-600 dark:text-blue-400" />
            <span>Чеклист ({workOrder.checklist.filter(c => c.completed).length}/{workOrder.checklist.length})</span>
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
                  <div {...provided.droppableProps} ref={provided.innerRef} className="space-y-2">
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
                            } ${snapshot.isDragging ? 'rotate-2' : ''}`}
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

      {/* Notes */}
      {workOrder.notes && (
        <Card>
          <CardHeader className="flex items-center gap-2">
            <FileText className="w-5 h-5 text-slate-600 dark:text-slate-400" />
            <span className="text-sm font-medium">Заметки</span>
          </CardHeader>
          <CardBody>
            <p className="text-sm text-slate-700 dark:text-slate-300 whitespace-pre-wrap">{workOrder.notes}</p>
          </CardBody>
        </Card>
      )}

      <WODetailPhotos workOrder={workOrder} />
      {id && <WOAuditLog workOrderId={id} />}
    </>
  );

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

      <WOPartsSection workOrder={workOrder} spareParts={spareParts} laborCost={laborCost} />
      <WOTimeTracking
        timeEntries={timeEntries}
        elapsed={elapsed}
        timeSubmitting={timeSubmitting}
        isEditable={isEditable}
        onStartTimer={handleStartTimer}
        onPauseTimer={handlePauseTimer}
        onResumeTimer={handleResumeTimer}
        onStopTimer={handleStopTimer}
        formatDuration={formatDuration}
      />
    </>
  );

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

      <WOModals
        completeModal={completeModal}
        cancelModal={cancelModal}
        completeNotes={completeNotes}
        cancelReason={cancelReason}
        completePhotos={completePhotos}
        completeParts={completeParts}
        spareParts={spareParts}
        submitting={submitting}
        onCompleteClose={() => setCompleteModal(false)}
        onCancelClose={() => setCancelModal(false)}
        onCompleteNotesChange={setCompleteNotes}
        onCancelReasonChange={setCancelReason}
        onFileUpload={handleFileUpload}
        onRemovePhoto={(i) => setCompletePhotos(prev => prev.filter((_, j) => j !== i))}
        onTogglePart={togglePartUsage}
        onComplete={handleComplete}
        onCancel={handleCancel}
      />
    </PermissionGuard>
  );
};
