import type { Meta, StoryObj } from '@storybook/react';
import { LazyImage } from './LazyImage';

const meta: Meta<typeof LazyImage> = {
  title: 'UI/LazyImage',
  component: LazyImage,
  tags: ['autodocs'],
  argTypes: {
    showSkeleton: { control: 'boolean' },
    aspectRatio: { control: 'text' },
    placeholderSize: { control: 'select', options: ['sm', 'md', 'lg'] },
  },
};

export default meta;
type Story = StoryObj<typeof LazyImage>;

// ── Loaded ───────────────────────────────────────────────────────────────

export const Loaded: Story = {
  args: {
    src: 'https://images.unsplash.com/photo-1558002038-1055907df827?w=400&h=300&fit=crop',
    alt: 'Security camera',
    className: 'w-64 h-48 rounded-lg',
    showSkeleton: true,
  },
};

// ── Loading State ────────────────────────────────────────────────────────

export const Loading: Story = {
  args: {
    src: 'https://slow-server.com/image.jpg',
    alt: 'Loading image',
    className: 'w-64 h-48 rounded-lg',
    showSkeleton: true,
  },
};

// ── Error State ──────────────────────────────────────────────────────────

export const ErrorState: Story = {
  args: {
    src: 'https://invalid-url.com/missing.jpg',
    alt: 'Broken image',
    className: 'w-64 h-48 rounded-lg',
  },
};

// ── With Aspect Ratio ────────────────────────────────────────────────────

export const WithAspectRatio: Story = {
  args: {
    src: 'https://images.unsplash.com/photo-1558002038-1055907df827?w=800&h=600&fit=crop',
    alt: 'Camera with aspect ratio',
    aspectRatio: '16/9',
    className: 'w-full max-w-md rounded-lg',
    showSkeleton: true,
  },
};

// ── Square Aspect ────────────────────────────────────────────────────────

export const Square: Story = {
  args: {
    src: 'https://images.unsplash.com/photo-1558002038-1055907df827?w=400&h=400&fit=crop',
    alt: 'Square image',
    aspectRatio: '1/1',
    className: 'w-32 rounded-lg',
    showSkeleton: true,
  },
};

// ── Small Thumbnail ──────────────────────────────────────────────────────

export const SmallThumbnail: Story = {
  args: {
    src: 'https://images.unsplash.com/photo-1558002038-1055907df827?w=100&h=100&fit=crop',
    alt: 'Small thumbnail',
    className: 'w-16 h-16 rounded-lg',
    placeholderSize: 'sm',
  },
};

// ── Without Skeleton ─────────────────────────────────────────────────────

export const WithoutSkeleton: Story = {
  args: {
    src: 'https://images.unsplash.com/photo-1558002038-1055907df827?w=400&h=300&fit=crop',
    alt: 'No skeleton',
    className: 'w-64 h-48 rounded-lg',
    showSkeleton: false,
  },
};

// ── Playground ───────────────────────────────────────────────────────────

export const Playground: Story = {
  args: {
    src: 'https://images.unsplash.com/photo-1558002038-1055907df827?w=400&h=300&fit=crop',
    alt: 'Playground image',
    className: 'w-64 h-48 rounded-lg',
    showSkeleton: true,
    aspectRatio: undefined,
  },
};
