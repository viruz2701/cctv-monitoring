// ═══════════════════════════════════════════════════════════════════════
// communityRegistry.ts — Community Protocol Registry API Service (PROTO-07)
//
// Публичный реестр Protocol Descriptor'ов, где community может
// публиковать и обмениваться дескрипторами для вендоров CCTV.
//
// API endpoints:
//   GET    /api/v1/community/descriptors              — список
//   GET    /api/v1/community/descriptors/:vendor      — детали
//   POST   /api/v1/community/descriptors              — публикация
//   POST   /api/v1/community/descriptors/:vendor/rate — оценка
//   GET    /api/v1/community/descriptors/:vendor/download — скачать
//
// Compliance:
//   - OWASP ASVS V5 (input validation на всех мутациях)
// ═══════════════════════════════════════════════════════════════════════

import { request } from './client';

// ─── Types ──────────────────────────────────────────────────────────

export interface CommunityDescriptorSummary {
  id: string;
  vendor: string;
  version: string;
  rating: number;
  downloads: number;
  verified: boolean;
  created_at: string;
  updated_at: string;
}

export interface CommunityDescriptor {
  id: string;
  vendor: string;
  version: string;
  descriptor: unknown;
  author_id: string;
  rating: number;
  downloads: number;
  verified: boolean;
  created_at: string;
  updated_at: string;
}

export interface CommunityDescriptorListResponse {
  descriptors: CommunityDescriptorSummary[];
  total: number;
  page: number;
  page_size: number;
  total_pages: number;
}

export interface CommunityDescriptorFilter {
  search?: string;
  min_rating?: number;
  verified?: boolean;
  page?: number;
  page_size?: number;
  sort_by?: 'rating' | 'downloads' | 'created_at' | 'vendor';
  sort_dir?: 'asc' | 'desc';
}

export interface PublishDescriptorRequest {
  vendor: string;
  version: string;
  descriptor: unknown;
}

export interface RateDescriptorRequest {
  score: number;
}

// ─── API paths ──────────────────────────────────────────────────────

const BASE = '/community/descriptors';

// ─── Community Registry API ─────────────────────────────────────────

export const communityRegistryApi = {
  /** Get list of community descriptors with filters & pagination */
  list: (filter?: CommunityDescriptorFilter): Promise<CommunityDescriptorListResponse> => {
    const params = new URLSearchParams();
    if (filter?.search) params.set('search', filter.search);
    if (filter?.min_rating) params.set('min_rating', String(filter.min_rating));
    if (filter?.verified !== undefined) params.set('verified', String(filter.verified));
    if (filter?.page) params.set('page', String(filter.page));
    if (filter?.page_size) params.set('page_size', String(filter.page_size));
    if (filter?.sort_by) params.set('sort_by', filter.sort_by);
    if (filter?.sort_dir) params.set('sort_dir', filter.sort_dir);

    const query = params.toString();
    return request<CommunityDescriptorListResponse>(`${BASE}${query ? `?${query}` : ''}`);
  },

  /** Get single community descriptor by vendor name */
  get: (vendor: string): Promise<CommunityDescriptor> =>
    request<CommunityDescriptor>(`${BASE}/${encodeURIComponent(vendor)}`),

  /** Publish a new community descriptor (auth required) */
  publish: (req: PublishDescriptorRequest): Promise<CommunityDescriptor> =>
    request<CommunityDescriptor>(BASE, {
      method: 'POST',
      body: JSON.stringify(req),
    }),

  /** Rate a community descriptor (1-5, auth required) */
  rate: (vendor: string, score: number): Promise<{ status: string; message: string }> =>
    request<{ status: string; message: string }>(`${BASE}/${encodeURIComponent(vendor)}/rate`, {
      method: 'POST',
      body: JSON.stringify({ score }),
    }),

  /** Download a community descriptor (increments counter) */
  download: (vendor: string): Promise<CommunityDescriptor> =>
    request<CommunityDescriptor>(`${BASE}/${encodeURIComponent(vendor)}/download`),
};
