import type { Meta, StoryObj } from '@storybook/react';
import { PriorityPicker, PriorityBadge, type Priority } from './PriorityPicker';
import { useState } from 'react';

const meta: Meta<typeof PriorityPicker> = {
  title: 'Molecules/PriorityPicker',
  component: PriorityPicker,
  tags: ['autodocs'],
  argTypes: {
    readOnly: { control: 'boolean' },
    showLabels: { control: 'boolean' },
    lang: { control: 'select', options: ['ru', 'en'] },
  },
};

export default meta;
type Story = StoryObj<typeof PriorityPicker>;

// ── All Priorities Selected ──────────────────────────────────────────────

const DefaultTemplate = () => {
  const [priority, setPriority] = useState<Priority>('medium');
  return <PriorityPicker value={priority} onChange={setPriority} />;
};

export const Default: Story = {
  render: () => <DefaultTemplate />,
};

// ── Critical ─────────────────────────────────────────────────────────────

const CriticalTemplate = () => {
  const [priority, setPriority] = useState<Priority>('critical');
  return <PriorityPicker value={priority} onChange={setPriority} />;
};

export const Critical: Story = {
  render: () => <CriticalTemplate />,
};

// ── High ─────────────────────────────────────────────────────────────────

const HighTemplate = () => {
  const [priority, setPriority] = useState<Priority>('high');
  return <PriorityPicker value={priority} onChange={setPriority} />;
};

export const High: Story = {
  render: () => <HighTemplate />,
};

// ── Medium ───────────────────────────────────────────────────────────────

const MediumTemplate = () => {
  const [priority, setPriority] = useState<Priority>('medium');
  return <PriorityPicker value={priority} onChange={setPriority} />;
};

export const Medium: Story = {
  render: () => <MediumTemplate />,
};

// ── Low ──────────────────────────────────────────────────────────────────

const LowTemplate = () => {
  const [priority, setPriority] = useState<Priority>('low');
  return <PriorityPicker value={priority} onChange={setPriority} />;
};

export const Low: Story = {
  render: () => <LowTemplate />,
};

// ── Read Only ────────────────────────────────────────────────────────────

export const ReadOnly: Story = {
  args: {
    value: 'high',
    readOnly: true,
    onChange: () => {},
  },
};

// ── English Labels ───────────────────────────────────────────────────────

const EnglishTemplate = () => {
  const [priority, setPriority] = useState<Priority>('medium');
  return <PriorityPicker value={priority} onChange={setPriority} lang="en" />;
};

export const English: Story = {
  render: () => <EnglishTemplate />,
};

// ── Without Labels (Dots Only) ───────────────────────────────────────────

const DotsOnlyTemplate = () => {
  const [priority, setPriority] = useState<Priority>('high');
  return <PriorityPicker value={priority} onChange={setPriority} showLabels={false} />;
};

export const DotsOnly: Story = {
  render: () => <DotsOnlyTemplate />,
};

// ── PriorityBadge ────────────────────────────────────────────────────────

const badgeMeta: Meta<typeof PriorityBadge> = {
  title: 'Molecules/PriorityPicker/PriorityBadge',
  component: PriorityBadge,
  tags: ['autodocs'],
};

export const BadgeCritical: StoryObj<typeof PriorityBadge> = {
  render: () => <PriorityBadge priority="critical" />,
};

export const BadgeHigh: StoryObj<typeof PriorityBadge> = {
  render: () => <PriorityBadge priority="high" />,
};

export const BadgeMedium: StoryObj<typeof PriorityBadge> = {
  render: () => <PriorityBadge priority="medium" />,
};

export const BadgeLow: StoryObj<typeof PriorityBadge> = {
  render: () => <PriorityBadge priority="low" />,
};

export const BadgeSmall: StoryObj<typeof PriorityBadge> = {
  render: () => <PriorityBadge priority="critical" size="sm" />,
};

export const BadgeEnglish: StoryObj<typeof PriorityBadge> = {
  render: () => <PriorityBadge priority="high" lang="en" />,
};

// ── Playground ───────────────────────────────────────────────────────────

const PlaygroundTemplate = () => {
  const [priority, setPriority] = useState<Priority>('medium');
  return (
    <div className="space-y-4">
      <PriorityPicker value={priority} onChange={setPriority} />
      <PriorityBadge priority={priority} />
    </div>
  );
};

export const Playground: Story = {
  render: () => <PlaygroundTemplate />,
};
