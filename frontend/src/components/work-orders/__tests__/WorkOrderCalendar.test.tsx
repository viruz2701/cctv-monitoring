import React from 'react';
import { render, screen, fireEvent, within } from '@testing-library/react';
import { describe, it, expect, vi, beforeEach, afterEach } from 'vitest';
import '@testing-library/jest-dom';
import { WorkOrderCalendar } from '../WorkOrderCalendar';
import type { WorkOrder } from '../../../services/workOrdersApi';
import type { User as ApiUser } from '../../../services/api';

// ═══════════════════════════════════════════════════════════════════════
// Mocks
// ═══════════════════════════════════════════════════════════════════════

vi.mock('react-i18next', () => ({
  useTranslation: () => ({
    t: (key: string) => key,
  }),
}));

// Mock FullCalendar (heavy external dependency)
vi.mock('@fullcalendar/react', () => ({
  default: ({ events, eventClick, select, eventDrop, eventContent, eventDidMount }: any) => (
    <div data-testid="full-calendar">
      <span data-testid="event-count">{events?.length ?? 0}</span>
      <div data-testid="calendar-events">
        {events?.map((evt: any) => (
          <div
            key={evt.id}
            data-testid={`cal-event-${evt.id}`}
            data-start={evt.start}
            data-bgcolor={evt.backgroundColor}
            data-bordercolor={evt.borderColor}
            data-textcolor={evt.textColor}
            data-classnames={evt.classNames?.join(' ')}
          >
            {evt.title}
          </div>
        ))}
      </div>
    </div>
  ),
}));

vi.mock('@fullcalendar/daygrid', () => ({ default: vi.fn() }));
vi.mock('@fullcalendar/interaction', () => ({ default: vi.fn() }));

// ═══════════════════════════════════════════════════════════════════════
// Fixtures
// ═══════════════════════════════════════════════════════════════════════

const mockTechnicians: ApiUser[] = [
  { id: 'tech-1', username: 'tech1', name: 'Tech One', email: 'tech1@test.com', role: 'technician' },
  { id: 'tech-2', username: 'tech2', name: 'Tech Two', email: 'tech2@test.com', role: 'technician' },
] as ApiUser[];

const mockWorkOrders: WorkOrder[] = [
  {
    id: 'wo-1',
    device_id: 'cam-1',
    device_name: 'Camera 1',
    type: 'corrective',
    status: 'open',
    priority: 'critical',
    assigned_to: 'tech-1',
    created_at: '2026-06-01T08:00:00Z',
    updated_at: '2026-06-01T08:00:00Z',
    sla_deadline: '2026-06-15T17:00:00Z',
    checklist: [],
    photos: [],
    parts_used: [],
  },
  {
    id: 'wo-2',
    device_id: 'cam-2',
    device_name: 'Camera 2',
    type: 'preventive',
    status: 'in_progress',
    priority: 'medium',
    assigned_to: 'tech-2',
    created_at: '2026-06-10T08:00:00Z',
    updated_at: '2026-06-10T08:00:00Z',
    sla_deadline: '2026-06-20T17:00:00Z',
    checklist: [],
    photos: [],
    parts_used: [],
  },
  {
    id: 'wo-3',
    device_id: 'cam-3',
    device_name: 'Camera 3',
    type: 'emergency',
    status: 'completed',
    priority: 'low',
    assigned_to: null,
    created_at: '2026-06-05T08:00:00Z',
    updated_at: '2026-06-05T08:00:00Z',
    sla_deadline: null,
    checklist: [],
    photos: [],
    parts_used: [],
  },
] as unknown as WorkOrder[];

const defaultProps = {
  workOrders: mockWorkOrders,
  technicians: mockTechnicians,
  onDateChange: vi.fn(),
  onEventClick: vi.fn(),
  onDateClick: vi.fn(),
};

// ═══════════════════════════════════════════════════════════════════════
// Helpers
// ═══════════════════════════════════════════════════════════════════════

function getToggleButtons(container: HTMLElement) {
  const toggleGroup = container.querySelector('[role="radiogroup"]');
  if (!toggleGroup) return { deadline: null, creation: null };
  const buttons = toggleGroup.querySelectorAll('button[role="radio"]');
  return {
    deadline: buttons[0] as HTMLButtonElement | null,
    creation: buttons[1] as HTMLButtonElement | null,
  };
}

// ═══════════════════════════════════════════════════════════════════════
// Tests
// ═══════════════════════════════════════════════════════════════════════

describe('WorkOrderCalendar — Date Mode Toggle (P1-UX.6)', () => {
  beforeEach(() => {
    localStorage.clear();
  });

  afterEach(() => {
    localStorage.clear();
  });

  // ── Render & toggle existence ───────────────────────────────────────
  it('renders date mode toggle with deadline and creation buttons', () => {
    const { container } = render(<WorkOrderCalendar {...defaultProps} />);
    const { deadline, creation } = getToggleButtons(container);

    expect(deadline).toBeInTheDocument();
    expect(creation).toBeInTheDocument();
    expect(deadline).toHaveAttribute('role', 'radio');
    expect(creation).toHaveAttribute('role', 'radio');
  });

  // ── Default mode ──────────────────────────────────────────────────
  it('defaults to deadline mode', () => {
    const { container } = render(<WorkOrderCalendar {...defaultProps} />);
    const { deadline, creation } = getToggleButtons(container);

    expect(deadline).toHaveAttribute('aria-checked', 'true');
    expect(creation).toHaveAttribute('aria-checked', 'false');
  });

  // ── Toggle interaction ────────────────────────────────────────────
  it('switches to creation mode when creation button is clicked', () => {
    const { container } = render(<WorkOrderCalendar {...defaultProps} />);
    const { deadline, creation } = getToggleButtons(container);

    fireEvent.click(creation!);

    expect(creation).toHaveAttribute('aria-checked', 'true');
    expect(deadline).toHaveAttribute('aria-checked', 'false');
  });

  it('switches back to deadline mode after toggling to creation', () => {
    const { container } = render(<WorkOrderCalendar {...defaultProps} />);
    const { deadline, creation } = getToggleButtons(container);

    fireEvent.click(creation!);
    fireEvent.click(deadline!);

    expect(deadline).toHaveAttribute('aria-checked', 'true');
    expect(creation).toHaveAttribute('aria-checked', 'false');
  });

  // ── Calendar events reflect date mode ─────────────────────────────
  it('shows events with sla_deadline in deadline mode', () => {
    render(<WorkOrderCalendar {...defaultProps} />);

    // wo-3 has no deadline, so only wo-1 and wo-2 should appear
    const events = screen.getByTestId('calendar-events');
    expect(within(events).queryByTestId('cal-event-wo-1')).toBeInTheDocument();
    expect(within(events).queryByTestId('cal-event-wo-2')).toBeInTheDocument();
    expect(within(events).queryByTestId('cal-event-wo-3')).not.toBeInTheDocument();
  });

  // ── localStorage persistence ──────────────────────────────────────
  it('persists deadline mode selection to localStorage', () => {
    const { container } = render(<WorkOrderCalendar {...defaultProps} />);
    const { creation } = getToggleButtons(container);

    fireEvent.click(creation!);
    expect(localStorage.getItem('woCalendar_dateMode')).toBe('"creation"');
  });

  it('persists creation mode selection to localStorage', () => {
    localStorage.setItem('woCalendar_dateMode', JSON.stringify('creation'));
    const { container } = render(<WorkOrderCalendar {...defaultProps} />);
    const { deadline, creation } = getToggleButtons(container);

    expect(creation).toHaveAttribute('aria-checked', 'true');
    expect(deadline).toHaveAttribute('aria-checked', 'false');
  });

  // ── Controlled mode (props) ───────────────────────────────────────
  it('uses controlled dateMode when provided via props', () => {
    const handleChange = vi.fn();
    const { container } = render(
      <WorkOrderCalendar
        {...defaultProps}
        dateMode="creation"
        onDateModeChange={handleChange}
      />,
    );
    const { deadline, creation } = getToggleButtons(container);

    expect(creation).toHaveAttribute('aria-checked', 'true');
    expect(deadline).toHaveAttribute('aria-checked', 'false');
  });

  it('calls onDateModeChange when controlled toggle is clicked', () => {
    const handleChange = vi.fn();
    const { container } = render(
      <WorkOrderCalendar
        {...defaultProps}
        dateMode="deadline"
        onDateModeChange={handleChange}
      />,
    );
    const { creation } = getToggleButtons(container);

    fireEvent.click(creation!);
    expect(handleChange).toHaveBeenCalledWith('creation');
  });

  // ── Color coding (via classNames — always reflects date mode) ────
  it('adds wo-date-deadline class in deadline mode', () => {
    render(<WorkOrderCalendar {...defaultProps} />);
    const event1 = screen.getByTestId('cal-event-wo-1');
    expect(event1).toHaveAttribute('data-classnames', expect.stringContaining('wo-date-deadline'));
  });

  it('adds wo-date-creation class in creation mode', () => {
    render(<WorkOrderCalendar {...defaultProps} />);
    const { creation } = getToggleButtons(
      (screen.getByTestId('full-calendar').closest('.work-order-calendar') as HTMLElement)!
    );

    fireEvent.click(creation!);

    const event1 = screen.getByTestId('cal-event-wo-1');
    expect(event1).toHaveAttribute('data-classnames', expect.stringContaining('wo-date-creation'));
  });

  // ── Legend ─────────────────────────────────────────────────────────
  it('shows date mode legend with deadline and creation colors', () => {
    render(<WorkOrderCalendar {...defaultProps} />);
    expect(screen.getByText('deadline_legend')).toBeInTheDocument();
    expect(screen.getByText('creation_legend')).toBeInTheDocument();
    expect(screen.getByText('date_mode_hint')).toBeInTheDocument();
  });
});
