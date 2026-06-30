// ═══════════════════════════════════════════════════════════════════════
// Descriptor Store (Zustand) — PROTO-06: Descriptor Editor UI
//
// ARCH-02: UI-состояние для редактора дескрипторов протоколов.
// Server state (дескрипторы из API) кешируется через React Query,
// этот store управляет только UI-состоянием редактора.
// ═══════════════════════════════════════════════════════════════════════

import { create } from 'zustand';
import { descriptorsApi } from '../services/api/descriptors';
import type {
  ProtocolDescriptor,
  ProtocolEndpoint,
  DescriptorListItem,
  EditorMode,
  DescriptorViewTab,
} from '../types/descriptor';

// ─── Default empty descriptor ──────────────────────────────────────

export function createEmptyDescriptor(): ProtocolDescriptor {
  return {
    vendor: '',
    version: '1.0.0',
    description: '',
    endpoints: [],
    auth: { type: 'none' },
    category: 'camera',
  };
}

// ─── Store interface ────────────────────────────────────────────────

interface DescriptorStoreState {
  /** Current editor mode */
  mode: EditorMode;
  /** Active view tab */
  tab: DescriptorViewTab;
  /** List of all descriptors */
  descriptors: DescriptorListItem[];
  /** Currently editing descriptor */
  currentDescriptor: ProtocolDescriptor;
  /** Loading state */
  loading: boolean;
  /** Error message */
  error: string | null;
  /** Dirty flag — unsaved changes exist */
  dirty: boolean;

  // ─── Actions ───

  /** Set editor mode */
  setMode: (mode: EditorMode) => void;
  /** Set active tab */
  setTab: (tab: DescriptorViewTab) => void;
  /** Load descriptor list from API */
  fetchDescriptors: () => Promise<void>;
  /** Load single descriptor for editing */
  loadDescriptor: (vendor: string) => Promise<void>;
  /** Start creating a new descriptor */
  startNew: () => void;
  /** Update the current descriptor (field-level) */
  updateDescriptor: (patch: Partial<ProtocolDescriptor>) => void;
  /** Add a new endpoint */
  addEndpoint: () => void;
  /** Update an endpoint by index */
  updateEndpoint: (index: number, patch: Partial<ProtocolEndpoint>) => void;
  /** Remove an endpoint by index */
  removeEndpoint: (index: number) => void;
  /** Save current descriptor to API */
  saveDescriptor: () => Promise<boolean>;
  /** Delete a descriptor by vendor */
  deleteDescriptor: (vendor: string) => Promise<boolean>;
  /** Clear error */
  clearError: () => void;
  /** Reset to initial state */
  reset: () => void;
}

// ─── Initial state ─────────────────────────────────────────────────

const initialState = {
  mode: 'list' as EditorMode,
  tab: 'form' as DescriptorViewTab,
  descriptors: [] as DescriptorListItem[],
  currentDescriptor: createEmptyDescriptor(),
  loading: false,
  error: null as string | null,
  dirty: false,
};

// ─── Store ──────────────────────────────────────────────────────────

export const useDescriptorStore = create<DescriptorStoreState>()(
  (set, get) => ({
    ...initialState,

    setMode: (mode) => set({ mode }),

    setTab: (tab) => set({ tab }),

    fetchDescriptors: async () => {
      set({ loading: true, error: null });
      try {
        const descriptors = await descriptorsApi.list();
        set({ descriptors, loading: false });
      } catch (err) {
        const message =
          err instanceof Error ? err.message : 'Failed to fetch descriptors';
        set({ error: message, loading: false });
      }
    },

    loadDescriptor: async (vendor: string) => {
      set({ loading: true, error: null, mode: 'edit' });
      try {
        const descriptor = await descriptorsApi.get(vendor);
        set({ currentDescriptor: descriptor, loading: false, dirty: false });
      } catch (err) {
        const message =
          err instanceof Error
            ? err.message
            : `Failed to load descriptor: ${vendor}`;
        set({ error: message, loading: false });
      }
    },

    startNew: () => {
      set({
        mode: 'create',
        currentDescriptor: createEmptyDescriptor(),
        dirty: false,
        error: null,
        tab: 'form',
      });
    },

    updateDescriptor: (patch) => {
      const current = get().currentDescriptor;
      set({
        currentDescriptor: { ...current, ...patch },
        dirty: true,
      });
    },

    addEndpoint: () => {
      const current = get().currentDescriptor;
      const newEndpoint: ProtocolEndpoint = {
        id: `endpoint_${Date.now()}`,
        method: 'GET',
        path: '/',
        name: '',
        description: '',
        successCodes: [200],
        timeout: 30,
      };
      set({
        currentDescriptor: {
          ...current,
          endpoints: [...current.endpoints, newEndpoint],
        },
        dirty: true,
      });
    },

    updateEndpoint: (index, patch) => {
      const current = get().currentDescriptor;
      const endpoints = [...current.endpoints];
      if (index >= 0 && index < endpoints.length) {
        endpoints[index] = { ...endpoints[index], ...patch };
        set({
          currentDescriptor: { ...current, endpoints },
          dirty: true,
        });
      }
    },

    removeEndpoint: (index) => {
      const current = get().currentDescriptor;
      const endpoints = current.endpoints.filter((_, i) => i !== index);
      set({
        currentDescriptor: { ...current, endpoints },
        dirty: true,
      });
    },

    saveDescriptor: async () => {
      const { currentDescriptor } = get();
      set({ loading: true, error: null });
      try {
        await descriptorsApi.save(currentDescriptor);
        set({
          loading: false,
          dirty: false,
          mode: 'list',
        });
        // Refresh list
        get().fetchDescriptors();
        return true;
      } catch (err) {
        const message =
          err instanceof Error ? err.message : 'Failed to save descriptor';
        set({ error: message, loading: false });
        return false;
      }
    },

    deleteDescriptor: async (vendor: string) => {
      set({ loading: true, error: null });
      try {
        await descriptorsApi.delete(vendor);
        set({ loading: false });
        // Refresh list
        get().fetchDescriptors();
        return true;
      } catch (err) {
        const message =
          err instanceof Error
            ? err.message
            : `Failed to delete descriptor: ${vendor}`;
        set({ error: message, loading: false });
        return false;
      }
    },

    clearError: () => set({ error: null }),

    reset: () => set({ ...initialState }),
  }),
);

// ─── Selector hooks ─────────────────────────────────────────────────

export const useDescriptorMode = () => useDescriptorStore((s) => s.mode);
export const useDescriptorTab = () => useDescriptorStore((s) => s.tab);
export const useDescriptorList = () => useDescriptorStore((s) => s.descriptors);
export const useCurrentDescriptor = () =>
  useDescriptorStore((s) => s.currentDescriptor);
export const useDescriptorLoading = () => useDescriptorStore((s) => s.loading);
export const useDescriptorError = () => useDescriptorStore((s) => s.error);
export const useDescriptorDirty = () => useDescriptorStore((s) => s.dirty);
