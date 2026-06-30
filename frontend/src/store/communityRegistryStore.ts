// ═══════════════════════════════════════════════════════════════════════
// Community Registry Store (Zustand) — PROTO-07
//
// Управляет состоянием публичного реестра Protocol Descriptor'ов.
// Server state (дескрипторы из API) кешируется через Zustand,
// с поддержкой фильтрации, пагинации, поиска.
// ═══════════════════════════════════════════════════════════════════════

import { create } from 'zustand';
import { communityRegistryApi } from '../services/api/communityRegistry';
import type {
  CommunityDescriptorSummary,
  CommunityDescriptor,
  CommunityDescriptorFilter,
} from '../services/api/communityRegistry';

// ─── Default filter ─────────────────────────────────────────────────

const DEFAULT_FILTER: CommunityDescriptorFilter = {
  page: 1,
  page_size: 20,
  sort_by: 'rating',
  sort_dir: 'desc',
};

// ─── Store interface ────────────────────────────────────────────────

interface CommunityRegistryStoreState {
  /** List of community descriptors */
  descriptors: CommunityDescriptorSummary[];
  /** Currently selected descriptor (detail view) */
  selectedDescriptor: CommunityDescriptor | null;
  /** Total count */
  total: number;
  /** Current page */
  page: number;
  /** Page size */
  pageSize: number;
  /** Total pages */
  totalPages: number;
  /** Active filters */
  filter: CommunityDescriptorFilter;
  /** Loading state */
  loading: boolean;
  /** Error message */
  error: string | null;
  /** Detail view loading */
  detailLoading: boolean;
  /** Detail view error */
  detailError: string | null;

  // ─── Actions ───

  /** Fetch descriptor list from API */
  fetchDescriptors: () => Promise<void>;
  /** Fetch single descriptor by vendor */
  fetchDescriptor: (vendor: string) => Promise<void>;
  /** Set filter and re-fetch */
  setFilter: (patch: Partial<CommunityDescriptorFilter>) => void;
  /** Reset filters */
  resetFilter: () => void;
  /** Go to page */
  setPage: (page: number) => void;
  /** Clear error */
  clearError: () => void;
  /** Reset to initial state */
  reset: () => void;
}

// ─── Initial state ─────────────────────────────────────────────────

const initialState = {
  descriptors: [] as CommunityDescriptorSummary[],
  selectedDescriptor: null as CommunityDescriptor | null,
  total: 0,
  page: 1,
  pageSize: 20,
  totalPages: 0,
  filter: { ...DEFAULT_FILTER },
  loading: false,
  error: null as string | null,
  detailLoading: false,
  detailError: null as string | null,
};

// ─── Store ──────────────────────────────────────────────────────────

export const useCommunityRegistryStore = create<CommunityRegistryStoreState>()(
  (set, get) => ({
    ...initialState,

    fetchDescriptors: async () => {
      const { filter } = get();
      set({ loading: true, error: null });

      try {
        const response = await communityRegistryApi.list({
          ...filter,
          page: get().page,
          page_size: get().pageSize,
        });

        set({
          descriptors: response.descriptors,
          total: response.total,
          page: response.page,
          pageSize: response.page_size,
          totalPages: response.total_pages,
          loading: false,
        });
      } catch (err) {
        const message =
          err instanceof Error
            ? err.message
            : 'Failed to fetch community descriptors';
        set({ error: message, loading: false });
      }
    },

    fetchDescriptor: async (vendor: string) => {
      set({ detailLoading: true, detailError: null });

      try {
        const descriptor = await communityRegistryApi.get(vendor);
        set({ selectedDescriptor: descriptor, detailLoading: false });
      } catch (err) {
        const message =
          err instanceof Error
            ? err.message
            : `Failed to fetch descriptor: ${vendor}`;
        set({ detailError: message, detailLoading: false });
      }
    },

    setFilter: (patch) => {
      const current = get().filter;
      const newFilter = { ...current, ...patch, page: 1 }; // Reset to page 1 on filter change
      set({ filter: newFilter, page: 1 });
      // Auto-fetch on filter change
      get().fetchDescriptors();
    },

    resetFilter: () => {
      set({ filter: { ...DEFAULT_FILTER }, page: 1 });
      get().fetchDescriptors();
    },

    setPage: (page: number) => {
      set({ page });
      get().fetchDescriptors();
    },

    clearError: () => set({ error: null, detailError: null }),

    reset: () => set({ ...initialState }),
  }),
);

// ─── Selector hooks ─────────────────────────────────────────────────

export const useCommunityDescriptors = () =>
  useCommunityRegistryStore((s) => s.descriptors);
export const useCommunityDescriptorDetail = () =>
  useCommunityRegistryStore((s) => s.selectedDescriptor);
export const useCommunityRegistryLoading = () =>
  useCommunityRegistryStore((s) => s.loading);
export const useCommunityRegistryError = () =>
  useCommunityRegistryStore((s) => s.error);
export const useCommunityRegistryPagination = () =>
  useCommunityRegistryStore((s) => ({
    total: s.total,
    page: s.page,
    pageSize: s.pageSize,
    totalPages: s.totalPages,
  }));
export const useCommunityRegistryFilter = () =>
  useCommunityRegistryStore((s) => s.filter);
