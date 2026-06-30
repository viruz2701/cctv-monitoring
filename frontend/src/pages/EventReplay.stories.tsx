import type { Meta, StoryObj } from '@storybook/react';
import { MemoryRouter } from 'react-router-dom';
import { QueryClient, QueryClientProvider } from '@tanstack/react-query';
import { EventReplay } from './EventReplay';

const queryClient = new QueryClient({
  defaultOptions: { queries: { retry: false } },
});

const meta: Meta<typeof EventReplay> = {
  title: 'Pages/EventReplay',
  component: EventReplay,
  tags: ['autodocs'],
  decorators: [
    (Story) => (
      <MemoryRouter>
        <QueryClientProvider client={queryClient}>
          <div className="min-h-screen bg-slate-50">
            <Story />
          </div>
        </QueryClientProvider>
      </MemoryRouter>
    ),
  ],
};

export default meta;
type Story = StoryObj<typeof EventReplay>;

export const Default: Story = {};
