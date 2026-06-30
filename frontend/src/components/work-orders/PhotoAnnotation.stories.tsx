import type { Meta, StoryObj } from '@storybook/react';
import { PhotoAnnotation } from './PhotoAnnotation';

const meta: Meta<typeof PhotoAnnotation> = {
  title: 'WorkOrders/PhotoAnnotation',
  component: PhotoAnnotation,
  tags: ['autodocs'],
};

export default meta;
type Story = StoryObj<typeof PhotoAnnotation>;

export const Default: Story = {
  args: {
    imageUrl: 'https://placehold.co/800x600/1e293b/ffffff?text=Camera+View',
  },
};
