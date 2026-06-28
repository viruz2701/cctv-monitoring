import type { Meta, StoryObj } from '@storybook/react';
import { SavedViews } from './SavedViews';

const meta: Meta<typeof SavedViews> = {
  title: 'UI/SavedViews',
  component: SavedViews,
  decorators: [
    (Story) => (
      <div className="p-8 max-w-md mx-auto">
        <Story />
      </div>
    ),
  ],
  parameters: {
    layout: 'centered',
    docs: {
      description: {
        component:
          'SavedViews — компонент для сохранения/загрузки/управления фильтрами. ' +
          'Соответствует P1-UX.8: Saved Filters.',
      },
    },
  },
  tags: ['autodocs'],
};

export default meta;
type Story = StoryObj<typeof SavedViews>;

const defaultFilterState = {
  filters: { status: 'active', priority: 'high' },
  sort: { column: 'created_at', direction: 'desc' as const },
};

export const Default: Story = {
  args: {
    page: 'devices',
    currentFilterState: defaultFilterState,
    onApplyView: (view) => console.log('Apply view:', view),
  },
};

export const WithViews: Story = {
  args: {
    page: 'work-orders',
    currentFilterState: defaultFilterState,
    onApplyView: (view) => console.log('Apply view:', view),
  },
  parameters: {
    docs: {
      description: {
        story: 'Page with pre-existing saved views.',
      },
    },
  },
};

export const Mobile: Story = {
  args: {
    page: 'sites',
    currentFilterState: defaultFilterState,
    onApplyView: (view) => console.log('Apply view:', view),
  },
  parameters: {
    viewport: { defaultViewport: 'mobile1' },
  },
};

export const CustomLabel: Story = {
  args: {
    page: 'alerts',
    currentFilterState: defaultFilterState,
    onApplyView: (view) => console.log('Apply view:', view),
    buttonLabel: '💾 Save Current Filters',
  },
};
