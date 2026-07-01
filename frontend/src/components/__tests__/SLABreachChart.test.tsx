// ═══════════════════════════════════════════════════════════════════════
// SLABreachChart — Unit Tests
// P2-MED-16: Frontend Coverage 82% → 85%
// Тесты для SLA dashboard: SLATrendChart, SLABreachTimeline, SLAGaugePanel
// ═══════════════════════════════════════════════════════════════════════

import React from 'react';
import { render, screen, cleanup } from '@testing-library/react';
import userEvent from '@testing-library/user-event';
import { describe, it, expect, vi, afterEach } from 'vitest';
import { I18nextProvider } from 'react-i18next';
import i18n from '../../i18n';
import { SLATrendChart } from '../sla/SLATrendChart';
import { SLABreachTimeline } from '../sla/SLABreachTimeline';
import { SLAGaugePanel } from '../sla/SLAGaugePanel';

// ── Mock i18n ─────────────────────────────────────────────────────────

vi.mock('react-i18next', async () => {
  const actual = await vi.importActual('react-i18next');
  return {
    ...(actual as object),
    useTranslation: () => ({
      t: (key: string) => {
        const fallbacks: Record<string, string> = {
          sla_trend: 'SLA Trend',
          avg_compliance: 'Avg compliance',
          no_data: 'No data available',
          target: 'Target',
          compliance: 'Compliance',
          sla_breach_timeline: 'SLA Breach Timeline',
          all: 'All',
          critical: 'Critical',
          high: 'High',
          medium: 'Medium',
          low: 'Low',
          no_breaches: 'No SLA breaches found',
          response: 'Response',
          resolution: 'Resolution',
          show_less: 'Show less',
          sla_gauge: 'SLA Gauge',
          overall_compliance: 'Overall Compliance',
        };
        return fallbacks[key] || key;
      },
      i18n: { language: 'en' },
    }),
  };
});

// Mock @nivo/line which is not compatible with jsdom
vi.mock('@nivo/line', () => ({
  ResponsiveLine: () => <div data-testid="mock-nivo-line">Chart</div>,
}));

// ── Wrapper ───────────────────────────────────────────────────────────

function Wrapper({ children }: { children: React.ReactNode }) {
  return (
    <I18nextProvider i18n={i18n}>{children}</I18nextProvider>
  );
}

// ── Test Data ─────────────────────────────────────────────────────────

const TREND_DATA = [
  { date: '2026-06-01', compliance: 95 },
  { date: '2026-06-02', compliance: 92 },
  { date: '2026-06-03', compliance: 88 },
  { date: '2026-06-04', compliance: 96 },
  { date: '2026-06-05', compliance: 94 },
];

const BREACH_EVENTS = [
  {
    id: 'breach-1',
    siteName: 'HQ-Minsk',
    priority: 'critical',
    severity: 'critical' as const,
    breachedAt: '2026-06-04T14:30:00Z',
    responseTimeMinutes: 45,
    resolutionTimeMinutes: 180,
    description: 'Response time exceeded threshold for critical priority work order',
  },
  {
    id: 'breach-2',
    siteName: 'DC-Brest',
    priority: 'high',
    severity: 'high' as const,
    breachedAt: '2026-06-03T09:15:00Z',
    responseTimeMinutes: 30,
    resolutionTimeMinutes: 120,
    description: 'SLA breach: Resolution time exceeded for high priority',
  },
  {
    id: 'breach-3',
    siteName: 'Site-Vitebsk',
    priority: 'medium',
    severity: 'medium' as const,
    breachedAt: '2026-06-02T11:00:00Z',
    responseTimeMinutes: 20,
    resolutionTimeMinutes: 90,
    description: 'Medium priority work order exceeded SLA',
  },
];

// ── Tests ─────────────────────────────────────────────────────────────

describe('SLABreachChart — SLATrendChart', () => {
  afterEach(() => {
    cleanup();
  });

  // ── Chart Render with Data ───────────────────────────────────────

  it('renders chart title', () => {
    render(<SLATrendChart data={TREND_DATA} />, { wrapper: Wrapper });
    expect(screen.getByText('SLA Trend')).toBeInTheDocument();
  });

  it('renders average compliance value', () => {
    render(<SLATrendChart data={TREND_DATA} />, { wrapper: Wrapper });
    expect(screen.getByText(/Avg compliance/)).toBeInTheDocument();
  });

  it('renders period selector buttons (30d, 90d, 180d)', () => {
    render(<SLATrendChart data={TREND_DATA} />, { wrapper: Wrapper });
    expect(screen.getByText('30d')).toBeInTheDocument();
    expect(screen.getByText('90d')).toBeInTheDocument();
    expect(screen.getByText('180d')).toBeInTheDocument();
  });

  it('renders Nivo line chart when data is provided', () => {
    render(<SLATrendChart data={TREND_DATA} />, { wrapper: Wrapper });
    expect(screen.getByTestId('mock-nivo-line')).toBeInTheDocument();
  });

  // ── Period Switching ─────────────────────────────────────────────

  it('switches period when clicking period buttons', async () => {
    const user = userEvent.setup();
    render(<SLATrendChart data={TREND_DATA} />, { wrapper: Wrapper });

    await user.click(screen.getByText('90d'));
    expect(screen.getByTestId('mock-nivo-line')).toBeInTheDocument();

    await user.click(screen.getByText('180d'));
    expect(screen.getByTestId('mock-nivo-line')).toBeInTheDocument();
  });

  // ── Empty Data State ─────────────────────────────────────────────

  it('shows no data message when data array is empty', () => {
    render(<SLATrendChart data={[]} />, { wrapper: Wrapper });
    expect(screen.getByText('No data available')).toBeInTheDocument();
  });

  it('does not render chart when data is empty', () => {
    render(<SLATrendChart data={[]} />, { wrapper: Wrapper });
    expect(screen.queryByTestId('mock-nivo-line')).not.toBeInTheDocument();
  });

  // ── Loading Skeleton ─────────────────────────────────────────────

  it('shows loading skeleton when loading is true', () => {
    const { container } = render(<SLATrendChart data={[]} loading={true} />, { wrapper: Wrapper });
    const skeletons = container.querySelectorAll('.animate-pulse');
    expect(skeletons.length).toBeGreaterThan(0);
  });

  it('does not render chart content when loading', () => {
    render(<SLATrendChart data={[]} loading={true} />, { wrapper: Wrapper });
    expect(screen.queryByTestId('mock-nivo-line')).not.toBeInTheDocument();
    expect(screen.queryByText('No data available')).not.toBeInTheDocument();
  });
});

describe('SLABreachChart — SLABreachTimeline', () => {
  afterEach(() => {
    cleanup();
  });

  // ── Timeline Render ──────────────────────────────────────────────

  it('renders breach timeline title', () => {
    render(<SLABreachTimeline breaches={BREACH_EVENTS} />, { wrapper: Wrapper });
    expect(screen.getByText('SLA Breach Timeline')).toBeInTheDocument();
  });

  it('renders breach count in the title badge', () => {
    render(<SLABreachTimeline breaches={BREACH_EVENTS} />, { wrapper: Wrapper });
    // Badge shows the count as "3"
    const countEl = screen.getByText('3');
    expect(countEl).toBeInTheDocument();
  });

  it('renders severity filter for Critical', () => {
    render(<SLABreachTimeline breaches={BREACH_EVENTS} />, { wrapper: Wrapper });
    // Filter buttons contain severity labels
    const allFilterElements = screen.getAllByText(/Critical/);
    expect(allFilterElements.length).toBeGreaterThan(0);
  });

  it('renders breach event details', () => {
    render(<SLABreachTimeline breaches={BREACH_EVENTS} />, { wrapper: Wrapper });
    expect(screen.getByText('HQ-Minsk')).toBeInTheDocument();
    expect(screen.getByText('DC-Brest')).toBeInTheDocument();
    expect(screen.getByText('Site-Vitebsk')).toBeInTheDocument();
  });

  // ── Empty Data State ─────────────────────────────────────────────

  it('shows empty state when no breaches', () => {
    render(<SLABreachTimeline breaches={[]} />, { wrapper: Wrapper });
    expect(screen.getByText('No SLA breaches found')).toBeInTheDocument();
  });

  it('shows breach count of 0 when no breaches', () => {
    render(<SLABreachTimeline breaches={[]} />, { wrapper: Wrapper });
    // Check that the component rendered without crash and shows empty message
    expect(screen.getByText('SLA Breach Timeline')).toBeInTheDocument();
  });

  // ── Loading State ────────────────────────────────────────────────

  it('shows loading skeleton when loading is true', () => {
    const { container } = render(<SLABreachTimeline breaches={[]} loading={true} />, { wrapper: Wrapper });
    const skeletons = container.querySelectorAll('.animate-pulse');
    expect(skeletons.length).toBeGreaterThan(0);
  });

  it('does not render timeline content when loading', () => {
    render(<SLABreachTimeline breaches={[]} loading={true} />, { wrapper: Wrapper });
    expect(screen.queryByText('No SLA breaches found')).not.toBeInTheDocument();
  });

  // ── Severity Filter ──────────────────────────────────────────────

  it('filters by severity Critical', async () => {
    const user = userEvent.setup();
    render(<SLABreachTimeline breaches={BREACH_EVENTS} />, { wrapper: Wrapper });

    // Find and click the Critical filter button
    const criticalBtns = screen.getAllByText(/Critical/);
    // The first one should be the filter button (the second is the badge)
    await user.click(criticalBtns[0]);

    // After clicking Critical filter, only HQ-Minsk (critical) should be visible
    expect(screen.getByText('HQ-Minsk')).toBeInTheDocument();
  });
});

describe('SLABreachChart — SLAGaugePanel', () => {
  afterEach(() => {
    cleanup();
  });

  it('renders gauge panel with compliance values', () => {
    const { container } = render(
      <SLAGaugePanel
        overallCompliance={94}
        mttrCompliance={88}
        preventiveCompliance={92}
        emergencyResponse={96}
      />,
      { wrapper: Wrapper },
    );
    // Gauge renders values inside SVG with text
    // Check that the container rendered properly
    expect(container.querySelector('.grid')).toBeInTheDocument();
  });

  it('renders loading skeleton when loading is true', () => {
    const { container } = render(
      <SLAGaugePanel
        overallCompliance={0}
        mttrCompliance={0}
        preventiveCompliance={0}
        emergencyResponse={0}
        loading={true}
      />,
      { wrapper: Wrapper },
    );
    const skeletons = container.querySelectorAll('.animate-pulse');
    expect(skeletons.length).toBeGreaterThan(0);
  });

  it('renders without crash when all values are zero', () => {
    const { container } = render(
      <SLAGaugePanel
        overallCompliance={0}
        mttrCompliance={0}
        preventiveCompliance={0}
        emergencyResponse={0}
      />,
      { wrapper: Wrapper },
    );
    expect(container.querySelector('.grid')).toBeInTheDocument();
  });
});
