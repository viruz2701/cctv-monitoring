// ═══════════════════════════════════════════════════════════════════════
// SparePartsContext — React Query backed
// ARCH-02: Server state managed via React Query, context retained for
// backward compatibility.
// ═══════════════════════════════════════════════════════════════════════

import React, { createContext, useContext, useCallback } from 'react';
import type { SparePart, CreateSparePartRequest, SparePartCategory } from '../services/sparePartsApi';
import {
  useSpareParts as useQuerySpareParts,
  useLowStockParts as useQueryLowStockParts,
  useSparePartCategories as useQuerySparePartCategories,
  useCreateSparePart,
  useUpdateSparePart,
  useDeleteSparePart,
  useAdjustStock,
  useCreateSparePartCategory,
  useUpdateSparePartCategory,
  useDeleteSparePartCategory,
} from '../hooks/useApiQuery';
import { useAuth } from '../hooks/useAuth';

interface SparePartsContextType {
  spareParts: SparePart[];
  lowStockParts: SparePart[];
  categories: SparePartCategory[];
  loading: boolean;
  error: string | null;
  fetchSpareParts: (filters?: Record<string, string>) => Promise<void>;
  fetchLowStockParts: () => Promise<void>;
  fetchCategories: () => Promise<void>;
  createSparePart: (data: CreateSparePartRequest) => Promise<SparePart>;
  updateSparePart: (id: string, data: Partial<CreateSparePartRequest>) => Promise<void>;
  deleteSparePart: (id: string) => Promise<void>;
  adjustStock: (id: string, quantity: number) => Promise<void>;
  createCategory: (data: { name: string; description?: string; color?: string }) => Promise<SparePartCategory>;
  updateCategory: (id: string, data: { name?: string; description?: string; color?: string }) => Promise<void>;
  deleteCategory: (id: string) => Promise<void>;
}

const SparePartsContext = createContext<SparePartsContextType | undefined>(undefined);

export const SparePartsProvider: React.FC<{ children: React.ReactNode }> = ({ children }) => {
  const { token } = useAuth();

  const {
    data: spareParts = [],
    isLoading: loading,
    error: queryError,
    refetch: refetchSpareParts,
  } = useQuerySpareParts();

  const {
    data: lowStockParts = [],
    refetch: refetchLowStock,
  } = useQueryLowStockParts();

  const {
    data: categories = [],
    refetch: refetchCategories,
  } = useQuerySparePartCategories();

  const createMutation = useCreateSparePart();
  const updateMutation = useUpdateSparePart();
  const deleteMutation = useDeleteSparePart();
  const adjustStockMutation = useAdjustStock();
  const createCategoryMutation = useCreateSparePartCategory();
  const updateCategoryMutation = useUpdateSparePartCategory();
  const deleteCategoryMutation = useDeleteSparePartCategory();

  const fetchSpareParts = useCallback(async (filters?: Record<string, string>) => {
    await refetchSpareParts();
  }, [refetchSpareParts]);

  const fetchLowStockParts = useCallback(async () => {
    await refetchLowStock();
  }, [refetchLowStock]);

  const fetchCategories = useCallback(async () => {
    await refetchCategories();
  }, [refetchCategories]);

  const createSparePart = useCallback(async (data: CreateSparePartRequest): Promise<SparePart> => {
    return createMutation.mutateAsync(data);
  }, [createMutation]);

  const updateSparePart = useCallback(async (id: string, data: Partial<CreateSparePartRequest>) => {
    await updateMutation.mutateAsync({ id, data });
  }, [updateMutation]);

  const deleteSparePart = useCallback(async (id: string) => {
    await deleteMutation.mutateAsync(id);
  }, [deleteMutation]);

  const adjustStock = useCallback(async (id: string, quantity: number) => {
    await adjustStockMutation.mutateAsync({ id, quantity });
  }, [adjustStockMutation]);

  const createCategory = useCallback(async (data: { name: string; description?: string; color?: string }): Promise<SparePartCategory> => {
    return createCategoryMutation.mutateAsync(data);
  }, [createCategoryMutation]);

  const updateCategory = useCallback(async (id: string, data: { name?: string; description?: string; color?: string }) => {
    await updateCategoryMutation.mutateAsync({ id, data });
  }, [updateCategoryMutation]);

  const deleteCategory = useCallback(async (id: string) => {
    await deleteCategoryMutation.mutateAsync(id);
  }, [deleteCategoryMutation]);

  return (
    <SparePartsContext.Provider
      value={{
        spareParts, lowStockParts, categories, loading,
        error: queryError instanceof Error ? queryError.message : null,
        fetchSpareParts, fetchLowStockParts, fetchCategories,
        createSparePart, updateSparePart, deleteSparePart, adjustStock,
        createCategory, updateCategory, deleteCategory,
      }}
    >
      {children}
    </SparePartsContext.Provider>
  );
};

// eslint-disable-next-line react-refresh/only-export-components
export const useSpareParts = () => {
  const context = useContext(SparePartsContext);
  if (!context) {
    throw new Error('useSpareParts must be used within SparePartsProvider');
  }
  return context;
};
