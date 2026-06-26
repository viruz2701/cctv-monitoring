// ═══════════════════════════════════════════════════════════════════════
// Onboarding Tour
// UX-14.1.6: react-joyride для новых пользователей
//
// P3-3.1: Onboarding Tour Role Adaptation
//   - Шаги зависят от роли пользователя (Technician → только relevant, Admin → все)
//   - Conditional steps (оценка condition() перед показом)
//   - Skip button для experienced users
//   - Tour completion tracking (localStorage) через useOnboardingStore
// ═══════════════════════════════════════════════════════════════════════

import React, { useEffect, useState, useCallback, useMemo } from 'react';
import { Joyride, ACTIONS, EVENTS, STATUS } from 'react-joyride';
import type { Step, EventData } from 'react-joyride';
import { useOnboardingStore } from '../../store/onboardingStore';
import { useAuth } from '../../hooks/useAuth';

// ═══════════════════════════════════════════════════════════════════════
// Tour Step — расширенный тип с role-based фильтрацией
// ═══════════════════════════════════════════════════════════════════════

interface TourStep extends Step {
  /** Роли, которым показывается этот шаг */
  roles: string[];
  /**
   * Опциональное условие для динамического скрытия шага.
   * Вызывается при монтировании. Если вернёт false — шаг пропускается.
   */
  condition?: () => boolean;
}

// ═══════════════════════════════════════════════════════════════════════
// Tour Steps — role-aware
// ═══════════════════════════════════════════════════════════════════════

const TOUR_STEPS: TourStep[] = [
  {
    target: 'body',
    roles: ['admin', 'manager', 'technician', 'viewer', 'support', 'owner'],
    content: (
      <div>
        <h3 className="text-lg font-semibold text-slate-900 dark:text-white mb-2">
          Welcome to CCTV Health Monitor
        </h3>
        <p className="text-sm text-slate-600 dark:text-slate-300">
          This quick tour will walk you through the key features of the platform.
          Let's get started!
        </p>
      </div>
    ),
    placement: 'center',
  },
  {
    target: 'aside',
    roles: ['admin', 'manager', 'technician', 'viewer', 'support', 'owner'],
    content: (
      <div>
        <h3 className="text-lg font-semibold text-slate-900 dark:text-white mb-2">
          Navigation Sidebar
        </h3>
        <p className="text-sm text-slate-600 dark:text-slate-300">
          Use the sidebar to navigate between sections. It adapts to your role —
          admins see additional management pages.
        </p>
        <p className="text-xs text-slate-400 dark:text-slate-500 mt-2">
          💡 You can collapse it using the toggle button on the right edge.
        </p>
      </div>
    ),
    placement: 'right',
  },
  {
    target: 'main',
    roles: ['admin', 'manager', 'technician', 'viewer', 'support', 'owner'],
    content: (
      <div>
        <h3 className="text-lg font-semibold text-slate-900 dark:text-white mb-2">
          Dashboard
        </h3>
        <p className="text-sm text-slate-600 dark:text-slate-300">
          The Dashboard gives you a real-time overview of your CCTV infrastructure:
          device health, active alerts, SLA compliance, and key metrics.
        </p>
      </div>
    ),
    placement: 'left',
  },
  {
    target: 'body',
    roles: ['admin', 'manager', 'technician', 'viewer', 'support', 'owner'],
    content: (
      <div>
        <h3 className="text-lg font-semibold text-slate-900 dark:text-white mb-2">
          Device Management
        </h3>
        <p className="text-sm text-slate-600 dark:text-slate-300">
          Navigate to <strong>Devices</strong> to view, filter, and manage all
          connected CCTV equipment. Each device shows real-time status, health,
          and recording state.
        </p>
        <p className="text-xs text-slate-400 dark:text-slate-500 mt-2">
          🔍 Use the search bar to quickly find devices by name or location.
        </p>
      </div>
    ),
    placement: 'center',
  },
  {
    target: 'body',
    roles: ['admin', 'manager', 'technician', 'support', 'owner'],
    content: (
      <div>
        <h3 className="text-lg font-semibold text-slate-900 dark:text-white mb-2">
          Alerts & Notifications
        </h3>
        <p className="text-sm text-slate-600 dark:text-slate-300">
          The <strong>Alerts</strong> section shows active alarms. Use the bell
          icon in the header for recent notifications. Critical alerts are
          highlighted with priority indicators.
        </p>
      </div>
    ),
    placement: 'center',
  },
  {
    target: 'body',
    roles: ['admin', 'manager', 'technician'],
    content: (
      <div>
        <h3 className="text-lg font-semibold text-slate-900 dark:text-white mb-2">
          Work Orders (CMMS)
        </h3>
        <p className="text-sm text-slate-600 dark:text-slate-300">
          Create and manage maintenance work orders. Track status, assign
          technicians, log spare parts usage, and monitor SLA compliance —
          all in one place.
        </p>
      </div>
    ),
    placement: 'center',
  },
  {
    target: 'body',
    roles: ['admin', 'manager'],
    content: (
      <div>
        <h3 className="text-lg font-semibold text-slate-900 dark:text-white mb-2">
          Analytics
        </h3>
        <p className="text-sm text-slate-600 dark:text-slate-300">
          View performance reports, device uptime trends, and SLA insights
          in the Analytics section. Filter by date range and export reports.
        </p>
      </div>
    ),
    placement: 'center',
  },
  {
    target: 'body',
    roles: ['admin'],
    content: (
      <div>
        <h3 className="text-lg font-semibold text-slate-900 dark:text-white mb-2">
          Users & Settings
        </h3>
        <p className="text-sm text-slate-600 dark:text-slate-300">
          As an <strong>Admin</strong>, you have access to <strong>User Management</strong>
          {' '}and <strong>System Settings</strong>. Manage roles, permissions,
          and platform configuration.
        </p>
      </div>
    ),
    placement: 'center',
  },
  {
    target: 'body',
    roles: ['admin', 'manager', 'technician', 'viewer', 'support', 'owner'],
    content: (
      <div>
        <h3 className="text-lg font-semibold text-slate-900 dark:text-white mb-2">
          ⌘K Command Palette
        </h3>
        <p className="text-sm text-slate-600 dark:text-slate-300">
          Press <kbd className="px-1.5 py-0.5 bg-slate-200 dark:bg-slate-700 rounded text-xs font-mono">⌘K</kbd>
          {' '}or <kbd className="px-1.5 py-0.5 bg-slate-200 dark:bg-slate-700 rounded text-xs font-mono">Ctrl+K</kbd>
          {' '}anytime to quickly navigate anywhere in the app. Search pages, actions,
          and settings with fuzzy matching.
        </p>
      </div>
    ),
    placement: 'center',
  },
  {
    target: 'body',
    roles: ['admin', 'manager', 'technician', 'viewer', 'support', 'owner'],
    content: (
      <div>
        <h3 className="text-lg font-semibold text-slate-900 dark:text-white mb-2">
          You're All Set! 🎉
        </h3>
        <p className="text-sm text-slate-600 dark:text-slate-300">
          You can restart this tour anytime from <strong>Settings → General</strong>.
          Happy monitoring!
        </p>
      </div>
    ),
    placement: 'center',
  },
];

// ═══════════════════════════════════════════════════════════════════════
// Joyride locale
// ═══════════════════════════════════════════════════════════════════════

const JOYRIDE_LOCALE = {
  back: 'Back',
  close: 'Close',
  last: 'Done',
  next: 'Next',
  skip: 'Skip Tour',
};

// ═══════════════════════════════════════════════════════════════════════
// Styles
// ═══════════════════════════════════════════════════════════════════════

const JOYRIDE_STYLES = {
  options: {
    primaryColor: '#2563eb',
    textColor: '#1e293b',
    backgroundColor: '#ffffff',
    arrowColor: '#ffffff',
    overlayColor: 'rgba(15, 23, 42, 0.5)',
    zIndex: 120,
    width: 400,
  },
  tooltipContainer: {
    textAlign: 'left' as const,
  },
  tooltipContent: {
    padding: '12px 4px',
  },
  buttonBack: {
    color: '#64748b',
    fontSize: '13px',
    fontWeight: 500,
    padding: '8px 16px',
    borderRadius: '8px',
  },
  buttonSkip: {
    color: '#94a3b8',
    fontSize: '13px',
    fontWeight: 500,
    padding: '8px 16px',
    borderRadius: '8px',
  },
  buttonNext: {
    fontSize: '13px',
    fontWeight: 600,
    padding: '8px 20px',
    borderRadius: '8px',
  },
};

// ═══════════════════════════════════════════════════════════════════════
// Default options for all steps
// ═══════════════════════════════════════════════════════════════════════

const DEFAULT_OPTIONS = {
  showProgress: true,
  skipBeacon: true,
  hideOverlay: false,
  overlayClickAction: 'close' as const,
};

// ═══════════════════════════════════════════════════════════════════════
// Component
// ═══════════════════════════════════════════════════════════════════════

export function OnboardingTour() {
  const { completed, running, markCompleted, startTour, stopTour } =
    useOnboardingStore();
  const { user } = useAuth();
  const [hasRun, setHasRun] = useState(false);

  // Роль пользователя (fallback для анонимного доступа)
  const role = user?.role ?? 'viewer';

  // Фильтрация шагов по роли и condition()
  const steps = useMemo<Step[]>(() => {
    return TOUR_STEPS.filter((step) => {
      // 1. Проверка роли
      if (!step.roles.includes(role)) return false;

      // 2. Проверка динамического условия (condition)
      if (step.condition && !step.condition()) return false;

      return true;
    });
  }, [role]);

  // Auto-start tour для новых пользователей
  useEffect(() => {
    if (!completed && !hasRun && steps.length > 0) {
      setHasRun(true);
      const timer = setTimeout(() => {
        startTour();
      }, 1500);
      return () => clearTimeout(timer);
    }
  }, [completed, hasRun, startTour, steps.length]);

  const handleEvent = useCallback(
    (data: EventData) => {
      const { action, status, type } = data;

      if (action === ACTIONS.CLOSE || action === ACTIONS.SKIP) {
        stopTour();
      }

      if (
        type === EVENTS.TOUR_END ||
        status === STATUS.FINISHED ||
        status === STATUS.SKIPPED
      ) {
        markCompleted();
      }
    },
    [stopTour, markCompleted]
  );

  // Если шагов нет после фильтрации — не рендерим
  if (steps.length === 0) return null;

  return (
    <Joyride
      steps={steps}
      run={running}
      continuous
      locale={JOYRIDE_LOCALE}
      styles={JOYRIDE_STYLES}
      options={DEFAULT_OPTIONS}
      onEvent={handleEvent}
    />
  );
}
