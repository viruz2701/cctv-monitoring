// ═══════════════════════════════════════════════════════════════════════
// Workspace Store (Zustand + localStorage)
// UX-14.3.1: Customizable Workspaces — управление layout'ами рабочих
// пространств с персистентностью в localStorage.
// ═══════════════════════════════════════════════════════════════════════

import { create } from 'zustand';
import { persist } from 'zustand/middleware';

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

interface WorkspaceState {
  workspaces: Workspace[];
  activeWorkspace: string | null;

  // CRUD
  createWorkspace: (workspace: Omit<Workspace, 'id'>) => string;
  updateWorkspace: (id: string, data: Partial<Omit<Workspace, 'id'>>) => void;
  deleteWorkspace: (id: string) => void;
  setActive: (id: string | null) => void;

  // Layout helpers
  updateLayout: (id: string, layout: WidgetLayout[]) => void;
  getActiveWorkspace: () => Workspace | undefined;
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
    }),
    {
      name: 'cctv-workspaces',
      partialize: (state) => ({
        workspaces: state.workspaces,
        activeWorkspace: state.activeWorkspace,
      }),
    }
  )
);
