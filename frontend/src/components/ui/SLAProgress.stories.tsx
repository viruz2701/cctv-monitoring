import type { Meta, StoryObj } from '@storybook/react';
import { SLAProgress } from './SLAProgress';

const meta: Meta<typeof SLAProgress> = {
  title: 'UI/SLAProgress',
  component: SLAProgress,
  tags: ['autodocs'],
  argTypes: {
    status: {
      control: 'select',
      options: ['on_track', 'at_risk', 'breached', 'completed', undefined],
    },
  },
};

export default meta;
type Story = StoryObj<typeof SLAProgress>;

// ── Helper ────────────────────────────────────────────────────────────────

const daysAgo = (days: number): string => {
  const d = new Date();
  d.setDate(d.getDate() - days);
  return d.toISOString();
};

const daysFromNow = (days: number): string => {
  const d = new Date();
  d.setDate(d.getDate() + days);
  return d.toISOString();
};

const hoursFromNow = (hours: number): string => {
  const d = new Date();
  d.setHours(d.getHours() + hours);
  return d.toISOString();
};

const hoursAgo = (hours: number): string => {
  const d = new Date();
  d.setHours(d.getHours() - hours);
  return d.toISOString();
};

// ── Statuses ──────────────────────────────────────────────────────────────

export const OnTrack: Story = {
  args: {
    createdAt: daysAgo(5),
    deadline: daysFromNow(10),
    status: 'on_track',
  },
};

export const AtRisk: Story = {
  args: {
    createdAt: daysAgo(8),
    deadline: hoursFromNow(6),
    status: 'at_risk',
  },
};

export const Breached: Story = {
  args: {
    createdAt: daysAgo(10),
    deadline: hoursAgo(24),
    status: 'breached',
  },
};

export const Completed: Story = {
  args: {
    createdAt: daysAgo(7),
    deadline: daysAgo(1),
    status: 'completed',
  },
};

// ── Auto Status ───────────────────────────────────────────────────────────

export const AutoOnTrack: Story = {
  args: {
    createdAt: daysAgo(1),
    deadline: daysFromNow(10),
  },
};

export const AutoAtRisk: Story = {
  args: {
    createdAt: daysAgo(20),
    deadline: hoursFromNow(2),
  },
};

export const AutoBreached: Story = {
  args: {
    createdAt: daysAgo(5),
    deadline: hoursAgo(3),
  },
};

// ── All Statuses Showcase ─────────────────────────────────────────────────

export const AllStatuses: StoryObj = {
  render: () => (
    <div className="flex flex-col gap-6 p-4 max-w-md">
      <div>
        <h3 className="text-xs font-semibold text-slate-500 uppercase mb-2">On Track</h3>
        <SLAProgress createdAt={daysAgo(3)} deadline={daysFromNow(7)} status="on_track" />
      </div>
      <div>
        <h3 className="text-xs font-semibold text-slate-500 uppercase mb-2">At Risk</h3>
        <SLAProgress createdAt={daysAgo(7)} deadline={hoursFromNow(4)} status="at_risk" />
      </div>
      <div>
        <h3 className="text-xs font-semibold text-slate-500 uppercase mb-2">Breached</h3>
        <SLAProgress createdAt={daysAgo(10)} deadline={hoursAgo(12)} status="breached" />
      </div>
      <div>
        <h3 className="text-xs font-semibold text-slate-500 uppercase mb-2">Completed</h3>
        <SLAProgress createdAt={daysAgo(14)} deadline={daysAgo(2)} status="completed" />
      </div>
    </div>
  ),
};

// ── Playground ────────────────────────────────────────────────────────────

export const Playground: Story = {
  args: {
    createdAt: daysAgo(5),
    deadline: daysFromNow(5),
    status: 'on_track',
  },
};
