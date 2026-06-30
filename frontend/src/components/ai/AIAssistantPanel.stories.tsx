import type { Meta, StoryObj } from '@storybook/react';
import { AIAssistantPanel } from './AIAssistantPanel';

const meta: Meta<typeof AIAssistantPanel> = {
  title: 'AI/AIAssistantPanel',
  component: AIAssistantPanel,
  tags: ['autodocs'],
};

export default meta;
type Story = StoryObj<typeof AIAssistantPanel>;

export const DefaultOpen: Story = {
  args: {
    defaultOpen: true,
  },
};

export const DefaultClosed: Story = {
  args: {
    defaultOpen: false,
  },
};
