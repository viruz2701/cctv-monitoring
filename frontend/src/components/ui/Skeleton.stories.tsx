import type { Meta, StoryObj } from '@storybook/react';
import {
  Skeleton,
  SkeletonLine,
  SkeletonAvatar,
  SkeletonCard,
  SkeletonTable,
  SkeletonChart,
  SkeletonStatsCard,
  SkeletonFilterBar,
  SkeletonPage,
  SkeletonNotification,
  SkeletonProfileField,
} from './Skeleton';

const meta: Meta<typeof Skeleton> = {
  title: 'UI/Skeleton',
  component: Skeleton,
  tags: ['autodocs'],
  argTypes: {
    variant: {
      control: 'select',
      options: ['text', 'circular', 'rectangular', 'rounded'],
    },
    count: { control: { type: 'number', min: 1, max: 10 } },
  },
};

export default meta;
type Story = StoryObj<typeof Skeleton>;

// ── Base Skeleton Variants ───────────────────────────────────────────────

export const Text: Story = {
  args: { variant: 'text', width: 200, count: 1 },
};

export const Circular: Story = {
  args: { variant: 'circular', width: 48, height: 48 },
};

export const Rectangular: Story = {
  args: { variant: 'rectangular', width: 300, height: 200 },
};

export const Rounded: Story = {
  args: { variant: 'rounded', width: 300, height: 40 },
};

export const MultipleLines: Story = {
  args: { variant: 'text', width: '80%', count: 3 },
};

// ── Specialized Skeletons ────────────────────────────────────────────────

export const SkeletonLineStory: StoryObj = {
  name: 'SkeletonLine',
  render: () => (
    <div className="w-96 space-y-4">
      <SkeletonLine count={1} />
      <SkeletonLine count={2} widths={['100%', '75%']} />
      <SkeletonLine count={3} widths={['90%', '80%', '60%']} lastWidth="45%" />
    </div>
  ),
};

export const SkeletonAvatarStory: StoryObj = {
  name: 'SkeletonAvatar',
  render: () => (
    <div className="space-y-4">
      <SkeletonAvatar size="sm" />
      <SkeletonAvatar size="md" />
      <SkeletonAvatar size="lg" />
      <SkeletonAvatar size="xl" />
    </div>
  ),
};

export const SkeletonCardStory: StoryObj = {
  name: 'SkeletonCard',
  render: () => (
    <div className="w-80">
      <SkeletonCard count={1} headerLines={1} bodyLines={3} />
    </div>
  ),
};

export const SkeletonCardWithAvatar: StoryObj = {
  name: 'SkeletonCard (with avatar)',
  render: () => (
    <div className="w-80">
      <SkeletonCard count={1} headerLines={2} bodyLines={4} avatar />
    </div>
  ),
};

export const SkeletonTableStory: StoryObj = {
  name: 'SkeletonTable',
  render: () => (
    <div className="w-full">
      <SkeletonTable rows={5} columns={4} />
    </div>
  ),
};

export const SkeletonChartStory: StoryObj = {
  name: 'SkeletonChart',
  render: () => (
    <div className="w-full max-w-2xl">
      <SkeletonChart height={260} withHeader />
    </div>
  ),
};

export const SkeletonStatsCardStory: StoryObj = {
  name: 'SkeletonStatsCard',
  render: () => (
    <div className="grid grid-cols-3 gap-4 w-full max-w-3xl">
      <SkeletonStatsCard count={3} />
    </div>
  ),
};

export const SkeletonStatsCardWithTrend: StoryObj = {
  name: 'SkeletonStatsCard (with trend)',
  render: () => (
    <div className="grid grid-cols-2 gap-4 w-full max-w-2xl">
      <SkeletonStatsCard count={2} withTrend />
    </div>
  ),
};

export const SkeletonFilterBarStory: StoryObj = {
  name: 'SkeletonFilterBar',
  render: () => (
    <div className="w-full max-w-4xl">
      <SkeletonFilterBar />
    </div>
  ),
};

export const SkeletonPageStory: StoryObj = {
  name: 'SkeletonPage',
  render: () => (
    <div className="w-full max-w-4xl">
      <SkeletonPage title subtitle filter>
        <SkeletonTable rows={4} columns={5} />
      </SkeletonPage>
    </div>
  ),
};

export const SkeletonNotificationStory: StoryObj = {
  name: 'SkeletonNotification',
  render: () => (
    <div className="w-full max-w-md">
      <SkeletonNotification count={3} />
    </div>
  ),
};

export const SkeletonProfileFieldStory: StoryObj = {
  name: 'SkeletonProfileField',
  render: () => (
    <div className="w-full max-w-md space-y-4">
      <SkeletonProfileField />
      <SkeletonProfileField />
      <SkeletonProfileField />
    </div>
  ),
};

// ── Composition: Dashboard Loading ───────────────────────────────────────

export const DashboardLoading: StoryObj = {
  name: 'Dashboard Loading State',
  render: () => (
    <div className="space-y-6 w-full max-w-6xl p-6">
      {/* Header */}
      <div className="space-y-2">
        <Skeleton variant="text" width={200} height={28} />
        <Skeleton variant="text" width={300} height={16} />
      </div>

      {/* Stats Cards */}
      <div className="grid grid-cols-4 gap-4">
        <SkeletonStatsCard count={4} />
      </div>

      {/* Chart + Table */}
      <div className="grid grid-cols-2 gap-6">
        <SkeletonChart height={300} withHeader />
        <SkeletonChart height={300} withHeader />
      </div>

      {/* Table */}
      <SkeletonTable rows={3} columns={6} />
    </div>
  ),
};
