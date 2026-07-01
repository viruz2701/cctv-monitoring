import type { Meta, StoryObj } from '@storybook/react';
import { Notification, NotificationList } from './Notification';

const meta: Meta<typeof Notification> = {
  title: 'UI/Notification',
  component: Notification,
  tags: ['autodocs'],
  argTypes: {
    type: {
      control: 'select',
      options: ['info', 'success', 'warning', 'error'],
    },
    assertive: { control: 'boolean' },
  },
};

export default meta;
type Story = StoryObj<typeof Notification>;

// ── Types ─────────────────────────────────────────────────────────────────

export const Info: Story = {
  args: {
    type: 'info',
    title: 'Information',
    children: 'System maintenance scheduled for tonight at 2:00 AM.',
  },
};

export const Success: Story = {
  args: {
    type: 'success',
    title: 'Operation Successful',
    children: 'Firmware update completed for 3 devices.',
  },
};

export const Warning: Story = {
  args: {
    type: 'warning',
    title: 'Storage Warning',
    children: 'NVR-01 disk usage has reached 85%. Consider cleaning old recordings.',
  },
};

export const Error: Story = {
  args: {
    type: 'error',
    title: 'Connection Lost',
    children: 'Unable to reach CAM-102 at 192.168.1.102. Check network connectivity.',
  },
};

// ── Without Title ─────────────────────────────────────────────────────────

export const WithoutTitle: Story = {
  args: {
    type: 'info',
    children: 'A simple notification without a title.',
  },
};

// ── NotificationList ──────────────────────────────────────────────────────

export const NotificationListExample: StoryObj<typeof NotificationList> = {
  render: () => (
    <NotificationList className="space-y-2 max-w-md">
      <Notification type="success" title="Device Online">
        CAM-101 is back online after network recovery.
      </Notification>
      <Notification type="warning" title="SLA Alert">
        Response time for NVR-01 exceeded threshold.
      </Notification>
      <Notification type="info" title="Scheduled Task">
        Weekly health check completed for all devices.
      </Notification>
    </NotificationList>
  ),
};

// ── All Types Showcase ────────────────────────────────────────────────────

export const AllTypes: StoryObj = {
  render: () => (
    <div className="flex flex-col gap-3 max-w-md p-4">
      <Notification type="info" title="Info">
        This is an informational notification.
      </Notification>
      <Notification type="success" title="Success">
        Operation completed successfully.
      </Notification>
      <Notification type="warning" title="Warning">
        Please review this warning message.
      </Notification>
      <Notification type="error" title="Error">
        An error occurred while processing.
      </Notification>
    </div>
  ),
};

// ── Long Content ──────────────────────────────────────────────────────────

export const LongContent: Story = {
  args: {
    type: 'info',
    title: 'Detailed Notification',
    children: 'This notification contains a much longer message that demonstrates how the component handles extended content. It wraps naturally and maintains readability across multiple lines of text within the notification container.',
  },
};

// ── Playground ────────────────────────────────────────────────────────────

export const Playground: Story = {
  args: {
    type: 'info',
    title: 'Custom Notification',
    children: 'This is a customizable notification with dynamic content.',
    assertive: false,
  },
};
