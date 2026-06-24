import React, { useEffect, useState } from 'react';
import { useTranslation } from 'react-i18next';
import { useNavigate } from 'react-router-dom';
import { request } from '../services/api';
import { Card, DataGrid, Badge, StatsCard } from '../components/ui';
import {
  HardDrive, Monitor, Activity, AlertTriangle,
  TrendingUp, DollarSign, Server, Wifi,
} from 'lucide-react';
import { formatCurrency, formatCurrencyCompact } from '../utils/currency';

// ── Types ────────────────────────────────────────────────────────────

interface Device {
  device_id: string;
  name?: string;
  vendor_type?: string;
  device_type?: string;
  status: string;
  location?: string;
  last_seen: string;
}

interface TCOEntry {
  device_id: string;
  device_name: string;
  vendor_type: string;
  device_type: string;
  manufacturer: string;
  total_purchase_cost: number;
  total_labor_cost: number;
  total_parts_cost: number;
  total_downtime_cost: number;
  tco: number;
  total_work_orders: number;
}

interface Prediction {
  device_id: string;
  failure_probability: number;
  explanation?: string;
  prediction_date: string;
}

// ── Component ────────────────────────────────────────────────────────

export const AssetOverview: React.FC = () => {
  const { t } = useTranslation();
  const navigate = useNavigate();
  const [devices, setDevices] = useState<Device[]>([]);
  const [tcoData, setTcoData] = useState<TCOEntry[]>([]);
  const [predictions, setPredictions] = useState<Prediction[]>([]);
  const [loading, setLoading] = useState(true);

  useEffect(() => {
    const fetchAll = async () => {
      setLoading(true);
      try {
        const [devs, tco, preds] = await Promise.all([
          request<Device[]>('/devices').catch(() => []),
          request<TCOEntry[]>('/analytics/tco').catch(() => []),
          request<Prediction[]>('/analytics/predictions').catch(() => []),
        ]);
        setDevices(devs || []);
        setTcoData(tco || []);
        setPredictions(preds || []);
      } catch (err) {
        console.error('Failed to load asset data', err);
      } finally {
        setLoading(false);
      }
    };
    fetchAll();
  }, []);

  // Computed metrics — с Array.isArray guard (OWASP ASVS V5: input validation)
  const safeDevices = Array.isArray(devices) ? devices : [];
  const safePredictions = Array.isArray(predictions) ? predictions : [];
  const safeTco = Array.isArray(tcoData) ? tcoData : [];
  
  const totalDevices = safeDevices.length;
  const onlineDevices = safeDevices.filter(d => d.status === 'online').length;
  const offlineDevices = safeDevices.filter(d => d.status !== 'online').length;
  const highRiskDevices = safePredictions.filter(p => p.failure_probability > 70).length;
  const totalTCO = safeTco.reduce((s, d) => s + d.tco, 0);

  const vendorTypes = [...new Set(safeDevices.map(d => d.vendor_type).filter(Boolean))];
  const deviceTypes = [...new Set(safeDevices.map(d => d.device_type).filter(Boolean))];

  return (
    <div className="space-y-6">
      <h1 className="text-2xl font-bold text-slate-900 dark:text-white flex items-center gap-2">
        <Monitor className="w-6 h-6" />
        {t('asset_overview') || 'Asset Overview'}
      </h1>

      {/* KPI Row */}
      <div className="grid grid-cols-2 md:grid-cols-4 lg:grid-cols-6 gap-3">
        <StatsCard title={t('total_devices') || 'Total Devices'} value={totalDevices} icon={HardDrive}
          iconBgColor="bg-blue-50" iconColor="text-blue-600" />
        <StatsCard title={t('online') || 'Online'} value={onlineDevices} icon={Wifi}
          iconBgColor="bg-emerald-50" iconColor="text-emerald-600" />
        <StatsCard title={t('offline') || 'Offline'} value={offlineDevices} icon={AlertTriangle}
          iconBgColor="bg-red-50" iconColor="text-red-600" />
        <StatsCard title={t('high_risk') || 'High Risk'} value={highRiskDevices} icon={Activity}
          iconBgColor="bg-amber-50" iconColor="text-amber-600" />
        <StatsCard title={t('total_tco') || 'Total TCO'} value={formatCurrencyCompact(totalTCO)} icon={DollarSign}
          iconBgColor="bg-purple-50" iconColor="text-purple-600" />
        <StatsCard title={t('vendor_types') || 'Vendors'} value={vendorTypes.length} icon={Server}
          iconBgColor="bg-slate-50" iconColor="text-slate-600" />
      </div>

      {/* Main Grid */}
      <div className="grid grid-cols-1 lg:grid-cols-2 gap-6">
        {/* Device Status */}
        <Card>
          <div className="p-5">
            <h3 className="text-sm font-semibold text-slate-900 dark:text-white mb-4 flex items-center gap-2">
              <HardDrive className="w-4 h-4" />
              {t('device_status') || 'Device Status'}
            </h3>
            <DataGrid
              data={safeDevices.slice(0, 100)}
              columns={[
                { key: 'name', header: t('name') || 'Name', sortable: true,
                  render: (d: Device) => d.name || d.device_id },
                { key: 'vendor_type', header: t('vendor') || 'Vendor', sortable: true,
                  render: (d: Device) => d.vendor_type || '—' },
                { key: 'device_type', header: t('type') || 'Type', sortable: true,
                  render: (d: Device) => d.device_type || '—' },
                { key: 'status', header: t('status') || 'Status', sortable: true,
                  render: (d: Device) => (
                    <Badge variant={d.status === 'online' ? 'success' : d.status === 'offline' ? 'danger' : 'warning'}>
                      {d.status}
                    </Badge>
                  ),
                },
                { key: 'location', header: t('location') || 'Location', sortable: true,
                  render: (d: Device) => d.location || '—' },
              ]}
              keyExtractor={(d) => d.device_id}
              variant="striped"
              defaultDensity="compact"
              pageSize={10}
              onRowClick={(d) => navigate(`/devices/${d.device_id}`)}
              exportFilename="device-status.csv"
            />
          </div>
        </Card>

        {/* TCO by Device */}
        <Card>
          <div className="p-5">
            <h3 className="text-sm font-semibold text-slate-900 dark:text-white mb-4 flex items-center gap-2">
              <TrendingUp className="w-4 h-4" />
              {t('tco_by_device') || 'TCO by Device'}
            </h3>
            <DataGrid
              data={tcoData}
              columns={[
                { key: 'device_name', header: t('device') || 'Device', sortable: true,
                  render: (t: TCOEntry) => t.device_name || t.device_id },
                { key: 'vendor_type', header: t('vendor') || 'Vendor', sortable: true },
                { key: 'tco', header: 'TCO', sortable: true,
                  render: (t: TCOEntry) => (
                    <span className="font-mono font-medium">{formatCurrency(t.tco)}</span>
                  ),
                },
                { key: 'total_work_orders', header: t('work_orders') || 'WOs', sortable: true },
                { key: 'total_downtime_cost', header: t('downtime_cost') || 'Downtime', sortable: true,
                  render: (t: TCOEntry) => formatCurrency(t.total_downtime_cost, { decimals: 0 }) },
              ]}
              keyExtractor={(t) => t.device_id}
              variant="striped"
              defaultDensity="compact"
              pageSize={10}
              exportFilename="tco-by-device.csv"
            />
          </div>
        </Card>
      </div>

      {/* High Risk Predictions */}
      {predictions.length > 0 && (
        <Card>
          <div className="p-5">
            <h3 className="text-sm font-semibold text-slate-900 dark:text-white mb-4 flex items-center gap-2">
              <AlertTriangle className="w-4 h-4 text-amber-500" />
              {t('high_risk_predictions') || 'High Risk Predictions'}
            </h3>
            <DataGrid
              data={predictions.filter(p => p.failure_probability > 50)}
              columns={[
                { key: 'device_id', header: t('device_id') || 'Device ID', sortable: true },
                { key: 'failure_probability', header: t('probability') || 'Probability', sortable: true,
                  render: (p: Prediction) => (
                    <Badge variant={p.failure_probability > 70 ? 'danger' : 'warning'}>
                      {p.failure_probability}%
                    </Badge>
                  ),
                },
                { key: 'explanation', header: t('explanation') || 'Explanation',
                  render: (p: Prediction) => p.explanation || '—' },
                { key: 'prediction_date', header: t('date') || 'Date', sortable: true },
              ]}
              keyExtractor={(p) => p.device_id + p.prediction_date}
              variant="striped"
              defaultDensity="compact"
              pageSize={10}
              exportFilename="high-risk-predictions.csv"
            />
          </div>
        </Card>
      )}
    </div>
  );
};
