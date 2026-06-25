import React from 'react';
import { useTranslation } from 'react-i18next';
import { Gauge } from '../ui/Gauge';
import { Card } from '../ui/Card';

// P0-4.2: Четыре ключевые SLA-метрики для панели gauges
interface SLAMetric {
  key: string;
  label: string;
  value: number;
}

interface SLAGaugePanelProps {
  overallCompliance: number;
  mttrCompliance: number;
  preventiveCompliance: number;
  emergencyResponse: number;
  loading?: boolean;
}

export function SLAGaugePanel({
  overallCompliance,
  mttrCompliance,
  preventiveCompliance,
  emergencyResponse,
  loading = false,
}: SLAGaugePanelProps) {
  const { t } = useTranslation();

  const metrics: SLAMetric[] = [
    { key: 'overall', label: t('overall_compliance'), value: overallCompliance },
    { key: 'mttr', label: t('mttr_compliance') || 'MTTR Compliance', value: mttrCompliance },
    { key: 'preventive', label: t('preventive_compliance') || 'Preventive Compliance', value: preventiveCompliance },
    { key: 'emergency', label: t('emergency_response') || 'Emergency Response', value: emergencyResponse },
  ];

  if (loading) {
    return (
      <Card className="mb-6">
        <div className="grid grid-cols-2 lg:grid-cols-4 gap-6">
          {metrics.map((m) => (
            <div key={m.key} className="flex flex-col items-center gap-3">
              <div className="w-32 h-4 bg-slate-200 dark:bg-slate-700 rounded animate-pulse" />
              <div className="w-32 h-32 rounded-full bg-slate-200 dark:bg-slate-700 animate-pulse" />
            </div>
          ))}
        </div>
      </Card>
    );
  }

  return (
    <Card className="mb-6">
      <h3 className="text-lg font-semibold text-slate-900 dark:text-white mb-4">
        {t('sla_gauge_panel') || 'SLA Compliance Overview'}
      </h3>
      <div className="grid grid-cols-2 lg:grid-cols-4 gap-6">
        {metrics.map((m) => (
          <Gauge
            key={m.key}
            value={m.value}
            max={100}
            label={m.label}
            size="md"
            unit="%"
            showValue
          />
        ))}
      </div>
    </Card>
  );
}
