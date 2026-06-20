import React from 'react';
import { Clock, AlertTriangle, CheckCircle } from 'lucide-react';

interface SLAProgressProps {
  deadline: string;
  createdAt: string;
  status?: 'on_track' | 'at_risk' | 'breached' | 'completed';
  className?: string;
}

export function SLAProgress({ deadline, createdAt, status, className = '' }: SLAProgressProps) {
  const now = new Date();
  const deadlineDate = new Date(deadline);
  const createdDate = new Date(createdAt);
  const totalMs = deadlineDate.getTime() - createdDate.getTime();
  const elapsedMs = now.getTime() - createdDate.getTime();
  const remainingMs = deadlineDate.getTime() - now.getTime();

  const pct = totalMs > 0 ? Math.min(100, Math.max(0, (elapsedMs / totalMs) * 100)) : 0;
  const isOverdue = remainingMs < 0;

  const resolvedStatus = status || (isOverdue ? 'breached' : pct > 75 ? 'at_risk' : 'on_track');

  const statusConfig = {
    on_track: {
      color: 'bg-emerald-500',
      bg: 'bg-emerald-100 dark:bg-emerald-900/30',
      icon: <CheckCircle size={14} className="text-emerald-600 dark:text-emerald-400" />,
      text: 'В срок',
      textColor: 'text-emerald-600 dark:text-emerald-400',
    },
    at_risk: {
      color: 'bg-amber-500',
      bg: 'bg-amber-100 dark:bg-amber-900/30',
      icon: <Clock size={14} className="text-amber-600 dark:text-amber-400" />,
      text: 'Под риском',
      textColor: 'text-amber-600 dark:text-amber-400',
    },
    breached: {
      color: 'bg-red-500',
      bg: 'bg-red-100 dark:bg-red-900/30',
      icon: <AlertTriangle size={14} className="text-red-600 dark:text-red-400" />,
      text: 'Просрочен',
      textColor: 'text-red-600 dark:text-red-400',
    },
    completed: {
      color: 'bg-blue-500',
      bg: 'bg-blue-100 dark:bg-blue-900/30',
      icon: <CheckCircle size={14} className="text-blue-600 dark:text-blue-400" />,
      text: 'Завершён',
      textColor: 'text-blue-600 dark:text-blue-400',
    },
  };

  const config = statusConfig[resolvedStatus];

  const formatTimeLeft = (ms: number): string => {
    if (ms <= 0) return 'Просрочено';
    const hours = Math.floor(ms / (1000 * 60 * 60));
    const minutes = Math.floor((ms % (1000 * 60 * 60)) / (1000 * 60));
    if (hours > 24) {
      const days = Math.floor(hours / 24);
      return `${days}д ${hours % 24}ч`;
    }
    return `${hours}ч ${minutes}м`;
  };

  return (
    <div className={`space-y-2 ${className}`}>
      <div className="flex items-center justify-between">
        <div className={`flex items-center gap-1.5 px-2 py-0.5 rounded-full text-xs font-medium ${config.bg} ${config.textColor}`}>
          {config.icon}
          {config.text}
        </div>
        {resolvedStatus !== 'completed' && (
          <span className="text-xs text-slate-500 dark:text-slate-400">
            {isOverdue ? 'Просрочено' : formatTimeLeft(remainingMs)}
          </span>
        )}
      </div>

      <div className="w-full h-2 bg-slate-200 dark:bg-slate-700 rounded-full overflow-hidden">
        <div
          className={`h-full rounded-full transition-all duration-500 ${config.color}`}
          style={{ width: `${Math.min(100, pct)}%` }}
        />
      </div>

      <div className="flex justify-between text-xs text-slate-400 dark:text-slate-500">
        <span>{createdDate.toLocaleDateString()}</span>
        <span>{deadlineDate.toLocaleDateString()}</span>
      </div>
    </div>
  );
}