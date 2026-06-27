import type { Meta, StoryObj } from '@storybook/react';
import { Input, SearchInput, Select, Textarea } from './Input';

const meta: Meta<typeof Input> = {
  title: 'UI/Input',
  component: Input,
  tags: ['autodocs'],
  argTypes: {
    type: { control: 'select', options: ['text', 'email', 'password', 'number', 'date', 'time'] },
    placeholder: { control: 'text' },
    disabled: { control: 'boolean' },
    error: { control: 'text' },
    helperText: { control: 'text' },
    label: { control: 'text' },
  },
};

export default meta;
type Story = StoryObj<typeof Input>;

// ── Input Variants ───────────────────────────────────────────────────────

export const Default: Story = {
  args: {
    placeholder: 'Enter text...',
  },
};

export const WithLabel: Story = {
  args: {
    label: 'Device Name',
    placeholder: 'e.g. NVR-01',
  },
};

export const WithError: Story = {
  args: {
    label: 'Email',
    type: 'email',
    value: 'invalid-email',
    error: 'Please enter a valid email address',
  },
};

export const WithHelperText: Story = {
  args: {
    label: 'IP Address',
    placeholder: '192.168.1.100',
    helperText: 'Enter the IPv4 address of the device',
  },
};

export const Disabled: Story = {
  args: {
    label: 'System ID',
    value: 'SYS-001-A',
    disabled: true,
  },
};

export const Password: Story = {
  args: {
    label: 'Password',
    type: 'password',
    placeholder: 'Enter password',
    value: 'secret123',
  },
};

export const DateInput: Story = {
  args: {
    label: 'Installation Date',
    type: 'date',
  },
};

// ── SearchInput ──────────────────────────────────────────────────────────

const searchMeta: Meta<typeof SearchInput> = {
  title: 'UI/Input/SearchInput',
  component: SearchInput,
  tags: ['autodocs'],
};

export const SearchDefault: StoryObj<typeof SearchInput> = {
  render: () => <SearchInput placeholder="Search devices..." />,
};

export const SearchWithValue: StoryObj<typeof SearchInput> = {
  render: () => <SearchInput placeholder="Search..." value="NVR-01" onChange={() => {}} />,
};

// ── Select ───────────────────────────────────────────────────────────────

const selectMeta: Meta<typeof Select> = {
  title: 'UI/Input/Select',
  component: Select,
  tags: ['autodocs'],
};

export const SelectDefault: StoryObj<typeof Select> = {
  render: () => (
    <Select
      label="Device Type"
      options={[
        { value: '', label: 'Select type...' },
        { value: 'camera', label: 'Camera' },
        { value: 'nvr', label: 'NVR' },
        { value: 'sensor', label: 'Sensor' },
      ]}
    />
  ),
};

export const SelectWithError: StoryObj<typeof Select> = {
  render: () => (
    <Select
      label="Priority"
      error="Please select a priority level"
      options={[
        { value: '', label: 'Select priority...' },
        { value: 'low', label: 'Low' },
        { value: 'medium', label: 'Medium' },
        { value: 'high', label: 'High' },
      ]}
    />
  ),
};

export const SelectDisabled: StoryObj<typeof Select> = {
  render: () => (
    <Select
      label="Region"
      disabled
      value="emea"
      options={[{ value: 'emea', label: 'EMEA' }]}
    />
  ),
};

// ── Textarea ─────────────────────────────────────────────────────────────

const textareaMeta: Meta<typeof Textarea> = {
  title: 'UI/Input/Textarea',
  component: Textarea,
  tags: ['autodocs'],
};

export const TextareaDefault: StoryObj<typeof Textarea> = {
  render: () => (
    <Textarea
      label="Notes"
      placeholder="Enter maintenance notes..."
    />
  ),
};

export const TextareaWithError: StoryObj<typeof Textarea> = {
  render: () => (
    <Textarea
      label="Description"
      value="Too short"
      error="Description must be at least 50 characters"
    />
  ),
};

// ── Playground ───────────────────────────────────────────────────────────

export const Playground: Story = {
  args: {
    label: 'Custom Input',
    placeholder: 'Type something...',
    helperText: 'This is a playground input',
  },
};
