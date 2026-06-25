import React, { useMemo, useState } from 'react';
import { useTranslation } from 'react-i18next';
import { Card } from '../ui/Card';
import { Skeleton } from '../ui/Skeleton';

// P0-4.3: Тепловая карта SLA compliance по сайтам и неделям

interface SiteCompliance {
  siteId: string;
  siteName: string;
  weeks: {
    weekStart: string; // ISO date
    compliance: number; // 0–100
    total: number;
    within: number;
  }[];
}

interface SLAHeatmapProps {
  data: SiteCompliance[];
  loading?: boolean;
}

// Цветовая шкала от красного (0%) до зелёного (100%)
function complianceColor(value: number): string {
  if (value >= 95) return '#16a34a';      // green
  if (value >= 80) return '#eab308';      // yellow
  if (value >= 60) return '#f97316';      // orange
  return '#dc2626';                        // red
}

function complianceBg(value: number): string {
  if (value >= 95) return 'bg-emerald-500';
  if (value >= 80) return 'bg-yellow-500';
  if (value >= 60) return 'bg-orange-500';
  return 'bg-red-500';
}

export function SLAHeatmap({ data, loading = false }: SLAHeatmapProps) {
  const { t } = useTranslation();
  const [hoveredCell, setHoveredCell] = useState<{
    site: string;
    week: string;
    compliance: number;
    total: number;
    within: number;
  } | null>(null);
  const [tooltipPos, setTooltipPos] = useState({ x: 0, y: 0 });

  // Collect unique week labels across all sites
  const weekLabels = useMemo(() => {
    const seen = new Set<string>();
    const labels: { key: string; short: string }[] = [];
    for (const site of data) {
      for (const w of site.weeks) {
        if (!seen.has(w.weekStart)) {
          seen.add(w.weekStart);
          const d = new Date(w.weekStart);
          labels.push({
            key: w.weekStart,
            short: `${d.getDate()}/${d.getMonth() + 1}`,
          });
        }
      }
    }
    // Sort by date
    labels.sort((a, b) => a.key.localeCompare(b.key));
    return labels;
  }, [data]);

  const getCompliance = (site: SiteCompliance, weekKey: string): number => {
    const w = site.weeks.find((x) => x.weekStart === weekKey);
    return w?.compliance ?? -1; // -1 = no data
  };

  const getWeekData = (site: SiteCompliance, weekKey: string) => {
    return site.weeks.find((x) => x.weekStart === weekKey);
  };

  const handleMouseEnter = (
    e: React.MouseEvent,
    siteName: string,
    weekKey: string,
    compliance: number,
    total: number,
    within: number,
  ) => {
    setHoveredCell({ site: siteName, week: weekKey, compliance, total, within });
    setTooltipPos({ x: e.clientX, y: e.clientY });
  };

  const handleMouseLeave = () => {
    setHoveredCell(null);
  };

  if (loading) {
    return (
      <Card className="mb-6">
        <div className="h-4 w-48 bg-slate-200 dark:bg-slate-700 rounded animate-pulse mb-4" />
        <Skeleton className="h-64 w-full" />
      </Card>
    );
  }

  if (!data.length || !weekLabels.length) {
    return (
      <Card className="mb-6">
        <h3 className="text-lg font-semibold text-slate-900 dark:text-white mb-2">
          {t('sla_heatmap') || 'SLA Heatmap'}
        </h3>
        <p className="text-sm text-slate-500 dark:text-slate-400">
          {t('no_data') || 'No data available'}
        </p>
      </Card>
    );
  }

  return (
    <Card className="mb-6 relative">
      <h3 className="text-lg font-semibold text-slate-900 dark:text-white mb-4">
        {t('sla_heatmap') || 'SLA Heatmap'}
      </h3>

      <div className="overflow-x-auto">
        <table className="w-full text-xs">
          <thead>
            <tr>
              <th className="text-left text-slate-500 dark:text-slate-400 font-medium px-2 py-1 sticky left-0 bg-white dark:bg-slate-800 z-10">
                {t('site') || 'Site'}
              </th>
              {weekLabels.map((wl) => (
                <th
                  key={wl.key}
                  className="text-center text-slate-500 dark:text-slate-400 font-medium px-1 py-1 min-w-[3rem]"
                >
                  {wl.short}
                </th>
              ))}
            </tr>
          </thead>
          <tbody>
            {data.map((site) => (
              <tr key={site.siteId}>
                <td className="text-left text-slate-700 dark:text-slate-300 font-medium px-2 py-1.5 sticky left-0 bg-white dark:bg-slate-800 z-10 whitespace-nowrap">
                  {site.siteName}
                </td>
                {weekLabels.map((wl) => {
                  const comp = getCompliance(site, wl.key);
                  const wd = getWeekData(site, wl.key);
                  const hasData = comp >= 0;
                  return (
                    <td
                      key={wl.key}
                      className="px-1 py-1.5"
                    >
                      {hasData ? (
                        <div
                          className={`w-8 h-8 rounded-md cursor-pointer transition-transform hover:scale-110 hover:ring-2 hover:ring-slate-400 ${complianceBg(comp)}`}
                          style={{ opacity: 0.3 + (comp / 100) * 0.7 }}
                          onMouseEnter={(e) =>
                            handleMouseEnter(
                              e,
                              site.siteName,
                              wl.key,
                              comp,
                              wd?.total ?? 0,
                              wd?.within ?? 0,
                            )
                          }
                          onMouseMove={(e) => setTooltipPos({ x: e.clientX, y: e.clientY })}
                          onMouseLeave={handleMouseLeave}
                        />
                      ) : (
                        <div className="w-8 h-8 rounded-md bg-slate-100 dark:bg-slate-700/50" />
                      )}
                    </td>
                  );
                })}
              </tr>
            ))}
          </tbody>
        </table>
      </div>

      {/* Legend */}
      <div className="flex items-center gap-4 mt-3 text-xs text-slate-500 dark:text-slate-400">
        <span>{t('legend') || 'Legend'}:</span>
        <div className="flex items-center gap-1">
          <span className="w-3 h-3 rounded-sm bg-emerald-500" /> ≥95%
        </div>
        <div className="flex items-center gap-1">
          <span className="w-3 h-3 rounded-sm bg-yellow-500" /> 80–94%
        </div>
        <div className="flex items-center gap-1">
          <span className="w-3 h-3 rounded-sm bg-orange-500" /> 60–79%
        </div>
        <div className="flex items-center gap-1">
          <span className="w-3 h-3 rounded-sm bg-red-500" /> {'<'}60%
        </div>
        <div className="flex items-center gap-1">
          <span className="w-3 h-3 rounded-sm bg-slate-100 dark:bg-slate-700" /> N/A
        </div>
      </div>

      {/* Tooltip */}
      {hoveredCell && (
        <div
          className="fixed z-50 bg-white dark:bg-slate-800 border border-slate-200 dark:border-slate-700 rounded-lg shadow-lg p-3 text-sm pointer-events-none"
          style={{
            left: tooltipPos.x + 12,
            top: tooltipPos.y - 10,
            transform: 'translateY(-100%)',
          }}
        >
          <p className="font-semibold text-slate-900 dark:text-white mb-1">
            {hoveredCell.site}
          </p>
          <p className="text-slate-500 dark:text-slate-400 mb-1">{hoveredCell.week}</p>
          <div className="flex items-center gap-2">
            <span
              className="w-2.5 h-2.5 rounded-full"
              style={{ backgroundColor: complianceColor(hoveredCell.compliance) }}
            />
            <span className="font-medium text-slate-900 dark:text-white">
              {hoveredCell.compliance.toFixed(1)}%
            </span>
          </div>
          <p className="text-slate-500 dark:text-slate-400 text-xs mt-1">
            {hoveredCell.within} / {hoveredCell.total} {t('within_sla') || 'within SLA'}
          </p>
        </div>
      )}
    </Card>
  );
}
