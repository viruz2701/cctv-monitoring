import type { Meta, StoryObj } from '@storybook/react';
import { FileUpload } from './FileUpload';

const meta: Meta<typeof FileUpload> = {
  title: 'UI/FileUpload',
  component: FileUpload,
  tags: ['autodocs'],
  argTypes: {
    multiple: { control: 'boolean' },
    disabled: { control: 'boolean' },
    maxFiles: { control: { type: 'number', min: 1, max: 50 } },
    maxSizeMB: { control: { type: 'number', min: 1, max: 500 } },
  },
};

export default meta;
type Story = StoryObj<typeof FileUpload>;

const handleUpload = async (files: File[]) => {
  await new Promise((resolve) => setTimeout(resolve, 1000));
  alert(`Uploaded ${files.length} file(s): ${files.map(f => f.name).join(', ')}`);
};

export const SingleFile: Story = {
  args: {
    onUpload: handleUpload,
    multiple: false,
    maxFiles: 1,
    label: 'Upload single image',
  },
};

export const MultipleFiles: Story = {
  args: {
    onUpload: handleUpload,
    multiple: true,
    maxFiles: 5,
    label: 'Upload multiple files',
  },
};

export const WithPreview: Story = {
  args: {
    onUpload: handleUpload,
    multiple: true,
    maxFiles: 3,
    label: 'Drag & drop or click to upload',
    hint: 'Supports JPG, PNG, PDF up to 10MB each',
  },
};

export const DragDrop: Story = {
  args: {
    onUpload: handleUpload,
    multiple: true,
    maxFiles: 10,
    label: 'Drop files anywhere to upload',
    hint: 'Firmware images and diagnostic logs',
  },
};

export const ErrorState: Story = {
  args: {
    onUpload: handleUpload,
    multiple: false,
    maxFiles: 1,
    maxSizeMB: 1,
    label: 'Upload (max 1MB)',
    hint: 'Try uploading a large file to see validation error',
  },
};

export const Disabled: Story = {
  args: {
    onUpload: handleUpload,
    disabled: true,
    label: 'Upload disabled',
  },
};

export const LimitedFiles: Story = {
  args: {
    onUpload: handleUpload,
    multiple: true,
    maxFiles: 2,
    maxSizeMB: 5,
    label: 'Upload up to 2 files',
    hint: 'Only 2 firmware files allowed',
  },
};

export const Playground: Story = {
  args: {
    onUpload: handleUpload,
    multiple: true,
    maxFiles: 10,
    maxSizeMB: 50,
    label: 'Custom upload zone',
  },
};
