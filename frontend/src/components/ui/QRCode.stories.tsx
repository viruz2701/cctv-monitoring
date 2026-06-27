import type { Meta, StoryObj } from '@storybook/react';
import { QRCode } from './QRCode';

const meta: Meta<typeof QRCode> = {
  title: 'UI/QRCode',
  component: QRCode,
  tags: ['autodocs'],
  argTypes: {
    size: { control: { type: 'number', min: 50, max: 400 } },
    bgColor: { control: 'color' },
    fgColor: { control: 'color' },
  },
};

export default meta;
type Story = StoryObj<typeof QRCode>;

// ── Default ──────────────────────────────────────────────────────────────

export const Default: Story = {
  args: {
    value: 'CCTV-DEVICE-NVR-01-2024',
    label: 'Scan to register device',
  },
};

// ── Custom Size ──────────────────────────────────────────────────────────

export const CustomSize: Story = {
  args: {
    value: 'SITE-MAIN-OFFICE',
    size: 300,
    label: 'Site QR Code',
  },
};

// ── Small ────────────────────────────────────────────────────────────────

export const Small: Story = {
  args: {
    value: 'SMALL-QR',
    size: 100,
  },
};

// ── With Label ───────────────────────────────────────────────────────────

export const WithLabel: Story = {
  args: {
    value: 'DEVICE-CAM-101',
    size: 200,
    label: 'Camera CAM-101',
  },
};

// ── Different Colors ─────────────────────────────────────────────────────

export const CustomColors: Story = {
  args: {
    value: 'COLORFUL-QR',
    size: 200,
    bgColor: '#f0fdf4',
    fgColor: '#166534',
    label: 'Custom colors',
  },
};

// ── Long Value ───────────────────────────────────────────────────────────

export const LongValue: Story = {
  args: {
    value: 'urn:uuid:550e8400-e29b-41d4-a716-446655440000:site:main-office:device:nvr-01',
    size: 200,
    label: 'Device UUID',
  },
};

// ── URL Value ────────────────────────────────────────────────────────────

export const UrlValue: Story = {
  args: {
    value: 'https://cctv.example.com/devices/register?token=abc123',
    size: 200,
    label: 'Registration URL',
  },
};

// ── Playground ───────────────────────────────────────────────────────────

export const Playground: Story = {
  args: {
    value: 'PLAYGROUND-QR-DEMO',
    size: 200,
    label: 'QR Code Demo',
    bgColor: '#ffffff',
    fgColor: '#000000',
  },
};
