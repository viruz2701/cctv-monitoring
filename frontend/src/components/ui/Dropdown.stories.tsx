import type { Meta, StoryObj } from '@storybook/react';
import { Dropdown, type DropdownItem } from './Dropdown';
import { Button } from './Button';
import { Settings, User, LogOut, Download, Edit, Trash2, Copy, Share2, ChevronDown, MoreVertical } from 'lucide-react';

const meta: Meta<typeof Dropdown> = {
  title: 'UI/Dropdown',
  component: Dropdown,
  tags: ['autodocs'],
  argTypes: {
    placement: {
      control: 'select',
      options: ['bottom-start', 'bottom-end', 'top-start', 'top-end'],
    },
    disabled: { control: 'boolean' },
  },
};

export default meta;
type Story = StoryObj<typeof Dropdown>;

// ── Basic Items ──────────────────────────────────────────────────────────

const basicItems: DropdownItem[] = [
  { id: 'edit', label: 'Edit', icon: <Edit className="w-4 h-4" /> },
  { id: 'copy', label: 'Copy', icon: <Copy className="w-4 h-4" /> },
  { id: 'download', label: 'Download', icon: <Download className="w-4 h-4" /> },
];

export const Default: Story = {
  args: {
    items: basicItems,
    onSelect: (item: DropdownItem) => console.log('Selected:', item.label),
    anchor: <Button variant="outline">Actions <ChevronDown className="w-4 h-4" /></Button>,
  },
};

// ── With Dividers ────────────────────────────────────────────────────────

const itemsWithDividers: DropdownItem[] = [
  { id: 'edit', label: 'Edit', icon: <Edit className="w-4 h-4" /> },
  { id: 'copy', label: 'Copy', icon: <Copy className="w-4 h-4" /> },
  { id: 'divider-1', label: '', divider: true },
  { id: 'share', label: 'Share', icon: <Share2 className="w-4 h-4" /> },
  { id: 'download', label: 'Export', icon: <Download className="w-4 h-4" /> },
  { id: 'divider-2', label: '', divider: true },
  { id: 'delete', label: 'Delete', icon: <Trash2 className="w-4 h-4" />, danger: true },
];

export const WithDividers: Story = {
  args: {
    items: itemsWithDividers,
    onSelect: (item: DropdownItem) => console.log('Selected:', item.label),
    anchor: <Button variant="outline">More <ChevronDown className="w-4 h-4" /></Button>,
  },
};

// ── Disabled Items ───────────────────────────────────────────────────────

const itemsWithDisabled: DropdownItem[] = [
  { id: 'edit', label: 'Edit', icon: <Edit className="w-4 h-4" /> },
  { id: 'share', label: 'Share', icon: <Share2 className="w-4 h-4" /> },
  { id: 'download', label: 'Download', icon: <Download className="w-4 h-4" />, disabled: true },
  { id: 'delete', label: 'Delete', icon: <Trash2 className="w-4 h-4" />, danger: true, disabled: true },
];

export const DisabledItems: Story = {
  args: {
    items: itemsWithDisabled,
    onSelect: (item: DropdownItem) => console.log('Selected:', item.label),
    anchor: <Button variant="outline">Options <ChevronDown className="w-4 h-4" /></Button>,
  },
};

// ── Placements ───────────────────────────────────────────────────────────

export const BottomEnd: Story = {
  args: {
    items: basicItems,
    placement: 'bottom-end',
    onSelect: (item: DropdownItem) => console.log('Selected:', item.label),
    anchor: <Button variant="outline"><MoreVertical className="w-4 h-4" /></Button>,
  },
};

export const TopStart: Story = {
  args: {
    items: basicItems,
    placement: 'top-start',
    onSelect: (item: DropdownItem) => console.log('Selected:', item.label),
    anchor: <Button variant="outline">Actions <ChevronDown className="w-4 h-4" /></Button>,
  },
};

export const TopEnd: Story = {
  args: {
    items: basicItems,
    placement: 'top-end',
    onSelect: (item: DropdownItem) => console.log('Selected:', item.label),
    anchor: <Button variant="outline"><MoreVertical className="w-4 h-4" /></Button>,
  },
};

// ── Use Cases ────────────────────────────────────────────────────────────

const profileMenuItems: DropdownItem[] = [
  { id: 'profile', label: 'Profile', icon: <User className="w-4 h-4" /> },
  { id: 'settings', label: 'Settings', icon: <Settings className="w-4 h-4" /> },
  { id: 'divider-1', label: '', divider: true },
  { id: 'logout', label: 'Log Out', icon: <LogOut className="w-4 h-4" />, danger: true },
];

export const UserMenu: Story = {
  args: {
    items: profileMenuItems,
    onSelect: (item: DropdownItem) => console.log('Selected:', item.label),
    anchor: (
      <div className="flex items-center gap-2 px-3 py-2 rounded-lg hover:bg-slate-100 dark:hover:bg-slate-800 cursor-pointer">
        <div className="w-8 h-8 bg-blue-500 rounded-full flex items-center justify-center text-white text-sm font-medium">
          A
        </div>
        <span className="text-sm font-medium text-slate-700 dark:text-slate-300">Admin</span>
        <ChevronDown className="w-4 h-4 text-slate-400" />
      </div>
    ),
  },
};

// ── Danger Items ─────────────────────────────────────────────────────────

const dangerItems: DropdownItem[] = [
  { id: 'edit', label: 'Edit', icon: <Edit className="w-4 h-4" /> },
  { id: 'divider-1', label: '', divider: true },
  { id: 'delete', label: 'Delete', icon: <Trash2 className="w-4 h-4" />, danger: true },
  { id: 'clear', label: 'Clear Data', icon: <Trash2 className="w-4 h-4" />, danger: true },
];

export const WithDangerItems: Story = {
  args: {
    items: dangerItems,
    onSelect: (item: DropdownItem) => console.log('Selected:', item.label),
    anchor: <Button variant="danger">Danger Actions <ChevronDown className="w-4 h-4" /></Button>,
  },
};

// ── Disabled ─────────────────────────────────────────────────────────────

export const Disabled: Story = {
  args: {
    items: basicItems,
    disabled: true,
    onSelect: (item: DropdownItem) => console.log('Selected:', item.label),
    anchor: <Button variant="outline" disabled>Disabled Menu <ChevronDown className="w-4 h-4" /></Button>,
  },
};

// ── Playground ───────────────────────────────────────────────────────────

export const Playground: Story = {
  args: {
    items: itemsWithDividers,
    placement: 'bottom-start',
    disabled: false,
    onSelect: (item: DropdownItem) => console.log('Selected:', item.label),
    anchor: <Button variant="primary">Open Menu <ChevronDown className="w-4 h-4" /></Button>,
  },
};
