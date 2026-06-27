import type { Meta, StoryObj } from '@storybook/react';
import { SLAProgressBar } from './SLAProgressBar';

const meta: Meta<typeof SLAProgressBar> = {
  title: 'Molecules/SLAProgressBar',
  component: SLAProgressBar,
  tags: ['autodocs'],
  argTypes: {
    elapsedMinutes: { control: { type: 'number', min: 0, max: 1440 } },
    totalMinutes: { control: { type: 'number', min: 1, max: 10080 } },
    compact: { control: 'boolean' },
  },
};

export default meta;
type Story = StoryObj<typeof SLAProgressBar>;

// ── Success (Early Stage) ────────────────────────────────────────────────

export const Success: Story = {
  args: {
    elapsedMinutes: 30,
    totalMinutes: 480,
    label: 'Server Rack A-101',
  },
};

// ── Warning (75% Used) ───────────────────────────────────────────────────

export const Warning: Story = {
  args: {
    elapsedMinutes: 360,
    totalMinutes: 480,
    label: 'NVR-02 maintenance',
  },
};

// ── Danger (90%+ Used) ───────────────────────────────────────────────────

export const Danger: Story = {
  args: {
    elapsedMinutes: 450,
    totalMinutes: 480,
    label: 'CAM-101 firmware update',
  },
};

// ── Overdue ──────────────────────────────────────────────────────────────

export const Overdue: Story = {
  args: {
    elapsedMinutes: 600,
    totalMinutes: 480,
    label: 'Critical patch deployment',
  },
};

// ── Compact Mode ─────────────────────────────────────────────────────────

export const Compact: Story = {
  args: {
    elapsedMinutes: 200,
    totalMinutes: 480,
    compact: true,
    label: 'Database migration',
  },
};

// ── Short SLA ────────────────────────────────────────────────────────────

export const ShortSLA: Story = {
  args: {
    elapsedMinutes: 10,
    totalMinutes: 60,
    label: 'Urgent ticket #1024',
  },
};

// ── Long SLA ─────────────────────────────────────────────────────────────

export const LongSLA: Story = {
  args: {
    elapsedMinutes: 1440,
    totalMinutes: 10080,
    label: 'Weekly maintenance window',
  },
};

// ── Without Label ────────────────────────────────────────────────────────

export const WithoutLabel: Story = {
  args: {
    elapsedMinutes: 240,
    totalMinutes: 480,
  },
};

// ── Playground ───────────────────────────────────────────────────────────

export const Playground: Story = {
  args: {
    elapsedMinutes: 240,
    totalMinutes: 480,
    label: 'Custom SLA task',
    compact: false,
  },
};
