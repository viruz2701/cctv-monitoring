import type { Meta, StoryObj } from '@storybook/react';
import { Breadcrumbs } from './Breadcrumbs';
import { BrowserRouter } from 'react-router-dom';
import { Home, Settings, Shield, Camera } from './Icons';

const meta: Meta<typeof Breadcrumbs> = {
  title: 'UI/Breadcrumbs',
  component: Breadcrumbs,
  tags: ['autodocs'],
  decorators: [
    (Story) => (
      <BrowserRouter>
        <Story />
      </BrowserRouter>
    ),
  ],
  argTypes: {
    maxItems: { control: { type: 'number', min: 2, max: 10 } },
  },
};

export default meta;
type Story = StoryObj<typeof Breadcrumbs>;

// ── Simple Navigation ────────────────────────────────────────────────────

export const Simple: Story = {
  args: {
    items: [
      { label: 'Dashboard', href: '/dashboard' },
      { label: 'Devices', href: '/devices' },
      { label: 'Camera NVR-01' },
    ],
  },
};

// ── With Icons ───────────────────────────────────────────────────────────

export const WithIcons: Story = {
  args: {
    items: [
      { label: 'Home', href: '/dashboard', icon: Home },
      { label: 'Admin', href: '/admin', icon: Settings },
      { label: 'Security', href: '/security', icon: Shield },
      { label: 'Permissions' },
    ],
  },
};

// ── Long (Truncated) ─────────────────────────────────────────────────────

export const Truncated: Story = {
  args: {
    items: [
      { label: 'Home', href: '/', icon: Home },
      { label: 'Organizations', href: '/orgs' },
      { label: 'Main Office', href: '/orgs/main' },
      { label: 'Sites', href: '/orgs/main/sites' },
      { label: 'Building A', href: '/orgs/main/sites/building-a' },
      { label: 'Floor 3', href: '/orgs/main/sites/building-a/floor3' },
      { label: 'Cameras', href: '/orgs/main/sites/building-a/floor3/cameras' },
      { label: 'NVR-01 Settings' },
    ],
    maxItems: 5,
  },
};

// ── Two Items ────────────────────────────────────────────────────────────

export const TwoItems: Story = {
  args: {
    items: [
      { label: 'Dashboard', href: '/dashboard' },
      { label: 'Analytics' },
    ],
  },
};

// ── Mobile View ──────────────────────────────────────────────────────────

export const MobileView: Story = {
  args: {
    items: [
      { label: 'Dashboard', href: '/dashboard' },
      { label: 'Devices', href: '/devices' },
      { label: 'Cameras', href: '/devices/cameras' },
      { label: 'NVR-01' },
    ],
  },
  parameters: {
    viewport: { defaultViewport: 'mobile1' },
  },
};

// ── Playground ───────────────────────────────────────────────────────────

export const Playground: Story = {
  args: {
    items: [
      { label: 'Home', href: '/', icon: Home },
      { label: 'Section', href: '/section' },
      { label: 'Current Page' },
    ],
    maxItems: 6,
  },
};
