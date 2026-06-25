import React, { useState, useEffect } from 'react';
import { useTranslation } from 'react-i18next';
import { useSearchParams, useNavigate } from 'react-router-dom';
import { request } from '../services/api';
import { Card, Button, Input, Badge } from '../components/ui';
import {
  ClipboardList, Send, Camera, CheckCircle,
  Loader2, AlertTriangle,
} from 'lucide-react';

export function WorkRequestPortal() {
  const { t } = useTranslation();
  const [searchParams] = useSearchParams();
  const navigate = useNavigate();
  const deviceId = searchParams.get('device_id') || '';

  const [form, setForm] = useState({ device_id: deviceId, name: '', email: '', phone: '', description: '' });
  const [submitting, setSubmitting] = useState(false);
  const [submitted, setSubmitted] = useState(false);
  const [recaptchaToken, setRecaptchaToken] = useState('');
  const [error, setError] = useState<string | null>(null);
  const [deviceName, setDeviceName] = useState('');

  useEffect(() => {
    if (deviceId) {
      request<{ name?: string }>(`/devices/${deviceId}`)
        .then(d => setDeviceName(d?.name || deviceId))
        .catch(() => setDeviceName(deviceId));
    }
  }, [deviceId]);

  const handleSubmit = async () => {
    if (!form.description.trim()) return;
    setSubmitting(true);
    setError(null);
    try {
      await request('/work-requests', {
        method: 'POST',
        body: JSON.stringify({
          device_id: form.device_id || undefined,
          requester_name: form.name || undefined,
          requester_email: form.email || undefined,
          requester_phone: form.phone || undefined,
          description: form.description,
          recaptcha_token: recaptchaToken || undefined,
          source: 'public_portal',
        }),
      });
      setSubmitted(true);
    } catch (err: any) {
      setError(err.message || 'Failed to submit request');
    } finally {
      setSubmitting(false);
    }
  };

  if (submitted) {
    return (
      <div className="min-h-screen bg-slate-50 flex items-center justify-center p-4">
        <Card>
          <div className="p-8 text-center max-w-md">
            <div className="w-16 h-16 mx-auto mb-4 bg-emerald-100 rounded-full flex items-center justify-center">
              <CheckCircle className="w-8 h-8 text-emerald-600" />
            </div>
            <h1 className="text-xl font-bold text-slate-900 mb-2">
              {t('request_submitted') || 'Заявка отправлена!'}
            </h1>
            <p className="text-sm text-slate-500 mb-6">
              {t('request_submitted_desc') || 'Мы свяжемся с вами в ближайшее время'}
            </p>
            <Button onClick={() => navigate('/')}>
              {t('back_to_home') || 'На главную'}
            </Button>
          </div>
        </Card>
      </div>
    );
  }

  return (
    <div className="min-h-screen bg-slate-50 flex items-center justify-center p-4">
      <Card>
        <div className="p-8 max-w-md w-full">
          {/* Header */}
          <div className="text-center mb-6">
            <div className="w-14 h-14 mx-auto mb-3 bg-blue-100 rounded-xl flex items-center justify-center">
              <ClipboardList className="w-7 h-7 text-blue-600" />
            </div>
            <h1 className="text-xl font-bold text-slate-900">
              {t('submit_work_request') || 'Подать заявку'}
            </h1>
            <p className="text-sm text-slate-500 mt-1">
              {t('work_request_desc') || 'Опишите проблему — мы направим техника'}
            </p>
          </div>

          {/* Device info */}
          {deviceId && (
            <div className="p-3 bg-blue-50 rounded-lg border border-blue-200 mb-4 flex items-center gap-2">
              <Camera className="w-4 h-4 text-blue-600" />
              <span className="text-sm text-blue-800 font-medium">{deviceName || deviceId}</span>
            </div>
          )}

          {/* Form */}
          <div className="space-y-3">
            <Input
              label={t('device_id') || 'ID устройства'}
              value={form.device_id}
              onChange={e => setForm({ ...form, device_id: e.target.value })}
              placeholder="Например: CAM-001"
            />
            <Input
              label={t('your_name') || 'Ваше имя'}
              value={form.name}
              onChange={e => setForm({ ...form, name: e.target.value })}
              placeholder="Иван Иванов"
            />
            <div className="grid grid-cols-2 gap-3">
              <Input
                label="Email"
                type="email"
                value={form.email}
                onChange={e => setForm({ ...form, email: e.target.value })}
                placeholder="ivan@example.com"
              />
              <Input
                label={t('phone') || 'Телефон'}
                value={form.phone}
                onChange={e => setForm({ ...form, phone: e.target.value })}
                placeholder="+375 29 123-45-67"
              />
            </div>
            <div>
              <label className="block text-sm font-medium text-slate-700 mb-1">
                {t('problem_description') || 'Описание проблемы'} *
              </label>
              <textarea
                className="w-full rounded-lg border border-slate-300 px-3 py-2 text-sm focus:ring-2 focus:ring-blue-500 min-h-[100px]"
                value={form.description}
                onChange={e => setForm({ ...form, description: e.target.value })}
                placeholder={t('problem_description_placeholder') || 'Опишите что случилось...'}
              />
            </div>

            {error && (
              <div className="p-3 bg-red-50 rounded-lg border border-red-200 text-sm text-red-700 flex items-center gap-2">
                <AlertTriangle className="w-4 h-4" />
                {error}
              </div>
            )}

            <Button
              className="w-full"
              icon={submitting ? <Loader2 className="w-4 h-4 animate-spin" /> : <Send className="w-4 h-4" />}
              onClick={handleSubmit}
              disabled={!form.description.trim() || submitting}
              loading={submitting}
            >
              {t('submit') || 'Отправить'}
            </Button>
          </div>

          {/* Footer */}
          <p className="text-[10px] text-slate-400 text-center mt-4">
            {t('request_privacy') || 'Нажимая "Отправить", вы соглашаетесь на обработку данных'}
          </p>
        </div>
      </Card>
    </div>
  );
}
