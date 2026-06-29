// ═══════════════════════════════════════════════════════════════════════
// Webhooks — страница управления вебхуками (P2-3.1)
//
// Режимы:
//   - list: список всех вебхуков
//   - builder: визуальный конструктор (создание/редактирование)
//
// Compliance:
//   - OWASP ASVS V5 (Input validation через Zod)
//   - OWASP ASVS V7 (Error handling — toast + traceID)
// ═══════════════════════════════════════════════════════════════════════

import React, { useState } from 'react';
import { useTranslation } from 'react-i18next';
import {
  Webhook, Plus, Trash2, Play, RefreshCw,
  CheckCircle, XCircle, Clock, Settings,
} from '../components/ui/Icons';
import { Card, Button, Badge, useToast, EmptyState } from '../components/ui';
import { useConfirmAction } from '../hooks/useConfirmAction';
import { WebhookBuilder } from '../components/webhooks/WebhookBuilder';
import {
  useWebhooks,
  useDeleteWebhook,
  useTestWebhook,
  webhookKeys,
} from '../hooks/useWebhooks';
import { useQueryClient } from '@tanstack/react-query';
import type { WebhookEndpoint } from '../services/api';

// ═══════════════════════════════════════════════════════════════════════
// Types
// ═══════════════════════════════════════════════════════════════════════

type ViewMode = 'list' | 'builder';

// ═══════════════════════════════════════════════════════════════════════
// EVENTS (shared with builder for consistency)
// ═══════════════════════════════════════════════════════════════════════

export const EVENT_OPTIONS = [
  { value: 'work_order.created', label: 'WO Created' },
  { value: 'work_order.updated', label: 'WO Updated' },
  { value: 'work_order.completed', label: 'WO Completed' },
  { value: 'work_order.cancelled', label: 'WO Cancelled' },
  { value: 'alarm.created', label: 'Alarm Created' },
  { value: 'alarm.resolved', label: 'Alarm Resolved' },
  { value: 'device.offline', label: 'Device Offline' },
  { value: 'device.online', label: 'Device Online' },
  { value: 'device.status_changed', label: 'Device Status Changed' },
  { value: 'sla.breached', label: 'SLA Breached' },
  { value: 'sla.at_risk', label: 'SLA At Risk' },
];

// ═══════════════════════════════════════════════════════════════════════
// Component
// ═══════════════════════════════════════════════════════════════════════

export function Webhooks() {
  const { t } = useTranslation();
  const toast = useToast();
  const { confirm, ConfirmDialog } = useConfirmAction();
  const queryClient = useQueryClient();

  // ─── State ─────────────────────────────────────────────────────────
  const [viewMode, setViewMode] = useState<ViewMode>('list');
  const [editingId, setEditingId] = useState<string | undefined>(undefined);
  const [testing, setTesting] = useState<string | null>(null);

  // ─── Data ──────────────────────────────────────────────────────────
  const { data: webhooks, isLoading, error } = useWebhooks();
  const deleteMutation = useDeleteWebhook();
  const testMutation = useTestWebhook();

  // ─── Handlers ──────────────────────────────────────────────────────

  const openCreate = () => {
    setEditingId(undefined);
    setViewMode('builder');
  };

  const openEdit = (wh: WebhookEndpoint) => {
    setEditingId(wh.id);
    setViewMode('builder');
  };

  const handleDelete = async (id: string) => {
    const confirmed = await confirm({
      title: t('delete_webhook') || 'Delete Webhook',
      message: t('webhook_delete_confirm') || 'Are you sure you want to delete this webhook?',
      confirmText: t('delete') || 'Delete',
      variant: 'danger',
    });
    if (!confirmed) return;
    try {
      await deleteMutation.mutateAsync(id);
      toast.success(t('webhook_deleted') || 'Webhook deleted');
    } catch (err: unknown) {
      const message = err instanceof Error ? err.message : String(err);
      toast.error(message || t('delete_failed') || 'Failed to delete');
    }
  };

  const handleTest = async (id: string) => {
    setTesting(id);
    try {
      const result = await testMutation.mutateAsync(id);
      toast.success(`${t('webhook_test_result') || 'Test result'}: ${result.status}`);
    } catch (err: unknown) {
      const message = err instanceof Error ? err.message : String(err);
      toast.error(`${t('webhook_test_failed') || 'Test failed'}: ${message}`);
    } finally {
      setTesting(null);
    }
  };

  const handleSaved = () => {
    setViewMode('list');
    setEditingId(undefined);
    queryClient.invalidateQueries({ queryKey: webhookKeys.all });
  };

  const handleCancel = () => {
    setViewMode('list');
    setEditingId(undefined);
  };

  // ─── Status Icon Helper ────────────────────────────────────────────

  const lastStatusIcon = (status?: string) => {
    if (status === 'success') return <CheckCircle className="w-4 h-4 text-emerald-500" />;
    if (status === 'failed') return <XCircle className="w-4 h-4 text-red-500" />;
    return <Clock className="w-4 h-4 text-slate-400" />;
  };

  // ══════════════════════════════════════════════════════════════════
  // Render: Builder Mode
  // ══════════════════════════════════════════════════════════════════

  if (viewMode === 'builder') {
    return (
      <div className="p-4 md:p-6">
        <WebhookBuilder
          webhookId={editingId}
          onSaved={handleSaved}
          onCancel={handleCancel}
        />
      </div>
    );
  }

  // ══════════════════════════════════════════════════════════════════
  // Render: List Mode
  // ══════════════════════════════════════════════════════════════════

  return (
    <div className="p-4 md:p-6 space-y-6">
      {/* ─── Header ─────────────────────────────────────────────────── */}
      <div className="flex items-center justify-between">
        <div>
          <h1 className="text-2xl font-bold text-slate-900 dark:text-white flex items-center gap-2">
            <Webhook className="w-6 h-6" />
            {t('webhooks') || 'Webhook Endpoints'}
          </h1>
          <p className="text-sm text-slate-500 dark:text-slate-400 mt-1">
            {t('webhooks_desc') || 'Управление исходящими вебхуками для интеграций'}
          </p>
        </div>
        <Button icon={<Plus className="w-4 h-4" />} onClick={openCreate}>
          {t('add_webhook') || 'Добавить'}
        </Button>
      </div>

      {/* ─── Loading ────────────────────────────────────────────────── */}
      {isLoading ? (
        <div className="flex items-center justify-center py-16">
          <RefreshCw className="w-6 h-6 animate-spin text-blue-500" />
        </div>
      ) : error ? (
        <div className="bg-white dark:bg-slate-800 rounded-xl border border-slate-200 dark:border-slate-700 p-12 flex flex-col items-center justify-center text-slate-400">
          <p className="text-sm font-medium text-red-500">
            {t('load_error') || 'Failed to load webhooks'}
          </p>
        </div>
      ) : !webhooks || webhooks.length === 0 ? (
        /* ─── Empty State ──────────────────────────────────────────── */
        <div className="bg-white dark:bg-slate-800 rounded-xl border border-slate-200 dark:border-slate-700">
          <EmptyState
            icon={<Webhook className="w-12 h-12" />}
            title={t('no_webhooks') || 'No webhooks'}
            description={t('webhooks_empty_desc') || 'Configure webhooks to receive real-time events from CCTV Monitor in your external systems'}
            hint={t('webhooks_hint') || 'Supports work order, alarm, device, and SLA events'}
            action={{ label: t('create_webhook') || 'Create Webhook', onClick: openCreate }}
            size="md"
          />
        </div>
      ) : (
        /* ─── Webhook List ──────────────────────────────────────────── */
        <div className="space-y-3">
          {webhooks.map((wh) => (
            <Card key={wh.id}>
              <div className="p-4">
                <div className="flex items-center justify-between mb-3">
                  <div className="flex items-center gap-3">
                    <div
                      className={`p-2 rounded-lg ${
                        wh.active ? 'bg-blue-50 dark:bg-blue-900/30' : 'bg-slate-100 dark:bg-slate-800'
                      }`}
                    >
                      <Webhook
                        className={`w-5 h-5 ${
                          wh.active
                            ? 'text-blue-600 dark:text-blue-400'
                            : 'text-slate-400 dark:text-slate-500'
                        }`}
                      />
                    </div>
                    <div>
                      <div className="flex items-center gap-2">
                        <span className="text-sm font-semibold text-slate-900 dark:text-white">
                          {wh.name}
                        </span>
                        {wh.active ? (
                          <Badge variant="success">Active</Badge>
                        ) : (
                          <Badge variant="info">Inactive</Badge>
                        )}
                      </div>
                      <code className="text-xs font-mono text-slate-500 dark:text-slate-400 break-all">
                        {wh.url}
                      </code>
                    </div>
                  </div>
                  <div className="flex items-center gap-1.5">
                    {lastStatusIcon(wh.last_status)}
                    <Button
                      size="sm"
                      variant="outline"
                      icon={<Play className="w-3 h-3" />}
                      onClick={() => handleTest(wh.id)}
                      loading={testing === wh.id}
                    >
                      {t('test') || 'Test'}
                    </Button>
                    <Button
                      size="sm"
                      variant="outline"
                      icon={<Settings className="w-3 h-3" />}
                      onClick={() => openEdit(wh)}
                    >
                      {t('edit') || 'Edit'}
                    </Button>
                    <Button
                      size="sm"
                      variant="ghost"
                      icon={<Trash2 className="w-3 h-3 text-red-500" />}
                      onClick={() => handleDelete(wh.id)}
                      loading={deleteMutation.isPending && deleteMutation.variables === wh.id}
                    />
                  </div>
                </div>

                {/* Events */}
                <div className="flex flex-wrap gap-1.5">
                  {wh.events?.map((ev) => {
                    const opt = EVENT_OPTIONS.find((o) => o.value === ev);
                    return (
                      <span
                        key={ev}
                        className="px-2 py-0.5 rounded text-[10px] font-medium bg-slate-100 dark:bg-slate-700 text-slate-600 dark:text-slate-400"
                      >
                        {opt?.label || ev}
                      </span>
                    );
                  })}
                </div>

                {/* Meta */}
                <div className="flex items-center gap-4 mt-2 text-[10px] text-slate-400 dark:text-slate-500">
                  <span>
                    {t('retry') || 'Retry'}: {wh.retry_count}x
                  </span>
                  <span>
                    {t('timeout') || 'Timeout'}: {wh.timeout_seconds}s
                  </span>
                  {wh.last_sent_at && (
                    <span>
                      {t('last_sent') || 'Last'}: {new Date(wh.last_sent_at).toLocaleString()}
                    </span>
                  )}
                </div>
              </div>
            </Card>
          ))}
        </div>
      )}

      {ConfirmDialog}
    </div>
  );
}
