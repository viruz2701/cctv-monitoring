import React from 'react';
import { Calendar, Package, CheckCircle, AlertCircle, Settings, User, Wrench } from 'lucide-react';

export interface TimelineEvent {
  id: string;
  timestamp: string;
  type: 'status_change' | 'assignment' | 'maintenance' | 'part' | 'note' | 'system';
  title: string;
  description?: string;
  user?: string;
}

interface TimelineProps {
  events: TimelineEvent[];
  className?: string;
}

const iconMap = {
  status_change: CheckCircle,
  assignment: User,
  maintenance: Wrench,
  part: Package,
  note: Calendar,
  system: Settings,
};

const colorMap = {
  status_change: 'bg-emerald-500',
  assignment: 'bg-blue-500',
  maintenance: 'bg-amber-500',
  part: 'bg-purple-500',
  note: 'bg-slate-500',
  system: 'bg-cyan-500',
};

export function Timeline({ events, className = '' }: TimelineProps) {
  return (
    <div className={`space-y-0 ${className}`}>
      {events.map((event, index) => {
        const Icon = iconMap[event.type] || AlertCircle;
        const dotColor = colorMap[event.type] || 'bg-slate-500';
        const isLast = index === events.length - 1;

        return (
          <div key={event.id} className="relative flex gap-4 pb-6">
            {!isLast && (
              <div className="absolute left-[15px] top-8 bottom-0 w-0.5 bg-slate-200 dark:bg-slate-700" />
            )}
            <div
              className={`relative z-10 flex items-center justify-center w-8 h-8 rounded-full ${dotColor} shrink-0`}
            >
              <Icon size={14} className="text-white" />
            </div>
            <div className="flex-1 min-w-0 pt-0.5">
              <div className="flex items-center gap-2 flex-wrap">
                <p className="text-sm font-medium text-slate-900 dark:text-white">{event.title}</p>
                {event.user && (
                  <span className="text-xs text-slate-500 dark:text-slate-400">{event.user}</span>
                )}
              </div>
              {event.description && (
                <p className="text-sm text-slate-600 dark:text-slate-400 mt-0.5">{event.description}</p>
              )}
              <p className="text-xs text-slate-400 dark:text-slate-500 mt-1">
                {new Date(event.timestamp).toLocaleString()}
              </p>
            </div>
          </div>
        );
      })}
      {events.length === 0 && (
        <p className="text-center text-slate-400 dark:text-slate-500 py-8 text-sm">Нет событий</p>
      )}
    </div>
  );
}