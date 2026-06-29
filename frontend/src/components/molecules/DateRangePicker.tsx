import React, { useState, useRef, useEffect, useCallback, useMemo, useId } from 'react';
import {
  format,
  startOfDay,
  endOfDay,
  subDays,
  startOfMonth,
  endOfMonth,
  isSameDay,
  isWithinInterval,
  isAfter,
  isBefore,
  addMonths,
  subMonths,
  startOfWeek,
  endOfWeek,
  eachDayOfInterval,
  isSameMonth,
} from 'date-fns';
import { ru } from 'date-fns/locale';
import { Calendar, ChevronLeft, ChevronRight, Clock } from '../ui/Icons';

// ═══════════════════════════════════════════════════════════════════════
// DateRangePicker — выбор диапазона дат
// Пресеты: Today, Last 7 days, Last 30 days, This month, Custom
// Calendar popup с date-fns
// ═══════════════════════════════════════════════════════════════════════

export type DatePreset = 'today' | 'last7' | 'last30' | 'thisMonth' | 'custom';

export interface DateRange {
  start: Date;
  end: Date;
  preset: DatePreset;
}

interface PresetOption {
  value: DatePreset;
  label: string;
}

const PRESETS: PresetOption[] = [
  { value: 'today', label: 'Сегодня' },
  { value: 'last7', label: 'Последние 7 дней' },
  { value: 'last30', label: 'Последние 30 дней' },
  { value: 'thisMonth', label: 'Этот месяц' },
  { value: 'custom', label: 'Произвольный' },
];

function getPresetRange(preset: DatePreset): { start: Date; end: Date } {
  const now = new Date();
  switch (preset) {
    case 'today':
      return { start: startOfDay(now), end: endOfDay(now) };
    case 'last7':
      return { start: startOfDay(subDays(now, 6)), end: endOfDay(now) };
    case 'last30':
      return { start: startOfDay(subDays(now, 29)), end: endOfDay(now) };
    case 'thisMonth':
      return { start: startOfMonth(now), end: endOfMonth(now) };
    default:
      return { start: startOfDay(subDays(now, 6)), end: endOfDay(now) };
  }
}

interface DateRangePickerProps {
  value?: DateRange;
  onChange: (range: DateRange) => void;
  className?: string;
  /** Min selectable date */
  minDate?: Date;
  /** Max selectable date */
  maxDate?: Date;
}

export function DateRangePicker({
  value,
  onChange,
  className = '',
  minDate,
  maxDate,
}: DateRangePickerProps) {
  const currentRange = value ?? { ...getPresetRange('last7'), preset: 'last7' };
  const [isOpen, setIsOpen] = useState(false);
  const [activePreset, setActivePreset] = useState<DatePreset>(currentRange.preset);
  const [calMonth, setCalMonth] = useState(() => startOfMonth(currentRange.start));
  const [selectingEnd, setSelectingEnd] = useState(false);
  const [tempStart, setTempStart] = useState<Date | null>(null);
  const containerRef = useRef<HTMLDivElement>(null);
  const popoverId = useId();

  // Click outside
  useEffect(() => {
    if (!isOpen) return;
    const handleClick = (e: MouseEvent) => {
      if (containerRef.current && !containerRef.current.contains(e.target as Node)) {
        setIsOpen(false);
      }
    };
    document.addEventListener('mousedown', handleClick);
    return () => document.removeEventListener('mousedown', handleClick);
  }, [isOpen]);

  const handlePresetClick = useCallback(
    (preset: DatePreset) => {
      setActivePreset(preset);
      if (preset !== 'custom') {
        const range = getPresetRange(preset);
        onChange({ ...range, preset });
        setIsOpen(false);
      }
    },
    [onChange],
  );

  // ── Calendar helpers ─────────────────────────────────────────────────

  const days = useMemo(() => {
    const monthStart = startOfMonth(calMonth);
    const monthEnd = endOfMonth(calMonth);
    const calStart = startOfWeek(monthStart, { weekStartsOn: 1 });
    const calEnd = endOfWeek(monthEnd, { weekStartsOn: 1 });
    return eachDayOfInterval({ start: calStart, end: calEnd });
  }, [calMonth]);

  const isInRange = useCallback(
    (day: Date) => {
      if (activePreset !== 'custom') {
        return isWithinInterval(day, { start: currentRange.start, end: currentRange.end });
      }
      if (!tempStart) return false;
      if (selectingEnd) {
        const s = isBefore(tempStart, day) ? tempStart : day;
        const e = isBefore(tempStart, day) ? day : tempStart;
        return isWithinInterval(day, { start: s, end: e });
      }
      return isSameDay(day, tempStart);
    },
    [activePreset, currentRange, tempStart, selectingEnd],
  );

  const isRangeStart = useCallback(
    (day: Date) => {
      if (activePreset !== 'custom') return isSameDay(day, currentRange.start);
      return tempStart ? isSameDay(day, tempStart) : false;
    },
    [activePreset, currentRange, tempStart],
  );

  const isRangeEnd = useCallback(
    (day: Date) => {
      if (activePreset !== 'custom') return isSameDay(day, currentRange.end);
      if (!tempStart || !selectingEnd) return false;
      return isSameDay(day, tempStart); // simplified
    },
    [activePreset, currentRange, tempStart, selectingEnd],
  );

  const handleDayClick = useCallback(
    (day: Date) => {
      if (activePreset !== 'custom') return;

      if (!selectingEnd || !tempStart) {
        setTempStart(day);
        setSelectingEnd(true);
      } else {
        const s = isBefore(tempStart, day) ? tempStart : day;
        const e = isBefore(tempStart, day) ? day : tempStart;
        onChange({ start: s, end: e, preset: 'custom' });
        setTempStart(null);
        setSelectingEnd(false);
        setIsOpen(false);
      }
    },
    [activePreset, selectingEnd, tempStart, onChange],
  );

  const dayLabels = ['Пн', 'Вт', 'Ср', 'Чт', 'Пт', 'Сб', 'Вс'];

  const formatDisplay = (range: DateRange): string => {
    if (range.preset !== 'custom') {
      return PRESETS.find((p) => p.value === range.preset)?.label ?? '';
    }
    return `${format(range.start, 'd MMM', { locale: ru })} — ${format(range.end, 'd MMM', { locale: ru })}`;
  };

  return (
    <div ref={containerRef} className={`relative ${className}`}>
      {/* Trigger */}
      <button
        type="button"
        aria-haspopup="dialog"
        aria-expanded={isOpen}
        aria-controls={popoverId}
        onClick={() => setIsOpen((p) => !p)}
        className="
          w-full flex items-center gap-2 px-3 py-2 text-sm
          bg-white dark:bg-slate-800
          border border-slate-300 dark:border-slate-600
          rounded-lg shadow-sm
          hover:border-slate-400 dark:hover:border-slate-500
          focus:outline-none focus:ring-2 focus:ring-blue-500
          transition-colors text-left
        "
      >
        <Calendar size={16} className="text-slate-400 flex-shrink-0" />
        <span className="flex-1 text-slate-700 dark:text-slate-300">
          {formatDisplay(currentRange)}
        </span>
      </button>

      {/* Popover */}
      {isOpen && (
        <div
          id={popoverId}
          role="dialog"
          aria-label="Выбор диапазона дат"
          className="
            absolute z-50 mt-1 w-[340px]
            bg-white dark:bg-slate-800
            border border-slate-200 dark:border-slate-700
            rounded-lg shadow-lg
            p-4
            animate-fadeIn
          "
        >
          {/* Presets */}
          <div className="grid grid-cols-2 gap-1 mb-3">
            {PRESETS.map((preset) => (
              <button
                key={preset.value}
                type="button"
                onClick={() => handlePresetClick(preset.value)}
                className={`
                  px-2.5 py-1.5 text-xs font-medium rounded-md transition-colors
                  ${activePreset === preset.value
                    ? 'bg-blue-100 dark:bg-blue-900/30 text-blue-700 dark:text-blue-300'
                    : 'text-slate-600 dark:text-slate-400 hover:bg-slate-100 dark:hover:bg-slate-700'
                  }
                `}
              >
                {preset.label}
              </button>
            ))}
          </div>

          {/* Calendar (only for custom) */}
          {activePreset === 'custom' && (
            <>
              {/* Month nav */}
              <div className="flex items-center justify-between mb-3">
                <button
                  type="button"
                  onClick={() => setCalMonth((m) => subMonths(m, 1))}
                  className="p-1 rounded-md hover:bg-slate-100 dark:hover:bg-slate-700 text-slate-500 dark:text-slate-400"
                  aria-label="Previous month"
                >
                  <ChevronLeft size={16} />
                </button>
                <span className="text-sm font-medium text-slate-900 dark:text-white">
                  {format(calMonth, 'LLLL yyyy', { locale: ru })}
                </span>
                <button
                  type="button"
                  onClick={() => setCalMonth((m) => addMonths(m, 1))}
                  className="p-1 rounded-md hover:bg-slate-100 dark:hover:bg-slate-700 text-slate-500 dark:text-slate-400"
                  aria-label="Next month"
                >
                  <ChevronRight size={16} />
                </button>
              </div>

              {/* Day labels */}
              <div className="grid grid-cols-7 mb-1">
                {dayLabels.map((d) => (
                  <div key={d} className="text-center text-[11px] font-medium text-slate-400 dark:text-slate-500 py-1">
                    {d}
                  </div>
                ))}
              </div>

              {/* Days grid */}
              <div className="grid grid-cols-7">
                {days.map((day) => {
                  const isCurrentMonth = isSameMonth(day, calMonth);
                  const disabled =
                    (minDate && isBefore(day, minDate)) || (maxDate && isAfter(day, maxDate));
                  const inRange = isInRange(day);
                  const isStart = isRangeStart(day);
                  const isEnd = isRangeEnd(day);

                  return (
                    <button
                      key={day.toISOString()}
                      type="button"
                      disabled={!isCurrentMonth || disabled}
                      onClick={() => handleDayClick(day)}
                      className={`
                        relative h-8 text-xs rounded-md transition-colors
                        ${!isCurrentMonth ? 'text-slate-300 dark:text-slate-600' : ''}
                        ${disabled ? 'opacity-30 cursor-not-allowed' : 'cursor-pointer'}
                        ${inRange && !isStart && !isEnd ? 'bg-blue-50 dark:bg-blue-900/20' : ''}
                        ${isStart || isEnd
                          ? 'bg-blue-600 text-white hover:bg-blue-700 font-semibold'
                          : isCurrentMonth && !disabled
                            ? 'text-slate-700 dark:text-slate-300 hover:bg-slate-100 dark:hover:bg-slate-700'
                            : ''
                        }
                      `}
                    >
                      {format(day, 'd')}
                    </button>
                  );
                })}
              </div>

              {/* Selection hint */}
              <div className="flex items-center gap-1.5 mt-2 text-[11px] text-slate-400 dark:text-slate-500">
                <Clock size={12} />
                {selectingEnd ? 'Выберите конечную дату' : 'Выберите начальную дату'}
              </div>
            </>
          )}
        </div>
      )}
    </div>
  );
}
