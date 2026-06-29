import React, { useMemo, useState } from 'react';
import { useTranslation } from 'react-i18next';
import {
  AlertTriangle,
  Clock,
  Filter,
  XCircle,
} from '../ui/Icons';
import { Card } from '../ui/Card';
import { Badge } from '../ui/Badge';
import { Button } from '../ui/Button';
import { Skeleton } from '../ui/Skeleton';

// P0-4.5: Timeline breach-событий с фильтром по severity

type Severity = 'critical' | 'high' | 'medium' | 'low';

interface BreachEvent {
  id: string;
  siteName: string;
  priority: string;
  severity: Severity;
  breachedAt: string; // ISO date
  responseTimeMinutes: number;
  resolutionTimeMinutes: number;
  description: string;
}

interface SLABreachTimelineProps {
  breaches: BreachEvent[];
  loading?: boolean;
}

const SEVERITY_CONFIG: Record<Severity, { label: string; variant: 'danger' | 'warning' | 'info' | 'neutral'; icon: typeof AlertTriangle }> = {
  critical: { label: 'Critical', variant: 'danger', icon: XCircle },
  high: { label: 'High', variant: 'warning', icon: AlertTriangle },
  medium: { label: 'Medium', variant: 'info', icon: Clock },
  low: { label: 'Low', variant: 'neutral', icon: Clock },
};

const SEVERITY_ORDER: Severity[] = ['critical', 'high', 'medium', 'low'];

export function SLABreachTimeline({ breaches, loading = false }: SLABreachTimelineProps) {
  const { t } = useTranslation();
  const [filterSeverity, setFilterSeverity] = useState<Severity | 'all'>('all');
  const [showAll, setShowAll] = useState(false);

  const filtered = useMemo(() => {
    let result = filterSeverity === 'all'
      ? [...breaches]
      : breaches.filter((b) => b.severity === filterSeverity);

    // Sort by date descending
    result.sort((a, b) => new Date(b.breachedAt).getTime() - new Date(a.breachedAt).getTime());

    if (!showAll) {
      result = result.slice(0, 10);
    }
    return result;
  }, [breaches, filterSeverity, showAll]);

  const severityCounts = useMemo(() => {
    const counts: Record<string, number> = { all: breaches.length };
    for (const s of SEVERITY_ORDER) {
      counts[s] = breaches.filter((b) => b.severity === s).length;
    }
    return counts;
  }, [breaches]);

  if (loading) {
    return (
      <Card className="mb-6">
        <div className="flex items-center justify-between mb-4">
          <Skeleton className="h-5 w-40" />
          <Skeleton className="h-8 w-48" />
        </div>
        {Array.from({ length: 4 }).map((_, i) => (
          <div key={i} className="flex items-start gap-3 mb-4">
            <Skeleton className="w-8 h-8 rounded-full flex-shrink-0" />
            <div className="flex-1">
              <Skeleton className="h-4 w-48 mb-1" />
              <Skeleton className="h-3 w-32" />
            </div>
          </div>
        ))}
      </Card>
    );
  }

  return (
    <Card className="mb-6">
      <div className="flex items-center justify-between mb-4">
        <div className="flex items-center gap-2">
          <h3 className="text-lg font-semibold text-slate-900 dark:text-white">
            {t('sla_breach_timeline') || 'SLA Breach Timeline'}
          </h3>
          {breaches.length > 0 && (
            <span className="text-xs text-slate-500 dark:text-slate-400 bg-slate-100 dark:bg-slate-700 px-2 py-0.5 rounded-full">
              {breaches.length}
            </span>
          )}
        </div>

        {/* Severity filter */}
        <div className="flex items-center gap-1 bg-slate-100 dark:bg-slate-700 rounded-lg p-0.5">
          <Filter className="w-3.5 h-3.5 text-slate-400 ml-1.5" />
          {(['all', ...SEVERITY_ORDER] as const).map((s) => (
            <button
              key={s}
              onClick={() => {
                setFilterSeverity(s);
                setShowAll(false);
              }}
              className={`px-2 py-1 text-xs font-medium rounded-md transition-colors ${
                filterSeverity === s
                  ? 'bg-white dark:bg-slate-600 text-blue-600 dark:text-blue-400 shadow-sm'
                  : 'text-slate-500 dark:text-slate-400 hover:text-slate-700 dark:hover:text-slate-200'
              }`}
            >
              {s === 'all' ? (t('all') || 'All') : (t(s) || s)}
              <span className="ml-1 opacity-60">({severityCounts[s]})</span>
            </button>
          ))}
        </div>
      </div>

      {filtered.length === 0 ? (
        <div className="flex flex-col items-center justify-center py-8 text-sm text-slate-500 dark:text-slate-400">
          <AlertTriangle className="w-8 h-8 mb-2 opacity-40" />
          <p>{t('no_breaches') || 'No SLA breaches found'}</p>
        </div>
      ) : (
        <div className="space-y-3">
          {filtered.map((breach, idx) => {
            const cfg = SEVERITY_CONFIG[breach.severity];
            const SeverityIcon = cfg.icon;
            const breachedDate = new Date(breach.breachedAt);

            return (
              <div
                key={breach.id}
                className={`relative flex items-start gap-3 p-3 rounded-lg transition-colors hover:bg-slate-50 dark:hover:bg-slate-700/50 ${
                  idx < filtered.length - 1
                    ? 'border-l-2 border-slate-200 dark:border-slate-600 ml-4'
                    : ''
                }`}
              >
                {/* Timeline dot */}
                <div
                  className={`absolute -left-[1.15rem] top-4 w-3 h-3 rounded-full border-2 border-white dark:border-slate-800 ${
                    breach.severity === 'critical'
                      ? 'bg-red-500'
                      : breach.severity === 'high'
                      ? 'bg-orange-500'
                      : breach.severity === 'medium'
                      ? 'bg-yellow-500'
                      : 'bg-slate-400'
                  }`}
                />

                <div className="flex-shrink-0 mt-0.5">
                  <SeverityIcon className={`w-4 h-4 ${
                    breach.severity === 'critical'
                      ? 'text-red-500'
                      : breach.severity === 'high'
                      ? 'text-orange-500'
                      : breach.severity === 'medium'
                      ? 'text-yellow-500'
                      : 'text-slate-400'
                  }`} />
                </div>

                <div className="flex-1 min-w-0">
                  <div className="flex items-center gap-2 flex-wrap">
                    <span className="font-medium text-sm text-slate-900 dark:text-white">
                      {breach.siteName}
                    </span>
                    <Badge variant={cfg.variant}>{cfg.label}</Badge>
                    <Badge variant="info">{t(breach.priority) || breach.priority}</Badge>
                  </div>
                  <p className="text-xs text-slate-500 dark:text-slate-400 mt-0.5 line-clamp-2">
                    {breach.description}
                  </p>
                  <div className="flex items-center gap-3 mt-1.5 text-xs text-slate-400 dark:text-slate-500">
                    <span>
                      {breachedDate.toLocaleDateString()} {breachedDate.toLocaleTimeString([], { hour: '2-digit', minute: '2-digit' })}
                    </span>
                    <span>
                      {t('response') || 'Response'}: {breach.responseTimeMinutes} min
                    </span>
                    <span>
                      {t('resolution') || 'Resolution'}: {breach.resolutionTimeMinutes} min
                    </span>
                  </div>
                </div>
              </div>
            );
          })}
        </div>
      )}

      {/* Show more / less */}
      {breaches.length > 10 && (
        <div className="text-center mt-4 pt-3 border-t border-slate-100 dark:border-slate-700">
          <Button
            variant="ghost"
            size="sm"
            onClick={() => setShowAll(!showAll)}
          >
            {showAll
              ? (t('show_less') || 'Show less')
              : (t('show_all_breaches', { count: breaches.length }) || `Show all ${breaches.length} breaches`)}
          </Button>
        </div>
      )}
    </Card>
  );
}
