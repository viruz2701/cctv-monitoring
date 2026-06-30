import type { Meta, StoryObj } from '@storybook/react';
import { LanguageSwitcher } from './LanguageSwitcher';

const meta: Meta<typeof LanguageSwitcher> = {
  title: 'UI/LanguageSwitcher',
  component: LanguageSwitcher,
  tags: ['autodocs'],
};

export default meta;
type Story = StoryObj<typeof LanguageSwitcher>;

export const Default: Story = {};

export const InHeader: Story = {
  decorators: [
    (Story) => (
      <div className="flex items-center gap-4 p-4 bg-slate-800 rounded-lg">
        <Story />
      </div>
    ),
  ],
};
