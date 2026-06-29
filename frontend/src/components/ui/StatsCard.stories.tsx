import type { Meta, StoryObj } from '@storybook/react';
import { StatsCard, MiniStatsCard } from './StatsCard';
import { Camera, HardDrive, Activity, Users, Wifi } from './Icons';

const meta: Meta<typeof StatsCard> = {
  title: 'UI/StatsCard',
  component: StatsCard,
  tags: ['autodocs'],
  argTypes: {
    iconColor: { control: 'text' },
    iconBgColor: { control: 'text' },
  },
};

export default meta;
type Story = StoryObj<typeof StatsCard>;

// ── With Trend Up ────────────────────────────────────────────────────────

export const TrendUp: Story = {
  args: {
    title: 'Devices Online',
    value: 42,
    subtitle: 'Total connected cameras',
    icon: Camera,
    iconColor: 'text-emerald-600',
    iconBgColor: 'bg-emerald-50 dark:bg-emerald-900/20',
    trend: {
      value: 12.5,
      label: 'vs last week',
      direction: 'up',
    },
  },
};

// ── With Trend Down ──────────────────────────────────────────────────────

export const TrendDown: Story = {
  args: {
    title: 'Failed Alerts',
    value: 8,
    subtitle: 'Unresolved critical issues',
    icon: Activity,
    iconColor: 'text-red-600',
    iconBgColor: 'bg-red-50 dark:bg-red-900/20',
    trend: {
      value: 23,
      label: 'increase from last week',
      direction: 'down',
    },
  },
};

// ── Without Trend ────────────────────────────────────────────────────────

export const WithoutTrend: Story = {
  args: {
    title: 'Total Storage',
    value: '14.2 TB',
    subtitle: 'Across all NVRs',
    icon: HardDrive,
    iconColor: 'text-blue-600',
    iconBgColor: 'bg-blue-50 dark:bg-blue-900/20',
  },
};

// ── With Large Value ─────────────────────────────────────────────────────

export const LargeValue: Story = {
  args: {
    title: 'Active Users',
    value: '1,247',
    subtitle: 'Currently monitoring',
    icon: Users,
    iconColor: 'text-purple-600',
    iconBgColor: 'bg-purple-50 dark:bg-purple-900/20',
    trend: {
      value: 5.2,
      label: 'vs yesterday',
      direction: 'up',
    },
  },
};

// ── Network Stats ────────────────────────────────────────────────────────

export const NetworkStats: Story = {
  args: {
    title: 'Network Uptime',
    value: '99.97%',
    subtitle: 'Last 30 days',
    icon: Wifi,
    iconColor: 'text-amber-600',
    iconBgColor: 'bg-amber-50 dark:bg-amber-900/20',
    trend: {
      value: 0.02,
      label: 'improvement',
      direction: 'up',
    },
  },
};

// ── MiniStatsCard Variants ───────────────────────────────────────────────

const miniMeta: Meta<typeof MiniStatsCard> = {
  title: 'UI/StatsCard/MiniStatsCard',
  component: MiniStatsCard,
  tags: ['autodocs'],
};

export const MiniBlue: StoryObj<typeof MiniStatsCard> = {
  render: () => <MiniStatsCard title="Cameras" value="24" icon={Camera} color="blue" />,
};

export const MiniGreen: StoryObj<typeof MiniStatsCard> = {
  render: () => <MiniStatsCard title="Online" value="22" icon={Activity} color="green" />,
};

export const MiniRed: StoryObj<typeof MiniStatsCard> = {
  render: () => <MiniStatsCard title="Offline" value="2" icon={Wifi} color="red" />,
};

export const MiniAmber: StoryObj<typeof MiniStatsCard> = {
  render: () => <MiniStatsCard title="Alerts" value="7" icon={HardDrive} color="amber" />,
};

export const MiniPurple: StoryObj<typeof MiniStatsCard> = {
  render: () => <MiniStatsCard title="Users" value="18" icon={Users} color="purple" />,
};

// ── Playground ───────────────────────────────────────────────────────────

export const Playground: Story = {
  args: {
    title: 'Custom Stat',
    value: 100,
    subtitle: 'Custom subtitle',
    icon: Activity,
    iconColor: 'text-blue-600',
    iconBgColor: 'bg-blue-50',
  },
};
