import React, { useEffect, useState } from 'react';
import { useTranslation } from 'react-i18next';
import { request } from '../services/api';
import { Card, Table, Badge } from '../components/ui';

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
