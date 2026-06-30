import type { Meta, StoryObj } from '@storybook/react';
import { PermissionGuard } from './PermissionGuard';

const meta: Meta<typeof PermissionGuard> = {
  title: 'Auth/PermissionGuard',
  component: PermissionGuard,
  tags: ['autodocs'],
};

export default meta;
type Story = StoryObj<typeof PermissionGuard>;

export const AdminAccess: Story = {
  args: {
    requiredRole: 'admin',
    children: <div className="p-4 bg-green-100 rounded">Admin Content Visible</div>,
  },
};

export const NoAccess: Story = {
  args: {
    requiredRole: 'admin',
    fallback: <div className="p-4 bg-red-100 rounded">Access Denied</div>,
    children: <div className="p-4 bg-green-100 rounded">Admin Content</div>,
  },
};

export const WithManageTickets: Story = {
  args: {
    requireManageTickets: true,
    children: <div className="p-4 bg-green-100 rounded">Ticket Management Visible</div>,
  },
};
