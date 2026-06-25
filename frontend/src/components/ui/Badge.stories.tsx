import type { Meta, StoryObj } from '@storybook/react';
import { Badge, StatusBadge, HealthBadge, PriorityBadge, TicketStatusBadge, RoleBadge } from './Badge';

const meta: Meta<typeof Badge> = {
  title: 'UI/Badge',
  component: Badge,
  tags: ['autodocs'],
  argTypes: {
    variant: {
      control: 'select',
      options: ['success', 'warning', 'danger', 'info', 'neutral', 'primary'],
    },
    size: {
      control: 'select',
      options: ['sm', 'md', 'lg'],
    },
    dot: { control: 'boolean' },
  },
};

export default meta;
type Story = StoryObj<typeof Badge>;

// ── Badge Variants ───────────────────────────────────────────────────────

export const Success: Story = {
  args: { variant: 'success', children: 'Success', dot: true },
};

export const Warning: Story = {
  args: { variant: 'warning', children: 'Warning', dot: true },
};

export const Danger: Story = {
  args: { variant: 'danger', children: 'Danger', dot: true },
};

export const Info: Story = {
  args: { variant: 'info', children: 'Info', dot: true },
};

export const Neutral: Story = {
  args: { variant: 'neutral', children: 'Neutral' },
};

export const Primary: Story = {
  args: { variant: 'primary', children: 'Primary' },
};

// ── Sizes ────────────────────────────────────────────────────────────────

export const Small: Story = {
  args: { size: 'sm', children: 'Small Badge' },
};

export const Medium: Story = {
  args: { size: 'md', children: 'Medium Badge' },
};

export const Large: Story = {
  args: { size: 'lg', children: 'Large Badge' },
};

// ── Dot Indicator ────────────────────────────────────────────────────────

export const WithDot: Story = {
  args: { dot: true, variant: 'success', children: 'Online' },
};

export const WithoutDot: Story = {
  args: { dot: false, variant: 'neutral', children: 'Draft' },
};

// ── Domain-Specific Badges ───────────────────────────────────────────────

export const StatusOnline: StoryObj = {
  render: () => <StatusBadge status="online" />,
};

export const StatusOffline: StoryObj = {
  render: () => <StatusBadge status="offline" />,
};

export const StatusWarning: StoryObj = {
  render: () => <StatusBadge status="warning" />,
};

export const HealthHealthy: StoryObj = {
  render: () => <HealthBadge health="healthy" />,
};

export const HealthDegraded: StoryObj = {
  render: () => <HealthBadge health="degraded" />,
};

export const HealthFaulty: StoryObj = {
  render: () => <HealthBadge health="faulty" />,
};

export const PriorityCritical: StoryObj = {
  render: () => <PriorityBadge priority="critical" />,
};

export const PriorityHigh: StoryObj = {
  render: () => <PriorityBadge priority="high" />,
};

export const PriorityMedium: StoryObj = {
  render: () => <PriorityBadge priority="medium" />,
};

export const PriorityLow: StoryObj = {
  render: () => <PriorityBadge priority="low" />,
};

export const TicketOpen: StoryObj = {
  render: () => <TicketStatusBadge status="open" />,
};

export const TicketInProgress: StoryObj = {
  render: () => <TicketStatusBadge status="in_progress" />,
};

export const TicketResolved: StoryObj = {
  render: () => <TicketStatusBadge status="resolved" />,
};

export const TicketClosed: StoryObj = {
  render: () => <TicketStatusBadge status="closed" />,
};

export const RoleAdmin: StoryObj = {
  render: () => <RoleBadge role="admin" />,
};

export const RoleTechnician: StoryObj = {
  render: () => <RoleBadge role="technician" />,
};

export const RoleViewer: StoryObj = {
  render: () => <RoleBadge role="viewer" />,
};

// ── Badge Playground ─────────────────────────────────────────────────────

export const Playground: Story = {
  args: {
    variant: 'primary',
    size: 'md',
    dot: false,
    children: 'Custom Badge',
  },
};
