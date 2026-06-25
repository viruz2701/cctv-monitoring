// ═══════════════════════════════════════════════════════════════════════
// Filter Store (Zustand + localStorage)
// UX-14.3.2: Saved Views & Filters — сохранение/загрузка фильтров и
// сортировки для страниц со списками.
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
}

interface FilterState {
  savedViews: SavedView[];

  // CRUD
  saveView: (name: string, page: string, filters: Record<string, string>, sort: { column: string; direction: 'asc' | 'desc' }) => void;
  loadView: (id: string) => SavedView | undefined;
  deleteView: (id: string) => void;
  renameView: (id: string, name: string) => void;

  // Queries
  getViewsForPage: (page: string) => SavedView[];
}

// ═══════════════════════════════════════════════════════════════════════
// Utils
// ═══════════════════════════════════════════════════════════════════════

const generateId = (): string =>
  `sv-${Date.now()}-${Math.random().toString(36).slice(2, 9)}`;

// ═══════════════════════════════════════════════════════════════════════
// Store
// ═══════════════════════════════════════════════════════════════════════

export const useFilterStore = create<FilterState>()(
  persist(
    (set, get) => ({
      savedViews: [],

      saveView: (name, page, filters, sort) => {
        const id = generateId();
        const view: SavedView = { id, name, page, filters, sort };
        set((state) => ({
          savedViews: [...state.savedViews, view],
        }));
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
            v.id === id ? { ...v, name } : v
          ),
        }));
      },

      getViewsForPage: (page) => {
        return get().savedViews.filter((v) => v.page === page);
      },
    }),
    {
      name: 'cctv-saved-views',
      partialize: (state) => ({
        savedViews: state.savedViews,
      }),
    }
  )
);
