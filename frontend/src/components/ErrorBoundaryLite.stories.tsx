import type { Meta, StoryObj } from '@storybook/react';
import { ErrorBoundaryLite } from './ErrorBoundaryLite';

const meta: Meta<typeof ErrorBoundaryLite> = {
  title: 'UI/ErrorBoundaryLite',
  component: ErrorBoundaryLite,
  tags: ['autodocs'],
};

export default meta;
type Story = StoryObj<typeof ErrorBoundaryLite>;

const GoodComponent = () => <div className="p-4 bg-green-100 rounded">Content loads fine</div>;

const BuggyComponent = () => {
  throw new Error('Test crash');
  return null;
};

export const Normal: Story = {
  args: {
    children: <GoodComponent />,
  },
};

export const WithError: Story = {
  args: {
    children: <BuggyComponent />,
  },
};
