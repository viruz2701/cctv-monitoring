import React from 'react';
import { render, screen } from '@testing-library/react';
import { describe, it, expect, vi } from 'vitest';
import { SpareParts } from '../SpareParts';

vi.mock('react-i18next', () => ({
  useTranslation: () => ({
    t: (key: string) => key,
  }),
}));

vi.mock('../../context/SparePartsContext', () => ({
  useSpareParts: () => ({
    spareParts: [
      { id: '1', name: 'HDD 1TB', sku: 'HDD-001', category: 'storage', stock: 3, min_stock: 5, cost: 100, supplier: 'Supplier A', location: 'Shelf 1' },
      { id: '2', name: 'Camera Lens', sku: 'LENS-001', category: 'optics', stock: 10, min_stock: 2, cost: 250, supplier: 'Supplier B', location: 'Shelf 2' },
    ],
    loading: false,
    createSparePart: vi.fn(),
  }),
}));

describe('SpareParts', () => {
  it('renders part list', () => {
    render(<SpareParts />);
    expect(screen.getByText('spare_parts')).toBeTruthy();
  });

  it('shows low stock alert badge when parts below minimum', () => {
    render(<SpareParts />);
    // HDD has stock=3, min_stock=5 → low stock
    expect(screen.getByText(/1.*low_stock_alerts|low_stock_alerts.*1/)).toBeTruthy();
  });

  it('shows add part button', () => {
    render(<SpareParts />);
    expect(screen.getByText('add_part')).toBeTruthy();
  });

  it('shows search input', () => {
    render(<SpareParts />);
    const searchInput = screen.getByPlaceholderText('search_parts');
    expect(searchInput).toBeTruthy();
  });
});
