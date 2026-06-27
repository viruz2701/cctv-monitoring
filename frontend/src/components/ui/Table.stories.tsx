import type { Meta, StoryObj } from '@storybook/react';
import { Table, Pagination } from './Table';
import { Badge } from './Badge';
import { useState } from 'react';

interface Device {
  id: string;
  name: string;
  status: 'online' | 'offline' | 'warning';
  ip: string;
  site: string;
  uptime: string;
}

const sampleDevices: Device[] = [
  { id: '1', name: 'NVR-01', status: 'online', ip: '192.168.1.100', site: 'Main Office', uptime: '72d 4h' },
  { id: '2', name: 'CAM-101', status: 'online', ip: '192.168.1.101', site: 'Main Office', uptime: '30d 12h' },
  { id: '3', name: 'CAM-102', status: 'offline', ip: '192.168.1.102', site: 'Main Office', uptime: '0d 0h' },
  { id: '4', name: 'NVR-02', status: 'warning', ip: '192.168.2.100', site: 'Warehouse', uptime: '15d 8h' },
  { id: '5', name: 'CAM-201', status: 'online', ip: '192.168.2.101', site: 'Warehouse', uptime: '45d 2h' },
  { id: '6', name: 'NVR-03', status: 'online', ip: '192.168.3.100', site: 'Branch Office', uptime: '90d 0h' },
];

const meta: Meta<typeof Table> = {
  title: 'UI/Table',
  component: Table,
  tags: ['autodocs'],
};

export default meta;
type Story = StoryObj<typeof Table>;

const columns: any = [
  { key: 'name', header: 'Name', sortable: true },
  { key: 'ip', header: 'IP Address' },
  { key: 'site', header: 'Site', sortable: true },
  { key: 'uptime', header: 'Uptime' },
  { key: 'status', header: 'Status', render: (item: Device) => (
    <Badge variant={item.status === 'online' ? 'success' : item.status === 'warning' ? 'warning' : 'danger'} size="sm">
      {item.status}
    </Badge>
  )},
];

// ── Basic ────────────────────────────────────────────────────────────────

export const Basic: Story = {
  args: {
    data: sampleDevices,
    columns,
    keyExtractor: (item: Device) => item.id,
  },
};

// ── Sortable ─────────────────────────────────────────────────────────────

const SortableTemplate = () => {
  const [sortColumn, setSortColumn] = useState('name');
  const [sortDirection, setSortDirection] = useState<'asc' | 'desc'>('asc');
  const sorted = [...sampleDevices].sort((a, b) => {
    const valA = (a as any)[sortColumn];
    const valB = (b as any)[sortColumn];
    return sortDirection === 'asc' ? String(valA).localeCompare(String(valB)) : String(valB).localeCompare(String(valA));
  });
  return (
    <Table
      data={sorted}
      columns={columns}
      keyExtractor={(item: Device) => item.id}
      sortColumn={sortColumn}
      sortDirection={sortDirection}
      onSort={(col) => {
        setSortDirection(prev => sortColumn === col && prev === 'asc' ? 'desc' : 'asc');
        setSortColumn(col);
      }}
    />
  );
};

export const Sortable: Story = {
  render: () => <SortableTemplate />,
};

// ── Loading ──────────────────────────────────────────────────────────────

export const Loading: Story = {
  args: {
    data: [],
    columns,
    keyExtractor: (item: Device) => item.id,
    loading: true,
  },
};

// ── Empty ────────────────────────────────────────────────────────────────

export const Empty: Story = {
  args: {
    data: [],
    columns,
    keyExtractor: (item: Device) => item.id,
    emptyMessage: 'No devices found matching your search criteria.',
  },
};

// ── Expandable Rows ──────────────────────────────────────────────────────

export const ExpandableRows: Story = {
  args: {
    data: sampleDevices.slice(0, 3),
    columns,
    keyExtractor: (item: Device) => item.id,
    expandable: (item: Device) => (
      <div className="px-6 py-4 bg-slate-50 dark:bg-slate-800/50">
        <div className="grid grid-cols-2 gap-4 text-sm">
          <div>
            <span className="text-slate-500">Model:</span>{' '}
            <span className="text-slate-900 dark:text-white font-medium">Pro Series</span>
          </div>
          <div>
            <span className="text-slate-500">Firmware:</span>{' '}
            <span className="text-slate-900 dark:text-white font-medium">v3.2.1</span>
          </div>
          <div>
            <span className="text-slate-500">Last Seen:</span>{' '}
            <span className="text-slate-900 dark:text-white font-medium">{new Date().toLocaleString()}</span>
          </div>
          <div>
            <span className="text-slate-500">Location:</span>{' '}
            <span className="text-slate-900 dark:text-white font-medium">{item.site}</span>
          </div>
        </div>
      </div>
    ),
  },
};

// ── Pagination Component ─────────────────────────────────────────────────

const paginationMeta: Meta<typeof Pagination> = {
  title: 'UI/Table/Pagination',
  component: Pagination,
  tags: ['autodocs'],
};

export const PaginationDefault: StoryObj<typeof Pagination> = {
  render: () => {
    const [page, setPage] = useState(1);
    return (
      <Pagination
        currentPage={page}
        totalPages={10}
        onPageChange={setPage}
        totalItems={95}
        itemsPerPage={10}
      />
    );
  },
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

export const PaginationManyPages: StoryObj<typeof Pagination> = {
  render: () => {
    const [page, setPage] = useState(42);
    return (
      <Pagination
        currentPage={page}
        totalPages={100}
        onPageChange={setPage}
        totalItems={1000}
        itemsPerPage={10}
      />
    );
  },
};

// ── Playground ───────────────────────────────────────────────────────────

export const Playground: Story = {
  args: {
    data: sampleDevices.slice(0, 3),
    columns: columns.slice(0, 3),
    keyExtractor: (item: Device) => item.id,
  },
};
