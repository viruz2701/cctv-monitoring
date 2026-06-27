import type { Meta, StoryObj } from '@storybook/react';
import { VisuallyHidden } from './VisuallyHidden';

const meta: Meta<typeof VisuallyHidden> = {
  title: 'UI/VisuallyHidden',
  component: VisuallyHidden,
  tags: ['autodocs'],
  argTypes: {
    as: {
      control: 'select',
      options: ['span', 'div', 'p'],
    },
  },
};

export default meta;
type Story = StoryObj<typeof VisuallyHidden>;

// ── Screen-Reader Only Text ──────────────────────────────────────────────

export const Default: Story = {
  render: () => (
    <div className="p-4">
      <p className="text-sm text-slate-600">
        This button has screen-reader only text:
      </p>
      <button
        className="mt-2 px-4 py-2 bg-blue-600 text-white rounded-lg text-sm"
        aria-label="Close dialog"
      >
        ✕
        <VisuallyHidden>Close dialog</VisuallyHidden>
      </button>
    </div>
  ),
};

// ── As Span ──────────────────────────────────────────────────────────────

export const AsSpan: Story = {
  args: {
    as: 'span',
    children: 'This text is only visible to screen readers (as span)',
  },
};

// ── As Div ───────────────────────────────────────────────────────────────

export const AsDiv: Story = {
  args: {
    as: 'div',
    children: 'This text is only visible to screen readers (as div)',
  },
};

// ── As Paragraph ─────────────────────────────────────────────────────────

export const AsParagraph: Story = {
  args: {
    as: 'p',
    children: 'This text is only visible to screen readers (as p)',
  },
};

// ── Practical Example: Loading Indicator ─────────────────────────────────

export const LoadingExample: Story = {
  render: () => (
    <div className="p-4">
      <div className="flex items-center gap-3">
        <div className="w-5 h-5 border-2 border-blue-500 border-t-transparent rounded-full animate-spin" />
        <span className="text-sm text-slate-600">Loading devices...</span>
        <VisuallyHidden>Please wait while device list is loading</VisuallyHidden>
      </div>
    </div>
  ),
};

// ── Practical Example: Form Label ────────────────────────────────────────

export const FormLabelExample: Story = {
  render: () => (
    <div className="p-4 space-y-2">
      <VisuallyHidden as="span">Search devices</VisuallyHidden>
      <div className="relative">
        <input
          type="search"
          placeholder="Search..."
          className="w-full px-4 py-2 border border-slate-300 rounded-lg text-sm"
          aria-label="Search devices"
        />
      </div>
      <p className="text-xs text-slate-400">
        The label "Search devices" is visually hidden but available to screen readers.
      </p>
    </div>
  ),
};

// ── Playground ───────────────────────────────────────────────────────────

export const Playground: Story = {
  args: {
    as: 'span',
    children: 'Visually hidden playground content',
  },
};
