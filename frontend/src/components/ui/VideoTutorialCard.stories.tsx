import type { Meta, StoryObj } from '@storybook/react';
import { VideoTutorialCard } from './VideoTutorialCard';

const meta: Meta<typeof VideoTutorialCard> = {
  title: 'UI/VideoTutorialCard',
  component: VideoTutorialCard,
  tags: ['autodocs'],
};

export default meta;
type Story = StoryObj<typeof VideoTutorialCard>;

// ── With Thumbnail ───────────────────────────────────────────────────────

export const WithThumbnail: Story = {
  args: {
    video: {
      id: '1',
      title: 'Getting Started with CCTV Monitor',
      description: 'Learn how to set up your first site, add cameras, and configure recording schedules in under 10 minutes.',
      duration: '8:24',
      category: 'Getting Started',
      videoUrl: 'https://www.youtube.com/embed/dQw4w9WgXcQ',
      publishedAt: '2024-01-15',
    },
  },
};

// ── Without Thumbnail (Coming Soon) ──────────────────────────────────────

export const ComingSoon: Story = {
  args: {
    video: {
      id: '2',
      title: 'Advanced Analytics Configuration',
      description: 'Configure motion detection zones, line crossing, and object detection for your CCTV cameras.',
      duration: '12:30',
      category: 'Advanced',
      videoUrl: null,
    },
  },
};

// ── Long Title ───────────────────────────────────────────────────────────

export const LongTitle: Story = {
  args: {
    video: {
      id: '3',
      title: 'How to Configure Multi-Site Monitoring with Advanced Alert Routing and Escalation Policies',
      description: 'A comprehensive guide to setting up monitoring across multiple sites with automatic alert routing.',
      duration: '15:45',
      category: 'Tutorial',
      videoUrl: 'https://www.youtube.com/embed/dQw4w9WgXcQ',
    },
  },
};

// ── Short Description ────────────────────────────────────────────────────

export const ShortDescription: Story = {
  args: {
    video: {
      id: '4',
      title: 'Quick Tip: NVR Reset',
      description: 'How to reset your NVR to factory settings.',
      duration: '1:30',
      category: 'Quick Tip',
      videoUrl: 'https://www.youtube.com/embed/dQw4w9WgXcQ',
    },
  },
};

// ── Different Category ───────────────────────────────────────────────────

export const MaintenanceGuide: Story = {
  args: {
    video: {
      id: '5',
      title: 'Preventive Maintenance Guide',
      description: 'Monthly preventive maintenance checklist for all CCTV equipment including cleaning, firmware updates, and storage checks.',
      duration: '22:15',
      category: 'Maintenance',
      videoUrl: 'https://www.youtube.com/embed/dQw4w9WgXcQ',
    },
  },
};

// ── With Published Date ──────────────────────────────────────────────────

export const WithPublishedDate: Story = {
  args: {
    video: {
      id: '6',
      title: 'Firmware Update Guide v3.2',
      description: 'Step-by-step guide for updating device firmware to version 3.2 with new security features.',
      duration: '6:50',
      category: 'Updates',
      videoUrl: 'https://www.youtube.com/embed/dQw4w9WgXcQ',
      publishedAt: '2024-06-01',
    },
  },
};

// ── Playground ───────────────────────────────────────────────────────────

export const Playground: Story = {
  args: {
    video: {
      id: '7',
      title: 'Custom Tutorial Video',
      description: 'This is a customizable tutorial card for demonstration purposes.',
      duration: '5:00',
      category: 'Demo',
      videoUrl: 'https://www.youtube.com/embed/dQw4w9WgXcQ',
    },
  },
};
