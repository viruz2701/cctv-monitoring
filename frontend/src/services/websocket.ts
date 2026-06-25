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

export function useAlarmWebSocket() {
    const { token, user } = useAuth();
    const toast = useToast();
    const wsRef = useRef<WebSocket | null>(null);
    const reconnectTimeoutRef = useRef<ReturnType<typeof setTimeout> | null>(null);
    const reconnectAttempts = useRef(0);
    const maxReconnectDelay = 30000; // 30 seconds
    const scheduleReconnectRef = useRef<() => void>(() => {});

    const connect = useCallback(() => {
        if (!token || !user) return;

        const wsUrl = `${getWsBaseUrl()}/api/v1/ws/alarms?token=${encodeURIComponent(token)}`;
        const ws = new WebSocket(wsUrl);

        ws.onopen = () => {
            console.log('WebSocket connected');
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

        ws.onclose = () => {
            console.log('WebSocket closed');
            wsRef.current = null;
            scheduleReconnectRef.current();
        };

        ws.onerror = (error) => {
            console.error('WebSocket error', error);
            ws.close();
        };

        wsRef.current = ws;
    }, [token, user, toast]);

    const scheduleReconnect = useCallback(() => {
        if (reconnectTimeoutRef.current) {
            clearTimeout(reconnectTimeoutRef.current);
        }

        const delay = Math.min(1000 * Math.pow(2, reconnectAttempts.current), maxReconnectDelay);
        reconnectAttempts.current += 1;

        console.log(`Scheduling WebSocket reconnect in ${delay}ms`);
        reconnectTimeoutRef.current = setTimeout(() => {
            connect();
        }, delay);
    }, [connect]);

    // Update the ref whenever scheduleReconnect changes
    useEffect(() => {
        scheduleReconnectRef.current = scheduleReconnect;
    }, [scheduleReconnect]);

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
