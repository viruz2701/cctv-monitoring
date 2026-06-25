import type { Meta, StoryObj } from '@storybook/react';
import { ProgressBar } from './ProgressBar';

const meta: Meta<typeof ProgressBar> = {
  title: 'UI/ProgressBar',
  component: ProgressBar,
  tags: ['autodocs'],
  argTypes: {
    value: {
      control: { type: 'range', min: 0, max: 100, step: 1 },
    },
    max: { control: { type: 'number', min: 1 } },
    variant: {
      control: 'select',
      options: ['success', 'warning', 'danger', 'info'],
    },
    size: {
      control: 'select',
      options: ['sm', 'md', 'lg'],
    },
    showLabel: { control: 'boolean' },
    animated: { control: 'boolean' },
  },
};

export default meta;
type Story = StoryObj<typeof ProgressBar>;

// ── Variants ─────────────────────────────────────────────────────────────

export const Info: Story = {
  args: { value: 60, variant: 'info', showLabel: true },
};

export const Success: Story = {
  args: { value: 85, variant: 'success', showLabel: true },
};

export const Warning: Story = {
  args: { value: 45, variant: 'warning', showLabel: true },
};

export const Danger: Story = {
  args: { value: 25, variant: 'danger', showLabel: true },
};

// ── Sizes ────────────────────────────────────────────────────────────────

export const Small: Story = {
  args: { value: 70, size: 'sm', variant: 'info' },
};

export const Medium: Story = {
  args: { value: 70, size: 'md', variant: 'info' },
};

export const Large: Story = {
  args: { value: 70, size: 'lg', variant: 'info' },
};

// ── Values ───────────────────────────────────────────────────────────────

export const Empty: Story = {
  args: { value: 0, variant: 'info', showLabel: true },
};

export const Halfway: Story = {
  args: { value: 50, variant: 'warning', showLabel: true },
};

export const AlmostComplete: Story = {
  args: { value: 95, variant: 'success', showLabel: true },
};

export const Complete: Story = {
  args: { value: 100, variant: 'success', showLabel: true },
};

// ── States ───────────────────────────────────────────────────────────────

export const WithLabel: Story = {
  args: { value: 73, variant: 'info', showLabel: true },
};

export const WithoutLabel: Story = {
  args: { value: 73, variant: 'info', showLabel: false },
};

export const Animated: Story = {
  args: {
    value: 60,
    variant: 'info',
    animated: true,
    showLabel: true,
  },
};

// ── Use Cases ────────────────────────────────────────────────────────────

export const SLABreachTime: Story = {
  args: {
    value: 82,
    variant: 'warning',
    size: 'sm',
    showLabel: true,
  },
  parameters: {
    docs: {
      description: {
        story: 'SLA compliance progress — warning at 80%+ threshold',
      },
    },
  },
};

export const StorageUsage: Story = {
  args: {
    value: 92,
    variant: 'danger',
    size: 'lg',
    showLabel: true,
    animated: true,
  },
  parameters: {
    docs: {
      description: {
        story: 'Storage capacity near limit — danger variant with animation',
      },
    },
  },
};

// ── Playground ───────────────────────────────────────────────────────────

export const Playground: Story = {
  args: {
    value: 60,
    max: 100,
    variant: 'info',
    size: 'md',
    showLabel: true,
    animated: false,
  },
};
