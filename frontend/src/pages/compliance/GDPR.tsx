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
  Trash2,
  Download,
  FileText,
  Plus,
  Eye,
  CheckCircle,
  XCircle,
  Clock,
  AlertTriangle,
  Shield,
  Globe,
  Search,
} from '../components/ui/Icons';

// ═══════════════════════════════════════════════════════════════════════
// Types
// ═══════════════════════════════════════════════════════════════════════

interface ErasureRequest {
  id: string;
  subject_id: string;
  subject_name: string;
  subject_email: string;
  scope: string;
  specific_systems?: string[];
  status: 'new' | 'verified' | 'in_progress' | 'completed' | 'rejected' | 'exempted';
  rejection_reason?: string;
  legal_basis?: string;
  exempted_systems?: string[];
  requested_at: string;
  completed_at?: string;
  created_at: string;
}

interface PortabilityExport {
  id: string;
  subject_id: string;
  subject_name: string;
  subject_email: string;
  format: 'json' | 'csv';
  data_categories: string[];
  data_payload?: string;
  file_size_bytes: number;
  expires_at: string;
  downloaded_at?: string;
  created_at: string;
}

interface ConsentAuditEntry {
  id: string;
  subject_id: string;
  action: string;
  purpose: string;
  old_status: string;
  new_status: string;
  changed_by: string;
  source_ip?: string;
  timestamp: string;
  consent_id: string;
}

interface DPIAReport {
  id: string;
  system_name: string;
  system_description: string;
  data_controller: string;
  dpo?: string;
  processing_purposes: string[];
  data_categories: string[];
  data_subjects: string[];
  risk_level: 'low' | 'medium' | 'high' | 'critical';
  risk_assessment: string;
  mitigation_measures: string[];
  residual_risk_level: string;
  dpia_required: boolean;
  dpo_reviewed: boolean;
  review_date: string;
  created_at: string;
}

interface DataTransferAgreement {
  id: string;
  transfer_from: string;
  transfer_to: string;
  mechanism: string;
  scc_status: string;
  controller_name: string;
  processor_name?: string;
  data_categories: string[];
  tia_completed: boolean;
  tia_date?: string;
  supplementary_measures?: string[];
  effective_date: string;
  expiry_date?: string;
  signed_by: string;
  created_at: string;
}

// ═══════════════════════════════════════════════════════════════════════
// Sub-components
// ═══════════════════════════════════════════════════════════════════════

function ErasureBadge({ status }: { status: string }) {
  const config: Record<string, { bg: string; text: string; label: string }> = {
    new: { bg: 'bg-blue-100 dark:bg-blue-900/30', text: 'text-blue-700 dark:text-blue-400', label: 'Новый' },
    verified: { bg: 'bg-indigo-100 dark:bg-indigo-900/30', text: 'text-indigo-700 dark:text-indigo-400', label: 'Подтверждён' },
    in_progress: { bg: 'bg-amber-100 dark:bg-amber-900/30', text: 'text-amber-700 dark:text-amber-400', label: 'В процессе' },
    completed: { bg: 'bg-emerald-100 dark:bg-emerald-900/30', text: 'text-emerald-700 dark:text-emerald-400', label: 'Завершено' },
    rejected: { bg: 'bg-red-100 dark:bg-red-900/30', text: 'text-red-700 dark:text-red-400', label: 'Отклонено' },
    exempted: { bg: 'bg-purple-100 dark:bg-purple-900/30', text: 'text-purple-700 dark:text-purple-400', label: 'Исключение' },
  };
  const c = config[status] || config.new;
  return (
    <span className={`inline-flex items-center px-2 py-0.5 rounded text-xs font-medium ${c.bg} ${c.text}`}>{c.label}</span>
  );
}

function RiskBadge({ level }: { level: string }) {
  const config: Record<string, string> = {
    low: 'bg-emerald-100 text-emerald-700 dark:bg-emerald-900/30 dark:text-emerald-400',
    medium: 'bg-amber-100 text-amber-700 dark:bg-amber-900/30 dark:text-amber-400',
    high: 'bg-orange-100 text-orange-700 dark:bg-orange-900/30 dark:text-orange-400',
    critical: 'bg-red-100 text-red-700 dark:bg-red-900/30 dark:text-red-400',
  };
  return (
    <span className={`inline-flex items-center px-2 py-0.5 rounded text-xs font-medium ${config[level] || config.low}`}>
      {level.toUpperCase()}
    </span>
  );
}

// ═══════════════════════════════════════════════════════════════════════
// Erasure Tab (Art. 17)
// ═══════════════════════════════════════════════════════════════════════

function ErasureTab() {
  const [requests, setRequests] = useState<ErasureRequest[]>([]);
  const [searchQuery, setSearchQuery] = useState('');
  const [showRequestModal, setShowRequestModal] = useState(false);
  const [loading, setLoading] = useState(false);

  const fetchData = useCallback(async (subjectId: string) => {
    if (!subjectId) { setRequests([]); return; }
    setLoading(true);
    try {
      const data = await request<ErasureRequest[]>(`/compliance/gdpr/erasure?subject_id=${encodeURIComponent(subjectId)}`);
      setRequests(data || []);
    } catch { /* ignore */ } finally { setLoading(false); }
  }, []);

  const handleSubmit = async (e: React.FormEvent<HTMLFormElement>) => {
    e.preventDefault();
    const fd = new FormData(e.currentTarget);
    try {
      await request('/compliance/gdpr/erasure', {
        method: 'POST',
        body: JSON.stringify({
          subject_id: fd.get('subject_id'),
          subject_name: fd.get('subject_name'),
          subject_email: fd.get('subject_email'),
          scope: fd.get('scope'),
          specific_systems: (fd.get('specific_systems') as string)?.split(',').map(s => s.trim()).filter(Boolean),
        }),
      });
      setShowRequestModal(false);
      fetchData(fd.get('subject_id') as string);
    } catch { alert('Ошибка при создании запроса'); }
  };

  const handleComplete = async (id: string, subjectId: string) => {
    if (!confirm('Подтвердить завершение удаления данных?')) return;
    try {
      await request('/compliance/gdpr/erasure/complete', {
        method: 'POST',
        body: JSON.stringify({ erasure_id: id }),
      });
      fetchData(subjectId);
    } catch { alert('Ошибка при завершении удаления'); }
  };

  return (
    <div className="space-y-4">
      <div className="flex items-center justify-between">
        <Input placeholder="Subject ID..." value={searchQuery}
          onChange={(e) => { setSearchQuery(e.target.value); if (e.target.value) fetchData(e.target.value); }}
          className="w-64" />
        <Button onClick={() => setShowRequestModal(true)}>
          <Plus className="w-4 h-4 mr-1" /> Запросить удаление
        </Button>
      </div>

      {loading ? (
        <div className="h-48 animate-pulse bg-slate-100 dark:bg-slate-800 rounded-lg" />
      ) : requests.length === 0 ? (
        <EmptyState icon={<Trash2 className="w-12 h-12" />} title="Нет запросов на удаление"
          description="Введите Subject ID для поиска или создайте новый запрос"
          action={{ label: 'Создать запрос', onClick: () => setShowRequestModal(true) }} />
      ) : (
        <Card>
          <div className="overflow-x-auto">
            <table className="w-full text-sm">
              <thead>
                <tr className="border-b border-slate-200 dark:border-slate-700">
                  <th className="text-left p-3 font-medium">Субъект</th>
                  <th className="text-left p-3 font-medium">Область</th>
                  <th className="text-left p-3 font-medium">Статус</th>
                  <th className="text-left p-3 font-medium">Запрошен</th>
                  <th className="text-left p-3 font-medium">Завершён</th>
                  <th className="text-right p-3 font-medium">Действия</th>
                </tr>
              </thead>
              <tbody>
                {requests.map((r) => (
                  <tr key={r.id} className="border-b border-slate-100 dark:border-slate-800 hover:bg-slate-50 dark:hover:bg-slate-800/50">
                    <td className="p-3">
                      <div className="font-medium">{r.subject_name || r.subject_id}</div>
                      <div className="text-xs text-slate-500">{r.subject_email}</div>
                    </td>
                    <td className="p-3">{r.scope}</td>
                    <td className="p-3"><ErasureBadge status={r.status} /></td>
                    <td className="p-3 text-slate-600 dark:text-slate-400">{new Date(r.requested_at).toLocaleDateString()}</td>
                    <td className="p-3 text-slate-600 dark:text-slate-400">
                      {r.completed_at ? new Date(r.completed_at).toLocaleDateString() : '—'}
                    </td>
                    <td className="p-3 text-right">
                      {r.status === 'verified' || r.status === 'in_progress' ? (
                        <button onClick={() => handleComplete(r.id, r.subject_id)}
                          className="text-emerald-600 hover:text-emerald-800 text-xs font-medium">
                          Завершить
                        </button>
                      ) : r.rejection_reason ? (
                        <span className="text-xs text-red-500" title={r.rejection_reason}>
                          <AlertTriangle className="w-3 h-3 inline mr-1" />{r.rejection_reason?.slice(0, 30)}...
                        </span>
                      ) : null}
                    </td>
                  </tr>
                ))}
              </tbody>
            </table>
          </div>
        </Card>
      )}

      <Modal isOpen={showRequestModal} onClose={() => setShowRequestModal(false)} title="Запрос на удаление данных (Art. 17)">
        <form onSubmit={handleSubmit} className="space-y-4">
          <div>
            <label className="block text-sm font-medium mb-1">Subject ID *</label>
            <Input name="subject_id" required placeholder="ID субъекта данных" />
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
            <label className="block text-sm font-medium mb-1">Область удаления *</label>
            <select name="scope" required className="w-full rounded-lg border border-slate-300 dark:border-slate-600 bg-white dark:bg-slate-800 p-2 text-sm">
              <option value="all">Все данные</option>
              <option value="video">Только видео</option>
              <option value="analytics">Только аналитика</option>
              <option value="credentials">Только учётные данные</option>
              <option value="specific">Конкретные системы</option>
            </select>
          </div>
          <div>
            <label className="block text-sm font-medium mb-1">Конкретные системы (через запятую)</label>
            <Input name="specific_systems" placeholder="camera_01, nvr_02" />
          </div>
          <div className="flex justify-end gap-2 pt-2">
            <Button type="button" variant="secondary" onClick={() => setShowRequestModal(false)}>Отмена</Button>
            <Button type="submit">Отправить запрос</Button>
          </div>
        </form>
      </Modal>
    </div>
  );
}

// ═══════════════════════════════════════════════════════════════════════
// Portability Tab (Art. 20)
// ═══════════════════════════════════════════════════════════════════════

function PortabilityTab() {
  const [exports, setExports] = useState<PortabilityExport[]>([]);
  const [searchQuery, setSearchQuery] = useState('');
  const [showCreateModal, setShowCreateModal] = useState(false);
  const [loading, setLoading] = useState(false);

  const fetchData = useCallback(async (subjectId: string) => {
    if (!subjectId) { setExports([]); return; }
    setLoading(true);
    try {
      const data = await request<PortabilityExport[]>(`/compliance/gdpr/portability?subject_id=${encodeURIComponent(subjectId)}`);
      setExports(data || []);
    } catch { /* ignore */ } finally { setLoading(false); }
  }, []);

  const handleCreate = async (e: React.FormEvent<HTMLFormElement>) => {
    e.preventDefault();
    const fd = new FormData(e.currentTarget);
    try {
      await request('/compliance/gdpr/portability', {
        method: 'POST',
        body: JSON.stringify({
          subject_id: fd.get('subject_id'),
          subject_name: fd.get('subject_name'),
          subject_email: fd.get('subject_email'),
          format: fd.get('format'),
          payload: fd.get('payload') || '{}',
        }),
      });
      setShowCreateModal(false);
      fetchData(fd.get('subject_id') as string);
    } catch { alert('Ошибка при создании экспорта'); }
  };

  return (
    <div className="space-y-4">
      <div className="flex items-center justify-between">
        <Input placeholder="Subject ID..." value={searchQuery}
          onChange={(e) => { setSearchQuery(e.target.value); if (e.target.value) fetchData(e.target.value); }}
          className="w-64" />
        <Button onClick={() => setShowCreateModal(true)}>
          <Plus className="w-4 h-4 mr-1" /> Создать экспорт
        </Button>
      </div>

      {loading ? (
        <div className="h-48 animate-pulse bg-slate-100 dark:bg-slate-800 rounded-lg" />
      ) : exports.length === 0 ? (
        <EmptyState icon={<Download className="w-12 h-12" />} title="Нет экспортов"
          description="Создайте экспорт данных для портабельности"
          action={{ label: 'Создать экспорт', onClick: () => setShowCreateModal(true) }} />
      ) : (
        <Card>
          <div className="overflow-x-auto">
            <table className="w-full text-sm">
              <thead>
                <tr className="border-b border-slate-200 dark:border-slate-700">
                  <th className="text-left p-3 font-medium">Субъект</th>
                  <th className="text-left p-3 font-medium">Формат</th>
                  <th className="text-left p-3 font-medium">Категории</th>
                  <th className="text-left p-3 font-medium">Размер</th>
                  <th className="text-left p-3 font-medium">Истекает</th>
                  <th className="text-right p-3 font-medium">Действия</th>
                </tr>
              </thead>
              <tbody>
                {exports.map((exp) => (
                  <tr key={exp.id} className="border-b border-slate-100 dark:border-slate-800 hover:bg-slate-50 dark:hover:bg-slate-800/50">
                    <td className="p-3">
                      <div className="font-medium">{exp.subject_name || exp.subject_id}</div>
                      <div className="text-xs text-slate-500">{exp.subject_email}</div>
                    </td>
                    <td className="p-3"><span className="font-mono text-xs bg-slate-100 dark:bg-slate-700 px-2 py-0.5 rounded">{exp.format.toUpperCase()}</span></td>
                    <td className="p-3">
                      <div className="flex flex-wrap gap-1">
                        {exp.data_categories.map((cat, i) => (
                          <span key={i} className="text-xs bg-slate-100 dark:bg-slate-700 px-1.5 py-0.5 rounded">{cat}</span>
                        ))}
                      </div>
                    </td>
                    <td className="p-3 text-slate-600 dark:text-slate-400">{formatBytes(exp.file_size_bytes)}</td>
                    <td className="p-3 text-slate-600 dark:text-slate-400">{new Date(exp.expires_at).toLocaleDateString()}</td>
                    <td className="p-3 text-right">
                      <button onClick={() => { if (exp.data_payload) { const blob = new Blob([exp.data_payload], { type: 'application/json' }); const url = URL.createObjectURL(blob); window.open(url); } }}
                        className="text-blue-600 hover:text-blue-800 text-xs font-medium">
                        <Download className="w-3 h-3 inline mr-1" />Скачать
                      </button>
                    </td>
                  </tr>
                ))}
              </tbody>
            </table>
          </div>
        </Card>
      )}

      <Modal isOpen={showCreateModal} onClose={() => setShowCreateModal(false)} title="Создание экспорта данных (Art. 20)">
        <form onSubmit={handleCreate} className="space-y-4">
          <div>
            <label className="block text-sm font-medium mb-1">Subject ID *</label>
            <Input name="subject_id" required placeholder="ID субъекта данных" />
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
            <label className="block text-sm font-medium mb-1">Формат *</label>
            <select name="format" required className="w-full rounded-lg border border-slate-300 dark:border-slate-600 bg-white dark:bg-slate-800 p-2 text-sm">
              <option value="json">JSON</option>
              <option value="csv">CSV</option>
            </select>
          </div>
          <div>
            <label className="block text-sm font-medium mb-1">Payload (JSON)</label>
            <textarea name="payload" rows={5} className="w-full rounded-lg border border-slate-300 dark:border-slate-600 bg-white dark:bg-slate-800 p-2 text-sm font-mono"
              placeholder='{"data": {...}}' />
          </div>
          <div className="flex justify-end gap-2 pt-2">
            <Button type="button" variant="secondary" onClick={() => setShowCreateModal(false)}>Отмена</Button>
            <Button type="submit">Создать</Button>
          </div>
        </form>
      </Modal>
    </div>
  );
}

// ═══════════════════════════════════════════════════════════════════════
// Consent Audit Tab (Art. 7)
// ═══════════════════════════════════════════════════════════════════════

function ConsentAuditTab() {
  const [entries, setEntries] = useState<ConsentAuditEntry[]>([]);
  const [searchQuery, setSearchQuery] = useState('');
  const [loading, setLoading] = useState(false);

  const fetchData = useCallback(async (subjectId: string) => {
    if (!subjectId) { setEntries([]); return; }
    setLoading(true);
    try {
      const data = await request<ConsentAuditEntry[]>(`/compliance/gdpr/consent-audit?subject_id=${encodeURIComponent(subjectId)}`);
      setEntries(data || []);
    } catch { /* ignore */ } finally { setLoading(false); }
  }, []);

  return (
    <div className="space-y-4">
      <Input placeholder="Subject ID для поиска аудита..."
        value={searchQuery}
        onChange={(e) => { setSearchQuery(e.target.value); if (e.target.value) fetchData(e.target.value); }}
        className="w-64" />

      {loading ? (
        <div className="h-48 animate-pulse bg-slate-100 dark:bg-slate-800 rounded-lg" />
      ) : entries.length === 0 ? (
        <EmptyState icon={<Shield className="w-12 h-12" />} title="Нет записей аудита"
          description="Введите Subject ID для просмотра истории изменений согласий" />
      ) : (
        <Card>
          <div className="overflow-x-auto">
            <table className="w-full text-sm">
              <thead>
                <tr className="border-b border-slate-200 dark:border-slate-700">
                  <th className="text-left p-3 font-medium">Дата</th>
                  <th className="text-left p-3 font-medium">Действие</th>
                  <th className="text-left p-3 font-medium">Цель</th>
                  <th className="text-left p-3 font-medium">Было</th>
                  <th className="text-left p-3 font-medium">Стало</th>
                  <th className="text-left p-3 font-medium">Кто изменил</th>
                  <th className="text-left p-3 font-medium">IP</th>
                </tr>
              </thead>
              <tbody>
                {entries.map((entry) => (
                  <tr key={entry.id} className="border-b border-slate-100 dark:border-slate-800 hover:bg-slate-50 dark:hover:bg-slate-800/50">
                    <td className="p-3 text-slate-600 dark:text-slate-400">{new Date(entry.timestamp).toLocaleString()}</td>
                    <td className="p-3 font-medium">{entry.action}</td>
                    <td className="p-3">{entry.purpose}</td>
                    <td className="p-3"><span className="text-xs bg-slate-100 dark:bg-slate-700 px-1.5 py-0.5 rounded">{entry.old_status}</span></td>
                    <td className="p-3"><span className="text-xs bg-blue-100 dark:bg-blue-900/30 text-blue-700 dark:text-blue-400 px-1.5 py-0.5 rounded">{entry.new_status}</span></td>
                    <td className="p-3 text-slate-600 dark:text-slate-400">{entry.changed_by}</td>
                    <td className="p-3 text-slate-400 font-mono text-xs">{entry.source_ip || '—'}</td>
                  </tr>
                ))}
              </tbody>
            </table>
          </div>
        </Card>
      )}
    </div>
  );
}

// ═══════════════════════════════════════════════════════════════════════
// DPIA Tab (Art. 35)
// ═══════════════════════════════════════════════════════════════════════

function DPIATab() {
  const [reports, setReports] = useState<DPIAReport[]>([]);
  const [showCreateModal, setShowCreateModal] = useState(false);
  const [selectedReport, setSelectedReport] = useState<DPIAReport | null>(null);
  const [loading, setLoading] = useState(true);

  const fetchData = useCallback(async () => {
    setLoading(true);
    try {
      const data = await request<DPIAReport[]>('/compliance/gdpr/dpia');
      setReports(data || []);
    } catch { /* ignore */ } finally { setLoading(false); }
  }, []);

  useEffect(() => { fetchData(); }, [fetchData]);

  const handleCreate = async (e: React.FormEvent<HTMLFormElement>) => {
    e.preventDefault();
    const fd = new FormData(e.currentTarget);
    try {
      await request('/compliance/gdpr/dpia', {
        method: 'POST',
        body: JSON.stringify({
          system_name: fd.get('system_name'),
          system_description: fd.get('system_description'),
          data_controller: fd.get('data_controller'),
          dpo: fd.get('dpo'),
          processing_purposes: [(fd.get('purpose') as string)],
          data_categories: [(fd.get('data_category') as string)],
          data_subjects: ['employees', 'visitors'],
          legal_basis: fd.get('legal_basis'),
          data_retention_period: fd.get('retention_period') || '90 days',
        }),
      });
      setShowCreateModal(false);
      fetchData();
    } catch { alert('Ошибка при создании DPIA'); }
  };

  if (loading) {
    return <div className="h-48 animate-pulse bg-slate-100 dark:bg-slate-800 rounded-lg" />;
  }

  return (
    <div className="space-y-4">
      <div className="flex items-center justify-between">
        <h3 className="text-lg font-semibold">DPIA Reports ({reports.length})</h3>
        <Button onClick={() => setShowCreateModal(true)}>
          <Plus className="w-4 h-4 mr-1" /> Новый DPIA
        </Button>
      </div>

      {reports.length === 0 ? (
        <EmptyState icon={<FileText className="w-12 h-12" />} title="Нет DPIA отчётов"
          description="Создайте оценку воздействия на защиту данных"
          action={{ label: 'Создать DPIA', onClick: () => setShowCreateModal(true) }} />
      ) : (
        <div className="grid grid-cols-1 md:grid-cols-2 gap-4">
          {reports.map((report) => (
            <div key={report.id} className="cursor-pointer hover:shadow-md transition-shadow" onClick={() => setSelectedReport(report)}>
              <Card>
              <div className="p-4 space-y-3">
                <div className="flex items-start justify-between">
                  <div>
                    <h4 className="font-semibold">{report.system_name}</h4>
                    <p className="text-xs text-slate-500">{report.data_controller}</p>
                  </div>
                  <RiskBadge level={report.risk_level} />
                </div>
                <p className="text-xs text-slate-600 dark:text-slate-400 line-clamp-2">{report.system_description}</p>
                <div className="flex items-center justify-between text-xs text-slate-500">
                  <span>DPIA: {report.dpia_required ? '✅' : '❌'}</span>
                  <span>DPO: {report.dpo_reviewed ? '✅' : '⏳'}</span>
                  <span>Review: {new Date(report.review_date).toLocaleDateString()}</span>
                </div>
              </Card>
            </div>
          ))}
        </div>
      )}

      <Modal isOpen={showCreateModal} onClose={() => setShowCreateModal(false)} title="Новый DPIA отчёт (Art. 35)">
        <form onSubmit={handleCreate} className="space-y-4">
          <div>
            <label className="block text-sm font-medium mb-1">Система *</label>
            <Input name="system_name" required placeholder="CCTV Health Monitor" />
          </div>
          <div>
            <label className="block text-sm font-medium mb-1">Описание</label>
            <textarea name="system_description" rows={3} className="w-full rounded-lg border border-slate-300 dark:border-slate-600 bg-white dark:bg-slate-800 p-2 text-sm"
              placeholder="Система видеонаблюдения и контроля доступа" />
          </div>
          <div>
            <label className="block text-sm font-medium mb-1">Контролёр данных *</label>
            <Input name="data_controller" required placeholder="ООО 'Организация'" />
          </div>
          <div>
            <label className="block text-sm font-medium mb-1">DPO</label>
            <Input name="dpo" placeholder="ФИО ответственного" />
          </div>
          <div className="grid grid-cols-2 gap-4">
            <div>
              <label className="block text-sm font-medium mb-1">Цель обработки</label>
              <Input name="purpose" placeholder="video_monitoring" />
            </div>
            <div>
              <label className="block text-sm font-medium mb-1">Категория данных</label>
              <select name="data_category" className="w-full rounded-lg border border-slate-300 dark:border-slate-600 bg-white dark:bg-slate-800 p-2 text-sm">
                <option value="biometric">Биометрические</option>
                <option value="location">Геолокация</option>
                <option value="identity">Персональные</option>
                <option value="video_archive">Видеоархив</option>
              </select>
            </div>
          </div>
          <div className="grid grid-cols-2 gap-4">
            <div>
              <label className="block text-sm font-medium mb-1">Правовое основание</label>
              <Input name="legal_basis" placeholder="GDPR Art. 6(1)(f)" />
            </div>
            <div>
              <label className="block text-sm font-medium mb-1">Срок хранения</label>
              <Input name="retention_period" placeholder="90 days" />
            </div>
          </div>
          <div className="flex justify-end gap-2 pt-2">
            <Button type="button" variant="secondary" onClick={() => setShowCreateModal(false)}>Отмена</Button>
            <Button type="submit">Создать DPIA</Button>
          </div>
        </form>
      </Modal>

      <Modal isOpen={!!selectedReport} onClose={() => setSelectedReport(null)} title={`DPIA: ${selectedReport?.system_name || ''}`} size="lg">
        {selectedReport && (
          <div className="space-y-4">
            <div className="grid grid-cols-2 gap-4">
              <div><label className="text-xs text-slate-500">Система</label><p className="font-medium">{selectedReport.system_name}</p></div>
              <div><label className="text-xs text-slate-500">Контролёр</label><p className="font-medium">{selectedReport.data_controller}</p></div>
            </div>
            <div><label className="text-xs text-slate-500">Описание</label><p className="text-sm">{selectedReport.system_description}</p></div>
            <div>
              <label className="text-xs text-slate-500">Уровень риска</label>
              <div className="mt-1"><RiskBadge level={selectedReport.risk_level} /></div>
            </div>
            <div><label className="text-xs text-slate-500">Оценка риска</label><p className="text-sm">{selectedReport.risk_assessment}</p></div>
            <div>
              <label className="text-xs text-slate-500">Меры минимизации</label>
              <ul className="list-disc list-inside text-sm mt-1">
                {selectedReport.mitigation_measures.map((m, i) => <li key={i}>{m}</li>)}
              </ul>
            </div>
            <div className="grid grid-cols-3 gap-4">
              <div><label className="text-xs text-slate-500">Остаточный риск</label><p className="font-medium">{selectedReport.residual_risk_level}</p></div>
              <div><label className="text-xs text-slate-500">DPIA required</label><p>{selectedReport.dpia_required ? '✅' : '❌'}</p></div>
              <div><label className="text-xs text-slate-500">DPO reviewed</label><p>{selectedReport.dpo_reviewed ? '✅' : '⏳'}</p></div>
            </div>
            <p className="text-xs text-slate-400">Создан: {new Date(selectedReport.created_at).toLocaleString()} | Review: {new Date(selectedReport.review_date).toLocaleDateString()}</p>
          </div>
        )}
      </Modal>
    </div>
  );
}

// ═══════════════════════════════════════════════════════════════════════
// Data Transfers Tab (Art. 44-49)
// ═══════════════════════════════════════════════════════════════════════

function TransfersTab() {
  const [agreements, setAgreements] = useState<DataTransferAgreement[]>([]);
  const [showCreateModal, setShowCreateModal] = useState(false);
  const [loading, setLoading] = useState(true);

  const fetchData = useCallback(async () => {
    setLoading(true);
    try {
      const data = await request<DataTransferAgreement[]>('/compliance/gdpr/transfers');
      setAgreements(data || []);
    } catch { /* ignore */ } finally { setLoading(false); }
  }, []);

  useEffect(() => { fetchData(); }, [fetchData]);

  const handleCreate = async (e: React.FormEvent<HTMLFormElement>) => {
    e.preventDefault();
    const fd = new FormData(e.currentTarget);
    try {
      await request('/compliance/gdpr/transfers', {
        method: 'POST',
        body: JSON.stringify({
          transfer_from: fd.get('transfer_from'),
          transfer_to: fd.get('transfer_to'),
          mechanism: fd.get('mechanism'),
          controller_name: fd.get('controller_name'),
          processor_name: fd.get('processor_name'),
          signed_by: fd.get('signed_by'),
          effective_date: new Date().toISOString(),
        }),
      });
      setShowCreateModal(false);
      fetchData();
    } catch { alert('Ошибка при создании соглашения'); }
  };

  const handleCompleteTIA = async (id: string) => {
    if (!confirm('Подтвердить завершение TIA?')) return;
    try {
      await request(`/compliance/gdpr/transfers/${id}/tia`, { method: 'POST' });
      fetchData();
    } catch { alert('Ошибка при завершении TIA'); }
  };

  if (loading) {
    return <div className="h-48 animate-pulse bg-slate-100 dark:bg-slate-800 rounded-lg" />;
  }

  return (
    <div className="space-y-4">
      <div className="flex items-center justify-between">
        <h3 className="text-lg font-semibold">Соглашения о передаче ({agreements.length})</h3>
        <Button onClick={() => setShowCreateModal(true)}>
          <Plus className="w-4 h-4 mr-1" /> Новое соглашение
        </Button>
      </div>

      {agreements.length === 0 ? (
        <EmptyState icon={<Globe className="w-12 h-12" />} title="Нет соглашений о передаче"
          description="Создайте SCC для трансграничной передачи данных"
          action={{ label: 'Создать SCC', onClick: () => setShowCreateModal(true) }} />
      ) : (
        <Card>
          <div className="overflow-x-auto">
            <table className="w-full text-sm">
              <thead>
                <tr className="border-b border-slate-200 dark:border-slate-700">
                  <th className="text-left p-3 font-medium">Из</th>
                  <th className="text-left p-3 font-medium">В</th>
                  <th className="text-left p-3 font-medium">Механизм</th>
                  <th className="text-left p-3 font-medium">Статус</th>
                  <th className="text-left p-3 font-medium">Контролёр</th>
                  <th className="text-center p-3 font-medium">TIA</th>
                  <th className="text-right p-3 font-medium">Действия</th>
                </tr>
              </thead>
              <tbody>
                {agreements.map((ag) => (
                  <tr key={ag.id} className="border-b border-slate-100 dark:border-slate-800 hover:bg-slate-50 dark:hover:bg-slate-800/50">
                    <td className="p-3 font-medium">{ag.transfer_from}</td>
                    <td className="p-3 font-medium">{ag.transfer_to}</td>
                    <td className="p-3">
                      <span className="font-mono text-xs bg-slate-100 dark:bg-slate-700 px-1.5 py-0.5 rounded">{ag.mechanism}</span>
                    </td>
                    <td className="p-3">
                      <span className={`text-xs px-1.5 py-0.5 rounded ${
                        ag.scc_status === 'active' ? 'bg-emerald-100 text-emerald-700 dark:bg-emerald-900/30 dark:text-emerald-400' :
                        ag.scc_status === 'negotiating' ? 'bg-amber-100 text-amber-700 dark:bg-amber-900/30 dark:text-amber-400' :
                        'bg-red-100 text-red-700 dark:bg-red-900/30 dark:text-red-400'
                      }`}>{ag.scc_status}</span>
                    </td>
                    <td className="p-3 text-slate-600 dark:text-slate-400">{ag.controller_name}</td>
                    <td className="p-3 text-center">
                      {ag.tia_completed
                        ? <span className="inline-flex items-center gap-1" title={ag.tia_date ? new Date(ag.tia_date).toLocaleDateString() : ''}>
                            <CheckCircle className="w-4 h-4 text-emerald-500 inline" />
                          </span>
                        : <button onClick={() => handleCompleteTIA(ag.id)} className="text-xs text-amber-600 hover:text-amber-800">Завершить TIA</button>}
                    </td>
                    <td className="p-3 text-right">
                      <span className="text-xs text-slate-500" title={`Подписал: ${ag.signed_by}`}>
                        <Eye className="w-3 h-3 inline mr-1" />{ag.signed_by?.slice(0, 15)}...
                      </span>
                    </td>
                  </tr>
                ))}
              </tbody>
            </table>
          </div>
        </Card>
      )}

      <Modal isOpen={showCreateModal} onClose={() => setShowCreateModal(false)} title="Новое SCC соглашение (Art. 46)">
        <form onSubmit={handleCreate} className="space-y-4">
          <div className="grid grid-cols-2 gap-4">
            <div>
              <label className="block text-sm font-medium mb-1">Из (страна) *</label>
              <Input name="transfer_from" required placeholder="EU" />
            </div>
            <div>
              <label className="block text-sm font-medium mb-1">В (страна) *</label>
              <Input name="transfer_to" required placeholder="US" />
            </div>
          </div>
          <div>
            <label className="block text-sm font-medium mb-1">Механизм *</label>
            <select name="mechanism" required className="w-full rounded-lg border border-slate-300 dark:border-slate-600 bg-white dark:bg-slate-800 p-2 text-sm">
              <option value="scc">SCC (Standard Contractual Clauses)</option>
              <option value="bcr">BCR (Binding Corporate Rules)</option>
              <option value="adequacy">Adequacy decision</option>
              <option value="derogation">Derogation (Art. 49)</option>
            </select>
          </div>
          <div>
            <label className="block text-sm font-medium mb-1">Контролёр данных *</label>
            <Input name="controller_name" required placeholder="ООО 'Организация'" />
          </div>
          <div>
            <label className="block text-sm font-medium mb-1">Обработчик</label>
            <Input name="processor_name" placeholder="ООО 'Обработчик'" />
          </div>
          <div>
            <label className="block text-sm font-medium mb-1">Подписант *</label>
            <Input name="signed_by" required placeholder="ФИО подписанта" />
          </div>
          <div className="flex justify-end gap-2 pt-2">
            <Button type="button" variant="secondary" onClick={() => setShowCreateModal(false)}>Отмена</Button>
            <Button type="submit">Создать SCC</Button>
          </div>
        </form>
      </Modal>
    </div>
  );
}

// ═══════════════════════════════════════════════════════════════════════
// Helpers
// ═══════════════════════════════════════════════════════════════════════

function formatBytes(bytes: number): string {
  if (bytes === 0) return '0 B';
  const k = 1024;
  const sizes = ['B', 'KB', 'MB', 'GB'];
  const i = Math.floor(Math.log(bytes) / Math.log(k));
  return parseFloat((bytes / Math.pow(k, i)).toFixed(1)) + ' ' + sizes[i];
}

// ═══════════════════════════════════════════════════════════════════════
// Main GDPR Component
// ═══════════════════════════════════════════════════════════════════════

export default function GDPR() {
  const { t } = useTranslation();
  const [activeTab, setActiveTab] = useState('erasure');

  const tabs = [
    { id: 'erasure', label: 'Right to be Forgotten', icon: <Trash2 className="w-4 h-4" /> },
    { id: 'portability', label: 'Data Portability', icon: <Download className="w-4 h-4" /> },
    { id: 'consent-audit', label: 'Consent Audit', icon: <Shield className="w-4 h-4" /> },
    { id: 'dpia', label: 'DPIA', icon: <FileText className="w-4 h-4" /> },
    { id: 'transfers', label: 'Data Transfers', icon: <Globe className="w-4 h-4" /> },
  ];

  return (
    <div className="p-6 space-y-6">
      <div>
        <h1 className="text-2xl font-bold text-slate-900 dark:text-white">GDPR Compliance</h1>
        <p className="text-sm text-slate-500 dark:text-slate-400 mt-1">
          Right to be Forgotten, Data Portability, DPIA, and Schrems II compliant data transfers
        </p>
      </div>

      <Tabs tabs={tabs} activeTab={activeTab} onChange={setActiveTab}>
        {activeTab === 'erasure' && <ErasureTab />}
        {activeTab === 'portability' && <PortabilityTab />}
        {activeTab === 'consent-audit' && <ConsentAuditTab />}
        {activeTab === 'dpia' && <DPIATab />}
        {activeTab === 'transfers' && <TransfersTab />}
      </Tabs>
    </div>
  );
}
