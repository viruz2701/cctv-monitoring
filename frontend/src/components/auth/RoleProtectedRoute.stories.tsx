import type { Meta, StoryObj } from '@storybook/react';
import { MemoryRouter } from 'react-router-dom';
import { RoleProtectedRoute } from './RoleProtectedRoute';

const meta: Meta<typeof RoleProtectedRoute> = {
  title: 'Auth/RoleProtectedRoute',
  component: RoleProtectedRoute,
  tags: ['autodocs'],
  decorators: [
    (Story) => (
      <MemoryRouter initialEntries={['/']}>
        <Story />
      </MemoryRouter>
    ),
  ],
};

export default meta;
type Story = StoryObj<typeof RoleProtectedRoute>;

export const AdminOnly: Story = {
  args: {
    allowedRoles: ['admin'],
  },
};

export const TechnicianOrAdmin: Story = {
  args: {
    allowedRoles: ['technician', 'admin'],
  },
};
