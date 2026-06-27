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

const now = new Date();
const hoursAgo = (h: number) => new Date(now.getTime() - h * 3600000).toISOString();
const hoursLater = (h: number) => new Date(now.getTime() + h * 3600000).toISOString();

// ── On Track (Green) ─────────────────────────────────────────────────────

export const OnTrack: Story = {
  args: {
    createdAt: hoursAgo(2),
    deadline: hoursLater(22),
    status: 'on_track',
  },
};

// ── At Risk (Yellow) ─────────────────────────────────────────────────────

export const AtRisk: Story = {
  args: {
    createdAt: hoursAgo(18),
    deadline: hoursLater(4),
    status: 'at_risk',
  },
};

// ── Breached (Red) ───────────────────────────────────────────────────────

export const Breached: Story = {
  args: {
    createdAt: hoursAgo(48),
    deadline: hoursAgo(2),
    status: 'breached',
  },
};

// ── Completed ────────────────────────────────────────────────────────────

export const Completed: Story = {
  args: {
    createdAt: hoursAgo(72),
    deadline: hoursAgo(48),
    status: 'completed',
  },
};

// ── Auto Status (Almost Breached) ────────────────────────────────────────

export const AutoStatus: Story = {
  args: {
    createdAt: hoursAgo(23),
    deadline: hoursLater(1),
  },
};

// ── Early Stage ──────────────────────────────────────────────────────────

export const EarlyStage: Story = {
  args: {
    createdAt: hoursAgo(1),
    deadline: hoursLater(47),
    status: 'on_track',
  },
};

// ── Overdue Auto ─────────────────────────────────────────────────────────

export const OverdueAuto: Story = {
  args: {
    createdAt: hoursAgo(48),
    deadline: hoursAgo(1),
  },
};

// ── Playground ───────────────────────────────────────────────────────────

export const Playground: Story = {
  args: {
    createdAt: hoursAgo(12),
    deadline: hoursLater(12),
    status: 'on_track',
  },
};
