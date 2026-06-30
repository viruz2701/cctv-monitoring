import type { Meta, StoryObj } from '@storybook/react';
import { PTZControls } from './PTZControls';

const meta: Meta<typeof PTZControls> = {
  title: 'P2P/PTZControls',
  component: PTZControls,
  tags: ['autodocs'],
};

export default meta;
type Story = StoryObj<typeof PTZControls>;

export const Enabled: Story = {
  args: {
    deviceId: 'cam-001',
    disabled: false,
  },
};

export const Disabled: Story = {
  args: {
    deviceId: 'cam-001',
    disabled: true,
  },
};
