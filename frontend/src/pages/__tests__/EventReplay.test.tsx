// ═══════════════════════════════════════════════════════════════════════
// EventReplay.test.tsx — Unit tests for EventReplay page
// P1-EVENTS: Browse, filter, and replay NATS JetStream events
// ═══════════════════════════════════════════════════════════════════════

import React from 'react';
import { render, screen, waitFor } from '@testing-library/react';
import userEvent from '@testing-library/user-event';
import { describe, it, expect, vi, beforeEach } from 'vitest';
import { EventReplay } from '../EventReplay';
import { request } from '../../services/api';

// ── Mocks ──────────────────────────────────────────────────────────────

vi.mock('../../services/api', () => ({
  request: vi.fn(),
}));

vi.mock('react-i18next', () => ({
  useTranslation: () => ({
    t: (key: string, fallback?: string) => fallback ?? key,
    i18n: { language: 'ru' },
  }),
}));

// Mock Toast — заглушка для useToast, возвращаемая из index.ts
vi.mock('../../components/ui', async () => {
  const actual = await vi.importActual('../../components/ui');
  return {
    ...actual,
    useToast: () => ({
      success: vi.fn(),
      error: vi.fn(),
      info: vi.fn(),
      warning: vi.fn(),
    }),
  };
});

// ── Test data ──────────────────────────────────────────────────────────

const mockStreams = [
  {
    name: 'camera.events',
    description: 'Camera motion events',
    subjects: 'camera.events.>',
    msg_count: 15234,
    byte_size: 104857600,
    max_age: '7d',
    storage: 'file',
    retention: 'limits',
  },
  {
    name: 'system.alerts',
    description: 'System alerts',
    subjects: 'system.alerts.>',
    msg_count: 892,
    byte_size: 204800,
    max_age: '30d',
    storage: 'memory',
    retention: 'interest',
  },
];

const mockMessages = [
  {
    seq: 1,
    subject: 'camera.events.motion',
    data: { camera_id: 'CAM-001', event: 'motion_detected' },
    timestamp: '2026-06-29T10:00:00Z',
    stream_name: 'camera.events',
  },
  {
    seq: 2,
    subject: 'camera.events.alert',
    data: { camera_id: 'CAM-002', event: 'line_cross' },
    timestamp: '2026-06-29T10:01:00Z',
    stream_name: 'camera.events',
  },
];

// ── Tests ──────────────────────────────────────────────────────────────

describe('EventReplay', () => {
  beforeEach(() => {
    vi.clearAllMocks();
    // Сброс default mock для request
    (request as ReturnType<typeof vi.fn>).mockResolvedValue({ streams: mockStreams });
  });

  // ── 1. Loads streams on mount ───────────────────────────────────────
  it('loads streams on mount and displays stream name', async () => {
    render(<EventReplay />);

    // Проверяем что API вызван с правильным endpoint
    await waitFor(() => {
      expect(request).toHaveBeenCalledWith('/events/streams');
    });

    // Проверяем отображение имени потока
    expect(await screen.findByText('camera.events')).toBeInTheDocument();
    expect(await screen.findByText('system.alerts')).toBeInTheDocument();
  });

  // ── 2. Shows loading state ──────────────────────────────────────────
  it('shows loading skeleton during streams fetch', async () => {
    // Не резолвим промис — держим loading состояние
    (request as ReturnType<typeof vi.fn>).mockReturnValue(new Promise(() => {}));

    const { container } = render(<EventReplay />);

    // Loader2 иконка с animate-spin классом отображается в CardHeader
    const spinner = container.querySelector('.animate-spin');
    expect(spinner).toBeInTheDocument();
  });

  // ── 3. Handles error state ──────────────────────────────────────────
  it('shows error message when API call fails', async () => {
    const errorMessage = 'Network error: failed to fetch streams';
    (request as ReturnType<typeof vi.fn>).mockRejectedValue(new Error(errorMessage));

    render(<EventReplay />);

    // Проверяем что отображается сообщение об ошибке
    expect(await screen.findByText(errorMessage)).toBeInTheDocument();

    // Проверяем кнопку Retry
    const retryButton = screen.getByRole('button', { name: /retry|common\.retry/i });
    expect(retryButton).toBeInTheDocument();
  });

  // ── 4. Shows empty state when no streams ────────────────────────────
  it('shows empty state when no streams returned', async () => {
    (request as ReturnType<typeof vi.fn>).mockResolvedValue({ streams: [] });

    render(<EventReplay />);

    // Ждём завершения загрузки
    await waitFor(() => {
      expect(request).toHaveBeenCalledWith('/events/streams');
    });

    // EmptyState с сообщением о пустом списке стримов
    expect(await screen.findByText('No streams found')).toBeInTheDocument();
    expect(await screen.findByText('No JetStream streams are configured.')).toBeInTheDocument();
  });

  // ── 5. Selecting stream loads messages ──────────────────────────────
  it('loads messages when stream is selected', async () => {
    // Для streams возвращаем список, для messages возвращаем сообщения
    (request as ReturnType<typeof vi.fn>).mockImplementation((url: string) => {
      if (url === '/events/streams') {
        return Promise.resolve({ streams: mockStreams });
      }
      if (url.startsWith('/events/streams/camera.events/messages')) {
        return Promise.resolve({
          messages: mockMessages,
          stream: 'camera.events',
          limit: 50,
          offset: 0,
        });
      }
      return Promise.resolve({});
    });

    render(<EventReplay />);

    // Дожидаемся появления stream name
    const streamButton = await screen.findByText('camera.events');
    expect(streamButton).toBeInTheDocument();

    // Кликаем по stream
    const user = userEvent.setup();
    await user.click(streamButton);

    // Проверяем что API вызван для получения сообщений
    await waitFor(() => {
      expect(request).toHaveBeenCalledWith(
        expect.stringContaining('/events/streams/camera.events/messages'),
      );
    });

    // Проверяем отображение сообщений
    expect(await screen.findByText('#1')).toBeInTheDocument();
    expect(await screen.findByText('#2')).toBeInTheDocument();
  });

  // ── 6. Replay modal opens and confirms ──────────────────────────────
  it('opens replay modal and confirms replay', async () => {
    (request as ReturnType<typeof vi.fn>).mockImplementation((url: string) => {
      if (url === '/events/streams') {
        return Promise.resolve({ streams: mockStreams });
      }
      if (url.startsWith('/events/streams/camera.events/messages')) {
        return Promise.resolve({
          messages: mockMessages,
          stream: 'camera.events',
          limit: 50,
          offset: 0,
        });
      }
      // Для replay endpoint
      if (url.includes('/replay/')) {
        return Promise.resolve({ status: 'ok' });
      }
      return Promise.resolve({});
    });

    render(<EventReplay />);

    // Кликаем по stream для загрузки сообщений
    const streamButton = await screen.findByText('camera.events');
    const user = userEvent.setup();
    await user.click(streamButton);

    // Ждём появления сообщений
    await screen.findByText('#1');

    // Кликаем по первой кнопке Replay в колонке actions (title="Replay")
    const actionReplayBtn = screen.getAllByTitle('Replay')[0];
    await user.click(actionReplayBtn);

    // Модальное окно подтверждения с заголовком
    expect(await screen.findByText('Replay Message')).toBeInTheDocument();

    // Находим кнопку подтверждения в модалке
    // Modal использует createPortal, findAllByRole находит все кнопки
    // Берём последнюю — она в модальном окне
    const allReplayBtns = screen.getAllByRole('button', { name: /^Replay$/ });
    const confirmBtn = allReplayBtns[allReplayBtns.length - 1];
    await user.click(confirmBtn);

    // Проверяем вызов API replay c правильными параметрами
    await waitFor(() => {
      expect(request).toHaveBeenCalledWith(
        expect.stringContaining('/events/streams/camera.events/replay/1'),
        expect.objectContaining({ method: 'POST' }),
      );
    });
  });
});
