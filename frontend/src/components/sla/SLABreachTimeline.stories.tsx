import type { Meta, StoryObj } from '@storybook/react';
import { SLABreachTimeline } from './SLABreachTimeline';

const meta: Meta<typeof SLABreachTimeline> = {
  title: 'SLA/SLABreachTimeline',
  component: SLABreachTimeline,
  tags: ['autodocs'],
};

export default meta;
type Story = StoryObj<typeof SLABreachTimeline>;

const sampleBreaches = [
  {
    id: 'b-1', siteName: 'Офис Москва', priority: 'P1',
    severity: 'critical' as const, breachedAt: '2025-06-28T14:30:00Z',
    responseTimeMinutes: 5, resolutionTimeMinutes: 120,
    description: 'NVR not recording — disk full',
  },
  {
    id: 'b-2', siteName: 'Склад Томилино', priority: 'P2',
    severity: 'high' as const, breachedAt: '2025-06-28T10:00:00Z',
    responseTimeMinutes: 15, resolutionTimeMinutes: 45,
    description: 'Camera 12 offline — power issue',
  },
  {
    id: 'b-3', siteName: 'Офис СПб', priority: 'P3',
    severity: 'medium' as const, breachedAt: '2025-06-27T08:00:00Z',
    responseTimeMinutes: 30, resolutionTimeMinutes: 180,
    description: 'Scheduled maintenance overdue',
  },
  {
    id: 'b-4', siteName: 'Офис Казань', priority: 'P4',
    severity: 'low' as const, breachedAt: '2025-06-26T16:00:00Z',
    responseTimeMinutes: 60, resolutionTimeMinutes: 300,
    description: 'Firmware update pending',
  },
];

export const WithBreaches: Story = {
  args: {
    breaches: sampleBreaches,
    loading: false,
  },
};

export const Loading: Story = {
  args: {
    breaches: [],
    loading: true,
  },
};

export const Empty: Story = {
  args: {
    breaches: [],
    loading: false,
  },
};
