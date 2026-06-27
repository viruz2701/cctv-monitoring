// ═══════════════════════════════════════════════════════════════════════
// useBulkOperations — Hook для управления bulk-операциями с real-time
// прогрессом через WebSocket, отменой и retry.
//
// P1-UX.10: Bulk Operations Progress
//   - WebSocket для real-time updates
//   - Cancel logic
//   - Retry logic (individual + batch)
// ═══════════════════════════════════════════════════════════════════════

import { useState, useRef, useCallback, useEffect } from 'react';
import { useAuth } from './useAuth';
import type { BulkProgressState, BulkProgressItem, BulkItemStatus } from '../components/ui/BulkProgressModal';

// ═══════════════════════════════════════════════════════════════════════
// Types
// ═══════════════════════════════════════════════════════════════════════

export interface BulkOperationConfig<T> {
  /** Endpoint для выполнения операции (POST) */
  endpoint: string;
  /** Элементы для обработки */
  items: T[];
  /** Функция для получения label элемента */
  getItemLabel: (item: T) => string;
  /** Функция для получения ID элемента */
  getItemId: (item: T) => string;
  /** Название операции (для отображения) */
  operationLabel: string;
  /** Payload для каждого элемента (опционально) */
  getItemPayload?: (item: T) => unknown;
  /** Контекст для WebSocket-сообщений (чтобы отличать разные операции) */
  operationContext?: string;
  /** Обработчик WebSocket-сообщения для прогресса */
  onProgressMessage?: (data: unknown, currentItems: BulkProgressItem[]) => BulkProgressItem[];
}

export interface BulkOperationResult {
  success: boolean;
  completed: number;
  failed: number;
}

interface WebSocketProgressMessage {
  type: 'bulk_progress';
  context?: string;
  batch_id: string;
  item_id: string;
  status: BulkItemStatus;
  error?: string;
}

// ═══════════════════════════════════════════════════════════════════════
// WebSocket helpers
// ═══════════════════════════════════════════════════════════════════════

const getWsBaseUrl = () => {
  if (import.meta.env.VITE_WS_URL) {
    return import.meta.env.VITE_WS_URL.replace('http', 'ws');
  }
  const protocol = window.location.protocol === 'https:' ? 'wss:' : 'ws:';
  return `${protocol}//${window.location.host}`;
};

// ═══════════════════════════════════════════════════════════════════════
// Hook
// ═══════════════════════════════════════════════════════════════════════

export function useBulkOperations() {
  const { token } = useAuth();
  const [progressState, setProgressState] = useState<BulkProgressState | null>(null);
  const wsRef = useRef<WebSocket | null>(null);
  const cancelRef = useRef(false);
  const batchIdRef = useRef<string | null>(null);
  const pendingItemsRef = useRef<BulkProgressItem[]>([]);

  // ═══ Cleanup WebSocket on unmount ═══
  useEffect(() => {
    return () => {
      if (wsRef.current) {
        wsRef.current.close();
        wsRef.current = null;
      }
    };
  }, []);

  // ═══ Подключение WebSocket ═══
  const connectWebSocket = useCallback((batchId: string, context?: string) => {
    if (wsRef.current) {
      wsRef.current.close();
    }

    if (!token) return;

    const wsUrl = `${getWsBaseUrl()}/api/v1/ws/bulk?token=${encodeURIComponent(token)}`;
    const ws = new WebSocket(wsUrl);

    ws.onopen = () => {
      // Subscribe to batch updates
      ws.send(JSON.stringify({
        type: 'subscribe_batch',
        batch_id: batchId,
        context,
      }));
    };

    ws.onmessage = (event) => {
      try {
        const data = JSON.parse(event.data) as WebSocketProgressMessage;
        if (data.type !== 'bulk_progress') return;
        if (context && data.context !== context) return;

        setProgressState((prev) => {
          if (!prev) return prev;

          const updatedItems = prev.items.map((item) => {
            if (item.id === data.item_id) {
              return {
                ...item,
                status: data.status,
                error: data.error || item.error,
              };
            }
            return item;
          });

          // Update ref too for retry
          pendingItemsRef.current = updatedItems;

          return {
            ...prev,
            items: updatedItems,
            isRunning: data.status === 'processing' || updatedItems.some((i) => i.status === 'pending' || i.status === 'processing'),
          };
        });
      } catch {
        // Parse error — ignore
      }
    };

    ws.onerror = () => {
      // WebSocket error — continue with REST polling fallback
    };

    ws.onclose = () => {
      wsRef.current = null;
    };

    wsRef.current = ws;
  }, [token]);

  // ═══ Запуск bulk-операции ═══
  const startOperation = useCallback(async <T>(
    config: BulkOperationConfig<T>,
  ): Promise<BulkOperationResult> => {
    cancelRef.current = false;

    // Create initial progress state
    const items: BulkProgressItem[] = config.items.map((item) => ({
      id: config.getItemId(item),
      label: config.getItemLabel(item),
      status: 'pending' as BulkItemStatus,
    }));

    pendingItemsRef.current = items;

    const batchId = `batch-${Date.now()}-${Math.random().toString(36).slice(2, 9)}`;
    batchIdRef.current = batchId;

    setProgressState({
      total: config.items.length,
      items,
      isRunning: true,
      isCancelled: false,
      operationLabel: config.operationLabel,
    });

    // Connect to WebSocket for real-time updates
    connectWebSocket(batchId, config.operationContext);

    let completed = 0;
    let failed = 0;

    try {
      // Send bulk operation to backend
      const response = await fetch(config.endpoint, {
        method: 'POST',
        headers: {
          'Content-Type': 'application/json',
          ...(token ? { Authorization: `Bearer ${token}` } : {}),
        },
        body: JSON.stringify({
          batch_id: batchId,
          items: config.items.map((item) => ({
            id: config.getItemId(item),
            payload: config.getItemPayload ? config.getItemPayload(item) : undefined,
          })),
        }),
      });

      if (!response.ok) {
        throw new Error(`Bulk operation failed: ${response.statusText}`);
      }

      // If no WebSocket, poll for progress
      const pollInterval = setInterval(async () => {
        if (cancelRef.current || !batchIdRef.current) {
          clearInterval(pollInterval);
          return;
        }

        try {
          const pollResponse = await fetch(
            `/api/v1/bulk/${batchIdRef.current}/progress`,
            { headers: token ? { Authorization: `Bearer ${token}` } : {} },
          );
          if (pollResponse.ok) {
            const pollData = await pollResponse.json();
            if (pollData.items) {
              setProgressState((prev) => {
                if (!prev) return prev;
                const updatedItems = prev.items.map((item) => {
                  const updated = pollData.items.find(
                    (p: { id: string }) => p.id === item.id,
                  );
                  return updated
                    ? { ...item, status: updated.status as BulkItemStatus, error: updated.error }
                    : item;
                });
                pendingItemsRef.current = updatedItems;
                return { ...prev, items: updatedItems };
              });
            }
          }
        } catch {
          // Poll error — ignore
        }
      }, 1000);

      // Wait for completion via WebSocket or timeout
      await new Promise<void>((resolve) => {
        const checkCompletion = setInterval(() => {
          setProgressState((prev) => {
            if (!prev) return prev;

            const done = prev.items.filter(
              (i) => i.status === 'done' || i.status === 'failed' || i.status === 'cancelled',
            );
            const allDone = done.length >= prev.total;

            if (allDone || cancelRef.current) {
              clearInterval(checkCompletion);
              clearInterval(pollInterval);
              completed = prev.items.filter((i) => i.status === 'done').length;
              failed = prev.items.filter((i) => i.status === 'failed').length;

              // Close WebSocket
              if (wsRef.current) {
                wsRef.current.close();
                wsRef.current = null;
              }

              resolve();
            }

            return prev;
          });
        }, 200);

        // Safety timeout — 5 minutes
        setTimeout(() => {
          clearInterval(checkCompletion);
          clearInterval(pollInterval);
          if (wsRef.current) {
            wsRef.current.close();
            wsRef.current = null;
          }
          resolve();
        }, 300000);
      });

      return { success: failed === 0, completed, failed };
    } catch (err) {
      // Mark all pending as failed
      setProgressState((prev) => {
        if (!prev) return prev;
        const updatedItems = prev.items.map((item) => ({
          ...item,
          status: item.status === 'pending' ? 'failed' as BulkItemStatus : item.status,
          error: item.status === 'pending' ? (err instanceof Error ? err.message : 'Operation failed') : item.error,
        }));
        pendingItemsRef.current = updatedItems;
        return {
          ...prev,
          items: updatedItems,
          isRunning: false,
        };
      });

      return { success: false, completed, failed: items.length };
    }
  }, [token, connectWebSocket]);

  // ═══ Отмена операции ═══
  const cancelOperation = useCallback(async () => {
    cancelRef.current = true;

    // Notify backend
    if (batchIdRef.current) {
      try {
        await fetch(`/api/v1/bulk/${batchIdRef.current}/cancel`, {
          method: 'POST',
          headers: token ? { Authorization: `Bearer ${token}` } : {},
        });
      } catch {
        // Best-effort cancel
      }
    }

    setProgressState((prev) => {
      if (!prev) return prev;
      return {
        ...prev,
        isCancelled: true,
        isRunning: false,
        items: prev.items.map((item) => ({
          ...item,
          status: item.status === 'pending' || item.status === 'processing'
            ? 'cancelled' as BulkItemStatus
            : item.status,
        })),
      };
    });

    if (wsRef.current) {
      wsRef.current.close();
      wsRef.current = null;
    }
  }, [token]);

  // ═══ Retry всех упавших элементов ═══
  const retryAll = useCallback(async <T>(
    endpoint: string,
    getItemPayload?: (item: BulkProgressItem) => unknown,
    operationContext?: string,
  ): Promise<void> => {
    const failedItems = pendingItemsRef.current.filter((i) => i.status === 'failed');
    if (failedItems.length === 0) return;

    // Reset failed items to pending
    const resetItems = pendingItemsRef.current.map((item) => ({
      ...item,
      status: item.status === 'failed' ? 'pending' as BulkItemStatus : item.status,
      error: item.status === 'failed' ? undefined : item.error,
    }));
    pendingItemsRef.current = resetItems;

    const batchId = `batch-retry-${Date.now()}-${Math.random().toString(36).slice(2, 9)}`;
    batchIdRef.current = batchId;
    cancelRef.current = false;

    setProgressState((prev) => {
      if (!prev) return prev;
      return {
        ...prev,
        items: resetItems,
        isRunning: true,
        isCancelled: false,
      };
    });

    connectWebSocket(batchId, operationContext);

    try {
      await fetch(endpoint, {
        method: 'POST',
        headers: {
          'Content-Type': 'application/json',
          ...(token ? { Authorization: `Bearer ${token}` } : {}),
        },
        body: JSON.stringify({
          batch_id: batchId,
          items: failedItems.map((item) => ({
            id: item.id,
            payload: getItemPayload ? getItemPayload(item) : undefined,
          })),
        }),
      });
    } catch {
      // Retry submission failed
    }
  }, [token, connectWebSocket]);

  // ═══ Retry одного элемента ═══
  const retryItem = useCallback(async (
    itemId: string,
    endpoint: string,
    getItemPayload?: (item: BulkProgressItem) => unknown,
  ): Promise<void> => {
    const item = pendingItemsRef.current.find((i) => i.id === itemId);
    if (!item) return;

    // Mark as pending
    const updatedItems = pendingItemsRef.current.map((i) => ({
      ...i,
      status: i.id === itemId ? 'pending' as BulkItemStatus : i.status,
      error: i.id === itemId ? undefined : i.error,
    }));
    pendingItemsRef.current = updatedItems;
    setProgressState((prev) => {
      if (!prev) return prev;
      return { ...prev, items: updatedItems, isRunning: true };
    });

    try {
      const response = await fetch(endpoint, {
        method: 'POST',
        headers: {
          'Content-Type': 'application/json',
          ...(token ? { Authorization: `Bearer ${token}` } : {}),
        },
        body: JSON.stringify({
          batch_id: `retry-${Date.now()}`,
          items: [{
            id: item.id,
            payload: getItemPayload ? getItemPayload(item) : undefined,
          }],
        }),
      });

      if (response.ok) {
        // Poll for single item result
        const result = await response.json();
        if (result.items?.[0]) {
          const updated = result.items[0];
          setProgressState((prev) => {
            if (!prev) return prev;
            const finalItems = prev.items.map((i) => ({
              ...i,
              status: i.id === itemId ? (updated.status as BulkItemStatus) : i.status,
              error: i.id === itemId ? updated.error : i.error,
            }));
            pendingItemsRef.current = finalItems;
            return {
              ...prev,
              items: finalItems,
              isRunning: finalItems.some((i) => i.status === 'pending' || i.status === 'processing'),
            };
          });
        }
      }
    } catch {
      // Retry failed — mark back as failed
      setProgressState((prev) => {
        if (!prev) return prev;
        const finalItems = prev.items.map((i) => ({
          ...i,
          status: i.id === itemId ? 'failed' as BulkItemStatus : i.status,
          error: i.id === itemId ? 'Retry failed' : i.error,
        }));
        pendingItemsRef.current = finalItems;
        return { ...prev, items: finalItems };
      });
    }
  }, [token]);

  // ═══ Закрыть модалку ═══
  const closeProgress = useCallback(() => {
    setProgressState(null);
    if (wsRef.current) {
      wsRef.current.close();
      wsRef.current = null;
    }
    batchIdRef.current = null;
    pendingItemsRef.current = [];
  }, []);

  return {
    /** Текущее состояние прогресса (null, если нет активной операции) */
    progressState,
    /** Запустить новую bulk-операцию */
    startOperation,
    /** Отменить текущую операцию */
    cancelOperation,
    /** Повторить все упавшие элементы */
    retryAll,
    /** Повторить один элемент */
    retryItem,
    /** Закрыть модалку прогресса */
    closeProgress,
  };
}
