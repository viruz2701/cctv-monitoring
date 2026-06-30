import type { Meta, StoryObj } from '@storybook/react';
import { MemoryRouter } from 'react-router-dom';
import { QueryClient, QueryClientProvider } from '@tanstack/react-query';
import { WebhookBuilder } from './WebhookBuilder';

const queryClient = new QueryClient({
  defaultOptions: { queries: { retry: false } },
});

const meta: Meta<typeof WebhookBuilder> = {
  title: 'Webhooks/WebhookBuilder',
  component: WebhookBuilder,
  tags: ['autodocs'],
  decorators: [
    (Story) => (
      <MemoryRouter>
        <QueryClientProvider client={queryClient}>
          <div className="max-w-2xl">
            <Story />
          </div>
        </QueryClientProvider>
      </MemoryRouter>
    ),
  ],
};

export default meta;
type Story = StoryObj<typeof WebhookBuilder>;

export const CreateMode: Story = {
  args: {
    webhookId: undefined,
  },
};

export const EditMode: Story = {
  args: {
    webhookId: 'wh-001',
  },
};
