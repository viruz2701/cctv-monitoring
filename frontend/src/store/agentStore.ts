// ═══════════════════════════════════════════════════════════════════════
// Agent Store (Zustand) — EDGE-11: Agent Monitoring Dashboard
//
// ARCH-02: UI-состояние для dashboard'а агентов.
// Server state (агенты из API) кешируется через React Query,
// этот store управляет только UI-состоянием.
// ═══════════════════════════════════════════════════════════════════════

import { create } from 'zustand';
import { agentsApi } from '../services/api/agents';
import type { Agent, AgentStats } from '../types/agent';

// ─── Store interface ────────────────────────────────────────────────

interface AgentStoreState {
  /** Список агентов */
  agents: Agent[];
  /** Вычисленная статистика */
  stats: AgentStats;
  /** Состояние загрузки */
  loading: boolean;
  /** Ошибка, если есть */
  error: string | null;

  /** Загрузить агентов с сервера */
  fetchAgents: () => Promise<void>;
  /** Отправить команду агенту */
  sendCommand: (agentId: string, command: string) => Promise<boolean>;
  /** Удалить агента */
  deleteAgent: (agentId: string) => Promise<boolean>;
  /** Очистить ошибку */
  clearError: () => void;
}

// ─── Helpers ────────────────────────────────────────────────────────

function computeStats(agents: Agent[]): AgentStats {
  return {
    total: agents.length,
    online: agents.filter((a) => a.status === 'online').length,
    offline: agents.filter((a) => a.status === 'offline').length,
    errors: agents.filter((a) => a.status === 'error').length,
  };
}

// ─── Store ──────────────────────────────────────────────────────────

export const useAgentStore = create<AgentStoreState>()((set, get) => ({
  agents: [],
  stats: { total: 0, online: 0, offline: 0, errors: 0 },
  loading: false,
  error: null,

  fetchAgents: async () => {
    set({ loading: true, error: null });
    try {
      const agents = await agentsApi.getAgents();
      set({ agents, stats: computeStats(agents), loading: false });
    } catch (err) {
      const message = err instanceof Error ? err.message : 'Failed to fetch agents';
      set({ error: message, loading: false });
    }
  },

  sendCommand: async (agentId: string, command: string) => {
    try {
      await agentsApi.sendCommand(agentId, command);
      return true;
    } catch (err) {
      const message = err instanceof Error ? err.message : 'Failed to send command';
      set({ error: message });
      return false;
    }
  },

  deleteAgent: async (agentId: string) => {
    try {
      await agentsApi.deleteAgent(agentId);
      // Удаляем агента из локального списка
      const agents = get().agents.filter((a) => a.id !== agentId);
      set({ agents, stats: computeStats(agents) });
      return true;
    } catch (err) {
      const message = err instanceof Error ? err.message : 'Failed to delete agent';
      set({ error: message });
      return false;
    }
  },

  clearError: () => set({ error: null }),
}));

// ─── Selector hooks для оптимальных re-render'ов ────────────────────

export const useAgentList = () => useAgentStore((s) => s.agents);
export const useAgentStats = () => useAgentStore((s) => s.stats);
export const useAgentLoading = () => useAgentStore((s) => s.loading);
export const useAgentError = () => useAgentStore((s) => s.error);
