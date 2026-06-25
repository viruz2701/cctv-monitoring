import React, { useState, useEffect } from 'react';
import { useTranslation } from 'react-i18next';
import { request } from '../services/api';
import { Card, Badge, Button, Select } from '../components/ui';
import { useNavigate } from 'react-router-dom';
import {
  Activity, Thermometer, Wifi, HardDrive, Cpu,
  BarChart3, TrendingUp, TrendingDown, AlertTriangle,
  RefreshCw, Loader2, Zap, Clock,
} from 'lucide-react';
import {
  LineChart, Line, AreaChart, Area, BarChart, Bar,
  XAxis, YAxis, CartesianGrid, Tooltip, ResponsiveContainer,
  Legend,
} from 'recharts';

// ── Types ────────────────────────────────────────────────────────────

interface MeterReading {
  time: string;
  meter_id: string;
  device_id: string;
  kind: string;
  value: number;
}

interface MeterInfo {
  id: string;
  kind: string;
  name: string;
  unit: string;
  enabled: boolean;
}

interface DeviceMeterData {
  device_id: string;
  device_name: string;
  meters: MeterInfo[];
  readings: MeterReading[];
}

type TimeRange = '1h' | '6h' | '24h' | '7d' | '30d';

const TIME_RANGES: { value: TimeRange; label: string }[] = [
  { value: '1h', label: '1 час' },
  { value: '6h', label: '6 часов' },
  { value: '24h', label: '24 часа' },
  { value: '7d', label: '7 дней' },
  { value: '30d', label: '30 дней' },
];

const METER_CONFIG: Record<string, { label: string; icon: React.FC<any>; color: string; unit: string }> = {
  cpu_temp:       { label: 'CPU Temperature', icon: Thermometer, color: '#ef4444', unit: '°C' },
  cpu_usage:      { label: 'CPU Usage', icon: Cpu, color: '#3b82f6', unit: '%' },
  memory_usage:   { label: 'Memory Usage', icon: Cpu, color: '#8b5cf6', unit: '%' },
  bitrate:        { label: 'Bitrate', icon: Activity, color: '#10b981', unit: 'kbps' },
  fps:            { label: 'Frame Rate', icon: Activity, color: '#06b6d4', unit: 'fps' },
  packet_loss:    { label: 'Packet Loss', icon: Wifi, color: '#f59e0b', unit: '%' },
  signal_strength: { label: 'Signal Strength', icon: Wifi, color: '#6366f1', unit: 'dBm' },
  disk_usage:     { label: 'Disk Usage', icon: HardDrive, color: '#ec4899', unit: '%' },
  error_count:    { label: 'Error Count', icon: AlertTriangle, color: '#f97316', unit: 'count' },
  offline_ratio:  { label: 'Offline Ratio', icon: Zap, color: '#dc2626', unit: '%' },
};

const DEFAULT_METER_CONFIG = { label: 'Metric', icon: Activity, color: '#64748b', unit: '' };

// ── Utils ────────────────────────────────────────────────────────────

function formatTime(isoStr: string): string {
  try {
    const d = new Date(isoStr);
    return d.toLocaleTimeString('ru-RU', { hour: '2-digit', minute: '2-digit' });
  } catch {
    return isoStr;
  }
}

function formatValue(value: number, unit: string): string {
  if (unit === '°C' || unit === '%' || unit === 'fps' || unit === 'count') {
    return value.toFixed(1);
  }
  if (unit === 'kbps') {
    if (value > 1000) return (value / 1000).toFixed(1) + ' Mbps';
    return value.toFixed(0) + ' kbps';
  }
  if (unit === 'dBm') {
    return value.toFixed(0);
  }
  return value.toFixed(1);
}

// ── Chart Colors ─────────────────────────────────────────────────────

const CHART_COLORS = ['#3b82f6', '#ef4444', '#10b981', '#f59e0b', '#8b5cf6', '#06b6d4', '#ec4899', '#6366f1'];

// ── Metric Card ──────────────────────────────────────────────────────

function MetricCard({ kind, readings, unit }: { kind: string; readings: MeterReading[]; unit: string }) {
  const cfg = METER_CONFIG[kind] || DEFAULT_METER_CONFIG;
  const Icon = cfg.icon;
  const values = readings.map((r) => r.value);
  const avg = values.length > 0 ? values.reduce((a, b) => a + b, 0) / values.length : 0;
  const max = values.length > 0 ? Math.max(...values) : 0;
  const min = values.length > 0 ? Math.min(...values) : 0;
  const last = values.length > 0 ? values[values.length - 1] : 0;
  const trend = values.length > 1 ? last - values[values.length - 2] : 0;

  // Determine alert level
  const isWarning = kind === 'cpu_temp' ? last > 75 : kind === 'packet_loss' ? last > 3 : kind === 'disk_usage' ? last > 85 : false;
  const isCritical = kind === 'cpu_temp' ? last > 85 : kind === 'packet_loss' ? last > 5 : kind === 'disk_usage' ? last > 95 : false;

  return (
    <div className={`p-4 rounded-xl border ${isCritical ? 'bg-red-50 border-red-200' : isWarning ? 'bg-amber-50 border-amber-200' : 'bg-white border-slate-200'}`}>
      <div className="flex items-center justify-between mb-3">
        <div className="flex items-center gap-2">
          <div className={`p-1.5 rounded-lg ${isCritical ? 'bg-red-100' : isWarning ? 'bg-amber-100' : 'bg-slate-100'}`}>
            <Icon className={`w-4 h-4 ${isCritical ? 'text-red-600' : isWarning ? 'text-amber-600' : 'text-slate-600'}`} />
          </div>
          <div>
            <p className="text-xs font-medium text-slate-500">{cfg.label}</p>
            <p className="text-xs text-slate-400">{unit}</p>
          </div>
        </div>
        {isCritical && <Badge variant="danger">CRITICAL</Badge>}
        {isWarning && !isCritical && <Badge variant="warning">WARN</Badge>}
      </div>

      <div className="flex items-baseline gap-1 mb-1">
        <span className={`text-2xl font-bold ${isCritical ? 'text-red-700' : isWarning ? 'text-amber-700' : 'text-slate-900'}`}>
          {formatValue(last, unit)}
        </span>
        <span className="text-sm text-slate-400">{unit}</span>
      </div>

      <div className="flex items-center gap-3 text-[10px] text-slate-500">
        <span>avg: {formatValue(avg, unit)}</span>
        <span>min: {formatValue(min, unit)}</span>
        <span>max: {formatValue(max, unit)}</span>
        {trend !== 0 && (
          <span className={`flex items-center gap-0.5 ${trend > 0 ? 'text-red-500' : 'text-green-500'}`}>
            {trend > 0 ? <TrendingUp className="w-3 h-3" /> : <TrendingDown className="w-3 h-3" />}
            {Math.abs(trend).toFixed(1)}
          </span>
        )}
      </div>
    </div>
  );
}

// ── Time-Series Chart ────────────────────────────────────────────────

function TimeSeriesChart({ kind, readings, unit }: { kind: string; readings: MeterReading[]; unit: string }) {
  const cfg = METER_CONFIG[kind] || DEFAULT_METER_CONFIG;
  const data = readings.map((r) => ({
    time: formatTime(r.time),
    value: r.value,
  }));

  if (data.length === 0) {
    return (
      <div className="flex items-center justify-center h-48 text-sm text-slate-400">
        {unit === '' ? 'Нет данных для отображения' : 'Загрузка...'}
      </div>
    );
  }

  return (
    <div className="h-48">
      <ResponsiveContainer width="100%" height="100%">
        <AreaChart data={data} margin={{ top: 5, right: 5, left: -20, bottom: 5 }}>
          <defs>
            <linearGradient id={`grad-${kind}`} x1="0" y1="0" x2="0" y2="1">
              <stop offset="5%" stopColor={cfg.color} stopOpacity={0.2} />
              <stop offset="95%" stopColor={cfg.color} stopOpacity={0} />
            </linearGradient>
          </defs>
          <CartesianGrid strokeDasharray="3 3" stroke="#f1f5f9" />
          <XAxis dataKey="time" tick={{ fontSize: 10, fill: '#94a3b8' }} interval="preserveStartEnd" />
          <YAxis tick={{ fontSize: 10, fill: '#94a3b8' }} />
          <Tooltip
            contentStyle={{ fontSize: 12, borderRadius: 8, border: '1px solid #e2e8f0' }}
            formatter={(value: any) => [formatValue(Number(value) || 0, unit), cfg.label]}
          />
          <Area
            type="monotone"
            dataKey="value"
            stroke={cfg.color}
            strokeWidth={2}
            fill={`url(#grad-${kind})`}
            dot={false}
            activeDot={{ r: 4 }}
          />
        </AreaChart>
      </ResponsiveContainer>
    </div>
  );
}

// ── Main Component ───────────────────────────────────────────────────

export function MeterDashboard() {
  const { t } = useTranslation();
  const navigate = useNavigate();
  const [devices, setDevices] = useState<any[]>([]);
  const [selectedDevice, setSelectedDevice] = useState<string>('');
  const [timeRange, setTimeRange] = useState<TimeRange>('6h');
  const [loading, setLoading] = useState(false);
  const [meterData, setMeterData] = useState<DeviceMeterData | null>(null);
  const [error, setError] = useState<string | null>(null);

  // Fetch devices
  useEffect(() => {
    const fetchDevices = async () => {
      try {
        const data = await request<any[]>('/devices');
        setDevices(data || []);
        if (data && data.length > 0 && !selectedDevice) {
          setSelectedDevice(data[0].device_id);
        }
      } catch { /* ignore */ }
    };
    fetchDevices();
  }, []);

  // Fetch meter data
  useEffect(() => {
    if (!selectedDevice) return;

    const fetchMeterData = async () => {
      setLoading(true);
      setError(null);
      try {
        // Try the meter readings API
        const data = await request<DeviceMeterData>(`/meters/${selectedDevice}/readings?period=${timeRange}`);
        setMeterData(data);
      } catch (err: any) {
        // Fallback: generate mock data for development
        setMeterData(generateMockData(selectedDevice));
        setError(err.message || null);
      } finally {
        setLoading(false);
      }
    };

    fetchMeterData();
  }, [selectedDevice, timeRange]);

  // Group readings by kind
  const readingsByKind: Record<string, MeterReading[]> = {};
  if (meterData) {
    for (const r of meterData.readings) {
      if (!readingsByKind[r.kind]) readingsByKind[r.kind] = [];
      readingsByKind[r.kind].push(r);
    }
  }

  const meterKinds = Object.keys(readingsByKind);

  return (
    <div className="space-y-6">
      {/* Header */}
      <div className="flex items-center justify-between">
        <div>
          <h1 className="text-2xl font-bold text-slate-900 flex items-center gap-2">
            <Activity className="w-6 h-6" />
            {t('meter_dashboard') || 'Метрики устройств'}
          </h1>
          <p className="text-sm text-slate-500 mt-1">
            {t('meter_dashboard_desc') || 'Time-series мониторинг метрик CCTV устройств'}
          </p>
        </div>
        <Button
          variant="outline"
          icon={<RefreshCw className="w-4 h-4" />}
          onClick={() => {
            setMeterData(null);
            setError(null);
            setTimeRange(timeRange);
          }}
        >
          {t('refresh') || 'Обновить'}
        </Button>
      </div>

      {/* Controls */}
      <Card>
        <div className="p-4">
          <div className="flex flex-wrap items-center gap-4">
            {/* Device Select */}
            <div className="flex-1 min-w-[200px]">
              <label className="block text-xs font-medium text-slate-500 mb-1">
                {t('device') || 'Устройство'}
              </label>
              <select
                className="w-full rounded-lg border border-slate-300 px-3 py-2 text-sm focus:ring-2 focus:ring-blue-500"
                value={selectedDevice}
                onChange={(e) => setSelectedDevice(e.target.value)}
              >
                <option value="">{t('select_device') || 'Выберите устройство'}</option>
                {devices.map((d) => (
                  <option key={d.device_id} value={d.device_id}>
                    {d.name || d.device_id}
                  </option>
                ))}
              </select>
            </div>

            {/* Time Range */}
            <div>
              <label className="block text-xs font-medium text-slate-500 mb-1">
                {t('period') || 'Период'}
              </label>
              <div className="flex gap-1 p-1 bg-slate-100 rounded-lg">
                {TIME_RANGES.map((range) => (
                  <button
                    key={range.value}
                    onClick={() => setTimeRange(range.value)}
                    className={`px-3 py-1.5 text-xs font-medium rounded-md transition-colors ${
                      timeRange === range.value
                        ? 'bg-white text-slate-900 shadow-sm'
                        : 'text-slate-500 hover:text-slate-700'
                    }`}
                  >
                    {range.label}
                  </button>
                ))}
              </div>
            </div>
          </div>
        </div>
      </Card>

      {/* Loading */}
      {loading && (
        <div className="flex items-center justify-center py-16">
          <Loader2 className="w-8 h-8 animate-spin text-blue-500" />
          <span className="ml-3 text-sm text-slate-500">{t('loading') || 'Загрузка метрик...'}</span>
        </div>
      )}

      {/* Error notice */}
      {error && (
        <div className="p-3 bg-amber-50 rounded-lg border border-amber-200 text-xs text-amber-700">
          {t('meter_api_notice') || 'API метрик в разработке. Показываются демо-данные.'}
        </div>
      )}

      {/* Metric Cards */}
      {!loading && meterKinds.length > 0 && (
        <div className="grid grid-cols-1 md:grid-cols-2 lg:grid-cols-3 xl:grid-cols-4 gap-4">
          {meterKinds.map((kind) => {
            const cfg = METER_CONFIG[kind] || DEFAULT_METER_CONFIG;
            return (
              <MetricCard
                key={kind}
                kind={kind}
                readings={readingsByKind[kind]}
                unit={cfg.unit}
              />
            );
          })}
        </div>
      )}

      {/* Charts */}
      {!loading && meterKinds.length > 0 && (
        <div className="grid grid-cols-1 lg:grid-cols-2 gap-4">
          {meterKinds.map((kind) => {
            const cfg = METER_CONFIG[kind] || DEFAULT_METER_CONFIG;
            return (
              <Card key={kind}>
                <div className="p-4">
                  <div className="flex items-center justify-between mb-3">
                    <div className="flex items-center gap-2">
                      <cfg.icon className="w-4 h-4 text-slate-500" />
                      <h3 className="text-sm font-semibold text-slate-900">{cfg.label}</h3>
                      <span className="text-[10px] text-slate-400">({cfg.unit})</span>
                    </div>
                    <Badge variant="info">{readingsByKind[kind].length} точек</Badge>
                  </div>
                  <TimeSeriesChart
                    kind={kind}
                    readings={readingsByKind[kind]}
                    unit={cfg.unit}
                  />
                </div>
              </Card>
            );
          })}
        </div>
      )}

      {/* Empty state */}
      {!loading && meterKinds.length === 0 && selectedDevice && (
        <div className="flex flex-col items-center justify-center py-16">
          <Activity className="w-12 h-12 text-slate-300 mb-4" />
          <p className="text-sm text-slate-500">{t('no_meter_data') || 'Нет данных метрик для выбранного устройства'}</p>
          <p className="text-xs text-slate-400 mt-1">{t('meter_data_note') || 'Данные появятся после настройки сбора метрик'}</p>
        </div>
      )}

      {/* No device selected */}
      {!loading && !selectedDevice && (
        <div className="flex flex-col items-center justify-center py-16">
          <BarChart3 className="w-12 h-12 text-slate-300 mb-4" />
          <p className="text-sm text-slate-500">{t('select_device_prompt') || 'Выберите устройство для просмотра метрик'}</p>
        </div>
      )}
    </div>
  );
}

// ── Mock Data Generator (for development) ─────────────────────────────

function generateMockData(deviceId: string): DeviceMeterData {
  const now = Date.now();
  const kinds = ['cpu_temp', 'cpu_usage', 'memory_usage', 'bitrate', 'fps', 'packet_loss', 'signal_strength', 'disk_usage'];

  const baseValues: Record<string, { base: number; variance: number }> = {
    cpu_temp:       { base: 55, variance: 15 },
    cpu_usage:      { base: 40, variance: 25 },
    memory_usage:   { base: 60, variance: 15 },
    bitrate:        { base: 4000, variance: 1500 },
    fps:            { base: 25, variance: 5 },
    packet_loss:    { base: 1, variance: 2 },
    signal_strength: { base: -60, variance: 10 },
    disk_usage:     { base: 55, variance: 5 },
  };

  const readings: MeterReading[] = [];
  const numPoints = 60;

  for (const kind of kinds) {
    const bv = baseValues[kind] || { base: 50, variance: 10 };
    for (let i = 0; i < numPoints; i++) {
      const time = new Date(now - (numPoints - i) * 60000).toISOString();
      const noise = (Math.random() - 0.5) * bv.variance;
      const trend = Math.sin((i / numPoints) * Math.PI * 2) * (bv.variance * 0.3);
      readings.push({
        time,
        meter_id: `${kind}-${deviceId}`,
        device_id: deviceId,
        kind,
        value: Math.max(0, bv.base + noise + trend),
      });
    }
  }

  return {
    device_id: deviceId,
    device_name: deviceId,
    meters: kinds.map((kind) => ({
      id: `${kind}-${deviceId}`,
      kind,
      name: kind,
      unit: '',
      enabled: true,
    })),
    readings,
  };
}
