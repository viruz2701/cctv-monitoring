import type { Meta, StoryObj } from '@storybook/react';
import { OfflineBanner } from './OfflineBanner';

const meta: Meta<typeof OfflineBanner> = {
  title: 'Layout/OfflineBanner',
  component: OfflineBanner,
  tags: ['autodocs'],
  argTypes: {
    queueCount: { control: { type: 'number', min: 0, max: 99 } },
  },
};

export default meta;
type Story = StoryObj<typeof OfflineBanner>;

// ── Offline Without Queue ────────────────────────────────────────────────

export const OfflineWithoutQueue: Story = {
  args: {
    queueCount: 0,
  },
  parameters: {
    // Simulate offline state
    docs: { description: { story: 'Shows "Offline mode" banner when browser is offline with no queued operations.' } },
  },
};

// ── Offline With Queue ───────────────────────────────────────────────────

export const OfflineWithQueue: Story = {
  args: {
    queueCount: 5,
  },
  parameters: {
    docs: { description: { story: 'Shows "Offline mode — 5 queued" when offline with pending operations.' } },
  },
};

// ── Offline Large Queue ──────────────────────────────────────────────────

export const OfflineLargeQueue: Story = {
  args: {
    queueCount: 23,
  },
};

// ── Online Syncing ───────────────────────────────────────────────────────

export const OnlineSyncing: Story = {
  args: {
    queueCount: 3,
  },
  parameters: {
    docs: { description: { story: 'When connection is restored, shows "Back online — syncing..." with remaining count.' } },
  },
};

// ── Online All Synced ────────────────────────────────────────────────────

export const OnlineAllSynced: Story = {
  args: {
    queueCount: 0,
  },
  parameters: {
    docs: { description: { story: 'When back online with no pending operations, the banner is hidden (returns null).' } },
  },
};

// ── Playground ───────────────────────────────────────────────────────────

export const Playground: Story = {
  args: {
    queueCount: 5,
  },
};
