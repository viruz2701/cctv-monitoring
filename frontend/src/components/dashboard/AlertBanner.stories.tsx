import type { Meta, StoryObj } from '@storybook/react';
import { AlertBanner } from './AlertBanner';
import { BrowserRouter } from 'react-router-dom';

const meta: Meta<typeof AlertBanner> = {
  title: 'Dashboard/AlertBanner',
  component: AlertBanner,
  tags: ['autodocs'],
  decorators: [
    (Story) => (
      <BrowserRouter>
        <div className="p-4 max-w-4xl">
          <Story />
        </div>
      </BrowserRouter>
    ),
  ],
};

export default meta;
type Story = StoryObj<typeof AlertBanner>;

// ── Default ──────────────────────────────────────────────────────────────

export const Default: Story = {};

// ── Playground ───────────────────────────────────────────────────────────

export const Playground: Story = {};
