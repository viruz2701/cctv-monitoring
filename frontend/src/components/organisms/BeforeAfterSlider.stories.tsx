import type { Meta, StoryObj } from '@storybook/react';
import { BeforeAfterSlider } from './BeforeAfterSlider';

const meta: Meta<typeof BeforeAfterSlider> = {
  title: 'Organisms/BeforeAfterSlider',
  component: BeforeAfterSlider,
  tags: ['autodocs'],
};

export default meta;
type Story = StoryObj<typeof BeforeAfterSlider>;

export const Default: Story = {
  args: {
    beforeImage: 'https://placehold.co/600x400/1e3a5f/ffffff?text=Before',
    afterImage: 'https://placehold.co/600x400/2d7d46/ffffff?text=After',
    beforeLabel: 'До',
    afterLabel: 'После',
  },
};

export const WithCustomLabels: Story = {
  args: {
    beforeImage: 'https://placehold.co/600x400/dc2626/ffffff?text=Original',
    afterImage: 'https://placehold.co/600x400/16a34a/ffffff?text=Repaired',
    beforeLabel: 'Original',
    afterLabel: 'Repaired',
  },
};
