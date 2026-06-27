import type { Meta, StoryObj } from '@storybook/react';
import { TechnicianSelector, type Technician } from './TechnicianSelector';
import { useState } from 'react';

const sampleTechnicians: Technician[] = [
  { id: '1', name: 'Алексей Иванов', role: 'Старший техник', team: 'Team Alpha', workload: 45, avatarColor: '#6366f1' },
  { id: '2', name: 'Мария Петрова', role: 'Инженер', team: 'Team Alpha', workload: 72, avatarColor: '#ec4899' },
  { id: '3', name: 'Дмитрий Смирнов', role: 'Техник', team: 'Team Beta', workload: 90, avatarColor: '#f59e0b' },
  { id: '4', name: 'Елена Козлова', role: 'Старший инженер', team: 'Team Beta', workload: 30, avatarColor: '#10b981' },
  { id: '5', name: 'Сергей Попов', role: 'Техник', team: 'Team Gamma', workload: 60, avatarColor: '#8b5cf6' },
  { id: '6', name: 'Анна Новикова', role: 'Стажёр', team: 'Team Gamma', workload: 15, avatarColor: '#06b6d4' },
  { id: '7', name: 'Иван Морозов', role: 'Ведущий инженер', team: 'Team Alpha', workload: 85, avatarColor: '#ef4444' },
];

const meta: Meta<typeof TechnicianSelector> = {
  title: 'Molecules/TechnicianSelector',
  component: TechnicianSelector,
  tags: ['autodocs'],
  argTypes: {
    multi: { control: 'boolean' },
  },
};

export default meta;
type Story = StoryObj<typeof TechnicianSelector>;

// ── Multi Select ─────────────────────────────────────────────────────────

const MultiTemplate = () => {
  const [selected, setSelected] = useState<string[]>([]);
  return (
    <TechnicianSelector
      technicians={sampleTechnicians}
      selectedIds={selected}
      onChange={setSelected}
      multi
    />
  );
};

export const MultiSelect: Story = {
  render: () => <MultiTemplate />,
};

// ── Single Select ────────────────────────────────────────────────────────

const SingleTemplate = () => {
  const [selected, setSelected] = useState<string[]>([]);
  return (
    <TechnicianSelector
      technicians={sampleTechnicians}
      selectedIds={selected}
      onChange={setSelected}
      multi={false}
    />
  );
};

export const SingleSelect: Story = {
  render: () => <SingleTemplate />,
};

// ── Pre-Selected ─────────────────────────────────────────────────────────

const PreSelectedTemplate = () => {
  const [selected, setSelected] = useState<string[]>(['1', '3', '5']);
  return (
    <TechnicianSelector
      technicians={sampleTechnicians}
      selectedIds={selected}
      onChange={setSelected}
      multi
    />
  );
};

export const PreSelected: Story = {
  render: () => <PreSelectedTemplate />,
};

// ── With High Workload ───────────────────────────────────────────────────

const HighWorkloadTemplate = () => {
  const [selected, setSelected] = useState<string[]>([]);
  const busyTechs: Technician[] = [
    { id: '10', name: 'Overloaded Tech', role: 'Техник', team: 'Team Alpha', workload: 95, avatarColor: '#ef4444' },
    { id: '11', name: 'Busy Engineer', role: 'Инженер', team: 'Team Alpha', workload: 88, avatarColor: '#f59e0b' },
    { id: '12', name: 'Available Tech', role: 'Техник', team: 'Team Beta', workload: 25, avatarColor: '#10b981' },
  ];
  return (
    <TechnicianSelector
      technicians={busyTechs}
      selectedIds={selected}
      onChange={setSelected}
      multi
      placeholder="Assign technicians..."
    />
  );
};

export const HighWorkload: Story = {
  render: () => <HighWorkloadTemplate />,
};

// ── Empty State ──────────────────────────────────────────────────────────

const EmptyTemplate = () => {
  const [selected, setSelected] = useState<string[]>([]);
  return (
    <TechnicianSelector
      technicians={[]}
      selectedIds={selected}
      onChange={setSelected}
      placeholder="No technicians available"
    />
  );
};

export const Empty: Story = {
  render: () => <EmptyTemplate />,
};

// ── Custom Placeholder ───────────────────────────────────────────────────

const CustomPlaceholderTemplate = () => {
  const [selected, setSelected] = useState<string[]>([]);
  return (
    <TechnicianSelector
      technicians={sampleTechnicians}
      selectedIds={selected}
      onChange={setSelected}
      multi={false}
      placeholder="Select a technician to assign..."
    />
  );
};

export const CustomPlaceholder: Story = {
  render: () => <CustomPlaceholderTemplate />,
};

// ── Playground ───────────────────────────────────────────────────────────

const PlaygroundTemplate = () => {
  const [selected, setSelected] = useState<string[]>([]);
  return (
    <TechnicianSelector
      technicians={sampleTechnicians}
      selectedIds={selected}
      onChange={setSelected}
      multi
    />
  );
};

export const Playground: Story = {
  render: () => <PlaygroundTemplate />,
};
