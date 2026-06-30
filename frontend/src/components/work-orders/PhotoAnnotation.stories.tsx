import type { Meta, StoryObj } from '@storybook/react';
import { PhotoAnnotation } from './PhotoAnnotation';

const meta: Meta<typeof PhotoAnnotation> = {
  title: 'WorkOrders/PhotoAnnotation',
  component: PhotoAnnotation,
  tags: ['autodocs'],
  parameters: {
    docs: {
      description: {
        component:
          'Advanced photo annotation tool with canvas-based drawing. ' +
          'Supports arrow, freehand, text, highlight, circle, blur, and measurement tools. ' +
          'Includes undo/redo, zoom, and PNG export with watermark.',
      },
    },
  },
};

export default meta;
type Story = StoryObj<typeof PhotoAnnotation>;

export const Default: Story = {
  args: {
    imageUrl: 'https://placehold.co/800x600/1e293b/ffffff?text=Camera+View',
  },
};

export const ReadOnly: Story = {
  args: {
    imageUrl: 'https://placehold.co/800x600/1e293b/ffffff?text=Camera+View',
    readOnly: true,
  },
};

export const WithInitialElements: Story = {
  args: {
    imageUrl: 'https://placehold.co/800x600/1e293b/ffffff?text=Camera+View',
    initialElements: [
      {
        id: 'el-1',
        type: 'arrow',
        color: '#ef4444',
        strokeWidth: 4,
        start: { x: 100, y: 100 },
        end: { x: 300, y: 200 },
      },
      {
        id: 'el-2',
        type: 'circle',
        color: '#22c55e',
        strokeWidth: 4,
        center: { x: 400, y: 300 },
        radius: 80,
      },
    ],
  },
};
