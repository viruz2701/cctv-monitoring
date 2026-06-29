// ═══════════════════════════════════════════════════════════════════════
// PlaybookMarketplace.test.tsx — Unit tests for PlaybookMarketplace page
// P1-MARKET: Card grid, filter by vendor/rating/verified, install
// ═══════════════════════════════════════════════════════════════════════

import React from 'react';
import { render, screen, waitFor } from '@testing-library/react';
import userEvent from '@testing-library/user-event';
import { describe, it, expect, vi, beforeEach } from 'vitest';
import { PlaybookMarketplace } from '../PlaybookMarketplace';
import { playbookMarketplaceApi } from '../../services/api/playbookMarketplace';

// ── Mocks ──────────────────────────────────────────────────────────────

vi.mock('../../services/api/playbookMarketplace', () => ({
  playbookMarketplaceApi: {
    list: vi.fn(),
    install: vi.fn(),
    rate: vi.fn(),
    share: vi.fn(),
  },
}));

vi.mock('react-i18next', () => ({
  useTranslation: () => ({
    t: (key: string, fallback?: string) => fallback ?? key,
    i18n: { language: 'ru' },
  }),
}));

// ── Test data ──────────────────────────────────────────────────────────

const mockPlaybooks = [
  {
    id: 'pb-001',
    name: 'Motion Detection Optimizer',
    description: 'Optimizes motion detection sensitivity for Hikvision cameras',
    vendor: 'hikvision' as const,
    version: '2.1.0',
    compat_matrix: ['DS-2CD2xxx', 'DS-2CD4xxx'],
    avg_rating: 4.5,
    review_count: 28,
    install_count: 143,
    verified: true,
    tenant_id: 'default',
    created_at: '2026-01-15T00:00:00Z',
    updated_at: '2026-06-01T00:00:00Z',
  },
  {
    id: 'pb-002',
    name: 'Line Crossing Detector',
    description: 'Advanced line crossing detection for Dahua cameras',
    vendor: 'dahua' as const,
    version: '1.8.0',
    compat_matrix: ['DH-IPC-HFW5xxx'],
    avg_rating: 4.2,
    review_count: 15,
    install_count: 89,
    verified: false,
    tenant_id: 'default',
    created_at: '2026-02-20T00:00:00Z',
    updated_at: '2026-05-15T00:00:00Z',
  },
];

const mockListResponse = {
  playbooks: mockPlaybooks,
  total: 2,
  limit: 20,
  offset: 0,
};

// ── Tests ──────────────────────────────────────────────────────────────

describe('PlaybookMarketplace', () => {
  beforeEach(() => {
    vi.clearAllMocks();
    (playbookMarketplaceApi.list as ReturnType<typeof vi.fn>).mockResolvedValue(mockListResponse);
    (playbookMarketplaceApi.install as ReturnType<typeof vi.fn>).mockResolvedValue({
      status: 'ok',
      message: 'Installed',
    });
    (playbookMarketplaceApi.rate as ReturnType<typeof vi.fn>).mockResolvedValue({
      status: 'ok',
      message: 'Rated',
    });
  });

  // ── 1. Loads playbooks on mount ─────────────────────────────────────
  it('loads playbooks on mount and displays cards', async () => {
    render(<PlaybookMarketplace />);

    // Проверяем вызов API list
    await waitFor(() => {
      expect(playbookMarketplaceApi.list).toHaveBeenCalledTimes(1);
    });

    // Проверяем отображение карточек playbook
    expect(await screen.findByText('Motion Detection Optimizer')).toBeInTheDocument();
    expect(await screen.findByText('Line Crossing Detector')).toBeInTheDocument();
  });

  // ── 2. Shows loading skeleton ───────────────────────────────────────
  it('shows loading skeleton while fetching playbooks', async () => {
    // Не резолвим промис — держим loading
    (playbookMarketplaceApi.list as ReturnType<typeof vi.fn>).mockReturnValue(new Promise(() => {}));

    const { container } = render(<PlaybookMarketplace />);

    // 6 skeleton cards с классом animate-pulse
    const skeletons = container.querySelectorAll('.animate-pulse');
    expect(skeletons.length).toBe(6);
  });

  // ── 3. Handles error state ──────────────────────────────────────────
  it('shows error message when API call fails', async () => {
    const errorMessage = 'Failed to load marketplace';
    (playbookMarketplaceApi.list as ReturnType<typeof vi.fn>).mockRejectedValue(
      new Error(errorMessage),
    );

    render(<PlaybookMarketplace />);

    // Проверяем отображение сообщения об ошибке
    expect(await screen.findByText(errorMessage)).toBeInTheDocument();
  });

  // ── 4. Empty state when no playbooks ────────────────────────────────
  it('shows empty state when no playbooks returned', async () => {
    (playbookMarketplaceApi.list as ReturnType<typeof vi.fn>).mockResolvedValue({
      playbooks: [],
      total: 0,
      limit: 20,
      offset: 0,
    });

    render(<PlaybookMarketplace />);

    // Empty state сообщение
    expect(await screen.findByText('No playbooks found')).toBeInTheDocument();
  });

  // ── 5. Vendor filter change triggers reload ──────────────────────────
  it('reloads playbooks when vendor filter changes', async () => {
    render(<PlaybookMarketplace />);

    // Ждём первую загрузку
    await waitFor(() => {
      expect(playbookMarketplaceApi.list).toHaveBeenCalledTimes(1);
    });

    // Находим select фильтра вендора
    const vendorSelect = screen.getByLabelText('Filter by vendor');
    expect(vendorSelect).toBeInTheDocument();

    // Меняем значение фильтра
    const user = userEvent.setup();
    await user.selectOptions(vendorSelect, 'hikvision');

    // Проверяем что API вызван снова с фильтром vendor
    await waitFor(() => {
      expect(playbookMarketplaceApi.list).toHaveBeenCalledWith(
        expect.objectContaining({ vendor: 'hikvision' }),
      );
    });
  });

  // ── 6. Install button triggers install ──────────────────────────────
  it('calls install API when Install button is clicked', async () => {
    render(<PlaybookMarketplace />);

    // Ждём загрузки карточек
    await screen.findByText('Motion Detection Optimizer');

    // Находим текст "Install" внутри кнопок карточек
    const installTexts = screen.getAllByText('Install');
    expect(installTexts.length).toBe(2);

    // Кликаем по тексту Install первой карточки — click всплывёт к button
    const user = userEvent.setup();
    await user.click(installTexts[0]);

    // Проверяем вызов API install с ID первого playbook
    await waitFor(() => {
      expect(playbookMarketplaceApi.install).toHaveBeenCalledWith('pb-001');
    });
  });
});
