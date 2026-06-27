// ═══════════════════════════════════════════════════════════════════════
// RCA Graph API (AI-01)
// ARCH.2: Выделен из monolithic api.ts.
// ═══════════════════════════════════════════════════════════════════════

import { request } from './client';

// ─── Types ──────────────────────────────────────────────────────────

export interface RCAGraphNode {
  id: string;
  type: string;
  data: {
    label: string;
    device_type: string;
    status: string;
    is_root_cause: boolean;
    is_failed: boolean;
    is_healthy: boolean;
  };
  position: { x: number; y: number };
}

export interface RCAGraphEdge {
  id: string;
  source: string;
  target: string;
  type: string;
  animated: boolean;
}

export interface RCAGraphResponse {
  nodes: RCAGraphNode[];
  edges: RCAGraphEdge[];
  root_cause_id: string;
  failed_device_id: string;
  impact_description: string;
  recommendation: string;
  blast_radius: number;
}

// ─── API Methods ────────────────────────────────────────────────────

export const rcaApi = {
  getGraph(deviceId: string): Promise<RCAGraphResponse> {
    return request<RCAGraphResponse>(`/rca/${deviceId}`);
  },
};
