import type { Meta, StoryObj } from '@storybook/react';
import RCAGraph from './RCAGraph';

const meta: Meta<typeof RCAGraph> = {
  title: 'RCA/RCAGraph',
  component: RCAGraph,
  tags: ['autodocs'],
};

export default meta;
type Story = StoryObj<typeof RCAGraph>;

export const Default: Story = {
  args: {
    deviceId: 'cam-012',
    onClose: () => {},
  },
};
