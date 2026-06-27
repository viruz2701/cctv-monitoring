import type { Meta, StoryObj } from '@storybook/react';
import { BulkProgressModal, type BulkProgressState } from './BulkProgressModal';

const meta: Meta<typeof BulkProgressModal> = {
  title: 'UI/BulkProgressModal',
  component: BulkProgressModal,
  tags: ['autodocs'],
};

export default meta;
type Story = StoryObj<typeof BulkProgressModal>;

// ── In Progress ──────────────────────────────────────────────────────────

const inProgressState: BulkProgressState = {
  total: 5,
  items: [
    { id: '1', label: 'NVR-01 firmware update', status: 'done' },
    { id: '2', label: 'CAM-101 firmware update', status: 'done' },
    { id: '3', label: 'CAM-102 firmware update', status: 'processing' },
    { id: '4', label: 'CAM-103 firmware update', status: 'pending' },
    { id: '5', label: 'NVR-02 firmware update', status: 'pending' },
  ],
  isRunning: true,
  isCancelled: false,
  operationLabel: 'Updating firmware...',
};

export const InProgress: Story = {
  args: {
    state: inProgressState,
    onCancel: () => alert('Cancelled'),
    onRetryAll: () => alert('Retry all'),
    onRetryItem: (id) => alert(`Retry ${id}`),
    onClose: () => alert('Close'),
  },
};

// ── Complete ─────────────────────────────────────────────────────────────

const completeState: BulkProgressState = {
  total: 5,
  items: [
    { id: '1', label: 'NVR-01 firmware update', status: 'done' },
    { id: '2', label: 'CAM-101 firmware update', status: 'done' },
    { id: '3', label: 'CAM-102 firmware update', status: 'done' },
    { id: '4', label: 'CAM-103 firmware update', status: 'done' },
    { id: '5', label: 'NVR-02 firmware update', status: 'done' },
  ],
  isRunning: false,
  isCancelled: false,
  operationLabel: 'Firmware update complete',
};

export const Complete: Story = {
  args: {
    state: completeState,
    onCancel: () => {},
    onRetryAll: () => {},
    onClose: () => alert('Close'),
  },
};

// ── With Errors ──────────────────────────────────────────────────────────

const withErrorsState: BulkProgressState = {
  total: 6,
  items: [
    { id: '1', label: 'NVR-01 firmware update', status: 'done' },
    { id: '2', label: 'CAM-101 firmware update', status: 'done' },
    { id: '3', label: 'CAM-102 firmware update', status: 'failed', error: 'Connection timeout after 30s' },
    { id: '4', label: 'CAM-103 firmware update', status: 'done' },
    { id: '5', label: 'NVR-02 firmware update', status: 'failed', error: 'Incompatible firmware version' },
    { id: '6', label: 'CAM-201 firmware update', status: 'cancelled' },
  ],
  isRunning: false,
  isCancelled: false,
  operationLabel: 'Bulk firmware update - 2 failed',
};

export const WithErrors: Story = {
  args: {
    state: withErrorsState,
    onCancel: () => {},
    onRetryAll: () => alert('Retrying all failed'),
    onRetryItem: (id) => alert(`Retrying ${id}`),
    onClose: () => alert('Close'),
  },
};

// ── Large Batch ──────────────────────────────────────────────────────────

const largeBatchState: BulkProgressState = {
  total: 20,
  items: Array.from({ length: 20 }, (_, i) => ({
    id: `${i}`,
    label: `Device ${String.fromCharCode(65 + Math.floor(i / 5))}-${(i % 5) + 1} configuration`,
    status: (i < 8 ? 'done' : i < 12 ? 'processing' : i < 15 ? 'failed' : 'pending') as any,
    error: i >= 12 && i < 15 ? 'Configuration rejected' : undefined,
  })),
  isRunning: true,
  isCancelled: false,
  operationLabel: 'Applying bulk configuration...',
};

export const LargeBatch: Story = {
  args: {
    state: largeBatchState,
    onCancel: () => alert('Cancel'),
    onRetryAll: () => alert('Retry all'),
    onRetryItem: (id) => alert(`Retry ${id}`),
    onClose: () => alert('Close'),
  },
};

// ── Cancelled ────────────────────────────────────────────────────────────

const cancelledState: BulkProgressState = {
  total: 10,
  items: [
    { id: '1', label: 'Device A-1 setup', status: 'done' },
    { id: '2', label: 'Device A-2 setup', status: 'done' },
    { id: '3', label: 'Device A-3 setup', status: 'cancelled' },
    { id: '4', label: 'Device A-4 setup', status: 'cancelled' },
    { id: '5', label: 'Device A-5 setup', status: 'cancelled' },
  ],
  isRunning: false,
  isCancelled: true,
  operationLabel: 'Bulk setup - cancelled',
};

export const Cancelled: Story = {
  args: {
    state: cancelledState,
    onCancel: () => {},
    onRetryAll: () => alert('Retry all'),
    onClose: () => alert('Close'),
  },
};

// ── Playground ───────────────────────────────────────────────────────────

export const Playground: Story = {
  args: {
    state: inProgressState,
    onCancel: () => alert('Cancel'),
    onRetryAll: () => alert('Retry all'),
    onRetryItem: (id) => alert(`Retry ${id}`),
    onClose: () => alert('Close'),
  },
};
