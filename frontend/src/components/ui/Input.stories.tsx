import type { Meta, StoryObj } from '@storybook/react';
import { Input, SearchInput, Select, Textarea } from './Input';

const meta: Meta<typeof Input> = {
  title: 'UI/Input',
  component: Input,
  tags: ['autodocs'],
  argTypes: {
    type: {
      control: 'select',
      options: ['text', 'email', 'password', 'number', 'tel', 'url'],
    },
    placeholder: { control: 'text' },
    disabled: { control: 'boolean' },
    required: { control: 'boolean' },
  },
};

export default meta;
type Story = StoryObj<typeof Input>;

// ── Input Variants ────────────────────────────────────────────────────────

export const Default: Story = {
  args: { placeholder: 'Enter text...' },
};

export const WithLabel: Story = {
  args: { label: 'Device Name', placeholder: 'Enter device name' },
};

export const WithValue: Story = {
  args: { label: 'IP Address', value: '192.168.1.100', readOnly: true },
};

export const WithError: Story = {
  args: { label: 'Email', type: 'email', value: 'invalid', error: 'Please enter a valid email address' },
};

export const WithHelperText: Story = {
  args: { label: 'Password', type: 'password', helperText: 'Must be at least 8 characters' },
};

export const Disabled: Story = {
  args: { label: 'Disabled Field', value: 'Cannot edit', disabled: true },
};

export const Required: Story = {
  args: { label: 'Required Field', required: true, placeholder: 'This is required' },
};

export const Password: Story = {
  args: { label: 'Password', type: 'password', placeholder: 'Enter password' },
};

export const Number: Story = {
  args: { label: 'Port', type: 'number', placeholder: '8080' },
};

// ── SearchInput ───────────────────────────────────────────────────────────

export const SearchDefault: StoryObj<typeof SearchInput> = {
  render: () => <SearchInput placeholder="Search devices..." />,
};

export const SearchWithValue: StoryObj<typeof SearchInput> = {
  render: () => <SearchInput placeholder="Search..." value="NVR-01" onChange={() => {}} />,
};

// ── Select ────────────────────────────────────────────────────────────────

const sampleOptions = [
  { value: '', label: 'Select an option...' },
  { value: 'camera', label: 'Camera' },
  { value: 'nvr', label: 'NVR' },
  { value: 'sensor', label: 'Sensor' },
  { value: 'gateway', label: 'Gateway' },
];

export const SelectDefault: StoryObj<typeof Select> = {
  render: () => <Select label="Device Type" options={sampleOptions} />,
};

export const SelectWithValue: StoryObj<typeof Select> = {
  render: () => <Select label="Device Type" options={sampleOptions} value="camera" />,
};

export const SelectWithError: StoryObj<typeof Select> = {
  render: () => (
    <Select
      label="Device Type"
      options={sampleOptions}
      value=""
      error="Please select a device type"
    />
  ),
};

export const SelectDisabled: StoryObj<typeof Select> = {
  render: () => <Select label="Device Type" options={sampleOptions} disabled />,
};

// ── Textarea ──────────────────────────────────────────────────────────────

export const TextareaDefault: StoryObj<typeof Textarea> = {
  render: () => <Textarea label="Description" placeholder="Enter description..." />,
};

export const TextareaWithValue: StoryObj<typeof Textarea> = {
  render: () => (
    <Textarea
      label="Notes"
      value="This is a sample text in the textarea component for demonstration purposes."
    />
  ),
};

export const TextareaWithError: StoryObj<typeof Textarea> = {
  render: () => <Textarea label="Notes" value="" error="This field is required" />,
};

// ── All Inputs Showcase ───────────────────────────────────────────────────

export const AllInputs: StoryObj = {
  render: () => (
    <div className="flex flex-col gap-6 p-4 max-w-sm">
      <Input label="Standard Input" placeholder="Enter text..." />
      <Input label="With Error" value="bad data" error="Invalid value" />
      <Input label="Disabled" value="Read only" disabled />
      <SearchInput placeholder="Search..." />
      <Select label="Select Option" options={sampleOptions} />
      <Textarea label="Text Area" placeholder="Enter notes..." />
    </div>
  ),
};

// ── Playground ────────────────────────────────────────────────────────────

export const Playground: Story = {
  args: {
    label: 'Custom Input',
    placeholder: 'Type something...',
    disabled: false,
    required: false,
    helperText: 'This is a helper text',
  },
};
