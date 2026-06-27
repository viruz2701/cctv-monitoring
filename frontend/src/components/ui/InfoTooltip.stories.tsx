import type { Meta, StoryObj } from '@storybook/react';
import { InfoTooltip } from './InfoTooltip';
import { BrowserRouter } from 'react-router-dom';

const meta: Meta<typeof InfoTooltip> = {
  title: 'UI/InfoTooltip',
  component: InfoTooltip,
  tags: ['autodocs'],
  decorators: [
    (Story) => (
      <BrowserRouter>
        <div className="flex items-center justify-center min-h-[200px]">
          <Story />
        </div>
      </BrowserRouter>
    ),
  ],
  argTypes: {
    position: {
      control: 'select',
      options: ['top', 'bottom', 'left', 'right'],
    },
    delay: { control: { type: 'number', min: 0, max: 1000 } },
    iconSize: { control: { type: 'number', min: 10, max: 32 } },
  },
};

export default meta;
type Story = StoryObj<typeof InfoTooltip>;

// ── Positions ────────────────────────────────────────────────────────────

export const Top: Story = {
  args: {
    text: 'This tooltip appears above the icon',
    position: 'top',
  },
};

export const Bottom: Story = {
  args: {
    text: 'This tooltip appears below the icon',
    position: 'bottom',
  },
};

export const Left: Story = {
  args: {
    text: 'This tooltip appears to the left',
    position: 'left',
  },
};

export const Right: Story = {
  args: {
    text: 'This tooltip appears to the right',
    position: 'right',
  },
};

// ── With Glossary Link ───────────────────────────────────────────────────

export const WithGlossaryLink: Story = {
  args: {
    text: 'SLA (Service Level Agreement) defines expected response times for maintenance requests',
    glossaryTerm: 'SLA',
    position: 'bottom',
  },
};

// ── With Icon Button ─────────────────────────────────────────────────────

export const WithIconButton: Story = {
  args: {
    text: 'Click the icon to learn more about compliance requirements',
    position: 'right',
    iconSize: 18,
  },
};

// ── Long Text ────────────────────────────────────────────────────────────

export const LongText: Story = {
  args: {
    text: 'This component provides contextual information about complex terms and compliance requirements as defined by IEC 62443 and ISO 27001 standards for industrial security.',
    position: 'top',
  },
};

// ── Custom Delay ─────────────────────────────────────────────────────────

export const CustomDelay: Story = {
  args: {
    text: 'This tooltip appears after 1 second delay',
    delay: 1000,
    position: 'bottom',
  },
};

// ── Playground ───────────────────────────────────────────────────────────

export const Playground: Story = {
  args: {
    text: 'Custom tooltip text with adjustable position',
    position: 'top',
    iconSize: 14,
    delay: 200,
  },
};
