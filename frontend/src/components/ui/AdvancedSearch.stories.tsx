import type { Meta, StoryObj } from '@storybook/react';
import { AdvancedSearch, type SearchFilters } from './AdvancedSearch';

const defaultFilters: SearchFilters = {
  query: '',
  facets: {},
};

const meta: Meta<typeof AdvancedSearch> = {
  title: 'UI/AdvancedSearch',
  component: AdvancedSearch,
  tags: ['autodocs'],
};

export default meta;
type Story = StoryObj<typeof AdvancedSearch>;

export const Default: Story = {
  args: {
    placeholder: 'Search devices, sites, work orders...',
    filters: defaultFilters,
    onFiltersChange: (filters) => console.log('Filters:', filters),
    onSearch: () => console.log('Search triggered'),
  },
};

export const WithQuery: Story = {
  args: {
    placeholder: 'Search...',
    filters: { query: 'camera', facets: {} },
    onFiltersChange: (filters) => console.log('Filters:', filters),
    onSearch: () => console.log('Search triggered'),
  },
};
