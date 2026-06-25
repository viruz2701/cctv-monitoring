import React, { useEffect, useState, useMemo, useCallback } from 'react';
import { useParams, useNavigate } from 'react-router-dom';
import { useTranslation } from 'react-i18next';
import {
  Info, Package, Timer, Camera, Clock, Loader2,
  CheckCircle, XCircle, ArrowLeft,
} from 'lucide-react';
import {
  Button, useToast, Modal,
  FileUpload, Tabs,
} from '../components/ui';
import { workOrdersApi, WorkOrder, PartUsage, TimeEntry, LaborCost } from '../services/workOrdersApi';
import { sparePartsApi, SparePart } from '../services/sparePartsApi';
import { useWorkOrders } from '../context/WorkOrdersContext';
import { PermissionGuard } from '../components/auth/PermissionGuard';
import { DragDropContext, Droppable, Draggable, DropResult } from '@hello-pangea/dnd';
import { WODetailHeader } from '../components/work-orders/WODetailHeader';
import { WODetailInfo } from '../components/work-orders/WODetailInfo';
import { WODetailParts } from '../components/work-orders/WODetailParts';
import { WODetailTime } from '../components/work-orders/WODetailTime';
import { WODetailPhotos } from '../components/work-orders/WODetailPhotos';
import { WODetailTimeline } from '../components/work-orders/WODetailTimeline';

const typeLabel: Record<string, string> = {
  preventive: 'Плановое',
  corrective: 'Корректирующее',
  emergency: 'Аварийное',
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

  // ── Tabs state ───────────────────────────────────────────────────
  const [activeTab, setActiveTab] = useState('info');
  const [loadedTabs, setLoadedTabs] = useState<Set<string>>(new Set(['info']));

  const handleTabChange = useCallback((tabId: string) => {
    setActiveTab(tabId);
    setLoadedTabs(prev => new Set(prev).add(tabId));
  }, []);

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

  // ── Tabs configuration ──────────────────────────────────────────
  const partsCount = workOrder?.parts_used?.length || 0;

  const tabs = [
    {
      id: 'info',
      label: t('workOrder.tabInfo') || 'Информация',
      icon: <Info className="w-4 h-4" />,
    },
    {
      id: 'parts',
      label: t('workOrder.tabParts') || 'Запчасти',
      icon: <Package className="w-4 h-4" />,
      badge: partsCount > 0 ? partsCount : undefined,
    },
    {
      id: 'time',
      label: t('workOrder.tabTime') || 'Время и трудозатраты',
      icon: <Timer className="w-4 h-4" />,
    },
    {
      id: 'photos',
      label: t('workOrder.tabPhotos') || 'Фото',
      icon: <Camera className="w-4 h-4" />,
    },
    {
      id: 'history',
      label: t('workOrder.tabHistory') || 'История',
      icon: <Clock className="w-4 h-4" />,
    },
  ];

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
      <div className="p-6 max-w-6xl mx-auto">
        {/* ═══ HEADER (Sticky, always visible) ═══ */}
        <WODetailHeader
          workOrder={workOrder}
          submitting={submitting}
          onStart={handleStart}
          onComplete={() => setCompleteModal(true)}
          onCancel={() => setCancelModal(true)}
          onBack={() => navigate('/work-orders')}
        />

        {/* ═══ TABS NAVIGATION ═══ */}
        <div className="mt-6">
          <Tabs
            tabs={tabs}
            activeTab={activeTab}
            onChange={handleTabChange}
            variant="default"
          >
            {/* Info Tab */}
            {activeTab === 'info' && (
              <div className="mt-6">
                <WODetailInfo workOrder={workOrder} />

                {/* Checklist with DragDrop (сохраняем DragDropContext) */}
                {workOrder.checklist?.length > 0 && (
                  <div className="mt-6">
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
                  </div>
                )}
              </div>
            )}

            {/* Parts Tab (lazy load) */}
            {activeTab === 'parts' && loadedTabs.has('parts') && (
              <div className="mt-6">
                <WODetailParts
                  workOrder={workOrder}
                  spareParts={spareParts}
                  selectedPartId={selectedPartId}
                  partQuantity={partQuantity}
                  partsCostLoading={partsCostLoading}
                  onSelectedPartIdChange={setSelectedPartId}
                  onPartQuantityChange={setPartQuantity}
                  onAddPart={handleAddPartWithCost}
                />
              </div>
            )}

            {/* Time Tab (lazy load) */}
            {activeTab === 'time' && loadedTabs.has('time') && (
              <div className="mt-6">
                <WODetailTime
                  timeEntries={timeEntries}
                  laborCost={laborCost}
                  elapsed={elapsed}
                  timeSubmitting={timeSubmitting}
                  workOrderStatus={workOrder.status}
                  totalCost={workOrder.total_cost}
                  totalLaborCost={workOrder.total_labor_cost}
                  totalPartsCost={workOrder.total_parts_cost}
                  onStartTimer={handleStartTimer}
                  onPauseTimer={handlePauseTimer}
                  onResumeTimer={handleResumeTimer}
                  onStopTimer={handleStopTimer}
                  onDeleteTimeEntry={handleDeleteTimeEntry}
                  formatDuration={formatDuration}
                />
              </div>
            )}

            {/* Photos Tab (lazy load) */}
            {activeTab === 'photos' && loadedTabs.has('photos') && (
              <div className="mt-6">
                <WODetailPhotos workOrder={workOrder} />
              </div>
            )}

            {/* History Tab (lazy load) */}
            {activeTab === 'history' && loadedTabs.has('history') && (
              <div className="mt-6">
                <WODetailTimeline events={timelineEvents} />
              </div>
            )}
          </Tabs>
        </div>

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
      </div>
    </PermissionGuard>
  );
};
