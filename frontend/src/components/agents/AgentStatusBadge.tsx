// ═══════════════════════════════════════════════════════════════════════
// AgentStatusBadge — статус edge-агента
// EDGE-11: Agent Monitoring Dashboard
//
// WCAG 2.1 AA: aria-label для screen readers, цветовой индикатор
// ═══════════════════════════════════════════════════════════════════════

import React from 'react';
import { useTranslation } from 'react-i18next';
import { Badge } from '../ui/Badge';
import type { AgentStatus } from '../../types/agent';

interface AgentStatusBadgeProps {
  status: AgentStatus;
}

/** Маппинг статуса агента на variant Badge'а */
const statusConfig: Record<AgentStatus, { variant: 'success' | 'danger' | 'warning'; labelKey: string }> = {
  online: { variant: 'success', labelKey: 'agentStatusOnline' },
  offline: { variant: 'danger', labelKey: 'agentStatusOffline' },
  error: { variant: 'warning', labelKey: 'agentStatusError' },
};

/**
 * AgentStatusBadge — бейдж статуса edge-агента.
 * - online: зелёный (success)
 * - offline: красный (danger)
 * - error: жёлтый (warning)
 */
export function AgentStatusBadge({ status }: AgentStatusBadgeProps) {
  const { t } = useTranslation();
  const config = statusConfig[status] ?? statusConfig.offline;

  return (
    <Badge
      variant={config.variant}
      dot
      ariaLabel={`Agent status: ${t(config.labelKey)}`}
    >
      {t(config.labelKey)}
    </Badge>
  );
}
