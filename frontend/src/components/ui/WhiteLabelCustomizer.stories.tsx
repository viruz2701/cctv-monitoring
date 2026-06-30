import type { Meta, StoryObj } from '@storybook/react';
import { WhiteLabelCustomizer } from './WhiteLabelCustomizer';

const meta: Meta<typeof WhiteLabelCustomizer> = {
  title: 'UI/WhiteLabelCustomizer',
  component: WhiteLabelCustomizer,
  tags: ['autodocs'],
};

export default meta;
type Story = StoryObj<typeof WhiteLabelCustomizer>;

export const Default: Story = {};
