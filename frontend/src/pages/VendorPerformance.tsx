import React, { useState, useEffect } from 'react';
import { useTranslation } from 'react-i18next';
import { request } from '../services/api';
import { Card, Badge, Button, StatsCard, DataGrid } from '../components/ui';
import {
  Building2, TrendingUp, TrendingDown, RefreshCw,
  BarChart3, Star, Truck, Clock, DollarSign,
} from '../components/ui/Icons';
import { ResponsiveBar } from '@nivo/bar';
import { ResponsivePie } from '@nivo/pie';

interface Vendor {
  id: string;
  name: string;
  contact_person?: string;
  email?: string;
  phone?: string;
  status: string;
  rating?: number;
  avg_delivery_days?: number;
  total_orders?: number;
  total_cost?: number;
  on_time_delivery?: number;
}

const PIE_COLORS = ['#10b981', '#3b82f6', '#f59e0b', '#ef4444', '#8b5cf6', '#06b6d4', '#ec4899'];

const nivoTheme = {
  axis: {
    ticks: { text: { fontSize: 10, fill: '#94a3b8' } },
    domain: { line: { stroke: '#f1f5f9', strokeWidth: 1 } },
  },
  grid: { line: { stroke: '#f1f5f9', strokeDasharray: '3 3', strokeWidth: 1 } },
};

export function VendorPerformance() {
  const { t } = useTranslation();
  const [vendors, setVendors] = useState<Vendor[]>([]);
  const [loading, setLoading] = useState(false);

  useEffect(() => { fetchData(); }, []);

  const fetchData = async () => {
    setLoading(true);
    try {
      const data = await request<Vendor[]>('/vendors');
      setVendors(data || []);
    } catch { setVendors([]); }
    finally { setLoading(false); }
  };

  const totalVendors = vendors.length;
  const activeVendors = vendors.filter(v => v.status === 'active').length;
  const avgRating = vendors.length > 0
    ? vendors.reduce((s, v) => s + (v.rating || 0), 0) / vendors.length : 0;
  const avgDelivery = vendors.length > 0
    ? vendors.reduce((s, v) => s + (v.avg_delivery_days || 0), 0) / vendors.length : 0;

  const barData = vendors.map(v => ({
    name: v.name,
    rating: (v.rating || 0),
    delivery: (v.avg_delivery_days || 0),
    orders: v.total_orders || 0,
  }));

  const pieData = vendors.filter(v => (v.total_cost || 0) > 0).map(v => ({
    id: v.name,
    label: v.name,
    value: v.total_cost || 0,
  }));

  const columns = [
    { key: 'name', header: t('vendor') || 'Поставщик', sortable: true },
    {
      key: 'rating', header: t('rating') || 'Рейтинг', sortable: true,
      render: (v: Vendor) => (
        <div className="flex items-center gap-1">
          <Star className="w-3.5 h-3.5 text-amber-400" />
          <span className="font-medium">{(v.rating || 0).toFixed(1)}</span>
        </div>
      ),
    },
    { key: 'avg_delivery_days', header: t('avg_delivery') || 'Доставка (дн)', sortable: true,
      render: (v: Vendor) => <span className="font-mono">{v.avg_delivery_days || '—'}</span> },
    { key: 'on_time_delivery', header: t('on_time') || 'В срок (%)', sortable: true,
      render: (v: Vendor) => v.on_time_delivery ? `${v.on_time_delivery}%` : '—' },
    { key: 'total_orders', header: t('orders') || 'Заказы', sortable: true,
      render: (v: Vendor) => <Badge variant="info">{v.total_orders || 0}</Badge> },
    { key: 'total_cost', header: t('total_cost') || 'Затраты', sortable: true,
      render: (v: Vendor) => <span className="font-mono">${(v.total_cost || 0).toFixed(2)}</span> },
    { key: 'status', header: t('status') || 'Статус',
      render: (v: Vendor) => v.status === 'active' ? <Badge variant="success">Active</Badge> : <Badge variant="info">Inactive</Badge> },
  ];

  return (
    <div className="space-y-6">
      <div className="flex items-center justify-between">
        <div>
          <h1 className="text-2xl font-bold text-slate-900 flex items-center gap-2">
            <Truck className="w-6 h-6" />
            {t('vendor_performance') || 'Аналитика поставщиков'}
          </h1>
          <p className="text-sm text-slate-500 mt-1">
            {t('vendor_performance_desc') || 'Производительность и надёжность поставщиков'}
          </p>
        </div>
        <Button variant="outline" icon={<RefreshCw className="w-4 h-4" />} onClick={fetchData} loading={loading}>
          {t('refresh') || 'Обновить'}
        </Button>
      </div>

      <div className="grid grid-cols-1 md:grid-cols-4 gap-4">
        <StatsCard title={t('total_vendors') || 'Поставщики'} value={totalVendors} icon={Building2} iconBgColor="bg-blue-50" iconColor="text-blue-600" />
        <StatsCard title={t('active_vendors') || 'Активны'} value={activeVendors} icon={Truck} iconBgColor="bg-emerald-50" iconColor="text-emerald-600" />
        <StatsCard title={t('avg_rating') || 'Средний рейтинг'} value={avgRating.toFixed(1)} icon={Star} iconBgColor="bg-amber-50" iconColor="text-amber-600" />
        <StatsCard title={t('avg_delivery') || 'Срок доставки'} value={`${avgDelivery.toFixed(1)} дн`} icon={Clock} iconBgColor="bg-purple-50" iconColor="text-purple-600" />
      </div>

      <div className="grid grid-cols-1 lg:grid-cols-2 gap-4">
        <Card>
          <div className="p-4">
            <h3 className="text-sm font-semibold text-slate-900 mb-4 flex items-center gap-2">
              <BarChart3 className="w-4 h-4" />
              {t('rating_vs_delivery') || 'Рейтинг и сроки доставки'}
            </h3>
            <div className="h-64">
              <ResponsiveBar
                data={barData}
                keys={['rating', 'delivery']}
                indexBy="name"
                margin={{ top: 10, right: 20, bottom: 40, left: 50 }}
                padding={0.3}
                groupMode="grouped"
                colors={['#f59e0b', '#3b82f6']}
                colorBy="indexValue"
                borderRadius={4}
                axisBottom={{
                  tickSize: 5, tickPadding: 5, tickRotation: -20,
                }}
                axisLeft={{
                  tickSize: 5, tickPadding: 5, tickRotation: 0,
                }}
                theme={nivoTheme}
                enableLabel={false}
                legends={[
                  {
                    dataFrom: 'keys',
                    anchor: 'bottom',
                    direction: 'row',
                    translateY: 36,
                    itemWidth: 100,
                    itemHeight: 14,
                    itemTextColor: '#94a3b8',
                    symbolSize: 10,
                    symbolShape: 'square',
                  },
                ]}
                tooltip={({ data: d, id, value }) => (
                  <div style={{ background: 'rgba(255,255,255,0.95)', border: '1px solid #e2e8f0', borderRadius: 8, padding: '4px 8px', fontSize: 12 }}>
                    <strong>{String(d.name)}</strong> — {String(id)}: {value}
                  </div>
                )}
              />
            </div>
          </div>
        </Card>

        <Card>
          <div className="p-4">
            <h3 className="text-sm font-semibold text-slate-900 mb-4 flex items-center gap-2">
              <DollarSign className="w-4 h-4" />
              {t('cost_distribution') || 'Распределение затрат'}
            </h3>
            <div className="h-64">
              <ResponsivePie
                data={pieData.length > 0 ? pieData : [{ id: 'No data', label: 'No data', value: 1 }]}
                margin={{ top: 20, right: 40, bottom: 20, left: 40 }}
                innerRadius={0.45}
                padAngle={2}
                cornerRadius={4}
                colors={PIE_COLORS}
                arcLinkLabelsSkipAngle={10}
                arcLinkLabelsTextColor="#64748b"
                arcLinkLabelsThickness={1}
                arcLinkLabelsColor={{ from: 'color' }}
                arcLabelsSkipAngle={10}
                arcLabelsTextColor="#ffffff"
                theme={nivoTheme}
                tooltip={({ datum }) => (
                  <div style={{ background: 'rgba(255,255,255,0.95)', border: '1px solid #e2e8f0', borderRadius: 8, padding: '4px 8px', fontSize: 12 }}>
                    <strong>{datum.label}</strong>: ${Number(datum.value).toFixed(2)}
                  </div>
                )}
                legends={[
                  {
                    anchor: 'bottom',
                    direction: 'row',
                    translateY: 36,
                    itemWidth: 80,
                    itemHeight: 14,
                    itemTextColor: '#94a3b8',
                    symbolSize: 10,
                    symbolShape: 'circle',
                  },
                ]}
              />
            </div>
          </div>
        </Card>
      </div>

      <Card>
        <div className="p-4">
          <DataGrid
            data={vendors}
            columns={columns}
            keyExtractor={(v: Vendor) => v.id}
            loading={loading}
            emptyMessage={t('no_vendors') || 'Нет поставщиков'}
            variant="striped"
            defaultDensity="compact"
            pageSize={25}
            exportFilename="vendor-performance.csv"
          />
        </div>
      </Card>
    </div>
  );
}
