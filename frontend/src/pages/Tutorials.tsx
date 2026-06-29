// ═══════════════════════════════════════════════════════════════════════
// Tutorials — UX-14.3.5: Onboarding Video Tutorials
//
// Страница со списком обучающих видео:
//   - Категории: Getting Started, Devices, CMMS, Reports, Mobile
//   - Grid карточек (VideoTutorialCard)
//   - Search по названиям
//   - Фильтр по категориям
// ═══════════════════════════════════════════════════════════════════════

import React, { useState, useMemo } from 'react';
import { Search, BookOpen, Monitor, HardDrive, FileText, Smartphone, Wrench, LayoutDashboard } from '../components/ui/Icons';
import { VideoTutorialCard } from '../components/ui/VideoTutorialCard';
import type { TutorialVideo } from '../components/ui/VideoTutorialCard';
import { useTranslation } from 'react-i18next';

// ═══════════════════════════════════════════════════════════════════════
// Tutorial data
// ═══════════════════════════════════════════════════════════════════════

const ALL_TUTORIALS: TutorialVideo[] = [
  // ── Getting Started ──────────────────────────────────────────────
  {
    id: 'gs-welcome',
    title: 'Welcome to CCTV Health Monitor',
    description: 'A quick tour of the platform, main features, and how to navigate the dashboard.',
    duration: '4:30',
    category: 'Getting Started',
    videoUrl: null,
  },
  {
    id: 'gs-login',
    title: 'Login & Account Setup',
    description: 'How to log in, reset your password, and configure your profile settings.',
    duration: '2:15',
    category: 'Getting Started',
    videoUrl: null,
  },
  {
    id: 'gs-dashboard',
    title: 'Dashboard Overview',
    description: 'Understanding the main dashboard widgets, KPIs, and health indicators.',
    duration: '5:00',
    category: 'Getting Started',
    videoUrl: null,
  },

  // ── Devices ──────────────────────────────────────────────────────
  {
    id: 'dev-add',
    title: 'Adding a New Device',
    description: 'Step-by-step guide to register a new CCTV camera or NVR in the system.',
    duration: '3:45',
    category: 'Devices',
    videoUrl: null,
  },
  {
    id: 'dev-monitoring',
    title: 'Device Monitoring & Alerts',
    description: 'How to monitor device health, configure alert thresholds, and respond to issues.',
    duration: '4:20',
    category: 'Devices',
    videoUrl: null,
  },
  {
    id: 'dev-detail',
    title: 'Device Details & History',
    description: 'Exploring device detail page, event history, and performance metrics.',
    duration: '3:10',
    category: 'Devices',
    videoUrl: null,
  },

  // ── CMMS ─────────────────────────────────────────────────────────
  {
    id: 'cmms-wo',
    title: 'Creating & Managing Work Orders',
    description: 'How to create, assign, and track work orders from creation to completion.',
    duration: '6:00',
    category: 'CMMS',
    videoUrl: null,
  },
  {
    id: 'cmms-schedule',
    title: 'Maintenance Scheduling',
    description: 'Setting up preventive maintenance schedules and recurring tasks.',
    duration: '4:45',
    category: 'CMMS',
    videoUrl: null,
  },
  {
    id: 'cmms-parts',
    title: 'Spare Parts Management',
    description: 'Managing inventory, tracking usage, and ordering spare parts.',
    duration: '3:30',
    category: 'CMMS',
    videoUrl: null,
  },

  // ── Reports ──────────────────────────────────────────────────────
  {
    id: 'rpt-generate',
    title: 'Generating Reports',
    description: 'How to generate custom reports, export data, and schedule automatic reports.',
    duration: '4:00',
    category: 'Reports',
    videoUrl: null,
  },
  {
    id: 'rpt-analytics',
    title: 'Analytics Dashboard',
    description: 'Using the analytics dashboard to gain insights into system performance.',
    duration: '5:30',
    category: 'Reports',
    videoUrl: null,
  },

  // ── Mobile ───────────────────────────────────────────────────────
  {
    id: 'mob-setup',
    title: 'Mobile App Setup',
    description: 'Installing and configuring the CCTV Health Monitor mobile app.',
    duration: '2:45',
    category: 'Mobile',
    videoUrl: null,
  },
  {
    id: 'mob-features',
    title: 'Mobile App Features',
    description: 'Using the mobile app for on-the-go monitoring, alerts, and work orders.',
    duration: '3:20',
    category: 'Mobile',
    videoUrl: null,
  },
];

const CATEGORY_ICONS: Record<string, React.ReactNode> = {
  'Getting Started': <LayoutDashboard size={16} />,
  'Devices': <HardDrive size={16} />,
  'CMMS': <Wrench size={16} />,
  'Reports': <FileText size={16} />,
  'Mobile': <Smartphone size={16} />,
};

export function Tutorials() {
  const { t } = useTranslation();
  const [search, setSearch] = useState('');
  const [activeCategory, setActiveCategory] = useState<string | null>(null);

  const categories = useMemo(() => {
    const cats = new Set(ALL_TUTORIALS.map((v) => v.category));
    return Array.from(cats);
  }, []);

  const filteredTutorials = useMemo(() => {
    let result = ALL_TUTORIALS;
    if (activeCategory) {
      result = result.filter((v) => v.category === activeCategory);
    }
    if (search.trim()) {
      const term = search.toLowerCase();
      result = result.filter(
        (v) =>
          v.title.toLowerCase().includes(term) ||
          v.description.toLowerCase().includes(term),
      );
    }
    return result;
  }, [search, activeCategory]);

  return (
    <div className="max-w-6xl mx-auto">
      {/* Header */}
      <div className="mb-6">
        <div className="flex items-center gap-3 mb-2">
          <div className="p-2 bg-blue-100 dark:bg-blue-900/30 rounded-lg">
            <BookOpen className="w-6 h-6 text-blue-600 dark:text-blue-400" />
          </div>
          <div>
            <h1 className="text-2xl font-bold text-slate-900 dark:text-white">
              {t('tutorials') || 'Video Tutorials'}
            </h1>
            <p className="text-sm text-slate-500 dark:text-slate-400 mt-1">
              {t('tutorials_subtitle') || 'Learn how to use CCTV Health Monitor effectively'}
            </p>
          </div>
        </div>
      </div>

      {/* Search */}
      <div className="relative mb-6">
        <Search className="absolute left-3.5 top-1/2 -translate-y-1/2 w-4 h-4 text-slate-400" aria-hidden="true" />
        <input
          type="text"
          placeholder={t('search_tutorials') || 'Search tutorials...'}
          value={search}
          onChange={(e) => setSearch(e.target.value)}
          className="w-full pl-10 pr-4 py-2.5 text-sm border border-slate-300 dark:border-slate-600 rounded-xl bg-white dark:bg-slate-800 text-slate-700 dark:text-slate-300 focus:outline-none focus:ring-2 focus:ring-blue-500 focus:border-transparent"
          aria-label="Search tutorials"
        />
      </div>

      {/* Category Filter */}
      <div className="flex flex-wrap gap-2 mb-6">
        <button
          onClick={() => setActiveCategory(null)}
          className={`flex items-center gap-1.5 px-3 py-1.5 text-xs font-medium rounded-lg border transition-all ${
            !activeCategory
              ? 'bg-blue-600 text-white border-blue-600'
              : 'bg-white dark:bg-slate-800 text-slate-600 dark:text-slate-400 border-slate-200 dark:border-slate-700 hover:bg-slate-50 dark:hover:bg-slate-700'
          }`}
        >
          <Monitor size={14} aria-hidden="true" />
          {t('all') || 'All'}
        </button>
        {categories.map((cat) => (
          <button
            key={cat}
            onClick={() => setActiveCategory(activeCategory === cat ? null : cat)}
            className={`flex items-center gap-1.5 px-3 py-1.5 text-xs font-medium rounded-lg border transition-all ${
              activeCategory === cat
                ? 'bg-blue-600 text-white border-blue-600'
                : 'bg-white dark:bg-slate-800 text-slate-600 dark:text-slate-400 border-slate-200 dark:border-slate-700 hover:bg-slate-50 dark:hover:bg-slate-700'
            }`}
          >
            {CATEGORY_ICONS[cat] || <Monitor size={14} />}
            {cat}
          </button>
        ))}
      </div>

      {/* Tutorials Grid */}
      {filteredTutorials.length === 0 ? (
        <div className="flex flex-col items-center justify-center py-16 text-center">
          <BookOpen className="w-16 h-16 text-slate-300 dark:text-slate-600 mb-4" aria-hidden="true" />
          <h3 className="text-lg font-semibold text-slate-700 dark:text-slate-300">
            {t('no_tutorials_found') || 'No tutorials found'}
          </h3>
          <p className="text-sm text-slate-500 dark:text-slate-400 mt-1">
            {search
              ? (t('try_different_search') || 'Try a different search term')
              : (t('no_tutorials_in_category') || 'No tutorials in this category yet')}
          </p>
          {search && (
            <button
              onClick={() => setSearch('')}
              className="mt-4 text-sm text-blue-600 dark:text-blue-400 hover:underline"
            >
              {t('clear_search') || 'Clear search'}
            </button>
          )}
        </div>
      ) : (
        <div className="grid grid-cols-1 sm:grid-cols-2 lg:grid-cols-3 gap-4">
          {filteredTutorials.map((video) => (
            <VideoTutorialCard key={video.id} video={video} />
          ))}
        </div>
      )}

      {/* Footer info */}
      <div className="mt-8 p-4 bg-amber-50 dark:bg-amber-900/10 border border-amber-200 dark:border-amber-800/30 rounded-xl">
        <p className="text-xs text-amber-700 dark:text-amber-400 flex items-center gap-2">
          <span className="inline-flex items-center justify-center w-5 h-5 bg-amber-200 dark:bg-amber-700 rounded-full text-xs font-bold">i</span>
          {t('tutorials_coming_soon') || 'Video tutorials are being produced. Cards marked "Coming Soon" will be available in a future update.'}
        </p>
      </div>
    </div>
  );
}
