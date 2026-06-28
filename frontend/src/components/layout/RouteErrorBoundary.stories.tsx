import type { Meta, StoryObj } from '@storybook/react';
import React from 'react';
import { BrowserRouter } from 'react-router-dom';
import { RouteErrorBoundary } from './RouteErrorBoundary';

function BuggyPage(): React.ReactElement {
  throw new Error('Failed to load page data: network error');
}

const meta: Meta<typeof RouteErrorBoundary> = {
  title: 'Layout/RouteErrorBoundary',
  component: RouteErrorBoundary,
  decorators: [
    (Story) => (
      <BrowserRouter>
        <Story />
      </BrowserRouter>
    ),
  ],
  parameters: {
    layout: 'fullscreen',
    docs: {
      description: {
        component:
          'RouteErrorBoundary — Error boundary для целых страниц. ' +
          'Соответствует P1-UX.11: Error Handling UI.',
      },
    },
  },
  tags: ['autodocs'],
};

export default meta;
type Story = StoryObj<typeof RouteErrorBoundary>;

export const Default: Story = {
  render: () => (
    <RouteErrorBoundary>
      <BuggyPage />
    </RouteErrorBoundary>
  ),
};

export const WithCustomFallback: Story = {
  render: () => (
    <RouteErrorBoundary
      fallback={
        <div className="flex items-center justify-center min-h-screen bg-slate-50">
          <div className="text-center p-8">
            <p className="text-4xl mb-4">💥</p>
            <h2 className="text-xl font-bold text-slate-800 mb-2">Custom Error View</h2>
            <p className="text-slate-500">This is a custom fallback for route errors.</p>
          </div>
        </div>
      }
    >
      <BuggyPage />
    </RouteErrorBoundary>
  ),
};

export const NoError: Story = {
  render: () => (
    <RouteErrorBoundary>
      <div className="p-8 text-center">
        <h2 className="text-xl font-semibold text-green-600">✅ Page loaded successfully</h2>
        <p className="text-slate-500 mt-2">No errors to display.</p>
      </div>
    </RouteErrorBoundary>
  ),
};
