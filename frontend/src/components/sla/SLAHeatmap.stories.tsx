import type { Meta, StoryObj } from '@storybook/react';
import { SLAHeatmap } from './SLAHeatmap';

const meta: Meta<typeof SLAHeatmap> = {
  title: 'SLA/SLAHeatmap',
  component: SLAHeatmap,
  tags: ['autodocs'],
};

export default meta;
type Story = StoryObj<typeof SLAHeatmap>;

const weekStarts = [
  '2025-06-02', '2025-06-09', '2025-06-16',
  '2025-06-23', '2025-06-30',
];

const sampleData = [
  {
    siteId: 'site-1', siteName: 'Офис Москва',
    weeks: weekStarts.map((ws) => ({
      weekStart: ws, compliance: Math.floor(Math.random() * 30) + 70, total: 20, within: 18,
    })),
  },
  {
    siteId: 'site-2', siteName: 'Склад Томилино',
    weeks: weekStarts.map((ws) => ({
      weekStart: ws, compliance: Math.floor(Math.random() * 40) + 60, total: 15, within: 12,
    })),
  },
  {
    siteId: 'site-3', siteName: 'Офис СПб',
    weeks: weekStarts.map((ws) => ({
      weekStart: ws, compliance: Math.floor(Math.random() * 20) + 80, total: 10, within: 9,
    })),
  },
  {
    siteId: 'site-4', siteName: 'Офис Казань',
    weeks: weekStarts.map((ws) => ({
      weekStart: ws, compliance: Math.floor(Math.random() * 50) + 50, total: 8, within: 5,
    })),
  },
];

export const WithData: Story = {
  args: {
    data: sampleData,
    loading: false,
  },
};

export const Loading: Story = {
  args: {
    data: [],
    loading: true,
  },
};

export const Empty: Story = {
  args: {
    data: [],
    loading: false,
  },
};
