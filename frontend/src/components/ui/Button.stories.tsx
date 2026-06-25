import type { Meta, StoryObj } from '@storybook/react';
import { Button, IconButton } from './Button';
import { Settings, Plus, Trash2 } from 'lucide-react';

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
  },
};

export default meta;
type Story = StoryObj<typeof Button>;

// ── Variants ─────────────────────────────────────────────────────────────

export const Primary: Story = {
  args: {
    variant: 'primary',
    children: 'Primary Button',
  },
};

export const Secondary: Story = {
  args: {
    variant: 'secondary',
    children: 'Secondary Button',
  },
};

export const Outline: Story = {
  args: {
    variant: 'outline',
    children: 'Outline Button',
  },
};

export const Ghost: Story = {
  args: {
    variant: 'ghost',
    children: 'Ghost Button',
  },
};

export const Danger: Story = {
  args: {
    variant: 'danger',
    children: 'Delete',
  },
};

// ── Sizes ────────────────────────────────────────────────────────────────

export const Small: Story = {
  args: {
    size: 'sm',
    children: 'Small',
  },
};

export const Medium: Story = {
  args: {
    size: 'md',
    children: 'Medium',
  },
};

export const Large: Story = {
  args: {
    size: 'lg',
    children: 'Large',
  },
};

// ── States ───────────────────────────────────────────────────────────────

export const Disabled: Story = {
  args: {
    disabled: true,
    children: 'Disabled',
  },
};

export const Loading: Story = {
  args: {
    loading: true,
    children: 'Saving...',
  },
};

export const WithIconLeft: Story = {
  args: {
    icon: <Settings className="w-4 h-4" />,
    iconPosition: 'left',
    children: 'Settings',
  },
};

export const WithIconRight: Story = {
  args: {
    icon: <Plus className="w-4 h-4" />,
    iconPosition: 'right',
    children: 'Add Device',
  },
};

export const FullWidth: Story = {
  args: {
    fullWidth: true,
    children: 'Full Width Button',
  },
};

export const IconOnly: StoryObj<typeof IconButton> = {
  render: () => (
    <IconButton
      icon={<Trash2 className="w-4 h-4" />}
      label="Delete item"
      variant="danger"
    />
  ),
};
