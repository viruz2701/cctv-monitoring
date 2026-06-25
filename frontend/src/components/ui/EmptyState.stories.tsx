import type { Meta, StoryObj } from '@storybook/react';
import { EmptyState } from './EmptyState';
import { HardDrive, Search, Camera, Wifi, AlertTriangle } from 'lucide-react';

const meta: Meta<typeof EmptyState> = {
  title: 'UI/EmptyState',
  component: EmptyState,
  tags: ['autodocs'],
  argTypes: {
    size: {
      control: 'select',
      options: ['sm', 'md', 'lg'],
    },
    icon: { control: false },
    action: { control: false },
    secondaryAction: { control: false },
  },
};

export default meta;
type Story = StoryObj<typeof EmptyState>;

// ── Sizes ────────────────────────────────────────────────────────────────

export const Small: Story = {
  args: {
    size: 'sm',
    icon: <HardDrive className="w-8 h-8" />,
    title: 'No devices found',
    description: 'This site has no devices configured yet.',
  },
};

export const Medium: Story = {
  args: {
    size: 'md',
    icon: <Search className="w-12 h-12" />,
    title: 'No results found',
    description: 'Try adjusting your search or filter criteria.',
    hint: 'You can search by device name, IP address, or serial number.',
  },
};

export const Large: Story = {
  args: {
    size: 'lg',
    icon: <Camera className="w-16 h-16" />,
    title: 'No cameras connected',
    description: 'Get started by adding your first camera to begin monitoring.',
    hint: 'Supports RTSP, ONVIF, and GB/T 28181 protocols.',
  },
};

// ── With Actions ─────────────────────────────────────────────────────────

export const WithPrimaryAction: Story = {
  args: {
    size: 'md',
    icon: <HardDrive className="w-12 h-12" />,
    title: 'No devices registered',
    description: 'Add your first CCTV device to start monitoring.',
    action: { label: 'Add Device', onClick: () => {} },
  },
};

export const WithBothActions: Story = {
  args: {
    size: 'md',
    icon: <Camera className="w-12 h-12" />,
    title: 'No cameras configured',
    description: 'Configure your first camera to start monitoring.',
    action: { label: 'Add Camera', onClick: () => {} },
    secondaryAction: { label: 'Learn more', onClick: () => {} },
  },
};

export const WithHintOnly: Story = {
  args: {
    size: 'md',
    icon: <Wifi className="w-12 h-12" />,
    title: 'No network devices',
    description: 'Network devices will appear here once discovered.',
    hint: 'Devices are auto-discovered on the local subnet every 5 minutes.',
  },
};

// ── Use Case: Alerts ─────────────────────────────────────────────────────

export const NoAlerts: Story = {
  args: {
    size: 'md',
    icon: <AlertTriangle className="w-12 h-12" />,
    title: 'All clear',
    description: 'No active alerts. Your system is running normally.',
  },
};

// ── Use Case: Search ─────────────────────────────────────────────────────

export const NoSearchResults: Story = {
  args: {
    size: 'md',
    icon: <Search className="w-12 h-12" />,
    title: 'No matching devices',
    description: 'No devices match your current search.',
    action: { label: 'Clear Filters', onClick: () => {} },
  },
};

// ── Playground ───────────────────────────────────────────────────────────

export const Playground: Story = {
  args: {
    size: 'md',
    icon: <HardDrive className="w-12 h-12" />,
    title: 'Empty State Title',
    description: 'Description of the empty state goes here.',
    hint: 'Optional hint text provides additional context.',
    action: { label: 'Action', onClick: () => {} },
    secondaryAction: { label: 'Secondary', onClick: () => {} },
  },
};
