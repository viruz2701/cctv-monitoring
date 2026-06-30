// ═══════════════════════════════════════════════════════════════════════
// Agent Types — EDGE-11: Agent Monitoring Dashboard
// ═══════════════════════════════════════════════════════════════════════

export type AgentStatus = 'online' | 'offline' | 'error';

export interface AgentTraffic {
  /** Входящий трафик, байт/с */
  in: number;
  /** Исходящий трафик, байт/с */
  out: number;
}

export interface Agent {
  id: string;
  name: string;
  site: string;
  status: AgentStatus;
  /** ISO timestamp последней активности */
  lastSeen: string;
  /** Версия edge-agent ПО */
  version: string;
  /** Трафик агента */
  traffic: AgentTraffic;
  /** Количество ошибок за последние 24ч */
  errors: number;
  /** Uptime в секундах */
  uptime: number;
  /** Загрузка CPU, 0-100 */
  cpu: number;
  /** Использование памяти, 0-100 */
  memory: number;
}

export interface AgentStats {
  total: number;
  online: number;
  offline: number;
  errors: number;
}

/** Ответ API со списком агентов */
export interface AgentListResponse {
  success: boolean;
  data: Agent[];
}

/** Ответ API с деталями агента */
export interface AgentDetailResponse {
  success: boolean;
  data: Agent;
}

/** Ответ API на команду */
export interface AgentCommandResponse {
  success: boolean;
  data: {
    status: 'queued';
  };
}
