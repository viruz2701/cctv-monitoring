import React, { useState } from 'react';
import { useTranslation } from 'react-i18next';
import { Card, Badge, Button } from '../components/ui';
import {
  Phone, Calendar, Users, Clock, Plus,
  ChevronLeft, ChevronRight, RefreshCw,
} from '../components/ui/Icons';

interface OnCallShift {
  id: string;
  technician: string;
  role: string;
  date: string;
  startTime: string;
  endTime: string;
  type: 'primary' | 'secondary' | 'backup';
}

const MOCK_SHIFTS: OnCallShift[] = [
  { id: '1', technician: 'Иван Петров', role: 'Senior Tech', date: '2026-06-25', startTime: '09:00', endTime: '18:00', type: 'primary' },
  { id: '2', technician: 'Мария Иванова', role: 'Tech', date: '2026-06-25', startTime: '18:00', endTime: '09:00', type: 'primary' },
  { id: '3', technician: 'Алексей Смирнов', role: 'Junior Tech', date: '2026-06-25', startTime: '09:00', endTime: '18:00', type: 'backup' },
  { id: '4', technician: 'Иван Петров', role: 'Senior Tech', date: '2026-06-26', startTime: '09:00', endTime: '18:00', type: 'primary' },
  { id: '5', technician: 'Дмитрий Козлов', role: 'Tech', date: '2026-06-26', startTime: '18:00', endTime: '09:00', type: 'primary' },
  { id: '6', technician: 'Ольга Новикова', role: 'Tech', date: '2026-06-27', startTime: '09:00', endTime: '18:00', type: 'primary' },
  { id: '7', technician: 'Мария Иванова', role: 'Tech', date: '2026-06-27', startTime: '18:00', endTime: '09:00', type: 'primary' },
  { id: '8', technician: 'Алексей Смирнов', role: 'Junior Tech', date: '2026-06-27', startTime: '09:00', endTime: '18:00', type: 'secondary' },
  { id: '9', technician: 'Дмитрий Козлов', role: 'Tech', date: '2026-06-28', startTime: '09:00', endTime: '18:00', type: 'primary' },
  { id: '10', technician: 'Ольга Новикова', role: 'Tech', date: '2026-06-28', startTime: '18:00', endTime: '09:00', type: 'primary' },
  { id: '11', technician: 'Иван Петров', role: 'Senior Tech', date: '2026-06-29', startTime: '09:00', endTime: '18:00', type: 'primary' },
  { id: '12', technician: 'Алексей Смирнов', role: 'Junior Tech', date: '2026-06-29', startTime: '18:00', endTime: '09:00', type: 'backup' },
  { id: '13', technician: 'Мария Иванова', role: 'Tech', date: '2026-06-30', startTime: '09:00', endTime: '18:00', type: 'primary' },
  { id: '14', technician: 'Дмитрий Козлов', role: 'Tech', date: '2026-06-30', startTime: '18:00', endTime: '09:00', type: 'primary' },
  { id: '15', technician: 'Ольга Новикова', role: 'Tech', date: '2026-07-01', startTime: '09:00', endTime: '18:00', type: 'primary' },
];

const TECHNICIANS = ['Иван Петров', 'Мария Иванова', 'Алексей Смирнов', 'Дмитрий Козлов', 'Ольга Новикова'];

const SHIFT_TYPE_STYLES: Record<string, { bg: string; text: string; label: string }> = {
  primary: { bg: 'bg-blue-100 border-blue-300', text: 'text-blue-800', label: 'Primary' },
  secondary: { bg: 'bg-emerald-100 border-emerald-300', text: 'text-emerald-800', label: 'Secondary' },
  backup: { bg: 'bg-amber-100 border-amber-300', text: 'text-amber-800', label: 'Backup' },
};

function getWeekDays(from: Date): Date[] {
  const days: Date[] = [];
  const start = new Date(from);
  start.setDate(start.getDate() - start.getDay() + 1);
  for (let i = 0; i < 7; i++) {
    const d = new Date(start);
    d.setDate(d.getDate() + i);
    days.push(d);
  }
  return days;
}

function formatDate(d: Date): string { return d.toISOString().split('T')[0]; }
function formatDay(d: Date): string {
  return d.toLocaleDateString('ru-RU', { weekday: 'short', day: 'numeric', month: 'short' });
}
function isToday(d: Date): boolean { return formatDate(d) === formatDate(new Date()); }

export function OnCallSchedule() {
  const { t } = useTranslation();
  const [weekStart, setWeekStart] = useState(() => {
    const d = new Date(); d.setDate(d.getDate() - d.getDay() + 1); return d;
  });
  const [shifts] = useState<OnCallShift[]>(MOCK_SHIFTS);
  const [filter, setFilter] = useState<string>('all');

  const days = getWeekDays(weekStart);
  const dateStrings = days.map(formatDate);

  const goPrev = () => { const d = new Date(weekStart); d.setDate(d.getDate() - 7); setWeekStart(d); };
  const goNext = () => { const d = new Date(weekStart); d.setDate(d.getDate() + 7); setWeekStart(d); };

  const filteredTechs = filter === 'all' ? TECHNICIANS : TECHNICIANS.filter(t => t === filter);

  return (
    <div className="space-y-6">
      <div className="flex items-center justify-between">
        <div>
          <h1 className="text-2xl font-bold text-slate-900 flex items-center gap-2">
            <Phone className="w-6 h-6" />
            {t('on_call_schedule') || 'График дежурств'}
          </h1>
          <p className="text-sm text-slate-500 mt-1">
            {t('on_call_desc') || 'Расписание дежурств техников'}
          </p>
        </div>
        <div className="flex items-center gap-2">
          <Button variant="outline" size="sm" icon={<Plus className="w-4 h-4" />}>
            {t('add_shift') || 'Добавить'}
          </Button>
          <Button variant="outline" size="sm" icon={<RefreshCw className="w-4 h-4" />}>
            {t('refresh') || 'Обновить'}
          </Button>
        </div>
      </div>

      {/* Legend */}
      <div className="flex items-center gap-4 text-xs">
        <span className="flex items-center gap-1.5">
          <span className="w-3 h-3 rounded-sm bg-blue-100 border border-blue-300" /> Primary
        </span>
        <span className="flex items-center gap-1.5">
          <span className="w-3 h-3 rounded-sm bg-emerald-100 border border-emerald-300" /> Secondary
        </span>
        <span className="flex items-center gap-1.5">
          <span className="w-3 h-3 rounded-sm bg-amber-100 border border-amber-300" /> Backup
        </span>
      </div>

      {/* Week navigation */}
      <div className="flex items-center justify-between">
        <button onClick={goPrev} className="p-2 rounded-lg hover:bg-slate-100">
          <ChevronLeft className="w-5 h-5 text-slate-600" />
        </button>
        <span className="text-sm font-semibold text-slate-900">
          {formatDay(days[0])} — {formatDay(days[6])}
        </span>
        <button onClick={goNext} className="p-2 rounded-lg hover:bg-slate-100">
          <ChevronRight className="w-5 h-5 text-slate-600" />
        </button>
      </div>

      {/* Schedule grid */}
      <Card>
        <div className="overflow-x-auto">
          <table className="w-full text-xs">
            <thead>
              <tr>
                <th className="text-left py-3 px-3 font-semibold text-slate-500 uppercase sticky left-0 bg-white z-10 min-w-[120px]">
                  {t('technician') || 'Техник'}
                </th>
                {days.map((d, i) => (
                  <th key={i} className={`text-center py-3 px-2 font-semibold uppercase min-w-[100px] ${
                    isToday(d) ? 'bg-blue-50 text-blue-700' : 'text-slate-500'
                  }`}>
                    <div>{d.toLocaleDateString('ru-RU', { weekday: 'short' })}</div>
                    <div className="text-lg font-bold">{d.getDate()}</div>
                  </th>
                ))}
              </tr>
            </thead>
            <tbody>
              {filteredTechs.map((tech) => (
                <tr key={tech} className="border-t border-slate-100">
                  <td className="py-3 px-3 font-medium text-slate-700 sticky left-0 bg-white">
                    {tech}
                  </td>
                  {dateStrings.map((ds, di) => {
                    const dayShifts = shifts.filter(s => s.technician === tech && s.date === ds);
                    return (
                      <td key={di} className={`text-center py-2 px-1 ${isToday(days[di]) ? 'bg-blue-50/50' : ''}`}>
                        {dayShifts.length > 0 ? (
                          <div className="flex flex-col gap-1">
                            {dayShifts.map(s => {
                              const style = SHIFT_TYPE_STYLES[s.type];
                              return (
                                <div key={s.id} className={`px-1.5 py-1 rounded border ${style.bg} ${style.text}`}>
                                  <div className="font-medium text-[10px]">{s.startTime}-{s.endTime}</div>
                                  <div className="text-[9px] opacity-75">{style.label}</div>
                                </div>
                              );
                            })}
                          </div>
                        ) : (
                          <span className="text-slate-300">—</span>
                        )}
                      </td>
                    );
                  })}
                </tr>
              ))}
            </tbody>
          </table>
        </div>
      </Card>

      {/* Upcoming on-call summary */}
      <Card>
        <div className="p-4">
          <h3 className="text-sm font-semibold text-slate-900 mb-3 flex items-center gap-2">
            <Clock className="w-4 h-4" />
            {t('upcoming_shifts') || 'Ближайшие дежурства'}
          </h3>
          <div className="space-y-2">
            {shifts.slice(0, 5).map(s => (
              <div key={s.id} className="flex items-center justify-between p-3 bg-slate-50 rounded-lg">
                <div className="flex items-center gap-3">
                  <div className={`w-2 h-2 rounded-full ${
                    s.type === 'primary' ? 'bg-blue-500' : s.type === 'secondary' ? 'bg-emerald-500' : 'bg-amber-500'
                  }`} />
                  <div>
                    <p className="text-sm font-medium text-slate-900">{s.technician}</p>
                    <p className="text-xs text-slate-500">{s.role}</p>
                  </div>
                </div>
                <div className="text-right">
                  <p className="text-xs font-medium text-slate-700">{s.date}</p>
                  <p className="text-[10px] text-slate-400">{s.startTime} — {s.endTime}</p>
                </div>
              </div>
            ))}
          </div>
        </div>
      </Card>
    </div>
  );
}
