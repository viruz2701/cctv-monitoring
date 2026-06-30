import type { Meta, StoryObj } from '@storybook/react';
import { WebhookLogFilter, DEFAULT_LOG_FILTER } from './WebhookLogFilter';

const meta: Meta<typeof WebhookLogFilter> = {
  title: 'Webhooks/WebhookLogFilter',
  component: WebhookLogFilter,
  tags: ['autodocs'],
};

export default meta;
type Story = StoryObj<typeof WebhookLogFilter>;

export const Default: Story = {
  args: {
    filter: DEFAULT_LOG_FILTER,
    onFilterChange: (f) => console.log('Filter:', f),
  },
};

export const Filtered: Story = {
  args: {
    filter: { ...DEFAULT_LOG_FILTER, status: 'failed', eventType: 'device.status_changed' },
    onFilterChange: (f) => console.log('Filter:', f),
  },
};
