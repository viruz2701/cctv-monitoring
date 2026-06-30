import type { Meta, StoryObj } from '@storybook/react';
import { SLAGaugePanel } from './SLAGaugePanel';

const meta: Meta<typeof SLAGaugePanel> = {
  title: 'SLA/SLAGaugePanel',
  component: SLAGaugePanel,
  tags: ['autodocs'],
};

export default meta;
type Story = StoryObj<typeof SLAGaugePanel>;

export const Good: Story = {
  args: {
    overallCompliance: 96,
    mttrCompliance: 88,
    preventiveCompliance: 92,
    emergencyResponse: 78,
    loading: false,
  },
};

export const NeedsAttention: Story = {
  args: {
    overallCompliance: 72,
    mttrCompliance: 65,
    preventiveCompliance: 80,
    emergencyResponse: 45,
    loading: false,
  },
};

export const Loading: Story = {
  args: {
    overallCompliance: 0,
    mttrCompliance: 0,
    preventiveCompliance: 0,
    emergencyResponse: 0,
    loading: true,
  },
};
