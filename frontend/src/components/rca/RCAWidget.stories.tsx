import type { Meta, StoryObj } from '@storybook/react';
import { MemoryRouter } from 'react-router-dom';
import { RCAWidget } from './RCAWidget';

const meta: Meta<typeof RCAWidget> = {
  title: 'RCA/RCAWidget',
  component: RCAWidget,
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
type Story = StoryObj<typeof RCAWidget>;

export const Default: Story = {
  args: {
    deviceId: 'cam-012',
  },
};
