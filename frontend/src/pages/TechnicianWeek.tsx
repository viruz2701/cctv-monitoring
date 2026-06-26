// TechnicianWeek — календарь загрузки техников (Week view).
//
// P2-2.3: Resource Planning Calendar
//   - Week view с technician rows
//   - Drag-and-drop WO assignment
//   - Conflict detection

import React, { useMemo, useState } from 'react';
import { useTranslation } from 'react-i18next';
import { ChevronLeft, ChevronRight, User, Clock, AlertCircle } from 'lucide-react';

interface Technician {
    id: string;
    name: string;
    role: string;
    avatar?: string;
}

interface WorkOrderSlot {
    id: string;
    title: string;
    technicianId: string;
    day: number; // 0-6
    startHour: number; // 0-23
    duration: number; // hours
    priority: 'critical' | 'high' | 'medium' | 'low';
    status: 'scheduled' | 'in_progress' | 'completed';
}

const HOURS = Array.from({ length: 12 }, (_, i) => i + 7); // 7:00 - 18:00
const DAYS = ['monday', 'tuesday', 'wednesday', 'thursday', 'friday', 'saturday', 'sunday'];

const priorityColors: Record<string, string> = {
    critical: 'bg-red-500 border-red-600',
    high: 'bg-amber-500 border-amber-600',
    medium: 'bg-blue-500 border-blue-600',
    low: 'bg-slate-400 border-slate-500',
};

export function TechnicianWeek() {
    const { t } = useTranslation();
    const [currentWeek, setCurrentWeek] = useState(0);

    // Mock data
    const technicians: Technician[] = [
        { id: 'tech-1', name: 'Ivan Petrov', role: 'Senior Technician' },
        { id: 'tech-2', name: 'Anna Sidorova', role: 'Technician' },
        { id: 'tech-3', name: 'Pavel Volkov', role: 'Junior Technician' },
    ];

    const slots: WorkOrderSlot[] = [
        { id: 'wo-1', title: 'Replace DVR-03', technicianId: 'tech-1', day: 0, startHour: 8, duration: 3, priority: 'high', status: 'scheduled' },
        { id: 'wo-2', title: 'Camera calibration', technicianId: 'tech-1', day: 0, startHour: 13, duration: 2, priority: 'medium', status: 'scheduled' },
        { id: 'wo-3', title: 'Emergency: NVR-07', technicianId: 'tech-2', day: 0, startHour: 9, duration: 4, priority: 'critical', status: 'in_progress' },
    ];

    const weekLabel = useMemo(() => {
        const d = new Date();
        d.setDate(d.getDate() + currentWeek * 7);
        const monday = new Date(d.setDate(d.getDate() - d.getDay() + 1));
        const friday = new Date(monday);
        friday.setDate(friday.getDate() + 4);
        return `${monday.toLocaleDateString()} - ${friday.toLocaleDateString()}`;
    }, [currentWeek]);

    return (
        <div className="p-4 md:p-6 space-y-4">
            {/* Header */}
            <div className="flex items-center justify-between">
                <div>
                    <h1 className="text-2xl font-bold text-slate-900 dark:text-white">
                        {t('resource_planning') || 'Resource Planning'}
                    </h1>
                    <p className="text-sm text-slate-500 dark:text-slate-400">
                        {t('technician_schedule') || 'Technician schedule'}
                    </p>
                </div>
                <div className="flex items-center gap-2">
                    <button onClick={() => setCurrentWeek(w => w - 1)} className="p-2 hover:bg-slate-100 dark:hover:bg-slate-800 rounded-lg">
                        <ChevronLeft className="w-5 h-5 text-slate-600 dark:text-slate-400" />
                    </button>
                    <span className="text-sm font-medium text-slate-700 dark:text-slate-300 min-w-[200px] text-center">{weekLabel}</span>
                    <button onClick={() => setCurrentWeek(w => w + 1)} className="p-2 hover:bg-slate-100 dark:hover:bg-slate-800 rounded-lg">
                        <ChevronRight className="w-5 h-5 text-slate-600 dark:text-slate-400" />
                    </button>
                    <button onClick={() => setCurrentWeek(0)} className="ml-2 px-3 py-1.5 text-sm bg-blue-600 text-white rounded-lg hover:bg-blue-700">
                        {t('today') || 'Today'}
                    </button>
                </div>
            </div>

            {/* Calendar Grid */}
            <div className="bg-white dark:bg-slate-800 rounded-xl border border-slate-200 dark:border-slate-700 overflow-hidden">
                {/* Day Headers */}
                <div className="grid grid-cols-[200px_repeat(7,1fr)] border-b border-slate-200 dark:border-slate-700">
                    <div className="p-3 text-xs font-semibold text-slate-400 uppercase border-r border-slate-200 dark:border-slate-700">
                        {t('technician') || 'Technician'}
                    </div>
                    {DAYS.map((day, i) => (
                        <div key={day} className={`p-3 text-center text-xs font-semibold uppercase ${i < 5 ? 'text-slate-600 dark:text-slate-400' : 'text-amber-600 dark:text-amber-400'}`}>
                            {t(day)}
                            <br />
                            <span className="text-[10px] font-normal">{new Date(2026, 5, 22 + i + currentWeek * 7).getDate()}</span>
                        </div>
                    ))}
                </div>

                {/* Technician Rows */}
                {technicians.map((tech) => (
                    <div key={tech.id} className="grid grid-cols-[200px_repeat(7,1fr)] border-b border-slate-100 dark:border-slate-700/50 last:border-b-0">
                        {/* Technician Info */}
                        <div className="p-3 flex items-center gap-3 border-r border-slate-100 dark:border-slate-700/50">
                            <div className="w-8 h-8 bg-blue-100 dark:bg-blue-900/30 rounded-full flex items-center justify-center">
                                <User className="w-4 h-4 text-blue-600 dark:text-blue-400" />
                            </div>
                            <div className="min-w-0">
                                <p className="text-sm font-medium text-slate-900 dark:text-white truncate">{tech.name}</p>
                                <p className="text-xs text-slate-400">{tech.role}</p>
                            </div>
                        </div>

                        {/* Day Columns */}
                        {Array.from({ length: 7 }, (_, dayIdx) => {
                            const daySlots = slots.filter(s => s.technicianId === tech.id && s.day === dayIdx);
                            const maxHours = 8;
                            const usedHours = daySlots.reduce((acc, s) => acc + s.duration, 0);
                            const isOverbooked = usedHours > maxHours;

                            return (
                                <div key={dayIdx} className="relative p-1.5 min-h-[80px] border-r border-slate-100 dark:border-slate-700/50 last:border-r-0">
                                    {/* Work Order Slots */}
                                    {daySlots.map((slot) => (
                                        <div
                                            key={slot.id}
                                            draggable
                                            className={`text-[10px] p-1.5 mb-1 rounded border-l-2 text-white cursor-pointer transition-shadow hover:shadow-md ${priorityColors[slot.priority]}`}
                                        >
                                            <p className="font-medium truncate">{slot.title}</p>
                                            <p className="opacity-80">{slot.startHour}:00 ({slot.duration}h)</p>
                                        </div>
                                    ))}

                                    {/* Overbooked Warning */}
                                    {isOverbooked && (
                                        <div className="flex items-center gap-1 text-[10px] text-red-500 mt-1">
                                            <AlertCircle className="w-3 h-3" />
                                            <span>{usedHours}h/{maxHours}h</span>
                                        </div>
                                    )}

                                    {/* Utilization Bar */}
                                    <div className="absolute bottom-0 left-0 right-0 h-1 bg-slate-100 dark:bg-slate-700">
                                        <div
                                            className={`h-full transition-all ${isOverbooked ? 'bg-red-500' : 'bg-emerald-500'}`}
                                            style={{ width: `${Math.min((usedHours / maxHours) * 100, 100)}%` }}
                                        />
                                    </div>
                                </div>
                            );
                        })}
                    </div>
                ))}
            </div>

            {/* Legend */}
            <div className="flex items-center gap-4 text-xs text-slate-500 dark:text-slate-400">
                <span className="flex items-center gap-1"><span className="w-3 h-3 rounded bg-red-500" /> Critical</span>
                <span className="flex items-center gap-1"><span className="w-3 h-3 rounded bg-amber-500" /> High</span>
                <span className="flex items-center gap-1"><span className="w-3 h-3 rounded bg-blue-500" /> Medium</span>
                <span className="flex items-center gap-1"><span className="w-3 h-3 rounded bg-slate-400" /> Low</span>
                <span className="flex items-center gap-1 ml-4"><AlertCircle className="w-3 h-3 text-red-500" /> Overbooked</span>
            </div>
        </div>
    );
}
