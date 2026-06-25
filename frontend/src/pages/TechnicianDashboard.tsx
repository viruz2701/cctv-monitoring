import React, { useEffect, useState } from 'react';
import { useTranslation } from 'react-i18next';
import { request } from '../services/api';
import { Card, DataGrid, Badge, Button, StatsCard } from '../components/ui';
import { useNavigate } from 'react-router-dom';
import {
  CheckCircle, Clock, AlertTriangle, Users, TrendingUp,
  Wrench, MapPin, Play, ArrowRight, Loader2,
} from 'lucide-react';

// ── Types ────────────────────────────────────────────────────────────

interface TechnicianWorkload {
  user_id: string;
  user_name: string;
  current_workload: number;
  max_workload: number;
  skills: string[];
  base_location: string;
}

interface AssignedWorkOrder {
  id: string;
  title: string;
  priority: string;
  status: string;
  site_name?: string;
  device_name?: string;
  device_id?: string;
  sla_deadline?: string;
  sla_status?: string;
  created_at: string;
}

interface TechnicianStats {
  total_assigned: number;
  in_progress: number;
  completed_today: number;
  sla_at_risk: number;
  sla_breached: number;
}

const PRIORITY_CONFIG: Record<string, { color: string; bg: string; label: string }> = {
  CRITICAL: { color: 'text-red-700', bg: 'bg-red-100', label: 'Критичный' },
  HIGH:     { color: 'text-orange-700', bg: 'bg-orange-100', label: 'Высокий' },
  MEDIUM:   { color: 'text-yellow-700', bg: 'bg-yellow-100', label: 'Средний' },
  LOW:      { color: 'text-green-700', bg: 'bg-green-100', label: 'Низкий' },
};

const STATUS_CONFIG: Record<string, { color: string; bg: string; label: string }> = {
  REQUESTED:  { color: 'text-slate-600', bg: 'bg-slate-100', label: 'Запрошен' },
  APPROVED:   { color: 'text-blue-600',  bg: 'bg-blue-100',  label: 'Утверждён' },
  ASSIGNED:   { color: 'text-indigo-600',bg: 'bg-indigo-100',label: 'Назначен' },
  IN_PROGRESS:{ color: 'text-amber-600', bg: 'bg-amber-100', label: 'В работе' },
  ON_HOLD:    { color: 'text-purple-600', bg: 'bg-purple-100',label: 'На паузе' },
  COMPLETED:  { color: 'text-emerald-600',bg: 'bg-emerald-100',label: 'Завершён' },
  CANCELLED:  { color: 'text-red-600',   bg: 'bg-red-100',   label: 'Отменён' },
};

// ── Component ────────────────────────────────────────────────────────

export const TechnicianDashboard: React.FC = () => {
  const { t } = useTranslation();
  const navigate = useNavigate();

  const [workloads, setWorkloads] = useState<TechnicianWorkload[]>([]);
  const [assignedWO, setAssignedWO] = useState<AssignedWorkOrder[]>([]);
  const [stats, setStats] = useState<TechnicianStats | null>(null);
  const [loading, setLoading] = useState({ workloads: false, wos: false });
  const [actionLoading, setActionLoading] = useState<string | null>(null);

  // ── Data Loading ─────────────────────────────────────────────────

  useEffect(() => {
    fetchWorkloads();
    fetchAssignedWorkOrders();
  }, []);

  const fetchWorkloads = async () => {
    setLoading(prev => ({ ...prev, workloads: true }));
    try {
      const data = await request<TechnicianWorkload[]>('/technicians/workload');
      setWorkloads(data || []);
    } catch { /* ignore */ }
    finally { setLoading(prev => ({ ...prev, workloads: false })); }
  };

  const fetchAssignedWorkOrders = async () => {
    setLoading(prev => ({ ...prev, wos: true }));
    try {
      const data = await request<AssignedWorkOrder[]>('/work-orders?assigned_to=me&limit=50');
      setAssignedWO(data || []);

      // Compute stats
      const total = data?.length || 0;
      const inProgress = data?.filter(wo => wo.status === 'IN_PROGRESS').length || 0;
      const completedToday = data?.filter(wo => {
        if (wo.status !== 'COMPLETED') return false;
        const today = new Date().toISOString().slice(0, 10);
        return wo.created_at?.startsWith(today);
      }).length || 0;
      const atRisk = data?.filter(wo => wo.sla_status === 'at_risk').length || 0;
      const breached = data?.filter(wo => wo.sla_status === 'breached').length || 0;
      setStats({ total_assigned: total, in_progress: inProgress, completed_today: completedToday, sla_at_risk: atRisk, sla_breached: breached });
    } catch { /* ignore */ }
    finally { setLoading(prev => ({ ...prev, wos: false })); }
  };

  // ── Quick Actions ────────────────────────────────────────────────

  const handleStartWork = async (woId: string) => {
    setActionLoading(woId);
    try {
      await request(`/work-orders/${woId}/start`, { method: 'POST' });
      fetchAssignedWorkOrders();
    } catch { /* ignore */ }
    finally { setActionLoading(null); }
  };

  const handleComplete = (woId: string) => {
    navigate(`/work-orders/${woId}`);
  };

  // ── Sort: priority + SLA status ──────────────────────────────────

  const priorityWeight = (p: string) => ({ CRITICAL: 0, HIGH: 1, MEDIUM: 2, LOW: 3 })[p] ?? 4;
  const slaWeight = (s: string) => ({ breached: 0, at_risk: 1, ok: 2, '': 3 })[s] ?? 3;

  const sortedWO = [...assignedWO].sort((a, b) => {
    const pa = priorityWeight(a.priority) - priorityWeight(b.priority);
    if (pa !== 0) return pa;
    return slaWeight(a.sla_status || '') - slaWeight(b.sla_status || '');
  });

  // ── WO Table Columns ─────────────────────────────────────────────

  const woColumns = [
    {
      key: 'priority',
      header: t('priority') || 'Приоритет',
      sortable: true,
      width: 100,
      render: (wo: AssignedWorkOrder) => {
        const cfg = PRIORITY_CONFIG[wo.priority] || PRIORITY_CONFIG.LOW;
        return <span className={`px-2 py-0.5 rounded text-xs font-bold ${cfg.color} ${cfg.bg}`}>{cfg.label}</span>;
      },
    },
    {
      key: 'title',
      header: t('work_order') || 'Наряд',
      sortable: true,
      render: (wo: AssignedWorkOrder) => (
        <div>
          <span className="text-sm font-medium text-slate-900">{wo.title || wo.device_name || wo.id.slice(0, 8)}</span>
          {wo.device_name && <p className="text-xs text-slate-500 mt-0.5">📹 {wo.device_name}</p>}
        </div>
      ),
    },
    {
      key: 'site_name',
      header: t('site') || 'Объект',
      render: (wo: AssignedWorkOrder) => wo.site_name ? <span className="text-sm text-slate-600">📍 {wo.site_name}</span> : '—',
    },
    {
      key: 'status',
      header: t('status') || 'Статус',
      sortable: true,
      render: (wo: AssignedWorkOrder) => {
        const cfg = STATUS_CONFIG[wo.status] || STATUS_CONFIG.REQUESTED;
        return <Badge variant={wo.status === 'COMPLETED' ? 'success' : wo.status === 'IN_PROGRESS' ? 'warning' : 'info'}>{cfg.label}</Badge>;
      },
    },
    {
      key: 'sla_status',
      header: 'SLA',
      sortable: true,
      render: (wo: AssignedWorkOrder) => {
        if (wo.sla_status === 'breached') return <Badge variant="danger">Просрочен</Badge>;
        if (wo.sla_status === 'at_risk') return <Badge variant="warning">На исходе</Badge>;
        if (wo.sla_status === 'ok') return <Badge variant="success">В норме</Badge>;
        return '—';
      },
    },
    {
      key: 'actions',
      header: '',
      width: 140,
      render: (wo: AssignedWorkOrder) => {
        if (wo.status === 'COMPLETED' || wo.status === 'CANCELLED') return null;
        return (
          <div className="flex items-center gap-1">
            {wo.status !== 'IN_PROGRESS' ? (
              <Button
                size="sm"
                variant="primary"
                icon={actionLoading === wo.id ? <Loader2 className="w-3 h-3 animate-spin" /> : <Play className="w-3 h-3" />}
                onClick={() => handleStartWork(wo.id)}
                loading={actionLoading === wo.id}
              >
                {t('start') || 'Старт'}
              </Button>
            ) : (
              <Button
                size="sm"
                variant="outline"
                icon={<CheckCircle className="w-3 h-3" />}
                onClick={() => handleComplete(wo.id)}
              >
                {t('complete') || 'Готово'}
              </Button>
            )}
          </div>
        );
      },
    },
  ];

  // ── Workload Table Columns ───────────────────────────────────────

  const workloadColumns = [
    { key: 'user_name', header: t('technician') || 'Техник', sortable: true },
    {
      key: 'workload',
      header: t('workload') || 'Загрузка',
      sortable: true,
      render: (item: TechnicianWorkload) => {
        const percent = item.max_workload > 0 ? (item.current_workload / item.max_workload) * 100 : 0;
        const color = percent > 80 ? 'bg-red-500' : percent > 50 ? 'bg-amber-500' : 'bg-emerald-500';
        return (
          <div className="flex items-center gap-2">
            <div className="w-32 bg-slate-200 rounded-full h-2">
              <div className={`${color} h-2 rounded-full`} style={{ width: `${Math.min(percent, 100)}%` }} />
            </div>
            <span className="text-xs font-mono text-slate-600">{item.current_workload}/{item.max_workload}</span>
          </div>
        );
      },
    },
    {
      key: 'skills',
      header: t('skills') || 'Навыки',
      render: (item: TechnicianWorkload) => (
        <div className="flex flex-wrap gap-1">
          {item.skills?.slice(0, 3).map((skill, i) => (
            <Badge key={i} variant="info" size="sm">{skill}</Badge>
          ))}
          {(item.skills?.length || 0) > 3 && <span className="px-1.5 py-0.5 text-[10px] font-medium text-slate-500 bg-slate-100 rounded">+{item.skills.length - 3}</span>}
        </div>
      ),
    },
    { key: 'base_location', header: t('location') || 'Локация', sortable: true },
  ];

  // ── Render ───────────────────────────────────────────────────────

  const slaAlerts = assignedWO.filter(wo => wo.sla_status === 'breached' || wo.sla_status === 'at_risk');

  return (
    <div className="space-y-6">
      {/* Header */}
      <div className="flex items-center justify-between">
        <div>
          <h1 className="text-2xl font-bold text-slate-900 flex items-center gap-2">
            <Users className="w-6 h-6" />
            {t('technician_dashboard') || 'Панель техника'}
          </h1>
          <p className="text-sm text-slate-500 mt-1">
            {t('technician_dashboard_desc') || 'Мои наряды, загрузка команды и KPI'}
          </p>
        </div>
        <Button
          icon={<TrendingUp className="w-4 h-4" />}
          onClick={() => navigate('/analytics')}
          variant="outline"
        >
          {t('analytics') || 'Аналитика'}
        </Button>
      </div>

      {/* SLA Alerts */}
      {slaAlerts.length > 0 && (() => {
        const isBreached = (stats?.sla_breached ?? 0) > 0;
        return (
          <div className={`p-4 rounded-lg border ${isBreached ? 'bg-red-50 border-red-200' : 'bg-amber-50 border-amber-200'}`}>
            <div className="flex items-start gap-3">
              <AlertTriangle className={`w-5 h-5 mt-0.5 ${isBreached ? 'text-red-600' : 'text-amber-600'}`} />
              <div className="flex-1">
                <p className="text-sm font-semibold text-slate-900">
                  {t('sla_alerts') || 'Требуется внимание'}
                </p>
                <p className="text-xs text-slate-600 mt-1">
                  {isBreached
                    ? `${stats!.sla_breached} ${t('sla_breached') || 'нарядов с просроченным SLA'}`
                    : `${stats?.sla_at_risk || 0} ${t('sla_at_risk') || 'нарядов с SLA на исходе'}`
                  }
                </p>
              </div>
              <button onClick={fetchAssignedWorkOrders} className="text-xs text-blue-600 hover:text-blue-800 font-medium">
                {t('refresh') || 'Обновить'}
              </button>
            </div>
          </div>
        );
      })()}

      {/* KPI Cards */}
      <div className="grid grid-cols-1 md:grid-cols-4 gap-4">
        <StatsCard
          title={t('assigned') || 'Назначено'}
          value={stats?.total_assigned ?? '—'}
          icon={Wrench}
          iconBgColor="bg-blue-50"
          iconColor="text-blue-600"
        />
        <StatsCard
          title={t('in_progress') || 'В работе'}
          value={stats?.in_progress ?? '—'}
          icon={Play}
          iconBgColor="bg-amber-50"
          iconColor="text-amber-600"
        />
        <StatsCard
          title={t('completed_today') || 'Завершено сегодня'}
          value={stats?.completed_today ?? '—'}
          icon={CheckCircle}
          iconBgColor="bg-emerald-50"
          iconColor="text-emerald-600"
        />
        <Card>
          <div className="p-4">
            <div className="flex items-center gap-3">
              <div className={`p-2.5 rounded-xl ${(stats?.sla_breached || 0) > 0 ? 'bg-red-50' : 'bg-slate-50'}`}>
                <AlertTriangle className={`w-5 h-5 ${(stats?.sla_breached || 0) > 0 ? 'text-red-600' : 'text-slate-500'}`} />
              </div>
              <div>
                <p className="text-xs text-slate-500">{t('sla_issues') || 'SLA проблемы'}</p>
                <p className={`text-xl font-bold ${(stats?.sla_breached || 0) > 0 ? 'text-red-700' : 'text-slate-900'}`}>
                  {(stats?.sla_breached || 0) + (stats?.sla_at_risk || 0)}
                </p>
              </div>
            </div>
          </div>
        </Card>
      </div>

      {/* My Work Orders */}
      <Card>
        <div className="p-5">
          <div className="flex items-center justify-between mb-4">
            <h3 className="text-sm font-semibold text-slate-900 flex items-center gap-2">
              <Wrench className="w-4 h-4" />
              {t('my_work_orders') || 'Мои наряды'}
              <Badge variant="info" size="sm">{assignedWO.length}</Badge>
            </h3>
            <Button
              size="sm"
              variant="outline"
              icon={<ArrowRight className="w-3 h-3" />}
              onClick={() => navigate('/work-orders')}
            >
              {t('view_all') || 'Все'}
            </Button>
          </div>
          <DataGrid
            data={sortedWO}
            columns={woColumns}
            keyExtractor={(wo: AssignedWorkOrder) => wo.id}
            loading={loading.wos}
            emptyMessage={t('no_assigned_work_orders') || 'Нет назначенных нарядов'}
            variant="striped"
            defaultDensity="compact"
            pageSize={10}
            exportFilename="my-work-orders.csv"
            onRowClick={(wo: AssignedWorkOrder) => navigate(`/work-orders/${wo.id}`)}
          />
        </div>
      </Card>

      {/* Team Workload */}
      <Card>
        <div className="p-5">
          <div className="flex items-center justify-between mb-4">
            <h3 className="text-sm font-semibold text-slate-900 flex items-center gap-2">
              <Users className="w-4 h-4" />
              {t('team_workload') || 'Загрузка команды'}
            </h3>
            <div className="flex items-center gap-4 text-xs text-slate-500">
              <span className="flex items-center gap-1"><span className="w-2 h-2 rounded-full bg-emerald-500" /> {t('available') || 'Доступен'}</span>
              <span className="flex items-center gap-1"><span className="w-2 h-2 rounded-full bg-amber-500" /> {t('busy') || 'Занят'}</span>
              <span className="flex items-center gap-1"><span className="w-2 h-2 rounded-full bg-red-500" /> {t('overloaded') || 'Перегружен'}</span>
            </div>
          </div>
          <DataGrid
            data={workloads}
            columns={workloadColumns}
            keyExtractor={(item: TechnicianWorkload) => item.user_id}
            loading={loading.workloads}
            emptyMessage={t('no_technicians') || 'Нет данных о техниках'}
            variant="striped"
            defaultDensity="compact"
            pageSize={10}
            exportFilename="technician-workload.csv"
          />
        </div>
      </Card>
    </div>
  );
};
