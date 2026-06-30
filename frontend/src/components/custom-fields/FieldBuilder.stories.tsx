import type { Meta, StoryObj } from '@storybook/react';
import { FieldBuilder } from './FieldBuilder';

const meta: Meta<typeof FieldBuilder> = {
  title: 'CustomFields/FieldBuilder',
  component: FieldBuilder,
  tags: ['autodocs'],
};

export default meta;
type Story = StoryObj<typeof FieldBuilder>;

export const ForDevice: Story = {
  args: {
    entityType: 'device',
    onFieldsChange: (fields) => console.log('Fields changed:', fields),
  },
};

export const ForWorkOrder: Story = {
  args: {
    entityType: 'work_order',
    onFieldsChange: (fields) => console.log('Fields changed:', fields),
  },
};
