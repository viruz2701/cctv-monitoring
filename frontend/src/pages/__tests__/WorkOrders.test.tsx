import { render, screen } from '@testing-library/react';
import { describe, it, expect, vi } from 'vitest';
import '@testing-library/jest-dom';
import { QueryClient, QueryClientProvider } from '@tanstack/react-query';
import { WorkOrders } from '../WorkOrders';

// Mock React Query hooks
vi.mock('../../hooks/useApiQuery', () => ({
  useWorkOrders: () => ({
    data: [
      {
        id: '1',
        device_id: 'device-1',
        device_name: 'Camera 1',
        type: 'corrective',
        status: 'open',
        priority: 'high',
        created_at: new Date().toISOString(),
        updated_at: new Date().toISOString(),
        checklist: [],
        photos: [],
        parts_used: [],
      },
    ],
    isLoading: false,
  }),
  useUsers: () => ({ data: [], isLoading: false }),
  useSites: () => ({ data: [], isLoading: false }),
  useDevices: () => ({ data: [], isLoading: false }),
  useCreateWorkOrder: () => ({ mutateAsync: vi.fn() }),
  useUpdateWorkOrder: () => ({ mutateAsync: vi.fn() }),
  useDeleteWorkOrder: () => ({ mutateAsync: vi.fn() }),
  queryKeys: {
    workOrders: { all: ['workOrders'] },
  },
}));

vi.mock('../../services/workOrdersApi', () => ({
  workOrdersApi: {
    getWorkOrders: vi.fn(),
    createWorkOrder: vi.fn(),
    updateWorkOrder: vi.fn(),
    deleteWorkOrder: vi.fn(),
    assignWorkOrder: vi.fn(),
    startWorkOrder: vi.fn(),
    completeWorkOrder: vi.fn(),
    cancelWorkOrder: vi.fn(),
    bulkActions: vi.fn(),
    getWorkOrder: vi.fn(),
    getTimeEntries: vi.fn(),
    getLaborCost: vi.fn(),
  },
  WorkOrder: {},
}));

// Mock i18n
vi.mock('react-i18next', () => ({
  useTranslation: () => ({
    t: (key: string) => key,
  }),
}));

// Mock modules that might cause issues in test environment
vi.mock('../../components/work-orders/QuickFilters', () => ({
  QuickFilters: () => null,
  useQuickFilter: () => ['all', vi.fn()],
}));

const queryClient = new QueryClient({
  defaultOptions: {
    queries: { retry: false },
  },
});

function renderWithProviders(ui: React.ReactElement) {
  return render(
    <QueryClientProvider client={queryClient}>
      {ui}
    </QueryClientProvider>
  );
}

describe('WorkOrders', () => {
  it('renders work orders list', () => {
    renderWithProviders(<WorkOrders />);
    expect(screen.getByText('work_orders')).toBeInTheDocument();
  });

  it('displays work order data', () => {
    renderWithProviders(<WorkOrders />);
    expect(screen.getByText('Camera 1')).toBeInTheDocument();
  });
});
