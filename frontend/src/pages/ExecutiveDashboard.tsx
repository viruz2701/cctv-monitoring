import React, { useState, useEffect } from 'react';
import { useTranslation } from 'react-i18next';
import { request } from '../services/api';
import { Card, Badge, StatsCard } from '../components/ui';
import { useNavigate } from 'react-router-dom';
import {
  TrendingUp, DollarSign, Activity, Users, Wifi,
  Clock, AlertTriangle, CheckCircle, Camera,
  BarChart3, ArrowUpRight, ArrowDownRight,
} from 'lucide-react';
import {
  LineChart, Line, BarChart, Bar, PieChart, Pie, Cell,
  XAxis, YAxis, CartesianGrid, Tooltip, ResponsiveContainer,
  Area, AreaChart, Legend,
} from 'recharts';

// ── Types ────────────────────────────────────────────────────────────

interface DashboardSnapshot {
  total_devices: number;
  online_devices: number;
  offline_devices: number;
  warning_devices: number;
  total_work_orders: number;
  open_work_orders: number;
  completed_today: number;
  sla_compliance: number;
  avg_response_time_hours: number;
  total_technicians: number;
  active_technicians: number;
  monthly_cost: number;
  monthly_savings: number;
}

const PIE_COLORS = ['#10b981', '#ef4444', '#f59e0b', '#3b82f6'];

// Weekly trend data (mock)
const WEEKLY_DATA = [
  { day: 'Пн', wos: 12, resolved: 10, alerts: 5 },
  { day: 'Вт', wos: 15, resolved: 14, alerts: 3 },
  { day: 'Ср', wos: 8, resolved: 9, alerts: 7 },
  { day: 'Чт', wos: 20, resolved: 18, alerts: 4 },
  { day: 'Пт', wos: 14, resolved: 15, alerts: 6 },
  { day: 'Сб', wos: 6, resolved: 5, alerts: 2 },
  { day: 'Вс', wos: 4, resolved: 3, alerts: 1 },
];

const MONTHLY_COST_DATA = [
  { month: 'Янв', labor: 12000, parts: 4500, additional: 1200 },
  { month: 'Фев', labor: 11000, parts: 3800, additional: 900 },
  { month: 'Мар', labor: 13500, parts: 5200, additional: 1500 },
  { month: 'Апр', labor: 12800, parts: 4800, additional: 1100 },
  { month: 'Май', labor: 14200, parts: 6100, additional: 1800 },
  { month: 'Июн', labor: 13800, parts: 5600, additional: 1400 },
];

export function ExecutiveDashboard() {
  const { t } = useTranslation();
  const navigate = useNavigate();
  const [snapshot, setSnapshot] = useState<DashboardSnapshot | null>(null);
  const [loading, setLoading] = useState(true);

  useEffect(() => {
    const fetch = async () => {
      try {
        const stats = await request<DashboardSnapshot>('/dashboard/stats');
        setSnapshot(stats);
      } catch {
        setSnapshot({
          total_devices: 1250, online_devices: 1120, offline_devices: 80, warning_devices: 50,
          total_work_orders: 342, open_work_orders: 89, completed_today: 12, sla_compliance: 94.5,
          avg_response_time_hours: 2.3, total_technicians: 24, active_technicians: 18,
          monthly_cost: 45200, monthly_savings: 8200,
        });
      } finally { setLoading(false); }
    };
    fetch();
  }, []);

  const s = snapshot;
  const uptime = s ? ((s.online_devices / s.total_devices) * 100).toFixed(1) : '—';
  const devicePie = s ? [
    { name: t('online') || 'Online', value: s.online_devices },
    { name: t('offline') || 'Offline', value: s.offline_devices },
    { name: t('warning') || 'Warning', value: s.warning_devices },
  ].filter(d => d.value > 0) : [];

  const totalMonthlyCost = MONTHLY_COST_DATA[MONTHLY_COST_DATA.length - 1];
  const costTrend = MONTHLY_COST_DATA.length > 1
    ? ((totalMonthlyCost.labor + totalMonthlyCost.parts + totalMonthlyCost.additional) /
       (MONTHLY_COST_DATA[MONTHLY_COST_DATA.length - 2].labor +
        MONTHLY_COST_DATA[MONTHLY_COST_DATA.length - 2].parts +
        MONTHLY_COST_DATA[MONTHLY_COST_DATA.length - 2].additional) - 1) * 100
    : 0;

  return (
    <div className="space-y-6">
      {/* Header */}
      <div>
        <h1 className="text-2xl font-bold text-slate-900 flex items-center gap-2">
          <BarChart3 className="w-6 h-6" />
          {t('executive_dashboard') || 'Executive Dashboard'}
        </h1>
        <p className="text-sm text-slate-500 mt-1">
          {t('executive_dashboard_desc') || 'Ключевые метрики для руководства'}
        </p>
      </div>

      {/* KPI Row 1: Operations */}
      <div className="grid grid-cols-1 md:grid-cols-2 lg:grid-cols-4 gap-4">
        <StatsCard title={t('uptime') || 'Uptime'} value={`${uptime}%`} icon={Wifi}
          iconBgColor={Number(uptime) >= 99 ? 'bg-emerald-50' : Number(uptime) >= 95 ? 'bg-amber-50' : 'bg-red-50'}
          iconColor={Number(uptime) >= 99 ? 'text-emerald-600' : Number(uptime) >= 95 ? 'text-amber-600' : 'text-red-600'} />
        <StatsCard title={t('sla_compliance') || 'SLA Compliance'} value={s ? `${s.sla_compliance}%` : '—'} icon={CheckCircle}
          iconBgColor={(s?.sla_compliance || 0) >= 95 ? 'bg-emerald-50' : 'bg-amber-50'}
          iconColor={(s?.sla_compliance || 0) >= 95 ? 'text-emerald-600' : 'text-amber-600'} />
        <StatsCard title={t('open_wo') || 'Open WOs'} value={s?.open_work_orders ?? '—'} icon={Activity}
          iconBgColor={(s?.open_work_orders || 0) > 100 ? 'bg-red-50' : 'bg-blue-50'}
          iconColor={(s?.open_work_orders || 0) > 100 ? 'text-red-600' : 'text-blue-600'} />
        <StatsCard title={t('avg_response') || 'Avg Response'} value={s ? `${s.avg_response_time_hours}h` : '—'} icon={Clock}
          iconBgColor={(s?.avg_response_time_hours || 0) < 2 ? 'bg-emerald-50' : 'bg-amber-50'}
          iconColor={(s?.avg_response_time_hours || 0) < 2 ? 'text-emerald-600' : 'text-amber-600'} />
      </div>

      {/* KPI Row 2: Business */}
      <div className="grid grid-cols-1 md:grid-cols-2 lg:grid-cols-4 gap-4">
        <StatsCard title={t('total_devices') || 'Devices'} value={s?.total_devices ?? '—'} icon={Camera}
          iconBgColor="bg-indigo-50" iconColor="text-indigo-600" />
        <StatsCard title={t('technicians') || 'Technicians'} value={`${s?.active_technicians ?? 0}/${s?.total_technicians ?? 0}`} icon={Users}
          iconBgColor="bg-purple-50" iconColor="text-purple-600" />
        <StatsCard title={t('completed_today') || 'Completed Today'} value={s?.completed_today ?? '—'} icon={CheckCircle}
          iconBgColor="bg-emerald-50" iconColor="text-emerald-600" />
        <StatsCard title={t('monthly_cost') || 'Monthly Cost'} value={s ? `$${(s.monthly_cost / 1000).toFixed(1)}k` : '—'} icon={DollarSign}
          iconBgColor="bg-rose-50" iconColor="text-rose-600"
          trend={{ value: costTrend, direction: costTrend > 0 ? 'up' : 'down', label: t('vs_last_month') || 'vs last month' }} />
      </div>

      {/* Charts */}
      <div className="grid grid-cols-1 lg:grid-cols-2 gap-6">
        {/* Weekly Trend */}
        <Card>
          <div className="p-4">
            <h3 className="text-sm font-semibold text-slate-900 mb-4">{t('weekly_trend') || 'Недельная динамика'}</h3>
            <div className="h-64">
              <ResponsiveContainer width="100%" height="100%">
                <AreaChart data={WEEKLY_DATA}>
                  <defs>
                    <linearGradient id="colorWos" x1="0" y1="0" x2="0" y2="1"><stop offset="5%" stopColor="#3b82f6" stopOpacity={0.2} /><stop offset="95%" stopColor="#3b82f6" stopOpacity={0} /></linearGradient>
                    <linearGradient id="colorResolved" x1="0" y1="0" x2="0" y2="1"><stop offset="5%" stopColor="#10b981" stopOpacity={0.2} /><stop offset="95%" stopColor="#10b981" stopOpacity={0} /></linearGradient>
                  </defs>
                  <CartesianGrid strokeDasharray="3 3" stroke="#f1f5f9" />
                  <XAxis dataKey="day" tick={{ fontSize: 11, fill: '#94a3b8' }} />
                  <YAxis tick={{ fontSize: 11, fill: '#94a3b8' }} />
                  <Tooltip contentStyle={{ fontSize: 12, borderRadius: 8 }} />
                  <Legend wrapperStyle={{ fontSize: 11 }} />
                  <Area type="monotone" dataKey="wos" name="Наряды" stroke="#3b82f6" fill="url(#colorWos)" strokeWidth={2} />
                  <Area type="monotone" dataKey="resolved" name="Решено" stroke="#10b981" fill="url(#colorResolved)" strokeWidth={2} />
                </AreaChart>
              </ResponsiveContainer>
            </div>
          </div>
        </Card>

        {/* Monthly Cost Breakdown */}
        <Card>
          <div className="p-4">
            <h3 className="text-sm font-semibold text-slate-900 mb-4">{t('monthly_cost_trend') || 'Динамика затрат'}</h3>
            <div className="h-64">
              <ResponsiveContainer width="100%" height="100%">
                <BarChart data={MONTHLY_COST_DATA}>
                  <CartesianGrid strokeDasharray="3 3" stroke="#f1f5f9" />
                  <XAxis dataKey="month" tick={{ fontSize: 11, fill: '#94a3b8' }} />
                  <YAxis tick={{ fontSize: 11, fill: '#94a3b8' }} />
                  <Tooltip contentStyle={{ fontSize: 12, borderRadius: 8 }} formatter={(v: any) => [`$${Number(v).toLocaleString()}`, '']} />
                  <Legend wrapperStyle={{ fontSize: 11 }} />
                  <Bar dataKey="labor" name="Labor" stackId="a" fill="#3b82f6" radius={[0, 0, 0, 0]} />
                  <Bar dataKey="parts" name="Parts" stackId="a" fill="#f59e0b" />
                  <Bar dataKey="additional" name="Additional" stackId="a" fill="#10b981" radius={[4, 4, 0, 0]} />
                </BarChart>
              </ResponsiveContainer>
            </div>
          </div>
        </Card>
      </div>

      {/* Device Status + KPI Quick Links */}
      <div className="grid grid-cols-1 lg:grid-cols-3 gap-6">
        {/* Device Status Pie */}
        <Card>
          <div className="p-4">
            <h3 className="text-sm font-semibold text-slate-900 mb-4">{t('device_status') || 'Статус устройств'}</h3>
            <div className="h-48">
              <ResponsiveContainer width="100%" height="100%">
                <PieChart>
                  <Pie data={devicePie} cx="50%" cy="50%" innerRadius={45} outerRadius={70} dataKey="value" paddingAngle={3}>
                    {devicePie.map((_, idx) => <Cell key={idx} fill={PIE_COLORS[idx]} />)}
                  </Pie>
                  <Tooltip />
                  <Legend wrapperStyle={{ fontSize: 11 }} />
                </PieChart>
              </ResponsiveContainer>
            </div>
          </div>
        </Card>

        {/* Quick Actions */}
        <Card>
          <div className="p-4">
            <h3 className="text-sm font-semibold text-slate-900 mb-4">{t('quick_actions') || 'Быстрые действия'}</h3>
            <div className="space-y-2">
              {[
                { label: t('cost_dashboard') || 'Cost Dashboard', path: '/cost-dashboard', icon: DollarSign, color: 'text-emerald-600 bg-emerald-50' },
                { label: t('workload_analytics') || 'Workload Analytics', path: '/workload-analytics', icon: Users, color: 'text-blue-600 bg-blue-50' },
                { label: t('sla') || 'SLA Dashboard', path: '/sla', icon: Activity, color: 'text-purple-600 bg-purple-50' },
                { label: t('meter_dashboard') || 'Meter Dashboard', path: '/meter-dashboard', icon: BarChart3, color: 'text-amber-600 bg-amber-50' },
              ].map(item => (
                <button key={item.path} onClick={() => navigate(item.path)}
                  className={`w-full flex items-center gap-3 p-3 rounded-lg ${item.color} hover:opacity-80 transition-opacity`}>
                  <item.icon className="w-5 h-5" />
                  <span className="text-sm font-medium">{item.label}</span>
                  <ArrowUpRight className="w-4 h-4 ml-auto" />
                </button>
              ))}
            </div>
          </div>
        </Card>

        {/* Key Metrics */}
        <Card>
          <div className="p-4">
            <h3 className="text-sm font-semibold text-slate-900 mb-4">{t('key_metrics') || 'Ключевые метрики'}</h3>
            <div className="space-y-3">
              {[
                { label: t('devices_per_tech') || 'Устр. на техника', value: s && s.active_technicians > 0 ? (s.total_devices / s.active_technicians).toFixed(0) : '—', unit: '', icon: Camera },
                { label: t('wo_per_day') || 'Нарядов/день', value: s ? (s.completed_today || 0).toFixed(0) : '—', unit: '', icon: Activity },
                { label: t('cost_per_wo') || 'Стоимость/наряд', value: s && s.total_work_orders > 0 ? `$${(s.monthly_cost / s.total_work_orders).toFixed(0)}` : '—', unit: '', icon: DollarSign },
                { label: t('tech_utilization') || 'Загрузка техников', value: s && s.total_technicians > 0 ? `${((s.active_technicians / s.total_technicians) * 100).toFixed(0)}%` : '—', unit: '', icon: Users },
              ].map((metric, idx) => (
                <div key={idx} className="flex items-center justify-between p-2.5 bg-slate-50 rounded-lg">
                  <div className="flex items-center gap-2">
                    <metric.icon className="w-4 h-4 text-slate-400" />
                    <span className="text-xs text-slate-600">{metric.label}</span>
                  </div>
                  <span className="text-sm font-bold text-slate-900">{metric.value}</span>
                </div>
              ))}
            </div>
          </div>
        </Card>
      </div>

      {/* Alert: SLA compliance warning */}
      {(s?.sla_compliance || 0) < 90 && (
        <div className="p-4 bg-red-50 rounded-lg border border-red-200 flex items-center gap-3">
          <AlertTriangle className="w-5 h-5 text-red-600" />
          <p className="text-sm text-red-700">
            {t('sla_compliance_warning') || 'Внимание! SLA compliance ниже 90%. Требуется анализ.'}
          </p>
        </div>
      )}
    </div>
  );
}
