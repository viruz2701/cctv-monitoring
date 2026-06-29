// ═══════════════════════════════════════════════════════════════════════
// Glossary — Справочник терминов CCTV Health Monitor
//
// P1-1.7: Contextual Tooltips
//   - Glossary page with search/filter
//   - Anchored entries (#term-id) from InfoTooltip links
//   - Category grouping
// ═══════════════════════════════════════════════════════════════════════

import React, { useState, useMemo, useEffect, useRef } from 'react';
import { useTranslation } from 'react-i18next';
import { useSearchParams } from 'react-router-dom';
import {
    BookOpen,
    Search,
    Info,
    ChevronDown,
    ChevronRight,
} from '../components/ui/Icons';
import { Input } from '../components/ui';

// ═══════════════════════════════════════════════════════════════════════
// Glossary Data
// ═══════════════════════════════════════════════════════════════════════

interface GlossaryEntry {
    id: string;
    term: string;
    definition: string;
    category: string;
    seeAlso?: string[];
}

const GLOSSARY_ENTRIES: GlossaryEntry[] = [
    // ── Device & Hardware ──────────────────────────────────────────
    {
        id: 'nvr',
        term: 'NVR (Network Video Recorder)',
        definition: 'Сетевой видеорегистратор — устройство для записи видео с IP-камер. Хранит видеопотоки, управляет записью по расписанию и по событиям.',
        category: 'device',
        seeAlso: ['dvr', 'camera'],
    },
    {
        id: 'dvr',
        term: 'DVR (Digital Video Recorder)',
        definition: 'Цифровой видеорегистратор — устройство для записи аналоговых видеосигналов в цифровом формате. Отличается от NVR типом подключаемых камер.',
        category: 'device',
        seeAlso: ['nvr'],
    },
    {
        id: 'mtbf',
        term: 'MTBF (Mean Time Between Failures)',
        definition: 'Среднее время наработки на отказ — показатель надёжности оборудования. Рассчитывается как отношение общего времени работы к числу отказов за период.',
        category: 'performance',
    },
    {
        id: 'mttr',
        term: 'MTTR (Mean Time To Repair)',
        definition: 'Среднее время восстановления — среднее время, необходимое для устранения неисправности и возврата устройства в рабочее состояние.',
        category: 'performance',
    },

    // ── Compliance & Security ──────────────────────────────────────
    {
        id: 'sla',
        term: 'SLA (Service Level Agreement)',
        definition: 'Соглашение об уровне обслуживания — договорённость между сервис-провайдером и заказчиком о допустимом времени реакции и устранения инцидентов.',
        category: 'compliance',
        seeAlso: ['sla-breach', 'kii'],
    },
    {
        id: 'sla-breach',
        term: 'SLA Breach (Нарушение SLA)',
        definition: 'Ситуация, когда время реакции или устранения инцидента превысило лимит, установленный в SLA. Требует немедленного вмешательства и формирования отчёта.',
        category: 'compliance',
        seeAlso: ['sla'],
    },
    {
        id: 'kii',
        term: 'КИИ (Критическая Информационная Инфраструктура)',
        definition: 'Объекты, информационные системы которых имеют критическое значение для национальной безопасности, экономики и общественной безопасности РБ. CCTV Health Monitor относится к классу KII-2.',
        category: 'compliance',
    },
    {
        id: 'asvs',
        term: 'OWASP ASVS Level 3',
        definition: 'Application Security Verification Standard — стандарт верификации безопасности приложений. Level 3 требует максимального уровня защиты, включая криптографию, контроль доступа и аудит.',
        category: 'compliance',
    },
    {
        id: 'iec-62443',
        term: 'IEC 62443',
        definition: 'Международный стандарт безопасности для систем промышленной автоматизации и управления (IACS). Определяет зоны безопасности (SL-1..SL-4) и требования к ним.',
        category: 'compliance',
        seeAlso: ['kii'],
    },

    // ── CCTV Operations ────────────────────────────────────────────
    {
        id: 'rca',
        term: 'RCA (Root Cause Analysis)',
        definition: 'Анализ первопричины — методика выявления исходной причины отказа или инцидента. Строит граф зависимостей устройств и определяет корневой элемент проблемы.',
        category: 'operations',
    },
    {
        id: 'blast-radius',
        term: 'Blast Radius (Радиус поражения)',
        definition: 'Метрика влияния отказа — количество устройств и сервисов, затронутых отказом одного элемента системы. Используется в RCA для оценки масштаба инцидента.',
        category: 'operations',
        seeAlso: ['rca'],
    },
    {
        id: 'health-score',
        term: 'Health Score (Индекс здоровья)',
        definition: 'Комплексная оценка состояния устройства на основе uptime, температуры, свободного места на диске, частоты ошибок и статуса записи.',
        category: 'operations',
    },

    // ── Work Orders & CMMS ─────────────────────────────────────────
    {
        id: 'work-order',
        term: 'Work Order (Наряд на работу)',
        definition: 'Электронный документ, содержащий задание на обслуживание или ремонт оборудования. Включает описание работ, приоритет, назначенного техника и SLA-таймер.',
        category: 'cmms',
    },
    {
        id: 'preventive-maintenance',
        term: 'Preventive Maintenance (Плановое ТО)',
        definition: 'Регламентное обслуживание оборудования по расписанию — замена расходников, проверка параметров, чистка. Цель — предотвращение отказов до их возникновения.',
        category: 'cmms',
        seeAlso: ['work-order'],
    },
    {
        id: 'corrective-maintenance',
        term: 'Corrective Maintenance (Внеплановое ТО)',
        definition: 'Внеплановое обслуживание по факту отказа или деградации оборудования. Инициируется автоматически при обнаружении аномалий или по заявке.',
        category: 'cmms',
        seeAlso: ['work-order', 'rca'],
    },
];

const CATEGORIES = [
    { key: 'device', label: 'Device & Hardware' },
    { key: 'performance', label: 'Performance & Reliability' },
    { key: 'compliance', label: 'Compliance & Security' },
    { key: 'operations', label: 'CCTV Operations' },
    { key: 'cmms', label: 'Work Orders & CMMS' },
];

// ═══════════════════════════════════════════════════════════════════════
// Component
// ═══════════════════════════════════════════════════════════════════════

export function Glossary() {
    const { t } = useTranslation();
    const [searchParams] = useSearchParams();
    const [search, setSearch] = useState('');
    const [expandedEntries, setExpandedEntries] = useState<Set<string>>(new Set());
    const entriesRef = useRef<Map<string, HTMLDivElement>>(new Map());

    // Handle URL hash for deep-linking from InfoTooltip
    useEffect(() => {
        const hash = window.location.hash.replace('#', '');
        if (hash) {
            setExpandedEntries(prev => new Set(prev).add(hash));
            setTimeout(() => {
                const el = entriesRef.current.get(hash);
                if (el) {
                    el.scrollIntoView({ behavior: 'smooth', block: 'center' });
                }
            }, 100);
        }
    }, []);

    const filtered = useMemo(() => {
        if (!search.trim()) return GLOSSARY_ENTRIES;
        const q = search.toLowerCase();
        return GLOSSARY_ENTRIES.filter(
            e => e.term.toLowerCase().includes(q)
                || e.definition.toLowerCase().includes(q)
                || e.category.toLowerCase().includes(q)
        );
    }, [search]);

    const grouped = useMemo(() => {
        const groups = new Map<string, GlossaryEntry[]>();
        for (const entry of filtered) {
            const existing = groups.get(entry.category) ?? [];
            existing.push(entry);
            groups.set(entry.category, existing);
        }
        return groups;
    }, [filtered]);

    const toggleEntry = (id: string) => {
        setExpandedEntries(prev => {
            const next = new Set(prev);
            if (next.has(id)) next.delete(id);
            else next.add(id);
            return next;
        });
    };

    return (
        <div className="p-4 md:p-6 max-w-4xl mx-auto space-y-6">
            {/* Header */}
            <div className="flex items-center justify-between">
                <div>
                    <h1 className="text-2xl font-bold text-slate-900 dark:text-white flex items-center gap-2">
                        <BookOpen className="w-6 h-6" />
                        {t('glossary') || 'Glossary'}
                    </h1>
                    <p className="text-sm text-slate-500 dark:text-slate-400 mt-1">
                        {t('glossary_description') || 'Reference of technical terms used in CCTV Health Monitor'}
                    </p>
                </div>
            </div>

            {/* Search */}
            <div className="relative max-w-md">
                <Search className="absolute left-3 top-1/2 -translate-y-1/2 w-4 h-4 text-slate-400" />
                <input
                    type="text"
                    value={search}
                    onChange={(e) => setSearch(e.target.value)}
                    placeholder={t('search_glossary') || 'Search terms...'}
                    className="w-full pl-10 pr-4 py-2.5 text-sm border border-slate-200 dark:border-slate-700 rounded-lg bg-white dark:bg-slate-800 text-slate-900 dark:text-white placeholder:text-slate-400 focus:outline-none focus:ring-2 focus:ring-blue-500"
                    aria-label={t('search_glossary') || 'Search glossary terms'}
                />
            </div>

            {/* Results count */}
            <p className="text-xs text-slate-400">
                {filtered.length} {t('terms') || 'terms'}
                {search && ` matching "${search}"`}
            </p>

            {/* Glossary Entries by Category */}
            {Array.from(grouped.entries()).map(([category, entries]) => {
                const cat = CATEGORIES.find(c => c.key === category);
                return (
                    <div key={category} className="space-y-2">
                        <h2 className="text-sm font-semibold text-slate-600 dark:text-slate-400 uppercase tracking-wider px-1">
                            {cat?.label || category}
                        </h2>
                        <div className="bg-white dark:bg-slate-800 rounded-xl border border-slate-200 dark:border-slate-700 divide-y divide-slate-100 dark:divide-slate-700/50 overflow-hidden">
                            {entries.map((entry) => (
                                <div
                                    key={entry.id}
                                    ref={(el) => { if (el) entriesRef.current.set(entry.id, el); }}
                                    id={entry.id}
                                    className="scroll-mt-20"
                                >
                                    <button
                                        onClick={() => toggleEntry(entry.id)}
                                        className="w-full flex items-start gap-3 px-4 py-3.5 text-left hover:bg-slate-50 dark:hover:bg-slate-700/50 transition-colors group"
                                        aria-expanded={expandedEntries.has(entry.id)}
                                    >
                                        <div className="flex-1 min-w-0">
                                            <div className="flex items-center gap-2">
                                                {expandedEntries.has(entry.id) ? (
                                                    <ChevronDown className="w-4 h-4 text-slate-400 shrink-0 transition-transform" />
                                                ) : (
                                                    <ChevronRight className="w-4 h-4 text-slate-400 shrink-0 transition-transform" />
                                                )}
                                                <span className="text-sm font-medium text-slate-900 dark:text-white group-hover:text-blue-600 dark:group-hover:text-blue-400 transition-colors">
                                                    {entry.term}
                                                </span>
                                            </div>
                                        </div>
                                        <Info className="w-4 h-4 text-slate-300 dark:text-slate-600 shrink-0 mt-0.5" />
                                    </button>

                                    {expandedEntries.has(entry.id) && (
                                        <div className="px-4 pb-4 pt-1 pl-12">
                                            <p className="text-sm text-slate-600 dark:text-slate-300 leading-relaxed">
                                                {entry.definition}
                                            </p>
                                            {entry.seeAlso && entry.seeAlso.length > 0 && (
                                                <div className="flex items-center gap-2 mt-2 flex-wrap">
                                                    <span className="text-xs text-slate-400">{t('see_also') || 'See also:'}</span>
                                                    {entry.seeAlso.map((ref) => {
                                                        const refEntry = GLOSSARY_ENTRIES.find(e => e.id === ref);
                                                        return refEntry ? (
                                                            <button
                                                                key={ref}
                                                                onClick={() => {
                                                                    toggleEntry(ref);
                                                                    const el = entriesRef.current.get(ref);
                                                                    if (el) el.scrollIntoView({ behavior: 'smooth', block: 'center' });
                                                                }}
                                                                className="text-xs font-medium text-blue-600 dark:text-blue-400 hover:underline"
                                                            >
                                                                {refEntry.term}
                                                            </button>
                                                        ) : null;
                                                    })}
                                                </div>
                                            )}
                                        </div>
                                    )}
                                </div>
                            ))}
                        </div>
                    </div>
                );
            })}

            {/* Empty state */}
            {filtered.length === 0 && (
                <div className="text-center py-12">
                    <BookOpen className="w-12 h-12 text-slate-300 dark:text-slate-600 mx-auto mb-3" />
                    <p className="text-sm font-medium text-slate-500 dark:text-slate-400">
                        {t('no_terms_found') || 'No terms found'}
                    </p>
                    <p className="text-xs text-slate-400 mt-1">
                        {t('try_different_search') || 'Try a different search term'}
                    </p>
                </div>
            )}
        </div>
    );
}