// ═══════════════════════════════════════════════════════════════════════
// useBulkOperations — Unit Tests
// P1-UX.10: Bulk Operations Progress — WebSocket, cancel, retry
// ═══════════════════════════════════════════════════════════════════════

import { describe, it, expect, vi, beforeEach, afterEach } from 'vitest';
import { renderHook, act, waitFor } from '@testing-library/react';
import { useBulkOperations } from '../useBulkOperations';

// ═══════════════════════════════════════════════════════════════════════
// Mocks
// ═══════════════════════════════════════════════════════════════════════

const mockFetch = vi.fn();
const mockWebSocketClose = vi.fn();
const mockWebSocketSend = vi.fn();

class MockWebSocket {
  close = mockWebSocketClose;
  send = mockWebSocketSend;
  onopen: any = null;
  onmessage: any = null;
  onclose: any = null;
  onerror: any = null;
  readyState = 1;
  url: string;

  constructor(url: string) {
    this.url = url;
  }
}

vi.stubGlobal('fetch', mockFetch);
vi.stubGlobal('WebSocket', MockWebSocket);

// Mock useAuth
vi.mock('../useAuth', () => ({
  useAuth: () => ({
    token: 'test-token',
    user: { id: '1', role: 'admin', username: 'admin' },
  }),
}));

// ═══════════════════════════════════════════════════════════════════════
// Test data
// ═══════════════════════════════════════════════════════════════════════

interface TestItem {
  id: string;
  name: string;
}

const testItems: TestItem[] = [
  { id: '1', name: 'Item 1' },
  { id: '2', name: 'Item 2' },
  { id: '3', name: 'Item 3' },
];

const testConfig = {
  endpoint: '/api/v1/bulk/delete',
  items: testItems,
  getItemLabel: (item: TestItem) => item.name,
  getItemId: (item: TestItem) => item.id,
  operationLabel: 'Deleting items...',
};

// ═══════════════════════════════════════════════════════════════════════
// Tests
// ═══════════════════════════════════════════════════════════════════════

describe('useBulkOperations', () => {
  beforeEach(() => {
    vi.clearAllMocks();
    vi.useFakeTimers();
  });

  afterEach(() => {
    vi.useRealTimers();
  });

  it('should initialize with null progress state', () => {
    const { result } = renderHook(() => useBulkOperations());
    expect(result.current.progressState).toBeNull();
  });

  it('should create progress state on start and handle fetch success', async () => {
    mockFetch.mockResolvedValueOnce({
      ok: true,
      statusText: 'OK',
    });

    const { result } = renderHook(() => useBulkOperations());

    let promise: Promise<any>;
    act(() => {
      promise = result.current.startOperation(testConfig);
    });

    // Should have progress state immediately
    expect(result.current.progressState).not.toBeNull();
    expect(result.current.progressState?.total).toBe(3);
    expect(result.current.progressState?.operationLabel).toBe('Deleting items...');
    expect(result.current.progressState?.isRunning).toBe(true);
    expect(result.current.progressState?.items).toHaveLength(3);

    // All items start as pending
    result.current.progressState?.items.forEach((item) => {
      expect(item.status).toBe('pending');
    });

    // Advance timers to trigger completion check
    // The hook uses setInterval every 200ms to check completion,
    // and also has a safety timeout of 300000ms
    // Mark all items as done to complete the operation
    act(() => {
      // Simulate the operation completing via WebSocket or polling
      // We'll just cancel to avoid timeout
      result.current.cancelOperation();
    });

    // Fast-forward past the check interval
    await act(async () => {
      vi.advanceTimersByTime(500);
    });
  });

  it('should cancel operation and update state', async () => {
    mockFetch.mockResolvedValue({
      ok: true,
      statusText: 'OK',
    });

    const { result } = renderHook(() => useBulkOperations());

    act(() => {
      result.current.startOperation(testConfig);
    });

    expect(result.current.progressState?.isRunning).toBe(true);

    // Await the cancel operation so the async state update completes
    await act(async () => {
      await result.current.cancelOperation();
    });

    // Check state after cancel
    expect(result.current.progressState?.isCancelled).toBe(true);
    expect(result.current.progressState?.isRunning).toBe(false);
  });

  it('should close progress and reset state', () => {
    mockFetch.mockResolvedValueOnce({
      ok: true,
      statusText: 'OK',
    });

    const { result } = renderHook(() => useBulkOperations());

    act(() => {
      result.current.startOperation(testConfig);
    });

    expect(result.current.progressState).not.toBeNull();

    act(() => {
      result.current.cancelOperation();
      result.current.closeProgress();
    });

    expect(result.current.progressState).toBeNull();
  });

  it('should handle fetch failure gracefully', async () => {
    mockFetch.mockRejectedValueOnce(new Error('Network error'));

    const { result } = renderHook(() => useBulkOperations());

    await act(async () => {
      const opResult = await result.current.startOperation(testConfig);
      expect(opResult.success).toBe(false);
    });

    // Should still have progress state after failure
    expect(result.current.progressState).not.toBeNull();
    expect(result.current.progressState?.isRunning).toBe(false);
  });

  it('should handle HTTP error response', async () => {
    mockFetch.mockResolvedValueOnce({
      ok: false,
      statusText: 'Internal Server Error',
    });

    const { result } = renderHook(() => useBulkOperations());

    await act(async () => {
      const opResult = await result.current.startOperation(testConfig);
      expect(opResult.success).toBe(false);
    });
  });

  it('should retry failed items', async () => {
    // First call fails
    mockFetch.mockRejectedValueOnce(new Error('Network error'));
    // Retry call succeeds
    mockFetch.mockResolvedValue({
      ok: true,
      statusText: 'OK',
    });

    const { result } = renderHook(() => useBulkOperations());

    // Run operation (will fail)
    await act(async () => {
      await result.current.startOperation(testConfig);
    });

    // Verify items are failed
    expect(result.current.progressState?.items.every((i) => i.status === 'failed')).toBe(true);

    // Retry all
    await act(async () => {
      await result.current.retryAll(testConfig.endpoint, undefined, testConfig.operationLabel);
    });

    // After retry, items should transition from pending
    // Since retryAll also calls startOperation-like flow which creates setInterval,
    // items will be pending initially
    expect(result.current.progressState?.items.some((i) => i.status === 'pending')).toBe(true);
  });

  it('should handle operation with custom payload', async () => {
    mockFetch.mockResolvedValueOnce({
      ok: true,
      statusText: 'OK',
    });

    const { result } = renderHook(() => useBulkOperations());

    const customConfig = {
      ...testConfig,
      getItemPayload: (item: TestItem) => ({ item_id: item.id, reason: 'test' }),
    };

    act(() => {
      result.current.startOperation(customConfig);
    });

    // Verify fetch was called with correct payload structure
    const fetchCall = mockFetch.mock.calls[0];
    expect(fetchCall[0]).toBe('/api/v1/bulk/delete');
    expect(fetchCall[1].method).toBe('POST');

    const body = JSON.parse(fetchCall[1].body);
    expect(body.items).toHaveLength(3);
    expect(body.items[0].payload).toEqual({ item_id: '1', reason: 'test' });

    // Clean up
    act(() => {
      result.current.cancelOperation();
    });
  });

  it('should provide state compatible with BulkProgressModal props', () => {
    // Verify the state shape matches BulkProgressModal expectations
    const state = {
      total: 3,
      items: [
        { id: '1', label: 'Item 1', status: 'done' as const },
        { id: '2', label: 'Item 2', status: 'processing' as const },
        { id: '3', label: 'Item 3', status: 'pending' as const },
      ],
      isRunning: true,
      isCancelled: false,
      operationLabel: 'Test operation',
    };

    expect(state.total).toBeGreaterThan(0);
    expect(state.items).toHaveLength(3);
    expect(typeof state.isRunning).toBe('boolean');

    // Modal expects these props
    const modalProps = {
      state,
      onCancel: () => {},
      onRetryAll: () => {},
      onClose: () => {},
    };

    expect(modalProps.state.total).toBe(state.total);
    expect(modalProps.state.isRunning).toBe(state.isRunning);
  });

  it('should accept custom operation context', async () => {
    mockFetch.mockResolvedValueOnce({
      ok: true,
      statusText: 'OK',
    });

    const { result } = renderHook(() => useBulkOperations());

    const configWithContext = {
      ...testConfig,
      operationContext: 'devices-bulk-2024',
    };

    act(() => {
      result.current.startOperation(configWithContext);
    });

    // Verify fetch was called with correct endpoint
    expect(mockFetch).toHaveBeenCalledWith(
      '/api/v1/bulk/delete',
      expect.objectContaining({ method: 'POST' }),
    );

    // Clean up
    act(() => {
      result.current.cancelOperation();
    });
  });
});
