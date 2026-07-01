import type { Meta, StoryObj } from '@storybook/react';
import { useState, useEffect } from 'react';
import { BulkProgressModal, type BulkProgressState } from './BulkProgressModal';

const meta: Meta<typeof BulkProgressModal> = {
  title: 'UI/BulkProgressModal',
  component: BulkProgressModal,
  tags: ['autodocs'],
};

export default meta;

// ── Sample Items ──────────────────────────────────────────────────────────

const sampleItems = [
  { id: '1', label: 'NVR-01 firmware update', status: 'done' as const },
  { id: '2', label: 'CAM-101 config sync', status: 'done' as const },
  { id: '3', label: 'CAM-102 reboot', status: 'failed' as const, error: 'Connection timeout' },
  { id: '4', label: 'GW-01 backup', status: 'processing' as const },
  { id: '5', label: 'SENSOR-TEMP calibration', status: 'pending' as const },
  { id: '6', label: 'NVR-02 disk cleanup', status: 'pending' as const },
];

// ── In Progress ───────────────────────────────────────────────────────────

function InProgressDemo() {
  const [state, setState] = useState<BulkProgressState>({
    total: 6,
    items: sampleItems,
    isRunning: true,
    isCancelled: false,
    operationLabel: 'Processing devices...',
  });

  return (
    <div className="p-4">
      <BulkProgressModal
        state={state}
        onCancel={() => setState((prev) => ({ ...prev, isCancelled: true, isRunning: false }))}
        onRetryAll={() => {}}
        onClose={() => {}}
      />
    </div>
  );
}

export const InProgress: StoryObj = {
  render: () => <InProgressDemo />,
};

// ── Completed ─────────────────────────────────────────────────────────────

function CompletedDemo() {
  const [state] = useState<BulkProgressState>({
    total: 6,
    items: sampleItems.map((item) =>
      item.status === 'pending' || item.status === 'processing'
        ? { ...item, status: 'done' as const }
        : item,
    ),
    isRunning: false,
    isCancelled: false,
    operationLabel: 'Processing devices...',
  });

  return (
    <div className="p-4">
      <BulkProgressModal
        state={state}
        onCancel={() => {}}
        onRetryAll={() => {}}
        onClose={() => {}}
      />
    </div>
  );
}

export const Completed: StoryObj = {
  render: () => <CompletedDemo />,
};

// ── With Errors ───────────────────────────────────────────────────────────

function WithErrorsDemo() {
  const [state] = useState<BulkProgressState>({
    total: 4,
    items: [
      { id: '1', label: 'Device A', status: 'done' },
      { id: '2', label: 'Device B', status: 'failed', error: 'Connection refused' },
      { id: '3', label: 'Device C', status: 'failed', error: 'Authentication failed' },
      { id: '4', label: 'Device D', status: 'done' },
    ],
    isRunning: false,
    isCancelled: false,
    operationLabel: 'Bulk firmware update',
  });

  return (
    <div className="p-4">
      <BulkProgressModal
        state={state}
        onCancel={() => {}}
        onRetryAll={() => alert('Retrying all failed items...')}
        onRetryItem={(id) => alert(`Retrying item ${id}`)}
        onClose={() => {}}
      />
    </div>
  );
}

export const WithErrors: StoryObj = {
  render: () => <WithErrorsDemo />,
};

// ── All Failed ────────────────────────────────────────────────────────────

function AllFailedDemo() {
  const [state] = useState<BulkProgressState>({
    total: 3,
    items: [
      { id: '1', label: 'NVR-01', status: 'failed', error: 'Connection timeout' },
      { id: '2', label: 'CAM-101', status: 'failed', error: 'Invalid credentials' },
      { id: '3', label: 'GW-01', status: 'failed', error: 'Service unavailable' },
    ],
    isRunning: false,
    isCancelled: false,
    operationLabel: 'Bulk operation',
  });

  return (
    <div className="p-4">
      <BulkProgressModal
        state={state}
        onCancel={() => {}}
        onRetryAll={() => alert('Retrying all...')}
        onRetryItem={(id) => alert(`Retrying ${id}`)}
        onClose={() => {}}
      />
    </div>
  );
}

export const AllFailed: StoryObj = {
  render: () => <AllFailedDemo />,
};

// ── Cancelled ─────────────────────────────────────────────────────────────

function CancelledDemo() {
  const [state] = useState<BulkProgressState>({
    total: 5,
    items: [
      { id: '1', label: 'Device A', status: 'done' },
      { id: '2', label: 'Device B', status: 'done' },
      { id: '3', label: 'Device C', status: 'cancelled' },
      { id: '4', label: 'Device D', status: 'cancelled' },
      { id: '5', label: 'Device E', status: 'cancelled' },
    ],
    isRunning: false,
    isCancelled: true,
    operationLabel: 'Bulk operation (cancelled)',
  });

  return (
    <div className="p-4">
      <BulkProgressModal
        state={state}
        onCancel={() => {}}
        onRetryAll={() => {}}
        onClose={() => {}}
      />
    </div>
  );
}

export const Cancelled: StoryObj = {
  render: () => <CancelledDemo />,
};

// ── Simulated Progress ────────────────────────────────────────────────────

function SimulatedProgressDemo() {
  const [state, setState] = useState<BulkProgressState>({
    total: 6,
    items: [
      { id: '1', label: 'Device-001 firmware update', status: 'processing' },
      { id: '2', label: 'Device-002 config sync', status: 'pending' },
      { id: '3', label: 'Device-003 reboot', status: 'pending' },
      { id: '4', label: 'Device-004 backup', status: 'pending' },
      { id: '5', label: 'Device-005 calibration', status: 'pending' },
      { id: '6', label: 'Device-006 disk cleanup', status: 'pending' },
    ],
    isRunning: true,
    isCancelled: false,
    operationLabel: 'Processing devices...',
  });

  useEffect(() => {
    const timer = setInterval(() => {
      setState((prev) => {
        const items = [...prev.items];
        const nextPending = items.findIndex((i) => i.status === 'pending');
        const nextProcessing = items.findIndex((i) => i.status === 'processing');

        if (nextProcessing >= 0) {
          items[nextProcessing] = {
            ...items[nextProcessing],
            status: Math.random() > 0.2 ? 'done' : 'failed',
            error: Math.random() > 0.2 ? undefined : 'Simulated error',
          };
        }

        if (nextPending >= 0 && nextProcessing === -1) {
          items[nextPending] = { ...items[nextPending], status: 'processing' };
        }

        const isRunning = items.some((i) => i.status === 'pending' || i.status === 'processing');

        return { ...prev, items, isRunning };
      });
    }, 1500);

    return () => clearInterval(timer);
  }, []);

  return (
    <div className="p-4">
      <BulkProgressModal
        state={state}
        onCancel={() => setState((prev) => ({ ...prev, isCancelled: true, isRunning: false }))}
        onRetryAll={() => {}}
        onClose={() => {}}
      />
    </div>
  );
}

export const SimulatedProgress: StoryObj = {
  render: () => <SimulatedProgressDemo />,
};
