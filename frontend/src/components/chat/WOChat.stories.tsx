import type { Meta, StoryObj } from '@storybook/react';
import { WOChat } from './WOChat';

const meta: Meta<typeof WOChat> = {
  title: 'Chat/WOChat',
  component: WOChat,
  tags: ['autodocs'],
};

export default meta;
type Story = StoryObj<typeof WOChat>;

export const Open: Story = {
  args: {
    woId: 'WO-2025-0042',
    isOpen: true,
    onClose: () => {},
  },
};

export const Closed: Story = {
  args: {
    woId: 'WO-2025-0042',
    isOpen: false,
    onClose: () => {},
  },
};
