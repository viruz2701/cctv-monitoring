import type { Meta, StoryObj } from '@storybook/react';
import { ToastProvider, useToast } from './Toast';
import { useEffect } from 'react';

// ── Wrapper to demonstrate toast notifications ──────────────────────────

function ToastDemo({ type, title, message, duration, undo }: {
  type: 'success' | 'error' | 'warning' | 'info';
  title: string;
  message?: string;
  duration?: number;
  undo?: boolean;
}) {
  const toast = useToast();

  useEffect(() => {
    const opts: any = { title };
    if (message) opts.message = message;
    if (duration) opts.duration = duration;
    if (undo) opts.undo = { label: 'Undo', onClick: () => alert('Undone!') };
    toast[type](opts);
  }, []);

  return (
    <div className="p-8 text-sm text-slate-500">
      Toast will appear in bottom-right corner.
    </div>
  );
}

const meta: Meta<typeof ToastProvider> = {
  title: 'UI/Toast',
  component: ToastProvider,
  tags: ['autodocs'],
};

export default meta;

// ── Success ──────────────────────────────────────────────────────────────

export const Success: StoryObj = {
  render: () => (
    <ToastProvider>
      <ToastDemo type="success" title="Device Updated" message="Firmware updated successfully" />
    </ToastProvider>
  ),
};

// ── Error ────────────────────────────────────────────────────────────────

export const Error: StoryObj = {
  render: () => (
    <ToastProvider>
      <ToastDemo type="error" title="Connection Failed" message="Unable to reach device at 192.168.1.100" />
    </ToastProvider>
  ),
};

// ── Warning ──────────────────────────────────────────────────────────────

export const Warning: StoryObj = {
  render: () => (
    <ToastProvider>
      <ToastDemo type="warning" title="Storage Almost Full" message="Disk usage at 92%" />
    </ToastProvider>
  ),
};

// ── Info ─────────────────────────────────────────────────────────────────

export const Info: StoryObj = {
  render: () => (
    <ToastProvider>
      <ToastDemo type="info" title="Scheduled Maintenance" message="System will restart at 02:00 AM" />
    </ToastProvider>
  ),
};

// ── With Undo ────────────────────────────────────────────────────────────

export const WithUndo: StoryObj = {
  render: () => (
    <ToastProvider>
      <ToastDemo type="info" title="Work Order Deleted" message="WO-2024-0156 has been removed" undo />
    </ToastProvider>
  ),
};

// ── Stacking ─────────────────────────────────────────────────────────────

function StackingDemo() {
  const toast = useToast();
  useEffect(() => {
    toast.success({ title: 'Operation 1 completed' });
    toast.info({ title: 'Processing next item' });
    toast.warning({ title: 'Disk space low' });
    toast.error({ title: 'Connection lost to NVR-02' });
  }, []);
  return <div className="p-8 text-sm text-slate-500">Multiple toasts stacking.</div>;
}

export const Stacking: StoryObj = {
  render: () => (
    <ToastProvider>
      <StackingDemo />
    </ToastProvider>
  ),
};

// ── Long Duration ────────────────────────────────────────────────────────

export const LongDuration: StoryObj = {
  render: () => (
    <ToastProvider>
      <ToastDemo type="info" title="Long Notification" message="This toast will stay for 10 seconds" duration={10000} />
    </ToastProvider>
  ),
};
