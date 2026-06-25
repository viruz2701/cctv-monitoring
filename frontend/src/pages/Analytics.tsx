import React, { useState, useEffect } from 'react';
import { Card, DataGrid, Badge, SkeletonStatsCard, SkeletonChart, SkeletonTable } from '../components/ui';
import { api, Prediction } from '../services/api';
import { useAuth } from '../hooks/useAuth';
import { useTranslation } from 'react-i18next';
import {
  LineChart, Line, AreaChart, Area, BarChart, Bar,
  XAxis, YAxis, Tooltip, ResponsiveContainer, CartesianGrid,
  PieChart, Pie, Cell, Legend,
} from 'recharts';
import { Activity, Clock, TrendingUp, Shield, AlertTriangle, CheckCircle } from 'lucide-react';

const mtbfTrendData = [
  { month: 'Янв', mtbf: 720 }, { month: 'Фев', mtbf: 680 },
  { month: 'Мар', mtbf: 750 }, { month: 'Апр', mtbf: 810 },
  { month: 'Май', mtbf: 790 }, { month: 'Июн', mtbf: 850 },
];

const mttrTrendData = [
  { month: 'Янв', mttr: 45 }, { month: 'Фев', mttr: 52 },
  { month: 'Мар', mttr: 38 }, { month: 'Апр', mttr: 42 },
  { month: 'Май', mttr: 35 }, { month: 'Июн', mttr: 30 },
];

const failureByTypeData = [
  { name: 'Камеры', value: 45, color: '#3b82f6' },
  { name: 'NVR', value: 25, color: '#f97316' },
  { name: 'Коммутаторы', value: 15, color: '#22c55e' },
  { name: 'Другое', value: 15, color: '#a855f7' },
];

export function Analytics() {
  const { t } = useTranslation();
  const { token } = useAuth();
  const [predictions, setPredictions] = useState<Prediction[]>([]);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState('');

  useEffect(() => {
    if (!token) return;
    api.getPredictions()
      .then(data => setPredictions(data))
      .catch(err => setError(err.message))
      .finally(() => setLoading(false));
  }, [token]);

  const avgMTBF = mtbfTrendData.reduce((a, b) => a + b.mtbf, 0) / mtbfTrendData.length;
  const avgMTTR = mttrTrendData.reduce((a, b) => a + b.mttr, 0) / mttrTrendData.length;
  const availability = ((avgMTBF / (avgMTBF + avgMTTR)) * 100).toFixed(2);

  if (loading) return (
    <div className="space-y-6" aria-label="Loading content">
      {/* Header skeleton */}
      <div className="space-y-2">
        <div className="h-7 w-48 bg-slate-200 dark:bg-slate-700 animate-pulse rounded" />
        <div className="h-4 w-64 bg-slate-200 dark:bg-slate-700 animate-pulse rounded" />
      </div>

      {/* Stats cards skeleton */}
      <div className="grid grid-cols-1 md:grid-cols-4 gap-4">
        <SkeletonStatsCard count={4} withTrend />
      </div>

      {/* Charts skeleton */}
      <div className="grid grid-cols-1 lg:grid-cols-2 gap-6">
        <SkeletonChart />
        <SkeletonChart />
      </div>

      {/* Table skeleton */}
      <SkeletonTable rows={5} columns={4} />
    </div>
  );
  if (error) return <div className="p-8 text-red-500">Error: {error}</div>;

  return (
    <div className="space-y-6">
      <h1 className="text-2xl font-bold text-slate-900 dark:text-white">{t('analytics_predictions')}</h1>

      {/* KPI Cards */}
      <div className="grid grid-cols-1 md:grid-cols-4 gap-4">
        <Card>
          <div className="p-5">
            <div className="flex items-center gap-3 mb-3">
              <div className="p-2 bg-blue-50 dark:bg-blue-900/30 rounded-lg">
                <Activity className="w-5 h-5 text-blue-600 dark:text-blue-400" />
              </div>
              <div>
                <p className="text-xs text-slate-500 dark:text-slate-400">MTBF</p>
                <p className="text-2xl font-bold text-slate-900 dark:text-white">
                  {Math.round(avgMTBF)} <span className="text-sm font-normal text-slate-500">ч</span>
                </p>
              </div>
            </div>
            <div className="flex items-center gap-1 text-xs text-emerald-600 dark:text-emerald-400">
              <TrendingUp className="w-3 h-3" />
              <span>+8% vs прошлый месяц</span>
            </div>
          </div>
        </Card>

        <Card>
          <div className="p-5">
            <div className="flex items-center gap-3 mb-3">
              <div className="p-2 bg-emerald-50 dark:bg-emerald-900/30 rounded-lg">
                <Clock className="w-5 h-5 text-emerald-600 dark:text-emerald-400" />
              </div>
              <div>
                <p className="text-xs text-slate-500 dark:text-slate-400">MTTR</p>
                <p className="text-2xl font-bold text-slate-900 dark:text-white">
                  {Math.round(avgMTTR)} <span className="text-sm font-normal text-slate-500">мин</span>
                </p>
              </div>
            </div>
            <div className="flex items-center gap-1 text-xs text-emerald-600 dark:text-emerald-400">
              <TrendingUp className="w-3 h-3" />
              <span>-12% vs прошлый месяц</span>
            </div>
          </div>
        </Card>

        <Card>
          <div className="p-5">
            <div className="flex items-center gap-3 mb-3">
              <div className="p-2 bg-purple-50 dark:bg-purple-900/30 rounded-lg">
                <Shield className="w-5 h-5 text-purple-600 dark:text-purple-400" />
              </div>
              <div>
                <p className="text-xs text-slate-500 dark:text-slate-400">Availability</p>
                <p className="text-2xl font-bold text-slate-900 dark:text-white">
                  {availability} <span className="text-sm font-normal text-slate-500">%</span>
                </p>
              </div>
            </div>
            <div className="flex items-center gap-1 text-xs text-emerald-600 dark:text-emerald-400">
              <CheckCircle className="w-3 h-3" />
              <span>99.9% target</span>
            </div>
          </div>
        </Card>

        <Card>
          <div className="p-5">
            <div className="flex items-center gap-3 mb-3">
              <div className="p-2 bg-amber-50 dark:bg-amber-900/30 rounded-lg">
                <AlertTriangle className="w-5 h-5 text-amber-600 dark:text-amber-400" />
              </div>
              <div>
                <p className="text-xs text-slate-500 dark:text-slate-400">High Risk</p>
                <p className="text-2xl font-bold text-slate-900 dark:text-white">
                  {predictions.filter(p => p.failure_probability > 70).length}
                </p>
              </div>
            </div>
            <div className="flex items-center gap-1 text-xs text-red-600 dark:text-red-400">
              <AlertTriangle className="w-3 h-3" />
              <span>требуют внимания</span>
            </div>
          </div>
        </Card>
      </div>

      {/* Charts Row */}
      <div className="grid grid-cols-1 lg:grid-cols-2 gap-6">
        {/* MTBF Trend */}
        <Card>
          <div className="p-5">
            <h3 className="text-sm font-semibold text-slate-900 dark:text-white mb-4">
              MTBF Trend (Mean Time Between Failures)
            </h3>
            <ResponsiveContainer width="100%" height={250}>
              <AreaChart data={mtbfTrendData}>
                <defs>
                  <linearGradient id="mtbfGradient" x1="0" y1="0" x2="0" y2="1">
                    <stop offset="5%" stopColor="#3b82f6" stopOpacity={0.3} />
                    <stop offset="95%" stopColor="#3b82f6" stopOpacity={0} />
                  </linearGradient>
                </defs>
                <CartesianGrid strokeDasharray="3 3" stroke="#e2e8f0" />
                <XAxis dataKey="month" tick={{ fontSize: 12 }} />
                <YAxis tick={{ fontSize: 12 }} unit=" ч" />
                <Tooltip />
                <Area type="monotone" dataKey="mtbf" stroke="#3b82f6" fill="url(#mtbfGradient)" strokeWidth={2} dot={{ r: 4 }} />
              </AreaChart>
            </ResponsiveContainer>
          </div>
        </Card>

        {/* MTTR Trend */}
        <Card>
          <div className="p-5">
            <h3 className="text-sm font-semibold text-slate-900 dark:text-white mb-4">
              MTTR Trend (Mean Time To Repair)
            </h3>
            <ResponsiveContainer width="100%" height={250}>
              <AreaChart data={mttrTrendData}>
                <defs>
                  <linearGradient id="mttrGradient" x1="0" y1="0" x2="0" y2="1">
                    <stop offset="5%" stopColor="#22c55e" stopOpacity={0.3} />
                    <stop offset="95%" stopColor="#22c55e" stopOpacity={0} />
                  </linearGradient>
                </defs>
                <CartesianGrid strokeDasharray="3 3" stroke="#e2e8f0" />
                <XAxis dataKey="month" tick={{ fontSize: 12 }} />
                <YAxis tick={{ fontSize: 12 }} unit=" мин" reversed />
                <Tooltip />
                <Area type="monotone" dataKey="mttr" stroke="#22c55e" fill="url(#mttrGradient)" strokeWidth={2} dot={{ r: 4 }} />
              </AreaChart>
            </ResponsiveContainer>
          </div>
        </Card>

        {/* Failure Distribution */}
        <Card>
          <div className="p-5">
            <h3 className="text-sm font-semibold text-slate-900 dark:text-white mb-4">
              Распределение отказов по типу
            </h3>
            <ResponsiveContainer width="100%" height={250}>
              <PieChart>
                <Pie
                  data={failureByTypeData}
                  cx="50%"
                  cy="50%"
                  innerRadius={60}
                  outerRadius={100}
                  paddingAngle={4}
                  dataKey="value"
                  label={({ name, percent }: any) => `${name} ${percent ? (percent * 100).toFixed(0) : 0}%`}
                >
                  {failureByTypeData.map((entry, index) => (
                    <Cell key={`cell-${index}`} fill={entry.color} />
                  ))}
                </Pie>
                <Tooltip />
              </PieChart>
            </ResponsiveContainer>
          </div>
        </Card>

        {/* Predictions Summary */}
        <Card>
          <div className="p-5">
            <h3 className="text-sm font-semibold text-slate-900 dark:text-white mb-4">
              Прогноз отказов по вероятности
            </h3>
            <ResponsiveContainer width="100%" height={250}>
              <BarChart data={[
                { name: 'Высокий (>70%)', count: predictions.filter(p => p.failure_probability > 70).length, fill: '#ef4444' },
                { name: 'Средний (30-70%)', count: predictions.filter(p => p.failure_probability > 30 && p.failure_probability <= 70).length, fill: '#f97316' },
                { name: 'Низкий (<30%)', count: predictions.filter(p => p.failure_probability <= 30).length, fill: '#22c55e' },
              ]}>
                <CartesianGrid strokeDasharray="3 3" stroke="#e2e8f0" />
                <XAxis dataKey="name" tick={{ fontSize: 11 }} />
                <YAxis tick={{ fontSize: 11 }} />
                <Tooltip />
                <Bar dataKey="count" radius={[4, 4, 0, 0]} />
              </BarChart>
            </ResponsiveContainer>
          </div>
        </Card>
      </div>

      {/* Predictions Table */}
      <Card>
        <div className="p-5">
          <h3 className="text-sm font-semibold text-slate-900 dark:text-white mb-4">
            Детальный прогноз по устройствам
          </h3>
          <DataGrid
            data={predictions}
            columns={[
              { header: t('device_id'), key: 'device_id', sortable: true },
              {
                header: t('failure_probability'),
                key: 'failure_probability',
                sortable: true,
                render: (p: Prediction) => (
                  <Badge variant={p.failure_probability > 70 ? 'danger' : p.failure_probability > 30 ? 'warning' : 'success'}>
                    {p.failure_probability}%
                  </Badge>
                ),
              },
              { header: t('explanation'), key: 'explanation' },
              {
                header: 'Осталось часов',
                key: 'expected_remaining_hours',
                sortable: true,
                render: (p: Prediction) => (p as any).expected_remaining_hours ? `${(p as any).expected_remaining_hours} ч` : '—',
              },
            ]}
            keyExtractor={(p) => p.device_id + p.prediction_date}
            emptyMessage={t('no_predictions')}
            variant="striped"
            defaultDensity="standard"
            pageSize={10}
          />
        </div>
      </Card>
    </div>
  );
}
