import type { Meta, StoryObj } from '@storybook/react';
import { P2PRegistrationForm } from './P2PRegistrationForm';

const meta: Meta<typeof P2PRegistrationForm> = {
  title: 'P2P/P2PRegistrationForm',
  component: P2PRegistrationForm,
  tags: ['autodocs'],
};

export default meta;
type Story = StoryObj<typeof P2PRegistrationForm>;

export const Open: Story = {
  args: {
    isOpen: true,
    onClose: () => {},
  },
};

export const Closed: Story = {
  args: {
    isOpen: false,
    onClose: () => {},
  },
};
