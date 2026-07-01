import type { Meta, StoryObj } from '@storybook/react';
import { Gauge } from './Gauge';

const meta: Meta<typeof Gauge> = {
  title: 'UI/Gauge',
  component: Gauge,
  tags: ['autodocs'],
  argTypes: {
    value: { control: { type: 'number', min: 0, max: 100 } },
    max: { control: { type: 'number', min: 1, max: 1000 } },
    size: { control: 'select', options: ['sm', 'md', 'lg'] },
    showValue: { control: 'boolean' },
    label: { control: 'text' },
    unit: { control: 'text' },
  },
};

export default meta;
type Story = StoryObj<typeof Gauge>;

// ── Health States ─────────────────────────────────────────────────────────

export const Healthy: Story = {
  args: { value: 98, label: 'System Health', size: 'md' },
};

export const Warning: Story = {
  args: { value: 72, label: 'Disk Usage', size: 'md' },
};

export const Critical: Story = {
  args: { value: 35, label: 'Memory', size: 'md' },
};

export const Low: Story = {
  args: { value: 15, label: 'Battery', size: 'md' },
};

// ── Sizes ─────────────────────────────────────────────────────────────────

export const Small: Story = {
  args: { value: 85, label: 'Small', size: 'sm' },
};

export const Medium: Story = {
  args: { value: 85, label: 'Medium', size: 'md' },
};

export const Large: Story = {
  args: { value: 85, label: 'Large', size: 'lg' },
};

// ── Custom Unit ───────────────────────────────────────────────────────────

export const CustomUnit: Story = {
  args: { value: 42, label: 'Temperature', unit: '°C', size: 'md' },
};

// ── Without Label / Without Value ─────────────────────────────────────────

export const WithoutLabel: Story = {
  args: { value: 88, size: 'md' },
};

export const WithoutValue: Story = {
  args: { value: 65, label: 'Progress', showValue: false, size: 'md' },
};

// ── Custom Thresholds ─────────────────────────────────────────────────────

export const CustomThresholds: Story = {
  args: {
    value: 95,
    label: 'Network Quality',
    size: 'md',
    thresholds: [
      { value: 90, color: '#16a34a', label: 'Excellent' },
      { value: 70, color: '#eab308', label: 'Good' },
      { value: 50, color: '#f97316', label: 'Fair' },
      { value: 0, color: '#dc2626', label: 'Poor' },
    ],
  },
};

// ── All Sizes Showcase ────────────────────────────────────────────────────

export const AllSizes: StoryObj = {
  render: () => (
    <div className="flex items-end gap-8 p-4">
      <Gauge value={92} label="SM" size="sm" />
      <Gauge value={92} label="MD" size="md" />
      <Gauge value={92} label="LG" size="lg" />
    </div>
  ),
};

// ── All Health States ─────────────────────────────────────────────────────

export const AllHealthStates: StoryObj = {
  render: () => (
    <div className="flex gap-8 p-4">
      <Gauge value={98} label="Healthy" size="sm" />
      <Gauge value={72} label="Warning" size="sm" />
      <Gauge value={35} label="Critical" size="sm" />
      <Gauge value={15} label="Low" size="sm" />
    </div>
  ),
};

// ── Full Width ────────────────────────────────────────────────────────────

export const FullWidth: StoryObj = {
  render: () => (
    <div className="flex justify-around p-4 bg-white dark:bg-slate-800 rounded-lg">
      <Gauge value={99.97} label="SLA Compliance" size="lg" />
    </div>
  ),
};

// ── Playground ────────────────────────────────────────────────────────────

export const Playground: Story = {
  args: {
    value: 75,
    label: 'Custom Gauge',
    size: 'md',
    showValue: true,
    unit: '%',
  },
};
