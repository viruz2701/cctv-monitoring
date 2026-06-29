import type { Meta, StoryObj } from '@storybook/react';
import { Tooltip } from './Tooltip';
import { Button } from './Button';
import { Info, Settings, Trash2 } from './Icons';

const meta: Meta<typeof Tooltip> = {
  title: 'UI/Tooltip',
  component: Tooltip,
  tags: ['autodocs'],
  argTypes: {
    position: {
      control: 'select',
      options: ['top', 'bottom', 'left', 'right'],
    },
    delay: { control: { type: 'number', min: 0, max: 2000 } },
    hideDelay: { control: { type: 'number', min: 0, max: 2000 } },
  },
};

export default meta;
type Story = StoryObj<typeof Tooltip>;

// ── Positions ────────────────────────────────────────────────────────────

export const Top: Story = {
  args: {
    content: 'Tooltip on top',
    position: 'top',
    children: <Button variant="outline">Hover me (top)</Button>,
  },
};

export const Bottom: Story = {
  args: {
    content: 'Tooltip on bottom',
    position: 'bottom',
    children: <Button variant="outline">Hover me (bottom)</Button>,
  },
};

export const Left: Story = {
  args: {
    content: 'Tooltip on left',
    position: 'left',
    children: <Button variant="outline">Hover me (left)</Button>,
  },
};

export const Right: Story = {
  args: {
    content: 'Tooltip on right',
    position: 'right',
    children: <Button variant="outline">Hover me (right)</Button>,
  },
};

// ── Use Cases ────────────────────────────────────────────────────────────

export const IconButtonTooltip: StoryObj = {
  render: () => (
    <div className="flex gap-4 p-8">
      <Tooltip content="Settings" position="top">
        <button className="p-2 rounded-lg hover:bg-slate-100 dark:hover:bg-slate-800">
          <Settings className="w-5 h-5 text-slate-600 dark:text-slate-400" />
        </button>
      </Tooltip>
      <Tooltip content="Delete item" position="top">
        <button className="p-2 rounded-lg hover:bg-slate-100 dark:hover:bg-slate-800 text-red-500">
          <Trash2 className="w-5 h-5" />
        </button>
      </Tooltip>
      <Tooltip content="More information" position="top">
        <button className="p-2 rounded-lg hover:bg-slate-100 dark:hover:bg-slate-800">
          <Info className="w-5 h-5 text-blue-500" />
        </button>
      </Tooltip>
    </div>
  ),
};

export const LongText: Story = {
  args: {
    content: 'This is a longer tooltip text that demonstrates text wrapping behavior for accessibility',
    position: 'top',
    children: <Button variant="outline">Long tooltip</Button>,
  },
};

export const WithDelay: Story = {
  args: {
    content: 'Appears after 1 second',
    position: 'top',
    delay: 1000,
    children: <Button variant="outline">Delayed (1s)</Button>,
  },
};

export const AllPositions: StoryObj = {
  render: () => (
    <div className="flex items-center justify-center p-16">
      <div className="grid grid-cols-2 gap-16">
        <Tooltip content="Top tooltip" position="top">
          <Button variant="outline">Top</Button>
        </Tooltip>
        <Tooltip content="Bottom tooltip" position="bottom">
          <Button variant="outline">Bottom</Button>
        </Tooltip>
        <Tooltip content="Left tooltip" position="left">
          <Button variant="outline">Left</Button>
        </Tooltip>
        <Tooltip content="Right tooltip" position="right">
          <Button variant="outline">Right</Button>
        </Tooltip>
      </div>
    </div>
  ),
};

// ── Playground ───────────────────────────────────────────────────────────

export const Playground: Story = {
  args: {
    content: 'Custom tooltip content',
    position: 'top',
    delay: 300,
    hideDelay: 150,
    children: <Button>Hover me</Button>,
  },
};
