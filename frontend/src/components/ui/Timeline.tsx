import React, { useState } from 'react';
import {
  Calendar, Package, CheckCircle, AlertCircle, Settings, User, Wrench,
  Camera, ChevronDown, ChevronRight, ArrowRight,
} from 'lucide-react';

// ── Diff Entry ──────────────────────────────────────────────────────────

export interface DiffEntry {
  field: string;
  oldValue?: string | number | null;
  newValue?: string | number | null;
}

// ── Timeline Event ──────────────────────────────────────────────────────

export type TimelineEventType =
  | 'status_change'
  | 'assignment'
  | 'maintenance'
  | 'part'
  | 'note'
  | 'system'
  | 'photo'
  | 'part_used';

export interface TimelineEvent {
  id: string;
  timestamp: string;
  type: TimelineEventType;
  title: string;
  description?: string;
  user?: string;
  /** Optional diff entries for change visualisation */
  diff?: DiffEntry[];
  /** Optional expandable detail content (rendered as-is) */
  details?: React.ReactNode;
}

// ── Props ───────────────────────────────────────────────────────────────

interface TimelineProps {
  events: TimelineEvent[];
  className?: string;
  /** Max items to show before collapsing (0 = no limit) */
  maxItems?: number;
}

// ── Icon & Colour Maps ─────────────────────────────────────────────────

const iconMap: Record<TimelineEventType, React.FC<{ size?: number; className?: string }>> = {
  status_change: CheckCircle,
  assignment: User,
  maintenance: Wrench,
  part: Package,
  note: Calendar,
  system: Settings,
  photo: Camera,
  part_used: Package,
};

const colorMap: Record<TimelineEventType, string> = {
  status_change: 'bg-emerald-500',
  assignment: 'bg-blue-500',
  maintenance: 'bg-amber-500',
  part: 'bg-purple-500',
  note: 'bg-slate-500',
  system: 'bg-cyan-500',
  photo: 'bg-pink-500',
  part_used: 'bg-violet-500',
};

// ── Diff Viewer ─────────────────────────────────────────────────────────

function DiffView({ diff }: { diff: DiffEntry[] }) {
  const [expanded, setExpanded] = useState(false);

  if (!diff || diff.length === 0) return null;

  const hasChanges = diff.some((d) => d.oldValue !== d.newValue);

  return (
    <div className="mt-2 border border-slate-200 dark:border-slate-700 rounded-lg overflow-hidden">
      <button
        onClick={() => setExpanded((p) => !p)}
        className="w-full flex items-center justify-between px-3 py-1.5 text-xs font-medium text-slate-600 dark:text-slate-400 bg-slate-50 dark:bg-slate-800/50 hover:bg-slate-100 dark:hover:bg-slate-700/50 transition-colors"
      >
        <span className="flex items-center gap-1.5">
          {expanded ? <ChevronDown className="w-3.5 h-3.5" /> : <ChevronRight className="w-3.5 h-3.5" />}
          {hasChanges
            ? `${diff.filter((d) => d.oldValue !== d.newValue).length} ${diff.filter((d) => d.oldValue !== d.newValue).length === 1 ? 'поле изменено' : 'полей изменено'}`
            : 'Без изменений'}
        </span>
      </button>

      {expanded && (
        <div className="divide-y divide-slate-100 dark:divide-slate-700/50 max-h-48 overflow-y-auto">
          {diff.map((entry, i) => {
            const changed = entry.oldValue !== entry.newValue;
            return (
              <div
                key={i}
                className={`grid grid-cols-[1fr_auto_1fr] gap-2 px-3 py-2 text-xs font-mono ${
                  changed ? 'bg-amber-50/50 dark:bg-amber-900/10' : ''
                }`}
              >
                {/* Field name */}
                <div className="col-span-3 text-[10px] font-semibold uppercase tracking-wider text-slate-500 dark:text-slate-500 mb-0.5">
                  {entry.field}
                </div>
                {/* Old value */}
                <div className="bg-red-50 dark:bg-red-900/20 rounded px-1.5 py-1 text-red-700 dark:text-red-400 line-through break-all min-w-0">
                  {entry.oldValue ?? <span className="text-slate-400 italic">—</span>}
                </div>
                {/* Arrow */}
                <div className="flex items-center text-slate-400">
                  <ArrowRight className="w-3 h-3" />
                </div>
                {/* New value */}
                <div className="bg-emerald-50 dark:bg-emerald-900/20 rounded px-1.5 py-1 text-emerald-700 dark:text-emerald-400 break-all min-w-0">
                  {entry.newValue ?? <span className="text-slate-400 italic">—</span>}
                </div>
              </div>
            );
          })}
        </div>
      )}
    </div>
  );
}

// ── Expandable Details ─────────────────────────────────────────────────

function ExpandableDetails({ children }: { children: React.ReactNode }) {
  const [expanded, setExpanded] = useState(false);

  return (
    <div className="mt-2">
      <button
        onClick={() => setExpanded((p) => !p)}
        className="flex items-center gap-1 text-xs text-blue-600 dark:text-blue-400 hover:text-blue-700 dark:hover:text-blue-300 transition-colors"
      >
        {expanded ? <ChevronDown className="w-3.5 h-3.5" /> : <ChevronRight className="w-3.5 h-3.5" />}
        {expanded ? 'Скрыть детали' : 'Показать детали'}
      </button>
      {expanded && (
        <div className="mt-1.5 p-3 bg-slate-50 dark:bg-slate-800/30 rounded-lg border border-slate-100 dark:border-slate-700/50 animate-fadeIn text-sm text-slate-700 dark:text-slate-300">
          {children}
        </div>
      )}
    </div>
  );
}

// ── Main Timeline Component ────────────────────────────────────────────

export function Timeline({ events, className = '', maxItems = 0 }: TimelineProps) {
  const [showAll, setShowAll] = useState(false);

  const visible = maxItems > 0 && !showAll ? events.slice(0, maxItems) : events;
  const hasMore = maxItems > 0 && events.length > maxItems;

  if (events.length === 0) {
    return (
      <p className="text-center text-slate-400 dark:text-slate-500 py-8 text-sm">Нет событий</p>
    );
  }

  return (
    <div className={`space-y-0 ${className}`}>
      {visible.map((event, index) => {
        const Icon = iconMap[event.type] || AlertCircle;
        const dotColor = colorMap[event.type] || 'bg-slate-500';
        const isLast = index === visible.length - 1;

        return (
          <div key={event.id} className="relative flex gap-4 pb-6">
            {!isLast && (
              <div className="absolute left-[15px] top-8 bottom-0 w-0.5 bg-slate-200 dark:bg-slate-700" />
            )}
            {/* Dot with Icon */}
            <div
              className={`relative z-10 flex items-center justify-center w-8 h-8 rounded-full ${dotColor} shrink-0 shadow-sm`}
            >
              <Icon size={14} className="text-white" />
            </div>
            {/* Content */}
            <div className="flex-1 min-w-0 pt-0.5">
              <div className="flex items-center gap-2 flex-wrap">
                <p className="text-sm font-medium text-slate-900 dark:text-white">{event.title}</p>
                {event.user && (
                  <span className="inline-flex items-center gap-1 text-xs text-slate-500 dark:text-slate-400 bg-slate-100 dark:bg-slate-800 rounded-full px-2 py-0.5">
                    <User size={10} />
                    {event.user}
                  </span>
                )}
              </div>
              {event.description && (
                <p className="text-sm text-slate-600 dark:text-slate-400 mt-0.5">{event.description}</p>
              )}
              {/* Diff view */}
              {event.diff && event.diff.length > 0 && <DiffView diff={event.diff} />}
              {/* Expandable details */}
              {event.details && <ExpandableDetails>{event.details}</ExpandableDetails>}
              {/* Timestamp */}
              <p className="text-xs text-slate-400 dark:text-slate-500 mt-1">
                {new Date(event.timestamp).toLocaleString()}
              </p>
            </div>
          </div>
        );
      })}

      {/* Show more / less */}
      {hasMore && (
        <button
          onClick={() => setShowAll((p) => !p)}
          className="flex items-center gap-1.5 text-xs font-medium text-blue-600 dark:text-blue-400 hover:text-blue-700 dark:hover:text-blue-300 transition-colors ml-12"
        >
          {showAll ? (
            <>Скрыть <ChevronDown className="w-3.5 h-3.5" /></>
          ) : (
          <>Показать ещё {events.length - maxItems} <ChevronRight className="w-3.5 h-3.5" /></>
          )}
        </button>
      )}
    </div>
  );
}
