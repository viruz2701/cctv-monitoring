import React, { useEffect, useState, useMemo } from 'react';
import { useTranslation } from 'react-i18next';
import { Activity } from 'lucide-react';
import { request } from '../services/api';
import { Card, Table, Badge, Gauge } from '../components/ui';

interface SLAConfig {
  id: string;
  priority: string;
  response_time_minutes: number;
  resolution_time_minutes: number;
}

interface SLAComplianceReport {
  priority: string;
  total_work_orders: number;
  within_sla: number;
  breached_sla: number;
  compliance_percent: number;
  avg_response_minutes: number;
  avg_resolution_minutes: number;
}

const GAUGE_THRESHOLDS = [
  { value: 90, color: '#16a34a', label: '≥90%' },
  { value: 70, color: '#d97706', label: '≥70%' },
  { value: 0, color: '#dc2626', label: '<70%' },
];

export const SLADashboard: React.FC = () => {
  const { t } = useTranslation();
  const [configs, setConfigs] = useState<SLAConfig[]>([]);
  const [reports, setReports] = useState<SLAComplianceReport[]>([]);
  const [loading, setLoading] = useState(false);

  useEffect(() => {
    const fetch = async () => {
      setLoading(true);
      try {
        const [c, r] = await Promise.all([
          request<SLAConfig[]>('/sla/config'),
          request<SLAComplianceReport[]>('/reports/sla-compliance'),
        ]);
        setConfigs(c || []);
        setReports(r || []);
      } catch (err) {
        console.error(err);
      } finally {
        setLoading(false);
      }
    };
    fetch();
  }, []);

  const overallCompliance = useMemo(() => {
    if (reports.length === 0) return 0;
    const total = reports.reduce((s, r) => s + r.total_work_orders, 0);
    const within = reports.reduce((s, r) => s + r.within_sla, 0);
    return total > 0 ? (within / total) * 100 : 0;
  }, [reports]);

  const getComplianceColor = (percent: number) => {
    if (percent >= 90) return 'success';
    if (percent >= 70) return 'warning';
    return 'danger';
  };

  const configColumns = [
    { key: 'priority', header: t('priority'), render: (item: SLAConfig) => <Badge variant="info">{t(item.priority)}</Badge> },
    { key: 'response_time_minutes', header: t('response_time'), render: (item: SLAConfig) => `${item.response_time_minutes} min` },
    { key: 'resolution_time_minutes', header: t('resolution_time'), render: (item: SLAConfig) => `${item.resolution_time_minutes} min` },
  ];

  const reportColumns = [
    { key: 'priority', header: t('priority'), render: (item: SLAComplianceReport) => <Badge variant="info">{t(item.priority)}</Badge> },
    { key: 'total_work_orders', header: t('total') },
    { key: 'within_sla', header: t('within_sla'), render: (item: SLAComplianceReport) => <span className="text-green-600">{item.within_sla}</span> },
    { key: 'breached_sla', header: t('breached'), render: (item: SLAComplianceReport) => <span className="text-red-600">{item.breached_sla}</span> },
    {
      key: 'compliance_percent',
      header: t('compliance'),
      render: (item: SLAComplianceReport) => (
        <Badge variant={getComplianceColor(item.compliance_percent) as 'success' | 'warning' | 'danger'}>
          {item.compliance_percent.toFixed(1)}%
        </Badge>
      ),
    },
    { key: 'avg_response_minutes', header: t('avg_response'), render: (item: SLAComplianceReport) => `${item.avg_response_minutes.toFixed(1)} min` },
    { key: 'avg_resolution_minutes', header: t('avg_resolution'), render: (item: SLAComplianceReport) => `${item.avg_resolution_minutes.toFixed(1)} min` },
  ];

  return (
    <div className="p-6">
      <h1 className="text-2xl font-bold mb-6">{t('sla_dashboard')}</h1>

      {reports.length > 0 && (
        <Card className="mb-6">
          <div className="flex items-center gap-2 mb-6">
            <Activity className="w-5 h-5 text-blue-600 dark:text-blue-400" />
            <h3 className="text-lg font-semibold">{t('overall_sla_compliance')}</h3>
          </div>
          <div className="flex flex-wrap items-center justify-center gap-8">
            <Gauge
              value={overallCompliance}
              max={100}
              label={t('overall_compliance')}
              size="lg"
              thresholds={GAUGE_THRESHOLDS}
              unit="%"
            />
            <div className="grid grid-cols-1 sm:grid-cols-3 gap-6">
              {reports.map((r) => (
                <Gauge
                  key={r.priority}
                  value={r.compliance_percent}
                  max={100}
                  label={t(r.priority)}
                  size="md"
                  thresholds={GAUGE_THRESHOLDS}
                  unit="%"
                />
              ))}
            </div>
          </div>
        </Card>
      )}

      <div className="grid grid-cols-1 lg:grid-cols-2 gap-6">
        <Card>
          <h3 className="text-lg font-semibold mb-4">{t('sla_configuration')}</h3>
          <Table data={configs} columns={configColumns} keyExtractor={(item) => item.id} loading={loading} />
        </Card>

        <Card>
          <h3 className="text-lg font-semibold mb-4">{t('sla_compliance_30d')}</h3>
          <Table data={reports} columns={reportColumns} keyExtractor={(item) => item.priority} loading={loading} />
        </Card>
      </div>
    </div>
  );
};
