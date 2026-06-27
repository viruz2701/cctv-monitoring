// ═══════════════════════════════════════════════════════════════════════
// DeviceWizard — Unit Tests
// P1-QA.8: Coverage 80%
// ═══════════════════════════════════════════════════════════════════════

import React from 'react';
import { describe, it, expect, vi, beforeEach, afterEach } from 'vitest';
import { render, screen, fireEvent, waitFor, cleanup } from '@testing-library/react';
import userEvent from '@testing-library/user-event';
import { BrowserRouter } from 'react-router-dom';
import { DeviceWizard } from '../DeviceWizard';

// ── Mocks ────────────────────────────────────────────────────────────

// Mock react-router-dom navigate
const mockNavigate = vi.fn();
vi.mock('react-router-dom', async () => {
  const actual = await vi.importActual('react-router-dom');
  return {
    ...actual,
    useNavigate: () => mockNavigate,
  };
});

// Mock API services
const mockDetectDevice = vi.fn();
const mockSearchCameraModels = vi.fn();
const mockGetCameraSpecs = vi.fn();
const mockCalculateDeviceCapacity = vi.fn();
const mockCreateWorkOrder = vi.fn();

vi.mock('../../../services/api', () => ({
  api: {
    detectDevice: (...args: any[]) => mockDetectDevice(...args),
    searchCameraModels: (...args: any[]) => mockSearchCameraModels(...args),
    getCameraSpecs: (...args: any[]) => mockGetCameraSpecs(...args),
    calculateDeviceCapacity: (...args: any[]) => mockCalculateDeviceCapacity(...args),
  },
}));

vi.mock('../../../services/workOrdersApi', () => ({
  workOrdersApi: {
    createWorkOrder: (...args: any[]) => mockCreateWorkOrder(...args),
  },
}));

// Mock hooks
vi.mock('../../../hooks/useApiQuery', () => ({
  useSites: () => ({
    data: [
      { id: 'site-1', name: 'Main Office' },
      { id: 'site-2', name: 'Branch Office' },
    ],
    isLoading: false,
  }),
}));

vi.mock('../../../hooks/useReducedMotion', () => ({
  useReducedMotion: () => false,
}));

vi.mock('../../../hooks/useRipple', () => ({
  useRipple: () => ({ createRipple: vi.fn(), ripples: null }),
}));

vi.mock('../../../hooks/useHapticFeedback', () => ({
  useHapticFeedback: () => ({ light: vi.fn(), medium: vi.fn(), heavy: vi.fn() }),
}));

// Mock UI components that have complex dependencies
vi.mock('../../ui/Button', () => ({
  Button: ({ children, onClick, disabled, icon, variant, size, loading }: any) => (
    <button
      data-testid="mock-button"
      data-variant={variant}
      data-size={size}
      data-loading={loading}
      disabled={disabled}
      onClick={onClick}
    >
      {icon}{children}
    </button>
  ),
}));

vi.mock('../../ui/Input', () => ({
  Input: (props: any) => <input data-testid="mock-input" {...props} />,
  Select: ({ value, onChange, options, label }: any) => (
    <div>
      {label && <label>{label}</label>}
      <select data-testid="mock-select" value={value} onChange={onChange}>
        {options?.map((opt: any) => (
          <option key={opt.value} value={opt.value}>{opt.label}</option>
        ))}
      </select>
    </div>
  ),
}));

vi.mock('../../ui/Card', () => ({
  Card: ({ children, className }: any) => (
    <div data-testid="mock-card" className={className}>{children}</div>
  ),
  CardBody: ({ children, className }: any) => (
    <div data-testid="mock-card-body" className={className}>{children}</div>
  ),
}));

vi.mock('../../ui/Toast', () => ({
  useToast: () => ({
    success: vi.fn(),
    error: vi.fn(),
  }),
}));

vi.mock('../../ui/QRCode', () => ({
  QRCode: ({ value, size, label }: any) => (
    <div data-testid="qr-code" data-value={value} data-size={size}>
      QR: {label}
    </div>
  ),
}));

vi.mock('../../ui/ProgressBar', () => ({
  ProgressBar: ({ value, max }: any) => (
    <div data-testid="progress-bar" data-value={value} data-max={max}>
      {value}/{max}
    </div>
  ),
}));

// ── Helpers ──────────────────────────────────────────────────────────

function renderWizard() {
  const onClose = vi.fn();
  const onComplete = vi.fn();
  const utils = render(
    <BrowserRouter>
      <DeviceWizard onClose={onClose} onComplete={onComplete} />
    </BrowserRouter>,
  );
  return { ...utils, onClose, onComplete };
}

const mockDetectionResult = {
  detected: true,
  model: 'DS-2CD2386G2-I',
  vendor: 'hikvision',
  firmware: '5.7.0',
  mac_address: 'AA:BB:CC:DD:EE:FF',
  ip: '192.168.1.100',
  onvif_profile_s: true,
  onvif_profile_t: true,
  rtsp_supported: true,
  http_api_supported: true,
  protocols: ['ONVIF', 'RTSP', 'HTTP'],
  stream_urls: ['rtsp://192.168.1.100:554/stream1'],
  error: null,
};

const mockCameraModel = {
  models: [
    { id: 'cam-1', brand: 'Hikvision', model: 'DS-2CD2386G2-I', type: 'bullet', resolution: '4K' },
    { id: 'cam-2', brand: 'Dahua', model: 'IPC-HFW5842', type: 'bullet', resolution: '4K' },
  ],
};

const mockCameraSpecs = {
  brand: 'Hikvision',
  model: 'DS-2CD2386G2-I',
  type: 'bullet',
  resolution: '4K',
  max_fps: 30,
  lens_mm: '6mm',
  poe_class: '802.3at',
  power_watts: 12.95,
  infrared: true,
  protocols: ['ONVIF', 'RTSP'],
  outdoor_rating: 'IP67',
};

const mockCapacityResult = {
  bandwidth_mbps: 48,
  storage_gb: 1500,
  poe_budget_watts: 51.8,
  recommended_nvr: 'Mid-range NVR',
  warnings: [],
};

// ── Tests ────────────────────────────────────────────────────────────

describe('DeviceWizard', () => {
  beforeEach(() => {
    vi.clearAllMocks();
  });

  afterEach(() => {
    cleanup();
  });

  // ── Rendering ─────────────────────────────────────────────────────

  it('Должен отрендерить все 5 шагов в прогресс-баре', () => {
    renderWizard();
    expect(screen.getByText('Step 1 of 5')).toBeDefined();
    expect(screen.getAllByText('IP / Auto-Detect').length).toBe(2);
    expect(screen.getByText('Compatibility')).toBeDefined();
    expect(screen.getByText('Capacity')).toBeDefined();
    expect(screen.getByText('QR Code')).toBeDefined();
    expect(screen.getByText('Work Order')).toBeDefined();
  });

  it('Должен иметь aria-label на диалоге', () => {
    renderWizard();
    expect(screen.getByRole('dialog')).toHaveAttribute(
      'aria-label',
      'Smart Device Onboarding Wizard',
    );
  });

  it('Должен показать 20% прогресс на первом шаге', () => {
    renderWizard();
    expect(screen.getByText('20%')).toBeDefined();
  });

  // ── Step 1: IP / Auto-Detect ──────────────────────────────────────

  it('Должен показать поле ввода IP-адреса', () => {
    renderWizard();
    expect(screen.getByPlaceholderText(/192\.168\.1\.100/)).toBeDefined();
  });

  it('Должен показать поля для username/password', () => {
    renderWizard();
    expect(screen.getByPlaceholderText('admin')).toBeDefined();
    expect(screen.getByPlaceholderText('••••••••')).toBeDefined();
  });

  it('Должен деактивировать кнопку Detect при пустом IP', () => {
    renderWizard();
    const detectBtn = screen.getByText('Detect Device').closest('button');
    expect(detectBtn).toBeDisabled();
  });

  it('Должен активировать кнопку Detect при введённом IP', async () => {
    renderWizard();
    const input = screen.getByPlaceholderText(/192\.168\.1\.100/);
    await userEvent.type(input, '192.168.1.50');
    const detectBtn = screen.getByText('Detect Device').closest('button');
    expect(detectBtn).not.toBeDisabled();
  });

  it('Должен вызвать api.detectDevice при нажатии Detect', async () => {
    mockDetectDevice.mockResolvedValueOnce(mockDetectionResult);
    renderWizard();
    const input = screen.getByPlaceholderText(/192\.168\.1\.100/);
    await userEvent.type(input, '192.168.1.100');
    fireEvent.click(screen.getByText('Detect Device'));
    await waitFor(() => {
      expect(mockDetectDevice).toHaveBeenCalledWith('192.168.1.100', {
        username: undefined,
        password: undefined,
      });
    });
  });

  it('Должен показать информацию об устройстве после успешного детекта', async () => {
    mockDetectDevice.mockResolvedValueOnce(mockDetectionResult);
    renderWizard();
    const input = screen.getByPlaceholderText(/192\.168\.1\.100/);
    await userEvent.type(input, '192.168.1.100');
    fireEvent.click(screen.getByText('Detect Device'));
    await waitFor(() => {
      expect(screen.getByText('DS-2CD2386G2-I')).toBeDefined();
      expect(screen.getByText('Device detected successfully')).toBeDefined();
    });
  });

  it('Должен показать протоколы ONVIF/RTSP для обнаруженного устройства', async () => {
    mockDetectDevice.mockResolvedValueOnce(mockDetectionResult);
    renderWizard();
    const input = screen.getByPlaceholderText(/192\.168\.1\.100/);
    await userEvent.type(input, '192.168.1.100');
    fireEvent.click(screen.getByText('Detect Device'));
    await waitFor(() => {
      expect(screen.getByText('ONVIF Profile S')).toBeDefined();
      expect(screen.getByText('ONVIF Profile T')).toBeDefined();
      expect(screen.getByText('RTSP')).toBeDefined();
    });
  });

  it('Должен показать ошибку при неудачном детекте', async () => {
    mockDetectDevice.mockResolvedValueOnce({
      detected: false,
      error: 'Device not found on this IP',
    });
    renderWizard();
    const input = screen.getByPlaceholderText(/192\.168\.1\.100/);
    await userEvent.type(input, '192.168.1.200');
    fireEvent.click(screen.getByText('Detect Device'));
    await waitFor(() => {
      expect(screen.getByText('Device not found on this IP')).toBeDefined();
    });
  });

  it('Должен обработать ошибку сети при детекте', async () => {
    mockDetectDevice.mockRejectedValueOnce(new Error('Network error'));
    renderWizard();
    const input = screen.getByPlaceholderText(/192\.168\.1\.100/);
    await userEvent.type(input, '192.168.1.100');
    fireEvent.click(screen.getByText('Detect Device'));
    await waitFor(() => {
      expect(screen.getByText('Network error')).toBeDefined();
    });
  });

  // ── Step 1: Camera Search ─────────────────────────────────────────

  it('Должен показать секцию Camera Database', () => {
    renderWizard();
    expect(screen.getByText('Camera Database')).toBeDefined();
  });

  it('Должен выполнить поиск камер по запросу', async () => {
    mockSearchCameraModels.mockResolvedValueOnce(mockCameraModel);
    renderWizard();
    const searchInput = screen.getByPlaceholderText(/Search by brand/);
    await userEvent.type(searchInput, 'Hikvision');
    fireEvent.click(screen.getByText('Search'));
    await waitFor(() => {
      expect(mockSearchCameraModels).toHaveBeenCalledWith('Hikvision');
    });
  });

  it('Должен показать результаты поиска камер', async () => {
    mockSearchCameraModels.mockResolvedValueOnce(mockCameraModel);
    renderWizard();
    const searchInput = screen.getByPlaceholderText(/Search by brand/);
    await userEvent.type(searchInput, 'Hikvision');
    fireEvent.click(screen.getByText('Search'));
    await waitFor(() => {
      expect(screen.getByText('DS-2CD2386G2-I')).toBeDefined();
    });
  });

  // ── Step 2: Compatibility ─────────────────────────────────────────

  it('Должен перейти на шаг 2 после детекта и показать проверку совместимости', async () => {
    mockDetectDevice.mockResolvedValueOnce(mockDetectionResult);
    renderWizard();
    const input = screen.getByPlaceholderText(/192\.168\.1\.100/);
    await userEvent.type(input, '192.168.1.100');
    fireEvent.click(screen.getByText('Detect Device'));
    await waitFor(() => expect(screen.getByText('Device detected successfully')).toBeDefined());
    const nextBtn = screen.getByText('Next').closest('button');
    expect(nextBtn).not.toBeDisabled();
    fireEvent.click(nextBtn!);
    await waitFor(() => {
      expect(screen.getByText('Protocol Compatibility Check')).toBeDefined();
    });
  });

  it('Должен подтвердить совместимость по кнопке Confirm Compatibility', async () => {
    mockDetectDevice.mockResolvedValueOnce(mockDetectionResult);
    renderWizard();
    const input = screen.getByPlaceholderText(/192\.168\.1\.100/);
    await userEvent.type(input, '192.168.1.100');
    fireEvent.click(screen.getByText('Detect Device'));
    await waitFor(() => expect(screen.getByText('Device detected successfully')).toBeDefined());
    fireEvent.click(screen.getByText('Next').closest('button')!);
    await waitFor(() => expect(screen.getByText('Confirm Compatibility')).toBeDefined());
    fireEvent.click(screen.getByText('Confirm Compatibility'));
    await waitFor(() => {
      expect(screen.getByText('Compatibility verified')).toBeDefined();
    });
  });

  // ── Step 3: Capacity ──────────────────────────────────────────────

  it('Должен рассчитать capacity при нажатии Calculate Capacity', async () => {
    mockCalculateDeviceCapacity.mockResolvedValueOnce(mockCapacityResult);
    mockDetectDevice.mockResolvedValueOnce(mockDetectionResult);
    renderWizard();
    const input = screen.getByPlaceholderText(/192\.168\.1\.100/);
    await userEvent.type(input, '192.168.1.100');
    fireEvent.click(screen.getByText('Detect Device'));
    await waitFor(() => expect(screen.getByText('Device detected successfully')).toBeDefined());
    fireEvent.click(screen.getByText('Next').closest('button')!);
    await waitFor(() => expect(screen.getByText('Confirm Compatibility')).toBeDefined());
    fireEvent.click(screen.getByText('Confirm Compatibility'));
    fireEvent.click(screen.getByText('Next').closest('button')!);
    await waitFor(() => expect(screen.getByText('Calculate Capacity')).toBeDefined());
    fireEvent.click(screen.getByText('Calculate Capacity'));
    await waitFor(() => {
      expect(screen.getByText('48')).toBeDefined();
      expect(screen.getByText('1500')).toBeDefined();
    });
  });

  it('Должен использовать локальный fallback при ошибке API capacity', async () => {
    mockCalculateDeviceCapacity.mockRejectedValueOnce(new Error('API error'));
    mockDetectDevice.mockResolvedValueOnce(mockDetectionResult);
    renderWizard();
    const input = screen.getByPlaceholderText(/192\.168\.1\.100/);
    await userEvent.type(input, '192.168.1.100');
    fireEvent.click(screen.getByText('Detect Device'));
    await waitFor(() => expect(screen.getByText('Device detected successfully')).toBeDefined());
    fireEvent.click(screen.getByText('Next').closest('button')!);
    await waitFor(() => expect(screen.getByText('Confirm Compatibility')).toBeDefined());
    fireEvent.click(screen.getByText('Confirm Compatibility'));
    fireEvent.click(screen.getByText('Next').closest('button')!);
    await waitFor(() => expect(screen.getByText('Calculate Capacity')).toBeDefined());
    fireEvent.click(screen.getByText('Calculate Capacity'));
    await waitFor(() => {
      expect(screen.getByText(/Mid-range NVR/)).toBeDefined();
    });
  });

  // ── Navigation ─────────────────────────────────────

  it('Должен вызвать onClose при нажатии Cancel на первом шаге', () => {
    const { onClose } = renderWizard();
    fireEvent.click(screen.getByText('Cancel'));
    expect(onClose).toHaveBeenCalled();
  });
});
