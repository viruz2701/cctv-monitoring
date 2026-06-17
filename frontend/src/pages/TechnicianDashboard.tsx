import React, { useEffect, useState } from 'react';
import { useTranslation } from 'react-i18next';
import { request } from '../services/api';
import { Card, Table, Badge } from '../components/ui';
import { CheckCircle, Clock, AlertTriangle, Users } from 'lucide-react';

interface TechnicianWorkload {
  user_id: string;
  user_name: string;
  current_workload: number;
  max_workload: number;
  skills: string[];
  base_location: string;
}

export const TechnicianDashboard: React.FC = () => {
  const { t } = useTranslation();
  const [workloads, setWorkloads] = useState<TechnicianWorkload[]>([]);
  const [loading, setLoading] = useState(false);

  useEffect(() => {
    const fetch = async () => {
      setLoading(true);
      try {
        const data = await request<TechnicianWorkload[]>('/technicians/workload');
        setWorkloads(data || []);
      } catch (err) {
        console.error(err);
      } finally {
        setLoading(false);
      }
    };
    fetch();
  }, []);

  const columns = [
    { key: 'user_name', header: t('technician') },
    {
      key: 'workload',
      header: t('workload'),
      render: (item: TechnicianWorkload) => {
        const percent = item.max_workload > 0 ? (item.current_workload / item.max_workload) * 100 : 0;
        const color = percent > 80 ? 'bg-red-500' : percent > 50 ? 'bg-yellow-500' : 'bg-green-500';
        return (
          <div className="flex items-center gap-2">
            <div className="w-32 bg-slate-200 dark:bg-slate-700 rounded-full h-2">
              <div className={`${color} h-2 rounded-full`} style={{ width: `${Math.min(percent, 100)}%` }} />
            </div>
            <span className="text-sm">{item.current_workload}/{item.max_workload}</span>
          </div>
        );
      },
    },
    {
      key: 'skills',
      header: t('skills'),
      render: (item: TechnicianWorkload) => (
        <div className="flex flex-wrap gap-1">
          {item.skills?.map((skill, i) => (
            <Badge key={i} variant="info" size="sm">{skill}</Badge>
          ))}
        </div>
      ),
    },
    { key: 'base_location', header: t('location') },
  ];

  return (
    <div className="p-6">
      <h1 className="text-2xl font-bold mb-6 flex items-center gap-2">
        <Users size={24} />
        {t('technician_dashboard')}
      </h1>

      <div className="grid grid-cols-1 md:grid-cols-3 gap-4 mb-6">
        <Card>
          <div className="flex items-center gap-3">
            <div className="p-3 bg-green-100 dark:bg-green-900/30 rounded-lg">
              <CheckCircle className="text-green-600" size={24} />
            </div>
            <div>
              <div className="text-2xl font-bold">{workloads.filter(w => w.current_workload < w.max_workload).length}</div>
              <div className="text-sm text-slate-500">{t('available')}</div>
            </div>
          </div>
        </Card>
        <Card>
          <div className="flex items-center gap-3">
            <div className="p-3 bg-yellow-100 dark:bg-yellow-900/30 rounded-lg">
              <Clock className="text-yellow-600" size={24} />
            </div>
            <div>
              <div className="text-2xl font-bold">{workloads.filter(w => w.current_workload >= w.max_workload * 0.8 && w.current_workload < w.max_workload).length}</div>
              <div className="text-sm text-slate-500">{t('busy')}</div>
            </div>
          </div>
        </Card>
        <Card>
          <div className="flex items-center gap-3">
            <div className="p-3 bg-red-100 dark:bg-red-900/30 rounded-lg">
              <AlertTriangle className="text-red-600" size={24} />
            </div>
            <div>
              <div className="text-2xl font-bold">{workloads.filter(w => w.current_workload >= w.max_workload).length}</div>
              <div className="text-sm text-slate-500">{t('overloaded')}</div>
            </div>
          </div>
        </Card>
      </div>

      <Card>
        <Table
          data={workloads}
          columns={columns}
          keyExtractor={(item) => item.user_id}
          loading={loading}
          emptyMessage={t('no_technicians')}
        />
      </Card>
    </div>
  );
};
