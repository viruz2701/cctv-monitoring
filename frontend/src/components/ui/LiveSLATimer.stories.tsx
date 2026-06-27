import type { Meta, StoryObj } from '@storybook/react';
import { LiveSLATimer } from './LiveSLATimer';

const meta: Meta<typeof LiveSLATimer> = {
  title: 'UI/LiveSLATimer',
  component: LiveSLATimer,
  tags: ['autodocs'],
  argTypes: {
    status: {
      control: 'select',
      options: ['on_track', 'at_risk', 'breached', 'completed', undefined],
    },
  },
};

export default meta;
type Story = StoryObj<typeof LiveSLATimer>;

const now = new Date();
const hoursAgo = (h: number) => new Date(now.getTime() - h * 3600000).toISOString();
const hoursLater = (h: number) => new Date(now.getTime() + h * 3600000).toISOString();

// ── Running ──────────────────────────────────────────────────────────────

export const Running: Story = {
  args: {
    createdAt: hoursAgo(4),
    deadline: hoursLater(20),
    status: 'on_track',
  },
};

// ── Warning ──────────────────────────────────────────────────────────────

export const Warning: Story = {
  args: {
    createdAt: hoursAgo(20),
    deadline: hoursLater(3),
    status: 'at_risk',
  },
};

// ── Critical ─────────────────────────────────────────────────────────────

export const Critical: Story = {
  args: {
    createdAt: hoursAgo(23),
    deadline: hoursLater(1),
    status: 'breached',
  },
};

// ── Completed ────────────────────────────────────────────────────────────

export const SLACompleted: Story = {
  args: {
    createdAt: hoursAgo(72),
    deadline: hoursAgo(48),
    status: 'completed',
  },
};

// ── Auto Status Running ──────────────────────────────────────────────────

export const AutoRunning: Story = {
  args: {
    createdAt: hoursAgo(2),
    deadline: hoursLater(22),
  },
};

// ── Auto Status Risk ─────────────────────────────────────────────────────

export const AutoRisk: Story = {
  args: {
    createdAt: hoursAgo(22),
    deadline: hoursLater(2),
  },
};

// ── Overdue ──────────────────────────────────────────────────────────────

export const Overdue: Story = {
  args: {
    createdAt: hoursAgo(48),
    deadline: hoursAgo(6),
  },
};

// ── Playground ───────────────────────────────────────────────────────────

export const Playground: Story = {
  args: {
    createdAt: hoursAgo(10),
    deadline: hoursLater(14),
    status: 'on_track',
  },
};
