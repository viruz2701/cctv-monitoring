// ═══════════════════════════════════════════════════════════════════════
// JournalList.tsx — List of TO compliance journals with filtering
//
// Track 3: TO Compliance Automation
//   - UX-3.1: TO Journals with Regulatory Templates
//
// Features:
//   - Multi-region display with flags
//   - Status badges with color coding
//   - Generate / download actions
//   - Responsive table layout
// ═══════════════════════════════════════════════════════════════════════

import React, { useCallback } from 'react';
import { Badge, Button } from '../ui';
import {
  FileText,
  Download,
  Sparkles,
  Clock,
  CheckCircle,
  Edit3,
  Archive,
} from '../ui/Icons';
import type { LucideIcon } from '../ui/Icons';
import type { TOJournal } from '../../services/toJournalsApi';

// ═══════════════════════════════════════════════════════════════════════
// Types
// ═══════════════════════════════════════════════════════════════════════

interface JournalListProps {
  journals: TOJournal[];
  onGenerate: (journalId: string) => void;
  statusConfig: Record<string, { label: string; variant: 'success' | 'warning' | 'danger' | 'info' | 'neutral' | 'primary' | 'outline' }>;
  regions: { code: string; name: string; flag: string }[];
}

// ═══════════════════════════════════════════════════════════════════════
// Helpers
// ═══════════════════════════════════════════════════════════════════════

function getRegionInfo(
  regionCode: string,
  regions: { code: string; name: string; flag: string }[],
): { name: string; flag: string } {
  const region = regions.find((r) => r.code === regionCode);
  return {
    name: region?.name ?? regionCode,
    flag: region?.flag ?? '🌐',
  };
}

function formatDate(dateStr: string | null): string {
  if (!dateStr) return '—';
  try {
    return new Intl.DateTimeFormat('en-GB', {
      day: '2-digit',
      month: 'short',
      year: 'numeric',
    }).format(new Date(dateStr));
  } catch {
    return dateStr;
  }
}

// ═══════════════════════════════════════════════════════════════════════
// Status Icon Map
// ═══════════════════════════════════════════════════════════════════════

const STATUS_ICONS: Record<string, LucideIcon> = {
  draft: Clock,
  generated: CheckCircle,
  signed: Edit3,
  archived: Archive,
};

// ═══════════════════════════════════════════════════════════════════════
// JournalList Component
// ═══════════════════════════════════════════════════════════════════════

export function JournalList({ journals, onGenerate, statusConfig, regions }: JournalListProps) {
  if (!journals.length) return null;

  return (
    <div className="overflow-x-auto">
      <table className="w-full text-sm">
        <thead>
          <tr className="border-b border-slate-200 dark:border-slate-700">
            <th className="text-left py-3 px-4 font-medium text-slate-500 dark:text-slate-400">
              Region
            </th>
            <th className="text-left py-3 px-4 font-medium text-slate-500 dark:text-slate-400">
              Title
            </th>
            <th className="text-left py-3 px-4 font-medium text-slate-500 dark:text-slate-400">
              Template
            </th>
            <th className="text-left py-3 px-4 font-medium text-slate-500 dark:text-slate-400">
              Technician
            </th>
            <th className="text-left py-3 px-4 font-medium text-slate-500 dark:text-slate-400">
              Period
            </th>
            <th className="text-left py-3 px-4 font-medium text-slate-500 dark:text-slate-400">
              Status
            </th>
            <th className="text-left py-3 px-4 font-medium text-slate-500 dark:text-slate-400">
              Generated
            </th>
            <th className="text-right py-3 px-4 font-medium text-slate-500 dark:text-slate-400">
              Actions
            </th>
          </tr>
        </thead>
        <tbody>
          {journals.map((journal) => {
            const region = getRegionInfo(journal.region_code, regions);
            const statusCfg = statusConfig[journal.status] ?? {
              label: journal.status,
              variant: 'default' as const,
            };
            const StatusIcon = STATUS_ICONS[journal.status] ?? FileText;

            return (
              <tr
                key={journal.id}
                className="border-b border-slate-100 dark:border-slate-800 hover:bg-slate-50 dark:hover:bg-slate-800/50 transition-colors"
              >
                {/* Region */}
                <td className="py-3 px-4 whitespace-nowrap">
                  <span className="inline-flex items-center gap-1.5">
                    <span className="text-lg">{region.flag}</span>
                    <span className="text-slate-700 dark:text-slate-300 font-medium">
                      {journal.region_code}
                    </span>
                  </span>
                </td>

                {/* Title */}
                <td className="py-3 px-4 max-w-[200px]">
                  <span className="text-slate-800 dark:text-slate-200 font-medium truncate block">
                    {journal.title}
                  </span>
                </td>

                {/* Template */}
                <td className="py-3 px-4 whitespace-nowrap">
                  <span className="text-slate-600 dark:text-slate-400 text-xs">
                    {journal.template_name}
                  </span>
                </td>

                {/* Technician */}
                <td className="py-3 px-4 whitespace-nowrap">
                  <span className="text-slate-600 dark:text-slate-400">
                    {journal.technician_name || '—'}
                  </span>
                </td>

                {/* Period */}
                <td className="py-3 px-4 whitespace-nowrap text-xs text-slate-500 dark:text-slate-400">
                  <div>{formatDate(journal.period_start)}</div>
                  <div>{formatDate(journal.period_end)}</div>
                </td>

                {/* Status */}
                <td className="py-3 px-4 whitespace-nowrap">
                  <Badge variant={statusCfg.variant}>
                    <span className="flex items-center gap-1">
                      <StatusIcon className="w-3 h-3" />
                      {statusCfg.label}
                    </span>
                  </Badge>
                </td>

                {/* Generated At */}
                <td className="py-3 px-4 whitespace-nowrap text-xs text-slate-500 dark:text-slate-400">
                  {formatDate(journal.generated_at)}
                </td>

                {/* Actions */}
                <td className="py-3 px-4 whitespace-nowrap text-right">
                  <div className="flex items-center justify-end gap-1">
                    {journal.status === 'draft' && (
                      <Button
                        variant="ghost"
                        size="sm"
                        icon={<Sparkles className="w-3.5 h-3.5" />}
                        onClick={() => onGenerate(journal.id)}
                        aria-label="AI Suggest"
                      />
                    )}
                    {journal.status === 'generated' && (
                      <Button
                        variant="ghost"
                        size="sm"
                        icon={<Download className="w-3.5 h-3.5" />}
                        aria-label="Download PDF"
                      />
                    )}
                    <Button
                      variant="ghost"
                      size="sm"
                      icon={<FileText className="w-3.5 h-3.5" />}
                      aria-label="View Details"
                    />
                  </div>
                </td>
              </tr>
            );
          })}
        </tbody>
      </table>
    </div>
  );
}
