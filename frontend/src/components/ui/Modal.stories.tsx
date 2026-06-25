import type { Meta, StoryObj } from '@storybook/react';
import { useState } from 'react';
import { Modal, ConfirmModal } from './Modal';
import { Button } from './Button';

const meta: Meta<typeof Modal> = {
  title: 'UI/Modal',
  component: Modal,
  tags: ['autodocs'],
  argTypes: {
    size: {
      control: 'select',
      options: ['sm', 'md', 'lg', 'xl', 'full'],
    },
    showClose: { control: 'boolean' },
  },
};

export default meta;
type Story = StoryObj<typeof Modal>;

// ── Sizes ────────────────────────────────────────────────────────────────

export const Small = () => {
  const [open, setOpen] = useState(true);
  return (
    <>
      <Button onClick={() => setOpen(true)}>Open Small Modal</Button>
      <Modal isOpen={open} onClose={() => setOpen(false)} title="Small Modal" size="sm">
        <p className="text-slate-600 dark:text-slate-300">
          This is a small modal. Perfect for confirmations and short messages.
        </p>
      </Modal>
    </>
  );
};

export const Medium = () => {
  const [open, setOpen] = useState(true);
  return (
    <>
      <Button onClick={() => setOpen(true)}>Open Medium Modal</Button>
      <Modal isOpen={open} onClose={() => setOpen(false)} title="Medium Modal" size="md">
        <p className="text-slate-600 dark:text-slate-300">
          This is a medium modal. Good for forms and detailed content.
        </p>
      </Modal>
    </>
  );
};

export const Large = () => {
  const [open, setOpen] = useState(true);
  return (
    <>
      <Button onClick={() => setOpen(true)}>Open Large Modal</Button>
      <Modal isOpen={open} onClose={() => setOpen(false)} title="Large Modal" size="lg">
        <div className="space-y-4">
          <p className="text-slate-600 dark:text-slate-300">
            Large modal with more content. Suitable for complex forms and data displays.
          </p>
          <div className="grid grid-cols-2 gap-4">
            {Array.from({ length: 4 }, (_, i) => (
              <div key={i} className="h-24 bg-slate-100 dark:bg-slate-700 rounded-lg flex items-center justify-center text-slate-400">
                Content Block {i + 1}
              </div>
            ))}
          </div>
        </div>
      </Modal>
    </>
  );
};

export const ExtraLarge = () => {
  const [open, setOpen] = useState(true);
  return (
    <>
      <Button onClick={() => setOpen(true)}>Open XL Modal</Button>
      <Modal isOpen={open} onClose={() => setOpen(false)} title="Extra Large Modal" size="xl">
        <div className="space-y-4">
          <p className="text-slate-600 dark:text-slate-300">
            Extra large modal for maximum content density.
          </p>
          <div className="grid grid-cols-3 gap-4">
            {Array.from({ length: 6 }, (_, i) => (
              <div key={i} className="h-32 bg-slate-100 dark:bg-slate-700 rounded-lg flex items-center justify-center text-slate-400">
                Block {i + 1}
              </div>
            ))}
          </div>
        </div>
      </Modal>
    </>
  );
};

// ── With Footer ──────────────────────────────────────────────────────────

export const WithFooter = () => {
  const [open, setOpen] = useState(true);
  return (
    <>
      <Button onClick={() => setOpen(true)}>Open Modal with Footer</Button>
      <Modal
        isOpen={open}
        onClose={() => setOpen(false)}
        title="Modal with Footer"
        size="md"
        footer={
          <div className="flex justify-end gap-3">
            <Button variant="outline" onClick={() => setOpen(false)}>Cancel</Button>
            <Button variant="primary" onClick={() => setOpen(false)}>Save Changes</Button>
          </div>
        }
      >
        <p className="text-slate-600 dark:text-slate-300">
          This modal has a footer with action buttons.
        </p>
      </Modal>
    </>
  );
};

// ── Without Close Button ─────────────────────────────────────────────────

export const WithoutCloseButton = () => {
  const [open, setOpen] = useState(true);
  return (
    <>
      <Button onClick={() => setOpen(true)}>Open Modal (No Close)</Button>
      <Modal isOpen={open} onClose={() => setOpen(false)} title="No Close Button" size="sm" showClose={false}>
        <p className="text-slate-600 dark:text-slate-300">
          This modal hides the close button. Users must use the footer action.
        </p>
      </Modal>
    </>
  );
};

// ── Without Title ────────────────────────────────────────────────────────

export const WithoutTitle = () => {
  const [open, setOpen] = useState(true);
  return (
    <>
      <Button onClick={() => setOpen(true)}>Open Modal (No Title)</Button>
      <Modal isOpen={open} onClose={() => setOpen(false)} size="sm">
        <p className="text-slate-600 dark:text-slate-300">
          A modal without a title bar, just content and close button.
        </p>
      </Modal>
    </>
  );
};

// ── ConfirmModal ─────────────────────────────────────────────────────────

export const ConfirmDanger: StoryObj = {
  render: () => {
    // eslint-disable-next-line react-hooks/rules-of-hooks
    const [open, setOpen] = useState(true);
    return (
      <>
        <Button variant="danger" onClick={() => setOpen(true)}>Delete Item</Button>
        <ConfirmModal
          isOpen={open}
          onClose={() => setOpen(false)}
          onConfirm={() => setOpen(false)}
          title="Delete Item"
          message="Are you sure you want to delete this item? This action cannot be undone."
          confirmText="Delete"
          cancelText="Cancel"
          variant="danger"
        />
      </>
    );
  },
};

export const ConfirmWarning: StoryObj = {
  render: () => {
    // eslint-disable-next-line react-hooks/rules-of-hooks
    const [open, setOpen] = useState(true);
    return (
      <>
        <Button variant="secondary" onClick={() => setOpen(true)}>Archive Item</Button>
        <ConfirmModal
          isOpen={open}
          onClose={() => setOpen(false)}
          onConfirm={() => setOpen(false)}
          title="Archive Item"
          message="This item will be archived and removed from active view."
          confirmText="Archive"
          cancelText="Cancel"
          variant="warning"
        />
      </>
    );
  },
};

// ── With Form ────────────────────────────────────────────────────────────

export const WithForm = () => {
  const [open, setOpen] = useState(true);
  return (
    <>
      <Button onClick={() => setOpen(true)}>Add Device</Button>
      <Modal
        isOpen={open}
        onClose={() => setOpen(false)}
        title="Add New Device"
        size="lg"
        footer={
          <div className="flex justify-end gap-3">
            <Button variant="outline" onClick={() => setOpen(false)}>Cancel</Button>
            <Button variant="primary" onClick={() => setOpen(false)}>Add Device</Button>
          </div>
        }
      >
        <div className="space-y-4">
          <div>
            <label className="block text-sm font-medium text-slate-700 dark:text-slate-300 mb-1">
              Device Name
            </label>
            <input
              type="text"
              className="w-full px-3 py-2 border border-slate-300 dark:border-slate-600 rounded-lg bg-white dark:bg-slate-800 text-slate-900 dark:text-white"
              placeholder="Enter device name"
            />
          </div>
          <div>
            <label className="block text-sm font-medium text-slate-700 dark:text-slate-300 mb-1">
              IP Address
            </label>
            <input
              type="text"
              className="w-full px-3 py-2 border border-slate-300 dark:border-slate-600 rounded-lg bg-white dark:bg-slate-800 text-slate-900 dark:text-white"
              placeholder="192.168.1.100"
            />
          </div>
          <div className="grid grid-cols-2 gap-4">
            <div>
              <label className="block text-sm font-medium text-slate-700 dark:text-slate-300 mb-1">
                Device Type
              </label>
              <select className="w-full px-3 py-2 border border-slate-300 dark:border-slate-600 rounded-lg bg-white dark:bg-slate-800 text-slate-900 dark:text-white">
                <option>Camera</option>
                <option>NVR</option>
                <option>Sensor</option>
              </select>
            </div>
            <div>
              <label className="block text-sm font-medium text-slate-700 dark:text-slate-300 mb-1">
                Site
              </label>
              <select className="w-full px-3 py-2 border border-slate-300 dark:border-slate-600 rounded-lg bg-white dark:bg-slate-800 text-slate-900 dark:text-white">
                <option>Main Office</option>
                <option>Branch Office</option>
              </select>
            </div>
          </div>
        </div>
      </Modal>
    </>
  );
};
