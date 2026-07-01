import React, { useCallback } from 'react';
import { useTranslation } from 'react-i18next';
import { Plus, Trash2, AlertCircle } from '../ui/Icons';

// ═══════════════════════════════════════════════════════════════════════
// RuleEditor — редактор правил периодичности ТО
// Позволяет задавать интервалы в месяцах/днях для разных типов ТО
// ═══════════════════════════════════════════════════════════════════════

export interface MaintenanceRule {
  id: string;
  /** Код типа ТО: TO-1, TO-2, TO-3, custom */
  typeCode: string;
  /** Человеческое название */
  label: string;
  /** Интервал в месяцах (для regulatory auto-fill) */
  intervalMonths: number;
  /** Альтернативно — интервал в днях */
  intervalDays: number;
  /** Длительность выполнения в минутах */
  durationMinutes: number;
  /** Описание работ */
  description: string;
  /** Регион применения (ISO code) */
  region?: string;
  /** Обязательность (regulatory required) */
  isRequired: boolean;
}

interface RuleEditorProps {
  rules: MaintenanceRule[];
  onChange: (rules: MaintenanceRule[]) => void;
  /** Доступные шаблоны для быстрого добавления */
  templates?: MaintenanceRule[];
  /** Максимальное количество правил */
  maxRules?: number;
}

const DEFAULT_RULES: MaintenanceRule[] = [
  {
    id: 'TO-1',
    typeCode: 'TO-1',
    label: 'ТО-1: Ежемесячное обслуживание',
    intervalMonths: 1,
    intervalDays: 30,
    durationMinutes: 60,
    description: 'Визуальный осмотр, проверка креплений, очистка optics',
    isRequired: true,
  },
  {
    id: 'TO-2',
    typeCode: 'TO-2',
    label: 'ТО-2: Квартальное обслуживание',
    intervalMonths: 3,
    intervalDays: 90,
    durationMinutes: 120,
    description: 'Глубокая диагностика, прошивка, проверка кабельных соединений',
    isRequired: true,
  },
  {
    id: 'TO-3',
    typeCode: 'TO-3',
    label: 'ТО-3: Годовое обслуживание',
    intervalMonths: 12,
    intervalDays: 365,
    durationMinutes: 240,
    description: 'Полная ревизия, замена изношенных компонентов, калибровка',
    isRequired: true,
  },
];

const REGIONAL_TEMPLATES: Record<string, Partial<MaintenanceRule>[]> = {
  BY: [
    { intervalMonths: 1, intervalDays: 30, label: 'ТО-1 (РБ): Ежемесячное' },
    { intervalMonths: 3, intervalDays: 90, label: 'ТО-2 (РБ): Квартальное' },
    { intervalMonths: 6, intervalDays: 180, label: 'ТО-3 (РБ): Полугодовое' },
  ],
  RU: [
    { intervalMonths: 1, intervalDays: 30, label: 'ТО-1 (РФ): Ежемесячное' },
    { intervalMonths: 3, intervalDays: 90, label: 'ТО-2 (РФ): Квартальное' },
    { intervalMonths: 12, intervalDays: 365, label: 'ТО-3 (РФ): Годовое' },
  ],
  EU: [
    { intervalMonths: 1, intervalDays: 30, label: 'TO-1 (EU): Monthly' },
    { intervalMonths: 6, intervalDays: 180, label: 'TO-2 (EU): Semi-Annual' },
    { intervalMonths: 12, intervalDays: 365, label: 'TO-3 (EU): Annual' },
  ],
  TR: [
    { intervalMonths: 1, intervalDays: 30, label: 'TO-1 (TR): Aylık' },
    { intervalMonths: 3, intervalDays: 90, label: 'TO-2 (TR): 3 Aylık' },
    { intervalMonths: 12, intervalDays: 365, label: 'TO-3 (TR): Yıllık' },
  ],
  VN: [
    { intervalMonths: 1, intervalDays: 30, label: 'TO-1 (VN): Hàng tháng' },
    { intervalMonths: 6, intervalDays: 180, label: 'TO-2 (VN): 6 tháng' },
    { intervalMonths: 12, intervalDays: 365, label: 'TO-3 (VN): Hàng năm' },
  ],
};

/** Получить правила для региона */
export function getRulesForRegion(
  region: string,
): MaintenanceRule[] {
  const template = REGIONAL_TEMPLATES[region] || REGIONAL_TEMPLATES.EU;
  return template.map((t, i) => ({
    ...DEFAULT_RULES[i],
    ...t,
  }));
}

/** Рассчитать следующую дату ТО на основе правила */
export function calculateNextDue(
  intervalMonths: number,
  intervalDays: number,
  fromDate?: Date,
): string {
  const date = fromDate ? new Date(fromDate) : new Date();
  date.setMonth(date.getMonth() + intervalMonths);
  date.setDate(date.getDate() + intervalDays);
  return date.toISOString().split('T')[0];
}

/** Форматировать периодичность в человекочитаемый вид */
export function formatInterval(months: number, days: number): string {
  const parts: string[] = [];
  if (months > 0) parts.push(`${months} мес.`);
  if (days > 0) parts.push(`${days} дн.`);
  return parts.join(' ') || '—';
}

export function RuleEditor({
  rules,
  onChange,
  templates,
  maxRules = 10,
}: RuleEditorProps) {
  const { t } = useTranslation();

  const addRule = useCallback(() => {
    if (rules.length >= maxRules) return;
    const newRule: MaintenanceRule = {
      id: `custom-${Date.now()}`,
      typeCode: 'custom',
      label: '',
      intervalMonths: 1,
      intervalDays: 30,
      durationMinutes: 60,
      description: '',
      isRequired: false,
    };
    onChange([...rules, newRule]);
  }, [rules, maxRules, onChange]);

  const removeRule = useCallback(
    (id: string) => {
      onChange(rules.filter((r) => r.id !== id));
    },
    [rules, onChange],
  );

  const updateRule = useCallback(
    (id: string, patch: Partial<MaintenanceRule>) => {
      onChange(rules.map((r) => (r.id === id ? { ...r, ...patch } : r)));
    },
    [rules, onChange],
  );

  const applyTemplate = useCallback(
    (template: MaintenanceRule) => {
      const exists = rules.some((r) => r.typeCode === template.typeCode);
      if (exists) return; // не дублируем
      if (rules.length >= maxRules) return;
      onChange([...rules, { ...template, id: `${template.typeCode}-${Date.now()}` }]);
    },
    [rules, maxRules, onChange],
  );

  return (
    <div className="space-y-4">
      {/* Быстрое добавление из шаблонов */}
      {templates && templates.length > 0 && (
        <div>
          <label className="block text-sm font-medium text-slate-700 dark:text-slate-300 mb-2">
            {t('quick_add_template') || 'Быстрое добавление из шаблонов'}
          </label>
          <div className="flex flex-wrap gap-2">
            {templates.map((tmpl) => {
              const isAdded = rules.some((r) => r.typeCode === tmpl.typeCode);
              return (
                <button
                  key={tmpl.id}
                  type="button"
                  disabled={isAdded}
                  onClick={() => applyTemplate(tmpl)}
                  className={`
                    px-3 py-1.5 text-xs font-medium rounded-lg border transition-colors
                    ${isAdded
                      ? 'bg-slate-100 dark:bg-slate-800 text-slate-400 dark:text-slate-500 border-slate-200 dark:border-slate-700 cursor-not-allowed'
                      : 'bg-blue-50 dark:bg-blue-900/20 text-blue-700 dark:text-blue-300 border-blue-200 dark:border-blue-800 hover:bg-blue-100 dark:hover:bg-blue-900/30'
                    }
                  `}
                >
                  {tmpl.label}
                </button>
              );
            })}
          </div>
        </div>
      )}

      {/* Список правил */}
      <div className="space-y-3">
        {rules.map((rule, idx) => (
          <div
            key={rule.id}
            className="border border-slate-200 dark:border-slate-700 rounded-lg p-4 bg-white dark:bg-slate-800/50"
          >
            <div className="flex items-start justify-between gap-2 mb-3">
              <div className="flex items-center gap-2">
                <span className="text-xs font-semibold text-slate-400 dark:text-slate-500 bg-slate-100 dark:bg-slate-800 px-2 py-0.5 rounded">
                  #{idx + 1}
                </span>
                {rule.isRequired && (
                  <span className="flex items-center gap-1 text-xs text-amber-600 dark:text-amber-400">
                    <AlertCircle size={12} />
                    {t('regulatory_required') || 'Обязательное'}
                  </span>
                )}
              </div>
              {!rule.isRequired && (
                <button
                  type="button"
                  onClick={() => removeRule(rule.id)}
                  className="p-1 text-slate-400 hover:text-red-500 transition-colors"
                  aria-label={t('remove_rule') || 'Удалить правило'}
                >
                  <Trash2 size={16} />
                </button>
              )}
            </div>

            <div className="grid grid-cols-2 sm:grid-cols-4 gap-3">
              {/* Название */}
              <div className="col-span-2 sm:col-span-4">
                <label className="block text-xs font-medium text-slate-500 dark:text-slate-400 mb-1">
                  {t('rule_name') || 'Название'}
                </label>
                <input
                  type="text"
                  value={rule.label}
                  onChange={(e) => updateRule(rule.id, { label: e.target.value })}
                  className="w-full px-3 py-1.5 text-sm border border-slate-300 dark:border-slate-600 rounded-lg bg-white dark:bg-slate-800 text-slate-900 dark:text-white focus:outline-none focus:ring-2 focus:ring-blue-500"
                  placeholder={t('rule_name_placeholder') || 'Напр. ТО-1: Ежемесячное'}
                />
              </div>

              {/* Интервал (месяцы) */}
              <div>
                <label className="block text-xs font-medium text-slate-500 dark:text-slate-400 mb-1">
                  {t('interval_months') || 'Интервал (мес.)'}
                </label>
                <input
                  type="number"
                  min={1}
                  max={60}
                  value={rule.intervalMonths}
                  onChange={(e) => {
                    const v = Math.max(1, parseInt(e.target.value) || 1);
                    updateRule(rule.id, {
                      intervalMonths: v,
                      intervalDays: v * 30,
                    });
                  }}
                  className="w-full px-3 py-1.5 text-sm border border-slate-300 dark:border-slate-600 rounded-lg bg-white dark:bg-slate-800 text-slate-900 dark:text-white focus:outline-none focus:ring-2 focus:ring-blue-500"
                />
              </div>

              {/* Интервал (дни) */}
              <div>
                <label className="block text-xs font-medium text-slate-500 dark:text-slate-400 mb-1">
                  {t('interval_days') || 'Интервал (дн.)'}
                </label>
                <input
                  type="number"
                  min={1}
                  max={1825}
                  value={rule.intervalDays}
                  onChange={(e) => {
                    const v = Math.max(1, parseInt(e.target.value) || 1);
                    updateRule(rule.id, { intervalDays: v });
                  }}
                  className="w-full px-3 py-1.5 text-sm border border-slate-300 dark:border-slate-600 rounded-lg bg-white dark:bg-slate-800 text-slate-900 dark:text-white focus:outline-none focus:ring-2 focus:ring-blue-500"
                />
              </div>

              {/* Длительность */}
              <div>
                <label className="block text-xs font-medium text-slate-500 dark:text-slate-400 mb-1">
                  {t('duration_minutes') || 'Длительность (мин.)'}
                </label>
                <input
                  type="number"
                  min={5}
                  max={1440}
                  step={5}
                  value={rule.durationMinutes}
                  onChange={(e) => {
                    const v = Math.max(5, parseInt(e.target.value) || 5);
                    updateRule(rule.id, { durationMinutes: v });
                  }}
                  className="w-full px-3 py-1.5 text-sm border border-slate-300 dark:border-slate-600 rounded-lg bg-white dark:bg-slate-800 text-slate-900 dark:text-white focus:outline-none focus:ring-2 focus:ring-blue-500"
                />
              </div>

              {/* Следующее ТО (readonly) */}
              <div>
                <label className="block text-xs font-medium text-slate-500 dark:text-slate-400 mb-1">
                  {t('next_to') || 'Следующее ТО'}
                </label>
                <input
                  type="date"
                  value={calculateNextDue(rule.intervalMonths, 0)}
                  readOnly
                  className="w-full px-3 py-1.5 text-sm border border-slate-200 dark:border-slate-700 rounded-lg bg-slate-50 dark:bg-slate-900 text-slate-500 dark:text-slate-400 cursor-not-allowed"
                />
              </div>
            </div>

            {/* Описание */}
            <div className="mt-3">
              <label className="block text-xs font-medium text-slate-500 dark:text-slate-400 mb-1">
                {t('description') || 'Описание'}
              </label>
              <input
                type="text"
                value={rule.description}
                onChange={(e) => updateRule(rule.id, { description: e.target.value })}
                className="w-full px-3 py-1.5 text-sm border border-slate-300 dark:border-slate-600 rounded-lg bg-white dark:bg-slate-800 text-slate-900 dark:text-white focus:outline-none focus:ring-2 focus:ring-blue-500"
                placeholder={t('description_placeholder') || 'Краткое описание работ'}
              />
            </div>

            {/* Preview: периодичность */}
            <div className="mt-2 text-xs text-slate-400 dark:text-slate-500">
              {t('periodicity') || 'Периодичность'}: {formatInterval(rule.intervalMonths, rule.intervalDays)}
              {' | '}
              {t('annual_count') || 'В год'}: {Math.round(365 / (rule.intervalDays || 30))} раз
            </div>
          </div>
        ))}
      </div>

      {/* Кнопка добавления */}
      {rules.length < maxRules && (
        <button
          type="button"
          onClick={addRule}
          className="flex items-center gap-2 px-4 py-2 text-sm font-medium text-blue-600 dark:text-blue-400 border-2 border-dashed border-slate-300 dark:border-slate-600 rounded-lg hover:border-blue-400 dark:hover:border-blue-500 hover:bg-blue-50 dark:hover:bg-blue-900/10 transition-colors w-full justify-center"
        >
          <Plus size={16} />
          {t('add_rule') || 'Добавить правило'}
        </button>
      )}

      {rules.length >= maxRules && (
        <p className="text-xs text-slate-400 text-center">
          {t('max_rules_reached') || `Максимум ${maxRules} правил`}
        </p>
      )}
    </div>
  );
}
