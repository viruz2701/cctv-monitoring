import React, { useEffect, useState } from 'react';
import { useTranslation } from 'react-i18next';
import { request } from '../services/api';
import { Card, DataGrid } from '../components/ui';
import { formatCurrency } from '../utils/currency';
import { BarChart3 } from 'lucide-react';

interface MaintenanceReport {
  device_id: string;
  device_name: string;
  total_work_orders: number;
  completed_count: number;
  overdue_count: number;
  mttr_minutes: number;
  total_cost: number;
}

export const MaintenanceReports: React.FC = () => {
  const { t } = useTranslation();
  const [reports, setReports] = useState<MaintenanceReport[]>([]);
  const [loading, setLoading] = useState(false);

  useEffect(() => {
    const fetch = async () => {
      setLoading(true);
      try {
        const data = await request<MaintenanceReport[]>('/reports/maintenance');
        setReports(data || []);
      } catch (err) {
        console.error(err);
      } finally {
        setLoading(false);
      }
    };
    fetch();
  }, []);

  const columns = [
    { key: 'device_name', header: t('device'), sortable: true },
    { key: 'total_work_orders', header: t('total_work_orders'), sortable: true },
    { key: 'completed_count', header: t('completed'), sortable: true, render: (item: MaintenanceReport) => <span className="text-green-600">{item.completed_count}</span> },
    { key: 'overdue_count', header: t('overdue'), sortable: true, render: (item: MaintenanceReport) => <span className={item.overdue_count > 0 ? 'text-red-600 font-bold' : ''}>{item.overdue_count}</span> },
    { key: 'mttr_minutes', header: 'MTTR', sortable: true, render: (item: MaintenanceReport) => `${item.mttr_minutes.toFixed(1)} min` },
    { key: 'total_cost', header: t('total_cost'), sortable: true, render: (item: MaintenanceReport) => formatCurrency(item.total_cost) },
  ];

  const totalCost = reports.reduce((sum, r) => sum + r.total_cost, 0);
  const totalOrders = reports.reduce((sum, r) => sum + r.total_work_orders, 0);
  const totalOverdue = reports.reduce((sum, r) => sum + r.overdue_count, 0);

  return (
    <div className="p-6">
      <h1 className="text-2xl font-bold mb-6 flex items-center gap-2">
        <BarChart3 size={24} />
        {t('maintenance_reports')}
      </h1>

      <div className="grid grid-cols-1 md:grid-cols-3 gap-4 mb-6">
        <Card>
          <div className="text-sm text-slate-500">{t('total_work_orders')}</div>
          <div className="text-3xl font-bold">{totalOrders}</div>
        </Card>
        <Card>
          <div className="text-sm text-slate-500">{t('overdue')}</div>
          <div className={`text-3xl font-bold ${totalOverdue > 0 ? 'text-red-600' : ''}`}>{totalOverdue}</div>
        </Card>
        <Card>
          <div className="text-sm text-slate-500">{t('total_cost')}</div>
          <div className="text-3xl font-bold">{formatCurrency(totalCost)}</div>
        </Card>
      </div>

      <Card>
        <DataGrid
          data={reports}
          columns={columns}
          keyExtractor={(item) => item.device_id}
          loading={loading}
          emptyMessage={t('no_data')}
          variant="striped"
          defaultDensity="standard"
          pageSize={10}
          exportFilename="maintenance-reports.csv"
        />
      </Card>
    </div>
  );
};
