// ═══════════════════════════════════════════════════════════════════════
// WorkflowNode — custom node for React Flow (P2-2.1)
//
// Единый кастомный nodeType для всех 4 видов нод:
//   - trigger    (фиолетовый) — событие
//   - condition  (янтарный)   — CEL условие, branch
//   - action     (синий)      — действие
//   - delay      (бирюзовый)  — ожидание
//
// Каждая нода имеет:
//   - Иконку + label
//   - Status indicator (idle/running/success/error)
//   - Input/Output handles для соединений
//   - Condition node: true/false handle labels
// ═══════════════════════════════════════════════════════════════════════

import React, { memo } from 'react';
import { Handle, Position } from '@xyflow/react';
import {
  Zap,
  GitBranch,
  Play,
  Timer,
  Loader2,
  CheckCircle2,
  AlertCircle,
} from 'lucide-react';
import type { WorkflowNodeData, WorkflowNodeKind } from '../../types/workflow';

// ═══════════════════════════════════════════════════════════════════════
// Style config per node kind
// ═══════════════════════════════════════════════════════════════════════

interface NodeStyle {
  border: string;
  bg: string;
  bgDark: string;
  accent: string;
  icon: React.ElementType;
}

const NODE_STYLES: Record<WorkflowNodeKind, NodeStyle> = {
  trigger: {
    border: 'border-purple-500',
    bg: 'bg-purple-50',
    bgDark: 'dark:bg-purple-950/30',
    accent: 'text-purple-600 dark:text-purple-400',
    icon: Zap,
  },
  condition: {
    border: 'border-amber-500',
    bg: 'bg-amber-50',
    bgDark: 'dark:bg-amber-950/30',
    accent: 'text-amber-600 dark:text-amber-400',
    icon: GitBranch,
  },
  action: {
    border: 'border-blue-500',
    bg: 'bg-blue-50',
    bgDark: 'dark:bg-blue-950/30',
    accent: 'text-blue-600 dark:text-blue-400',
    icon: Play,
  },
  delay: {
    border: 'border-teal-500',
    bg: 'bg-teal-50',
    bgDark: 'dark:bg-teal-950/30',
    accent: 'text-teal-600 dark:text-teal-400',
    icon: Timer,
  },
};

// ═══════════════════════════════════════════════════════════════════════
// Status indicator
// ═══════════════════════════════════════════════════════════════════════

function StatusBadge({ status }: { status?: string }) {
  switch (status) {
    case 'running':
      return <Loader2 className="w-4 h-4 text-blue-500 animate-spin" />;
    case 'success':
      return <CheckCircle2 className="w-4 h-4 text-green-500" />;
    case 'error':
      return <AlertCircle className="w-4 h-4 text-red-500" />;
    default:
      return null;
  }
}

// ═══════════════════════════════════════════════════════════════════════
// Props — принимаем data напрямую как any для совместимости с @xyflow/react
// ═══════════════════════════════════════════════════════════════════════

interface WorkflowNodeComponentProps {
  data: WorkflowNodeData;
  selected?: boolean;
}

function WorkflowNodeComponent({ data, selected }: WorkflowNodeComponentProps) {
  const { kind, label, description, config, status, errorMessage } = data;
  const style = NODE_STYLES[kind] ?? NODE_STYLES.action;
  const Icon = style.icon;

  const detailText =
    kind === 'trigger'
      ? `Event: ${(config.eventType as string) ?? '—'}`
      : kind === 'condition'
        ? (config.celExpression as string) || '—'
        : kind === 'action'
          ? `${(config.actionType as string) ?? '—'} → ${(config.target as string) ?? '—'}`
          : kind === 'delay'
            ? `${String(config.duration ?? '—')} ${config.unit as string ?? 's'}`
            : '';

  return (
    <div
      className={[
        'relative px-4 py-3 rounded-xl border-2 min-w-[180px] max-w-[260px]',
        'shadow-sm backdrop-blur-sm',
        'transition-all duration-200',
        style.bg,
        style.bgDark,
        style.border,
        selected ? 'ring-2 ring-offset-2 ring-blue-500 shadow-lg' : '',
        status === 'running' ? 'animate-pulse' : '',
        status === 'error' ? 'ring-2 ring-red-400' : '',
      ].join(' ')}
    >
      {/* ─── Input Handle ───────────────────────────────────────────── */}
      <Handle
        type="target"
        position={Position.Left}
        className="!w-3 !h-3 !border-2 !border-slate-400 !bg-white dark:!bg-slate-800"
      />

      {/* ─── Header: Icon + Label + Status ──────────────────────────── */}
      <div className="flex items-center gap-2 mb-1">
        <Icon className={`w-4 h-4 ${style.accent}`} />
        <span className="text-sm font-semibold text-slate-800 dark:text-slate-200 truncate">
          {label}
        </span>
        <div className="ml-auto">
          <StatusBadge status={status} />
        </div>
      </div>

      {/* ─── Description ────────────────────────────────────────────── */}
      {description && (
        <p className="text-xs text-slate-500 dark:text-slate-400 mb-1 leading-tight">
          {description}
        </p>
      )}

      {/* ─── Detail / Config Preview ────────────────────────────────── */}
      <p className="text-[11px] font-mono text-slate-400 dark:text-slate-500 truncate">
        {detailText}
      </p>

      {/* ─── Error Message ──────────────────────────────────────────── */}
      {errorMessage && (
        <p className="text-[11px] text-red-500 mt-1 leading-tight">
          {errorMessage}
        </p>
      )}

      {/* ─── Output Handle(s) ───────────────────────────────────────── */}
      {kind === 'condition' ? (
        <>
          <Handle
            type="source"
            position={Position.Right}
            id="true"
            className="!w-3 !h-3 !border-2 !border-green-500 !bg-white dark:!bg-slate-800"
            style={{ top: '40%' }}
          />
          <span className="absolute text-[10px] text-green-600 dark:text-green-400 font-medium"
            style={{ right: -28, top: '36%' }}>
            T
          </span>
          <Handle
            type="source"
            position={Position.Right}
            id="false"
            className="!w-3 !h-3 !border-2 !border-red-500 !bg-white dark:!bg-slate-800"
            style={{ top: '70%' }}
          />
          <span className="absolute text-[10px] text-red-600 dark:text-red-400 font-medium"
            style={{ right: -28, top: '66%' }}>
            F
          </span>
        </>
      ) : (
        <Handle
          type="source"
          position={Position.Right}
          className="!w-3 !h-3 !border-2 !border-slate-400 !bg-white dark:!bg-slate-800"
        />
      )}
    </div>
  );
}

export const WorkflowNode = memo(WorkflowNodeComponent);
