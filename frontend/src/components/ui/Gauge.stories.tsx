import type { Meta, StoryObj } from '@storybook/react';
import { Gauge } from './Gauge';

const meta: Meta<typeof Gauge> = {
  title: 'UI/Gauge',
  component: Gauge,
  tags: ['autodocs'],
  argTypes: {
    value: { control: { type: 'number', min: 0, max: 100 } },
    size: { control: 'select', options: ['sm', 'md', 'lg'] },
    showValue: { control: 'boolean' },
    label: { control: 'text' },
    unit: { control: 'text' },
  },
};

export default meta;
type Story = StoryObj<typeof Gauge>;

// ── Various Values ───────────────────────────────────────────────────────

export const Empty: Story = {
  args: { value: 0, label: 'Empty' },
};

export const Quarter: Story = {
  args: { value: 25, label: '25%' },
};

export const Half: Story = {
  args: { value: 50, label: '50%' },
};

export const ThreeQuarter: Story = {
  args: { value: 75, label: '75%' },
};

export const Full: Story = {
  args: { value: 100, label: '100%' },
};

// ── Thresholds ───────────────────────────────────────────────────────────

export const GreenThreshold: Story = {
  args: {
    value: 96,
    label: 'SLA Compliance',
    thresholds: [
      { value: 95, color: '#16a34a', label: '≥95%' },
      { value: 80, color: '#eab308', label: '80–94%' },
      { value: 60, color: '#f97316', label: '60–79%' },
      { value: 0, color: '#dc2626', label: '<60%' },
    ],
  },
};

export const YellowThreshold: Story = {
  args: {
    value: 85,
    label: 'SLA Compliance',
    thresholds: [
      { value: 95, color: '#16a34a', label: '≥95%' },
      { value: 80, color: '#eab308', label: '80–94%' },
      { value: 60, color: '#f97316', label: '60–79%' },
      { value: 0, color: '#dc2626', label: '<60%' },
    ],
  },
};

export const OrangeThreshold: Story = {
  args: {
    value: 65,
    label: 'SLA Compliance',
    thresholds: [
      { value: 95, color: '#16a34a', label: '≥95%' },
      { value: 80, color: '#eab308', label: '80–94%' },
      { value: 60, color: '#f97316', label: '60–79%' },
      { value: 0, color: '#dc2626', label: '<60%' },
    ],
  },
};

export const RedThreshold: Story = {
  args: {
    value: 45,
    label: 'SLA Compliance',
    thresholds: [
      { value: 95, color: '#16a34a', label: '≥95%' },
      { value: 80, color: '#eab308', label: '80–94%' },
      { value: 60, color: '#f97316', label: '60–79%' },
      { value: 0, color: '#dc2626', label: '<60%' },
    ],
  },
};

// ── Sizes ────────────────────────────────────────────────────────────────

export const Small: Story = {
  args: { value: 72, size: 'sm', label: 'Small' },
};

export const Medium: Story = {
  args: { value: 72, size: 'md', label: 'Medium' },
};

export const Large: Story = {
  args: { value: 72, size: 'lg', label: 'Large' },
};

// ── With Label ───────────────────────────────────────────────────────────

export const WithLabel: Story = {
  args: {
    value: 88,
    label: 'CPU Usage',
    unit: '%',
  },
};

// ── Without Value Display ────────────────────────────────────────────────

export const WithoutValue: Story = {
  args: {
    value: 60,
    label: 'Hidden Value',
    showValue: false,
  },
};

// ── Custom Unit ──────────────────────────────────────────────────────────

export const CustomUnit: Story = {
  args: {
    value: 34,
    label: 'Temperature',
    unit: '°C',
    max: 60,
  },
};

// ── Playground ───────────────────────────────────────────────────────────

export const Playground: Story = {
  args: {
    value: 72,
    label: 'SLA Score',
    size: 'md',
    showValue: true,
    unit: '%',
  },
};
