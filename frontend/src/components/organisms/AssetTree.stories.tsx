import type { Meta, StoryObj } from '@storybook/react';
import { MemoryRouter } from 'react-router-dom';
import { AssetTree, type AssetTreeNode } from './AssetTree';

const meta: Meta<typeof AssetTree> = {
  title: 'Organisms/AssetTree',
  component: AssetTree,
  tags: ['autodocs'],
  decorators: [
    (Story) => (
      <MemoryRouter>
        <div className="max-w-md">
          <Story />
        </div>
      </MemoryRouter>
    ),
  ],
};

export default meta;
type Story = StoryObj<typeof AssetTree>;

const mockTree: AssetTreeNode[] = [
  {
    id: 'org-1', name: 'Главный офис', type: 'organization',
    status: 'online', children: [
      {
        id: 'site-1', name: 'Офис Москва', type: 'site',
        status: 'online', children: [
          {
            id: 'bld-1', name: 'Бизнес-центр "Москва-Сити"', type: 'building',
            status: 'online', children: [
              {
                id: 'fl-1', name: 'Этаж 12', type: 'floor',
                status: 'online', children: [
                  {
                    id: 'room-1', name: 'Серверная', type: 'room',
                    status: 'online', children: [
                      { id: 'cam-1', name: 'Камера 1', type: 'device', status: 'online', children: [], deviceCount: 0, level: 5 },
                      { id: 'cam-2', name: 'Камера 2', type: 'device', status: 'offline', children: [], deviceCount: 0, level: 5 },
                    ], deviceCount: 2, level: 4,
                  },
                ], deviceCount: 2, level: 3,
              },
            ], deviceCount: 2, level: 2,
          },
        ], deviceCount: 2, level: 1,
      },
    ], deviceCount: 2, level: 0,
  },
];

export const WithData: Story = {
  args: {
    data: mockTree,
    loading: false,
  },
};

export const Loading: Story = {
  args: {
    data: [],
    loading: true,
  },
};

export const Empty: Story = {
  args: {
    data: [],
    loading: false,
  },
};
