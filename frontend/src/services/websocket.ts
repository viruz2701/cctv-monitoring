import { useEffect, useRef, useCallback } from 'react';
import { useQueryClient } from '@tanstack/react-query';
import { queryKeys } from '../hooks/useApiQuery';
import { useToast } from '../components/ui/Toast';
import { useAuth } from '../hooks/useAuth';
import type { Alert } from '../types';

const getWsBaseUrl = () => {
    if (import.meta.env.VITE_WS_URL) {
        return import.meta.env.VITE_WS_URL.replace('http', 'ws');
    }
    const protocol = window.location.protocol === 'https:' ? 'wss:' : 'ws:';
    return `${protocol}//${window.location.host}`;
};

// Коды закрытия WebSocket, указывающие на ошибку авторизации — reconnect бесполезен
// 1006 — Abnormal Closure (сеть, таймаут прокси): НЕ auth-ошибка, reconnect возможен
// 1001 — Going Away (сервер уходит): НЕ auth-ошибка, reconnect возможен
// 4001, 4003, 4004 — кастомные коды с сервера: реальные auth-ошибки
const AUTH_CLOSE_CODES = new Set([4001, 4003, 4004]);

const MAX_RECONNECT_ATTEMPTS = 5;

export function useAlarmWebSocket() {
    const { token, user } = useAuth();
    const toast = useToast();
    const wsRef = useRef<WebSocket | null>(null);
    const reconnectTimeoutRef = useRef<ReturnType<typeof setTimeout> | null>(null);
    const reconnectAttempts = useRef(0);
    const maxReconnectDelay = 30000; // 30 seconds
    const scheduleReconnectRef = useRef<() => void>(() => {});
    const wsSupportedRef = useRef(true);

    const connect = useCallback(() => {
        if (!token || !user || !wsSupportedRef.current) return;

        const wsUrl = `${getWsBaseUrl()}/api/v1/ws/alarms?token=${encodeURIComponent(token)}`;
        const ws = new WebSocket(wsUrl);

        ws.onopen = () => {
            reconnectAttempts.current = 0;
        };

        ws.onmessage = (event) => {
            try {
                const data = JSON.parse(event.data);
                if (data.type === 'alarm' && data.alarm) {
                    toast.error(data.alarm.description || `Alarm triggered on device ${data.alarm.device_id}`);
                    // Alarm data comes from useAlarms() React Query hook on next refetch
                }
            } catch (err) {
                console.error('Failed to parse WebSocket message', err);
            }
        };

        ws.onclose = (event) => {
            wsRef.current = null;

            // Если код закрытия указывает на ошибку авторизации — не реконнектимся
            if (AUTH_CLOSE_CODES.has(event.code)) {
                console.warn(`WebSocket closed with auth error code ${event.code} — disabling reconnection`);
                wsSupportedRef.current = false;
                return;
            }

            if (reconnectAttempts.current >= MAX_RECONNECT_ATTEMPTS) {
                console.warn(`WebSocket max reconnect attempts (${MAX_RECONNECT_ATTEMPTS}) reached — disabling`);
                wsSupportedRef.current = false;
                return;
            }

            scheduleReconnectRef.current();
        };

        ws.onerror = () => {
            console.error('WebSocket error');
            // Не вызываем ws.close() — onclose сработает автоматически
        };

        wsRef.current = ws;
    }, [token, user, toast]);

    const scheduleReconnect = useCallback(() => {
        if (reconnectTimeoutRef.current) {
            clearTimeout(reconnectTimeoutRef.current);
        }

        const delay = Math.min(1000 * Math.pow(2, reconnectAttempts.current), maxReconnectDelay);
        reconnectAttempts.current += 1;

        reconnectTimeoutRef.current = setTimeout(() => {
            connect();
        }, delay);
    }, [connect]);

    // Update the ref whenever scheduleReconnect changes
    useEffect(() => {
        scheduleReconnectRef.current = scheduleReconnect;
    }, [scheduleReconnect]);

    // Сброс флага wsSupported при смене токена (пользователь перелогинился)
    useEffect(() => {
        wsSupportedRef.current = true;
    }, [token]);

    useEffect(() => {
        if (token && user) {
            connect();
        } else {
            if (wsRef.current) {
                wsRef.current.close();
                wsRef.current = null;
            }
            if (reconnectTimeoutRef.current) {
                clearTimeout(reconnectTimeoutRef.current);
                reconnectTimeoutRef.current = null;
            }
            reconnectAttempts.current = 0;
        }

        return () => {
            if (wsRef.current) {
                wsRef.current.close();
                wsRef.current = null;
            }
            if (reconnectTimeoutRef.current) {
                clearTimeout(reconnectTimeoutRef.current);
                reconnectTimeoutRef.current = null;
            }
        };
    }, [token, user, connect]);
}
