import React, { useState, useEffect } from 'react';
import { useTranslation } from 'react-i18next';
import { request } from '../services/api';
import { Card, DataGrid, Badge, Button, StatsCard } from '../components/ui';
import { useNavigate } from 'react-router-dom';
import {
  Clock, AlertTriangle, CheckCircle, Filter,
  BarChart3, PieChart, TrendingUp, Calendar, RefreshCw,
} from 'lucide-react';
import { ResponsiveBar } from '@nivo/bar';
import { ResponsivePie } from '@nivo/pie';

// ── Types ────────────────────────────────────────────────────────────

interface AgingWorkOrder {
  id: string;
  title: string;
  device_name: string;
  priority: string;
  status: string;
  created_at: string;
  updated_at: string;
  sla_deadline?: string;
  sla_status?: string;
  assignee_name?: string;
}

interface AgingBucket {
  label: string;
  fromHours: number;
  toHours: number;
  color: string;
  count: number;
  wos: AgingWorkOrder[];
}

// ── Constants ────────────────────────────────────────────────────────

const AGING_BUCKETS: AgingBucket[] = [
  { label: '< 1ч', fromHours: 0, toHours: 1, color: '#10b981', count: 0, wos: [] },
  { label: '1-4ч', fromHours: 1, toHours: 4, color: '#06b6d4', count: 0, wos: [] },
  { label: '4-8ч', fromHours: 4, toHours: 8, color: '#3b82f6', count: 0, wos: [] },
  { label: '8-24ч', fromHours: 8, toHours: 24, color: '#f59e0b', count: 0, wos: [] },
  { label: '1-3д', fromHours: 24, toHours: 72, color: '#f97316', count: 0, wos: [] },
  { label: '3-7д', fromHours: 72, toHours: 168, color: '#ef4444', count: 0, wos: [] },
  { label: '> 7д', fromHours: 168, toHours: Infinity, color: '#dc2626', count: 0, wos: [] },
];

const PRIORITY_COLORS: Record<string, string> = {
  CRITICAL: '#dc2626',
  HIGH: '#f97316',
  MEDIUM: '#f59e0b',
  LOW: '#10b981',
};

// ── Helpers ──────────────────────────────────────────────────────────

function hoursSince(dateStr: string): number {
  return (Date.now() - new Date(dateStr).getTime()) / 3600000;
}

function formatDuration(hours: number): string {
  if (hours < 1) return `${Math.round(hours * 60)} мин`;
  if (hours < 24) return `${Math.round(hours)} ч`;
  return `${Math.round(hours / 24)} д ${Math.round(hours % 24)} ч`;
}

// ── Main Component ───────────────────────────────────────────────────

export function WOAging() {
  const { t } = useTranslation();
  const navigate = useNavigate();
  const [workOrders, setWorkOrders] = useState<AgingWorkOrder[]>([]);
  const [loading, setLoading] = useState(false);
  const [filter, setFilter] = useState<string>('all');

  useEffect(() => { fetchData(); }, []);

  const fetchData = async () => {
    setLoading(true);
    try {
      const data = await request<AgingWorkOrder[]>('/work-orders?limit=500');
      setWorkOrders(data?.filter(wo => wo.status !== 'COMPLETED' && wo.status !== 'CANCELLED') || []);
    } catch { /* ignore */ }
    finally { setLoading(false); }
  };

  // Buckets
  const filteredWO = filter === 'all'
    ? workOrders
    : workOrders.filter(wo => wo.priority === filter);

  const buckets = AGING_BUCKETS.map(b => {
    const wos = filteredWO.filter(wo => {
      const h = hoursSince(wo.created_at);
      return h >= b.fromHours && h < b.toHours;
    });
    return { ...b, count: wos.length, wos };
  });

  const totalActive = filteredWO.length;
  const overdue = filteredWO.filter(wo => wo.sla_status === 'breached').length;
  const atRisk = filteredWO.filter(wo => wo.sla_status === 'at_risk').length;
  const avgAge = totalActive > 0
    ? filteredWO.reduce((sum, wo) => sum + hoursSince(wo.created_at), 0) / totalActive
    : 0;

  // Chart data
  const chartData = buckets.filter(b => b.count > 0).map(b => ({
    name: b.label,
    count: b.count,
    fill: b.color,
  }));

  const pieData = buckets.filter(b => b.count > 0).map(b => ({
    id: b.label,
    label: b.label,
    value: b.count,
    color: b.color,
  }));

  // Table columns
  const columns = [
    {
      key: 'created_at',
      header: t('age') || 'Возраст',
      sortable: true,
      render: (wo: AgingWorkOrder) => {
        const h = hoursSince(wo.created_at);
        const bucket = AGING_BUCKETS.find(b => h >= b.fromHours && h < b.toHours);
        return (
          <div className="flex items-center gap-2">
            <div className="w-2 h-2 rounded-full" style={{ backgroundColor: bucket?.color || '#94a3b8' }} />
            <span className="text-sm font-mono">{formatDuration(h)}</span>
          </div>
        );
      },
    },
    {
      key: 'title',
      header: t('work_order') || 'Наряд',
      sortable: true,
      render: (wo: AgingWorkOrder) => (
        <div>
          <span className="text-sm font-medium text-slate-900">{wo.title || wo.device_name || wo.id.slice(0, 8)}</span>
          {wo.device_name && <p className="text-xs text-slate-500">{wo.device_name}</p>}
        </div>
      ),
    },
    {
      key: 'priority',
      header: t('priority') || 'Приоритет',
      sortable: true,
      render: (wo: AgingWorkOrder) => (
        <span className={`px-2 py-0.5 rounded text-xs font-bold ${
          wo.priority === 'CRITICAL' ? 'bg-red-100 text-red-700' :
          wo.priority === 'HIGH' ? 'bg-orange-100 text-orange-700' :
          wo.priority === 'MEDIUM' ? 'bg-yellow-100 text-yellow-700' :
          'bg-green-100 text-green-700'
        }`}>
          {wo.priority}
        </span>
      ),
    },
    {
      key: 'sla_status',
      header: 'SLA',
      render: (wo: AgingWorkOrder) => {
        if (wo.sla_status === 'breached') return <Badge variant="danger">Просрочен</Badge>;
        if (wo.sla_status === 'at_risk') return <Badge variant="warning">На исходе</Badge>;
        return <Badge variant="info">В норме</Badge>;
      },
    },
    {
      key: 'assignee_name',
      header: t('assignee') || 'Исполнитель',
      render: (wo: AgingWorkOrder) => wo.assignee_name || '—',
    },
    {
      key: 'actions',
      header: '',
      render: (wo: AgingWorkOrder) => (
        <button
          onClick={() => navigate(`/work-orders/${wo.id}`)}
          className="text-xs text-blue-600 hover:text-blue-800 font-medium"
        >
          {t('open') || 'Открыть'}
        </button>
      ),
    },
  ];

  const nivoTheme = {
    axis: {
      ticks: { text: { fontSize: 11, fill: '#94a3b8' } },
      domain: { line: { stroke: '#f1f5f9', strokeWidth: 1 } },
    },
    grid: { line: { stroke: '#f1f5f9', strokeDasharray: '3 3', strokeWidth: 1 } },
  };

  return (
    <div className="space-y-6">
      {/* Header */}
      <div className="flex items-center justify-between">
        <div>
          <h1 className="text-2xl font-bold text-slate-900 flex items-center gap-2">
            <Clock className="w-6 h-6" />
            {t('wo_aging') || 'Анализ возраста нарядов'}
          </h1>
          <p className="text-sm text-slate-500 mt-1">
            {t('wo_aging_desc') || 'Распределение активных нарядов по времени с момента создания'}
          </p>
        </div>
        <Button variant="outline" icon={<RefreshCw className="w-4 h-4" />} onClick={fetchData} loading={loading}>
          {t('refresh') || 'Обновить'}
        </Button>
      </div>

      {/* KPI Cards */}
      <div className="grid grid-cols-1 md:grid-cols-4 gap-4">
        <StatsCard
          title={t('active_work_orders') || 'Активные наряды'}
          value={totalActive}
          icon={Clock}
          iconBgColor="bg-blue-50"
          iconColor="text-blue-600"
        />
        <StatsCard
          title={t('overdue') || 'Просрочено (SLA)'}
          value={overdue}
          icon={AlertTriangle}
          iconBgColor="bg-red-50"
          iconColor="text-red-600"
        />
        <StatsCard
          title={t('at_risk') || 'На исходе (SLA)'}
          value={atRisk}
          icon={TrendingUp}
          iconBgColor="bg-amber-50"
          iconColor="text-amber-600"
        />
        <StatsCard
          title={t('avg_age') || 'Средний возраст'}
          value={formatDuration(avgAge)}
          icon={Calendar}
          iconBgColor="bg-purple-50"
          iconColor="text-purple-600"
        />
      </div>

      {/* Filters */}
      <div className="flex items-center gap-2">
        <Filter className="w-4 h-4 text-slate-400" />
        {['all', 'CRITICAL', 'HIGH', 'MEDIUM', 'LOW'].map((p) => (
          <button
            key={p}
            onClick={() => setFilter(p)}
            className={`px-3 py-1.5 text-xs font-medium rounded-lg transition-colors ${
              filter === p
                ? 'bg-blue-600 text-white'
                : 'bg-slate-100 text-slate-600 hover:bg-slate-200'
            }`}
          >
            {p === 'all' ? (t('all') || 'Все') : p}
          </button>
        ))}
      </div>

      {/* Charts */}
      <div className="grid grid-cols-1 lg:grid-cols-2 gap-4">
        {/* Bar Chart */}
        <Card>
          <div className="p-4">
            <h3 className="text-sm font-semibold text-slate-900 mb-4 flex items-center gap-2">
              <BarChart3 className="w-4 h-4" />
              {t('aging_distribution') || 'Распределение по возрасту'}
            </h3>
            {chartData.length > 0 ? (
              <div className="h-64">
                <ResponsiveBar
                  data={chartData}
                  keys={['count']}
                  indexBy="name"
                  margin={{ top: 10, right: 20, bottom: 30, left: 50 }}
                  padding={0.3}
                  colors={{ datum: 'data.fill' }}
                  colorBy="indexValue"
                  borderRadius={4}
                  axisBottom={{
                    tickSize: 5, tickPadding: 5, tickRotation: 0,
                  }}
                  axisLeft={{
                    tickSize: 5, tickPadding: 5, tickRotation: 0,
                  }}
                  theme={nivoTheme}
                  enableLabel={false}
                  tooltip={({ data: d }) => (
                    <div style={{ background: 'rgba(255,255,255,0.95)', border: '1px solid #e2e8f0', borderRadius: 8, padding: '4px 8px', fontSize: 12 }}>
                      <strong>{String(d.name)}</strong>: {Number(d.count)} {t('work_orders') || 'Нарядов'}
                    </div>
                  )}
                />
              </div>
            ) : (
              <div className="flex items-center justify-center h-64 text-sm text-slate-400">
                {t('no_data') || 'Нет данных'}
              </div>
            )}
          </div>
        </Card>

        {/* Pie Chart */}
        <Card>
          <div className="p-4">
            <h3 className="text-sm font-semibold text-slate-900 mb-4 flex items-center gap-2">
              <PieChart className="w-4 h-4" />
              {t('aging_pie') || 'Соотношение возрастных групп'}
            </h3>
            {pieData.length > 0 ? (
              <div className="h-64">
                <ResponsivePie
                  data={pieData}
                  margin={{ top: 20, right: 40, bottom: 20, left: 40 }}
                  innerRadius={0.45}
                  padAngle={2}
                  cornerRadius={4}
                  colors={{ datum: 'data.color' }}
                  arcLinkLabelsSkipAngle={10}
                  arcLinkLabelsTextColor="#64748b"
                  arcLinkLabelsThickness={1}
                  arcLinkLabelsColor={{ from: 'color' }}
                  arcLabelsSkipAngle={10}
                  arcLabelsTextColor="#ffffff"
                  theme={nivoTheme}
                  tooltip={({ datum }) => (
                    <div style={{ background: 'rgba(255,255,255,0.95)', border: '1px solid #e2e8f0', borderRadius: 8, padding: '4px 8px', fontSize: 12 }}>
                      <strong>{datum.label}</strong>: {datum.value} {t('work_orders') || 'Нарядов'}
                    </div>
                  )}
                  legends={[
                    {
                      anchor: 'bottom',
                      direction: 'row',
                      translateY: 36,
                      itemWidth: 60,
                      itemHeight: 14,
                      itemTextColor: '#94a3b8',
                      symbolSize: 10,
                      symbolShape: 'circle',
                    },
                  ]}
                />
              </div>
            ) : (
              <div className="flex items-center justify-center h-64 text-sm text-slate-400">
                {t('no_data') || 'Нет данных'}
              </div>
            )}
          </div>
        </Card>
      </div>

      {/* Aging Buckets Table */}
      <div className="grid grid-cols-1 md:grid-cols-2 lg:grid-cols-4 gap-3">
        {buckets.map((bucket) => (
          <div
            key={bucket.label}
            className="p-4 rounded-xl border border-slate-200 bg-white cursor-pointer hover:shadow-md transition-shadow"
            onClick={() => {
              const el = document.getElementById(`aging-table-${bucket.label.replace(/[<>\/]/g, '')}`);
              el?.scrollIntoView({ behavior: 'smooth' });
            }}
          >
            <div className="flex items-center justify-between mb-2">
              <span className="text-xs font-medium text-slate-500">{bucket.label}</span>
              <div className="w-3 h-3 rounded-full" style={{ backgroundColor: bucket.color }} />
            </div>
            <p className="text-2xl font-bold text-slate-900">{bucket.count}</p>
            <p className="text-xs text-slate-400 mt-1">
              {totalActive > 0 ? `${(bucket.count / totalActive * 100).toFixed(1)}%` : '0%'}
            </p>
          </div>
        ))}
      </div>

      {/* Detail Table */}
      <Card>
        <div className="p-4">
          <DataGrid
            data={filteredWO}
            columns={columns}
            keyExtractor={(wo: AgingWorkOrder) => wo.id}
            loading={loading}
            emptyMessage={t('no_active_work_orders') || 'Нет активных нарядов'}
            variant="striped"
            defaultDensity="compact"
            pageSize={25}
            exportFilename="wo-aging.csv"
          />
        </div>
      </Card>
    </div>
  );
}
