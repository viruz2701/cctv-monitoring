import type { Meta, StoryObj } from '@storybook/react';
import { Timeline, type TimelineEvent } from './Timeline';

const meta: Meta<typeof Timeline> = {
  title: 'UI/Timeline',
  component: Timeline,
  tags: ['autodocs'],
  argTypes: {
    maxItems: { control: { type: 'number', min: 0, max: 20 } },
  },
};

export default meta;
type Story = StoryObj<typeof Timeline>;

const now = new Date();

const sampleEvents: TimelineEvent[] = [
  {
    id: '1',
    timestamp: new Date(now.getTime() - 1000 * 60 * 5).toISOString(),
    type: 'status_change',
    title: 'Статус изменён на Online',
    description: 'Устройство NVR-01 восстановило соединение',
    user: 'System',
  },
  {
    id: '2',
    timestamp: new Date(now.getTime() - 1000 * 60 * 30).toISOString(),
    type: 'maintenance',
    title: 'Плановое ТО',
    description: 'Замена термопасты и очистка фильтров',
    user: 'Алексей Иванов',
    diff: [
      { field: 'Температура CPU', oldValue: '78°C', newValue: '52°C' },
      { field: 'Скорость вентилятора', oldValue: '3200 RPM', newValue: '2100 RPM' },
    ],
  },
  {
    id: '3',
    timestamp: new Date(now.getTime() - 1000 * 60 * 120).toISOString(),
    type: 'part',
    title: 'Замена жёсткого диска',
    description: 'HDD заменён на SSD 1TB',
    user: 'Сергей Петров',
    details: <p>Установлен Samsung 870 EVO 1TB. Старый HDD (Seagate 500GB) отправлен на утилизацию.</p>,
  },
  {
    id: '4',
    timestamp: new Date(now.getTime() - 1000 * 60 * 240).toISOString(),
    type: 'assignment',
    title: 'Назначен ответственный',
    user: 'Дмитрий Смирнов',
  },
  {
    id: '5',
    timestamp: new Date(now.getTime() - 1000 * 60 * 480).toISOString(),
    type: 'note',
    title: 'Добавлена заметка',
    description: 'Рекомендуется обновить прошивку до версии 3.2.1',
    user: 'Елена Козлова',
  },
  {
    id: '6',
    timestamp: new Date(now.getTime() - 1000 * 60 * 1440).toISOString(),
    type: 'system',
    title: 'Автоматическая диагностика',
    description: 'Все системы работают в штатном режиме',
  },
  {
    id: '7',
    timestamp: new Date(now.getTime() - 1000 * 60 * 2880).toISOString(),
    type: 'photo',
    title: 'Добавлено фото повреждения',
    user: 'Иван Петров',
  },
];

// ── Various Events ───────────────────────────────────────────────────────

export const VariousEvents: Story = {
  args: {
    events: sampleEvents,
  },
};

// ── With Diff Entries ────────────────────────────────────────────────────

export const WithDiffEntries: Story = {
  args: {
    events: [sampleEvents[1], sampleEvents[2]],
  },
};

// ── Empty ────────────────────────────────────────────────────────────────

export const Empty: Story = {
  args: {
    events: [],
  },
};

// ── Max Items (Collapsible) ──────────────────────────────────────────────

export const Collapsible: Story = {
  args: {
    events: sampleEvents,
    maxItems: 3,
  },
};

// ── Single Event ─────────────────────────────────────────────────────────

export const SingleEvent: Story = {
  args: {
    events: [sampleEvents[0]],
  },
};

// ── All Event Types ──────────────────────────────────────────────────────

export const AllEventTypes: Story = {
  args: {
    events: [
      {
        id: 'status',
        timestamp: new Date().toISOString(),
        type: 'status_change',
        title: 'Status Change',
        description: 'Device went online',
      },
      {
        id: 'assign',
        timestamp: new Date().toISOString(),
        type: 'assignment',
        title: 'Assignment',
        description: 'Assigned to technician',
        user: 'John Doe',
      },
      {
        id: 'maint',
        timestamp: new Date().toISOString(),
        type: 'maintenance',
        title: 'Maintenance',
        description: 'Routine check completed',
      },
      {
        id: 'part',
        timestamp: new Date().toISOString(),
        type: 'part',
        title: 'Part Replaced',
        description: 'Hard drive replaced',
      },
      {
        id: 'note',
        timestamp: new Date().toISOString(),
        type: 'note',
        title: 'Note Added',
        description: 'Additional observation noted',
      },
      {
        id: 'sys',
        timestamp: new Date().toISOString(),
        type: 'system',
        title: 'System Event',
        description: 'Automatic sync completed',
      },
      {
        id: 'photo',
        timestamp: new Date().toISOString(),
        type: 'photo',
        title: 'Photo Added',
        description: 'Inspection photo uploaded',
      },
      {
        id: 'part_used',
        timestamp: new Date().toISOString(),
        type: 'part_used',
        title: 'Part Used',
        description: 'Screw kit M4 used',
      },
    ],
  },
};

// ── Playground ───────────────────────────────────────────────────────────

export const Playground: Story = {
  args: {
    events: sampleEvents.slice(0, 3),
    maxItems: 0,
  },
};
