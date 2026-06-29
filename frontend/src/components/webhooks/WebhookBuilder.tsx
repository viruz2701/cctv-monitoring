// ═══════════════════════════════════════════════════════════════════════
// WebhookBuilder — визуальный конструктор вебхуков (P2-3.1)
//
// Features:
//   - React Hook Form + Zod валидация
//   - Event type selector (группировка по категориям)
//   - Payload preview (JSON с подсветкой)
//   - Secret token field + reveal toggle + copy
//   - URL validation
//   - Test mode с mock payload
//   - Delivery logs из БД
//
// Compliance:
//   - OWASP ASVS V5 (Input validation — Zod schema + URL validation)
//   - OWASP ASVS V6 (Stored cryptography — secret field masking)
//   - OWASP ASVS V7 (Error handling — no info leakage)
// ═══════════════════════════════════════════════════════════════════════

import React, { useState, useCallback, useMemo } from 'react';
import { useForm } from 'react-hook-form';
import { zodResolver } from '@hookform/resolvers/zod';
import { z } from 'zod';
import { useTranslation } from 'react-i18next';
import {
  Webhook,
  Send,
  Copy,
  Check,
  Eye,
  EyeOff,
  RefreshCw,
  Clock,
  CheckCircle,
  XCircle,
  AlertTriangle,
  Trash2,
  Play,
  BarChart3,
  Shield,
} from '../ui/Icons';
import { Button, Input, Badge, Card, useToast } from '../ui';
import {
  useWebhook,
  useWebhookLogsQuery,
  useCreateWebhook,
  useUpdateWebhook,
  useDeleteWebhook,
  useTestWebhookWithPayload,
  useWebhookStats,
  EVENT_GROUPS,
  EVENT_PAYLOADS,
  type WebhookFormData,
  type TestWebhookResult,
  type WebhookLogEntry,
} from '../../hooks/useWebhooks';
import type { WebhookEndpoint } from '../../services/api';
import { WebhookRetryPolicy } from './WebhookRetryPolicy';
import { WebhookStatsCards } from './WebhookStatsCards';
import { HmacVerificationHelper } from './HmacVerificationHelper';
import { WebhookLogFilter, filterLogs, DEFAULT_LOG_FILTER } from './WebhookLogFilter';
import type { LogFilterState } from './WebhookLogFilter';

// ═══════════════════════════════════════════════════════════════════════
// Zod Schema
// ═══════════════════════════════════════════════════════════════════════

const webhookSchema = z.object({
  name: z
    .string()
    .min(1, 'Name is required')
    .max(100, 'Name must be under 100 characters'),
  url: z
    .string()
    .min(1, 'URL is required')
    .url('Must be a valid URL')
    .refine(
      (val) => val.startsWith('https://') || val.startsWith('http://localhost'),
      'Only HTTPS URLs are allowed (except localhost)'
    ),
  events: z.array(z.string()).min(1, 'At least one event must be selected'),
  secret: z.string().max(256, 'Secret must be under 256 characters').optional().default(''),
  active: z.boolean().default(true),
  retry_count: z.number().int().min(0).max(10).default(3),
  timeout_seconds: z.number().int().min(1).max(120).default(30),
  retry_interval_seconds: z.number().int().min(10).max(3600).default(60),
  retry_backoff: z.boolean().default(true),
  max_retry_duration_seconds: z.number().int().min(60).max(86400).default(3600),
});

type WebhookFormValues = z.infer<typeof webhookSchema>;

// ═══════════════════════════════════════════════════════════════════════
// Props
// ═══════════════════════════════════════════════════════════════════════

interface WebhookBuilderProps {
  webhookId?: string;
  onSaved?: (webhook: WebhookEndpoint) => void;
  onCancel?: () => void;
}

// ═══════════════════════════════════════════════════════════════════════
// Component
// ═══════════════════════════════════════════════════════════════════════

export function WebhookBuilder({ webhookId, onSaved, onCancel }: WebhookBuilderProps) {
  const { t } = useTranslation();
  const toast = useToast();
  const isEditing = !!webhookId;

  // ─── Data ───────────────────────────────────────────────────────────
  const { data: existingWebhook, isLoading: loadingWebhook } = useWebhook(webhookId);
  const { data: deliveryLogs } = useWebhookLogsQuery(webhookId);

  // ─── Mutations ──────────────────────────────────────────────────────
  const createMutation = useCreateWebhook();
  const updateMutation = useUpdateWebhook();
  const deleteMutation = useDeleteWebhook();
  const testMutation = useTestWebhookWithPayload();

  // ─── Local State ────────────────────────────────────────────────────
  const [showSecret, setShowSecret] = useState(false);
  const [copied, setCopied] = useState<'url' | 'secret' | null>(null);
  const [selectedPreviewEvent, setSelectedPreviewEvent] = useState<string | null>(null);
  const [testResult, setTestResult] = useState<TestWebhookResult | null>(null);
  const [showDeleteConfirm, setShowDeleteConfirm] = useState(false);
  const [activeTab, setActiveTab] = useState<'config' | 'logs'>('config');
  const [logFilter, setLogFilter] = useState<LogFilterState>(DEFAULT_LOG_FILTER);

  // ─── Form ───────────────────────────────────────────────────────────
  const {
    register,
    handleSubmit,
    watch,
    setValue,
    formState: { errors, isSubmitting },
  } = useForm<WebhookFormValues>({
    resolver: zodResolver(webhookSchema),
    defaultValues: {
      name: '',
      url: '',
      events: [],
      secret: '',
      active: true,
      retry_count: 3,
      timeout_seconds: 30,
      retry_interval_seconds: 60,
      retry_backoff: true,
      max_retry_duration_seconds: 3600,
    },
    values: existingWebhook
      ? {
          name: existingWebhook.name,
          url: existingWebhook.url,
          events: existingWebhook.events || [],
          secret: '',
          active: existingWebhook.active,
          retry_count: existingWebhook.retry_count,
          timeout_seconds: existingWebhook.timeout_seconds,
          retry_interval_seconds: existingWebhook.retry_interval_seconds ?? 60,
          retry_backoff: existingWebhook.retry_backoff ?? true,
          max_retry_duration_seconds: existingWebhook.max_retry_duration_seconds ?? 3600,
        }
      : undefined,
  });

  const selectedEvents = watch('events');
  const watchRetryBackoff = watch('retry_backoff');
  const webhookUrl = existingWebhook
    ? `${window.location.origin}/api/v1/integrations/extended/webhooks/${existingWebhook.id}/trigger`
    : null;

  // ─── Event Toggle ───────────────────────────────────────────────────
  const toggleEvent = useCallback(
    (event: string) => {
      const current = selectedEvents || [];
      const updated = current.includes(event)
        ? current.filter((e) => e !== event)
        : [...current, event];
      setValue('events', updated, { shouldValidate: true });
    },
    [selectedEvents, setValue]
  );

  const selectAllEvents = useCallback(() => {
    const all = EVENT_GROUPS.flatMap((g) => g.events.map((e) => e.value));
    setValue('events', all, { shouldValidate: true });
  }, [setValue]);

  const clearAllEvents = useCallback(() => {
    setValue('events', [], { shouldValidate: true });
  }, [setValue]);

  // ─── Copy Handler ──────────────────────────────────────────────────
  const copyToClipboard = useCallback(
    async (text: string, type: 'url' | 'secret') => {
      try {
        await navigator.clipboard.writeText(text);
        setCopied(type);
        setTimeout(() => setCopied(null), 2000);
        toast.success(t('copied') || 'Copied!');
      } catch {
        toast.error(t('copy_failed') || 'Failed to copy');
      }
    },
    [toast, t]
  );

  // ─── Save Handler ──────────────────────────────────────────────────
  const onSubmit = useCallback(
    async (data: WebhookFormValues) => {
      try {
        if (isEditing && webhookId) {
          const result = await updateMutation.mutateAsync({
            id: webhookId,
            data,
          });
          toast.success(t('webhook_updated') || 'Webhook updated');
          onSaved?.(result);
        } else {
          const result = await createMutation.mutateAsync(data);
          toast.success(t('webhook_created') || 'Webhook created');
          onSaved?.(result);
        }
      } catch (err: any) {
        toast.error(err.message || t('save_failed') || 'Failed to save webhook');
      }
    },
    [isEditing, webhookId, updateMutation, createMutation, toast, t, onSaved]
  );

  // ─── Delete Handler ────────────────────────────────────────────────
  const handleDelete = useCallback(async () => {
    if (!webhookId) return;
    try {
      await deleteMutation.mutateAsync(webhookId);
      toast.success(t('webhook_deleted') || 'Webhook deleted');
      onCancel?.();
    } catch (err: any) {
      toast.error(err.message || t('delete_failed') || 'Failed to delete');
    }
  }, [webhookId, deleteMutation, toast, t, onCancel]);

  // ─── Test Handler ──────────────────────────────────────────────────
  const handleTest = useCallback(async () => {
    if (!webhookId) return;
    const previewEvent = selectedPreviewEvent || selectedEvents?.[0];
    if (!previewEvent) {
      toast.warning(t('select_event_to_test') || 'Select an event type to test');
      return;
    }
    const payload = EVENT_PAYLOADS[previewEvent] || {
      event: previewEvent,
      timestamp: new Date().toISOString(),
      data: { message: 'Test payload' },
    };

    try {
      const result = await testMutation.mutateAsync({
        id: webhookId,
        payload,
      });
      setTestResult(result);
      toast.success(
        `${t('webhook_test_result') || 'Test result'}: ${result.status} (${result.status_code})`
      );
    } catch (err: any) {
      setTestResult({
        status: 'error',
        status_code: 0,
        duration_ms: 0,
        response_body: '',
        error: err.message,
      });
      toast.error(`${t('webhook_test_failed') || 'Test failed'}: ${err.message}`);
    }
  }, [webhookId, selectedPreviewEvent, selectedEvents, testMutation, toast, t]);

  // ─── Loading ────────────────────────────────────────────────────────
  if (isEditing && loadingWebhook) {
    return (
      <div className="flex items-center justify-center py-16">
        <RefreshCw className="w-6 h-6 animate-spin text-blue-500" />
      </div>
    );
  }

  // ═════════════════════════════════════════════════════════════════════
  // Render
  // ═════════════════════════════════════════════════════════════════════

  return (
    <div className="space-y-6">
      {/* ─── Header ─────────────────────────────────────────────────── */}
      <div className="flex items-center justify-between">
        <div className="flex items-center gap-3">
          <div className="p-2 rounded-lg bg-blue-50 dark:bg-blue-900/30">
            <Webhook className="w-5 h-5 text-blue-600 dark:text-blue-400" />
          </div>
          <div>
            <h2 className="text-lg font-bold text-slate-900 dark:text-white">
              {isEditing
                ? (t('edit_webhook') || 'Edit Webhook')
                : (t('create_webhook') || 'Create Webhook')}
            </h2>
            {existingWebhook && (
              <p className="text-xs text-slate-500 dark:text-slate-400">
                {t('created') || 'Created'}: {new Date(existingWebhook.created_at).toLocaleString()}
              </p>
            )}
          </div>
        </div>
        <div className="flex items-center gap-2">
          {isEditing && (
            <Button
              variant="danger"
              size="sm"
              icon={<Trash2 className="w-3 h-3" />}
              onClick={() => setShowDeleteConfirm(true)}
              loading={deleteMutation.isPending}
            >
              {t('delete') || 'Delete'}
            </Button>
          )}
          <Button variant="ghost" size="sm" onClick={onCancel}>
            {t('cancel') || 'Cancel'}
          </Button>
          <Button
            size="sm"
            icon={<Send className="w-3 h-3" />}
            onClick={handleSubmit(onSubmit)}
            loading={isSubmitting || createMutation.isPending || updateMutation.isPending}
          >
            {isEditing ? (t('save') || 'Save') : (t('create') || 'Create')}
          </Button>
        </div>
      </div>

      {isEditing && webhookUrl && (
        <div className="flex items-center gap-2 px-4 py-2 bg-slate-50 dark:bg-slate-800/50 rounded-lg border border-slate-200 dark:border-slate-700">
          <code className="flex-1 text-xs font-mono text-slate-600 dark:text-slate-400 truncate">
            {webhookUrl}
          </code>
          <Button
            size="sm"
            variant="outline"
            icon={copied === 'url' ? <Check className="w-3 h-3 text-emerald-500" /> : <Copy className="w-3 h-3" />}
            onClick={() => copyToClipboard(webhookUrl, 'url')}
          >
            {copied === 'url' ? (t('copied') || 'Copied') : (t('copy_url') || 'Copy URL')}
          </Button>
        </div>
      )}

      {/* ─── Tabs ───────────────────────────────────────────────────── */}
      {isEditing && (
        <div className="flex gap-1 border-b border-slate-200 dark:border-slate-700">
          <button
            type="button"
            onClick={() => setActiveTab('config')}
            className={`px-4 py-2 text-sm font-medium rounded-t-lg transition-colors ${
              activeTab === 'config'
                ? 'text-blue-600 border-b-2 border-blue-600 bg-blue-50/50 dark:bg-blue-900/20 dark:text-blue-400 dark:border-blue-400'
                : 'text-slate-500 hover:text-slate-700 dark:text-slate-400 dark:hover:text-slate-200'
            }`}
          >
            {t('configuration') || 'Configuration'}
          </button>
          <button
            type="button"
            onClick={() => setActiveTab('logs')}
            className={`px-4 py-2 text-sm font-medium rounded-t-lg transition-colors flex items-center gap-1.5 ${
              activeTab === 'logs'
                ? 'text-blue-600 border-b-2 border-blue-600 bg-blue-50/50 dark:bg-blue-900/20 dark:text-blue-400 dark:border-blue-400'
                : 'text-slate-500 hover:text-slate-700 dark:text-slate-400 dark:hover:text-slate-200'
            }`}
          >
            <Clock className="w-3.5 h-3.5" />
            {t('delivery_logs') || 'Delivery Logs'}
            {deliveryLogs && deliveryLogs.length > 0 && (
              <Badge variant="info">{deliveryLogs.length}</Badge>
            )}
          </button>
        </div>
      )}

      {/* ══════════════════════════════════════════════════════════════ */}
      {/* Stats Dashboard */}
      {/* ══════════════════════════════════════════════════════════════ */}
      {isEditing && (
        <div className="space-y-2">
          <div className="flex items-center gap-2">
            <BarChart3 className="w-4 h-4 text-slate-500" />
            <h3 className="text-xs font-semibold text-slate-500 dark:text-slate-400 uppercase tracking-wider">
              {t('webhook_stats') || 'Webhook Statistics'}
            </h3>
          </div>
          <WebhookStatsCards webhookId={webhookId} />
        </div>
      )}

      {/* ══════════════════════════════════════════════════════════════ */}
      {/* Tab: Configuration */}
      {/* ══════════════════════════════════════════════════════════════ */}
      {activeTab === 'config' && (
        <form onSubmit={handleSubmit(onSubmit)} className="space-y-6">
          <div className="grid grid-cols-1 lg:grid-cols-3 gap-6">
            {/* ─── Left Column: Basic Info ─────────────────────────── */}
            <div className="lg:col-span-2 space-y-4">
              <Card>
                <div className="p-4 space-y-4">
                  <h3 className="text-sm font-bold text-slate-800 dark:text-slate-200">
                    {t('basic_info') || 'Basic Information'}
                  </h3>

                  {/* Name */}
                  <Input
                    label={t('name') || 'Name'}
                    placeholder="My Integration"
                    error={errors.name?.message}
                    {...register('name')}
                  />

                  {/* URL */}
                  <div>
                    <Input
                      label="URL"
                      placeholder="https://example.com/webhook"
                      error={errors.url?.message}
                      {...register('url')}
                    />
                    <p className="mt-1 text-[10px] text-slate-400">
                      {t('url_hint') || 'Only HTTPS URLs are accepted (except localhost for development)'}
                    </p>
                  </div>

                  {/* Retry Policy (WebhookRetryPolicy) */}
                  <WebhookRetryPolicy
                    register={register}
                    errors={errors}
                    watchRetryBackoff={watchRetryBackoff}
                  />

                  {/* Active Toggle */}
                  <label className="flex items-center gap-3 p-3 rounded-lg border border-slate-200 dark:border-slate-700 cursor-pointer hover:bg-slate-50 dark:hover:bg-slate-800/50">
                    <input
                      type="checkbox"
                      className="w-4 h-4 rounded border-slate-300 text-blue-600 focus:ring-blue-500"
                      {...register('active')}
                    />
                    <div>
                      <span className="text-sm font-medium text-slate-700 dark:text-slate-200">
                        {t('active') || 'Active'}
                      </span>
                      <p className="text-[10px] text-slate-400">
                        {t('active_hint') || 'When inactive, no events will be sent to this URL'}
                      </p>
                    </div>
                  </label>
                </div>
              </Card>

              {/* ─── Secret Token ──────────────────────────────────── */}
              <Card>
                <div className="p-4 space-y-3">
                  <h3 className="text-sm font-bold text-slate-800 dark:text-slate-200">
                    {t('secret_token') || 'Secret Token'}
                  </h3>
                  <div className="relative">
                    <input
                      type={showSecret ? 'text' : 'password'}
                      placeholder={t('webhook_secret_hint') || 'HMAC secret for payload verification'}
                      className={`w-full px-3.5 py-2.5 text-sm font-mono border rounded-lg bg-white dark:bg-slate-900 pr-20 ${
                        errors.secret
                          ? 'border-red-300 focus:ring-red-500'
                          : 'border-slate-300 dark:border-slate-700'
                      } text-slate-900 dark:text-white focus:outline-none focus:ring-2 focus:ring-blue-500`}
                      {...register('secret')}
                    />
                    <div className="absolute right-1.5 top-1/2 -translate-y-1/2 flex items-center gap-1">
                      <button
                        type="button"
                        onClick={() => setShowSecret(!showSecret)}
                        className="p-1.5 rounded hover:bg-slate-100 dark:hover:bg-slate-700 text-slate-400"
                        tabIndex={-1}
                      >
                        {showSecret ? <EyeOff className="w-4 h-4" /> : <Eye className="w-4 h-4" />}
                      </button>
                      {existingWebhook?.secret && (
                        <button
                          type="button"
                          onClick={() => copyToClipboard(existingWebhook.secret!, 'secret')}
                          className="p-1.5 rounded hover:bg-slate-100 dark:hover:bg-slate-700 text-slate-400"
                          tabIndex={-1}
                        >
                          {copied === 'secret' ? (
                            <Check className="w-4 h-4 text-emerald-500" />
                          ) : (
                            <Copy className="w-4 h-4" />
                          )}
                        </button>
                      )}
                    </div>
                  </div>
                  {errors.secret && (
                    <p className="text-sm text-red-600">{errors.secret.message}</p>
                  )}
                  <p className="text-[10px] text-slate-400">
                    {t('secret_hint') ||
                      'This secret is used to sign payloads sent to your webhook URL. Verify using HMAC-SHA256 on your end.'}
                  </p>
                </div>
              </Card>

              {/* ─── HMAC Verification Helper ──────────────────────── */}
              <HmacVerificationHelper
                secret={watch('secret') || existingWebhook?.secret || ''}
              />
            </div>

            {/* ─── Right Column: Event Selector + Preview ──────────── */}
            <div className="space-y-4">
              {/* Event Type Selector */}
              <Card>
                <div className="p-4 space-y-3">
                  <div className="flex items-center justify-between">
                    <h3 className="text-sm font-bold text-slate-800 dark:text-slate-200">
                      {t('events') || 'Events'}
                    </h3>
                    <div className="flex items-center gap-1">
                      <button
                        type="button"
                        onClick={selectAllEvents}
                        className="text-[10px] px-2 py-0.5 rounded text-blue-600 hover:bg-blue-50 dark:hover:bg-blue-900/30 font-medium"
                      >
                        {t('select_all') || 'All'}
                      </button>
                      <button
                        type="button"
                        onClick={clearAllEvents}
                        className="text-[10px] px-2 py-0.5 rounded text-slate-500 hover:bg-slate-50 dark:hover:bg-slate-800 font-medium"
                      >
                        {t('clear') || 'Clear'}
                      </button>
                    </div>
                  </div>

                  {errors.events && (
                    <p className="text-xs text-red-600">{errors.events.message}</p>
                  )}

                  <div className="space-y-2 max-h-80 overflow-y-auto">
                    {EVENT_GROUPS.map((group) => (
                      <div key={group.label}>
                        <p className="text-[10px] font-semibold text-slate-400 uppercase tracking-wider mb-1">
                          {group.label}
                        </p>
                        <div className="space-y-0.5">
                          {group.events.map((ev) => {
                            const isSelected = (selectedEvents || []).includes(ev.value);
                            return (
                              <label
                                key={ev.value}
                                className={`flex items-center gap-2 px-2 py-1.5 rounded cursor-pointer transition-colors ${
                                  isSelected
                                    ? 'bg-blue-50 dark:bg-blue-900/20'
                                    : 'hover:bg-slate-50 dark:hover:bg-slate-800'
                                }`}
                              >
                                <input
                                  type="checkbox"
                                  checked={isSelected}
                                  onChange={() => toggleEvent(ev.value)}
                                  className="rounded border-slate-300 text-blue-600 focus:ring-blue-500 w-3.5 h-3.5"
                                />
                                <span className="text-xs text-slate-700 dark:text-slate-300 flex-1">
                                  {ev.label}
                                </span>
                                <button
                                  type="button"
                                  onClick={(e) => {
                                    e.preventDefault();
                                    setSelectedPreviewEvent(
                                      selectedPreviewEvent === ev.value ? null : ev.value
                                    );
                                  }}
                                  className={`text-[10px] px-1.5 py-0.5 rounded font-mono transition-colors ${
                                    selectedPreviewEvent === ev.value
                                      ? 'bg-blue-100 text-blue-700 dark:bg-blue-900/40 dark:text-blue-300'
                                      : 'text-slate-400 hover:text-slate-600 dark:hover:text-slate-300'
                                  }`}
                                  title={t('preview_payload') || 'Preview payload'}
                                >
                                  {'{ }'}
                                </button>
                              </label>
                            );
                          })}
                        </div>
                      </div>
                    ))}
                  </div>

                  {/* Selected Event Chips */}
                  {selectedEvents && selectedEvents.length > 0 && (
                    <div className="flex flex-wrap gap-1 pt-2 border-t border-slate-100 dark:border-slate-700">
                      {selectedEvents.map((ev) => (
                        <span
                          key={ev}
                          className="inline-flex items-center gap-1 px-2 py-0.5 rounded-full text-[10px] font-medium bg-blue-50 text-blue-700 dark:bg-blue-900/30 dark:text-blue-300"
                        >
                          {ev}
                          <button
                            type="button"
                            onClick={() => toggleEvent(ev)}
                            className="hover:text-blue-900 dark:hover:text-blue-100"
                          >
                            ×
                          </button>
                        </span>
                      ))}
                    </div>
                  )}
                </div>
              </Card>

              {/* Payload Preview */}
              <Card>
                <div className="p-4 space-y-2">
                  <div className="flex items-center justify-between">
                    <h3 className="text-sm font-bold text-slate-800 dark:text-slate-200">
                      {t('payload_preview') || 'Payload Preview'}
                    </h3>
                    {selectedPreviewEvent && (
                      <Badge variant="info">{selectedPreviewEvent}</Badge>
                    )}
                  </div>
                  <pre className="text-[11px] font-mono leading-relaxed bg-slate-50 dark:bg-slate-900 border border-slate-200 dark:border-slate-700 rounded-lg p-3 overflow-auto max-h-48 text-slate-800 dark:text-slate-200">
                    {selectedPreviewEvent && EVENT_PAYLOADS[selectedPreviewEvent]
                      ? syntaxHighlight(JSON.stringify(EVENT_PAYLOADS[selectedPreviewEvent], null, 2))
                      : JSON.stringify(
                          {
                            event: 'select_an_event',
                            timestamp: new Date().toISOString(),
                            data: { message: 'Select an event type above to see the payload preview' },
                          },
                          null,
                          2
                        )}
                  </pre>
                </div>
              </Card>
            </div>
          </div>

          {/* ─── Test Mode ─────────────────────────────────────────── */}
          {isEditing && (
            <Card>
              <div className="p-4 space-y-4">
                <div className="flex items-center justify-between">
                  <div className="flex items-center gap-2">
                    <Play className="w-4 h-4 text-amber-500" />
                    <h3 className="text-sm font-bold text-slate-800 dark:text-slate-200">
                      {t('test_mode') || 'Test Mode'}
                    </h3>
                  </div>
                  <div className="flex items-center gap-2">
                    {selectedEvents && selectedEvents.length > 0 && (
                      <select
                        value={selectedPreviewEvent || selectedEvents[0]}
                        onChange={(e) => setSelectedPreviewEvent(e.target.value)}
                        className="text-xs px-2 py-1 border border-slate-300 dark:border-slate-600 rounded bg-white dark:bg-slate-900 text-slate-700 dark:text-slate-300"
                      >
                        {(selectedEvents || []).map((ev) => (
                          <option key={ev} value={ev}>
                            {ev}
                          </option>
                        ))}
                      </select>
                    )}
                    <Button
                      size="sm"
                      variant="outline"
                      icon={<Send className="w-3 h-3" />}
                      onClick={handleTest}
                      loading={testMutation.isPending}
                      disabled={!selectedEvents?.length}
                    >
                      {t('send_test') || 'Send Test'}
                    </Button>
                  </div>
                </div>

                {testResult && (
                  <div className="space-y-2">
                    {/* Status Badge */}
                    <div className="flex items-center gap-3">
                      <span
                        className={`inline-flex items-center gap-1 px-2 py-0.5 rounded-full text-xs font-medium ${
                          testResult.status === 'success'
                            ? 'bg-emerald-50 text-emerald-700 dark:bg-emerald-900/30 dark:text-emerald-300'
                            : 'bg-red-50 text-red-700 dark:bg-red-900/30 dark:text-red-300'
                        }`}
                      >
                        {testResult.status === 'success' ? (
                          <CheckCircle className="w-3 h-3" />
                        ) : (
                          <XCircle className="w-3 h-3" />
                        )}
                        {testResult.status} — {testResult.status_code}
                      </span>
                      <span className="text-[10px] text-slate-400">
                        {testResult.duration_ms}ms
                      </span>
                    </div>

                    {/* Response */}
                    <pre className="text-[11px] font-mono bg-slate-50 dark:bg-slate-900 border border-slate-200 dark:border-slate-700 rounded-lg p-3 overflow-auto max-h-32 text-slate-800 dark:text-slate-200">
                      {testResult.error
                        ? `Error: ${testResult.error}`
                        : syntaxHighlight(
                            JSON.stringify(
                              { response_body: testResult.response_body },
                              null,
                              2
                            )
                          )}
                    </pre>
                  </div>
                )}

                {!testResult && !testMutation.isPending && (
                  <p className="text-xs text-slate-400">
                    {t('test_hint') ||
                      'Select an event type above and click "Send Test" to verify your webhook endpoint'}
                  </p>
                )}
              </div>
            </Card>
          )}
        </form>
      )}

      {/* ══════════════════════════════════════════════════════════════ */}
      {/* Tab: Delivery Logs */}
      {/* ══════════════════════════════════════════════════════════════ */}
      {activeTab === 'logs' && (
        <div className="space-y-4">
          {/* Filter Bar */}
          <WebhookLogFilter
            logs={deliveryLogs}
            filter={logFilter}
            onFilterChange={setLogFilter}
          />

          {/* Filtered Logs Table */}
          <DeliveryLogsTable
            logs={deliveryLogs ? filterLogs(deliveryLogs, logFilter) : undefined}
            isLoading={false}
          />
        </div>
      )}

      {/* ─── Delete Confirmation ────────────────────────────────────── */}
      {showDeleteConfirm && (
        <div className="fixed inset-0 z-50 flex items-center justify-center bg-black/50">
          <div className="bg-white dark:bg-slate-800 rounded-xl shadow-xl border border-slate-200 dark:border-slate-700 p-6 max-w-md w-full mx-4 space-y-4">
            <div className="flex items-center gap-3">
              <div className="p-2 rounded-full bg-red-50 dark:bg-red-900/30">
                <AlertTriangle className="w-5 h-5 text-red-500" />
              </div>
              <div>
                <h3 className="text-sm font-bold text-slate-900 dark:text-white">
                  {t('delete_webhook') || 'Delete Webhook'}
                </h3>
                <p className="text-xs text-slate-500">
                  {t('webhook_delete_confirm') ||
                    'Are you sure you want to delete this webhook? This action cannot be undone.'}
                </p>
              </div>
            </div>
            <div className="flex justify-end gap-2">
              <Button variant="ghost" size="sm" onClick={() => setShowDeleteConfirm(false)}>
                {t('cancel') || 'Cancel'}
              </Button>
              <Button variant="danger" size="sm" onClick={handleDelete} loading={deleteMutation.isPending}>
                {t('delete') || 'Delete'}
              </Button>
            </div>
          </div>
        </div>
      )}
    </div>
  );
}

// ═══════════════════════════════════════════════════════════════════════
// Delivery Logs Table
// ═══════════════════════════════════════════════════════════════════════

interface DeliveryLogsTableProps {
  logs?: Array<{
    id: string;
    created_at: string;
    status: 'success' | 'failed';
    request_url?: string;
    event_type?: string;
    response_status: number;
    response_body?: string;
    duration_ms: number;
    retry_attempt: number;
    error_message?: string;
  }>;
  isLoading: boolean;
}

function DeliveryLogsTable({ logs, isLoading }: DeliveryLogsTableProps) {
  const { t } = useTranslation();
  const [expandedLog, setExpandedLog] = useState<string | null>(null);

  if (isLoading) {
    return (
      <div className="flex items-center justify-center py-16">
        <RefreshCw className="w-6 h-6 animate-spin text-blue-500" />
      </div>
    );
  }

  if (!logs || logs.length === 0) {
    return (
      <div className="bg-white dark:bg-slate-800 rounded-xl border border-slate-200 dark:border-slate-700 p-12 flex flex-col items-center justify-center text-slate-400">
        <Clock className="w-12 h-12 mb-3 opacity-50" />
        <p className="text-sm font-medium text-slate-500 dark:text-slate-300 mb-1">
          {t('no_delivery_logs') || 'No delivery logs yet'}
        </p>
        <p className="text-xs text-center max-w-sm">
          {t('delivery_logs_hint') ||
            'Delivery logs will appear here once the webhook starts receiving events. Use the Test button to send a mock event.'}
        </p>
      </div>
    );
  }

  return (
    <div className="space-y-2">
      {logs.map((log) => (
        <Card key={log.id}>
          <div className="p-3">
            <div className="flex items-center justify-between">
              <div className="flex items-center gap-2">
                {log.status === 'success' ? (
                  <CheckCircle className="w-4 h-4 text-emerald-500 shrink-0" />
                ) : (
                  <XCircle className="w-4 h-4 text-red-500 shrink-0" />
                )}
                <div>
                  <div className="flex items-center gap-2">
                    <span className="text-xs font-medium text-slate-900 dark:text-white">
                      {log.event_type || 'webhook.test'}
                    </span>
                    <span
                      className={`text-[10px] px-1.5 py-0.5 rounded font-medium ${
                        log.status === 'success'
                          ? 'bg-emerald-50 text-emerald-700 dark:bg-emerald-900/30 dark:text-emerald-300'
                          : 'bg-red-50 text-red-700 dark:bg-red-900/30 dark:text-red-300'
                      }`}
                    >
                      {log.response_status}
                    </span>
                  </div>
                  <div className="flex items-center gap-3 text-[10px] text-slate-400">
                    <span>{new Date(log.created_at).toLocaleString()}</span>
                    <span>{log.duration_ms}ms</span>
                    {log.retry_attempt > 0 && (
                      <span>
                        {t('retry') || 'Retry'} #{log.retry_attempt}
                      </span>
                    )}
                  </div>
                </div>
              </div>
              <button
                type="button"
                onClick={() => setExpandedLog(expandedLog === log.id ? null : log.id)}
                className="text-[10px] text-blue-600 hover:text-blue-800 dark:text-blue-400 font-medium"
              >
                {expandedLog === log.id ? (t('hide') || 'Hide') : (t('details') || 'Details')}
              </button>
            </div>

            {expandedLog === log.id && (
              <div className="mt-3 pt-3 border-t border-slate-100 dark:border-slate-700 space-y-2">
                <div className="grid grid-cols-2 gap-2 text-[10px]">
                  <div>
                    <span className="text-slate-400">{t('url') || 'URL'}:</span>{' '}
                    <code className="text-slate-600 dark:text-slate-300 font-mono break-all">
                      {log.request_url || '-'}
                    </code>
                  </div>
                  <div>
                    <span className="text-slate-400">{t('duration') || 'Duration'}:</span>{' '}
                    <span className="text-slate-600 dark:text-slate-300">{log.duration_ms}ms</span>
                  </div>
                  <div>
                    <span className="text-slate-400">{t('retry_attempt') || 'Retry'}:</span>{' '}
                    <span className="text-slate-600 dark:text-slate-300">#{log.retry_attempt}</span>
                  </div>
                  <div>
                    <span className="text-slate-400">{t('status_code') || 'Status'}:</span>{' '}
                    <span className="text-slate-600 dark:text-slate-300">{log.response_status}</span>
                  </div>
                </div>

                {log.error_message && (
                  <div className="text-[10px] text-red-600 bg-red-50 dark:bg-red-900/20 rounded p-2">
                    {log.error_message}
                  </div>
                )}

                {log.response_body && (
                  <div>
                    <p className="text-[10px] font-medium text-slate-400 mb-1">
                      {t('response') || 'Response'}:
                    </p>
                    <pre className="text-[10px] font-mono bg-slate-50 dark:bg-slate-900 border border-slate-200 dark:border-slate-700 rounded p-2 overflow-auto max-h-24 text-slate-600 dark:text-slate-300">
                      {syntaxHighlight(
                        (() => {
                          try {
                            return JSON.stringify(JSON.parse(log.response_body), null, 2);
                          } catch {
                            return log.response_body;
                          }
                        })()
                      )}
                    </pre>
                  </div>
                )}
              </div>
            )}
          </div>
        </Card>
      ))}
    </div>
  );
}

// ═══════════════════════════════════════════════════════════════════════
// JSON Syntax Highlighting
// ═══════════════════════════════════════════════════════════════════════
// Простая подсветка JSON через RegExp — без внешних зависимостей
// ═══════════════════════════════════════════════════════════════════════

function syntaxHighlight(json: string): React.ReactNode {
  return json.replace(
    /("(?:[^"\\]|\\.)*")(?=\s*[:,\]\}])|("(?:[^"\\]|\\.)*")|(true|false|null)|(-?\d+(?:\.\d+)?(?:[eE][+-]?\d+)?)/g,
    (match, key, str, bool, num) => {
      if (key) {
        return `<span class="text-blue-600 dark:text-blue-400">${key}</span>`;
      }
      if (str) {
        return `<span class="text-emerald-600 dark:text-emerald-400">${str}</span>`;
      }
      if (bool) {
        return `<span class="text-purple-600 dark:text-purple-400">${bool}</span>`;
      }
      if (num) {
        return `<span class="text-amber-600 dark:text-amber-400">${num}</span>`;
      }
      return match;
    }
  );
}
