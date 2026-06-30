// ═══════════════════════════════════════════════════════════════════════
// descriptors.ts — Protocol Descriptors API Service (PROTO-06)
//
// CRUD operations for declarative protocol descriptors.
//
// API endpoints:
//   GET    /api/v1/protocols/descriptors          — list
//   GET    /api/v1/protocols/descriptors/:vendor  — get one
//   POST   /api/v1/protocols/descriptors          — create/update
//   DELETE /api/v1/protocols/descriptors/:vendor  — delete
//   POST   /api/v1/protocols/descriptors/test     — test endpoint
//
// Compliance:
//   - OWASP ASVS V5 (input validation on all mutations)
// ═══════════════════════════════════════════════════════════════════════

import { request } from './client';
import type {
  ProtocolDescriptor,
  DescriptorListItem,
  DescriptorTestRequest,
  DescriptorTestResponse,
} from '../../types/descriptor';

// ─── API paths ──────────────────────────────────────────────────────

const BASE = '/protocols/descriptors';

// ─── Descriptors API ────────────────────────────────────────────────

export const descriptorsApi = {
  /** Get list of all descriptors */
  list: (): Promise<DescriptorListItem[]> =>
    request<DescriptorListItem[]>(BASE),

  /** Get single descriptor by vendor name */
  get: (vendor: string): Promise<ProtocolDescriptor> =>
    request<ProtocolDescriptor>(`${BASE}/${encodeURIComponent(vendor)}`),

  /** Create or update a descriptor */
  save: (descriptor: ProtocolDescriptor): Promise<ProtocolDescriptor> =>
    request<ProtocolDescriptor>(BASE, {
      method: 'POST',
      body: JSON.stringify(descriptor),
    }),

  /** Delete a descriptor */
  delete: (vendor: string): Promise<void> =>
    request<void>(`${BASE}/${encodeURIComponent(vendor)}`, {
      method: 'DELETE',
    }),

  /** Test an endpoint against a live device */
  test: (req: DescriptorTestRequest): Promise<DescriptorTestResponse> =>
    request<DescriptorTestResponse>(`${BASE}/test`, {
      method: 'POST',
      body: JSON.stringify(req),
    }),
};
