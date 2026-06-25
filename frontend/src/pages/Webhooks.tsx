import React, { useState, useEffect } from 'react';
import { useTranslation } from 'react-i18next';
import { api, WebhookEndpoint } from '../services/api';
import { Card, Button, Badge, Input, Modal, useToast, EmptyState } from '../components/ui';
import { useConfirmAction } from '../hooks/useConfirmAction';
import {
  Webhook, Plus, Trash2, Play, RefreshCw,
  CheckCircle, XCircle, Clock, AlertTriangle,
} from 'lucide-react';

const EVENT_OPTIONS = [
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

export function Webhooks() {
  const { t } = useTranslation();
  const toast = useToast();
  const { confirm, ConfirmDialog } = useConfirmAction();
  const [webhooks, setWebhooks] = useState<WebhookEndpoint[]>([]);
  const [loading, setLoading] = useState(true);
  const [testing, setTesting] = useState<string | null>(null);
  const [showModal, setShowModal] = useState(false);
  const [editing, setEditing] = useState<WebhookEndpoint | null>(null);
  const [form, setForm] = useState({ name: '', url: '', events: [] as string[], secret: '', active: true });

  useEffect(() => { load(); }, []);

  const load = async () => {
    setLoading(true);
    try {
      const data = await api.getWebhooks();
      setWebhooks(data || []);
    } catch { setWebhooks([]); }
    finally { setLoading(false); }
  };

  const openCreate = () => {
    setEditing(null);
    setForm({ name: '', url: '', events: [], secret: '', active: true });
    setShowModal(true);
  };

  const openEdit = (wh: WebhookEndpoint) => {
    setEditing(wh);
    setForm({ name: wh.name, url: wh.url, events: wh.events || [], secret: '', active: wh.active });
    setShowModal(true);
  };

  const handleSave = async () => {
    try {
      if (editing) {
        const updates: any = { name: form.name, url: form.url, events: form.events, active: form.active };
        if (form.secret) updates.secret = form.secret;
        await api.updateWebhook(editing.id, updates);
        toast.success(t('webhook_updated'));
      } else {
        await api.createWebhook(form);
        toast.success(t('webhook_created'));
      }
      setShowModal(false);
      load();
    } catch (err: any) {
      toast.error(err.message);
    }
  };

  const handleDelete = async (id: string) => {
    const confirmed = await confirm({
      title: t('delete_webhook') || 'Delete Webhook',
      message: t('webhook_delete_confirm'),
      confirmText: t('delete') || 'Delete',
      variant: 'danger',
    });
    if (!confirmed) return;
    try {
      await api.deleteWebhook(id);
      toast.success(t('webhook_deleted'));
      load();
    } catch (err: any) {
      toast.error(err.message);
    }
  };

  const handleTest = async (id: string) => {
    setTesting(id);
    try {
      const result = await api.testWebhook(id);
      toast.success(`${t('webhook_test_result')}: ${result.status}`);
    } catch (err: any) {
      toast.error(`${t('webhook_test_failed')}: ${err.message}`);
    } finally { setTesting(null); }
  };

  const toggleEvent = (ev: string) => {
    setForm(prev => ({
      ...prev,
      events: prev.events.includes(ev)
        ? prev.events.filter(e => e !== ev)
        : [...prev.events, ev],
    }));
  };

  const lastStatusIcon = (status?: string) => {
    if (status === 'success') return <CheckCircle className="w-4 h-4 text-emerald-500" />;
    if (status === 'failed') return <XCircle className="w-4 h-4 text-red-500" />;
    return <Clock className="w-4 h-4 text-slate-400" />;
  };

  return (
    <div className="space-y-6">
      <div className="flex items-center justify-between">
        <div>
          <h1 className="text-2xl font-bold text-slate-900 flex items-center gap-2">
            <Webhook className="w-6 h-6" />
            {t('webhooks') || 'Webhook Endpoints'}
          </h1>
          <p className="text-sm text-slate-500 mt-1">
            {t('webhooks_desc') || 'Управление исходящими вебхуками для интеграций'}
          </p>
        </div>
        <Button icon={<Plus className="w-4 h-4" />} onClick={openCreate}>
          {t('add_webhook') || 'Добавить'}
        </Button>
      </div>

      {loading ? (
        <div className="flex items-center justify-center py-16">
          <RefreshCw className="w-6 h-6 animate-spin text-blue-500" />
        </div>
      ) : webhooks.length === 0 ? (
        <div className="bg-white dark:bg-slate-900 rounded-xl border border-slate-200 dark:border-slate-800">
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
        <div className="space-y-3">
          {webhooks.map(wh => (
            <Card key={wh.id}>
              <div className="p-4">
                <div className="flex items-center justify-between mb-3">
                  <div className="flex items-center gap-3">
                    <div className={`p-2 rounded-lg ${wh.active ? 'bg-blue-50' : 'bg-slate-100'}`}>
                      <Webhook className={`w-5 h-5 ${wh.active ? 'text-blue-600' : 'text-slate-400'}`} />
                    </div>
                    <div>
                      <div className="flex items-center gap-2">
                        <span className="text-sm font-semibold text-slate-900">{wh.name}</span>
                        {wh.active ? <Badge variant="success">Active</Badge> : <Badge variant="info">Inactive</Badge>}
                      </div>
                      <code className="text-xs font-mono text-slate-500">{wh.url}</code>
                    </div>
                  </div>
                  <div className="flex items-center gap-2">
                    {lastStatusIcon(wh.last_status)}
                    <Button size="sm" variant="outline" icon={<Play className="w-3 h-3" />}
                      onClick={() => handleTest(wh.id)} loading={testing === wh.id}>
                      {t('test') || 'Test'}
                    </Button>
                    <Button size="sm" variant="outline" icon={<RefreshCw className="w-3 h-3" />}
                      onClick={() => openEdit(wh)}>
                      {t('edit') || 'Edit'}
                    </Button>
                    <Button size="sm" variant="ghost" icon={<Trash2 className="w-3 h-3 text-red-500" />}
                      onClick={() => handleDelete(wh.id)} />
                  </div>
                </div>

                <div className="flex flex-wrap gap-1.5">
                  {wh.events?.map(ev => (
                    <span key={ev} className="px-2 py-0.5 rounded text-[10px] font-medium bg-slate-100 text-slate-600">
                      {ev}
                    </span>
                  ))}
                </div>

                <div className="flex items-center gap-4 mt-2 text-[10px] text-slate-400">
                  <span>Retry: {wh.retry_count}x</span>
                  <span>Timeout: {wh.timeout_seconds}s</span>
                  {wh.last_sent_at && <span>Last: {new Date(wh.last_sent_at).toLocaleString()}</span>}
                </div>
              </div>
            </Card>
          ))}
        </div>
      )}

      {/* Create/Edit Modal */}
      <Modal isOpen={showModal} onClose={() => setShowModal(false)}
        title={editing ? (t('edit_webhook') || 'Edit Webhook') : (t('create_webhook') || 'Create Webhook')} size="lg">
        <div className="space-y-4">
          <Input label={t('name') || 'Name'} value={form.name}
            onChange={e => setForm({ ...form, name: e.target.value })} placeholder="My Integration" />
          <Input label="URL" value={form.url}
            onChange={e => setForm({ ...form, url: e.target.value })}
            placeholder="https://example.com/webhook" />

          <div>
            <label className="block text-sm font-medium text-slate-700 mb-2">
              {t('events') || 'Events'}
            </label>
            <div className="grid grid-cols-2 gap-1.5 max-h-48 overflow-y-auto">
              {EVENT_OPTIONS.map(opt => (
                <label key={opt.value} className="flex items-center gap-2 p-1.5 rounded hover:bg-slate-50 cursor-pointer">
                  <input type="checkbox" checked={form.events.includes(opt.value)}
                    onChange={() => toggleEvent(opt.value)}
                    className="rounded border-slate-300 text-blue-600" />
                  <span className="text-xs text-slate-700">{opt.label}</span>
                </label>
              ))}
            </div>
          </div>

          <Input label={t('secret') || 'Secret (optional)'} type="password" value={form.secret}
            onChange={e => setForm({ ...form, secret: e.target.value })}
            placeholder={t('webhook_secret_hint') || 'HMAC secret for verification'} />

          <label className="flex items-center gap-2">
            <input type="checkbox" checked={form.active}
              onChange={e => setForm({ ...form, active: e.target.checked })}
              className="rounded border-slate-300 text-blue-600" />
            <span className="text-sm text-slate-700">{t('active') || 'Active'}</span>
          </label>

          <div className="flex justify-end gap-3 pt-4">
            <Button variant="ghost" onClick={() => setShowModal(false)}>{t('cancel') || 'Cancel'}</Button>
            <Button onClick={handleSave} disabled={!form.name || !form.url}>
              {editing ? (t('save') || 'Save') : (t('create') || 'Create')}
            </Button>
          </div>
        </div>
      </Modal>

      {ConfirmDialog}
    </div>
  );
}
