import React, { useState } from 'react';
import type { Meta, StoryObj } from '@storybook/react';
import { Table, Pagination } from './Table';
import { Badge } from './Badge';

// ── Sample Data ───────────────────────────────────────────────────────────

interface Device {
  id: string;
  name: string;
  ip: string;
  status: 'online' | 'offline' | 'warning';
  type: string;
  lastSeen: string;
}

const sampleDevices: Device[] = [
  { id: '1', name: 'NVR-01', ip: '192.168.1.10', status: 'online', type: 'NVR', lastSeen: '2024-01-15 14:30' },
  { id: '2', name: 'CAM-101', ip: '192.168.1.101', status: 'online', type: 'Camera', lastSeen: '2024-01-15 14:29' },
  { id: '3', name: 'CAM-102', ip: '192.168.1.102', status: 'offline', type: 'Camera', lastSeen: '2024-01-14 09:15' },
  { id: '4', name: 'GW-01', ip: '192.168.1.1', status: 'online', type: 'Gateway', lastSeen: '2024-01-15 14:30' },
  { id: '5', name: 'SENSOR-TEMP', ip: '192.168.1.50', status: 'warning', type: 'Sensor', lastSeen: '2024-01-15 12:00' },
];

const columns = [
  { key: 'name' as const, header: 'Name', sortable: true },
  { key: 'ip' as const, header: 'IP Address', sortable: true },
  {
    key: 'status' as const,
    header: 'Status',
    sortable: true,
    render: (item: Device) => (
      <Badge
        variant={item.status === 'online' ? 'success' : item.status === 'warning' ? 'warning' : 'danger'}
        dot
        size="sm"
      >
        {item.status}
      </Badge>
    ),
  },
  { key: 'type' as const, header: 'Type' },
  { key: 'lastSeen' as const, header: 'Last Seen', sortable: true },
];

const meta: Meta<typeof Table> = {
  title: 'UI/Table',
  component: Table,
  tags: ['autodocs'],
  argTypes: {
    loading: { control: 'boolean' },
    emptyMessage: { control: 'text' },
  },
};

export default meta;

// ── Default Table ─────────────────────────────────────────────────────────

export const Default: StoryObj<typeof Table> = {
  render: () => (
    <Table
      data={sampleDevices}
      columns={columns}
      keyExtractor={(item) => item.id}
    />
  ),
};

// ── With Sort ─────────────────────────────────────────────────────────────

function SortDemo() {
  const [sortColumn, setSortColumn] = useState('name');
  const [sortDirection, setSortDirection] = useState<'asc' | 'desc'>('asc');

  const handleSort = (column: string) => {
    if (sortColumn === column) {
      setSortDirection((prev) => (prev === 'asc' ? 'desc' : 'asc'));
    } else {
      setSortColumn(column);
      setSortDirection('asc');
    }
  };

  const sorted = [...sampleDevices].sort((a, b) => {
    const aVal = String(a[sortColumn as keyof Device] ?? '');
    const bVal = String(b[sortColumn as keyof Device] ?? '');
    return sortDirection === 'asc' ? aVal.localeCompare(bVal) : bVal.localeCompare(aVal);
  });

  return (
    <Table
      data={sorted}
      columns={columns}
      keyExtractor={(item) => item.id}
      sortColumn={sortColumn}
      sortDirection={sortDirection}
      onSort={handleSort}
    />
  );
}

export const WithSort: StoryObj = {
  render: () => <SortDemo />,
};

// ── With Row Click ────────────────────────────────────────────────────────

export const WithRowClick: StoryObj<typeof Table> = {
  render: () => (
    <Table
      data={sampleDevices}
      columns={columns}
      keyExtractor={(item) => item.id}
      onRowClick={(item) => alert(`Clicked: ${item.name}`)}
    />
  ),
};

// ── Loading ───────────────────────────────────────────────────────────────

export const Loading: StoryObj<typeof Table> = {
  render: () => (
    <Table
      data={[]}
      columns={columns}
      keyExtractor={(item) => item.id}
      loading
    />
  ),
};

// ── Empty ─────────────────────────────────────────────────────────────────

export const Empty: StoryObj<typeof Table> = {
  render: () => (
    <Table
      data={[]}
      columns={columns}
      keyExtractor={(item) => item.id}
      emptyMessage="No devices found"
    />
  ),
};

// ── With Expandable Rows ──────────────────────────────────────────────────

export const WithExpandableRows: StoryObj<typeof Table> = {
  render: () => (
    <Table
      data={sampleDevices.slice(0, 3)}
      columns={columns}
      keyExtractor={(item) => item.id}
      expandable={() => (
        <div className="p-4 bg-slate-50 dark:bg-slate-800/50">
          <div className="grid grid-cols-2 gap-4 text-sm">
            <div>
              <span className="text-slate-500">Firmware:</span>{' '}
              <span className="text-slate-700 dark:text-slate-300">v2.1.4</span>
            </div>
            <div>
              <span className="text-slate-500">Location:</span>{' '}
              <span className="text-slate-700 dark:text-slate-300">Building A, Floor 3</span>
            </div>
          </div>
        </div>
      )}
    />
  ),
};

// ── Pagination Component ──────────────────────────────────────────────────

const paginationMeta: Meta<typeof Pagination> = {
  title: 'UI/Table/Pagination',
  component: Pagination,
  tags: ['autodocs'],
};

export { paginationMeta };

function PaginationDefaultDemo() {
  const [page, setPage] = useState(1);
  return (
    <Pagination
      currentPage={page}
      totalPages={10}
      onPageChange={setPage}
      totalItems={97}
      itemsPerPage={10}
    />
  );
}

export const PaginationDefault: StoryObj<typeof Pagination> = {
  render: () => <PaginationDefaultDemo />,
};

export const PaginationFirstPage: StoryObj<typeof Pagination> = {
  render: () => (
    <Pagination
      currentPage={1}
      totalPages={5}
      onPageChange={() => {}}
      totalItems={50}
      itemsPerPage={10}
    />
  ),
};

export const PaginationLastPage: StoryObj<typeof Pagination> = {
  render: () => (
    <Pagination
      currentPage={5}
      totalPages={5}
      onPageChange={() => {}}
      totalItems={50}
      itemsPerPage={10}
    />
  ),
};

export const PaginationManyPages: StoryObj<typeof Pagination> = {
  render: () => (
    <Pagination
      currentPage={25}
      totalPages={50}
      onPageChange={() => {}}
      totalItems={500}
      itemsPerPage={10}
    />
  ),
};

export const PaginationSinglePage: StoryObj<typeof Pagination> = {
  render: () => (
    <Pagination
      currentPage={1}
      totalPages={1}
      onPageChange={() => {}}
      totalItems={3}
      itemsPerPage={10}
    />
  ),
};
