import React, { useEffect, useState } from 'react';
import { useTranslation } from 'react-i18next';
import { request } from '../services/api';
import { Card, DataGrid, Badge, StatsCard, Button } from '../components/ui';
import {
  DollarSign, Briefcase, Wrench, Truck, TrendingUp,
  PieChart, BarChart3, Clock, Download, AlertTriangle,
} from 'lucide-react';
import jsPDF from 'jspdf';
import 'jspdf-autotable';

// ── Types ────────────────────────────────────────────────────────────

interface WorkOrderCostSummary {
  total_work_orders: number;
  total_labor_cost: number;
  total_parts_cost: number;
  total_additional_cost: number;
  total_cost: number;
  avg_cost_per_order: number;
  currency: string;
}

interface WorkOrderCostBreakdown {
  category: string;
  amount: number;
  count: number;
  percent: number;
}

interface CostResponse {
  summary: WorkOrderCostSummary;
  breakdown: WorkOrderCostBreakdown[];
}

interface TCOPerDevice {
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
  total_downtime_events: number;
}

// ── Page Component ───────────────────────────────────────────────────

export const TotalCostDashboard: React.FC = () => {
  const { t } = useTranslation();
  const [data, setData] = useState<CostResponse | null>(null);
  const [tcoData, setTcoData] = useState<TCOPerDevice[]>([]);
  const [loading, setLoading] = useState(false);
  const [tcoLoading, setTcoLoading] = useState(false);

  useEffect(() => {
    fetchCostData();
    fetchTCOData();
  }, []);

  const fetchCostData = async () => {
    setLoading(true);
    try {
      const result = await request<CostResponse>('/analytics/wo-costs');
      setData(result);
    } catch (err) {
      console.error('Failed to fetch cost data', err);
    } finally {
      setLoading(false);
    }
  };

  const fetchTCOData = async () => {
    setTcoLoading(true);
    try {
      const result = await request<TCOPerDevice[]>('/analytics/tco?limit=50');
      setTcoData(result);
    } catch (err) {
      console.error('Failed to fetch TCO data', err);
    } finally {
      setTcoLoading(false);
    }
  };

  // ── PDF Export (BIZ-01) ──────────────────────────────────────────

  const exportPDF = () => {
    const doc = new jsPDF();
    const pageWidth = doc.internal.pageSize.getWidth();

    // Title
    doc.setFontSize(18);
    doc.text('TCO & Downtime Cost Report', pageWidth / 2, 20, { align: 'center' });
    doc.setFontSize(10);
    doc.text(`Generated: ${new Date().toLocaleDateString('ru-RU')}`, pageWidth / 2, 28, { align: 'center' });

    // Work Order Costs Summary
    if (data?.summary) {
      doc.setFontSize(14);
      doc.text('Work Order Cost Summary', 14, 42);
      doc.setFontSize(10);
      const summaryLines = [
        [`Total Cost:`, `$${data.summary.total_cost.toLocaleString('en-US', { minimumFractionDigits: 2 })}`],
        [`Labor Cost:`, `$${data.summary.total_labor_cost.toLocaleString('en-US', { minimumFractionDigits: 2 })}`],
        [`Parts Cost:`, `$${data.summary.total_parts_cost.toLocaleString('en-US', { minimumFractionDigits: 2 })}`],
        [`Additional Cost:`, `$${data.summary.total_additional_cost.toLocaleString('en-US', { minimumFractionDigits: 2 })}`],
        [`Total Work Orders:`, `${data.summary.total_work_orders}`],
        [`Avg Cost/Order:`, `$${data.summary.avg_cost_per_order.toLocaleString('en-US', { minimumFractionDigits: 2 })}`],
      ];
      (doc as any).autoTable({
        startY: 46,
        head: [['Metric', 'Value']],
        body: summaryLines,
        theme: 'striped',
        styles: { fontSize: 9 },
      });
    }

    // TCO by Device
    if (tcoData.length > 0) {
      const yPos = (doc as any).lastAutoTable?.finalY || 80;
      doc.setFontSize(14);
      doc.text('TCO by Device (Top 20)', 14, yPos + 14);
      const tcoRows = tcoData.slice(0, 20).map((d) => [
        d.device_name,
        d.device_type,
        `$${d.total_downtime_cost.toFixed(2)}`,
        `$${d.tco.toFixed(2)}`,
        `${d.total_downtime_events}`,
      ]);
      (doc as any).autoTable({
        startY: yPos + 18,
        head: [['Device', 'Type', 'Downtime Cost', 'TCO', 'Events']],
        body: tcoRows,
        theme: 'striped',
        styles: { fontSize: 8 },
      });
    }

    // Summary
    const finalY = (doc as any).lastAutoTable?.finalY || 100;
    const totalDowntimeCost = tcoData.reduce((sum, d) => sum + d.total_downtime_cost, 0);
    const totalTCO = tcoData.reduce((sum, d) => sum + d.tco, 0);
    doc.setFontSize(10);
    doc.text(`Total Downtime Cost: $${totalDowntimeCost.toLocaleString('en-US', { minimumFractionDigits: 2 })}`, 14, finalY + 10);
    doc.text(`Total TCO (all devices): $${totalTCO.toLocaleString('en-US', { minimumFractionDigits: 2 })}`, 14, finalY + 18);

    doc.save('tco-downtime-report.pdf');
  };

  const summary = data?.summary;
  const breakdown = data?.breakdown || [];

  // ── TCO Stats ──────────────────────────────────────────────────
  const totalDowntimeCost = tcoData.reduce((sum, d) => sum + d.total_downtime_cost, 0);
  const totalTCO = tcoData.reduce((sum, d) => sum + d.tco, 0);
  const topDowntimeDevices = [...tcoData]
    .sort((a, b) => b.total_downtime_cost - a.total_downtime_cost)
    .slice(0, 5);

  const categoryConfig: Record<string, { label: string; icon: React.FC<any>; color: string; bg: string }> = {
    labor: { label: t('labor_cost'), icon: Briefcase, color: 'text-blue-600', bg: 'bg-blue-50' },
    parts: { label: t('parts_cost'), icon: Wrench, color: 'text-emerald-600', bg: 'bg-emerald-50' },
    additional: { label: t('additional_cost'), icon: Truck, color: 'text-amber-600', bg: 'bg-amber-50' },
  };

  return (
    <div className="space-y-6">
      {/* Header */}
      <div className="flex items-center justify-between">
        <h1 className="text-2xl font-bold text-slate-900 flex items-center gap-2">
          <DollarSign className="w-6 h-6" />
          {t('total_cost_dashboard') || 'TCO & Downtime Cost'}
        </h1>
        <Button variant="outline" icon={<Download className="w-4 h-4" />} onClick={exportPDF}>
          {t('export_pdf') || 'PDF Report'}
        </Button>
      </div>

      {/* KPI Cards */}
      <div className="grid grid-cols-1 md:grid-cols-4 gap-4">
        <StatsCard
          title={t('total_cost') || 'Total Cost'}
          value={summary ? `$${summary.total_cost.toLocaleString('en-US', { minimumFractionDigits: 2 })}` : '—'}
          icon={DollarSign}
          iconBgColor="bg-slate-50"
          iconColor="text-slate-600"
        />
        <StatsCard
          title={t('labor_cost') || 'Labor'}
          value={summary ? `$${summary.total_labor_cost.toLocaleString('en-US', { minimumFractionDigits: 2 })}` : '—'}
          icon={Briefcase}
          iconBgColor="bg-blue-50"
          iconColor="text-blue-600"
        />
        <StatsCard
          title={t('parts_cost') || 'Parts'}
          value={summary ? `$${summary.total_parts_cost.toLocaleString('en-US', { minimumFractionDigits: 2 })}` : '—'}
          icon={Wrench}
          iconBgColor="bg-emerald-50"
          iconColor="text-emerald-600"
        />
        <StatsCard
          title={t('additional_cost') || 'Additional'}
          value={summary ? `$${summary.total_additional_cost.toLocaleString('en-US', { minimumFractionDigits: 2 })}` : '—'}
          icon={Truck}
          iconBgColor="bg-amber-50"
          iconColor="text-amber-600"
        />
      </div>

      {/* BIZ-01: Downtime Cost Section */}
      <Card>
        <div className="p-5">
          <h3 className="text-sm font-semibold text-slate-900 mb-4 flex items-center gap-2">
            <Clock className="w-4 h-4 text-red-500" />
            {t('downtime_costs') || '💰 Стоимость простоев'}
          </h3>

          <div className="grid grid-cols-1 md:grid-cols-3 gap-4 mb-4">
            <div className="p-4 bg-red-50 rounded-lg border border-red-200">
              <p className="text-xs text-red-600 font-medium">{t('total_downtime_cost') || 'Общая стоимость простоев'}</p>
              <p className="text-2xl font-bold text-red-700 mt-1">
                ${totalDowntimeCost.toLocaleString('en-US', { minimumFractionDigits: 2 })}
              </p>
            </div>
            <div className="p-4 bg-purple-50 rounded-lg border border-purple-200">
              <p className="text-xs text-purple-600 font-medium">{t('total_tco') || 'Общий TCO (все устройства)'}</p>
              <p className="text-2xl font-bold text-purple-700 mt-1">
                ${totalTCO.toLocaleString('en-US', { minimumFractionDigits: 2 })}
              </p>
            </div>
            <div className="p-4 bg-amber-50 rounded-lg border border-amber-200">
              <p className="text-xs text-amber-600 font-medium">{t('downtime_percent') || '% downtime от TCO'}</p>
              <p className="text-2xl font-bold text-amber-700 mt-1">
                {totalTCO > 0 ? ((totalDowntimeCost / totalTCO) * 100).toFixed(1) : '0.0'}%
              </p>
            </div>
          </div>

          {/* Top 5 by Downtime Cost */}
          <h4 className="text-xs font-semibold text-slate-700 mb-2 uppercase tracking-wide">
            {t('top_downtime_devices') || 'Топ-5 устройств по стоимости простоев'}
          </h4>
          <div className="space-y-2">
            {topDowntimeDevices.map((device, idx) => (
              <div key={device.device_id} className="flex items-center justify-between p-3 bg-slate-50 rounded-lg">
                <div className="flex items-center gap-3">
                  <span className="text-sm font-bold text-slate-400 w-5">#{idx + 1}</span>
                  <div>
                    <p className="text-sm font-medium text-slate-900">{device.device_name}</p>
                    <p className="text-xs text-slate-500">{device.device_type} · {device.vendor_type}</p>
                  </div>
                </div>
                <div className="text-right">
                  <p className="text-sm font-bold text-red-600">
                    ${device.total_downtime_cost.toLocaleString('en-US', { minimumFractionDigits: 2 })}
                  </p>
                  <p className="text-xs text-slate-500">{device.total_downtime_events} событий</p>
                </div>
              </div>
            ))}
            {topDowntimeDevices.length === 0 && (
              <p className="text-sm text-slate-500 text-center py-4">{t('no_downtime_data') || 'Нет данных по простоям'}</p>
            )}
          </div>
        </div>
      </Card>

      {/* Secondary Metrics */}
      {summary && (
        <div className="grid grid-cols-1 md:grid-cols-3 gap-4">
          <Card>
            <div className="p-4">
              <div className="flex items-center gap-3">
                <div className="p-2.5 bg-purple-50 rounded-xl">
                  <BarChart3 className="w-5 h-5 text-purple-600" />
                </div>
                <div>
                  <p className="text-xs text-slate-500">{t('total_work_orders') || 'Total WO'}</p>
                  <p className="text-xl font-bold text-slate-900">{summary.total_work_orders}</p>
                </div>
              </div>
            </div>
          </Card>
          <Card>
            <div className="p-4">
              <div className="flex items-center gap-3">
                <div className="p-2.5 bg-indigo-50 rounded-xl">
                  <TrendingUp className="w-5 h-5 text-indigo-600" />
                </div>
                <div>
                  <p className="text-xs text-slate-500">{t('avg_cost_per_order') || 'Avg Cost/Order'}</p>
                  <p className="text-xl font-bold text-slate-900">${summary.avg_cost_per_order.toFixed(2)}</p>
                </div>
              </div>
            </div>
          </Card>
          <Card>
            <div className="p-4">
              <div className="flex items-center gap-3">
                <div className="p-2.5 bg-rose-50 rounded-xl">
                  <PieChart className="w-5 h-5 text-rose-600" />
                </div>
                <div>
                  <p className="text-xs text-slate-500">{t('currency') || 'Currency'}</p>
                  <p className="text-xl font-bold text-slate-900">{summary.currency}</p>
                </div>
              </div>
            </div>
          </Card>
        </div>
      )}

      {/* Cost Breakdown Table */}
      <Card>
        <div className="p-5">
          <h3 className="text-sm font-semibold text-slate-900 mb-4 flex items-center gap-2">
            <PieChart className="w-4 h-4" />
            {t('cost_breakdown') || 'Cost Breakdown by Category'}
          </h3>
          <DataGrid
            data={breakdown}
            columns={[
              {
                key: 'category',
                header: t('category') || 'Category',
                sortable: true,
                render: (item: WorkOrderCostBreakdown) => {
                  const cfg = categoryConfig[item.category] || categoryConfig.additional;
                  const Icon = cfg.icon;
                  return (
                    <div className="flex items-center gap-2">
                      <div className={`p-1.5 rounded-lg ${cfg.bg}`}>
                        <Icon className={`w-4 h-4 ${cfg.color}`} />
                      </div>
                      <span className="font-medium">{cfg.label}</span>
                    </div>
                  );
                },
              },
              {
                key: 'amount', header: t('amount') || 'Amount', sortable: true,
                render: (item: WorkOrderCostBreakdown) => (
                  <span className="font-mono font-medium">${item.amount.toFixed(2)}</span>
                ),
              },
              {
                key: 'count', header: t('entries') || 'Entries', sortable: true,
                render: (item: WorkOrderCostBreakdown) => <Badge variant="info">{item.count}</Badge>,
              },
              {
                key: 'percent', header: '%', sortable: true,
                render: (item: WorkOrderCostBreakdown) => (
                  <div className="flex items-center gap-2">
                    <div className="w-24 bg-slate-200 rounded-full h-2">
                      <div className={`h-2 rounded-full ${item.category === 'labor' ? 'bg-blue-500' : item.category === 'parts' ? 'bg-emerald-500' : 'bg-amber-500'}`}
                        style={{ width: `${Math.min(item.percent, 100)}%` }} />
                    </div>
                    <span className="text-xs font-mono">{item.percent.toFixed(1)}%</span>
                  </div>
                ),
              },
            ]}
            keyExtractor={(item) => item.category}
            emptyMessage={t('no_cost_data') || 'No cost data'}
            variant="striped"
            defaultDensity="compact"
            exportFilename="cost-breakdown.csv"
          />
        </div>
      </Card>

      {/* TCO by Device Table */}
      <Card>
        <div className="p-5">
          <h3 className="text-sm font-semibold text-slate-900 mb-4 flex items-center gap-2">
            <DollarSign className="w-4 h-4" />
            {t('tco_by_device') || 'TCO по устройствам'}
          </h3>
          <DataGrid
            data={tcoData}
            columns={[
              { key: 'device_name', header: t('device') || 'Device', sortable: true,
                render: (item: TCOPerDevice) => (
                  <div>
                    <p className="font-medium text-sm">{item.device_name}</p>
                    <p className="text-xs text-slate-500">{item.device_type} · {item.vendor_type}</p>
                  </div>
                ),
              },
              {
                key: 'total_downtime_cost', header: t('downtime_cost') || 'Downtime $', sortable: true,
                render: (item: TCOPerDevice) => (
                  <span className="font-mono text-red-600 font-medium">
                    ${item.total_downtime_cost.toFixed(2)}
                  </span>
                ),
              },
              {
                key: 'tco', header: 'TCO', sortable: true,
                render: (item: TCOPerDevice) => (
                  <span className="font-mono font-bold">${item.tco.toFixed(2)}</span>
                ),
              },
              {
                key: 'total_downtime_events', header: t('events') || 'События', sortable: true,
              },
              {
                key: 'total_work_orders', header: t('work_orders') || 'Наряды', sortable: true,
              },
            ]}
            keyExtractor={(item) => item.device_id}
            emptyMessage={t('no_tco_data') || 'Нет данных TCO'}
            variant="striped"
            defaultDensity="compact"
            exportFilename="tco-by-device.csv"
          />
        </div>
      </Card>
    </div>
  );
};
