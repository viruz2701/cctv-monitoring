// ═══════════════════════════════════════════════════════════════════════
// Anomaly Detection API Service
// P2-AI.4: Anomaly Detection
// ═══════════════════════════════════════════════════════════════════════

import { request } from './client';

// ─── Types ──────────────────────────────────────────────────────────

export interface AnomalyResult {
  id: string;
  device_id: string;
  metric_type: string;
  current_value: number;
  mean_value: number;
  std_dev: number;
  z_score: number;
  severity: 'low' | 'medium' | 'high' | 'critical';
  status: 'new' | 'acknowledged' | 'resolved';
  description: string;
  detected_at: string;
  resolved_at?: string;
  trace_id: string;
}

export interface AnomalyListResponse {
  anomalies: AnomalyResult[];
  meta: {
    total: number;
    limit: number;
  };
}

export interface AnomalyStats {
  active_anomalies: number;
  metric_buffers: number;
  total_metric_points: number;
  nats_connected: boolean;
  last_evaluation: string;
}

export interface FeedMetricRequest {
  device_id: string;
  metric_type: string;
  value: number;
}

export interface FeedMetricResponse {
  status: string;
  anomaly?: AnomalyResult;
}

// ─── Anomaly Detection API ─────────────────────────────────────────

export const anomaliesApi = {
  /**
   * Get list of anomalies with optional filters.
   * GET /api/v1/ai/anomalies
   */
  getAnomalies(params?: {
    device_id?: string;
    metric_type?: string;
    severity?: string;
    status?: string;
    limit?: number;
  }): Promise<AnomalyListResponse> {
    const query = new URLSearchParams();
    if (params?.device_id) query.append('device_id', params.device_id);
    if (params?.metric_type) query.append('metric_type', params.metric_type);
    if (params?.severity) query.append('severity', params.severity);
    if (params?.status) query.append('status', params.status);
    if (params?.limit) query.append('limit', String(params.limit));
    const qs = query.toString() ? `?${query.toString()}` : '';
    return request<AnomalyListResponse>(`/ai/anomalies${qs}`);
  },

  /**
   * Feed a metric for anomaly analysis.
   * POST /api/v1/ai/anomalies/feed
   */
  feedMetric(data: FeedMetricRequest): Promise<FeedMetricResponse> {
    return request<FeedMetricResponse>('/ai/anomalies/feed', {
      method: 'POST',
      body: JSON.stringify(data),
    });
  },

  /**
   * Acknowledge an anomaly.
   * POST /api/v1/ai/anomalies/{id}/acknowledge
   */
  acknowledgeAnomaly(id: string): Promise<{ status: string }> {
    return request<{ status: string }>(`/ai/anomalies/${id}/acknowledge`, {
      method: 'POST',
    });
  },

  /**
   * Resolve an anomaly.
   * POST /api/v1/ai/anomalies/{id}/resolve
   */
  resolveAnomaly(id: string): Promise<{ status: string }> {
    return request<{ status: string }>(`/ai/anomalies/${id}/resolve`, {
      method: 'POST',
    });
  },

  /**
   * Get anomaly detection stats.
   * GET /api/v1/ai/anomalies/stats
   */
  getStats(): Promise<AnomalyStats> {
    return request<AnomalyStats>('/ai/anomalies/stats');
  },
};
