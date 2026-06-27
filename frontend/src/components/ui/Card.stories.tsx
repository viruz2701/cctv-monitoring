import type { Meta, StoryObj } from '@storybook/react';
import { Card, CardHeader, CardBody, CardFooter } from './Card';
import { Settings, Edit3 } from 'lucide-react';

const meta: Meta<typeof Card> = {
  title: 'UI/Card',
  component: Card,
  tags: ['autodocs'],
  argTypes: {
    variant: {
      control: 'select',
      options: ['default', 'elevated', 'bordered'],
    },
    padding: {
      control: 'select',
      options: ['none', 'sm', 'md', 'lg'],
    },
  },
};

export default meta;
type Story = StoryObj<typeof Card>;

// ── Variants ─────────────────────────────────────────────────────────────

export const Default: Story = {
  args: {
    variant: 'default',
    padding: 'md',
    children: 'This is a default card with medium padding.',
  },
};

export const Elevated: Story = {
  args: {
    variant: 'elevated',
    padding: 'md',
    children: 'This elevated card has a shadow and stands out from the page.',
  },
};

export const Bordered: Story = {
  args: {
    variant: 'bordered',
    padding: 'md',
    children: 'This card has a more prominent border.',
  },
};

// ── Padding Sizes ────────────────────────────────────────────────────────

export const NoPadding: Story = {
  args: {
    variant: 'default',
    padding: 'none',
    children: (
      <div className="p-4">
        <p className="text-slate-700 dark:text-slate-300 text-sm">Card with no padding — useful for custom layouts.</p>
      </div>
    ),
  },
};

export const SmallPadding: Story = {
  args: {
    variant: 'default',
    padding: 'sm',
    children: 'Compact card with small padding.',
  },
};

export const LargePadding: Story = {
  args: {
    variant: 'default',
    padding: 'lg',
    children: 'Spacious card with large padding.',
  },
};

// ── Composition: CardHeader / CardBody / CardFooter ──────────────────────

export const WithHeaderBodyFooter: Story = {
  render: () => (
    <Card variant="default">
      <CardHeader action={<Settings className="w-5 h-5 text-slate-400" />}>
        Device Settings
      </CardHeader>
      <CardBody>
        <div className="space-y-3">
          <div className="flex justify-between text-sm">
            <span className="text-slate-500">Model</span>
            <span className="text-slate-900 dark:text-white font-medium">NVR-1040P</span>
          </div>
          <div className="flex justify-between text-sm">
            <span className="text-slate-500">Firmware</span>
            <span className="text-slate-900 dark:text-white font-medium">v3.2.1</span>
          </div>
          <div className="flex justify-between text-sm">
            <span className="text-slate-500">IP Address</span>
            <span className="text-slate-900 dark:text-white font-medium">192.168.1.100</span>
          </div>
        </div>
      </CardBody>
      <CardFooter>
        <button className="flex items-center gap-2 text-sm text-blue-600 hover:text-blue-700 font-medium">
          <Edit3 className="w-4 h-4" />
          Edit Device
        </button>
      </CardFooter>
    </Card>
  ),
};

// ── Elevated with Composition ────────────────────────────────────────────

export const ElevatedComposition: Story = {
  render: () => (
    <Card variant="elevated">
      <CardHeader>System Health</CardHeader>
      <CardBody>
        <p className="text-sm text-slate-600 dark:text-slate-400">
          All systems operational. 24 devices online, 2 offline.
        </p>
      </CardBody>
      <CardFooter>
        <span className="text-xs text-emerald-600 dark:text-emerald-400">● Healthy</span>
      </CardFooter>
    </Card>
  ),
};

// ── Playground ───────────────────────────────────────────────────────────

export const Playground: Story = {
  args: {
    variant: 'default',
    padding: 'md',
    children: 'Customize this card with the controls panel.',
  },
};
