// P3-DX.2: Onboarding Tour для всех ролей
//
// Role-specific туры с react-joyride:
//   - Technician: WO, checklist, gatekeeper, signature
//   - Manager: dashboard, reports, SLA, team
//   - Admin: system health, users, settings, compliance
//
// Compliance: ISO 27001 A.7.2 (Awareness), IEC 62443 SR 2.1

import React, { useEffect, useState } from 'react';
import Joyride, {
  type Step,
  type CallBackProps,
  type Status,
  ACTIONS,
  EVENTS,
  STATUS,
} from 'react-joyride';
import { useTranslation } from 'react-i18next';
import { useAuth } from '../../hooks/useAuth';
import { useOnboardingStore } from '../../store/onboardingStore';

// ── Role-based tour steps ────────────────────────────────────────────

const TECHNICIAN_STEPS: Step[] = [
  {
    target: 'body',
    content: 'Добро пожаловать в CCTV Health Monitor! Этот тур покажет основные возможности для техника.',
    title: '🚀 Онбординг — Техник',
    placement: 'center',
    disableBeacon: true,
  },
  {
    target: '[data-tour="work-orders"]',
    content: 'Здесь вы видите все назначенные вам заказы. Статус, приоритет, deadline.',
    title: '📋 Рабочие заказы',
    placement: 'bottom',
  },
  {
    target: '[data-tour="wo-checklist"]',
    content: 'При выполнении ТО заполняйте чек-лист. Каждый пункт нужно отметить Pass/Fail.',
    title: '✅ Чек-лист ТО',
    placement: 'top',
  },
  {
    target: '[data-tour="gatekeeper"]',
    content: 'Gatekeeper автоматически проверяет GPS, EXIF и AI для верификации выезда.',
    title: '🔒 Gatekeeper',
    placement: 'bottom',
  },
  {
    target: '[data-tour="signature"]',
    content: 'После завершения работ — поставьте электронную подпись.',
    title: '✍️ Электронная подпись',
    placement: 'top',
  },
  {
    target: '[data-tour="offline"]',
    content: 'При отсутствии интернета — данные сохраняются локально и синхронизируются при подключении.',
    title: '📡 Offline-режим',
    placement: 'bottom',
  },
];

const MANAGER_STEPS: Step[] = [
  {
    target: 'body',
    content: 'Добро пожаловать! Вот основные инструменты для менеджера.',
    title: '🚀 Онбординг — Менеджер',
    placement: 'center',
    disableBeacon: true,
  },
  {
    target: '[data-tour="dashboard"]',
    content: 'Дашборд показывает KPI: открытые WO, SLA breaches, статус устройств.',
    title: '📊 Дашборд',
    placement: 'bottom',
  },
  {
    target: '[data-tour="sla"]',
    content: 'SLA дашборд: compliance rate, breaches, escalation chain.',
    title: '⚡ SLA & Compliance',
    placement: 'left',
  },
  {
    target: '[data-tour="reports"]',
    content: 'Генерация отчётов: PDF/Excel. Экспорт для регуляторов.',
    title: '📄 Отчёты',
    placement: 'top',
  },
  {
    target: '[data-tour="team"]',
    content: 'Управление командой: загрузка, навыки, расписание.',
    title: '👥 Команда',
    placement: 'bottom',
  },
  {
    target: '[data-tour="compliance"]',
    content: 'Compliance Shield — мониторинг соответствия регуляторным требованиям.',
    title: '🛡️ Compliance',
    placement: 'left',
  },
];

const ADMIN_STEPS: Step[] = [
  {
    target: 'body',
    content: 'Добро пожаловать! Полный доступ ко всем функциям системы.',
    title: '🚀 Онбординг — Администратор',
    placement: 'center',
    disableBeacon: true,
  },
  {
    target: '[data-tour="system-health"]',
    content: 'Состояние всех сервисов: NATS, PostgreSQL, Gatekeeper, AI.',
    title: '🖥️ System Health',
    placement: 'bottom',
  },
  {
    target: '[data-tour="users"]',
    content: 'Управление пользователями: создание, роли, permission groups.',
    title: '👤 Пользователи',
    placement: 'left',
  },
  {
    target: '[data-tour="settings"]',
    content: 'Настройки системы: регион, криптография, уведомления.',
    title: '⚙️ Настройки',
    placement: 'top',
  },
  {
    target: '[data-tour="audit"]',
    content: 'Audit log — все мутации данных с HMAC-подписью (ISO 27001 A.12.4).',
    title: '📜 Audit Log',
    placement: 'bottom',
  },
  {
    target: '[data-tour="api-keys"]',
    content: 'API ключи для внешней интеграции: P2P gateway, ServiceNow, Jira.',
    title: '🔑 API Keys',
    placement: 'left',
  },
];

// ── Theme для Joyride ────────────────────────────────────────────────

const JOYRIDE_THEME = {
  primary: '#3B82F6',
  textColor: '#1F2937',
  backgroundColor: '#FFFFFF',
  spotlightShadow: '0 0 15px rgba(59, 130, 246, 0.5)',
  overlayColor: 'rgba(0, 0, 0, 0.6)',
  arrowColor: '#FFFFFF',
};

// ── Props ────────────────────────────────────────────────────────────

interface OnboardingTourProps {
  /** Показать кнопку запуска тура */
  showTrigger?: boolean;
}

// ── Component ────────────────────────────────────────────────────────

export function OnboardingTour({ showTrigger = true }: OnboardingTourProps) {
  const { t } = useTranslation();
  const { user } = useAuth();
  const { completed, running, startTour, stopTour, markCompleted } = useOnboardingStore();
  const [run, setRun] = useState(false);
  const [steps, setSteps] = useState<Step[]>([]);

  // Определяем шаги по роли
  useEffect(() => {
    const role = user?.role || 'technician';
    switch (role) {
      case 'technician':
        setSteps(TECHNICIAN_STEPS);
        break;
      case 'manager':
        setSteps(MANAGER_STEPS);
        break;
      case 'admin':
      case 'support':
        setSteps(ADMIN_STEPS);
        break;
      default:
        setSteps(TECHNICIAN_STEPS);
    }
  }, [user?.role]);

  // Автостарт при первом входе
  useEffect(() => {
    if (!completed && !running && user) {
      const timer = setTimeout(() => setRun(true), 500);
      startTour();
      return () => clearTimeout(timer);
    }
  }, [completed, running, user, startTour]);

  const handleJoyrideCallback = (data: CallBackProps) => {
    const { action, index, status, type } = data;

    if ([STATUS.FINISHED, STATUS.SKIPPED].includes(status as Status)) {
      setRun(false);
      markCompleted();
    }

    if (action === ACTIONS.CLOSE || action === ACTIONS.SKIP) {
      setRun(false);
      markCompleted();
    }
  };

  const handleStartTour = () => {
    startTour();
    setRun(true);
  };

  return (
    <>
      <Joyride
        callback={handleJoyrideCallback}
        continuous
        disableCloseOnEsc
        disableOverlayClose
        hideBackButton={false}
        locale={{
          back: t('common:back'),
          close: t('common:close'),
          last: t('onboarding:done'),
          next: t('common:next'),
          skip: t('onboarding:skip'),
        }}
        run={run}
        scrollToFirstStep
        showProgress
        showSkipButton
        steps={steps}
        styles={{
          options: {
            primaryColor: JOYRIDE_THEME.primary,
            textColor: JOYRIDE_THEME.textColor,
            backgroundColor: JOYRIDE_THEME.backgroundColor,
            arrowColor: JOYRIDE_THEME.arrowColor,
            overlayColor: JOYRIDE_THEME.overlayColor,
            spotlightShadow: JOYRIDE_THEME.spotlightShadow,
            zIndex: 10000,
          },
          buttonNext: {
            backgroundColor: JOYRIDE_THEME.primary,
            color: '#fff',
            fontSize: 14,
          },
          buttonBack: {
            color: JOYRIDE_THEME.textColor,
            fontSize: 14,
          },
          buttonSkip: {
            color: '#9CA3AF',
            fontSize: 14,
          },
          tooltipContent: {
            padding: 20,
            fontSize: 14,
            lineHeight: 1.5,
          },
          tooltipTitle: {
            fontSize: 16,
            fontWeight: 600,
            marginBottom: 8,
          },
        }}
      />

      {showTrigger && !completed && !run && (
        <button
          onClick={handleStartTour}
          className="fixed bottom-4 right-4 z-50 px-4 py-2 bg-blue-600 text-white rounded-lg shadow-lg hover:bg-blue-700 transition-colors flex items-center gap-2 text-sm"
          aria-label={t('onboarding:start_tour')}
          data-tour="restart-tour"
        >
          <svg className="w-4 h-4" fill="none" viewBox="0 0 24 24" stroke="currentColor" strokeWidth={2}>
            <path strokeLinecap="round" strokeLinejoin="round" d="M13 10V3L4 14h7v7l9-11h-7z" />
          </svg>
          {t('onboarding:restart_tour')}
        </button>
      )}
    </>
  );
}
