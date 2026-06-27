import type { Meta, StoryObj } from '@storybook/react';
import { Alert } from './Alert';

const meta: Meta<typeof Alert> = {
  title: 'UI/Alert',
  component: Alert,
  tags: ['autodocs'],
  argTypes: {
    variant: {
      control: 'select',
      options: ['info', 'success', 'warning', 'error'],
    },
    title: { control: 'text' },
    assertive: { control: 'boolean' },
  },
};

export default meta;
type Story = StoryObj<typeof Alert>;

// ── Variants ─────────────────────────────────────────────────────────────

export const Info: Story = {
  args: {
    variant: 'info',
    children: 'System is operating normally. All services are running.',
    title: 'System Status',
  },
};

export const Success: Story = {
  args: {
    variant: 'success',
    children: 'Device firmware has been updated successfully.',
    title: 'Update Complete',
  },
};

export const Warning: Story = {
  args: {
    variant: 'warning',
    children: 'Storage is at 85% capacity. Consider archiving old recordings.',
    title: 'Storage Warning',
  },
};

export const Error: Story = {
  args: {
    variant: 'error',
    children: 'Failed to connect to camera 192.168.1.100. Check network connection.',
    title: 'Connection Error',
  },
};

// ── With Close Button ────────────────────────────────────────────────────

export const WithClose: Story = {
  args: {
    variant: 'info',
    title: 'Dismissable',
    children: 'This alert can be closed by clicking the X button.',
    onClose: () => alert('Alert closed!'),
  },
};

// ── Long Content ─────────────────────────────────────────────────────────

export const LongContent: Story = {
  args: {
    variant: 'warning',
    title: 'Multiple Issues Detected',
    children: (
      <ul className="list-disc pl-4 space-y-1">
        <li>Camera NVR-01 has been offline for 3 hours</li>
        <li>Storage pool /data is at 92% capacity</li>
        <li>SSL certificate for api.example.com expires in 7 days</li>
        <li>Firmware update available for 12 devices</li>
      </ul>
    ),
  },
};

// ── Without Title ────────────────────────────────────────────────────────

export const WithoutTitle: Story = {
  args: {
    variant: 'error',
    children: 'Critical system error: Database connection timeout exceeded.',
  },
};

// ── Playground ───────────────────────────────────────────────────────────

export const Playground: Story = {
  args: {
    variant: 'info',
    title: 'Playground Alert',
    children: 'Customize this alert with the controls panel.',
    onClose: undefined,
  },
};
