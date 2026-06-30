import type { Meta, StoryObj } from '@storybook/react';
import { ConditionalChecklist } from './ConditionalChecklist';

const meta: Meta<typeof ConditionalChecklist> = {
  title: 'Checklists/ConditionalChecklist',
  component: ConditionalChecklist,
  tags: ['autodocs'],
};

export default meta;
type Story = StoryObj<typeof ConditionalChecklist>;

export const Default: Story = {
  args: {
    workOrderId: 'WO-2025-0042',
    deviceType: 'nvr',
    onComplete: (summary) => console.log('Completed:', summary),
  },
};

export const WithPresetTemplate: Story = {
  args: {
    workOrderId: 'WO-2025-0043',
    templateId: 'nvr-monthly',
    onComplete: (summary) => console.log('Completed:', summary),
  },
};
