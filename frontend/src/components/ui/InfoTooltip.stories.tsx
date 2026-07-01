import type { Meta, StoryObj } from '@storybook/react';
import { InfoTooltip } from './InfoTooltip';
import { MemoryRouter } from 'react-router-dom';

const meta: Meta<typeof InfoTooltip> = {
  title: 'UI/InfoTooltip',
  component: InfoTooltip,
  tags: ['autodocs'],
  argTypes: {
    position: {
      control: 'select',
      options: ['top', 'bottom', 'left', 'right'],
    },
    delay: { control: { type: 'number', min: 0, max: 2000 } },
    iconSize: { control: { type: 'number', min: 10, max: 32 } },
  },
  decorators: [
    (Story) => (
      <MemoryRouter>
        <div className="flex items-center justify-center p-20">
          <Story />
        </div>
      </MemoryRouter>
    ),
  ],
};

export default meta;
type Story = StoryObj<typeof InfoTooltip>;

// ── Positions ─────────────────────────────────────────────────────────────

export const Top: Story = {
  args: { text: 'Tooltip on top position', position: 'top' },
};

export const Bottom: Story = {
  args: { text: 'Tooltip on bottom position', position: 'bottom' },
};

export const Left: Story = {
  args: { text: 'Tooltip on left position', position: 'left' },
};

export const Right: Story = {
  args: { text: 'Tooltip on right position', position: 'right' },
};

// ─── Use Cases ────────────────────────────────────────────────────────────

export const ComplianceTerm: Story = {
  args: {
    text: 'IEC 62443-3-3 requirement for industrial control systems security',
    position: 'top',
  },
};

export const WithGlossaryLink: Story = {
  args: {
    text: 'Service Level Agreement defines expected uptime and response times',
    glossaryTerm: 'SLA',
    position: 'top',
  },
};

export const LongText: Story = {
  args: {
    text: 'This is a longer explanation that demonstrates how the tooltip handles wrapping and extended content in the tooltip container',
    position: 'top',
  },
};

export const Delayed: Story = {
  args: {
    text: 'Appears after 800ms delay',
    position: 'top',
    delay: 800,
  },
};

export const LargeIcon: Story = {
  args: {
    text: 'Custom icon size',
    position: 'top',
    iconSize: 20,
  },
};

// ── All Positions ─────────────────────────────────────────────────────────

export const AllPositions: StoryObj = {
  decorators: [
    (Story) => (
      <MemoryRouter>
        <Story />
      </MemoryRouter>
    ),
  ],
  render: () => (
    <div className="flex items-center justify-center p-20">
      <div className="relative grid grid-cols-2 gap-20">
        <div className="flex flex-col items-center gap-8">
          <span className="text-xs text-slate-400">Top</span>
          <InfoTooltip text="Top tooltip" position="top" />
        </div>
        <div className="flex flex-col items-center gap-8">
          <span className="text-xs text-slate-400">Bottom</span>
          <InfoTooltip text="Bottom tooltip" position="bottom" />
        </div>
        <div className="flex flex-col items-center gap-8">
          <span className="text-xs text-slate-400">Left</span>
          <InfoTooltip text="Left tooltip" position="left" />
        </div>
        <div className="flex flex-col items-center gap-8">
          <span className="text-xs text-slate-400">Right</span>
          <InfoTooltip text="Right tooltip" position="right" />
        </div>
      </div>
    </div>
  ),
};

// ── Playground ────────────────────────────────────────────────────────────

export const Playground: Story = {
  args: {
    text: 'Custom tooltip content',
    position: 'top',
    delay: 200,
    iconSize: 14,
  },
};
