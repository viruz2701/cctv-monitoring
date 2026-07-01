import React, { useState, useMemo, useCallback } from 'react';
import { useTranslation } from 'react-i18next';
import { Modal, Button, Badge } from '../ui';
import {
  X,
  Check,
  ChevronLeft,
  ChevronRight,
  AlertCircle,
  CalendarDays,
  Wrench,
  Users,
  FileSpreadsheet,
  Clock,
  Search,
  Download,
  RotateCcw,
} from '../ui/Icons';
import { TechnicianSelector, type Technician } from '../molecules/TechnicianSelector';
import { RuleEditor, getRulesForRegion, calculateNextDue, formatInterval, type MaintenanceRule } from './RuleEditor';

// ═══════════════════════════════════════════════════════════════════════
// ScheduleBuilder — 5-step wizard for annual maintenance schedule
// UX-4.4: Schedule Builder
// ═══════════════════════════════════════════════════════════════════════

// ── Types ─────────────────────────────────────────────────────────────

export interface DeviceOption {
  id: string;
  name: string;
  siteId: string;
  siteName: string;
  type: string;
  vendor?: string;
  model?: string;
}

export interface ScheduleEntry {
  deviceId: string;
  deviceName: string;
  siteName: string;
  ruleType: string;
  ruleLabel: string;
  intervalMonths: number;
  intervalDays: number;
  durationMinutes: number;
  nextDue: string;
  technicianIds: string[];
  description: string;
}

interface Conflict {
  date: string;
  technicianId: string;
  technicianName: string;
  entries: ScheduleEntry[];
}

interface ScheduleBuilderProps {
  isOpen: boolean;
  onClose: () => void;
  devices: DeviceOption[];
  technicians: Technician[];
  sites: { id: string; name: string }[];
  onGenerate: (entries: ScheduleEntry[]) => Promise<void>;
  /** Регион для regulatory template */
  region?: string;
}

// ── Step Configuration ────────────────────────────────────────────────

interface StepDef {
  id: number;
  label: string;
  icon: React.ReactNode;
}

const STEPS: StepDef[] = [
  { id: 0, label: 'Выбор устройств', icon: <Search size={16} /> },
  { id: 1, label: 'Шаблон ТО', icon: <Wrench size={16} /> },
  { id: 2, label: 'Назначение техников', icon: <Users size={16} /> },
  { id: 3, label: 'Проверка конфликтов', icon: <AlertCircle size={16} /> },
  { id: 4, label: 'Генерация', icon: <CalendarDays size={16} /> },
];

// ── Helpers ───────────────────────────────────────────────────────────

function detectConflicts(
  entries: ScheduleEntry[],
  technicians: Technician[],
): Conflict[] {
  const conflictMap = new Map<string, Conflict>();

  for (const entry of entries) {
    for (const techId of entry.technicianIds) {
      const key = `${entry.nextDue}|${techId}`;
      if (!conflictMap.has(key)) {
        const tech = technicians.find((t) => t.id === techId);
        conflictMap.set(key, {
          date: entry.nextDue,
          technicianId: techId,
          technicianName: tech?.name ?? techId,
          entries: [],
        });
      }
      conflictMap.get(key)!.entries.push(entry);
    }
  }

  return Array.from(conflictMap.values()).filter((c) => c.entries.length > 1);
}

function exportToExcel(entries: ScheduleEntry[]): void {
  const BOM = '\uFEFF';
  const headers = [
    'Device', 'Site', 'TO Type', 'Interval', 'Duration (min)',
    'Next Due', 'Technicians', 'Description',
  ];
  const rows = entries.map((e) => [
    e.deviceName,
    e.siteName,
    e.ruleLabel,
    formatInterval(e.intervalMonths, e.intervalDays),
    String(e.durationMinutes),
    e.nextDue,
    e.technicianIds.join('; '),
    e.description,
  ]);

  const csv = BOM + [headers.join(','), ...rows.map((r) => r.map((c) => `"${c}"`).join(','))].join('\n');
  const blob = new Blob([csv], { type: 'text/csv;charset=utf-8;' });
  const url = URL.createObjectURL(blob);
  const a = document.createElement('a');
  a.href = url;
  a.download = `schedule-${new Date().toISOString().split('T')[0]}.csv`;
  a.click();
  URL.revokeObjectURL(url);
}

// ── Filter Helpers ────────────────────────────────────────────────────

function getUniqueSites(devices: DeviceOption[]): string[] {
  return [...new Set(devices.map((d) => d.siteName))];
}

function getUniqueTypes(devices: DeviceOption[]): string[] {
  return [...new Set(devices.map((d) => d.type))];
}

function getUniqueVendors(devices: DeviceOption[]): string[] {
  return [...new Set(devices.filter((d) => d.vendor).map((d) => d.vendor!))];
}

// ── Main Component ────────────────────────────────────────────────────

export function ScheduleBuilder({
  isOpen,
  onClose,
  devices,
  technicians,
  sites,
  onGenerate,
  region = 'EU',
}: ScheduleBuilderProps) {
  const { t } = useTranslation();
  const [step, setStep] = useState(0);
  const [isGenerating, setIsGenerating] = useState(false);

  // Step 1: Selection state
  const [selectedSiteFilter, setSelectedSiteFilter] = useState<string>('all');
  const [selectedTypeFilter, setSelectedTypeFilter] = useState<string>('all');
  const [selectedVendorFilter, setSelectedVendorFilter] = useState<string>('all');
  const [searchQuery, setSearchQuery] = useState('');
  const [selectedDeviceIds, setSelectedDeviceIds] = useState<Set<string>>(new Set());

  // Step 2: Rules state
  const [rules, setRules] = useState<MaintenanceRule[]>(() => getRulesForRegion(region));

  // Step 3: Technician assignment
  const [technicianMap, setTechnicianMap] = useState<Map<string, string[]>>(new Map());

  // Step 4: Generated entries (for conflict review)
  const [entries, setEntries] = useState<ScheduleEntry[]>([]);

  // ── Derived state ────────────────────────────────────────────────────

  const filteredDevices = useMemo(() => {
    return devices.filter((d) => {
      if (selectedSiteFilter !== 'all' && d.siteName !== selectedSiteFilter) return false;
      if (selectedTypeFilter !== 'all' && d.type !== selectedTypeFilter) return false;
      if (selectedVendorFilter !== 'all' && d.vendor !== selectedVendorFilter) return false;
      if (searchQuery && !d.name.toLowerCase().includes(searchQuery.toLowerCase())) return false;
      return true;
    });
  }, [devices, selectedSiteFilter, selectedTypeFilter, selectedVendorFilter, searchQuery]);

  const selectedDevices = useMemo(
    () => devices.filter((d) => selectedDeviceIds.has(d.id)),
    [devices, selectedDeviceIds],
  );

  const isAllSelected = filteredDevices.length > 0 && filteredDevices.every((d) => selectedDeviceIds.has(d.id));

  // ── Step validation ──────────────────────────────────────────────────

  const canProceedFromStep = useCallback(
    (s: number): boolean => {
      switch (s) {
        case 0: return selectedDeviceIds.size > 0;
        case 1: return rules.length > 0;
        case 2: return true; // optional assignment
        case 3: return entries.length > 0;
        default: return true;
      }
    },
    [selectedDeviceIds, rules, entries],
  );

  const conflicts = useMemo(() => detectConflicts(entries, technicians), [entries, technicians]);

  const hasConflicts = conflicts.length > 0;

  // ── Handlers ─────────────────────────────────────────────────────────

  const toggleDeviceSelection = useCallback((deviceId: string) => {
    setSelectedDeviceIds((prev) => {
      const next = new Set(prev);
      if (next.has(deviceId)) {
        next.delete(deviceId);
      } else {
        next.add(deviceId);
      }
      return next;
    });
  }, []);

  const toggleSelectAll = useCallback(() => {
    if (isAllSelected) {
      setSelectedDeviceIds(new Set());
    } else {
      setSelectedDeviceIds(new Set(filteredDevices.map((d) => d.id)));
    }
  }, [filteredDevices, isAllSelected]);

  const handleTechnicianChange = useCallback(
    (deviceId: string, techIds: string[]) => {
      setTechnicianMap((prev) => {
        const next = new Map(prev);
        if (techIds.length > 0) {
          next.set(deviceId, techIds);
        } else {
          next.delete(deviceId);
        }
        return next;
      });
    },
    [],
  );

  /** Генерация расписания на основе выбранных устройств и правил */
  const generateSchedule = useCallback(() => {
    const generated: ScheduleEntry[] = [];

    for (const dev of selectedDevices) {
      for (const rule of rules) {
        const techIds = technicianMap.get(dev.id) ?? [];
        generated.push({
          deviceId: dev.id,
          deviceName: dev.name,
          siteName: dev.siteName,
          ruleType: rule.typeCode,
          ruleLabel: rule.label,
          intervalMonths: rule.intervalMonths,
          intervalDays: rule.intervalDays,
          durationMinutes: rule.durationMinutes,
          nextDue: calculateNextDue(rule.intervalMonths, 0),
          technicianIds: techIds,
          description: rule.description,
        });
      }
    }

    setEntries(generated);
  }, [selectedDevices, rules, technicianMap]);

  const handleGenerate = useCallback(async () => {
    setIsGenerating(true);
    try {
      await onGenerate(entries);
      onClose();
    } finally {
      setIsGenerating(false);
    }
  }, [entries, onGenerate, onClose]);

  const handleExport = useCallback(() => {
    exportToExcel(entries);
  }, [entries]);

  // ── Step Content ─────────────────────────────────────────────────────

  const renderStepContent = () => {
    switch (step) {
      case 0: return renderDeviceSelection();
      case 1: return renderTemplateStep();
      case 2: return renderTechnicianAssignment();
      case 3: return renderConflictReview();
      case 4: return renderGenerationStep();
      default: return null;
    }
  };

  // ── Step 0: Device Selection ─────────────────────────────────────────

  const renderDeviceSelection = () => {
    const sites = getUniqueSites(devices);
    const types = getUniqueTypes(devices);
    const vendors = getUniqueVendors(devices);

    return (
      <div className="space-y-4">
        <p className="text-sm text-slate-600 dark:text-slate-400">
          {t('schedule_step1_desc') || 'Выберите устройства для включения в годовой график ТО'}
        </p>

        {/* Filters */}
        <div className="grid grid-cols-1 sm:grid-cols-4 gap-3">
          <select
            value={selectedSiteFilter}
            onChange={(e) => setSelectedSiteFilter(e.target.value)}
            className="w-full px-3 py-2 text-sm border border-slate-300 dark:border-slate-600 rounded-lg bg-white dark:bg-slate-800 text-slate-900 dark:text-white focus:outline-none focus:ring-2 focus:ring-blue-500"
          >
            <option value="all">{t('all_sites') || 'Все площадки'}</option>
            {sites.map((s) => <option key={s} value={s}>{s}</option>)}
          </select>

          <select
            value={selectedTypeFilter}
            onChange={(e) => setSelectedTypeFilter(e.target.value)}
            className="w-full px-3 py-2 text-sm border border-slate-300 dark:border-slate-600 rounded-lg bg-white dark:bg-slate-800 text-slate-900 dark:text-white focus:outline-none focus:ring-2 focus:ring-blue-500"
          >
            <option value="all">{t('all_types') || 'Все типы'}</option>
            {types.map((t) => <option key={t} value={t}>{t}</option>)}
          </select>

          <select
            value={selectedVendorFilter}
            onChange={(e) => setSelectedVendorFilter(e.target.value)}
            className="w-full px-3 py-2 text-sm border border-slate-300 dark:border-slate-600 rounded-lg bg-white dark:bg-slate-800 text-slate-900 dark:text-white focus:outline-none focus:ring-2 focus:ring-blue-500"
          >
            <option value="all">{t('all_vendors') || 'Все производители'}</option>
            {vendors.map((v) => <option key={v} value={v}>{v}</option>)}
          </select>

          <div className="relative">
            <Search size={14} className="absolute left-3 top-1/2 -translate-y-1/2 text-slate-400" />
            <input
              type="text"
              value={searchQuery}
              onChange={(e) => setSearchQuery(e.target.value)}
              placeholder={t('search_device') || 'Поиск устройства...'}
              className="w-full pl-9 pr-3 py-2 text-sm border border-slate-300 dark:border-slate-600 rounded-lg bg-white dark:bg-slate-800 text-slate-900 dark:text-white focus:outline-none focus:ring-2 focus:ring-blue-500"
            />
          </div>
        </div>

        {/* Select All / Count */}
        <div className="flex items-center justify-between">
          <label className="flex items-center gap-2 text-sm text-slate-600 dark:text-slate-400">
            <input
              type="checkbox"
              checked={isAllSelected}
              onChange={toggleSelectAll}
              className="rounded border-slate-300 dark:border-slate-600"
            />
            {t('select_all') || 'Выбрать всё'} ({filteredDevices.length})
          </label>
          <span className="text-xs text-slate-400">
            {t('selected') || 'Выбрано'}: {selectedDeviceIds.size}
          </span>
        </div>

        {/* Device List */}
        <div className="max-h-64 overflow-y-auto border border-slate-200 dark:border-slate-700 rounded-lg divide-y divide-slate-200 dark:divide-slate-700">
          {filteredDevices.length === 0 ? (
            <div className="p-6 text-center text-sm text-slate-400">
              {t('no_devices_found') || 'Устройства не найдены'}
            </div>
          ) : (
            filteredDevices.map((dev) => (
              <label
                key={dev.id}
                className="flex items-center gap-3 px-4 py-2.5 hover:bg-slate-50 dark:hover:bg-slate-800/50 cursor-pointer transition-colors"
              >
                <input
                  type="checkbox"
                  checked={selectedDeviceIds.has(dev.id)}
                  onChange={() => toggleDeviceSelection(dev.id)}
                  className="rounded border-slate-300 dark:border-slate-600"
                />
                <div className="flex-1 min-w-0">
                  <div className="text-sm font-medium text-slate-900 dark:text-white truncate">
                    {dev.name}
                  </div>
                  <div className="text-xs text-slate-500 dark:text-slate-400 truncate">
                    {dev.siteName} · {dev.type}{dev.vendor ? ` · ${dev.vendor}` : ''}
                  </div>
                </div>
                {dev.model && (
                  <Badge variant="neutral">{dev.model}</Badge>
                )}
              </label>
            ))
          )}
        </div>
      </div>
    );
  };

  // ── Step 1: Template / Rules ─────────────────────────────────────────

  const renderTemplateStep = () => {
    const templates = getRulesForRegion(region);

    return (
      <div className="space-y-4">
        <p className="text-sm text-slate-600 dark:text-slate-400">
          {t('schedule_step2_desc') || 'Настройте правила периодичности ТО. Вы можете использовать regulatory шаблоны или создать свои.'}
        </p>

        {/* Region badge */}
        <div className="flex items-center gap-2">
          <span className="text-xs font-medium text-slate-500 dark:text-slate-400">
            {t('region') || 'Регион'}:
          </span>
          <Badge variant="info">{region}</Badge>
          <button
            type="button"
            onClick={() => setRules(getRulesForRegion(region))}
            className="text-xs text-blue-600 dark:text-blue-400 hover:underline ml-2"
          >
            {t('reset_to_template') || 'Сбросить к шаблону'}
          </button>
        </div>

        {/* Summary: selected devices count */}
        <div className="bg-blue-50 dark:bg-blue-900/10 border border-blue-200 dark:border-blue-800 rounded-lg p-3">
          <p className="text-sm text-blue-700 dark:text-blue-300">
            {t('rules_applied_to') || 'Правила будут применены к'}: <strong>{selectedDevices.length}</strong>{' '}
            {t('devices_lower') || 'устройствам'}
          </p>
          <p className="text-xs text-blue-500 dark:text-blue-400 mt-1">
            {t('total_entries_preview') || 'Всего будет создано'}: {selectedDevices.length * rules.length}{' '}
            {t('schedule_entries') || 'записей'}
          </p>
        </div>

        <RuleEditor
          rules={rules}
          onChange={setRules}
          templates={templates}
        />
      </div>
    );
  };

  // ── Step 2: Technician Assignment ────────────────────────────────────

  const renderTechnicianAssignment = () => {
    // Назначаем техников на уровне device
    // Если устройству не назначен техник, оно получит "unassigned"
    return (
      <div className="space-y-4">
        <p className="text-sm text-slate-600 dark:text-slate-400">
          {t('schedule_step3_desc') || 'Назначьте техников на устройства. Один техник может обслуживать несколько устройств.'}
        </p>

        <div className="max-h-96 overflow-y-auto space-y-3">
          {selectedDevices.length === 0 ? (
            <div className="text-sm text-slate-400 text-center py-8">
              {t('no_devices_selected') || 'Нет выбранных устройств'}
            </div>
          ) : (
            selectedDevices.map((dev) => (
              <div
                key={dev.id}
                className="border border-slate-200 dark:border-slate-700 rounded-lg p-4 bg-white dark:bg-slate-800/50"
              >
                <div className="flex items-center justify-between mb-2">
                  <div>
                    <span className="text-sm font-medium text-slate-900 dark:text-white">{dev.name}</span>
                    <span className="text-xs text-slate-500 dark:text-slate-400 ml-2">{dev.siteName}</span>
                  </div>
                  <Badge variant="neutral">{dev.type}</Badge>
                </div>
                <TechnicianSelector
                  technicians={technicians}
                  selectedIds={technicianMap.get(dev.id) ?? []}
                  onChange={(ids) => handleTechnicianChange(dev.id, ids)}
                  placeholder={t('assign_technician') || 'Назначить техника...'}
                />
              </div>
            ))
          )}
        </div>
      </div>
    );
  };

  // ── Step 3: Conflict Review ──────────────────────────────────────────

  const renderConflictReview = () => {
    if (entries.length === 0) {
      return (
        <div className="text-center py-8">
          <p className="text-sm text-slate-500 dark:text-slate-400">
            {t('generate_first') || 'Сгенерируйте расписание для проверки конфликтов'}
          </p>
          <Button onClick={generateSchedule} className="mt-4">
            {t('generate_schedule') || 'Сгенерировать расписание'}
          </Button>
        </div>
      );
    }

    return (
      <div className="space-y-4">
        <div className="flex items-center justify-between">
          <p className="text-sm text-slate-600 dark:text-slate-400">
            {t('schedule_step4_desc') || 'Проверьте расписание на конфликты'}
          </p>
          <span className="text-xs text-slate-500">
            {t('total_entries') || 'Всего записей'}: {entries.length}
          </span>
        </div>

        {/* Conflicts */}
        {hasConflicts ? (
          <div className="bg-red-50 dark:bg-red-900/10 border border-red-200 dark:border-red-800 rounded-lg p-4">
            <div className="flex items-center gap-2 text-red-700 dark:text-red-300 font-medium mb-2">
              <AlertCircle size={16} />
              {t('conflicts_found') || 'Найдены конфликты'} ({conflicts.length})
            </div>
            <div className="space-y-2">
              {conflicts.map((conflict, idx) => (
                <div
                  key={`${conflict.date}-${conflict.technicianId}-${idx}`}
                  className="bg-white dark:bg-slate-800 rounded p-3 text-sm"
                >
                  <div className="flex items-center gap-2 text-slate-700 dark:text-slate-300 font-medium mb-1">
                    <Clock size={14} />
                    {conflict.date}
                    <span className="text-slate-400">→</span>
                    {conflict.technicianName}
                  </div>
                  <ul className="list-disc list-inside text-xs text-slate-500 dark:text-slate-400 space-y-0.5">
                    {conflict.entries.map((e, i) => (
                      <li key={i}>
                        {e.deviceName} — {e.ruleLabel} ({e.durationMinutes} мин.)
                      </li>
                    ))}
                  </ul>
                </div>
              ))}
            </div>
          </div>
        ) : (
          <div className="bg-emerald-50 dark:bg-emerald-900/10 border border-emerald-200 dark:border-emerald-800 rounded-lg p-4">
            <div className="flex items-center gap-2 text-emerald-700 dark:text-emerald-300 font-medium">
              <Check size={16} />
              {t('no_conflicts') || 'Конфликтов не найдено'}
            </div>
          </div>
        )}

        {/* Preview Table */}
        <div className="border border-slate-200 dark:border-slate-700 rounded-lg overflow-hidden">
          <div className="max-h-48 overflow-y-auto">
            <table className="w-full text-sm">
              <thead className="bg-slate-50 dark:bg-slate-800/50 sticky top-0">
                <tr>
                  <th className="text-left px-3 py-2 text-xs font-medium text-slate-500 dark:text-slate-400">
                    {t('device') || 'Устройство'}
                  </th>
                  <th className="text-left px-3 py-2 text-xs font-medium text-slate-500 dark:text-slate-400">
                    {t('to_type') || 'Тип ТО'}
                  </th>
                  <th className="text-left px-3 py-2 text-xs font-medium text-slate-500 dark:text-slate-400">
                    {t('next_due') || 'Следующее'}
                  </th>
                  <th className="text-left px-3 py-2 text-xs font-medium text-slate-500 dark:text-slate-400">
                    {t('technician') || 'Техник'}
                  </th>
                </tr>
              </thead>
              <tbody className="divide-y divide-slate-200 dark:divide-slate-700">
                {entries.map((entry, idx) => (
                  <tr key={idx} className="hover:bg-slate-50 dark:hover:bg-slate-800/30">
                    <td className="px-3 py-2 text-slate-900 dark:text-white">{entry.deviceName}</td>
                    <td className="px-3 py-2 text-slate-600 dark:text-slate-400">{entry.ruleLabel}</td>
                    <td className="px-3 py-2 text-slate-600 dark:text-slate-400">{entry.nextDue}</td>
                    <td className="px-3 py-2 text-slate-600 dark:text-slate-400">
                      {entry.technicianIds.length > 0
                        ? entry.technicianIds.map((tid) => technicians.find((t) => t.id === tid)?.name ?? tid).join(', ')
                        : <span className="text-slate-400 italic">{t('unassigned') || 'Не назначен'}</span>
                      }
                    </td>
                  </tr>
                ))}
              </tbody>
            </table>
          </div>
        </div>

        {/* Regenerate */}
        <div className="flex gap-2">
          <Button variant="secondary" onClick={generateSchedule} icon={<RotateCcw size={16} />}>
            {t('regenerate') || 'Перегенерировать'}
          </Button>
          <Button variant="secondary" onClick={handleExport} icon={<Download size={16} />}>
            {t('export_csv') || 'Экспорт CSV'}
          </Button>
        </div>
      </div>
    );
  };

  // ── Step 4: Final Generation ─────────────────────────────────────────

  const renderGenerationStep = () => {
    return (
      <div className="space-y-6">
        <p className="text-sm text-slate-600 dark:text-slate-400">
          {t('schedule_step5_desc') || 'Подтвердите создание годового графика ТО'}
        </p>

        {/* Summary Cards */}
        <div className="grid grid-cols-2 sm:grid-cols-4 gap-4">
          <div className="bg-slate-50 dark:bg-slate-800/50 border border-slate-200 dark:border-slate-700 rounded-lg p-4 text-center">
            <div className="text-2xl font-bold text-slate-900 dark:text-white">{selectedDevices.length}</div>
            <div className="text-xs text-slate-500 dark:text-slate-400 mt-1">
              {t('devices') || 'Устройства'}
            </div>
          </div>
          <div className="bg-slate-50 dark:bg-slate-800/50 border border-slate-200 dark:border-slate-700 rounded-lg p-4 text-center">
            <div className="text-2xl font-bold text-slate-900 dark:text-white">{rules.length}</div>
            <div className="text-xs text-slate-500 dark:text-slate-400 mt-1">
              {t('rule_types') || 'Типы ТО'}
            </div>
          </div>
          <div className="bg-slate-50 dark:bg-slate-800/50 border border-slate-200 dark:border-slate-700 rounded-lg p-4 text-center">
            <div className="text-2xl font-bold text-slate-900 dark:text-white">{entries.length}</div>
            <div className="text-xs text-slate-500 dark:text-slate-400 mt-1">
              {t('entries') || 'Записей'}
            </div>
          </div>
          <div className="bg-slate-50 dark:bg-slate-800/50 border border-slate-200 dark:border-slate-700 rounded-lg p-4 text-center">
            <div className="text-2xl font-bold text-slate-900 dark:text-white">
              {entries.reduce((sum, e) => sum + e.durationMinutes, 0)}
            </div>
            <div className="text-xs text-slate-500 dark:text-slate-400 mt-1">
              {t('total_minutes') || 'Всего минут'}
            </div>
          </div>
        </div>

        {/* Conflict warning */}
        {hasConflicts && (
          <div className="bg-amber-50 dark:bg-amber-900/10 border border-amber-200 dark:border-amber-800 rounded-lg p-4">
            <div className="flex items-center gap-2 text-amber-700 dark:text-amber-300 font-medium">
              <AlertCircle size={16} />
              {t('conflicts_warning') || 'Обнаружены конфликты. Рекомендуем вернуться к шагу 4 и исправить.'}
            </div>
          </div>
        )}

        {/* Export option */}
        <div className="border border-slate-200 dark:border-slate-700 rounded-lg p-4">
          <div className="flex items-center justify-between">
            <div className="flex items-center gap-3">
              <FileSpreadsheet size={24} className="text-emerald-600 dark:text-emerald-400" />
              <div>
                <div className="text-sm font-medium text-slate-900 dark:text-white">
                  {t('export_annual_plan') || 'Экспорт годового плана'}
                </div>
                <div className="text-xs text-slate-500 dark:text-slate-400">
                  {t('export_desc') || 'Скачать в формате CSV для Excel'}
                </div>
              </div>
            </div>
            <Button variant="secondary" onClick={handleExport} icon={<Download size={16} />}>
              {t('export') || 'Экспорт'}
            </Button>
          </div>
        </div>
      </div>
    );
  };

  // ── Navigation ───────────────────────────────────────────────────────

  const handleNext = useCallback(() => {
    if (step === 0) {
      // Generate entries when moving to step 1 from step 0
      // Actually, entries are generated at step 3 or 4
    }
    if (step === 2) {
      // Auto-generate when moving to review step
      generateSchedule();
    }
    setStep((prev) => Math.min(prev + 1, 4));
  }, [step, generateSchedule]);

  const handleBack = useCallback(() => {
    setStep((prev) => Math.max(prev - 1, 0));
  }, []);

  // ── Render ───────────────────────────────────────────────────────────

  return (
    <Modal
      isOpen={isOpen}
      onClose={onClose}
      size="xl"
      title={`${t('schedule_builder') || 'Schedule Builder'}`}
      footer={
        <div className="flex items-center justify-between w-full">
          {/* Step indicator */}
          <div className="flex items-center gap-1 text-xs text-slate-400">
            <span className="font-medium text-slate-600 dark:text-slate-300">{t('step') || 'Шаг'}</span>
            <span>{step + 1}</span>
            <span>/</span>
            <span>{STEPS.length}</span>
          </div>

          <div className="flex items-center gap-2">
            {step > 0 && (
              <Button variant="ghost" onClick={handleBack} icon={<ChevronLeft size={16} />}>
                {t('back') || 'Назад'}
              </Button>
            )}

            {step < 4 && (
              <Button
                onClick={handleNext}
                disabled={!canProceedFromStep(step)}
                icon={<ChevronRight size={16} />}
                iconPosition="right"
              >
                {t('next') || 'Далее'}
              </Button>
            )}

            {step === 4 && (
              <Button
                onClick={handleGenerate}
                loading={isGenerating}
                disabled={entries.length === 0 || isGenerating}
                icon={<Check size={16} />}
              >
                {t('create_schedule') || 'Создать график'}
              </Button>
            )}
          </div>
        </div>
      }
    >
      {/* Step Tabs */}
      <div className="flex items-center gap-1 mb-6 overflow-x-auto pb-2">
        {STEPS.map((s) => {
          const isActive = s.id === step;
          const isCompleted = s.id < step;
          return (
            <button
              key={s.id}
              type="button"
              disabled={s.id > step}
              onClick={() => {
                // Can only go to completed or current step
                if (s.id <= step) setStep(s.id);
              }}
              className={`
                flex items-center gap-1.5 px-3 py-1.5 text-xs font-medium rounded-lg
                transition-colors whitespace-nowrap
                ${isActive
                  ? 'bg-blue-100 dark:bg-blue-900/30 text-blue-700 dark:text-blue-300'
                  : isCompleted
                    ? 'bg-emerald-50 dark:bg-emerald-900/20 text-emerald-600 dark:text-emerald-400'
                    : 'text-slate-400 dark:text-slate-500 cursor-not-allowed'
                }
              `}
            >
              {isCompleted ? <Check size={12} /> : s.icon}
              {s.label}
            </button>
          );
        })}
      </div>

      {/* Step Content */}
      {renderStepContent()}
    </Modal>
  );
}

// Re-export types
export type { MaintenanceRule } from './RuleEditor';
export { getRulesForRegion, calculateNextDue, formatInterval } from './RuleEditor';
