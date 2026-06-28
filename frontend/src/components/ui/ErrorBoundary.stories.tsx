import type { Meta, StoryObj } from '@storybook/react';
import { BrowserRouter } from 'react-router-dom';
import { ErrorBoundary } from './ErrorBoundary';
import { Button } from './Button';

const meta: Meta<typeof ErrorBoundary> = {
  title: 'UI/ErrorBoundary',
  component: ErrorBoundary,
  decorators: [
    (Story) => (
      <BrowserRouter>
        <div className="p-8 max-w-2xl mx-auto">
          <Story />
        </div>
      </BrowserRouter>
    ),
  ],
  parameters: {
    layout: 'centered',
    docs: {
      description: {
        component:
          'ErrorBoundary — унифицированный Error Boundary для асинхронных операций. ' +
          'Соответствует P1-UX.11: Error Handling UI.',
      },
    },
  },
  tags: ['autodocs'],
};

export default meta;
type Story = StoryObj<typeof ErrorBoundary>;

// ── Throws on render ─────────────────────────────────────────────

function BuggyComponent({ shouldThrow = false }: { shouldThrow?: boolean }) {
  if (shouldThrow) {
    throw new Error('Test error: something went wrong!');
  }
  return <div className="text-green-600 font-medium">✅ Component rendered successfully</div>;
}

export const Default: Story = {
  render: () => (
    <ErrorBoundary>
      <BuggyComponent shouldThrow={false} />
    </ErrorBoundary>
  ),
};

export const WithError: Story = {
  render: () => (
    <ErrorBoundary componentName="TestComponent">
      <BuggyComponent shouldThrow={true} />
    </ErrorBoundary>
  ),
};

export const WithCustomFallback: Story = {
  render: () => (
    <ErrorBoundary
      componentName="CustomFallback"
      fallback={(error, retry) => (
        <div className="bg-red-50 border border-red-200 rounded-lg p-6 text-center">
          <p className="text-red-700 font-semibold mb-2">⚠️ Custom Error UI</p>
          <p className="text-red-600 text-sm mb-4">{error?.message}</p>
          <Button onClick={retry} variant="primary" size="sm">
            Retry
          </Button>
        </div>
      )}
    >
      <BuggyComponent shouldThrow={true} />
    </ErrorBoundary>
  ),
};

export const WithHomeButton: Story = {
  render: () => (
    <ErrorBoundary componentName="DetailPage" showHome={true}>
      <BuggyComponent shouldThrow={true} />
    </ErrorBoundary>
  ),
};

export const NoError: Story = {
  render: () => (
    <ErrorBoundary componentName="SafeComponent">
      <div className="bg-gray-50 border border-gray-200 rounded-lg p-6">
        <h3 className="text-lg font-semibold mb-2">Content without errors</h3>
        <p className="text-gray-600">
          ErrorBoundary wraps children normally when no error occurs.
        </p>
      </div>
    </ErrorBoundary>
  ),
};
