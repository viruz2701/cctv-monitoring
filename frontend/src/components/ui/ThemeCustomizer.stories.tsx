import type { Meta, StoryObj } from '@storybook/react';
import { ThemeCustomizer } from './ThemeCustomizer';

const meta: Meta<typeof ThemeCustomizer> = {
  title: 'UI/ThemeCustomizer',
  component: ThemeCustomizer,
  tags: ['autodocs'],
};

export default meta;
type Story = StoryObj<typeof ThemeCustomizer>;

export const Default: Story = {};
