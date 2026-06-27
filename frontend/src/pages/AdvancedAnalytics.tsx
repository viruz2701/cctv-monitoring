// ═══════════════════════════════════════════════════════════════════════
// Advanced Analytics Dashboard (P2-3)
//
// Features:
//   - Predictive Maintenance Widget (устройства, требующие внимания)
//   - Cost Analysis Dashboard (TCO, тренды, топ дорогих устройств)
//   - Vendor Performance Scorecards (MTBF/MTTR рейтинг)
//
// Data sources:
//   - usePredictions() — ML predictions from /api/v1/analytics/predictions
//   - api.getCostData(), api.getCostTrend(), api.getTopExpensiveDevices()
//   - api.getReliabilityData()
//
// Compliance:
//   - OWASP ASVS V2.1.1 (Input validation via Zod — не требуется, read-only)
//   - IEC 62443 SR 3.1 (RBAC — через RoleProtectedRoute)
// ═══════════════════════════════════════════════════════════════════════

import { useState, useEffect, useMemo } from 'react';
import { Card, DataGrid, Badge } from '../components/ui';
import { api } from '../services/api';
import { usePredictions } from '../hooks/useApiQuery';
import type {
  Prediction,
  CostData,
  CostTrend,
  TopExpensiveDevice,
  VendorReliability,
} from '../services/api';
import {
  LineChart, Line, AreaChart, Area, BarChart, Bar,
  XAxis, YAxis, Tooltip, ResponsiveContainer, CartesianGrid,
  PieChart, Pie, Cell, Legend,
} from 'recharts';
import {
  AlertTriangle,
  TrendingUp,
  DollarSign,
  Truck,
  Activity,
  Clock,
  Shield,
  HardDrive,
} from 'lucide-react';
import { SkeletonAdvancedAnalytics } from '../components/layout';

// ═══════════════════════════════════════════════════════════════════════
// Constants
// ═══════════════════════════════════════════════════════════════════════

const RISK_COLORS = {
  high: { bg: 'bg-red-50 dark:bg-red-900/20 border-red-200 dark:border-red-800', icon: '🔴', threshold: 70 },
  medium: { bg: 'bg-orange-50 dark:bg-orange-900/20 border-orange-200 dark:border-orange-800', icon: '🟠', threshold: 50 },
  low: { bg: 'bg-yellow-50 dark:bg-yellow-900/20 border-yellow-200 dark:border-yellow-800', icon: '🟡', threshold: 30 },
  safe: { bg: 'bg-green-50 dark:bg-green-900/20 border-green-200 dark:border-green-800', icon: '🟢', threshold: 0 },
} as const;

function getRiskConfig(probability: number) {
  if (probability > 70) return RISK_COLORS.high;
  if (probability > 50) return RISK_COLORS.medium;
  if (probability > 30) return RISK_COLORS.low;
  return RISK_COLORS.safe;
}

function getRiskBadgeVariant(probability: number): 'danger' | 'warning' | 'success' {
  if (probability > 70) return 'danger';
  if (probability > 30) return 'warning';
  return 'success';
}

function formatDate(dateStr: string): string {
  const d = new Date(dateStr);
  return d.toLocaleDateString('ru-RU', { day: 'numeric', month: 'short', year: 'numeric' });
}

const MONTH_NAMES = ['Янв', 'Фев', 'Мар', 'Апр', 'Май', 'Июн', 'Июл', 'Авг', 'Сен', 'Окт', 'Ноя', 'Дек'];

// ═══════════════════════════════════════════════════════════════════════
// Sub-components
// ═══════════════════════════════════════════════════════════════════════

function PredictiveCard({ prediction }: { prediction: Prediction }) {
  const risk = getRiskConfig(prediction.failure_probability);
  const badgeVar = getRiskBadgeVariant(prediction.failure_probability);

  return (
    <div className={`rounded-xl border p-4 transition-shadow hover:shadow-md ${risk.bg}`}>
      <div className="flex items-start justify-between mb-3">
        <div className="flex items-center gap-2">
          <HardDrive className="w-4 h-4 text-slate-500 dark:text-slate-400" />
          <span className="text-sm font-medium text-slate-900 dark:text-white truncate max-w-[160px]">
            {prediction.device_id}
          </span>
        </div>
        <span className="text-lg" role="img" aria-label={`Risk level: ${badgeVar}`}>
          {risk.icon}
        </span>
      </div>

      <div className="flex items-baseline gap-1 mb-2">
        <span className="text-2xl font-bold text-slate-900 dark:text-white">
          {prediction.failure_probability}%
        </span>
        <span className="text-xs text-slate-500 dark:text-slate-400">вероятность</span>
      </div>

      <Badge variant={badgeVar} size="sm">
        {prediction.failure_probability > 70
          ? 'Критический'
          : prediction.failure_probability > 50
            ? 'Высокий'
            : prediction.failure_probability > 30
              ? 'Средний'
              : 'Низкий'}
      </Badge>

      <p className="mt-2 text-xs text-slate-500 dark:text-slate-400 line-clamp-2">
        {prediction.explanation}
      </p>

      <div className="mt-2 text-[11px] text-slate-400 dark:text-slate-500">
        Прогноз: {formatDate(prediction.prediction_date)}
      </div>
    </div>
  );
}

// ═══════════════════════════════════════════════════════════════════════
// Vendor Scorecard Row
// ═══════════════════════════════════════════════════════════════════════

function VendorScore({ vendor }: { vendor: VendorReliability }) {
  const scoreColor = vendor.score >= 80
    ? 'text-emerald-600 dark:text-emerald-400'
    : vendor.score >= 60
      ? 'text-amber-600 dark:text-amber-400'
      : 'text-red-600 dark:text-red-400';

  const scoreBg = vendor.score >= 80
    ? 'bg-emerald-100 dark:bg-emerald-900/30'
    : vendor.score >= 60
      ? 'bg-amber-100 dark:bg-amber-900/30'
      : 'bg-red-100 dark:bg-red-900/30';

  return (
    <div className="flex items-center justify-between p-3 bg-white dark:bg-slate-800 rounded-lg border border-slate-200 dark:border-slate-700">
      <div className="flex items-center gap-3 min-w-0">
        <Truck className="w-5 h-5 text-slate-400 flex-shrink-0" />
        <div className="min-w-0">
          <p className="text-sm font-medium text-slate-900 dark:text-white truncate">
            {vendor.vendor}
          </p>
          <p className="text-xs text-slate-500 dark:text-slate-400">
            {vendor.device_count} devices
          </p>
        </div>
      </div>

      <div className="flex items-center gap-4 flex-shrink-0">
        <div className="text-right">
          <p className="text-xs text-slate-400 dark:text-slate-500">MTBF</p>
          <p className="text-sm font-medium text-slate-900 dark:text-white">
            {vendor.mtbf_hours.toLocaleString()} <span className="text-xs text-slate-400">ч</span>
          </p>
        </div>
        <div className="text-right">
          <p className="text-xs text-slate-400 dark:text-slate-500">MTTR</p>
          <p className="text-sm font-medium text-slate-900 dark:text-white">
            {vendor.mttr_minutes} <span className="text-xs text-slate-400">мин</span>
          </p>
        </div>
        <div className={`w-10 h-10 rounded-full flex items-center justify-center ${scoreBg}`}>
          <span className={`text-sm font-bold ${scoreColor}`}>{vendor.score}</span>
        </div>
      </div>
    </div>
  );
}

// ═══════════════════════════════════════════════════════════════════════
// Cost Bar Chart (TCO by site/device type)
// ═══════════════════════════════════════════════════════════════════════

function TCOBarChart({ data }: { data: CostData[] }) {
  const chartData = useMemo(() => {
    const grouped = new Map<string, CostData>();
    for (const d of data) {
      const key = d.site_name || d.site_id;
      const existing = grouped.get(key);
      if (existing) {
        existing.total_cost += d.total_cost;
        existing.maintenance_cost += d.maintenance_cost;
        existing.energy_cost += d.energy_cost;
        existing.labor_cost += d.labor_cost;
        existing.spare_parts_cost += d.spare_parts_cost;
      } else {
        grouped.set(key, { ...d });
      }
    }
    return Array.from(grouped.values())
      .sort((a, b) => b.total_cost - a.total_cost)
      .slice(0, 10)
      .map(d => ({
        name: d.site_name || d.site_id,
        total: Math.round(d.total_cost / 1000),
        maintenance: Math.round(d.maintenance_cost / 1000),
        energy: Math.round(d.energy_cost / 1000),
      }));
  }, [data]);

  if (chartData.length === 0) return null;

  return (
    <ResponsiveContainer width="100%" height={280}>
      <BarChart data={chartData} barGap={2}>
        <CartesianGrid strokeDasharray="3 3" stroke="#e2e8f0" />
        <XAxis dataKey="name" tick={{ fontSize: 11 }} angle={-20} textAnchor="end" height={50} />
        <YAxis tick={{ fontSize: 11 }} unit="k" />
        <Tooltip formatter={(value: any) => [`$${Number(value).toLocaleString()}k`, undefined]} />
        <Legend />
        <Bar dataKey="maintenance" name="Maintenance" fill="#3b82f6" radius={[2, 2, 0, 0]} stackId="a" />
        <Bar dataKey="energy" name="Energy" fill="#f97316" radius={[2, 2, 0, 0]} stackId="a" />
      </BarChart>
    </ResponsiveContainer>
  );
}

// ═══════════════════════════════════════════════════════════════════════
// Cost Trend Area Chart
// ═══════════════════════════════════════════════════════════════════════

function CostTrendChart({ data }: { data: CostTrend[] }) {
  if (data.length === 0) return null;

  return (
    <ResponsiveContainer width="100%" height={280}>
      <AreaChart data={data}>
        <defs>
          <linearGradient id="costTrendGradient" x1="0" y1="0" x2="0" y2="1">
            <stop offset="5%" stopColor="#3b82f6" stopOpacity={0.3} />
            <stop offset="95%" stopColor="#3b82f6" stopOpacity={0} />
          </linearGradient>
        </defs>
        <CartesianGrid strokeDasharray="3 3" stroke="#e2e8f0" />
        <XAxis dataKey="month" tick={{ fontSize: 11 }} />
        <YAxis tick={{ fontSize: 11 }} unit="$" />
        <Tooltip formatter={(value: any) => [`$${Number(value).toLocaleString()}`, undefined]} />
        <Area
          type="monotone"
          dataKey="total_cost"
          stroke="#3b82f6"
          fill="url(#costTrendGradient)"
          strokeWidth={2}
          dot={{ r: 3 }}
          name="Total Cost"
        />
      </AreaChart>
    </ResponsiveContainer>
  );
}

// ═══════════════════════════════════════════════════════════════════════
// Risk Distribution Pie Chart
// ═══════════════════════════════════════════════════════════════════════

function RiskPieChart({ predictions }: { predictions: Prediction[] }) {
  const data = useMemo(() => [
    { name: 'Высокий (>70%)', value: predictions.filter(p => p.failure_probability > 70).length, color: '#ef4444' },
    { name: 'Средний (50-70%)', value: predictions.filter(p => p.failure_probability > 50 && p.failure_probability <= 70).length, color: '#f97316' },
    { name: 'Умеренный (30-50%)', value: predictions.filter(p => p.failure_probability > 30 && p.failure_probability <= 50).length, color: '#eab308' },
    { name: 'Низкий (<30%)', value: predictions.filter(p => p.failure_probability <= 30).length, color: '#22c55e' },
  ].filter(d => d.value > 0), [predictions]);

  if (data.length === 0) return null;

  return (
    <ResponsiveContainer width="100%" height={220}>
      <PieChart>
        <Pie
          data={data}
          cx="50%"
          cy="50%"
          innerRadius={50}
          outerRadius={90}
          paddingAngle={3}
          dataKey="value"
          label={({ name, percent }: any) => `${name} ${(percent * 100).toFixed(0)}%`}
        >
          {data.map((entry, idx) => (
            <Cell key={`cell-${idx}`} fill={entry.color} />
          ))}
        </Pie>
        <Tooltip />
      </PieChart>
    </ResponsiveContainer>
  );
}

// ═══════════════════════════════════════════════════════════════════════
// Main Component
// ═══════════════════════════════════════════════════════════════════════

export function AdvancedAnalytics() {
  // ── State ──────────────────────────────────────────────────────────
  const [costData, setCostData] = useState<CostData[]>([]);
  const [costTrend, setCostTrend] = useState<CostTrend[]>([]);
  const [topDevices, setTopDevices] = useState<TopExpensiveDevice[]>([]);
  const [reliability, setReliability] = useState<VendorReliability[]>([]);
  const [overallMtbf, setOverallMtbf] = useState(0);
  const [overallMttr, setOverallMttr] = useState(0);
  const [loadingCost, setLoadingCost] = useState(true);
  const [loadingReliability, setLoadingReliability] = useState(true);
  const [costError, setCostError] = useState('');
  const [reliabilityError, setReliabilityError] = useState('');

  // ── Predictions via React Query ────────────────────────────────────
  const { data: predictions = [], isLoading: loadingPredictions, error: predictionsError } = usePredictions();

  // ── Fetch Cost Data ────────────────────────────────────────────────
  useEffect(() => {
    let cancelled = false;
    async function fetchCostData() {
      try {
        const [cost, trend, top] = await Promise.all([
          api.getCostData({ months: 6 }),
          api.getCostTrend(6),
          api.getTopExpensiveDevices(10),
        ]);
        if (!cancelled) {
          setCostData(cost);
          setCostTrend(trend);
          setTopDevices(top);
        }
      } catch (err: any) {
        if (!cancelled) setCostError(err.message || 'Failed to load cost data');
      } finally {
        if (!cancelled) setLoadingCost(false);
      }
    }
    fetchCostData();
    return () => { cancelled = true; };
  }, []);

  // ── Fetch Reliability Data ─────────────────────────────────────────
  useEffect(() => {
    let cancelled = false;
    async function fetchReliability() {
      try {
        const data = await api.getReliabilityData();
        if (!cancelled) {
          setReliability(data.vendors || []);
          setOverallMtbf(data.overall_mtbf);
          setOverallMttr(data.overall_mttr);
        }
      } catch (err: any) {
        if (!cancelled) setReliabilityError(err.message || 'Failed to load reliability data');
      } finally {
        if (!cancelled) setLoadingReliability(false);
      }
    }
    fetchReliability();
    return () => { cancelled = true; };
  }, []);

  // ── Devices needing attention (next 7 days) ────────────────────────
  const attentionDevices = useMemo(() => {
    const sevenDaysFromNow = new Date();
    sevenDaysFromNow.setDate(sevenDaysFromNow.getDate() + 7);
    return predictions
      .filter(p => new Date(p.prediction_date) <= sevenDaysFromNow)
      .sort((a, b) => b.failure_probability - a.failure_probability);
  }, [predictions]);

  // ── Stats ──────────────────────────────────────────────────────────
  const totalPredicted = predictions.length;
  const highRiskCount = predictions.filter(p => p.failure_probability > 70).length;
  const topDeviceCost = topDevices.length > 0
    ? Math.max(...topDevices.map(d => d.total_cost))
    : 0;

  const errorMessage = predictionsError
    ? (predictionsError as Error).message || 'Unknown prediction error'
    : costError || reliabilityError || '';

  // ══════════════════════════════════════════════════════════════════
  // Render
  // ══════════════════════════════════════════════════════════════════

  const isLoading = loadingPredictions || loadingCost || loadingReliability;

  if (isLoading) {
    return <SkeletonAdvancedAnalytics />;
  }

  if (errorMessage) {
    return (
      <div className="p-8 text-center">
        <AlertTriangle className="w-12 h-12 mx-auto mb-4 text-red-400" />
        <h2 className="text-lg font-semibold text-slate-900 dark:text-white mb-2">
          Ошибка загрузки данных
        </h2>
        <p className="text-sm text-slate-500 dark:text-slate-400">{errorMessage}</p>
      </div>
    );
  }

  return (
    <div className="space-y-6">
      {/* Header */}
      <div className="flex items-center justify-between">
        <div>
          <h1 className="text-2xl font-bold text-slate-900 dark:text-white">
            Advanced Analytics
          </h1>
          <p className="text-sm text-slate-500 dark:text-slate-400 mt-1">
            Predictive maintenance, cost analysis & vendor performance
          </p>
        </div>
      </div>

      {/* ══════════════════════════════════════════════════════════════
          KPI Cards
          ══════════════════════════════════════════════════════════════ */}
      <div className="grid grid-cols-1 md:grid-cols-4 gap-4">
        <Card>
          <div className="p-5">
            <div className="flex items-center gap-3 mb-3">
              <div className="p-2 bg-red-50 dark:bg-red-900/30 rounded-lg">
                <AlertTriangle className="w-5 h-5 text-red-600 dark:text-red-400" />
              </div>
              <div>
                <p className="text-xs text-slate-500 dark:text-slate-400">High Risk</p>
                <p className="text-2xl font-bold text-slate-900 dark:text-white">{highRiskCount}</p>
              </div>
            </div>
          </div>
        </Card>

        <Card>
          <div className="p-5">
            <div className="flex items-center gap-3 mb-3">
              <div className="p-2 bg-blue-50 dark:bg-blue-900/30 rounded-lg">
                <Activity className="w-5 h-5 text-blue-600 dark:text-blue-400" />
              </div>
              <div>
                <p className="text-xs text-slate-500 dark:text-slate-400">Predictions</p>
                <p className="text-2xl font-bold text-slate-900 dark:text-white">{totalPredicted}</p>
              </div>
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
                <p className="text-xs text-slate-500 dark:text-slate-400">MTBF</p>
                <p className="text-2xl font-bold text-slate-900 dark:text-white">
                  {overallMtbf.toLocaleString()} <span className="text-sm font-normal text-slate-500">ч</span>
                </p>
              </div>
            </div>
          </div>
        </Card>

        <Card>
          <div className="p-5">
            <div className="flex items-center gap-3 mb-3">
              <div className="p-2 bg-purple-50 dark:bg-purple-900/30 rounded-lg">
                <DollarSign className="w-5 h-5 text-purple-600 dark:text-purple-400" />
              </div>
              <div>
                <p className="text-xs text-slate-500 dark:text-slate-400">Top Device Cost</p>
                <p className="text-2xl font-bold text-slate-900 dark:text-white">${topDeviceCost.toLocaleString()}</p>
              </div>
            </div>
          </div>
        </Card>
      </div>

      {/* ══════════════════════════════════════════════════════════════
          Section 1: Predictive Maintenance Widget
          ══════════════════════════════════════════════════════════════ */}
      <Card>
        <div className="p-5">
          <div className="flex items-center justify-between mb-4">
            <div>
              <h2 className="text-lg font-semibold text-slate-900 dark:text-white flex items-center gap-2">
                <TrendingUp className="w-5 h-5 text-blue-500" />
                Predictive Maintenance
              </h2>
              <p className="text-xs text-slate-500 dark:text-slate-400 mt-0.5">
                Устройства, требующие внимания в следующие 7 дней
              </p>
            </div>
            <Badge variant={highRiskCount > 0 ? 'danger' : 'success'} size="sm">
              {attentionDevices.length} devices
            </Badge>
          </div>

          {attentionDevices.length === 0 ? (
            <div className="text-center py-8">
              <Shield className="w-10 h-10 mx-auto mb-3 text-emerald-400" />
              <p className="text-sm font-medium text-slate-700 dark:text-slate-300">
                Все устройства в норме
              </p>
              <p className="text-xs text-slate-500 dark:text-slate-400 mt-1">
                Нет прогнозов отказов на ближайшие 7 дней
              </p>
            </div>
          ) : (
            <>
              <div className="grid grid-cols-1 sm:grid-cols-2 lg:grid-cols-3 xl:grid-cols-4 gap-4">
                {attentionDevices.slice(0, 8).map(p => (
                  <PredictiveCard key={p.device_id + p.prediction_date} prediction={p} />
                ))}
              </div>

              {attentionDevices.length > 8 && (
                <div className="mt-3 text-center">
                  <button
                    onClick={() => {
                      const el = document.getElementById('predictions-full-list');
                      if (el) el.classList.toggle('hidden');
                    }}
                    className="text-xs text-blue-600 dark:text-blue-400 hover:underline"
                  >
                    Показать все {attentionDevices.length} устройств
                  </button>
                </div>
              )}

              {/* Hidden full list */}
              <div id="predictions-full-list" className="hidden mt-4">
                <div className="grid grid-cols-1 sm:grid-cols-2 lg:grid-cols-3 xl:grid-cols-4 gap-4">
                  {attentionDevices.slice(8).map(p => (
                    <PredictiveCard key={p.device_id + p.prediction_date} prediction={p} />
                  ))}
                </div>
              </div>
            </>
          )}
        </div>
      </Card>

      {/* ══════════════════════════════════════════════════════════════
          Section 2: Cost Analysis Dashboard
          ══════════════════════════════════════════════════════════════ */}
      <Card>
        <div className="p-5">
          <div className="flex items-center gap-2 mb-4">
            <DollarSign className="w-5 h-5 text-emerald-500" />
            <h2 className="text-lg font-semibold text-slate-900 dark:text-white">
              Cost Analysis Dashboard
            </h2>
          </div>

          {costError ? (
            <div className="text-center py-6 text-sm text-red-500">{costError}</div>
          ) : (
            <div className="grid grid-cols-1 lg:grid-cols-2 gap-6">
              {/* TCO Bar Chart */}
              <div>
                <h3 className="text-sm font-semibold text-slate-900 dark:text-white mb-3">
                  TCO по сайтам (k$)
                </h3>
                <TCOBarChart data={costData} />
              </div>

              {/* Cost Trend */}
              <div>
                <h3 className="text-sm font-semibold text-slate-900 dark:text-white mb-3">
                  Cost Trend (6 months)
                </h3>
                <CostTrendChart data={costTrend} />
              </div>
            </div>
          )}

          {/* Top 10 Most Expensive Devices */}
          {topDevices.length > 0 && (
            <div className="mt-6">
              <h3 className="text-sm font-semibold text-slate-900 dark:text-white mb-3">
                Top 10 Most Expensive Devices
              </h3>
              <DataGrid
                data={topDevices}
                columns={[
                  { header: 'Device', key: 'device_name', sortable: true },
                  { header: 'Site', key: 'site_name', sortable: true },
                  {
                    header: 'Total Cost',
                    key: 'total_cost',
                    sortable: true,
                    render: (d: TopExpensiveDevice) => (
                      <span className="font-medium text-slate-900 dark:text-white">
                        ${d.total_cost.toLocaleString()}
                      </span>
                    ),
                  },
                  {
                    header: 'Maintenance',
                    key: 'breakdown',
                    render: (d: TopExpensiveDevice) => (
                      <span className="text-xs text-slate-500">
                        ${d.breakdown.maintenance.toLocaleString()}
                      </span>
                    ),
                  },
                  {
                    header: 'Energy',
                    key: 'breakdown',
                    render: (d: TopExpensiveDevice) => (
                      <span className="text-xs text-slate-500">
                        ${d.breakdown.energy.toLocaleString()}
                      </span>
                    ),
                  },
                ]}
                keyExtractor={(d) => d.device_id}
                variant="striped"
                defaultDensity="standard"
                pageSize={5}
                emptyMessage="No cost data available"
              />
            </div>
          )}
        </div>
      </Card>

      {/* ══════════════════════════════════════════════════════════════
          Section 3: Vendor Performance Scorecards + Risk Distribution
          ══════════════════════════════════════════════════════════════ */}
      <div className="grid grid-cols-1 lg:grid-cols-3 gap-6">
        {/* Vendor Performance — 2 cols */}
        <div className="lg:col-span-2">
          <Card>
            <div className="p-5">
              <div className="flex items-center justify-between mb-4">
                <div className="flex items-center gap-2">
                  <Truck className="w-5 h-5 text-indigo-500" />
                  <h2 className="text-lg font-semibold text-slate-900 dark:text-white">
                    Vendor Performance Scorecards
                  </h2>
                </div>
                <Badge variant="primary" size="sm">
                  MTBF {overallMtbf.toLocaleString()}h / MTTR {overallMttr}min
                </Badge>
              </div>

              {reliabilityError ? (
                <div className="text-center py-6 text-sm text-red-500">{reliabilityError}</div>
              ) : reliability.length === 0 ? (
                <div className="text-center py-8">
                  <Truck className="w-10 h-10 mx-auto mb-3 text-slate-300 dark:text-slate-600" />
                  <p className="text-sm text-slate-500 dark:text-slate-400">
                    No vendor reliability data available
                  </p>
                </div>
              ) : (
                <div className="space-y-2">
                  {reliability.map(v => (
                    <VendorScore key={v.vendor} vendor={v} />
                  ))}
                </div>
              )}
            </div>
          </Card>
        </div>

        {/* Risk Distribution Pie — 1 col */}
        <div>
          <Card>
            <div className="p-5">
              <div className="flex items-center gap-2 mb-4">
                <Shield className="w-5 h-5 text-amber-500" />
                <h2 className="text-lg font-semibold text-slate-900 dark:text-white">
                  Risk Distribution
                </h2>
              </div>
              <RiskPieChart predictions={predictions} />
            </div>
          </Card>
        </div>
      </div>

      {/* ══════════════════════════════════════════════════════════════
          Vendor DataGrid (full details)
          ══════════════════════════════════════════════════════════════ */}
      {reliability.length > 0 && (
        <Card>
          <div className="p-5">
            <h3 className="text-sm font-semibold text-slate-900 dark:text-white mb-4">
              Vendor Reliability Details
            </h3>
            <DataGrid
              data={reliability}
              columns={[
                { header: 'Vendor', key: 'vendor', sortable: true },
                { header: 'Device Count', key: 'device_count', sortable: true },
                {
                  header: 'MTBF (hours)',
                  key: 'mtbf_hours',
                  sortable: true,
                  render: (v: VendorReliability) => (
                    <span className="font-medium">{v.mtbf_hours.toLocaleString()} ч</span>
                  ),
                },
                {
                  header: 'MTTR (min)',
                  key: 'mttr_minutes',
                  sortable: true,
                  render: (v: VendorReliability) => (
                    <span className="font-medium">{v.mttr_minutes} мин</span>
                  ),
                },
                {
                  header: 'Failure Rate',
                  key: 'failure_rate',
                  sortable: true,
                  render: (v: VendorReliability) => (
                    <span>{(v.failure_rate * 100).toFixed(1)}%</span>
                  ),
                },
                {
                  header: 'Score',
                  key: 'score',
                  sortable: true,
                  render: (v: VendorReliability) => (
                    <Badge variant={v.score >= 80 ? 'success' : v.score >= 60 ? 'warning' : 'danger'}>
                      {v.score}
                    </Badge>
                  ),
                },
              ]}
              keyExtractor={(v) => v.vendor}
              variant="striped"
              defaultDensity="standard"
              pageSize={10}
              emptyMessage="No vendor data"
            />
          </div>
        </Card>
      )}
    </div>
  );
}
