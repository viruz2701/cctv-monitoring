import React, { useEffect, useState } from 'react';
import { Clock, AlertTriangle, CheckCircle, Hourglass } from '../ui/Icons';

interface SLATimerProps {
  deadline: string;
  createdAt: string;
  status?: 'on_track' | 'at_risk' | 'breached' | 'completed' | 'no_sla';
  className?: string;
}

type SLAStatus = 'on_track' | 'at_risk' | 'breached' | 'completed' | 'no_sla';

interface SLAStatusStyle {
  dotColor: string;
  barColor: string;
  bg: string;
  border: string;
  text: string;
  label: string;
  icon: React.FC<{ className?: string }>;
  pulse: boolean;
}

/**
 * SLATimer — countdown timer showing remaining time until SLA deadline.
 *
 * Visual states:
 *  🟢 on_track  — emerald (≥25% time remaining)
 *  🟡 at_risk   — amber  (<25% time remaining)
 *  🔴 breached  — red    (deadline passed)
 *  🔵 completed — blue   (SLA fulfilled)
 *
 * Pulse animation triggers when < 1 hour remains until breach.
 */
export const SLATimer: React.FC<SLATimerProps> = ({
  deadline,
  createdAt,
  status,
  className = '',
}) => {
  const [now, setNow] = useState(new Date());

  // Live countdown every second
  useEffect(() => {
    if (status === 'completed' || status === 'breached' || status === 'no_sla') return;
    const interval = setInterval(() => setNow(new Date()), 1000);
    return () => clearInterval(interval);
  }, [status]);

  const deadlineDate = new Date(deadline);
  const createdDate = new Date(createdAt);
  const totalMs = deadlineDate.getTime() - createdDate.getTime();
  const elapsedMs = now.getTime() - createdDate.getTime();
  const remainingMs = deadlineDate.getTime() - now.getTime();

  // Compliance % (how much of SLA window has been used)
  const pctUsed = totalMs > 0 ? Math.min(100, Math.max(0, (elapsedMs / totalMs) * 100)) : 0;
  const pctRemaining = 100 - pctUsed;

  const isOverdue = remainingMs < 0;
  const isLessThanOneHour = remainingMs > 0 && remainingMs < 3600000;

  // Resolve effective status — narrow type for config lookup
  const rawStatus: SLAStatus = (() => {
    if (status && status !== 'no_sla') return status;
    if (status === 'no_sla') return 'no_sla';
    if (isOverdue) return 'breached';
    if (pctRemaining < 25) return 'at_risk';
    return 'on_track';
  })();

  // ── Status config ────────────────────────────────────────────────

  const statusConfig: Record<SLAStatus, SLAStatusStyle> = {
    on_track: {
      dotColor: 'bg-emerald-500',
      barColor: 'bg-emerald-500',
      bg: 'bg-emerald-50 dark:bg-emerald-900/20',
      border: 'border-emerald-200 dark:border-emerald-800',
      text: 'text-emerald-700 dark:text-emerald-300',
      label: 'В срок',
      icon: CheckCircle,
      pulse: false,
    },
    at_risk: {
      dotColor: 'bg-amber-500',
      barColor: 'bg-amber-500',
      bg: 'bg-amber-50 dark:bg-amber-900/20',
      border: 'border-amber-200 dark:border-amber-800',
      text: 'text-amber-700 dark:text-amber-300',
      label: 'Под риском',
      icon: Clock,
      pulse: isLessThanOneHour,
    },
    breached: {
      dotColor: 'bg-red-500',
      barColor: 'bg-red-500',
      bg: 'bg-red-50 dark:bg-red-900/20',
      border: 'border-red-200 dark:border-red-800',
      text: 'text-red-700 dark:text-red-300',
      label: 'Просрочен',
      icon: AlertTriangle,
      pulse: false,
    },
    completed: {
      dotColor: 'bg-blue-500',
      barColor: 'bg-blue-500',
      bg: 'bg-blue-50 dark:bg-blue-900/20',
      border: 'border-blue-200 dark:border-blue-800',
      text: 'text-blue-700 dark:text-blue-300',
      label: 'Выполнен',
      icon: CheckCircle,
      pulse: false,
    },
    no_sla: {
      dotColor: 'bg-slate-400',
      barColor: 'bg-slate-400',
      bg: 'bg-slate-50 dark:bg-slate-800/50',
      border: 'border-slate-200 dark:border-slate-700',
      text: 'text-slate-500 dark:text-slate-400',
      label: 'Без SLA',
      icon: Hourglass,
      pulse: false,
    },
  };

  const config = statusConfig[rawStatus];
  const Icon = config.icon;

  // ── Formatting helpers ───────────────────────────────────────────

  const formatTimeLeft = (ms: number): string => {
    if (ms <= 0) return 'Просрочено';
    const totalSeconds = Math.floor(ms / 1000);
    const hours = Math.floor(totalSeconds / 3600);
    const minutes = Math.floor((totalSeconds % 3600) / 60);
    const seconds = totalSeconds % 60;

    if (hours > 48) {
      const days = Math.floor(hours / 24);
      return `${days}д ${hours % 24}ч`;
    }
    if (hours > 0) {
      return `${hours}ч ${minutes}м ${seconds}с`;
    }
    return `${minutes}м ${seconds}с`;
  };

  return (
    <div className={`rounded-lg border ${config.border} ${config.bg} p-3 ${className}`}>
      {/* ═══ Header row ═══ */}
      <div className="flex items-center justify-between mb-2">
        <div className={`flex items-center gap-1.5 text-sm font-medium ${config.text}`}>
          <Icon className="w-4 h-4" />
          <span>SLA: {config.label}</span>
        </div>

        {rawStatus !== 'completed' && rawStatus !== 'no_sla' && (
          <span className={`text-xs font-mono tabular-nums ${config.text}`}>
            {isOverdue ? 'Просрочено' : formatTimeLeft(remainingMs)}
          </span>
        )}
      </div>

      {/* ═══ Progress bar ═══ */}
      <div className="relative h-2.5 bg-slate-200 dark:bg-slate-700 rounded-full overflow-hidden">
        <div
          className={`h-full rounded-full transition-all duration-1000 ease-linear ${config.barColor}`}
          style={{ width: `${Math.min(100, pctUsed)}%` }}
        />
        {/* Pulse overlay for at_risk + <1h */}
        {config.pulse && (
          <div className="absolute inset-0 rounded-full bg-amber-400/20 animate-pulse" />
        )}
      </div>

      {/* ═══ Stats row ═══ */}
      <div className="flex items-center justify-between mt-1.5 text-[10px] text-slate-400 dark:text-slate-500">
        <span>
          Создан: {createdDate.toLocaleDateString()} {createdDate.toLocaleTimeString([], { hour: '2-digit', minute: '2-digit' })}
        </span>
        <span>
          Дедлайн: {deadlineDate.toLocaleDateString()} {deadlineDate.toLocaleTimeString([], { hour: '2-digit', minute: '2-digit' })}
        </span>
      </div>

      {/* ═══ Compliance % ═══ */}
      {rawStatus !== 'no_sla' && (
        <div className="mt-1.5 flex items-center justify-between text-[10px]">
          <span className="text-slate-400 dark:text-slate-500">
            Compliance: {pctRemaining.toFixed(0)}%
          </span>
          {rawStatus === 'breached' && (
            <span className="text-red-500 font-medium">
              Превышение: {formatTimeLeft(Math.abs(remainingMs))}
            </span>
          )}
        </div>
      )}
    </div>
  );
};

export default SLATimer;
