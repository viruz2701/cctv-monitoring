// ═══════════════════════════════════════════════════════════════════════
// Saved Views Store (Zustand + localStorage)
// P1-1.3: Widget Registry & Saved Views
//   - Сохранение layout + visible widgets для каждого таба дашборда
//   - localStorage persistence
//   - Импорт/экспорт как JSON
// ═══════════════════════════════════════════════════════════════════════

import { create } from 'zustand';
import { persist } from 'zustand/middleware';
import type { LayoutItem } from 'react-grid-layout';

// ═══════════════════════════════════════════════════════════════════════
// Types
// ═══════════════════════════════════════════════════════════════════════

export interface SavedView {
    id: string;
    name: string;
    tabId: string;
    layout: LayoutItem[];
    visibleWidgets: string[];
    createdAt: string;
    updatedAt: string;
    isDefault?: boolean;
}

interface SavedViewsState {
    views: SavedView[];

    // Actions
    addView: (view: Omit<SavedView, 'id' | 'createdAt' | 'updatedAt'>) => void;
    removeView: (id: string) => void;
    updateView: (id: string, updates: Partial<SavedView>) => void;
    setDefaultView: (id: string) => void;
    exportViews: () => string;
    importViews: (json: string) => boolean;
    getViewsForTab: (tabId: string) => SavedView[];
    loadViews: () => void;
}

// ═══════════════════════════════════════════════════════════════════════
// Constants
// ═══════════════════════════════════════════════════════════════════════

const STORAGE_KEY = 'saved_dashboard_views';

// ═══════════════════════════════════════════════════════════════════════
// Utils
// ═══════════════════════════════════════════════════════════════════════

let nextId = 1;

function generateId(): string {
    return `view-${Date.now()}-${nextId++}`;
}

function loadFromStorage(): SavedView[] {
    try {
        const raw = localStorage.getItem(STORAGE_KEY);
        if (raw) return JSON.parse(raw) as SavedView[];
    } catch {
        // Corrupted or inaccessible storage — return empty
    }
    return [];
}

function saveToStorage(views: SavedView[]): void {
    try {
        localStorage.setItem(STORAGE_KEY, JSON.stringify(views));
    } catch {
        // localStorage full or unavailable — silently fail
    }
}

// ═══════════════════════════════════════════════════════════════════════
// Store
// ═══════════════════════════════════════════════════════════════════════

export const useSavedViewsStore = create<SavedViewsState>()(
    persist(
        (set, get) => ({
            views: [],

            loadViews: () => {
                const views = loadFromStorage();
                set({ views });
            },

            addView: (viewData) => {
                const now = new Date().toISOString();
                const view: SavedView = {
                    ...viewData,
                    id: generateId(),
                    createdAt: now,
                    updatedAt: now,
                };
                set((state) => {
                    const views = [...state.views, view];
                    saveToStorage(views);
                    return { views };
                });
            },

            removeView: (id) => {
                set((state) => {
                    const views = state.views.filter((v) => v.id !== id);
                    saveToStorage(views);
                    return { views };
                });
            },

            updateView: (id, updates) => {
                set((state) => {
                    const views = state.views.map((v) =>
                        v.id === id
                            ? { ...v, ...updates, updatedAt: new Date().toISOString() }
                            : v,
                    );
                    saveToStorage(views);
                    return { views };
                });
            },

            setDefaultView: (id) => {
                set((state) => {
                    const views = state.views.map((v) => ({
                        ...v,
                        isDefault: v.id === id,
                    }));
                    saveToStorage(views);
                    return { views };
                });
            },

            exportViews: () => {
                return JSON.stringify(get().views, null, 2);
            },

            importViews: (json) => {
                try {
                    const parsed = JSON.parse(json) as unknown;
                    if (!Array.isArray(parsed)) return false;
                    set((state) => {
                        const views = [...state.views, ...parsed] as SavedView[];
                        saveToStorage(views);
                        return { views };
                    });
                    return true;
                } catch {
                    return false;
                }
            },

            getViewsForTab: (tabId) => {
                return get().views.filter((v) => v.tabId === tabId);
            },
        }),
        {
            name: STORAGE_KEY,
            partialize: (state) => ({
                views: state.views,
            }),
        },
    ),
);
