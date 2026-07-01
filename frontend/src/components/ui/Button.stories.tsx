import type { Meta, StoryObj } from '@storybook/react';
import { Button, IconButton } from './Button';
import { Camera, Settings, Trash2, Plus } from './Icons';

const meta: Meta<typeof Button> = {
  title: 'UI/Button',
  component: Button,
  tags: ['autodocs'],
  argTypes: {
    variant: {
      control: 'select',
      options: ['primary', 'secondary', 'outline', 'ghost', 'danger'],
    },
    size: {
      control: 'select',
      options: ['sm', 'md', 'lg'],
    },
    loading: { control: 'boolean' },
    disabled: { control: 'boolean' },
    fullWidth: { control: 'boolean' },
    iconPosition: {
      control: 'select',
      options: ['left', 'right'],
    },
  },
};

export default meta;
type Story = StoryObj<typeof Button>;

// ── Variants ──────────────────────────────────────────────────────────────

export const Primary: Story = {
  args: { variant: 'primary', children: 'Primary Button' },
};

export const Secondary: Story = {
  args: { variant: 'secondary', children: 'Secondary Button' },
};

export const Outline: Story = {
  args: { variant: 'outline', children: 'Outline Button' },
};

export const Ghost: Story = {
  args: { variant: 'ghost', children: 'Ghost Button' },
};

export const Danger: Story = {
  args: { variant: 'danger', children: 'Danger Button' },
};

// ── Sizes ─────────────────────────────────────────────────────────────────

export const Small: Story = {
  args: { size: 'sm', children: 'Small' },
};

export const Medium: Story = {
  args: { size: 'md', children: 'Medium' },
};

export const Large: Story = {
  args: { size: 'lg', children: 'Large' },
};

// ── States ────────────────────────────────────────────────────────────────

export const Disabled: Story = {
  args: { disabled: true, children: 'Disabled' },
};

export const Loading: Story = {
  args: { loading: true, children: 'Saving...' },
};

export const FullWidth: Story = {
  args: { fullWidth: true, children: 'Full Width Button' },
};

// ── With Icon ─────────────────────────────────────────────────────────────

export const WithIconLeft: Story = {
  args: { icon: <Camera size={16} />, iconPosition: 'left', children: 'Add Camera' },
};

export const WithIconRight: Story = {
  args: { icon: <Settings size={16} />, iconPosition: 'right', children: 'Settings' },
};

export const IconOnly: Story = {
  args: { icon: <Plus size={16} />, children: 'Add', size: 'sm' },
};

// ── All Variants Showcase ─────────────────────────────────────────────────

export const AllVariants: StoryObj = {
  render: () => (
    <div className="flex flex-wrap gap-4 p-4">
      <div className="flex flex-col gap-3">
        <Button variant="primary">Primary</Button>
        <Button variant="secondary">Secondary</Button>
        <Button variant="outline">Outline</Button>
        <Button variant="ghost">Ghost</Button>
        <Button variant="danger">Danger</Button>
      </div>
      <div className="flex flex-col gap-3">
        <Button variant="primary" size="sm">Small</Button>
        <Button variant="primary" size="md">Medium</Button>
        <Button variant="primary" size="lg">Large</Button>
      </div>
      <div className="flex flex-col gap-3">
        <Button variant="primary" loading>Loading</Button>
        <Button variant="primary" disabled>Disabled</Button>
        <Button variant="outline" disabled>Disabled</Button>
      </div>
    </div>
  ),
};

// ── IconButton Variants ───────────────────────────────────────────────────

export const IconButtonPrimary: StoryObj = {
  render: () => (
    <div className="flex gap-2 p-4">
      <IconButton icon={<Settings size={16} />} label="Settings" variant="ghost" />
      <IconButton icon={<Trash2 size={16} />} label="Delete" variant="danger" />
      <IconButton icon={<Plus size={16} />} label="Add" variant="primary" size="sm" />
    </div>
  ),
};

// ── Playground ────────────────────────────────────────────────────────────

export const Playground: Story = {
  args: {
    variant: 'primary',
    size: 'md',
    children: 'Custom Button',
    loading: false,
    disabled: false,
    fullWidth: false,
  },
};
