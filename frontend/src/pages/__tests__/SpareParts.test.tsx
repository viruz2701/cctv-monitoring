import React from 'react';
import { render, screen } from '@testing-library/react';
import { describe, it, expect, vi } from 'vitest';
import { QueryClient, QueryClientProvider } from '@tanstack/react-query';
import { SpareParts } from '../SpareParts';

const queryClient = new QueryClient({
  defaultOptions: { queries: { retry: false } },
});

function renderWithProviders(ui: React.ReactElement) {
  return render(<QueryClientProvider client={queryClient}>{ui}</QueryClientProvider>);
}

vi.mock('react-i18next', () => ({
  useTranslation: () => ({
    t: (key: string) => key,
  }),
}));

const mockParts = [
  { id: '1', name: 'HDD 1TB', sku: 'HDD-001', category: 'storage', stock: 3, min_stock: 5, cost: 100, supplier: 'Supplier A', location: 'Shelf 1' },
  { id: '2', name: 'Camera Lens', sku: 'LENS-001', category: 'optics', stock: 10, min_stock: 2, cost: 250, supplier: 'Supplier B', location: 'Shelf 2' },
];

vi.mock('../../hooks/useApiQuery', () => ({
  useSpareParts: () => ({ data: mockParts, isLoading: false }),
  useLowStockParts: () => ({ data: [mockParts[0]], isLoading: false }),
  useSparePartCategories: () => ({ data: ['storage', 'optics'], isLoading: false }),
  useCreateSparePartCategory: () => ({ mutateAsync: vi.fn() }),
  useUpdateSparePartCategory: () => ({ mutateAsync: vi.fn() }),
  useDeleteSparePartCategory: () => ({ mutateAsync: vi.fn() }),
  useCreateSparePart: () => ({ mutateAsync: vi.fn() }),
  useUpdateSparePart: () => ({ mutateAsync: vi.fn() }),
  useDeleteSparePart: () => ({ mutateAsync: vi.fn() }),
  useAdjustStock: () => ({ mutateAsync: vi.fn() }),
  useDevices: () => ({ data: [], isLoading: false }),
  useSites: () => ({ data: [], isLoading: false }),
}));

describe('SpareParts', () => {
  it('renders part list', async () => {
    renderWithProviders(<SpareParts />);
    const title = await screen.findByText('spare_parts', {}, { timeout: 3000 });
    expect(title).toBeTruthy();
  });

  it('shows low stock indicator when parts below minimum', async () => {
    renderWithProviders(<SpareParts />);
    const badge = await screen.findByText('Low Stock', {}, { timeout: 3000 });
    expect(badge).toBeTruthy();
  });

  it('shows add part button', async () => {
    renderWithProviders(<SpareParts />);
    const btn = await screen.findByText('add_part', {}, { timeout: 3000 });
    expect(btn).toBeTruthy();
  });

  it('shows search input', async () => {
    renderWithProviders(<SpareParts />);
    const input = await screen.findByPlaceholderText('search_parts', {}, { timeout: 3000 });
    expect(input).toBeTruthy();
  });
});
