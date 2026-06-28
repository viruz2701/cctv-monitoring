import type { Meta, StoryObj } from '@storybook/react';
import { DragDropDashboard } from './DragDropDashboard';

const meta: Meta<typeof DragDropDashboard> = {
  title: 'Dashboard/DragDropDashboard',
  component: DragDropDashboard,
  decorators: [
    (Story) => (
      <div className="p-4 max-w-6xl mx-auto">
        <Story />
      </div>
    ),
  ],
  parameters: {
    layout: 'fullscreen',
    docs: {
      description: {
        component:
          'DragDropDashboard — Responsive draggable grid wrapper. ' +
          'Соответствует UX-14.2.2.',
      },
    },
  },
  tags: ['autodocs'],
};

export default meta;
type Story = StoryObj<typeof DragDropDashboard>;

const sampleWidgets = [
  { id: 'stats', content: <div className="bg-blue-100 rounded-lg p-4 h-32 flex items-center justify-center text-blue-700 font-medium">📊 Статистика</div>, minW: 2 },
  { id: 'chart', content: <div className="bg-green-100 rounded-lg p-4 h-64 flex items-center justify-center text-green-700 font-medium">📈 График</div>, minW: 2, minH: 2 },
  { id: 'alerts', content: <div className="bg-red-100 rounded-lg p-4 h-32 flex items-center justify-center text-red-700 font-medium">🔔 Оповещения</div> },
  { id: 'map', content: <div className="bg-purple-100 rounded-lg p-4 h-48 flex items-center justify-center text-purple-700 font-medium">🗺️ Карта</div>, minW: 2 },
  { id: 'recent', content: <div className="bg-yellow-100 rounded-lg p-4 h-40 flex items-center justify-center text-yellow-700 font-medium">🕐 Недавние</div> },
  { id: 'calendar', content: <div className="bg-pink-100 rounded-lg p-4 h-48 flex items-center justify-center text-pink-700 font-medium">📅 Календарь</div> },
];

export const Default: Story = {
  args: {
    widgets: sampleWidgets,
  },
};

export const CustomizeMode: Story = {
  args: {
    widgets: sampleWidgets,
    customizeMode: true,
    visibleWidgets: ['stats', 'chart', 'alerts', 'map'],
    onToggleWidget: (id, visible) => console.log(`Toggle ${id}: ${visible}`),
    onResetLayout: () => console.log('Reset layout'),
  },
};

export const LimitedWidgets: Story = {
  args: {
    widgets: sampleWidgets.slice(0, 3),
    visibleWidgets: ['stats', 'chart'],
  },
};

export const WithCustomStorageKey: Story = {
  args: {
    widgets: sampleWidgets,
    storageKey: 'storybook-dashboard-layout',
  },
};
