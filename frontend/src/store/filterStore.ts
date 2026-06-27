// ═══════════════════════════════════════════════════════════════════════
// Filter Store (Zustand + localStorage)
// UX-14.3.2: Saved Views & Filters — сохранение/загрузка фильтров и
// сортировки для страниц со списками.
// P1-UX.9: Export/import, URL sharing, default filters per role
// ═══════════════════════════════════════════════════════════════════════

import { create } from 'zustand';
import { persist } from 'zustand/middleware';

// ═══════════════════════════════════════════════════════════════════════
// Types
// ═══════════════════════════════════════════════════════════════════════

export interface SavedView {
  id: string;
  name: string;
  page: string;
  filters: Record<string, string>;
  sort: {
    column: string;
    direction: 'asc' | 'desc';
  };
  /** P1-UX.9: Роль, для которой этот фильтр является default */
  defaultForRole?: string;
}

export interface FilterStateExport {
  version: 1;
  exportedAt: string;
  views: SavedView[];
}

interface FilterState {
  savedViews: SavedView[];

  // CRUD
  saveView: (
    name: string,
    page: string,
    filters: Record<string, string>,
    sort: { column: string; direction: 'asc' | 'desc' },
    defaultForRole?: string,
  ) => void;
  loadView: (id: string) => SavedView | undefined;
  deleteView: (id: string) => void;
  renameView: (id: string, name: string) => void;

  // Queries
  getViewsForPage: (page: string) => SavedView[];

  // P1-UX.9: Default filters per role
  setDefaultForRole: (viewId: string, page: string, role: string) => void;
  getDefaultViewForRole: (page: string, role: string) => SavedView | undefined;
  clearDefaultForRole: (page: string, role: string) => void;

  // P1-UX.9: Export/Import
  exportViews: (page?: string) => string;
  importViews: (json: string) => { success: boolean; count: number; errors: string[] };

  // P1-UX.9: URL sharing helpers
  encodeFilterStateToUrl: (filters: Record<string, string>, sort: { column: string; direction: 'asc' | 'desc' }) => string;
  decodeFilterStateFromUrl: (encoded: string) => { filters: Record<string, string>; sort: { column: string; direction: 'asc' | 'desc' } } | null;
}

// ═══════════════════════════════════════════════════════════════════════
// Constants
// ═══════════════════════════════════════════════════════════════════════

const DEFAULT_ROLES = ['admin', 'manager', 'technician', 'viewer'] as const;

// ═══════════════════════════════════════════════════════════════════════
// Utils
// ═══════════════════════════════════════════════════════════════════════

const generateId = (): string =>
  `sv-${Date.now()}-${Math.random().toString(36).slice(2, 9)}`;

/**
 * P1-UX.9: Кодирует состояние фильтра в компактную строку для URL.
 * Используется base64url-encoded JSON для передачи через query params.
 */
function encodeFilterState(
  filters: Record<string, string>,
  sort: { column: string; direction: 'asc' | 'desc' },
): string {
  try {
    const data = JSON.stringify({ f: filters, s: sort });
    // Base64url (URL-safe, без padding)
    const base64 = btoa(data);
    return base64.replace(/\+/g, '-').replace(/\//g, '_').replace(/=+$/, '');
  } catch {
    return '';
  }
}

/**
 * P1-UX.9: Декодирует состояние фильтра из URL-encoded строки.
 */
function decodeFilterState(encoded: string): {
  filters: Record<string, string>;
  sort: { column: string; direction: 'asc' | 'desc' };
} | null {
  try {
    const base64 = encoded.replace(/-/g, '+').replace(/_/g, '/');
    const padded = base64.padEnd(base64.length + ((4 - (base64.length % 4)) % 4), '=');
    const data = atob(padded);
    const parsed = JSON.parse(data);
    if (parsed && typeof parsed === 'object' && parsed.f && parsed.s) {
      return { filters: parsed.f as Record<string, string>, sort: parsed.s as { column: string; direction: 'asc' | 'desc' } };
    }
    return null;
  } catch {
    return null;
  }
}

// ═══════════════════════════════════════════════════════════════════════
// Predefined per-role default filters
// ═══════════════════════════════════════════════════════════════════════

/**
 * P1-UX.9: Роль-специфичные дефолтные фильтры для каждой страницы.
 * Применяются, если пользователь не сохранил свои кастомные настройки.
 */
export const ROLE_DEFAULT_FILTERS: Record<string, Record<string, Record<string, string>>> = {
  devices: {
    technician: { status: 'active', assigned_to: 'me' },
    viewer: { status: 'active' },
    manager: { status: 'active' },
    admin: {},
  },
  'work-orders': {
    technician: { status: 'open,in_progress', assigned_to: 'me' },
    manager: { status: 'open,in_progress', priority: 'high,critical' },
    viewer: { status: 'open' },
    admin: {},
  },
  alerts: {
    technician: { status: 'active', severity: 'critical,high' },
    manager: { severity: 'critical,high' },
    viewer: { status: 'active' },
    admin: {},
  },
};

// ═══════════════════════════════════════════════════════════════════════
// Store
// ═══════════════════════════════════════════════════════════════════════

export const useFilterStore = create<FilterState>()(
  persist(
    (set, get) => ({
      savedViews: [],

      saveView: (name, page, filters, sort, defaultForRole) => {
        const id = generateId();

        // Если этот view устанавливается как default для роли, сбросить предыдущий
        let updatedViews = [...get().savedViews];
        if (defaultForRole) {
          updatedViews = updatedViews.map((v) =>
            v.page === page && v.defaultForRole === defaultForRole
              ? { ...v, defaultForRole: undefined }
              : v,
          );
        }

        const view: SavedView = { id, name, page, filters, sort, defaultForRole };
        set({ savedViews: [...updatedViews, view] });
      },

      loadView: (id) => {
        return get().savedViews.find((v) => v.id === id);
      },

      deleteView: (id) => {
        set((state) => ({
          savedViews: state.savedViews.filter((v) => v.id !== id),
        }));
      },

      renameView: (id, name) => {
        set((state) => ({
          savedViews: state.savedViews.map((v) =>
            v.id === id ? { ...v, name } : v,
          ),
        }));
      },

      getViewsForPage: (page) => {
        return get().savedViews.filter((v) => v.page === page);
      },

      // ═══ P1-UX.9: Default filters per role ═══

      setDefaultForRole: (viewId, page, role) => {
        set((state) => ({
          savedViews: state.savedViews.map((v) =>
            v.id === viewId
              ? { ...v, defaultForRole: role }
              : v.page === page && v.defaultForRole === role
                ? { ...v, defaultForRole: undefined }
                : v,
          ),
        }));
      },

      getDefaultViewForRole: (page, role) => {
        return get().savedViews.find(
          (v) => v.page === page && v.defaultForRole === role,
        );
      },

      clearDefaultForRole: (page, role) => {
        set((state) => ({
          savedViews: state.savedViews.map((v) =>
            v.page === page && v.defaultForRole === role
              ? { ...v, defaultForRole: undefined }
              : v,
          ),
        }));
      },

      // ═══ P1-UX.9: Export/Import ═══

      exportViews: (page) => {
        const views = page
          ? get().savedViews.filter((v) => v.page === page)
          : get().savedViews;
        const payload: FilterStateExport = {
          version: 1,
          exportedAt: new Date().toISOString(),
          views,
        };
        return JSON.stringify(payload, null, 2);
      },

      importViews: (json) => {
        try {
          const parsed = JSON.parse(json) as FilterStateExport;

          // Validate format
          if (!parsed || parsed.version !== 1 || !Array.isArray(parsed.views)) {
            return { success: false, count: 0, errors: ['Invalid export format'] };
          }

          const errors: string[] = [];
          let imported = 0;

          const validViews = parsed.views.filter((v) => {
            if (!v.name || !v.page || !v.filters || !v.sort) {
              errors.push(`View "${v.name || 'unknown'}" missing required fields`);
              return false;
            }
            return true;
          });

          if (validViews.length > 0) {
            set((state) => ({
              savedViews: [...state.savedViews, ...validViews],
            }));
            imported = validViews.length;
          }

          return { success: errors.length === 0, count: imported, errors };
        } catch (err) {
          return {
            success: false,
            count: 0,
            errors: [err instanceof Error ? err.message : 'Failed to parse import JSON'],
          };
        }
      },

      // ═══ P1-UX.9: URL sharing helpers ═══

      encodeFilterStateToUrl: (filters, sort) => {
        return encodeFilterState(filters, sort);
      },

      decodeFilterStateFromUrl: (encoded) => {
        return decodeFilterState(encoded);
      },
    }),
    {
      name: 'cctv-saved-views',
      partialize: (state) => ({
        savedViews: state.savedViews,
      }),
    },
  ),
);

// ═══════════════════════════════════════════════════════════════════════
// Exported helpers
// ═══════════════════════════════════════════════════════════════════════

export { encodeFilterState, decodeFilterState, DEFAULT_ROLES };
