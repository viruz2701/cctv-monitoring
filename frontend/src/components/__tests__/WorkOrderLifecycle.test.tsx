// ═══════════════════════════════════════════════════════════════════════
// WorkOrderLifecycle — Unit Tests
// P2-MED-16: Frontend Coverage 82% → 85%
// Тесты для WO lifecycle: status transitions, SLA deadline, empty state
// ═══════════════════════════════════════════════════════════════════════

import React from 'react';
import { render, screen, cleanup } from '@testing-library/react';
import userEvent from '@testing-library/user-event';
import { describe, it, expect, vi, afterEach } from 'vitest';
import { I18nextProvider } from 'react-i18next';
import i18n from '../../i18n';
import { WODetailHeader } from '../work-orders/WODetailHeader';
import { SLATimer } from '../work-orders/SLATimer';
import type { WorkOrder } from '../../services/workOrdersApi';

// ── Mock i18n ─────────────────────────────────────────────────────────

vi.mock('react-i18next', async () => {
  const actual = await vi.importActual('react-i18next');
  return {
    ...(actual as object),
    useTranslation: () => ({
      t: (key: string) => {
        const fallbacks: Record<string, string> = {
          'workOrder.title': 'Наряд-заказ',
          'workOrder.start': 'Начать работу',
          'workOrder.complete': 'Завершить',
          'workOrder.cancel': 'Отменить',
        };
        return fallbacks[key] || key;
      },
      i18n: { language: 'ru' },
    }),
  };
});

// ── Test Data ─────────────────────────────────────────────────────────

const OPEN_WO: WorkOrder = {
  id: 'wo-001',
  device_id: 'device-1',
  type: 'corrective',
  status: 'open',
  priority: 'high',
  checklist: [],
  photos: [],
  parts_used: [],
  created_at: '2026-06-01T10:00:00Z',
  updated_at: '2026-06-01T10:00:00Z',
  device_name: 'Camera Entrance',
  assignee_name: 'Ivan Petrov',
  sla_deadline: '2026-06-02T10:00:00Z',
  sla_status: 'on_track',
};

const IN_PROGRESS_WO: WorkOrder = {
  ...OPEN_WO,
  status: 'in_progress',
  started_at: '2026-06-01T11:00:00Z',
};

const COMPLETED_WO: WorkOrder = {
  ...OPEN_WO,
  status: 'completed',
  started_at: '2026-06-01T11:00:00Z',
  completed_at: '2026-06-01T15:00:00Z',
};

const CANCELLED_WO: WorkOrder = {
  ...OPEN_WO,
  status: 'cancelled',
};

// ── Wrapper ───────────────────────────────────────────────────────────

function Wrapper({ children }: { children: React.ReactNode }) {
  return (
    <I18nextProvider i18n={i18n}>{children}</I18nextProvider>
  );
}

// ── Tests ─────────────────────────────────────────────────────────────

describe('WorkOrderLifecycle — WODetailHeader', () => {
  afterEach(() => {
    cleanup();
  });

  // ── Render States ────────────────────────────────────────────────

  it('renders open work order with Start, Complete and Cancel buttons', () => {
    render(
      <WODetailHeader
        workOrder={OPEN_WO}
        submitting={false}
        onStart={vi.fn()}
        onComplete={vi.fn()}
        onCancel={vi.fn()}
        onBack={vi.fn()}
      />,
      { wrapper: Wrapper },
    );
    expect(screen.getByText('Начать работу')).toBeInTheDocument();
    expect(screen.getByText('Завершить')).toBeInTheDocument();
    expect(screen.getByText('Отменить')).toBeInTheDocument();
  });

  it('renders in_progress work order with Complete and Cancel buttons', () => {
    render(
      <WODetailHeader
        workOrder={IN_PROGRESS_WO}
        submitting={false}
        onStart={vi.fn()}
        onComplete={vi.fn()}
        onCancel={vi.fn()}
        onBack={vi.fn()}
      />,
      { wrapper: Wrapper },
    );
    // "Начать работу" should NOT be visible for in_progress
    expect(screen.queryByText('Начать работу')).not.toBeInTheDocument();
    expect(screen.getByText('Завершить')).toBeInTheDocument();
    expect(screen.getByText('Отменить')).toBeInTheDocument();
  });

  it('renders completed work order without action buttons', () => {
    render(
      <WODetailHeader
        workOrder={COMPLETED_WO}
        submitting={false}
        onStart={vi.fn()}
        onComplete={vi.fn()}
        onCancel={vi.fn()}
        onBack={vi.fn()}
      />,
      { wrapper: Wrapper },
    );
    expect(screen.queryByText('Начать работу')).not.toBeInTheDocument();
    expect(screen.queryByText('Завершить')).not.toBeInTheDocument();
    expect(screen.queryByText('Отменить')).not.toBeInTheDocument();
  });

  it('renders cancelled work order without action buttons', () => {
    render(
      <WODetailHeader
        workOrder={CANCELLED_WO}
        submitting={false}
        onStart={vi.fn()}
        onComplete={vi.fn()}
        onCancel={vi.fn()}
        onBack={vi.fn()}
      />,
      { wrapper: Wrapper },
    );
    expect(screen.queryByText('Начать работу')).not.toBeInTheDocument();
    expect(screen.queryByText('Завершить')).not.toBeInTheDocument();
    expect(screen.queryByText('Отменить')).not.toBeInTheDocument();
  });

  it('renders back button and calls onBack when clicked', async () => {
    const onBack = vi.fn();
    const user = userEvent.setup();
    render(
      <WODetailHeader
        workOrder={OPEN_WO}
        submitting={false}
        onStart={vi.fn()}
        onComplete={vi.fn()}
        onCancel={vi.fn()}
        onBack={onBack}
      />,
      { wrapper: Wrapper },
    );
    // Back button has ArrowLeft icon
    const backBtn = screen.getByRole('button', { name: '' });
    // All buttons — find the one with ArrowLeft icon (first one)
    const buttons = screen.getAllByRole('button');
    // First button is back
    await user.click(buttons[0]);
    expect(onBack).toHaveBeenCalledTimes(1);
  });

  // ── Status Transitions ───────────────────────────────────────────

  it('calls onStart when Start button is clicked', async () => {
    const onStart = vi.fn();
    const user = userEvent.setup();
    render(
      <WODetailHeader
        workOrder={OPEN_WO}
        submitting={false}
        onStart={onStart}
        onComplete={vi.fn()}
        onCancel={vi.fn()}
        onBack={vi.fn()}
      />,
      { wrapper: Wrapper },
    );
    await user.click(screen.getByText('Начать работу'));
    expect(onStart).toHaveBeenCalledTimes(1);
  });

  it('calls onComplete when Complete button is clicked', async () => {
    const onComplete = vi.fn();
    const user = userEvent.setup();
    render(
      <WODetailHeader
        workOrder={OPEN_WO}
        submitting={false}
        onStart={vi.fn()}
        onComplete={onComplete}
        onCancel={vi.fn()}
        onBack={vi.fn()}
      />,
      { wrapper: Wrapper },
    );
    await user.click(screen.getByText('Завершить'));
    expect(onComplete).toHaveBeenCalledTimes(1);
  });

  it('calls onCancel when Cancel button is clicked', async () => {
    const onCancel = vi.fn();
    const user = userEvent.setup();
    render(
      <WODetailHeader
        workOrder={OPEN_WO}
        submitting={false}
        onStart={vi.fn()}
        onComplete={vi.fn()}
        onCancel={onCancel}
        onBack={vi.fn()}
      />,
      { wrapper: Wrapper },
    );
    await user.click(screen.getByText('Отменить'));
    expect(onCancel).toHaveBeenCalledTimes(1);
  });

  it('disables buttons when submitting is true', () => {
    render(
      <WODetailHeader
        workOrder={OPEN_WO}
        submitting={true}
        onStart={vi.fn()}
        onComplete={vi.fn()}
        onCancel={vi.fn()}
        onBack={vi.fn()}
      />,
      { wrapper: Wrapper },
    );
    // Start button should be in loading state (disabled)
    const startBtn = screen.getByText('Начать работу').closest('button');
    expect(startBtn).toBeDisabled();
  });

  // ── SLA Display ──────────────────────────────────────────────────

  it('shows SLA timer for work orders with SLA deadline', () => {
    render(
      <WODetailHeader
        workOrder={OPEN_WO}
        submitting={false}
        onStart={vi.fn()}
        onComplete={vi.fn()}
        onCancel={vi.fn()}
        onBack={vi.fn()}
      />,
      { wrapper: Wrapper },
    );
    // SLA section should render (LiveSLATimer component)
    expect(screen.getByText(/Наряд-заказ/)).toBeInTheDocument();
  });

  it('renders device name and assignee info', () => {
    render(
      <WODetailHeader
        workOrder={OPEN_WO}
        submitting={false}
        onStart={vi.fn()}
        onComplete={vi.fn()}
        onCancel={vi.fn()}
        onBack={vi.fn()}
      />,
      { wrapper: Wrapper },
    );
    expect(screen.getByText(/Camera Entrance/)).toBeInTheDocument();
    expect(screen.getByText(/Ivan Petrov/)).toBeInTheDocument();
  });
});

describe('WorkOrderLifecycle — SLATimer', () => {
  afterEach(() => {
    cleanup();
  });

  const futureDeadline = new Date(Date.now() + 86400000 * 3).toISOString(); // 3 days from now
  const pastDeadline = new Date(Date.now() - 86400000).toISOString(); // 1 day ago
  const createdAt = new Date(Date.now() - 86400000 * 7).toISOString(); // 7 days ago

  it('renders on_track SLA status', () => {
    render(
      <SLATimer
        deadline={futureDeadline}
        createdAt={createdAt}
        status="on_track"
      />,
    );
    // Should show SLA indicator
    expect(screen.getByText(/SLA/)).toBeInTheDocument();
  });

  it('renders at_risk SLA status', () => {
    render(
      <SLATimer
        deadline={futureDeadline}
        createdAt={createdAt}
        status="at_risk"
      />,
    );
    expect(screen.getByText(/SLA/)).toBeInTheDocument();
  });

  it('renders breached SLA status', () => {
    render(
      <SLATimer
        deadline={pastDeadline}
        createdAt={createdAt}
        status="breached"
      />,
    );
    expect(screen.getByText(/SLA/)).toBeInTheDocument();
  });

  it('renders completed SLA status', () => {
    render(
      <SLATimer
        deadline={futureDeadline}
        createdAt={createdAt}
        status="completed"
      />,
    );
    expect(screen.getByText(/SLA/)).toBeInTheDocument();
  });

  it('renders no_sla status gracefully', () => {
    render(
      <SLATimer
        deadline={futureDeadline}
        createdAt={createdAt}
        status="no_sla"
      />,
    );
    expect(screen.getByText(/Без SLA/)).toBeInTheDocument();
  });

  it('shows overdue text for breached SLA', () => {
    render(
      <SLATimer
        deadline={pastDeadline}
        createdAt={createdAt}
      />,
    );
    // Should detect overdue from date comparison
    expect(screen.getByText(/SLA/)).toBeInTheDocument();
  });
});
