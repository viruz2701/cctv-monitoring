import type { Meta, StoryObj } from '@storybook/react';
import { SLATrendChart } from './SLATrendChart';

const meta: Meta<typeof SLATrendChart> = {
  title: 'SLA/SLATrendChart',
  component: SLATrendChart,
  tags: ['autodocs'],
};

export default meta;
type Story = StoryObj<typeof SLATrendChart>;

function generateTrendData(days: number) {
  const data = [];
  const now = new Date();
  for (let i = days; i >= 0; i--) {
    const date = new Date(now);
    date.setDate(date.getDate() - i);
    data.push({
      date: date.toISOString(),
      compliance: Math.round(85 + Math.random() * 15),
    });
  }
  return data;
}

export const WithData: Story = {
  args: {
    data: generateTrendData(90),
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
