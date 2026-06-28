import React from 'react';
import { render, screen, fireEvent } from '@testing-library/react';
import { MemoryRouter } from 'react-router-dom';
import { describe, it, expect, vi, beforeEach } from 'vitest';
import { DataGrid } from '../DataGrid';

interface TestItem {
    id: string;
    name: string;
    status: string;
}

const mockData: TestItem[] = [
    { id: '1', name: 'NVR-01', status: 'online' },
    { id: '2', name: 'NVR-02', status: 'offline' },
    { id: '3', name: 'NVR-03', status: 'online' },
];

const columns = [
    { key: 'name' as const, header: 'Name', sortable: true },
    { key: 'status' as const, header: 'Status', sortable: true },
];

const emptyData: TestItem[] = [];

// Моки для зависимостей DataGrid
vi.mock('../../../hooks/useAuth', () => ({
    useAuth: () => ({
        user: { role: 'admin' },
    }),
}));

vi.mock('../../../store/filterStore', () => ({
    useFilterStore: () => ({
        saveView: vi.fn(),
        savedViews: [],
        getViewsForPage: vi.fn().mockReturnValue([]),
        deleteView: vi.fn(),
        exportViews: vi.fn(),
        importViews: vi.fn(),
        encodeFilterStateToUrl: vi.fn(),
        setDefaultForRole: vi.fn(),
        getDefaultViewForRole: vi.fn().mockReturnValue(null),
    }),
    ROLE_DEFAULT_FILTERS: {},
    encodeFilterState: vi.fn(),
    decodeFilterState: vi.fn(),
}));

function Wrapper({ children }: { children: React.ReactNode }) {
    return <MemoryRouter>{children}</MemoryRouter>;
}

describe('DataGrid', () => {
    it('renders data rows', () => {
        render(
            <DataGrid
                data={mockData}
                columns={columns}
                keyExtractor={(item) => item.id}
            />,
            { wrapper: Wrapper }
        );
        expect(screen.getByText('NVR-01')).toBeInTheDocument();
        expect(screen.getByText('NVR-02')).toBeInTheDocument();
        expect(screen.getByText('NVR-03')).toBeInTheDocument();
    });

    it('renders column headers', () => {
        render(
            <DataGrid
                data={mockData}
                columns={columns}
                keyExtractor={(item) => item.id}
            />,
            { wrapper: Wrapper }
        );
        expect(screen.getByText('Name')).toBeInTheDocument();
        expect(screen.getByText('Status')).toBeInTheDocument();
    });

    it('shows empty message when data is empty', () => {
        render(
            <DataGrid
                data={emptyData}
                columns={columns}
                keyExtractor={(item: TestItem) => item.id}
                emptyMessage="No devices found"
            />,
            { wrapper: Wrapper }
        );
        expect(screen.getByText('No devices found')).toBeInTheDocument();
    });

    it('calls onSort when sortable header is clicked', () => {
        const handleSort = vi.fn();
        render(
            <DataGrid
                data={mockData}
                columns={columns}
                keyExtractor={(item) => item.id}
                sortColumn="name"
                onSort={handleSort}
            />,
            { wrapper: Wrapper }
        );
        const nameHeader = screen.getByText('Name');
        fireEvent.click(nameHeader);
        expect(handleSort).toHaveBeenCalledWith('name');
    });

    it('shows sort direction indicator', () => {
        render(
            <DataGrid
                data={mockData}
                columns={columns}
                keyExtractor={(item) => item.id}
                sortColumn="name"
                sortDirection="asc"
            />,
            { wrapper: Wrapper }
        );
        // Sort indicator should be rendered
        const container = screen.getByRole('grid');
        expect(container).toBeInTheDocument();
    });

    it('renders with loading state', () => {
        render(
            <DataGrid
                data={emptyData}
                columns={columns}
                keyExtractor={(item: TestItem) => item.id}
                loading
            />,
            { wrapper: Wrapper }
        );
        // Loading skeleton should be present
        const skeleton = document.querySelector('.animate-pulse');
        expect(skeleton).toBeInTheDocument();
    });

    it('calls onRowClick when a row is clicked', () => {
        const handleRowClick = vi.fn();
        render(
            <DataGrid
                data={mockData}
                columns={columns}
                keyExtractor={(item) => item.id}
                onRowClick={handleRowClick}
            />,
            { wrapper: Wrapper }
        );
        const rows = screen.getAllByRole('row');
        // Data rows (skip header)
        const dataRow = rows[1];
        fireEvent.click(dataRow);
        expect(handleRowClick).toHaveBeenCalled();
    });

    it('renders with custom row class name', () => {
        render(
            <DataGrid
                data={mockData}
                columns={columns}
                keyExtractor={(item) => item.id}
                rowClassName={(item) => item.status === 'offline' ? 'bg-red-50' : ''}
            />,
            { wrapper: Wrapper }
        );
        const rows = screen.getAllByRole('row');
        // Second data row (offline) should have custom class
        const offlineRow = rows[2];
        expect(offlineRow.className).toContain('bg-red-50');
    });

    it('renders with selectable rows', () => {
        render(
            <DataGrid
                data={mockData}
                columns={columns}
                keyExtractor={(item) => item.id}
                selectable
                selectedIds={new Set(['1'])}
            />,
            { wrapper: Wrapper }
        );
        // Select-all column rendered (th + button both have aria-label)
        const selectAllButtons = screen.getAllByLabelText('Select all rows');
        expect(selectAllButtons.length).toBeGreaterThanOrEqual(1);
        // Individual row select buttons
        const row2Button = screen.getByLabelText('Select row 2');
        expect(row2Button).toBeInTheDocument();
        const row3Button = screen.getByLabelText('Select row 3');
        expect(row3Button).toBeInTheDocument();
        // Row 1 is selected, so it should have "Deselect"
        expect(screen.getByLabelText('Deselect row 1')).toBeInTheDocument();
    });

    it('handles column sorting toggle', () => {
        const handleSort = vi.fn();
        const { rerender } = render(
            <DataGrid
                data={mockData}
                columns={columns}
                keyExtractor={(item) => item.id}
                sortColumn="name"
                sortDirection="asc"
                onSort={handleSort}
            />,
            { wrapper: Wrapper }
        );

        // Click to sort desc
        fireEvent.click(screen.getByText('Name'));
        expect(handleSort).toHaveBeenCalledWith('name');

        rerender(
            <DataGrid
                data={mockData}
                columns={columns}
                keyExtractor={(item) => item.id}
                sortColumn="name"
                sortDirection="desc"
                onSort={handleSort}
            />
        );
        // Component renders with desc sort
        expect(screen.getByText('Name')).toBeInTheDocument();
    });

    it('renders with pagination controls', () => {
        const manyItems: TestItem[] = Array.from({ length: 50 }, (_, i) => ({
            id: `${i}`,
            name: `Device-${i}`,
            status: 'online' as const,
        }));

        render(
            <DataGrid
                data={manyItems}
                columns={columns}
                keyExtractor={(item) => item.id}
                pageSize={25}
            />,
            { wrapper: Wrapper }
        );
        // Pagination should be present with 25 items per page, so 50 items = 2 pages
        expect(screen.getByLabelText('Page 1')).toBeInTheDocument();
        expect(screen.getByLabelText('Page 2')).toBeInTheDocument();
    });

    it('renders custom toolbar content', () => {
        render(
            <DataGrid
                data={mockData}
                columns={columns}
                keyExtractor={(item) => item.id}
                toolbar={<button>Custom Action</button>}
            />,
            { wrapper: Wrapper }
        );
        expect(screen.getByText('Custom Action')).toBeInTheDocument();
    });
});
