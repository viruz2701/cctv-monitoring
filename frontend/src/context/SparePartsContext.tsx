import React, { createContext, useContext, useState, useEffect, useCallback } from 'react';
import { sparePartsApi, SparePart, CreateSparePartRequest, SparePartCategory } from '../services/sparePartsApi';
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
  const [spareParts, setSpareParts] = useState<SparePart[]>([]);
  const [lowStockParts, setLowStockParts] = useState<SparePart[]>([]);
  const [categories, setCategories] = useState<SparePartCategory[]>([]);
  const [loading, setLoading] = useState(false);
  const [error, setError] = useState<string | null>(null);

  const fetchSpareParts = useCallback(async (filters?: Record<string, string>) => {
    setLoading(true);
    setError(null);
    try {
      const data = await sparePartsApi.getSpareParts(filters);
      setSpareParts(data || []);
    } catch (err) {
      setError(err instanceof Error ? err.message : 'Failed to fetch spare parts');
    } finally {
      setLoading(false);
    }
  }, []);

  const fetchLowStockParts = useCallback(async () => {
    try {
      const data = await sparePartsApi.getLowStockParts();
      setLowStockParts(data || []);
    } catch (err) {
      console.error('Failed to fetch low stock parts:', err);
    }
  }, []);

  const fetchCategories = useCallback(async () => {
    try {
      const data = await sparePartsApi.getCategories();
      setCategories(data || []);
    } catch (err) {
      console.error('Failed to fetch categories:', err);
    }
  }, []);

  const createSparePart = async (data: CreateSparePartRequest): Promise<SparePart> => {
    const part = await sparePartsApi.createSparePart(data);
    setSpareParts((prev) => [...prev, part]);
    return part;
  };

  const updateSparePart = async (id: string, data: Partial<CreateSparePartRequest>) => {
    await sparePartsApi.updateSparePart(id, data);
    setSpareParts((prev) => prev.map((p) => (p.id === id ? { ...p, ...data } : p)));
  };

  const deleteSparePart = async (id: string) => {
    await sparePartsApi.deleteSparePart(id);
    setSpareParts((prev) => prev.filter((p) => p.id !== id));
  };

  const adjustStock = async (id: string, quantity: number) => {
    await sparePartsApi.adjustStock(id, quantity);
    setSpareParts((prev) => prev.map((p) => (p.id === id ? { ...p, stock: quantity } : p)));
  };

  const createCategory = async (data: { name: string; description?: string; color?: string }): Promise<SparePartCategory> => {
    const cat = await sparePartsApi.createCategory(data);
    setCategories((prev) => [...prev, cat]);
    return cat;
  };

  const updateCategory = async (id: string, data: { name?: string; description?: string; color?: string }) => {
    await sparePartsApi.updateCategory(id, data);
    setCategories((prev) => prev.map((c) => (c.id === id ? { ...c, ...data } : c)));
  };

  const deleteCategory = async (id: string) => {
    await sparePartsApi.deleteCategory(id);
    setCategories((prev) => prev.filter((c) => c.id !== id));
  };

  useEffect(() => {
    if (!token) return;
    fetchSpareParts();
    fetchLowStockParts();
    fetchCategories();
  }, [fetchSpareParts, fetchLowStockParts, fetchCategories, token]);

  return (
    <SparePartsContext.Provider
      value={{
        spareParts, lowStockParts, categories, loading, error,
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
