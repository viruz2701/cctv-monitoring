import React from 'react';
import type { Meta, StoryObj } from '@storybook/react';
import { DataGrid } from './DataGrid';

// ── Sample Data ───────────────────────────────────────────────────────────

interface Device {
  id: string;
  name: string;
  ip: string;
  status: string;
  type: string;
  uptime: string;
  firmware: string;
}

const sampleDevices: Device[] = [
  { id: '1', name: 'NVR-01', ip: '192.168.1.10', status: 'Online', type: 'NVR', uptime: '99.9%', firmware: 'v2.1.4' },
  { id: '2', name: 'CAM-101', ip: '192.168.1.101', status: 'Online', type: 'Camera', uptime: '98.5%', firmware: 'v3.0.1' },
  { id: '3', name: 'CAM-102', ip: '192.168.1.102', status: 'Offline', type: 'Camera', uptime: '0%', firmware: 'v3.0.1' },
  { id: '4', name: 'GW-01', ip: '192.168.1.1', status: 'Online', type: 'Gateway', uptime: '99.99%', firmware: 'v1.8.2' },
  { id: '5', name: 'SENSOR-TEMP', ip: '192.168.1.50', status: 'Warning', type: 'Sensor', uptime: '95.2%', firmware: 'v1.0.5' },
];

const largeDataset: Device[] = Array.from({ length: 100 }, (_, i) => ({
  id: String(i + 1),
  name: `Device-${String(i + 1).padStart(3, '0')}`,
  ip: `192.168.${Math.floor(i / 255)}.${(i % 255) + 1}`,
  status: ['Online', 'Offline', 'Warning'][i % 3],
  type: ['Camera', 'NVR', 'Sensor', 'Gateway'][i % 4],
  uptime: `${(Math.random() * 20 + 80).toFixed(1)}%`,
  firmware: `v${Math.floor(Math.random() * 3)}.${Math.floor(Math.random() * 10)}.${Math.floor(Math.random() * 10)}`,
}));

const columns = [
  { key: 'name' as const, header: 'Name', sortable: true, width: 160, hideable: true },
  { key: 'ip' as const, header: 'IP Address', sortable: true, width: 140, hideable: true },
  { key: 'status' as const, header: 'Status', sortable: true, width: 100, hideable: true },
  { key: 'type' as const, header: 'Type', sortable: true, width: 100, hideable: true },
  { key: 'uptime' as const, header: 'Uptime', sortable: true, width: 100, hideable: true },
  { key: 'firmware' as const, header: 'Firmware', sortable: false, width: 100, hideable: true },
];

const meta: Meta<typeof DataGrid> = {
  title: 'UI/DataGrid',
  component: DataGrid,
  tags: ['autodocs'],
  argTypes: {
    variant: {
      control: 'select',
      options: ['default', 'striped', 'bordered', 'minimal'],
    },
    defaultDensity: {
      control: 'select',
      options: ['compact', 'standard', 'comfortable'],
    },
  },
};

export default meta;

// ── Default ───────────────────────────────────────────────────────────────

export const Default: StoryObj<typeof DataGrid> = {
  render: () => (
    <DataGrid
      data={sampleDevices}
      columns={columns}
      keyExtractor={(item: Device) => item.id}
    />
  ),
};

// ── With Data ─────────────────────────────────────────────────────────────

export const WithData: StoryObj<typeof DataGrid> = {
  render: () => (
    <DataGrid
      data={sampleDevices}
      columns={columns}
      keyExtractor={(item: Device) => item.id}
      exportFilename="devices.csv"
    />
  ),
};

// ── Empty ─────────────────────────────────────────────────────────────────

export const Empty: StoryObj<typeof DataGrid> = {
  render: () => (
    <DataGrid
      data={[]}
      columns={columns}
      keyExtractor={(item: Device) => item.id}
      emptyMessage="No devices registered yet"
    />
  ),
};

// ── Loading ───────────────────────────────────────────────────────────────

export const Loading: StoryObj<typeof DataGrid> = {
  render: () => (
    <DataGrid
      data={[]}
      columns={columns}
      keyExtractor={(item: Device) => item.id}
      loading
    />
  ),
};

// ── With Selection ────────────────────────────────────────────────────────

function SelectionDemo() {
  const [selectedIds, setSelectedIds] = React.useState<Set<string>>(new Set(['1', '3']));
  return (
    <DataGrid
      data={sampleDevices}
      columns={columns}
      keyExtractor={(item: Device) => item.id}
      selectable
      selectedIds={selectedIds}
      onSelectionChange={setSelectedIds}
      bulkActions={[
        { label: 'Delete', onClick: (items) => alert(`Delete ${items.length} items`), variant: 'danger' },
        { label: 'Export', onClick: (items) => alert(`Export ${items.length} items`) },
      ]}
    />
  );
}

export const WithSelection: StoryObj = {
  render: () => <SelectionDemo />,
};

// ── Striped Variant ───────────────────────────────────────────────────────

export const Striped: StoryObj<typeof DataGrid> = {
  render: () => (
    <DataGrid
      data={sampleDevices}
      columns={columns}
      keyExtractor={(item: Device) => item.id}
      variant="striped"
    />
  ),
};

// ── Bordered Variant ──────────────────────────────────────────────────────

export const Bordered: StoryObj<typeof DataGrid> = {
  render: () => (
    <DataGrid
      data={sampleDevices}
      columns={columns}
      keyExtractor={(item: Device) => item.id}
      variant="bordered"
    />
  ),
};

// ── Minimal Variant ───────────────────────────────────────────────────────

export const Minimal: StoryObj<typeof DataGrid> = {
  render: () => (
    <DataGrid
      data={sampleDevices}
      columns={columns}
      keyExtractor={(item: Device) => item.id}
      variant="minimal"
    />
  ),
};

// ── Compact Density ───────────────────────────────────────────────────────

export const CompactDensity: StoryObj<typeof DataGrid> = {
  render: () => (
    <DataGrid
      data={sampleDevices}
      columns={columns}
      keyExtractor={(item: Device) => item.id}
      defaultDensity="compact"
    />
  ),
};

// ── Comfortable Density ───────────────────────────────────────────────────

export const ComfortableDensity: StoryObj<typeof DataGrid> = {
  render: () => (
    <DataGrid
      data={sampleDevices}
      columns={columns}
      keyExtractor={(item: Device) => item.id}
      defaultDensity="comfortable"
    />
  ),
};

// ── With Sort ─────────────────────────────────────────────────────────────

function SortDemo() {
  const [sortColumn, setSortColumn] = React.useState('name');
  const [sortDirection, setSortDirection] = React.useState<'asc' | 'desc'>('asc');

  const handleSort = (column: string) => {
    if (sortColumn === column) {
      setSortDirection((prev) => (prev === 'asc' ? 'desc' : 'asc'));
    } else {
      setSortColumn(column);
      setSortDirection('asc');
    }
  };

  return (
    <DataGrid
      data={sampleDevices}
      columns={columns}
      keyExtractor={(item: Device) => item.id}
      sortColumn={sortColumn}
      sortDirection={sortDirection}
      onSort={handleSort}
    />
  );
}

export const WithSort: StoryObj = {
  render: () => <SortDemo />,
};

// ── Large Dataset ─────────────────────────────────────────────────────────

export const LargeDataset: StoryObj<typeof DataGrid> = {
  render: () => (
    <DataGrid
      data={largeDataset}
      columns={columns}
      keyExtractor={(item: Device) => item.id}
      pageSize={15}
    />
  ),
};

// ── With Pivot ────────────────────────────────────────────────────────────

function PivotDemo() {
  const [pivotCols, setPivotCols] = React.useState<string[]>(['type']);
  return (
    <DataGrid
      data={sampleDevices}
      columns={columns}
      keyExtractor={(item: Device) => item.id}
      pivotable
      pivotColumns={pivotCols}
      onPivotChange={setPivotCols}
    />
  );
}

export const WithPivot: StoryObj = {
  render: () => <PivotDemo />,
};

// ── Playground ────────────────────────────────────────────────────────────

export const Playground: StoryObj<typeof DataGrid> = {
  render: () => (
    <DataGrid
      data={sampleDevices}
      columns={columns}
      keyExtractor={(item: Device) => item.id}
      variant="default"
      defaultDensity="standard"
      exportable
    />
  ),
};
