// ═══════════════════════════════════════════════════════════════════════
// collaboration — WebSocket hook for real-time collaboration
//
// P3-NICE.1: Real-time Collaboration
//   - Presence indicators ("Ivan is editing this WO")
//   - Cursor sharing
//   - Conflict warnings
//   - Room-based subscriptions
//
// Использует существующий WebSocket-инфраструктуру (websocket.ts).
// ═══════════════════════════════════════════════════════════════════════

import { useEffect, useRef, useCallback, useState } from 'react';
import { useAuth } from '../hooks/useAuth';
import { useToast } from '../components/ui/Toast';
import { useQueryClient } from '@tanstack/react-query';

// ═══ Types ═════════════════════════════════════════════════════════════════

export interface PresenceInfo {
  user_id: string;
  user_name: string;
  resource: string;
  action: 'viewing' | 'editing';
  updated_at: string;
}

export interface CursorInfo {
  user_id: string;
  user_name: string;
  resource: string;
  field: string;
  offset: number;
  line: number;
  col: number;
}

export interface CursorPosition {
  line: number;
  col: number;
  field: string;
}

interface CollabMessage {
  type: 'presence' | 'cursor' | 'join' | 'leave' | 'presences_list' | 'cursors_list';
  resource?: string;
  payload?: Record<string, unknown>;
  presences?: PresenceInfo[];
  cursors?: CursorInfo[];
}

// ═══ WebSocket URL ═══════════════════════════════════════════════════════

const getWsBaseUrl = () => {
  if (import.meta.env.VITE_WS_URL) {
    return import.meta.env.VITE_WS_URL.replace('http', 'ws');
  }
  const protocol = window.location.protocol === 'https:' ? 'wss:' : 'ws:';
  return `${protocol}//${window.location.host}`;
};

// ═══ Hook ═════════════════════════════════════════════════════════════════

interface UseCollaborationOptions {
  resource: string; // e.g. "work_order:wo-123"
  onPresenceChange?: (presences: PresenceInfo[]) => void;
  onCursorChange?: (cursors: CursorInfo[]) => void;
  onConflict?: (users: string[]) => void;
}

interface UseCollaborationReturn {
  presences: PresenceInfo[];
  cursors: CursorInfo[];
  hasConflict: boolean;
  conflictUsers: string[];
  setEditing: (editing: boolean) => void;
  updateCursor: (pos: CursorPosition) => void;
  isConnected: boolean;
}

export function useCollaboration({
  resource,
  onPresenceChange,
  onCursorChange,
  onConflict,
}: UseCollaborationOptions): UseCollaborationReturn {
  const { token, user } = useAuth();
  const toast = useToast();
  const wsRef = useRef<WebSocket | null>(null);

  const [presences, setPresences] = useState<PresenceInfo[]>([]);
  const [cursors, setCursors] = useState<CursorInfo[]>([]);
  const [isConnected, setIsConnected] = useState(false);

  const editingRef = useRef(false);
  const heartbeatRef = useRef<ReturnType<typeof setInterval> | null>(null);

  // ── Send message ──────────────────────────────────────────────────
  const send = useCallback((msg: CollabMessage) => {
    if (wsRef.current?.readyState === WebSocket.OPEN) {
      wsRef.current.send(JSON.stringify(msg));
    }
  }, []);

  // ── Set editing state ─────────────────────────────────────────────
  const setEditing = useCallback(
    (editing: boolean) => {
      editingRef.current = editing;
      send({
        type: editing ? 'presence' : 'leave',
        resource,
        payload: { action: editing ? 'editing' : 'viewing' },
      });
    },
    [resource, send],
  );

  // ── Update cursor ────────────────────────────────────────────────
  const updateCursor = useCallback(
    (pos: CursorPosition) => {
      if (!editingRef.current) return;
      send({
        type: 'cursor',
        resource,
        payload: { ...pos },
      });
    },
    [resource, send],
  );

  // ── Check for conflicts ──────────────────────────────────────────
  const hasConflict = presences.some(
    (p) => p.action === 'editing' && p.user_id !== user?.id,
  );
  const conflictUsers = presences
    .filter((p) => p.action === 'editing' && p.user_id !== user?.id)
    .map((p) => p.user_name);

  // ── Connect ──────────────────────────────────────────────────────
  useEffect(() => {
    if (!token || !user) return;

    const wsUrl = `${getWsBaseUrl()}/api/v1/ws/collab?token=${encodeURIComponent(token)}`;
    const ws = new WebSocket(wsUrl);

    ws.onopen = () => {
      setIsConnected(true);
      // Join room
      send({ type: 'join', resource });
      // Heartbeat every 25s
      heartbeatRef.current = setInterval(() => {
        send({ type: 'presence', resource, payload: { action: editingRef.current ? 'editing' : 'viewing' } });
      }, 25000);
    };

    ws.onmessage = (event) => {
      try {
        const msg: CollabMessage = JSON.parse(event.data);

        switch (msg.type) {
          case 'presences_list':
            if (msg.presences) {
              setPresences(msg.presences);
              onPresenceChange?.(msg.presences);
              const editors = msg.presences.filter(
                (p) => p.action === 'editing' && p.user_id !== user?.id,
              );
              if (editors.length > 0) {
                onConflict?.(editors.map((e) => e.user_name));
                toast.warning(
                  `${editors.map((e) => e.user_name).join(', ')} ${
                    editors.length === 1 ? 'редактирует' : 'редактируют'
                  } этот наряд-заказ`,
                );
              }
            }
            break;

          case 'cursors_list':
            if (msg.cursors) {
              setCursors(msg.cursors);
              onCursorChange?.(msg.cursors);
            }
            break;

          case 'presence':
            if (msg.payload?.action === 'editing') {
              setEditing(true);
            }
            break;
        }
      } catch (err) {
        console.error('Failed to parse collaboration message', err);
      }
    };

    ws.onclose = () => {
      setIsConnected(false);
      if (heartbeatRef.current) {
        clearInterval(heartbeatRef.current);
        heartbeatRef.current = null;
      }
    };

    wsRef.current = ws;

    return () => {
      if (heartbeatRef.current) {
        clearInterval(heartbeatRef.current);
      }
      send({ type: 'leave', resource });
      ws.close();
      wsRef.current = null;
    };
  }, [token, user, resource, send, onPresenceChange, onCursorChange, onConflict, toast]);

  return {
    presences,
    cursors,
    hasConflict,
    conflictUsers,
    setEditing,
    updateCursor,
    isConnected,
  };
}
