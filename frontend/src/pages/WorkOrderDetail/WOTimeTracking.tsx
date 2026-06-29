// ═══════════════════════════════════════════════════════════════════════
// WOTimeTracking — секция учёта времени в правой колонке
// ═══════════════════════════════════════════════════════════════════════

import React from 'react';
import {
  Timer, Play, Square, XCircle,
} from '../../components/ui/Icons';
import { Card, CardHeader, CardBody, Button, Badge } from '../../components/ui';
import type { TimeEntry } from '../../services/workOrdersApi';

interface WOTimeTrackingProps {
  timeEntries: TimeEntry[];
  elapsed: number;
  timeSubmitting: boolean;
  isEditable: boolean;
  onStartTimer: () => void;
  onPauseTimer: (entryId: string) => void;
  onResumeTimer: (entryId: string) => void;
  onStopTimer: (entryId: string) => void;
  formatDuration: (seconds: number) => string;
}

export const WOTimeTracking: React.FC<WOTimeTrackingProps> = ({
  timeEntries,
  elapsed,
  timeSubmitting,
  isEditable,
  onStartTimer,
  onPauseTimer,
  onResumeTimer,
  onStopTimer,
  formatDuration,
}) => {
  const runningEntry = timeEntries.find(e => e.status === 'running');
  const pausedEntry = timeEntries.find(e => e.status === 'paused');

  return (
    <Card>
      <CardHeader className="flex items-center gap-2">
        <Timer className="w-5 h-5 text-blue-600 dark:text-blue-400" />
        <span className="text-sm font-medium">Учёт времени</span>
      </CardHeader>
      <CardBody>
        {runningEntry || pausedEntry ? (
          <div className="text-center">
            <div className="text-2xl font-mono font-bold text-slate-900 dark:text-white tracking-wider">
              {formatDuration(elapsed)}
            </div>
            <div className="flex justify-center gap-2 mt-3">
              {runningEntry ? (
                <>
                  <Button
                    size="sm"
                    variant="secondary"
                    icon={<Timer className="w-3.5 h-3.5" />}
                    onClick={() => onPauseTimer(runningEntry.id)}
                    loading={timeSubmitting}
                  >
                    Пауза
                  </Button>
                  <Button
                    size="sm"
                    variant="danger"
                    icon={<XCircle className="w-3.5 h-3.5" />}
                    onClick={() => onStopTimer(runningEntry.id)}
                    loading={timeSubmitting}
                  >
                    Стоп
                  </Button>
                </>
              ) : pausedEntry ? (
                <>
                  <Button
                    size="sm"
                    variant="primary"
                    icon={<Play className="w-3.5 h-3.5" />}
                    onClick={() => onResumeTimer(pausedEntry.id)}
                    loading={timeSubmitting}
                  >
                    Продолжить
                  </Button>
                  <Button
                    size="sm"
                    variant="danger"
                    icon={<Square className="w-3.5 h-3.5" />}
                    onClick={() => onStopTimer(pausedEntry.id)}
                    loading={timeSubmitting}
                  >
                    Стоп
                  </Button>
                </>
              ) : null}
            </div>
          </div>
        ) : (
          <Button
            fullWidth
            size="sm"
            icon={<Play className="w-4 h-4" />}
            onClick={onStartTimer}
            loading={timeSubmitting}
            disabled={!isEditable}
          >
            Начать учёт времени
          </Button>
        )}

        {/* Recent entries */}
        {timeEntries.length > 0 && (
          <div className="mt-3 space-y-1.5">
            {[...timeEntries]
              .sort((a, b) => new Date(b.started_at).getTime() - new Date(a.started_at).getTime())
              .slice(0, 3)
              .map(entry => (
                <div key={entry.id} className="flex items-center justify-between p-2 bg-slate-50 dark:bg-slate-800/50 rounded text-xs">
                  <span className="font-mono text-slate-600 dark:text-slate-400">
                    {formatDuration(entry.total_seconds)}
                  </span>
                  <Badge variant={
                    entry.status === 'running' ? 'success' :
                    entry.status === 'paused' ? 'warning' : 'neutral'
                  } size="sm">
                    {entry.status === 'running' ? 'Активен' :
                     entry.status === 'paused' ? 'Пауза' : 'Остановлен'}
                  </Badge>
                </div>
              ))}
          </div>
        )}
      </CardBody>
    </Card>
  );
};
