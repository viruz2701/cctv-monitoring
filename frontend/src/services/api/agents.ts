// ═══════════════════════════════════════════════════════════════════════
// Agents API — EDGE-11: Agent Monitoring Dashboard
// ═══════════════════════════════════════════════════════════════════════

import { request } from './client';
import type {
  Agent,
  AgentListResponse,
  AgentDetailResponse,
  AgentCommandResponse,
} from '../../types/agent';

export const agentsApi = {
  /** GET /api/v1/agents — список всех edge-агентов */
  getAgents(): Promise<Agent[]> {
    return request<AgentListResponse>('/agents').then((r) => r.data);
  },

  /** GET /api/v1/agents/:id — детали агента */
  getAgent(id: string): Promise<Agent> {
    return request<AgentDetailResponse>(`/agents/${id}`).then((r) => r.data);
  },

  /** POST /api/v1/agents/:id/command — отправить команду агенту */
  sendCommand(agentId: string, command: string): Promise<{ status: 'queued' }> {
    return request<AgentCommandResponse>(`/agents/${agentId}/command`, {
      method: 'POST',
      body: JSON.stringify({ command }),
    }).then((r) => r.data);
  },

  /** DELETE /api/v1/agents/:id — удалить агента */
  deleteAgent(agentId: string): Promise<void> {
    return request<void>(`/agents/${agentId}`, { method: 'DELETE' });
  },
};
