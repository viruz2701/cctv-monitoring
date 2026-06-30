import type { Meta, StoryObj } from '@storybook/react';
import { DeviceAuditLog } from './DeviceAuditLog';

const meta: Meta<typeof DeviceAuditLog> = {
  title: 'Devices/DeviceAuditLog',
  component: DeviceAuditLog,
  tags: ['autodocs'],
};

export default meta;
type Story = StoryObj<typeof DeviceAuditLog>;

export const Default: Story = {
  args: {
    deviceId: 'dev-001',
  },
};
