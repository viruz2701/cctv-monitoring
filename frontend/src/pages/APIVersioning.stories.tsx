import type { Meta, StoryObj } from '@storybook/react';
import { MemoryRouter } from 'react-router-dom';
import { QueryClient, QueryClientProvider } from '@tanstack/react-query';
import { APIVersioning } from './APIVersioning';

const queryClient = new QueryClient({
  defaultOptions: { queries: { retry: false } },
});

const meta: Meta<typeof APIVersioning> = {
  title: 'Pages/APIVersioning',
  component: APIVersioning,
  tags: ['autodocs'],
  decorators: [
    (Story) => (
      <MemoryRouter>
        <QueryClientProvider client={queryClient}>
          <div className="min-h-screen bg-slate-50 p-6">
            <Story />
          </div>
        </QueryClientProvider>
      </MemoryRouter>
    ),
  ],
};

export default meta;
type Story = StoryObj<typeof APIVersioning>;

export const Default: Story = {};
