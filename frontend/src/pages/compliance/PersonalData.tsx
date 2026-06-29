import React, { useEffect, useState, useCallback } from 'react';
import { useTranslation } from 'react-i18next';
import { request } from '../../services/api';
import {
  Card,
  Badge,
  Button,
  Input,
  Modal,
  EmptyState,
  Tabs,
} from '../../components/ui';
import {
  UserCheck,
  FileText,
  Download,
  Plus,
  Eye,
  XCircle,
  CheckCircle,
  Clock,
  AlertTriangle,
  Shield,
  Database,
  FileSpreadsheet,
} from '../components/ui/Icons';

// ═══════════════════════════════════════════════════════════════════════
// Types
// ═══════════════════════════════════════════════════════════════════════

interface ConsentRecord {
  id: string;
  subject_id: string;
  subject_name: string;
  purpose: string;
  status: 'granted' | 'revoked' | 'expired' | 'pending';
  granted_at: string;
  revoked_at?: string;
  expires_at?: string;
  source: string;
  notes?: string;
}

interface DSARRequest {
  id: string;
  subject_id: string;
  subject_name: string;
  subject_email: string;
  request_type: string;
  description: string;
  status: 'new' | 'verified' | 'in_review' | 'gathering' | 'fulfilled' | 'rejected' | 'expired';
  deadline_at: string;
  assigned_to?: string;
  rejection_reason?: string;
  created_at: string;
}

interface DataInventoryItem {
  id: string;
  category: string;
  description: string;
  data_fields: string[];
  storage_location: string;
  purpose: string;
  retention_days: number;
  anonymized: boolean;
  encrypted: boolean;
  legal_basis: string;
}

interface RoskomnadzorReport {
  operator_name: string;
  operator_inn: string;
  data_categories: string[];
  subject_count: number;
  processing_purposes: string[];
  cross_border: boolean;
  data_retention_days: number;
  security_measures: string[];
  dpia_completed: boolean;
  generated_at: string;
}

// ═══════════════════════════════════════════════════════════════════════
// Constants
// ═══════════════════════════════════════════════════════════════════════

const CONSENT_PURPOSE_LABELS: Record<string, string> = {
  video_monitoring: 'Видеонаблюдение',
  analytics: 'Аналитика поведения',
  access_control: 'Контроль доступа',
  regulatory_compliance: 'Соответствие регуляторам',
  emergency_response: 'Реагирование на ЧС',
  data_retention: 'Хранение архива',
  third_party_sharing: 'Передача третьим лицам',
};

const CATEGORY_LABELS: Record<string, string> = {
  biometric: 'Биометрические',
  location: 'Геолокация',
  identity: 'Персональные (ФИО)',
  contact: 'Контактные',
  schedule: 'Рабочий график',
  credentials: 'Учётные данные',
  video_archive: 'Видеоархив',
};

// ═══════════════════════════════════════════════════════════════════════
// Sub-components
// ═══════════════════════════════════════════════════════════════════════

function ConsentBadge({ status }: { status: string }) {
  const config: Record<string, { bg: string; text: string; label: string }> = {
    granted: { bg: 'bg-emerald-100 dark:bg-emerald-900/30', text: 'text-emerald-700 dark:text-emerald-400', label: 'Получено' },
    revoked: { bg: 'bg-red-100 dark:bg-red-900/30', text: 'text-red-700 dark:text-red-400', label: 'Отозвано' },
    expired: { bg: 'bg-slate-100 dark:bg-slate-700', text: 'text-slate-600 dark:text-slate-400', label: 'Истекло' },
    pending: { bg: 'bg-amber-100 dark:bg-amber-900/30', text: 'text-amber-700 dark:text-amber-400', label: 'Ожидает' },
  };
  const c = config[status] || config.pending;
  return (
    <span className={`inline-flex items-center gap-1 px-2 py-0.5 rounded text-xs font-medium ${c.bg} ${c.text}`}>
      {c.label}
    </span>
  );
}

function DSARBadge({ status }: { status: string }) {
  const config: Record<string, { bg: string; text: string; label: string }> = {
    new: { bg: 'bg-blue-100 dark:bg-blue-900/30', text: 'text-blue-700 dark:text-blue-400', label: 'Новый' },
    verified: { bg: 'bg-indigo-100 dark:bg-indigo-900/30', text: 'text-indigo-700 dark:text-indigo-400', label: 'Подтверждён' },
    in_review: { bg: 'bg-amber-100 dark:bg-amber-900/30', text: 'text-amber-700 dark:text-amber-400', label: 'На проверке' },
    gathering: { bg: 'bg-purple-100 dark:bg-purple-900/30', text: 'text-purple-700 dark:text-purple-400', label: 'Сбор данных' },
    fulfilled: { bg: 'bg-emerald-100 dark:bg-emerald-900/30', text: 'text-emerald-700 dark:text-emerald-400', label: 'Выполнен' },
    rejected: { bg: 'bg-red-100 dark:bg-red-900/30', text: 'text-red-700 dark:text-red-400', label: 'Отклонён' },
    expired: { bg: 'bg-slate-100 dark:bg-slate-700', text: 'text-slate-600 dark:text-slate-400', label: 'Просрочен' },
  };
  const c = config[status] || config.new;
  return (
    <span className={`inline-flex items-center px-2 py-0.5 rounded text-xs font-medium ${c.bg} ${c.text}`}>
      {c.label}
    </span>
  );
}

// ═══════════════════════════════════════════════════════════════════════
// Consent Tab
// ═══════════════════════════════════════════════════════════════════════

function ConsentTab() {
  const { t } = useTranslation();
  const [consents, setConsents] = useState<ConsentRecord[]>([]);
  const [searchQuery, setSearchQuery] = useState('');
  const [showGrantModal, setShowGrantModal] = useState(false);
  const [loading, setLoading] = useState(true);

  const fetchData = useCallback(async (subjectId?: string) => {
    try {
      const params = subjectId ? `?subject_id=${encodeURIComponent(subjectId)}` : '?subject_id=all';
      const data = await request<ConsentRecord[]>(`/compliance/personal-data/consent${params}`);
      setConsents(data || []);
    } catch {
      // ignore
    } finally {
      setLoading(false);
    }
  }, []);

  useEffect(() => { fetchData(); }, [fetchData]);

  const handleGrant = async (e: React.FormEvent<HTMLFormElement>) => {
    e.preventDefault();
    const form = e.currentTarget;
    const fd = new FormData(form);
    try {
      await request('/compliance/personal-data/consent', {
        method: 'POST',
        body: JSON.stringify({
          subject_id: fd.get('subject_id'),
          subject_name: fd.get('subject_name'),
          purpose: fd.get('purpose'),
          source: fd.get('source') || 'web',
          expires_in_days: parseInt(fd.get('expires_in_days') as string) || 0,
        }),
      });
      setShowGrantModal(false);
      fetchData();
    } catch { alert('Ошибка при получении согласия'); }
  };

  const handleRevoke = async (consentId: string) => {
    if (!confirm('Отозвать согласие?')) return;
    try {
      await request('/compliance/personal-data/consent/revoke', {
        method: 'POST',
        body: JSON.stringify({ consent_id: consentId }),
      });
      fetchData();
    } catch { alert('Ошибка при отзыве согласия'); }
  };

  if (loading) {
    return <div className="h-48 animate-pulse bg-slate-100 dark:bg-slate-800 rounded-lg" />;
  }

  return (
    <div className="space-y-4">
      <div className="flex items-center justify-between">
        <Input
          placeholder="Поиск по subject_id..."
          value={searchQuery}
          onChange={(e) => setSearchQuery(e.target.value)}
          className="w-64"
        />
        <Button onClick={() => setShowGrantModal(true)}>
          <Plus className="w-4 h-4 mr-1" /> Получить согласие
        </Button>
      </div>

      {consents.length === 0 ? (
        <EmptyState icon={<UserCheck className="w-12 h-12" />} title="Нет согласий"
          description="Зарегистрируйте согласие на обработку ПД"
          action={{ label: 'Получить согласие', onClick: () => setShowGrantModal(true) }} />
      ) : (
        <Card>
          <div className="overflow-x-auto">
            <table className="w-full text-sm">
              <thead>
                <tr className="border-b border-slate-200 dark:border-slate-700">
                  <th className="text-left p-3 font-medium text-slate-600 dark:text-slate-400">Субъект</th>
                  <th className="text-left p-3 font-medium text-slate-600 dark:text-slate-400">Цель</th>
                  <th className="text-left p-3 font-medium text-slate-600 dark:text-slate-400">Статус</th>
                  <th className="text-left p-3 font-medium text-slate-600 dark:text-slate-400">Получено</th>
                  <th className="text-left p-3 font-medium text-slate-600 dark:text-slate-400">Истекает</th>
                  <th className="text-left p-3 font-medium text-slate-600 dark:text-slate-400">Источник</th>
                  <th className="text-right p-3 font-medium text-slate-600 dark:text-slate-400">Действия</th>
                </tr>
              </thead>
              <tbody>
                {consents
                  .filter(c => !searchQuery || c.subject_id.includes(searchQuery) || c.subject_name?.includes(searchQuery))
                  .map((consent) => (
                    <tr key={consent.id} className="border-b border-slate-100 dark:border-slate-800 hover:bg-slate-50 dark:hover:bg-slate-800/50">
                      <td className="p-3">
                        <div className="font-medium">{consent.subject_name || consent.subject_id}</div>
                        <div className="text-xs text-slate-500">{consent.subject_id}</div>
                      </td>
                      <td className="p-3">{CONSENT_PURPOSE_LABELS[consent.purpose] || consent.purpose}</td>
                      <td className="p-3"><ConsentBadge status={consent.status} /></td>
                      <td className="p-3 text-slate-600 dark:text-slate-400">{new Date(consent.granted_at).toLocaleDateString()}</td>
                      <td className="p-3 text-slate-600 dark:text-slate-400">
                        {consent.expires_at ? new Date(consent.expires_at).toLocaleDateString() : '—'}
                      </td>
                      <td className="p-3 text-slate-600 dark:text-slate-400">{consent.source}</td>
                      <td className="p-3 text-right">
                        {consent.status === 'granted' && (
                          <button onClick={() => handleRevoke(consent.id)}
                            className="text-red-600 hover:text-red-800 text-xs font-medium">
                            Отозвать
                          </button>
                        )}
                      </td>
                    </tr>
                  ))}
              </tbody>
            </table>
          </div>
        </Card>
      )}

      <Modal isOpen={showGrantModal} onClose={() => setShowGrantModal(false)} title="Получение согласия на обработку ПД">
        <form onSubmit={handleGrant} className="space-y-4">
          <div>
            <label className="block text-sm font-medium mb-1">Subject ID *</label>
            <Input name="subject_id" required placeholder="ID субъекта ПД" />
          </div>
          <div>
            <label className="block text-sm font-medium mb-1">ФИО субъекта</label>
            <Input name="subject_name" placeholder="Иванов Иван Иванович" />
          </div>
          <div>
            <label className="block text-sm font-medium mb-1">Цель обработки *</label>
            <select name="purpose" required className="w-full rounded-lg border border-slate-300 dark:border-slate-600 bg-white dark:bg-slate-800 p-2 text-sm">
              {Object.entries(CONSENT_PURPOSE_LABELS).map(([key, label]) => (
                <option key={key} value={key}>{label}</option>
              ))}
            </select>
          </div>
          <div>
            <label className="block text-sm font-medium mb-1">Источник</label>
            <select name="source" className="w-full rounded-lg border border-slate-300 dark:border-slate-600 bg-white dark:bg-slate-800 p-2 text-sm">
              <option value="web">Веб-интерфейс</option>
              <option value="mobile">Мобильное приложение</option>
              <option value="paper">Бумажный носитель</option>
            </select>
          </div>
          <div>
            <label className="block text-sm font-medium mb-1">Срок действия (дней, 0 = без срока)</label>
            <Input name="expires_in_days" type="number" min="0" defaultValue="365" />
          </div>
          <div className="flex justify-end gap-2 pt-2">
            <Button type="button" variant="secondary" onClick={() => setShowGrantModal(false)}>Отмена</Button>
            <Button type="submit">Сохранить</Button>
          </div>
        </form>
      </Modal>
    </div>
  );
}

// ═══════════════════════════════════════════════════════════════════════
// DSAR Tab
// ═══════════════════════════════════════════════════════════════════════

function DSARTab() {
  const [dsars, setDsars] = useState<DSARRequest[]>([]);
  const [searchQuery, setSearchQuery] = useState('');
  const [showSubmitModal, setShowSubmitModal] = useState(false);
  const [showFulfillModal, setShowFulfillModal] = useState(false);
  const [selectedDSAR, setSelectedDSAR] = useState<DSARRequest | null>(null);
  const [fulfillData, setFulfillData] = useState('');
  const [loading, setLoading] = useState(false);

  const fetchData = useCallback(async (subjectId: string) => {
    if (!subjectId) { setDsars([]); return; }
    setLoading(true);
    try {
      const data = await request<DSARRequest[]>(`/compliance/personal-data/dsar?subject_id=${encodeURIComponent(subjectId)}`);
      setDsars(data || []);
    } catch { /* ignore */ } finally { setLoading(false); }
  }, []);

  const handleSubmit = async (e: React.FormEvent<HTMLFormElement>) => {
    e.preventDefault();
    const fd = new FormData(e.currentTarget);
    try {
      const data = await request<DSARRequest>('/compliance/personal-data/dsar', {
        method: 'POST',
        body: JSON.stringify({
          subject_id: fd.get('subject_id'),
          subject_name: fd.get('subject_name'),
          subject_email: fd.get('subject_email'),
          request_type: fd.get('request_type'),
          description: fd.get('description'),
        }),
      });
      setShowSubmitModal(false);
      if (data) fetchData(data.subject_id);
    } catch { alert('Ошибка при подаче DSAR'); }
  };

  const handleFulfill = async () => {
    if (!selectedDSAR) return;
    try {
      await request('/compliance/personal-data/dsar/fulfill', {
        method: 'POST',
        body: JSON.stringify({ dsar_id: selectedDSAR.id, response_data: fulfillData }),
      });
      setShowFulfillModal(false);
      setSelectedDSAR(null);
      setFulfillData('');
      fetchData(selectedDSAR.subject_id);
    } catch { alert('Ошибка при выполнении DSAR'); }
  };

  return (
    <div className="space-y-4">
      <div className="flex items-center justify-between">
        <Input
          placeholder="Subject ID для поиска DSAR..."
          value={searchQuery}
          onChange={(e) => {
            setSearchQuery(e.target.value);
            if (e.target.value) fetchData(e.target.value);
          }}
          className="w-64"
        />
        <Button onClick={() => setShowSubmitModal(true)}>
          <Plus className="w-4 h-4 mr-1" /> Новый DSAR
        </Button>
      </div>

      {loading ? (
        <div className="h-48 animate-pulse bg-slate-100 dark:bg-slate-800 rounded-lg" />
      ) : dsars.length === 0 ? (
        <EmptyState icon={<FileText className="w-12 h-12" />} title="Нет DSAR запросов"
          description="Введите Subject ID для поиска или создайте новый запрос"
          action={{ label: 'Создать DSAR', onClick: () => setShowSubmitModal(true) }} />
      ) : (
        <Card>
          <div className="overflow-x-auto">
            <table className="w-full text-sm">
              <thead>
                <tr className="border-b border-slate-200 dark:border-slate-700">
                  <th className="text-left p-3 font-medium">Субъект</th>
                  <th className="text-left p-3 font-medium">Тип</th>
                  <th className="text-left p-3 font-medium">Статус</th>
                  <th className="text-left p-3 font-medium">Дедлайн</th>
                  <th className="text-left p-3 font-medium">Email</th>
                  <th className="text-right p-3 font-medium">Действия</th>
                </tr>
              </thead>
              <tbody>
                {dsars.map((dsar) => (
                  <tr key={dsar.id} className="border-b border-slate-100 dark:border-slate-800 hover:bg-slate-50 dark:hover:bg-slate-800/50">
                    <td className="p-3">
                      <div className="font-medium">{dsar.subject_name || dsar.subject_id}</div>
                      <div className="text-xs text-slate-500">{dsar.subject_id}</div>
                    </td>
                    <td className="p-3">{dsar.request_type}</td>
                    <td className="p-3"><DSARBadge status={dsar.status} /></td>
                    <td className="p-3 text-slate-600 dark:text-slate-400">{new Date(dsar.deadline_at).toLocaleDateString()}</td>
                    <td className="p-3 text-slate-600 dark:text-slate-400">{dsar.subject_email}</td>
                    <td className="p-3 text-right">
                      <div className="flex justify-end gap-1">
                        <button onClick={() => setSelectedDSAR(dsar)}
                          className="text-blue-600 hover:text-blue-800 text-xs font-medium px-2">
                          <Eye className="w-4 h-4 inline mr-1" />Детали
                        </button>
                        {dsar.status === 'gathering' && (
                          <button onClick={() => { setSelectedDSAR(dsar); setShowFulfillModal(true); }}
                            className="text-emerald-600 hover:text-emerald-800 text-xs font-medium px-2">
                            Выполнить
                          </button>
                        )}
                      </div>
                    </td>
                  </tr>
                ))}
              </tbody>
            </table>
          </div>
        </Card>
      )}

      <Modal isOpen={showSubmitModal} onClose={() => setShowSubmitModal(false)} title="Новый DSAR запрос">
        <form onSubmit={handleSubmit} className="space-y-4">
          <div>
            <label className="block text-sm font-medium mb-1">Subject ID *</label>
            <Input name="subject_id" required placeholder="ID субъекта ПД" />
          </div>
          <div>
            <label className="block text-sm font-medium mb-1">ФИО</label>
            <Input name="subject_name" placeholder="Иванов Иван Иванович" />
          </div>
          <div>
            <label className="block text-sm font-medium mb-1">Email</label>
            <Input name="subject_email" type="email" placeholder="email@example.com" />
          </div>
          <div>
            <label className="block text-sm font-medium mb-1">Тип запроса *</label>
            <select name="request_type" required className="w-full rounded-lg border border-slate-300 dark:border-slate-600 bg-white dark:bg-slate-800 p-2 text-sm">
              <option value="access">Доступ к данным</option>
              <option value="rectification">Исправление данных</option>
              <option value="erasure">Удаление данных</option>
              <option value="restriction">Ограничение обработки</option>
              <option value="portability">Переносимость данных</option>
            </select>
          </div>
          <div>
            <label className="block text-sm font-medium mb-1">Описание</label>
            <textarea name="description" rows={3} className="w-full rounded-lg border border-slate-300 dark:border-slate-600 bg-white dark:bg-slate-800 p-2 text-sm" />
          </div>
          <div className="flex justify-end gap-2 pt-2">
            <Button type="button" variant="secondary" onClick={() => setShowSubmitModal(false)}>Отмена</Button>
            <Button type="submit">Отправить</Button>
          </div>
        </form>
      </Modal>

      <Modal isOpen={showFulfillModal} onClose={() => setShowFulfillModal(false)}
        title={selectedDSAR ? `Выполнение DSAR: ${selectedDSAR.id}` : ''}>
        <div className="space-y-4">
          <p className="text-sm text-slate-600 dark:text-slate-400">
            Предоставьте данные субъекту ПД по запросу {selectedDSAR?.request_type}
          </p>
          <div>
            <label className="block text-sm font-medium mb-1">Данные для ответа (JSON)</label>
            <textarea value={fulfillData} onChange={(e) => setFulfillData(e.target.value)}
              rows={8}
              className="w-full rounded-lg border border-slate-300 dark:border-slate-600 bg-white dark:bg-slate-800 p-2 text-sm font-mono"
              placeholder='{"data": {...}}' />
          </div>
          <div className="flex justify-end gap-2">
            <Button variant="secondary" onClick={() => setShowFulfillModal(false)}>Отмена</Button>
            <Button onClick={handleFulfill}>Подтвердить выполнение</Button>
          </div>
        </div>
      </Modal>
    </div>
  );
}

// ═══════════════════════════════════════════════════════════════════════
// Inventory Tab
// ═══════════════════════════════════════════════════════════════════════

function InventoryTab() {
  const [inventory, setInventory] = useState<DataInventoryItem[]>([]);
  const [showRegisterModal, setShowRegisterModal] = useState(false);
  const [showReportModal, setShowReportModal] = useState(false);
  const [reportData, setReportData] = useState<RoskomnadzorReport | null>(null);
  const [loading, setLoading] = useState(true);

  const fetchData = useCallback(async (anonymize = false) => {
    setLoading(true);
    try {
      const params = anonymize ? '?anonymize=true' : '';
      const data = await request<DataInventoryItem[]>(`/compliance/personal-data/inventory${params}`);
      setInventory(data || []);
    } catch { /* ignore */ } finally { setLoading(false); }
  }, []);

  useEffect(() => { fetchData(); }, [fetchData]);

  const handleRegister = async (e: React.FormEvent<HTMLFormElement>) => {
    e.preventDefault();
    const fd = new FormData(e.currentTarget);
    try {
      await request('/compliance/personal-data/inventory', {
        method: 'POST',
        body: JSON.stringify({
          category: fd.get('category'),
          description: fd.get('description'),
          data_fields: (fd.get('data_fields') as string).split(',').map(s => s.trim()),
          storage_location: fd.get('storage_location'),
          purpose: fd.get('purpose'),
          retention_days: parseInt(fd.get('retention_days') as string) || 90,
          legal_basis: fd.get('legal_basis'),
        }),
      });
      setShowRegisterModal(false);
      fetchData();
    } catch { alert('Ошибка при регистрации элемента'); }
  };

  const handleGenerateReport = async () => {
    const operatorName = prompt('Наименование оператора:');
    const operatorINN = prompt('ИНН оператора:');
    if (!operatorName || !operatorINN) return;
    try {
      const report = await request<RoskomnadzorReport>('/compliance/personal-data/report/rkn', {
        method: 'POST',
        body: JSON.stringify({ operator_name: operatorName, operator_inn: operatorINN, operator_address: '', subject_count: 0 }),
      });
      setReportData(report);
      setShowReportModal(true);
    } catch { alert('Ошибка при генерации отчёта'); }
  };

  const handleExport = (format: 'json' | 'csv', anonymize: boolean) => {
    const params = new URLSearchParams({ format, anonymize: anonymize.toString() });
    window.open(`/api/v1/compliance/personal-data/inventory/export?${params}`, '_blank');
  };

  if (loading) {
    return <div className="h-48 animate-pulse bg-slate-100 dark:bg-slate-800 rounded-lg" />;
  }

  return (
    <div className="space-y-4">
      <div className="flex items-center justify-between flex-wrap gap-2">
        <div className="flex items-center gap-2">
          <Button variant="secondary" onClick={() => handleExport('json', false)}>
            <Download className="w-4 h-4 mr-1" /> JSON
          </Button>
          <Button variant="secondary" onClick={() => handleExport('csv', false)}>
            <FileSpreadsheet className="w-4 h-4 mr-1" /> CSV
          </Button>
          <Button variant="secondary" onClick={() => handleExport('json', true)}>
            <Shield className="w-4 h-4 mr-1" /> JSON (обезличенный)
          </Button>
        </div>
        <div className="flex items-center gap-2">
          <Button onClick={handleGenerateReport}>
            <FileText className="w-4 h-4 mr-1" /> Отчёт РКН
          </Button>
          <Button onClick={() => setShowRegisterModal(true)}>
            <Plus className="w-4 h-4 mr-1" /> Добавить
          </Button>
        </div>
      </div>

      {inventory.length === 0 ? (
        <EmptyState icon={<Database className="w-12 h-12" />} title="Реестр ПД пуст"
          description="Зарегистрируйте категории персональных данных"
          action={{ label: 'Добавить элемент', onClick: () => setShowRegisterModal(true) }} />
      ) : (
        <Card>
          <div className="overflow-x-auto">
            <table className="w-full text-sm">
              <thead>
                <tr className="border-b border-slate-200 dark:border-slate-700">
                  <th className="text-left p-3 font-medium">Категория</th>
                  <th className="text-left p-3 font-medium">Описание</th>
                  <th className="text-left p-3 font-medium">Поля</th>
                  <th className="text-left p-3 font-medium">Хранение</th>
                  <th className="text-left p-3 font-medium">Срок (дн)</th>
                  <th className="text-center p-3 font-medium">Зашифровано</th>
                  <th className="text-center p-3 font-medium">Обезличено</th>
                </tr>
              </thead>
              <tbody>
                {inventory.map((item) => (
                  <tr key={item.id} className="border-b border-slate-100 dark:border-slate-800 hover:bg-slate-50 dark:hover:bg-slate-800/50">
                    <td className="p-3">
                      <span className="inline-flex items-center px-2 py-0.5 rounded text-xs font-medium bg-slate-100 dark:bg-slate-700 text-slate-700 dark:text-slate-300">
                        {CATEGORY_LABELS[item.category] || item.category}
                      </span>
                    </td>
                    <td className="p-3">{item.description}</td>
                    <td className="p-3">
                      <div className="flex flex-wrap gap-1">
                        {item.data_fields.map((field, i) => (
                          <span key={i} className="inline-flex items-center px-2 py-0.5 rounded text-xs font-medium border border-slate-300 dark:border-slate-600 text-slate-600 dark:text-slate-400">
                            {field}
                          </span>
                        ))}
                      </div>
                    </td>
                    <td className="p-3 text-slate-600 dark:text-slate-400">{item.storage_location}</td>
                    <td className="p-3 text-slate-600 dark:text-slate-400">{item.retention_days}</td>
                    <td className="p-3 text-center">
                      {item.encrypted
                        ? <CheckCircle className="w-4 h-4 text-emerald-500 inline" />
                        : <XCircle className="w-4 h-4 text-red-500 inline" />}
                    </td>
                    <td className="p-3 text-center">
                      {item.anonymized
                        ? <CheckCircle className="w-4 h-4 text-emerald-500 inline" />
                        : <span className="text-slate-400">—</span>}
                    </td>
                  </tr>
                ))}
              </tbody>
            </table>
          </div>
        </Card>
      )}

      <Modal isOpen={showRegisterModal} onClose={() => setShowRegisterModal(false)} title="Регистрация категории ПД">
        <form onSubmit={handleRegister} className="space-y-4">
          <div>
            <label className="block text-sm font-medium mb-1">Категория *</label>
            <select name="category" required className="w-full rounded-lg border border-slate-300 dark:border-slate-600 bg-white dark:bg-slate-800 p-2 text-sm">
              {Object.entries(CATEGORY_LABELS).map(([key, label]) => (
                <option key={key} value={key}>{label}</option>
              ))}
            </select>
          </div>
          <div>
            <label className="block text-sm font-medium mb-1">Описание</label>
            <Input name="description" placeholder="Например: IP-камеры в зоне контроля доступа" />
          </div>
          <div>
            <label className="block text-sm font-medium mb-1">Поля данных * (через запятую)</label>
            <Input name="data_fields" required placeholder="video_feed, face_image, timestamp" />
          </div>
          <div>
            <label className="block text-sm font-medium mb-1">Место хранения</label>
            <Input name="storage_location" placeholder="PostgreSQL / TimescaleDB / S3" />
          </div>
          <div>
            <label className="block text-sm font-medium mb-1">Цель обработки</label>
            <select name="purpose" className="w-full rounded-lg border border-slate-300 dark:border-slate-600 bg-white dark:bg-slate-800 p-2 text-sm">
              {Object.entries(CONSENT_PURPOSE_LABELS).map(([key, label]) => (
                <option key={key} value={key}>{label}</option>
              ))}
            </select>
          </div>
          <div className="grid grid-cols-2 gap-4">
            <div>
              <label className="block text-sm font-medium mb-1">Срок хранения (дн)</label>
              <Input name="retention_days" type="number" min="1" defaultValue="90" />
            </div>
            <div>
              <label className="block text-sm font-medium mb-1">Правовое основание</label>
              <Input name="legal_basis" placeholder="152-ФЗ ст. 6" />
            </div>
          </div>
          <div className="flex justify-end gap-2 pt-2">
            <Button type="button" variant="secondary" onClick={() => setShowRegisterModal(false)}>Отмена</Button>
            <Button type="submit">Сохранить</Button>
          </div>
        </form>
      </Modal>

      <Modal isOpen={showReportModal} onClose={() => setShowReportModal(false)} title="Отчёт для Роскомнадзора">
        {reportData && (
          <div className="space-y-4">
            <div className="grid grid-cols-2 gap-4">
              <div>
                <label className="text-xs text-slate-500">Оператор</label>
                <p className="font-medium">{reportData.operator_name}</p>
              </div>
              <div>
                <label className="text-xs text-slate-500">ИНН</label>
                <p className="font-medium">{reportData.operator_inn}</p>
              </div>
            </div>
            <div>
              <label className="text-xs text-slate-500">Категории ПД</label>
              <div className="flex flex-wrap gap-1 mt-1">
                {reportData.data_categories.map((cat, i) => (
                  <span key={i} className="inline-flex items-center px-2 py-0.5 rounded text-xs font-medium bg-slate-100 dark:bg-slate-700">{CATEGORY_LABELS[cat] || cat}</span>
                ))}
              </div>
            </div>
            <div>
              <label className="text-xs text-slate-500">Цели обработки</label>
              <div className="flex flex-wrap gap-1 mt-1">
                {reportData.processing_purposes.map((p, i) => (
                  <span key={i} className="inline-flex items-center px-2 py-0.5 rounded text-xs font-medium bg-slate-100 dark:bg-slate-700">{CONSENT_PURPOSE_LABELS[p] || p}</span>
                ))}
              </div>
            </div>
            <div className="grid grid-cols-3 gap-4">
              <div><label className="text-xs text-slate-500">Субъектов</label><p className="font-medium">{reportData.subject_count}</p></div>
              <div><label className="text-xs text-slate-500">Срок хранения (дн)</label><p className="font-medium">{reportData.data_retention_days}</p></div>
              <div><label className="text-xs text-slate-500">Трансграничная</label><p className="font-medium">{reportData.cross_border ? 'Да' : 'Нет'}</p></div>
            </div>
            <div>
              <label className="text-xs text-slate-500">Меры защиты</label>
              <ul className="list-disc list-inside text-sm mt-1">
                {reportData.security_measures.map((m, i) => <li key={i}>{m}</li>)}
              </ul>
            </div>
            <p className="text-xs text-slate-400">Сгенерирован: {new Date(reportData.generated_at).toLocaleString()}</p>
          </div>
        )}
      </Modal>
    </div>
  );
}

// ═══════════════════════════════════════════════════════════════════════
// Main PersonalData Component
// ═══════════════════════════════════════════════════════════════════════

export default function PersonalData() {
  const { t } = useTranslation();
  const [activeTab, setActiveTab] = useState('consent');

  const tabs = [
    { id: 'consent', label: 'Согласия (152-ФЗ)', icon: <UserCheck className="w-4 h-4" /> },
    { id: 'dsar', label: 'DSAR (ст. 14)', icon: <FileText className="w-4 h-4" /> },
    { id: 'inventory', label: 'Реестр ПД', icon: <Database className="w-4 h-4" /> },
  ];

  return (
    <div className="p-6 space-y-6">
      <div className="flex items-center justify-between">
        <div>
          <h1 className="text-2xl font-bold text-slate-900 dark:text-white">152-ФЗ: Персональные данные</h1>
          <p className="text-sm text-slate-500 dark:text-slate-400 mt-1">
            Управление согласиями, DSAR запросами и реестром ПД
          </p>
        </div>
      </div>

      <Tabs tabs={tabs} activeTab={activeTab} onChange={setActiveTab}>
        {activeTab === 'consent' && <ConsentTab />}
        {activeTab === 'dsar' && <DSARTab />}
        {activeTab === 'inventory' && <InventoryTab />}
      </Tabs>
    </div>
  );
}
