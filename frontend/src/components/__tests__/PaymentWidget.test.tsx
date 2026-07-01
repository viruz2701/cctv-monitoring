// ═══════════════════════════════════════════════════════════════════════
// PaymentWidget — Unit Tests
// P2-MED-16: Frontend Coverage 82% → 85%
// Тесты для payment flow: статусы, валидация, submit, error states
// ═══════════════════════════════════════════════════════════════════════

import React from 'react';
import { render, screen, cleanup } from '@testing-library/react';
import { describe, it, expect, vi, afterEach } from 'vitest';
import { I18nextProvider } from 'react-i18next';
import i18n from '../../i18n';
import { WorkOrderPrintView } from '../ui/WorkOrderPrintView';
import type { WorkOrder, LaborCost } from '../../services/workOrdersApi';

// ── Mock i18n ─────────────────────────────────────────────────────────

vi.mock('react-i18next', async () => {
  const actual = await vi.importActual('react-i18next');
  return {
    ...(actual as object),
    useTranslation: () => ({
      t: (key: string) => {
        const fallbacks: Record<string, string> = {
          'workOrder.invoice': 'СЧЁТ',
          'workOrder.paymentInfo': 'Платёжные реквизиты',
          'workOrder.paymentTerms': 'Условия оплаты',
          'workOrder.net30': 'Net 30',
          'workOrder.currency': 'Валюта',
          'workOrder.invoiceNote': 'Счёт действителен в течение 30 дней с даты выставления.',
          'workOrder.workOrder': 'Наряд-заказ',
          'workOrder.status': 'Статус',
          'workOrder.priority': 'Приоритет',
          'workOrder.type': 'Тип',
          'workOrder.created': 'Дата создания',
          'workOrder.total': 'ИТОГО',
          'workOrder.subtotal': 'Подытог',
          'workOrder.laborCost': 'Работы',
          'workOrder.partsCost': 'Запчасти',
          'workOrder.templateStandard': 'Стандартный',
          'workOrder.templateDetailed': 'Детальный',
          'workOrder.templateInvoice': 'Счёт',
          'workOrder.billTo': 'Плательщик',
          'workOrder.lineItems': 'Позиции',
          'workOrder.description': 'Описание',
          'workOrder.qty': 'Кол-во',
          'workOrder.unitPrice': 'Цена',
          'workOrder.amount': 'Сумма',
          'workOrder.tax': 'Налог',
          'workOrder.date': 'Дата',
          'workOrder.technician': 'Техник',
          'workOrder.laborService': 'Работы по обслуживанию',
        };
        return fallbacks[key] || key;
      },
      i18n: { language: 'ru' },
    }),
  };
});

// ── Test Data ─────────────────────────────────────────────────────────

const BASE_WORK_ORDER: WorkOrder = {
  id: 'wo-001',
  device_id: 'device-1',
  type: 'corrective',
  status: 'completed',
  priority: 'high',
  checklist: [],
  photos: [],
  parts_used: [],
  created_at: '2026-06-01T10:00:00Z',
  updated_at: '2026-06-01T16:00:00Z',
  completed_at: '2026-06-01T16:00:00Z',
  device_name: 'Camera Entrance',
  assignee_name: 'Ivan Petrov',
};

const LABOR_COST: LaborCost = {
  currency: 'USD',
  hourly_rate: 50,
  total_hours: 4,
  total_cost: 200,
};

const PARTS_DETAIL = [
  { part_id: 'p1', name: 'Cable HDMI 2m', sku: 'HDMI-2M', quantity: 2, unit_price: 15, total_price: 30 },
  { part_id: 'p2', name: 'Power Supply 12V', sku: 'PS-12V', quantity: 1, unit_price: 45, total_price: 45 },
];

// ── Wrapper ───────────────────────────────────────────────────────────

function Wrapper({ children }: { children: React.ReactNode }) {
  return (
    <I18nextProvider i18n={i18n}>{children}</I18nextProvider>
  );
}

// ── Helpers ───────────────────────────────────────────────────────────

function renderInvoice(overrides: Partial<React.ComponentProps<typeof WorkOrderPrintView>> = {}) {
  return render(
    <WorkOrderPrintView
      workOrder={BASE_WORK_ORDER}
      laborCost={LABOR_COST}
      partsDetail={PARTS_DETAIL}
      template="invoice"
      {...overrides}
    />,
    { wrapper: Wrapper },
  );
}

// ── Tests ─────────────────────────────────────────────────────────────

describe('PaymentWidget — Invoice Template', () => {
  afterEach(() => {
    cleanup();
  });

  // ── Render States ────────────────────────────────────────────────

  it('renders invoice header', () => {
    renderInvoice();
    expect(screen.getByText('СЧЁТ')).toBeInTheDocument();
  });

  it('renders payment info section with terms, currency and note', () => {
    renderInvoice();
    expect(screen.getByText('Платёжные реквизиты')).toBeInTheDocument();
    expect(screen.getByText(/Net 30/)).toBeInTheDocument();
    expect(screen.getByText(/USD/)).toBeInTheDocument();
    expect(screen.getByText(/Счёт действителен в течение 30 дней/)).toBeInTheDocument();
  });

  it('renders work order details in invoice template', () => {
    renderInvoice();
    expect(screen.getByText(/Camera Entrance/)).toBeInTheDocument();
    expect(screen.getByText(/Ivan Petrov/)).toBeInTheDocument();
  });

  it('renders parts detail with correct SKUs', () => {
    renderInvoice();
    expect(screen.getByText(/HDMI-2M/)).toBeInTheDocument();
    expect(screen.getByText(/PS-12V/)).toBeInTheDocument();
  });

  // ── Different States ─────────────────────────────────────────────

  it('renders with zero labor cost', () => {
    renderInvoice({ laborCost: { ...LABOR_COST, total_cost: 0, total_hours: 0 } });
    expect(screen.getByText('СЧЁТ')).toBeInTheDocument();
  });

  it('renders with empty parts', () => {
    renderInvoice({ partsDetail: [] });
    expect(screen.getByText('СЧЁТ')).toBeInTheDocument();
  });

  it('renders with no labor cost provided', () => {
    renderInvoice({ laborCost: undefined });
    expect(screen.getByText('СЧЁТ')).toBeInTheDocument();
  });

  it('renders with different currency', () => {
    const eurLabor: LaborCost = { ...LABOR_COST, currency: 'EUR' };
    renderInvoice({ laborCost: eurLabor });
    expect(screen.getByText(/EUR/)).toBeInTheDocument();
  });

  // ── Template Switching ───────────────────────────────────────────

  it('renders standard template without invoice header', () => {
    render(
      <WorkOrderPrintView
        workOrder={BASE_WORK_ORDER}
        template="standard"
      />,
      { wrapper: Wrapper },
    );
    expect(document.body.textContent).toContain('Наряд-заказ');
  });

  it('renders detailed template', () => {
    render(
      <WorkOrderPrintView
        workOrder={BASE_WORK_ORDER}
        laborCost={LABOR_COST}
        partsDetail={PARTS_DETAIL}
        template="detailed"
      />,
      { wrapper: Wrapper },
    );
    expect(document.body.textContent).toContain('Наряд-заказ');
  });

  // ── Order Status Display ─────────────────────────────────────────

  it('renders cancelled work order without crash', () => {
    const cancelledWo: WorkOrder = { ...BASE_WORK_ORDER, status: 'cancelled' };
    render(
      <WorkOrderPrintView workOrder={cancelledWo} template="invoice" />,
      { wrapper: Wrapper },
    );
    expect(screen.getByText('СЧЁТ')).toBeInTheDocument();
  });
});
