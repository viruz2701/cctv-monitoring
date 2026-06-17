import { render, screen } from '@testing-library/react';
import { describe, it, expect, vi } from 'vitest';
import '@testing-library/jest-dom';
import { WorkOrders } from '../WorkOrders';
import { WorkOrdersProvider } from '../../context/WorkOrdersContext';

// Mock the context
vi.mock('../../context/WorkOrdersContext', () => ({
  useWorkOrders: () => ({
    workOrders: [
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
    loading: false,
    error: null,
    fetchWorkOrders: vi.fn(),
    createWorkOrder: vi.fn(),
    updateWorkOrder: vi.fn(),
    deleteWorkOrder: vi.fn(),
    assignWorkOrder: vi.fn(),
    startWorkOrder: vi.fn(),
    completeWorkOrder: vi.fn(),
    cancelWorkOrder: vi.fn(),
  }),
  WorkOrdersProvider: ({ children }: { children: React.ReactNode }) => <>{children}</>,
}));

// Mock i18n
vi.mock('react-i18next', () => ({
  useTranslation: () => ({
    t: (key: string) => key,
  }),
}));

describe('WorkOrders', () => {
  it('renders work orders list', () => {
    render(
      <WorkOrdersProvider>
        <WorkOrders />
      </WorkOrdersProvider>
    );
    expect(screen.getByText('work_orders')).toBeInTheDocument();
  });

  it('displays work order data', () => {
    render(
      <WorkOrdersProvider>
        <WorkOrders />
      </WorkOrdersProvider>
    );
    expect(screen.getByText('Camera 1')).toBeInTheDocument();
  });
});
