import type { Meta, StoryObj } from '@storybook/react';
import { MemoryRouter } from 'react-router-dom';
import { QueryClient, QueryClientProvider } from '@tanstack/react-query';
import { PlaybookMarketplace } from './PlaybookMarketplace';

const queryClient = new QueryClient({
  defaultOptions: { queries: { retry: false } },
});

const meta: Meta<typeof PlaybookMarketplace> = {
  title: 'Pages/PlaybookMarketplace',
  component: PlaybookMarketplace,
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
type Story = StoryObj<typeof PlaybookMarketplace>;

export const Default: Story = {};
