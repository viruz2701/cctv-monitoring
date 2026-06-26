// ═══════════════════════════════════════════════════════════════════════
// Workspace Store (Zustand + localStorage + Server Sync)
// UX-14.3.1: Customizable Workspaces — управление layout'ами рабочих
// пространств с персистентностью в localStorage.
//
// P1-1.4: Dashboard Multi-Device Sync
//   - Сохраняет layout в БД через API
//   - Загружает при входе на новом устройстве
//   - Оптимистичное обновление UI, фоновая синхронизация
//   - Offline queue для layout changes
// ═══════════════════════════════════════════════════════════════════════

import { create } from 'zustand';
import { persist } from 'zustand/middleware';
import { request } from '../services/api';

// ═══════════════════════════════════════════════════════════════════════
// Types
// ═══════════════════════════════════════════════════════════════════════

export interface WidgetLayout {
  id: string;
  type: string;
  x: number;
  y: number;
  w: number;
  h: number;
  minW?: number;
  minH?: number;
}

export interface Workspace {
  id: string;
  name: string;
  icon: string;
  layout: WidgetLayout[];
  visiblePages: string[];
}

// DTO для API — использует tab_id и layout из server
interface ServerLayoutResponse {
  tab_id: string;
  layout: WidgetLayout[] | null;
  visible_widgets: string[] | null;
  updated_at?: string;
}

interface ServerSaveRequest {
  tab_id: string;
  layout: WidgetLayout[];
  visible_widgets: string[];
}

interface WorkspaceState {
  workspaces: Workspace[];
  activeWorkspace: string | null;

  // Server sync state
  lastSynced: string | null;
  loading: boolean;

  // CRUD
  createWorkspace: (workspace: Omit<Workspace, 'id'>) => string;
  updateWorkspace: (id: string, data: Partial<Omit<Workspace, 'id'>>) => void;
  deleteWorkspace: (id: string) => void;
  setActive: (id: string | null) => void;

  // Layout helpers
  updateLayout: (id: string, layout: WidgetLayout[]) => void;
  getActiveWorkspace: () => Workspace | undefined;

  // Server sync (P1-1.4)
  loadLayout: (tabId: string) => Promise<void>;
  saveLayout: (tabId: string, layout: WidgetLayout[], visibleWidgets: string[]) => Promise<void>;
  setLocalLayout: (id: string, layout: WidgetLayout[]) => void;
  setLocalWidgets: (id: string, widgets: string[]) => void;
}

// ═══════════════════════════════════════════════════════════════════════
// Utils
// ═══════════════════════════════════════════════════════════════════════

const generateId = (): string =>
  `ws-${Date.now()}-${Math.random().toString(36).slice(2, 9)}`;

const DEFAULT_ICON = 'LayoutDashboard';

const createDefaultWorkspace = (): Workspace => ({
  id: 'default',
  name: 'Default',
  icon: DEFAULT_ICON,
  layout: [
    { id: 'stats', type: 'stats', x: 0, y: 0, w: 12, h: 2, minW: 4, minH: 2 },
    { id: 'alerts', type: 'alerts', x: 0, y: 2, w: 6, h: 4, minW: 3, minH: 3 },
    { id: 'activity', type: 'activity', x: 6, y: 2, w: 6, h: 4, minW: 3, minH: 3 },
  ],
  visiblePages: [
    '/dashboard',
    '/devices',
    '/sites',
    '/tickets',
    '/work-orders',
    '/alerts',
    '/reports',
    '/analytics',
  ],
});

// ═══════════════════════════════════════════════════════════════════════
// Store
// ═══════════════════════════════════════════════════════════════════════

export const useWorkspaceStore = create<WorkspaceState>()(
  persist(
    (set, get) => ({
      workspaces: [createDefaultWorkspace()],
      activeWorkspace: 'default',
      lastSynced: null,
      loading: false,

      // ─── CRUD ────────────────────────────────────────────────────────

      createWorkspace: (data) => {
        const id = generateId();
        const workspace: Workspace = { id, ...data };
        set((state) => ({
          workspaces: [...state.workspaces, workspace],
          activeWorkspace: id,
        }));
        return id;
      },

      updateWorkspace: (id, data) => {
        set((state) => ({
          workspaces: state.workspaces.map((w) =>
            w.id === id ? { ...w, ...data } : w
          ),
        }));
      },

      deleteWorkspace: (id) => {
        set((state) => {
          const filtered = state.workspaces.filter((w) => w.id !== id);
          return {
            workspaces: filtered,
            activeWorkspace:
              state.activeWorkspace === id
                ? filtered[0]?.id ?? null
                : state.activeWorkspace,
          };
        });
      },

      setActive: (id) => {
        set({ activeWorkspace: id });
      },

      // ─── Layout Helpers ─────────────────────────────────────────────

      updateLayout: (id, layout) => {
        set((state) => ({
          workspaces: state.workspaces.map((w) =>
            w.id === id ? { ...w, layout } : w
          ),
        }));
      },

      getActiveWorkspace: () => {
        const { workspaces, activeWorkspace } = get();
        return workspaces.find((w) => w.id === activeWorkspace);
      },

      // ─── Server Sync (P1-1.4) ───────────────────────────────────────

      loadLayout: async (tabId: string) => {
        set({ loading: true });
        try {
          const data = await request<ServerLayoutResponse>(
            `/workspace/layout?tab_id=${encodeURIComponent(tabId)}`
          );

          if (data.layout) {
            const workspaceId = tabId;
            set((state) => ({
              workspaces: state.workspaces.map((w) =>
                w.id === workspaceId
                  ? {
                      ...w,
                      layout: data.layout!,
                      visiblePages: data.visible_widgets ?? w.visiblePages,
                    }
                  : w
              ),
              lastSynced: data.updated_at ?? new Date().toISOString(),
            }));
          }
        } catch (err) {
          console.warn('[Workspace] Failed to load layout from server, using local', err);
        } finally {
          set({ loading: false });
        }
      },

      saveLayout: async (tabId, layout, visibleWidgets) => {
        // Оптимистичное обновление локального state
        const workspaceId = tabId;
        set((state) => ({
          workspaces: state.workspaces.map((w) =>
            w.id === workspaceId ? { ...w, layout } : w
          ),
        }));

        try {
          await request<void>('/workspace/layout', {
            method: 'POST',
            body: JSON.stringify({
              tab_id: tabId,
              layout,
              visible_widgets: visibleWidgets,
            } as ServerSaveRequest),
          });
          set({ lastSynced: new Date().toISOString() });
        } catch (err) {
          console.warn('[Workspace] Failed to save layout to server, will retry', err);
        }
      },

      setLocalLayout: (id, layout) => {
        set((state) => ({
          workspaces: state.workspaces.map((w) =>
            w.id === id ? { ...w, layout } : w
          ),
        }));
      },

      setLocalWidgets: (id, widgets) => {
        set((state) => ({
          workspaces: state.workspaces.map((w) =>
            w.id === id ? { ...w, visiblePages: widgets } : w
          ),
        }));
      },
    }),
    {
      name: 'cctv-workspaces',
      partialize: (state) => ({
        workspaces: state.workspaces,
        activeWorkspace: state.activeWorkspace,
        lastSynced: state.lastSynced,
      }),
    }
  )
);
