import type { Meta, StoryObj } from '@storybook/react';
import { PageSuspense } from './PageSuspense';
import { BrowserRouter } from 'react-router-dom';

const meta: Meta<typeof PageSuspense> = {
  title: 'Layout/PageSuspense',
  component: PageSuspense,
  tags: ['autodocs'],
  decorators: [
    (Story) => (
      <BrowserRouter>
        <div className="p-4">
          <Story />
        </div>
      </BrowserRouter>
    ),
  ],
};

export default meta;
type Story = StoryObj<typeof PageSuspense>;

// ── With Simple Content ──────────────────────────────────────────────────

export const WithContent: Story = {
  render: () => (
    <PageSuspense>
      <div className="p-6 bg-white dark:bg-slate-800 rounded-lg border border-slate-200 dark:border-slate-700">
        <h2 className="text-lg font-semibold text-slate-900 dark:text-white">Dashboard Content</h2>
        <p className="mt-2 text-sm text-slate-600 dark:text-slate-400">
          This content is wrapped in PageSuspense with ErrorBoundary and Suspense.
        </p>
      </div>
    </PageSuspense>
  ),
};

// ── With Long Content ────────────────────────────────────────────────────

export const WithLongContent: Story = {
  render: () => (
    <PageSuspense>
      <div className="space-y-4">
        {Array.from({ length: 5 }, (_, i) => (
          <div key={i} className="p-4 bg-white dark:bg-slate-800 rounded-lg border border-slate-200 dark:border-slate-700">
            <h3 className="font-medium text-slate-900 dark:text-white">Section {i + 1}</h3>
            <p className="text-sm text-slate-500 mt-1">Content loaded via React.lazy() with Suspense fallback.</p>
          </div>
        ))}
      </div>
    </PageSuspense>
  ),
};

// ── Loading State (SkeletonPage) ─────────────────────────────────────────

export const LoadingState: Story = {
  render: () => (
    <PageSuspense>
      <div />
    </PageSuspense>
  ),
  parameters: {
    docs: { description: { story: 'Shows SkeletonPage as fallback while lazy component loads.' } },
  },
};

// ── Playground ───────────────────────────────────────────────────────────

export const Playground: Story = {
  render: () => (
    <PageSuspense>
      <div className="p-8 text-center text-sm text-slate-500">
        Page content wrapped in Suspense + ErrorBoundary
      </div>
    </PageSuspense>
  ),
};
