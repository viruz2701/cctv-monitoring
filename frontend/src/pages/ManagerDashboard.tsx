import React, { useEffect, useState } from 'react';
import { useTranslation } from 'react-i18next';
import { useNavigate } from 'react-router-dom';
import { request } from '../services/api';
import { Card, DataGrid, Badge, StatsCard } from '../components/ui';
import {
  LayoutDashboard, AlertTriangle, CheckCircle, Clock, DollarSign,
  Users, Activity, TrendingUp, FileText, BarChart3,
} from 'lucide-react';
import { formatCurrency } from '../utils/currency';

// ── Types ────────────────────────────────────────────────────────────

interface SLAComplianceReport {
  priority: string;
  total_work_orders: number;
  within_sla: number;
  breached_sla: number;
  compliance_percent: number;
  avg_response_minutes: number;
  avg_resolution_minutes: number;
}

interface ReliabilityMetric {
  vendor_type: string;
  device_type: string;
  device_count: number;
  mtbf_hours: number;
  mttr_minutes: number;
  total_downtime_minutes: number;
  total_completions: number;
}

interface WorkOrderCostSummary {
  total_work_orders: number;
  total_labor_cost: number;
  total_parts_cost: number;
  total_additional_cost: number;
  total_cost: number;
  avg_cost_per_order: number;
  currency: string;
}

interface TechnicianWorkload {
  user_id: string;
  user_name: string;
  current_workload: number;
  max_workload: number;
  skills: string[];
  base_location: string;
}

// ── Component ────────────────────────────────────────────────────────

export const ManagerDashboard: React.FC = () => {
  const { t } = useTranslation();
  const navigate = useNavigate();
  const [slaData, setSlaData] = useState<SLAComplianceReport[]>([]);
  const [reliability, setReliability] = useState<ReliabilityMetric[]>([]);
  const [costData, setCostData] = useState<WorkOrderCostSummary | null>(null);
  const [technicians, setTechnicians] = useState<TechnicianWorkload[]>([]);
  const [loading, setLoading] = useState(true);

  useEffect(() => {
    const fetchAll = async () => {
      setLoading(true);
      try {
        const [sla, rel, cost, tech] = await Promise.all([
          request<SLAComplianceReport[]>('/reports/sla-compliance').catch(() => []),
          request<ReliabilityMetric[]>('/analytics/reliability').catch(() => []),
          request<{ summary: WorkOrderCostSummary }>('/analytics/wo-costs').catch(() => null),
          request<TechnicianWorkload[]>('/technicians/workload').catch(() => []),
        ]);
        setSlaData(sla || []);
        setReliability(rel || []);
        setCostData(cost?.summary || null);
        setTechnicians(tech || []);
      } catch (err) {
        console.error('Failed to load manager dashboard data', err);
      } finally {
        setLoading(false);
      }
    };
    fetchAll();
  }, []);

  // Computed metrics
  const totalWO = costData?.total_work_orders || 0;
  const totalCost = costData?.total_cost || 0;
  const slaCompliance = slaData.length > 0
    ? (slaData.reduce((s, r) => s + r.within_sla, 0) / Math.max(slaData.reduce((s, r) => s + r.total_work_orders, 0), 1)) * 100
    : 0;
  const avgMTBF = reliability.length > 0
    ? reliability.reduce((s, r) => s + r.mtbf_hours, 0) / reliability.length
    : 0;
  const overloadedTechs = technicians.filter(t => t.current_workload >= t.max_workload).length;
  const totalBreached = slaData.reduce((s, r) => s + r.breached_sla, 0);

  const getComplianceColor = (pct: number) =>
    pct >= 90 ? 'success' as const : pct >= 70 ? 'warning' as const : 'danger' as const;

  return (
    <div className="space-y-6">
      <h1 className="text-2xl font-bold text-slate-900 dark:text-white flex items-center gap-2">
        <LayoutDashboard className="w-6 h-6" />
        {t('manager_dashboard') || 'Manager Dashboard'}
      </h1>

      {/* KPI Row */}
      <div className="grid grid-cols-2 md:grid-cols-4 lg:grid-cols-6 gap-3">
        <StatsCard title={t('total_work_orders') || 'Work Orders'} value={totalWO} icon={FileText}
          iconBgColor="bg-blue-50" iconColor="text-blue-600" />
        <StatsCard title={t('total_cost') || 'Total Cost'} value={formatCurrency(totalCost)} icon={DollarSign}
          iconBgColor="bg-slate-50" iconColor="text-slate-600" />
        <StatsCard title={t('sla_compliance') || 'SLA Compliance'} value={`${slaCompliance.toFixed(1)}%`} icon={Activity}
          iconBgColor="bg-emerald-50" iconColor="text-emerald-600" />
        <StatsCard title={t('breached') || 'Breached'} value={totalBreached} icon={AlertTriangle}
          iconBgColor="bg-red-50" iconColor="text-red-600" />
        <StatsCard title={t('avg_mtbf') || 'Avg MTBF'} value={`${Math.round(avgMTBF)}h`} icon={TrendingUp}
          iconBgColor="bg-purple-50" iconColor="text-purple-600" />
        <StatsCard title={t('overloaded') || 'Overloaded'} value={overloadedTechs} icon={Users}
          iconBgColor="bg-amber-50" iconColor="text-amber-600" />
      </div>

      {/* Secondary Row */}
      <div className="grid grid-cols-1 lg:grid-cols-2 gap-6">
        {/* SLA Compliance by Priority */}
        <Card>
          <div className="p-5">
            <h3 className="text-sm font-semibold text-slate-900 dark:text-white mb-4 flex items-center gap-2">
              <Activity className="w-4 h-4" />
              {t('sla_compliance_by_priority') || 'SLA Compliance by Priority'}
            </h3>
            <DataGrid
              data={slaData}
              columns={[
                { key: 'priority', header: t('priority') || 'Priority', sortable: true,
                  render: (r: SLAComplianceReport) => <Badge variant="info">{t(r.priority)}</Badge> },
                { key: 'total_work_orders', header: t('total') || 'Total', sortable: true },
                { key: 'within_sla', header: t('within_sla') || 'Within SLA', sortable: true,
                  render: (r: SLAComplianceReport) => <span className="text-green-600 font-medium">{r.within_sla}</span> },
                { key: 'breached_sla', header: t('breached') || 'Breached', sortable: true,
                  render: (r: SLAComplianceReport) => <span className="text-red-600 font-medium">{r.breached_sla}</span> },
                { key: 'compliance_percent', header: '%', sortable: true,
                  render: (r: SLAComplianceReport) => (
                    <Badge variant={getComplianceColor(r.compliance_percent)}>
                      {r.compliance_percent.toFixed(1)}%
                    </Badge>
                  ),
                },
              ]}
              keyExtractor={(r) => r.priority}
              variant="striped"
              defaultDensity="compact"
              pageSize={5}
            />
          </div>
        </Card>

        {/* Technician Workload */}
        <Card>
          <div className="p-5">
            <h3 className="text-sm font-semibold text-slate-900 dark:text-white mb-4 flex items-center gap-2">
              <Users className="w-4 h-4" />
              {t('technician_workload') || 'Technician Workload'}
            </h3>
            <DataGrid
              data={technicians}
              columns={[
                { key: 'user_name', header: t('technician') || 'Technician', sortable: true },
                { key: 'workload', header: t('workload') || 'Workload', sortable: true,
                  render: (t: TechnicianWorkload) => {
                    const pct = t.max_workload > 0 ? (t.current_workload / t.max_workload) * 100 : 0;
                    return (
                      <div className="flex items-center gap-2">
                        <div className="w-24 bg-slate-200 dark:bg-slate-700 rounded-full h-2">
                          <div className={`h-2 rounded-full ${
                            pct > 80 ? 'bg-red-500' : pct > 50 ? 'bg-yellow-500' : 'bg-green-500'
                          }`} style={{ width: `${Math.min(pct, 100)}%` }} />
                        </div>
                        <span className="text-xs">{t.current_workload}/{t.max_workload}</span>
                      </div>
                    );
                  },
                },
                { key: 'base_location', header: t('location') || 'Location', sortable: true },
              ]}
              keyExtractor={(t) => t.user_id}
              variant="striped"
              defaultDensity="compact"
              pageSize={5}
              onRowClick={(item) => navigate(`/technician-dashboard`)}
            />
          </div>
        </Card>
      </div>

      {/* Reliability Metrics */}
      <Card>
        <div className="p-5">
          <h3 className="text-sm font-semibold text-slate-900 dark:text-white mb-4 flex items-center gap-2">
            <BarChart3 className="w-4 h-4" />
            {t('reliability_metrics') || 'Reliability Metrics (MTBF/MTTR)'}
          </h3>
          <DataGrid
            data={reliability}
            columns={[
              { key: 'vendor_type', header: t('vendor') || 'Vendor', sortable: true },
              { key: 'device_type', header: t('device_type') || 'Device Type', sortable: true },
              { key: 'device_count', header: t('device_count') || 'Count', sortable: true },
              { key: 'mtbf_hours', header: 'MTBF (h)', sortable: true,
                render: (r: ReliabilityMetric) => r.mtbf_hours.toFixed(1) },
              { key: 'mttr_minutes', header: 'MTTR (min)', sortable: true,
                render: (r: ReliabilityMetric) => r.mttr_minutes.toFixed(1) },
              { key: 'total_completions', header: t('completions') || 'Completions', sortable: true },
            ]}
            keyExtractor={(r) => `${r.vendor_type}-${r.device_type}`}
            variant="striped"
            defaultDensity="compact"
            pageSize={10}
            exportFilename="reliability-metrics.csv"
          />
        </div>
      </Card>

      {/* Quick Links */}
      <div className="grid grid-cols-2 md:grid-cols-4 gap-3">
        <button onClick={() => navigate('/sla')}
          className="flex items-center gap-2 p-3 bg-white dark:bg-slate-800 rounded-xl border border-slate-200 dark:border-slate-700 hover:shadow-md transition-shadow text-left">
          <Activity className="w-5 h-5 text-blue-600" />
          <span className="text-sm font-medium">{t('sla_dashboard') || 'SLA Dashboard'}</span>
        </button>
        <button onClick={() => navigate('/work-orders')}
          className="flex items-center gap-2 p-3 bg-white dark:bg-slate-800 rounded-xl border border-slate-200 dark:border-slate-700 hover:shadow-md transition-shadow text-left">
          <FileText className="w-5 h-5 text-emerald-600" />
          <span className="text-sm font-medium">{t('work_orders') || 'Work Orders'}</span>
        </button>
        <button onClick={() => navigate('/cost-dashboard')}
          className="flex items-center gap-2 p-3 bg-white dark:bg-slate-800 rounded-xl border border-slate-200 dark:border-slate-700 hover:shadow-md transition-shadow text-left">
          <DollarSign className="w-5 h-5 text-amber-600" />
          <span className="text-sm font-medium">{t('cost_dashboard') || 'Cost Dashboard'}</span>
        </button>
        <button onClick={() => navigate('/technician-dashboard')}
          className="flex items-center gap-2 p-3 bg-white dark:bg-slate-800 rounded-xl border border-slate-200 dark:border-slate-700 hover:shadow-md transition-shadow text-left">
          <Users className="w-5 h-5 text-purple-600" />
          <span className="text-sm font-medium">{t('technicians') || 'Technicians'}</span>
        </button>
      </div>
    </div>
  );
};
