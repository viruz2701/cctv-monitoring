import type { Meta, StoryObj } from '@storybook/react';
import { Header } from './Header';
import { BrowserRouter } from 'react-router-dom';

const meta: Meta<typeof Header> = {
  title: 'Layout/Header',
  component: Header,
  tags: ['autodocs'],
  decorators: [
    (Story) => (
      <BrowserRouter>
        <div className="h-16">
          <Story />
        </div>
      </BrowserRouter>
    ),
  ],
  argTypes: {
    sidebarCollapsed: { control: 'boolean' },
  },
};

export default meta;
type Story = StoryObj<typeof Header>;

// ── Expanded Sidebar ─────────────────────────────────────────────────────

export const Expanded: Story = {
  args: {
    sidebarCollapsed: false,
    onMobileMenuToggle: () => alert('Mobile menu toggle'),
  },
};

// ── Collapsed Sidebar ────────────────────────────────────────────────────

export const CollapsedSidebar: Story = {
  args: {
    sidebarCollapsed: true,
    onMobileMenuToggle: () => alert('Mobile menu toggle'),
  },
};

// ── Playground ───────────────────────────────────────────────────────────

export const Playground: Story = {
  args: {
    sidebarCollapsed: false,
  },
};
