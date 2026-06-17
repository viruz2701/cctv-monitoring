import { request } from './api';

export interface SparePart {
  id: string;
  name: string;
  sku: string;
  category?: string;
  stock: number;
  min_stock: number;
  location?: string;
  compatible_devices: string[];
  cost: number;
  supplier?: string;
  created_at: string;
  updated_at: string;
}

export interface CreateSparePartRequest {
  name: string;
  sku?: string;
  category?: string;
  stock?: number;
  min_stock?: number;
  location?: string;
  compatible_devices?: string[];
  cost?: number;
  supplier?: string;
}

export const sparePartsApi = {
  getSpareParts: (filters?: Record<string, string>) => {
    const params = new URLSearchParams(filters).toString();
    return request<SparePart[]>(`/spare-parts${params ? '?' + params : ''}`);
  },

  getSparePart: (id: string) => {
    return request<SparePart>(`/spare-parts/${id}`);
  },

  createSparePart: (data: CreateSparePartRequest) => {
    return request<SparePart>('/spare-parts', {
      method: 'POST',
      body: JSON.stringify(data),
    });
  },

  updateSparePart: (id: string, data: Partial<CreateSparePartRequest>) => {
    return request<{ status: string }>(`/spare-parts/${id}`, {
      method: 'PUT',
      body: JSON.stringify(data),
    });
  },

  deleteSparePart: (id: string) => {
    return request<{ status: string }>(`/spare-parts/${id}`, {
      method: 'DELETE',
    });
  },

  getLowStockParts: () => {
    return request<SparePart[]>('/spare-parts/low-stock');
  },

  adjustStock: (id: string, quantity: number) => {
    return request<{ status: string }>(`/spare-parts/${id}/adjust`, {
      method: 'POST',
      body: JSON.stringify({ quantity }),
    });
  },
};
