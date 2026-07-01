// ═══════════════════════════════════════════════════════════════════════
// Analytics & Logs API
// ARCH.2: Выделен из monolithic api.ts.
// ═══════════════════════════════════════════════════════════════════════

import { request } from './client';

// ─── Types ──────────────────────────────────────────────────────────

export interface Prediction {
  device_id: string;
  prediction_date: string;
  failure_probability: number;
  explanation: string;
  model_version?: string;
}

export interface CostData {
  site_id: string;
  site_name: string;
  device_type: string;
  device_count: number;
  maintenance_cost: number;
  energy_cost: number;
  labor_cost: number;
  spare_parts_cost: number;
  total_cost: number;
  month: string;
}

export interface CostTrend {
  month: string;
  maintenance_cost: number;
  energy_cost: number;
  labor_cost: number;
  spare_parts_cost: number;
  total_cost: number;
}

export interface TopExpensiveDevice {
  device_id: string;
  device_name: string;
  site_name: string;
  total_cost: number;
  breakdown: {
    maintenance: number;
    energy: number;
    labor: number;
    spare_parts: number;
  };
}

export interface VendorReliability {
  vendor: string;
  device_count: number;
  mtbf_hours: number;
  mttr_minutes: number;
  failure_rate: number;
  score: number;
}

export interface SLAMetrics {
  overall_compliance: number;
  total_breaches: number;
  avg_response_time: number;
  avg_resolution_time: number;
}

export interface ReliabilityData {
  vendors: VendorReliability[];
  overall_mtbf: number;
  overall_mttr: number;
  total_devices: number;
}

export interface ParsedLog {
  device_id: string;
  log_level: string;
  event_code: number;
  message: string;
  source: string;
  timestamp: string;
  raw?: string;
}

// ─── Predictions API ────────────────────────────────────────────────

export const predictionsApi = {
  getPredictions(deviceId?: string, limit?: number): Promise<Prediction[]> {
    const params = new URLSearchParams();
    if (deviceId) params.append('device_id', deviceId);
    if (limit) params.append('limit', String(limit));
    const query = params.toString() ? `?${params.toString()}` : '';
    return request<Prediction[] | null>(`/analytics/predictions${query}`).then((data) => data || []);
  },

  triggerRun(): Promise<{ status: string }> {
    return request<{ status: string }>('/analytics/predictions/run', {
      method: 'POST',
    });
  },
};

// ─── Cost Analysis API ──────────────────────────────────────────────

export const costApi = {
  getCostData(params?: { site_id?: string; months?: number }): Promise<CostData[]> {
    const query = new URLSearchParams();
    if (params?.site_id) query.append('site_id', params.site_id);
    if (params?.months) query.append('months', String(params.months));
    const qs = query.toString() ? `?${query.toString()}` : '';
    return request<CostData[] | null>(`/analytics/cost${qs}`).then((data) => data || []);
  },

  getCostTrend(months?: number): Promise<CostTrend[]> {
    const query = months ? `?months=${months}` : '';
    return request<CostTrend[] | null>(`/analytics/cost/trend${query}`).then((data) => data || []);
  },

  getTopExpensiveDevices(limit?: number): Promise<TopExpensiveDevice[]> {
    const query = limit ? `?limit=${limit}` : '';
    return request<TopExpensiveDevice[] | null>(`/analytics/cost/top${query}`).then((data) => data || []);
  },
};

// ─── Reliability API ────────────────────────────────────────────────

export const reliabilityApi = {
  getData(): Promise<ReliabilityData> {
    return request<ReliabilityData>('/analytics/reliability');
  },
};

// ─── SLA Metrics API ────────────────────────────────────────────────

export const slaApi = {
  getMetrics(): Promise<SLAMetrics> {
    return request<SLAMetrics>('/analytics/sla');
  },
};

// ─── Logs Search API ────────────────────────────────────────────────

export const logsApi = {
  search(params: {
    device_id?: string;
    level?: string;
    keyword?: string;
    time_from?: string;
    time_to?: string;
  }): Promise<ParsedLog[]> {
    const query = new URLSearchParams();
    if (params.device_id) query.append('device_id', params.device_id);
    if (params.level) query.append('level', params.level);
    if (params.keyword) query.append('keyword', params.keyword);
    if (params.time_from) query.append('time_from', params.time_from);
    if (params.time_to) query.append('time_to', params.time_to);
    return request<ParsedLog[]>(`/logs/search?${query.toString()}`);
  },
};

// ─── BI Query API (P2-BI) ──────────────────────────────────────────

export interface Field {
  key: string;
  label: string;
  type: string;
  agg?: string;
  sql_expr?: string;
}

export interface QueryTemplate {
  id: string;
  name: string;
  description: string;
  dimensions: Field[];
  measures: Field[];
  date_field: string;
}

export interface FilterCondition {
  field: string;
  op: string;
  value: unknown;
}

export interface QueryParams {
  template_id: string;
  dimensions?: string[];
  measures?: string[];
  filters?: FilterCondition[];
  time_from?: string;
  time_to?: string;
  limit?: number;
  offset?: number;
  order_by?: string;
  order_dir?: string;
}

export interface QueryResult {
  columns: string[];
  rows: unknown[][];
  total: number;
  took: string;
}

export const biApi = {
  getTemplates(): Promise<QueryTemplate[]> {
    return request<QueryTemplate[] | null>('/analytics/bi/templates').then((data) => data || []);
  },

  executeQuery(params: QueryParams): Promise<QueryResult> {
    return request<QueryResult>('/analytics/bi/query', {
      method: 'POST',
      body: JSON.stringify(params),
    });
  },
};
