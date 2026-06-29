// ═══════════════════════════════════════════════════════════════════════
// Playbook Marketplace API
// P1-MARKET: Публичный marketplace pre-built playbooks
//   GET    /playbook-marketplace          — list with filters
//   GET    /playbook-marketplace/{id}     — get by ID
//   POST   /playbook-marketplace/{id}/install — install to tenant
//   POST   /playbook-marketplace/{id}/rate    — rate (1-5)
//   GET    /playbook-marketplace/my       — installed by current tenant
//   POST   /playbook-marketplace/{id}/share  — private share
// ═══════════════════════════════════════════════════════════════════════

import { request } from './client';

// ─── Types ──────────────────────────────────────────────────────────────

export interface MarketplacePlaybook {
  id: string;
  name: string;
  description: string;
  vendor: 'hikvision' | 'dahua' | 'axis' | 'uniview' | 'generic';
  version: string;
  compat_matrix: string[];
  avg_rating: number;
  review_count: number;
  install_count: number;
  verified: boolean;
  tenant_id: string;
  playbook_data?: Record<string, unknown>;
  created_at: string;
  updated_at: string;
}

export interface MarketplaceListResponse {
  playbooks: MarketplacePlaybook[];
  total: number;
  limit: number;
  offset: number;
}

export interface MarketplaceFilter {
  vendor?: string;
  min_rating?: number;
  search?: string;
  verified?: boolean;
  limit?: number;
  offset?: number;
}

// ─── API Calls ──────────────────────────────────────────────────────────

export const playbookMarketplaceApi = {
  /** List available playbooks with filters */
  list(filter: MarketplaceFilter = {}): Promise<MarketplaceListResponse> {
    const params = new URLSearchParams();
    if (filter.vendor) params.set('vendor', filter.vendor);
    if (filter.min_rating !== undefined) params.set('min_rating', String(filter.min_rating));
    if (filter.search) params.set('search', filter.search);
    if (filter.verified !== undefined) params.set('verified', String(filter.verified));
    if (filter.limit !== undefined) params.set('limit', String(filter.limit));
    if (filter.offset !== undefined) params.set('offset', String(filter.offset));

    const qs = params.toString();
    return request<MarketplaceListResponse>(`/playbook-marketplace${qs ? `?${qs}` : ''}`);
  },

  /** Get playbook by ID (with full playbook_data) */
  get(id: string): Promise<MarketplacePlaybook> {
    return request<MarketplacePlaybook>(`/playbook-marketplace/${id}`);
  },

  /** Install a playbook to current tenant */
  install(id: string): Promise<{ status: string; message: string }> {
    return request<{ status: string; message: string }>(`/playbook-marketplace/${id}/install`, {
      method: 'POST',
    });
  },

  /** Rate a playbook (1-5 stars, optional review) */
  rate(id: string, score: number, review?: string): Promise<{ status: string; message: string }> {
    return request<{ status: string; message: string }>(`/playbook-marketplace/${id}/rate`, {
      method: 'POST',
      body: JSON.stringify({ score, review }),
    });
  },

  /** Get playbooks installed by current tenant */
  getMyPlaybooks(): Promise<{ playbooks: MarketplacePlaybook[]; total: number }> {
    return request<{ playbooks: MarketplacePlaybook[]; total: number }>('/playbook-marketplace/my');
  },

  /** Share a playbook with another tenant */
  share(id: string, targetTenant: string): Promise<{ status: string; message: string }> {
    return request<{ status: string; message: string }>(`/playbook-marketplace/${id}/share`, {
      method: 'POST',
      body: JSON.stringify({ target_tenant: targetTenant }),
    });
  },
};
