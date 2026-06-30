import type { Meta, StoryObj } from '@storybook/react';
import { WebhookRetryPolicy } from './WebhookRetryPolicy';

const meta: Meta<typeof WebhookRetryPolicy> = {
  title: 'Webhooks/WebhookRetryPolicy',
  component: WebhookRetryPolicy,
  tags: ['autodocs'],
};

export default meta;
type Story = StoryObj<typeof WebhookRetryPolicy>;

const mockRegister = () => {
  const fn = (name: string) => ({ name, onChange: () => {}, onBlur: () => {}, ref: () => {} });
  fn as any;
  return fn;
};

export const Default: Story = {
  args: {
    register: (() => {}) as any,
    errors: {},
    watchRetryBackoff: false,
  },
};

export const WithBackoff: Story = {
  args: {
    register: (() => {}) as any,
    errors: {},
    watchRetryBackoff: true,
  },
};
