import React, { useEffect, useState } from 'react';
import { AlertTriangle, Clock, CheckCircle } from 'lucide-react';

interface LiveSLATimerProps {
  deadline: string;
  createdAt: string;
  status?: 'on_track' | 'at_risk' | 'breached' | 'completed';
  className?: string;
}

const statusConfig = {
  on_track: {
    color: 'bg-emerald-500',
    bg: 'bg-emerald-50 dark:bg-emerald-900/20',
    text: 'emerald-600 dark:text-emerald-400',
    border: 'border-emerald-200 dark:border-emerald-800',
    icon: CheckCircle,
    label: 'В срок',
  },
  at_risk: {
    color: 'bg-amber-500',
    bg: 'bg-amber-50 dark:bg-amber-900/20',
    text: 'amber-600 dark:text-amber-400',
    border: 'border-amber-200 dark:border-amber-800',
    icon: Clock,
    label: 'Под риском',
  },
  breached: {
    color: 'bg-red-500',
    bg: 'bg-red-50 dark:bg-red-900/20',
    text: 'red-600 dark:text-red-400',
    border: 'border-red-200 dark:border-red-800',
    icon: AlertTriangle,
    label: 'Просрочен',
  },
  completed: {
    color: 'bg-blue-500',
    bg: 'bg-blue-50 dark:bg-blue-900/20',
    text: 'blue-600 dark:text-blue-400',
    border: 'border-blue-200 dark:border-blue-800',
    icon: CheckCircle,
    label: 'Завершён',
  },
};

export function LiveSLATimer({ deadline, createdAt, status, className = '' }: LiveSLATimerProps) {
  const [now, setNow] = useState(new Date());

  // Live update every second
  useEffect(() => {
    if (status === 'completed' || status === 'breached') return;
    const interval = setInterval(() => setNow(new Date()), 1000);
    return () => clearInterval(interval);
  }, [status]);

  const deadlineDate = new Date(deadline);
  const createdDate = new Date(createdAt);
  const totalMs = deadlineDate.getTime() - createdDate.getTime();
  const elapsedMs = now.getTime() - createdDate.getTime();
  const remainingMs = deadlineDate.getTime() - now.getTime();

  const pct = totalMs > 0 ? Math.min(100, Math.max(0, (elapsedMs / totalMs) * 100)) : 0;
  const isOverdue = remainingMs < 0;

  const resolvedStatus = status || (isOverdue ? 'breached' : pct > 75 ? 'at_risk' : 'on_track');
  const config = statusConfig[resolvedStatus];
  const Icon = config.icon;

  const formatTimeLeft = (ms: number): string => {
    if (ms <= 0) return 'Просрочено';
    const totalSeconds = Math.floor(ms / 1000);
    const hours = Math.floor(totalSeconds / 3600);
    const minutes = Math.floor((totalSeconds % 3600) / 60);
    const seconds = totalSeconds % 60;
    if (hours > 24) {
      const days = Math.floor(hours / 24);
      return `${days}д ${hours % 24}ч ${minutes}м`;
    }
    return `${hours}ч ${minutes}м ${seconds}с`;
  };

  return (
    <div className={`rounded-lg border ${config.border} ${config.bg} p-3 ${className}`}>
      {/* Status header */}
      <div className="flex items-center justify-between mb-2">
        <div className={`flex items-center gap-1.5 text-sm font-medium text-${config.text}`}>
          <Icon className="w-4 h-4" />
          <span>SLA: {config.label}</span>
        </div>
        {resolvedStatus !== 'completed' && (
          <span className={`text-xs font-mono tabular-nums text-${config.text}`}>
            {isOverdue ? 'Просрочено' : formatTimeLeft(remainingMs)}
          </span>
        )}
      </div>

      {/* Progress bar */}
      <div className="relative h-2.5 bg-slate-200 dark:bg-slate-700 rounded-full overflow-hidden">
        <div
          className={`h-full rounded-full transition-all duration-1000 ease-linear ${config.color}`}
          style={{ width: `${Math.min(100, pct)}%` }}
        />
        {/* Pulse animation for at_risk */}
        {resolvedStatus === 'at_risk' && (
          <div className="absolute inset-0 rounded-full bg-amber-400/20 animate-pulse" />
        )}
      </div>

      {/* Time range */}
      <div className="flex justify-between mt-1.5">
        <span className="text-[10px] text-slate-400 dark:text-slate-500">
          {createdDate.toLocaleDateString()} {createdDate.toLocaleTimeString([], { hour: '2-digit', minute: '2-digit' })}
        </span>
        <span className="text-[10px] text-slate-400 dark:text-slate-500">
          {deadlineDate.toLocaleDateString()} {deadlineDate.toLocaleTimeString([], { hour: '2-digit', minute: '2-digit' })}
        </span>
      </div>
    </div>
  );
}
