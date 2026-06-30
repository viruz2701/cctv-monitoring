import type { Meta, StoryObj } from '@storybook/react';
import { WebhookStatsCards } from './WebhookStatsCards';

const meta: Meta<typeof WebhookStatsCards> = {
  title: 'Webhooks/WebhookStatsCards',
  component: WebhookStatsCards,
  tags: ['autodocs'],
};

export default meta;
type Story = StoryObj<typeof WebhookStatsCards>;

export const WithId: Story = {
  args: {
    webhookId: 'wh-001',
  },
};

export const WithoutId: Story = {
  args: {
    webhookId: undefined,
  },
};
