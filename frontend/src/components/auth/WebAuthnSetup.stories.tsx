import type { Meta, StoryObj } from '@storybook/react';
import { WebAuthnSetup } from './WebAuthnSetup';

const meta: Meta<typeof WebAuthnSetup> = {
  title: 'Auth/WebAuthnSetup',
  component: WebAuthnSetup,
  tags: ['autodocs'],
};

export default meta;
type Story = StoryObj<typeof WebAuthnSetup>;

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
