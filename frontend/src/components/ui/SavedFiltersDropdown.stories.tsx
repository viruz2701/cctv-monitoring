import type { Meta, StoryObj } from '@storybook/react';
import { SavedFiltersDropdown } from './SavedFiltersDropdown';

const meta: Meta<typeof SavedFiltersDropdown> = {
  title: 'UI/SavedFiltersDropdown',
  component: SavedFiltersDropdown,
  tags: ['autodocs'],
};

export default meta;
type Story = StoryObj<typeof SavedFiltersDropdown>;

export const Default: Story = {
  args: {
    page: 'devices',
    currentFilterState: { filters: {}, sort: { column: 'name', direction: 'asc' } },
    onApplyView: (view) => console.log('Applied view:', view),
  },
};
