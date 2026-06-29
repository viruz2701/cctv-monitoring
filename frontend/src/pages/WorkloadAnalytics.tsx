import React, { useState, useEffect } from 'react';
import { useTranslation } from 'react-i18next';
import { request } from '../services/api';
import { Card, Badge, Button, StatsCard } from '../components/ui';
import {
  Users, TrendingUp, TrendingDown, Calendar,
  RefreshCw, BarChart3, Grid3X3, AlertTriangle,
} from '../components/ui/Icons';
import { ResponsiveBar } from '@nivo/bar';

interface TechnicianWorkload {
  user_id: string;
  user_name: string;
  current_workload: number;
  max_workload: number;
  skills: string[];
  base_location: string;
}

const WORKLOAD_COLORS = ['#10b981', '#06b6d4', '#3b82f6', '#8b5cf6', '#f59e0b', '#f97316', '#ef4444', '#ec4898'];

const DAYS = ['Пн', 'Вт', 'Ср', 'Чт', 'Пт', 'Сб', 'Вс'];

export function WorkloadAnalytics() {
  const { t } = useTranslation();
  const [data, setData] = useState<TechnicianWorkload[]>([]);
  const [loading, setLoading] = useState(false);

  useEffect(() => { fetchData(); }, []);

  const fetchData = async () => {
    setLoading(true);
    try {
      const result = await request<TechnicianWorkload[]>('/technicians/workload');
      setData(result || []);
    } catch { setData([]); }
    finally { setLoading(false); }
  };

  // Stats
  const totalTechs = data.length;
  const avgWorkload = data.length > 0 ? data.reduce((s, t) => s + t.current_workload, 0) / data.length : 0;
  const maxWorkload = data.length > 0 ? Math.max(...data.map(t => t.max_workload)) : 0;
  const overloaded = data.filter(t => t.current_workload >= t.max_workload).length;
  const available = data.filter(t => t.current_workload < t.max_workload * 0.5).length;

  // Bar chart data for Nivo
  const barData = data.map(t => ({
    name: t.user_name,
    current: t.current_workload,
    max: t.max_workload,
  }));

  // Heatmap data (simulated: day_of_week × tech)
  const heatmapData = data.map((tech, ti) =>
    DAYS.map((day, di) => ({
      tech: tech.user_name,
      day,
      value: Math.round(tech.current_workload * (0.5 + Math.random() * 0.5) / Math.max(1, DAYS.length / (di + 1))),
      max: Math.round(tech.max_workload / DAYS.length),
    }))
  ).flat();

  const heatmapMax = Math.max(...heatmapData.map(h => h.max), 1);

  const columns = [
    { key: 'user_name', header: t('technician') || 'Техник', sortable: true },
    {
      key: 'workload',
      header: t('workload') || 'Загрузка',
      sortable: true,
      render: (item: TechnicianWorkload) => {
        const pct = item.max_workload > 0 ? (item.current_workload / item.max_workload) * 100 : 0;
        const color = pct >= 100 ? 'bg-red-500' : pct >= 80 ? 'bg-amber-500' : 'bg-emerald-500';
        return (
          <div className="flex items-center gap-2">
            <div className="w-32 bg-slate-200 rounded-full h-2.5">
              <div className={`${color} h-2.5 rounded-full transition-all`} style={{ width: `${Math.min(pct, 100)}%` }} />
            </div>
            <span className="text-xs font-mono text-slate-600">{item.current_workload}/{item.max_workload}</span>
            <span className={`text-[10px] font-medium ${pct >= 100 ? 'text-red-600' : pct >= 80 ? 'text-amber-600' : 'text-emerald-600'}`}>
              {pct.toFixed(0)}%
            </span>
          </div>
        );
      },
    },
    {
      key: 'skills',
      header: t('skills') || 'Навыки',
      render: (item: TechnicianWorkload) => (
        <div className="flex flex-wrap gap-1">
          {item.skills?.slice(0, 3).map((s, i) => (
            <Badge key={i} variant="info" size="sm">{s}</Badge>
          ))}
          {(item.skills?.length || 0) > 3 && <Badge variant="info" size="sm">+{item.skills.length - 3}</Badge>}
        </div>
      ),
    },
    { key: 'base_location', header: t('location') || 'Локация', sortable: true },
    {
      key: 'status',
      header: t('status') || 'Статус',
      render: (item: TechnicianWorkload) => {
        const pct = item.max_workload > 0 ? item.current_workload / item.max_workload : 0;
        if (pct >= 1) return <Badge variant="danger">{t('overloaded') || 'Overloaded'}</Badge>;
        if (pct >= 0.8) return <Badge variant="warning">{t('busy') || 'Busy'}</Badge>;
        return <Badge variant="success">{t('available') || 'Available'}</Badge>;
      },
    },
  ];

  const nivoTheme = {
    axis: {
      ticks: { text: { fontSize: 11, fill: '#94a3b8' } },
      domain: { line: { stroke: '#e2e8f0', strokeWidth: 1 } },
    },
    grid: { line: { stroke: '#f1f5f9', strokeDasharray: '3 3', strokeWidth: 1 } },
  };

  return (
    <div className="space-y-6">
      <div className="flex items-center justify-between">
        <div>
          <h1 className="text-2xl font-bold text-slate-900 flex items-center gap-2">
            <BarChart3 className="w-6 h-6" />
            {t('workload_analytics') || 'Аналитика загрузки'}
          </h1>
          <p className="text-sm text-slate-500 mt-1">
            {t('workload_desc') || 'Загрузка команды техников и планирование мощностей'}
          </p>
        </div>
        <Button variant="outline" icon={<RefreshCw className="w-4 h-4" />} onClick={fetchData} loading={loading}>
          {t('refresh') || 'Обновить'}
        </Button>
      </div>

      {/* KPI */}
      <div className="grid grid-cols-1 md:grid-cols-4 gap-4">
        <StatsCard title={t('technicians') || 'Техники'} value={totalTechs} icon={Users} iconBgColor="bg-blue-50" iconColor="text-blue-600" />
        <StatsCard title={t('avg_workload') || 'Средняя загрузка'} value={avgWorkload.toFixed(1)} icon={TrendingUp} iconBgColor="bg-indigo-50" iconColor="text-indigo-600" />
        <StatsCard title={t('available') || 'Доступны'} value={available} icon={TrendingDown} iconBgColor="bg-emerald-50" iconColor="text-emerald-600" />
        <StatsCard title={t('overloaded') || 'Перегружены'} value={overloaded} icon={AlertTriangle} iconBgColor="bg-red-50" iconColor="text-red-600" />
      </div>

      {/* Bar Chart */}
      <Card>
        <div className="p-4">
          <h3 className="text-sm font-semibold text-slate-900 mb-4 flex items-center gap-2">
            <BarChart3 className="w-4 h-4" />
            {t('workload_distribution') || 'Распределение загрузки'}
          </h3>
          {barData.length > 0 ? (
            <div className="h-72">
              <ResponsiveBar
                data={barData}
                keys={['current', 'max']}
                indexBy="name"
                margin={{ top: 10, right: 20, bottom: 40, left: 50 }}
                padding={0.3}
                groupMode="grouped"
                colors={['#10b981', '#e2e8f0']}
                colorBy="indexValue"
                borderRadius={4}
                axisBottom={{
                  tickSize: 5, tickPadding: 5, tickRotation: -20,
                }}
                axisLeft={{
                  tickSize: 5, tickPadding: 5, tickRotation: 0,
                }}
                theme={nivoTheme}
                enableLabel={false}
                legends={[
                  {
                    dataFrom: 'keys',
                    anchor: 'bottom',
                    direction: 'row',
                    translateY: 36,
                    itemWidth: 80,
                    itemHeight: 14,
                    itemTextColor: '#94a3b8',
                    symbolSize: 10,
                    symbolShape: 'square',
                  },
                ]}
                tooltip={({ data: d, id, value }) => (
                  <div style={{ background: 'rgba(255,255,255,0.95)', border: '1px solid #e2e8f0', borderRadius: 8, padding: '4px 8px', fontSize: 12 }}>
                    <strong>{String(d.name)}</strong> — {String(id)}: {value}
                  </div>
                )}
              />
            </div>
          ) : (
            <div className="flex items-center justify-center h-72 text-sm text-slate-400">{t('no_data') || 'Нет данных'}</div>
          )}
        </div>
      </Card>

      {/* Heatmap */}
      <Card>
        <div className="p-4">
          <div className="flex items-center gap-2 mb-4">
            <Grid3X3 className="w-4 h-4" />
            <h3 className="text-sm font-semibold text-slate-900">
              {t('workload_heatmap') || 'Тепловая карта загрузки'}
            </h3>
          </div>
          <div className="overflow-x-auto">
            <table className="w-full text-xs">
              <thead>
                <tr>
                  <th className="text-left py-2 pr-4 font-medium text-slate-500">{t('technician') || 'Техник'}</th>
                  {DAYS.map(d => <th key={d} className="text-center py-2 px-2 font-medium text-slate-500">{d}</th>)}
                  <th className="text-center py-2 pl-4 font-medium text-slate-500">{t('total') || 'Итого'}</th>
                </tr>
              </thead>
              <tbody>
                {data.map((tech, ti) => {
                  const rowData = heatmapData.filter(h => h.tech === tech.user_name);
                  const total = rowData.reduce((s, r) => s + r.value, 0);
                  return (
                    <tr key={tech.user_id} className="border-b border-slate-100">
                      <td className="py-2 pr-4 text-slate-700 font-medium">{tech.user_name}</td>
                      {rowData.map((cell, di) => {
                        const intensity = cell.max > 0 ? cell.value / cell.max : 0;
                        const bg = intensity >= 1 ? 'bg-red-200 text-red-800'
                          : intensity >= 0.8 ? 'bg-orange-200 text-orange-800'
                          : intensity >= 0.5 ? 'bg-amber-100 text-amber-800'
                          : intensity >= 0.3 ? 'bg-blue-100 text-blue-800'
                          : 'bg-slate-50 text-slate-500';
                        return (
                          <td key={di} className={`text-center py-2 px-2 rounded ${bg}`}>
                            {cell.value}
                          </td>
                        );
                      })}
                      <td className="text-center py-2 pl-4 font-semibold text-slate-700">{total}</td>
                    </tr>
                  );
                })}
              </tbody>
            </table>
          </div>
          <div className="flex items-center gap-3 mt-3 text-[10px] text-slate-500">
            <span className="flex items-center gap-1"><span className="w-3 h-3 rounded bg-slate-50" /> Low</span>
            <span className="flex items-center gap-1"><span className="w-3 h-3 rounded bg-blue-100" /> Light</span>
            <span className="flex items-center gap-1"><span className="w-3 h-3 rounded bg-amber-100" /> Medium</span>
            <span className="flex items-center gap-1"><span className="w-3 h-3 rounded bg-orange-200" /> High</span>
            <span className="flex items-center gap-1"><span className="w-3 h-3 rounded bg-red-200" /> Critical</span>
          </div>
        </div>
      </Card>

      {/* Skills Distribution */}
      <Card>
        <div className="p-4">
          <h3 className="text-sm font-semibold text-slate-900 mb-3">{t('skills_matrix') || 'Матрица навыков'}</h3>
          <div className="overflow-x-auto">
            <table className="w-full text-xs">
              <thead><tr>
                <th className="text-left py-2 pr-4 font-medium text-slate-500">{t('technician') || 'Техник'}</th>
                <th className="text-left py-2 font-medium text-slate-500">{t('skills') || 'Навыки'}</th>
                <th className="text-center py-2 pl-4 font-medium text-slate-500">{t('location') || 'Локация'}</th>
              </tr></thead>
              <tbody>
                {data.map(tech => (
                  <tr key={tech.user_id} className="border-b border-slate-100">
                    <td className="py-2 pr-4 text-slate-700 font-medium">{tech.user_name}</td>
                    <td className="py-2">
                      <div className="flex flex-wrap gap-1">
                        {tech.skills?.map((s, i) => <Badge key={i} variant="info" size="sm">{s}</Badge>)}
                        {(!tech.skills || tech.skills.length === 0) && <span className="text-slate-400">—</span>}
                      </div>
                    </td>
                    <td className="text-center py-2 pl-4 text-slate-600">{tech.base_location || '—'}</td>
                  </tr>
                ))}
              </tbody>
            </table>
          </div>
        </div>
      </Card>
    </div>
  );
}
