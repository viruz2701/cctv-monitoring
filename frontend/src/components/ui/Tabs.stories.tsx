import type { Meta, StoryObj } from '@storybook/react';
import { Tabs } from './Tabs';
import { useState } from 'react';
import { Camera, Settings, Activity, AlertTriangle } from 'lucide-react';

const meta: Meta<typeof Tabs> = {
  title: 'UI/Tabs',
  component: Tabs,
  tags: ['autodocs'],
  argTypes: {
    variant: {
      control: 'select',
      options: ['default', 'pills', 'underline'],
    },
  },
};

export default meta;
type Story = StoryObj<typeof Tabs>;

// ── Default Variant ──────────────────────────────────────────────────────

const DefaultTemplate = () => {
  const [active, setActive] = useState('overview');
  return (
    <Tabs
      tabs={[
        { id: 'overview', label: 'Overview' },
        { id: 'devices', label: 'Devices' },
        { id: 'settings', label: 'Settings' },
      ]}
      activeTab={active}
      onChange={setActive}
      variant="default"
    >
      <div className="p-6 text-sm text-slate-600 dark:text-slate-400 bg-white dark:bg-slate-800 rounded-lg border border-slate-200 dark:border-slate-700">
        {active === 'overview' && 'Overview content — system summary and key metrics.'}
        {active === 'devices' && 'Devices content — list of all connected devices.'}
        {active === 'settings' && 'Settings content — configuration options.'}
      </div>
    </Tabs>
  );
};

export const Default: Story = {
  render: () => <DefaultTemplate />,
};

// ── Pills Variant ────────────────────────────────────────────────────────

const PillsTemplate = () => {
  const [active, setActive] = useState('all');
  return (
    <Tabs
      tabs={[
        { id: 'all', label: 'All' },
        { id: 'online', label: 'Online' },
        { id: 'offline', label: 'Offline' },
        { id: 'warning', label: 'Warning' },
      ]}
      activeTab={active}
      onChange={setActive}
      variant="pills"
    >
      <div className="p-4 text-sm text-slate-500">Showing {active} devices.</div>
    </Tabs>
  );
};

export const Pills: Story = {
  render: () => <PillsTemplate />,
};

// ── Underline Variant ────────────────────────────────────────────────────

const UnderlineTemplate = () => {
  const [active, setActive] = useState('general');
  return (
    <Tabs
      tabs={[
        { id: 'general', label: 'General' },
        { id: 'network', label: 'Network' },
        { id: 'storage', label: 'Storage' },
      ]}
      activeTab={active}
      onChange={setActive}
      variant="underline"
    >
      <div className="p-4 text-sm text-slate-500">{active} settings panel.</div>
    </Tabs>
  );
};

export const Underline: Story = {
  render: () => <UnderlineTemplate />,
};

// ── With Icons ───────────────────────────────────────────────────────────

const WithIconsTemplate = () => {
  const [active, setActive] = useState('cameras');
  return (
    <Tabs
      tabs={[
        { id: 'cameras', label: 'Cameras', icon: <Camera className="w-4 h-4" /> },
        { id: 'activity', label: 'Activity', icon: <Activity className="w-4 h-4" /> },
        { id: 'alerts', label: 'Alerts', icon: <AlertTriangle className="w-4 h-4" /> },
      ]}
      activeTab={active}
      onChange={setActive}
      variant="default"
    >
      <div className="p-4 text-sm text-slate-500">{active} content with icons.</div>
    </Tabs>
  );
};

export const WithIcons: Story = {
  render: () => <WithIconsTemplate />,
};

// ── With Badges ──────────────────────────────────────────────────────────

const WithBadgesTemplate = () => {
  const [active, setActive] = useState('all');
  return (
    <Tabs
      tabs={[
        { id: 'all', label: 'All', badge: 24 },
        { id: 'open', label: 'Open', badge: 5 },
        { id: 'critical', label: 'Critical', badge: 2 },
        { id: 'resolved', label: 'Resolved', badge: 17 },
      ]}
      activeTab={active}
      onChange={setActive}
      variant="default"
    >
      <div className="p-4 text-sm text-slate-500">Showing {active} tickets.</div>
    </Tabs>
  );
};

export const WithBadges: Story = {
  render: () => <WithBadgesTemplate />,
};

// ── Disabled Tab ─────────────────────────────────────────────────────────

const DisabledTabTemplate = () => {
  const [active, setActive] = useState('basic');
  return (
    <Tabs
      tabs={[
        { id: 'basic', label: 'Basic' },
        { id: 'advanced', label: 'Advanced' },
        { id: 'expert', label: 'Expert', disabled: true },
      ]}
      activeTab={active}
      onChange={setActive}
      variant="pills"
    >
      <div className="p-4 text-sm text-slate-500">{active} mode selected.</div>
    </Tabs>
  );
};

export const DisabledTab: Story = {
  render: () => <DisabledTabTemplate />,
};

// ── Controlled ───────────────────────────────────────────────────────────

const ControlledTemplate = () => {
  const [active, setActive] = useState('tab1');
  return (
    <div className="space-y-4">
      <div className="flex gap-2">
        <button onClick={() => setActive('tab1')} className="px-3 py-1 text-xs bg-blue-100 text-blue-700 rounded">Tab 1</button>
        <button onClick={() => setActive('tab2')} className="px-3 py-1 text-xs bg-blue-100 text-blue-700 rounded">Tab 2</button>
        <button onClick={() => setActive('tab3')} className="px-3 py-1 text-xs bg-blue-100 text-blue-700 rounded">Tab 3</button>
      </div>
      <Tabs
        tabs={[
          { id: 'tab1', label: 'Tab 1' },
          { id: 'tab2', label: 'Tab 2' },
          { id: 'tab3', label: 'Tab 3' },
        ]}
        activeTab={active}
        onChange={setActive}
        variant="underline"
      >
        <div className="p-4 text-sm text-slate-500">Content for {active}</div>
      </Tabs>
    </div>
  );
};

export const Controlled: Story = {
  render: () => <ControlledTemplate />,
};

// ── Playground ───────────────────────────────────────────────────────────

const PlaygroundTemplate = (args: any) => {
  const [active, setActive] = useState('first');
  return (
    <Tabs
      {...args}
      activeTab={active}
      onChange={setActive}
    >
      <div className="p-4 text-sm text-slate-500">Content area.</div>
    </Tabs>
  );
};

export const Playground: Story = {
  render: (args) => <PlaygroundTemplate {...args} />,
  args: {
    tabs: [
      { id: 'first', label: 'First' },
      { id: 'second', label: 'Second' },
      { id: 'third', label: 'Third' },
    ],
    variant: 'default',
  },
};
