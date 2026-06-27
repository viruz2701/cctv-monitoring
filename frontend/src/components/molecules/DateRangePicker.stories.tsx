import type { Meta, StoryObj } from '@storybook/react';
import { DateRangePicker, type DateRange } from './DateRangePicker';
import { useState } from 'react';

const meta: Meta<typeof DateRangePicker> = {
  title: 'Molecules/DateRangePicker',
  component: DateRangePicker,
  tags: ['autodocs'],
};

export default meta;
type Story = StoryObj<typeof DateRangePicker>;

// ── Default (Last 7 Days) ────────────────────────────────────────────────

const DefaultTemplate = () => {
  const [range, setRange] = useState<DateRange>();
  return <DateRangePicker value={range} onChange={setRange} />;
};

export const Default: Story = {
  render: () => <DefaultTemplate />,
};

// ── Today Preset ─────────────────────────────────────────────────────────

const TodayTemplate = () => {
  const [range, setRange] = useState<DateRange>({
    start: new Date(),
    end: new Date(),
    preset: 'today',
  });
  return <DateRangePicker value={range} onChange={setRange} />;
};

export const Today: Story = {
  render: () => <TodayTemplate />,
};

// ── Last 30 Days ─────────────────────────────────────────────────────────

const Last30Template = () => {
  const [range, setRange] = useState<DateRange>();
  return <DateRangePicker value={range} onChange={setRange} />;
};

export const Last30Days: Story = {
  render: () => <Last30Template />,
};

// ── This Month ───────────────────────────────────────────────────────────

const ThisMonthTemplate = () => {
  const [range, setRange] = useState<DateRange>();
  return <DateRangePicker value={range} onChange={setRange} />;
};

export const ThisMonth: Story = {
  render: () => <ThisMonthTemplate />,
};

// ── Custom Range ─────────────────────────────────────────────────────────

const CustomTemplate = () => {
  const [range, setRange] = useState<DateRange>({
    start: new Date(2024, 0, 1),
    end: new Date(2024, 11, 31),
    preset: 'custom',
  });
  return <DateRangePicker value={range} onChange={setRange} />;
};

export const CustomRange: Story = {
  render: () => <CustomTemplate />,
};

// ── With Min/Max Date ────────────────────────────────────────────────────

const MinMaxTemplate = () => {
  const [range, setRange] = useState<DateRange>();
  return (
    <DateRangePicker
      value={range}
      onChange={setRange}
      minDate={new Date(2024, 0, 1)}
      maxDate={new Date(2026, 11, 31)}
    />
  );
};

export const WithDateLimits: Story = {
  render: () => <MinMaxTemplate />,
};

// ── Playground ───────────────────────────────────────────────────────────

const PlaygroundTemplate = () => {
  const [range, setRange] = useState<DateRange>();
  return <DateRangePicker value={range} onChange={setRange} className="w-72" />;
};

export const Playground: Story = {
  render: () => <PlaygroundTemplate />,
};
