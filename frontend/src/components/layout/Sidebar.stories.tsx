import type { Meta, StoryObj } from '@storybook/react';
import { Sidebar } from './Sidebar';
import { BrowserRouter } from 'react-router-dom';

const meta: Meta<typeof Sidebar> = {
  title: 'Layout/Sidebar',
  component: Sidebar,
  tags: ['autodocs'],
  decorators: [
    (Story) => (
      <BrowserRouter>
        <div className="min-h-[600px]">
          <Story />
        </div>
      </BrowserRouter>
    ),
  ],
  argTypes: {
    collapsed: { control: 'boolean' },
  },
};

export default meta;
type Story = StoryObj<typeof Sidebar>;

// ── Expanded ─────────────────────────────────────────────────────────────

export const Expanded: Story = {
  args: {
    collapsed: false,
    onToggle: () => alert('Toggle'),
    onMobileClose: () => alert('Close mobile'),
  },
};

// ── Collapsed ────────────────────────────────────────────────────────────

export const Collapsed: Story = {
  args: {
    collapsed: true,
    onToggle: () => alert('Toggle'),
    onMobileClose: () => alert('Close mobile'),
  },
};

// ── Mobile Open ──────────────────────────────────────────────────────────

export const MobileOpen: Story = {
  args: {
    collapsed: false,
    mobileOpen: true,
    onToggle: () => alert('Toggle'),
    onMobileClose: () => alert('Close mobile'),
  },
};

// ── Playground ───────────────────────────────────────────────────────────

export const Playground: Story = {
  args: {
    collapsed: false,
    mobileOpen: false,
    onToggle: () => {},
    onMobileClose: () => {},
  },
};
