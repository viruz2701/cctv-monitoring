import type { Meta, StoryObj } from '@storybook/react';
import { ShortcutsCheatsheet } from './ShortcutsCheatsheet';

const meta: Meta<typeof ShortcutsCheatsheet> = {
  title: 'UI/ShortcutsCheatsheet',
  component: ShortcutsCheatsheet,
  tags: ['autodocs'],
};

export default meta;
type Story = StoryObj<typeof ShortcutsCheatsheet>;

export const Default: Story = {
  args: {
    isOpen: true,
    onClose: () => {},
  },
};
