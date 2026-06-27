import type { Meta, StoryObj } from '@storybook/react';
import { WorkspaceSwitcher } from './WorkspaceSwitcher';
import { BrowserRouter } from 'react-router-dom';

const meta: Meta<typeof WorkspaceSwitcher> = {
  title: 'Layout/WorkspaceSwitcher',
  component: WorkspaceSwitcher,
  tags: ['autodocs'],
  decorators: [
    (Story) => (
      <BrowserRouter>
        <div className="p-4">
          <Story />
        </div>
      </BrowserRouter>
    ),
  ],
};

export default meta;
type Story = StoryObj<typeof WorkspaceSwitcher>;

// ── Default ──────────────────────────────────────────────────────────────

export const Default: Story = {};

// ── Playground ───────────────────────────────────────────────────────────

export const Playground: Story = {};
