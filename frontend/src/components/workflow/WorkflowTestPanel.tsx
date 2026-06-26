// ═══════════════════════════════════════════════════════════════════════
// WorkflowTestPanel — панель тестирования workflow (P2-2.1)
//
// Позволяет:
//   - Запустить workflow с mock event
//   - Настроить mock event данные
//   - Просмотреть результаты выполнения (node by node)
//   - Сбросить результаты
// ═══════════════════════════════════════════════════════════════════════

import React, { useCallback, useState } from 'react';
import { useTranslation } from 'react-i18next';
import {
  Play,
  RotateCcw,
  CheckCircle2,
  AlertCircle,
  Loader2,
  Clock,
  X,
} from 'lucide-react';
import { useWorkflowStore } from '../../store/workflowStore';
import {
  type WorkflowTestResult,
  type WorkflowNodeStatus,
} from '../../types/workflow';

// ═══════════════════════════════════════════════════════════════════════
// Mock Event Templates
// ═══════════════════════════════════════════════════════════════════════

const MOCK_EVENT_TEMPLATES = [
  {
    label: 'Motion — Critical',
    event: {
      type: 'motion_detected',
      priority: 'critical',
      timestamp: new Date().toISOString(),
      camera_id: 'CAM-001',
      source: 'sensor_01',
      data: { zone: 'restricted', confidence: 0.95 },
    },
  },
  {
    label: 'Motion — Low Priority',
    event: {
      type: 'motion_detected',
      priority: 'low',
      timestamp: new Date().toISOString(),
      camera_id: 'CAM-002',
      source: 'sensor_02',
      data: { zone: 'perimeter', confidence: 0.4 },
    },
  },
  {
    label: 'Device Offline',
    event: {
      type: 'device_offline',
      priority: 'high',
      timestamp: new Date().toISOString(),
      camera_id: 'CAM-003',
      source: 'heartbeat',
      data: { last_seen: new Date(Date.now() - 300000).toISOString() },
    },
  },
  {
    label: 'Schedule Trigger',
    event: {
      type: 'schedule',
      priority: 'medium',
      timestamp: new Date().toISOString(),
      camera_id: 'CAM-001',
      source: 'cron',
      data: { schedule: 'night_patrol', recurrence: 'daily' },
    },
  },
];

// ═══════════════════════════════════════════════════════════════════════
// Simulate node execution (асинхронный)
// ═══════════════════════════════════════════════════════════════════════

async function simulateNodeExecution(
  nodeKind: string,
  config: Record<string, unknown>,
  mockEvent: Record<string, unknown>
): Promise<{ status: WorkflowNodeStatus; output?: unknown; error?: string }> {
  // Симуляция задержки
  await new Promise((r) => setTimeout(r, 300 + Math.random() * 700));

  switch (nodeKind) {
    case 'trigger': {
      // Trigger always succeeds — просто передаёт событие
      return {
        status: 'success',
        output: { event: mockEvent, matched: true },
      };
    }

    case 'condition': {
      const celExpr = (config.celExpression as string) ?? '';
      if (!celExpr) {
        return { status: 'error', error: 'No CEL expression defined' };
      }
      // Простейшая симуляция CEL: проверка priority
      const mockPriority = (mockEvent.priority as string) ?? '';
      const trueBranch = (config.trueBranch as string) ?? '';
      const falseBranch = (config.falseBranch as string) ?? '';

      // Симуляция: если в выражении упоминается приоритет — проверяем
      const evalResult =
        celExpr.includes(mockPriority) ||
        (celExpr.includes('critical') && mockPriority === 'critical') ||
        (celExpr.includes('high') && mockPriority === 'high');

      return {
        status: 'success',
        output: {
          condition: celExpr,
          result: evalResult,
          branch: evalResult ? (trueBranch || 'true') : (falseBranch || 'false'),
        },
      };
    }

    case 'action': {
      const actionType = (config.actionType as string) ?? 'unknown';
      const target = (config.target as string) ?? '—';
      // Симуляция: 90% успеха
      const success = Math.random() > 0.1;
      if (!success) {
        return {
          status: 'error',
          error: `Action "${actionType}" failed on target "${target}": timeout`,
        };
      }
      return {
        status: 'success',
        output: {
          action: actionType,
          target,
          executed_at: new Date().toISOString(),
          result: 'ok',
        },
      };
    }

    case 'delay': {
      const duration = Number(config.duration ?? 30);
      const unit = (config.unit as string) ?? 'seconds';
      // В тестовом режиме не ждём реально, но показываем
      return {
        status: 'success',
        output: {
          duration,
          unit,
          note: 'Delay skipped in test mode (would wait in production)',
        },
      };
    }

    default:
      return { status: 'error', error: `Unknown node kind: ${nodeKind}` };
  }
}

// ═══════════════════════════════════════════════════════════════════════
// Component
// ═══════════════════════════════════════════════════════════════════════

export function WorkflowTestPanel() {
  const { t } = useTranslation();
  const [mockEvent, setMockEvent] = useState<string>(
    JSON.stringify(MOCK_EVENT_TEMPLATES[0].event, null, 2)
  );
  const [isRunning, setIsRunning] = useState(false);
  const [results, setResults] = useState<WorkflowTestResult[]>([]);

  const nodes = useWorkflowStore((s) => s.nodes);
  const edges = useWorkflowStore((s) => s.edges);
  const updateNodeStatus = useWorkflowStore((s) => s.updateNodeStatus);
  const clearTestResults = useWorkflowStore((s) => s.clearTestResults);
  const toggleTestMode = useWorkflowStore((s) => s.toggleTestMode);

  // ─── Parse mock event ──────────────────────────────────────────────
  const parsedEvent = useCallback((): Record<string, unknown> | null => {
    try {
      return JSON.parse(mockEvent) as Record<string, unknown>;
    } catch {
      return null;
    }
  }, [mockEvent]);

  const isValidJson = parsedEvent() !== null;

  // ─── Run Test ──────────────────────────────────────────────────────
  const handleRunTest = useCallback(async () => {
    const event = parsedEvent();
    if (!event) return;

    setIsRunning(true);
    setResults([]);

    // Сброс статусов
    nodes.forEach((n) => updateNodeStatus(n.id, 'idle'));

    const testResults: WorkflowTestResult[] = [];

    // Топологическая сортировка по edges
    const visited = new Set<string>();
    const sortedNodes: typeof nodes = [];

    function visit(nodeId: string) {
      if (visited.has(nodeId)) return;
      visited.add(nodeId);
      // Сначала обрабатываем все precursor'ы
      edges
        .filter((e) => e.target === nodeId)
        .forEach((e) => visit(e.source));
      const node = nodes.find((n) => n.id === nodeId);
      if (node) sortedNodes.push(node);
    }

    // Стартуем с нод без incoming edges
    const startNodes = nodes.filter(
      (n) => !edges.some((e) => e.target === n.id)
    );
    startNodes.forEach((n) => visit(n.id));

    // Если топологическая сортировка не сработала — берём как есть
    const executionOrder = sortedNodes.length > 0 ? sortedNodes : nodes;

    for (const node of executionOrder) {
      const config = node.data.config ?? {};
      const kind = node.data.kind;

      updateNodeStatus(node.id, 'running');

      const startTime = performance.now();
      const result = await simulateNodeExecution(kind, config, event);
      const durationMs = Math.round(performance.now() - startTime);

      updateNodeStatus(node.id, result.status, result.error);

      testResults.push({
        nodeId: node.id,
        status: result.status,
        output: result.output,
        error: result.error,
        durationMs,
      });
    }

    setResults(testResults);
    setIsRunning(false);
  }, [nodes, edges, parsedEvent, updateNodeStatus]);

  // ─── Clear ─────────────────────────────────────────────────────────
  const handleClear = useCallback(() => {
    setResults([]);
    clearTestResults();
  }, [clearTestResults]);

  // ─── Node name lookup ──────────────────────────────────────────────
  const getNodeName = useCallback(
    (nodeId: string) => {
      const node = nodes.find((n) => n.id === nodeId);
      return node?.data.label ?? nodeId;
    },
    [nodes]
  );

  // ─── Status Icon ───────────────────────────────────────────────────
  const StatusIcon = ({ status }: { status: WorkflowNodeStatus }) => {
    switch (status) {
      case 'running':
        return <Loader2 className="w-4 h-4 text-blue-500 animate-spin" />;
      case 'success':
        return <CheckCircle2 className="w-4 h-4 text-green-500" />;
      case 'error':
        return <AlertCircle className="w-4 h-4 text-red-500" />;
      default:
        return <Clock className="w-4 h-4 text-slate-300" />;
    }
  };

  return (
    <div className="h-full flex flex-col bg-white dark:bg-slate-800 border-l border-slate-200 dark:border-slate-700">
      {/* ─── Header ──────────────────────────────────────────────── */}
      <div className="flex items-center justify-between px-4 py-3 border-b border-slate-200 dark:border-slate-700">
        <h3 className="text-sm font-bold text-slate-800 dark:text-slate-200 flex items-center gap-2">
          <Loader2 className="w-4 h-4 text-amber-500" />
          {t('test_mode') || 'Test Mode'}
        </h3>
        <div className="flex items-center gap-1">
          <button
            type="button"
            onClick={handleRunTest}
            disabled={isRunning || !isValidJson}
            className={[
              'flex items-center gap-1 px-3 py-1.5 rounded-lg text-sm font-medium transition-colors',
              isRunning
                ? 'bg-slate-200 dark:bg-slate-700 text-slate-400 cursor-not-allowed'
                : 'bg-blue-600 text-white hover:bg-blue-700',
            ].join(' ')}
          >
            <Play className="w-3.5 h-3.5" />
            {isRunning ? 'Running...' : 'Run'}
          </button>
          <button
            type="button"
            onClick={handleClear}
            disabled={results.length === 0}
            className="p-1.5 rounded-lg hover:bg-slate-100 dark:hover:bg-slate-700 text-slate-500 disabled:opacity-30"
            title="Clear results"
          >
            <RotateCcw className="w-4 h-4" />
          </button>
          <button
            type="button"
            onClick={toggleTestMode}
            className="p-1.5 rounded-lg hover:bg-slate-100 dark:hover:bg-slate-700 text-slate-500"
            title="Exit test mode"
          >
            <X className="w-4 h-4" />
          </button>
        </div>
      </div>

      {/* ─── Mock Event Editor ─────────────────────────────────────── */}
      <div className="p-4 border-b border-slate-200 dark:border-slate-700 space-y-2">
        <div className="flex items-center justify-between">
          <label className="text-xs font-semibold text-slate-500 dark:text-slate-400 uppercase tracking-wider">
            {t('mock_event') || 'Mock Event'}
          </label>
          <div className="flex gap-1">
            {MOCK_EVENT_TEMPLATES.map((template) => (
              <button
                key={template.label}
                type="button"
                onClick={() => setMockEvent(JSON.stringify(template.event, null, 2))}
                className="text-[10px] px-1.5 py-0.5 rounded bg-slate-100 dark:bg-slate-700 text-slate-600 dark:text-slate-400 hover:bg-slate-200 dark:hover:bg-slate-600"
              >
                {template.label}
              </button>
            ))}
          </div>
        </div>
        <textarea
          value={mockEvent}
          onChange={(e) => setMockEvent(e.target.value)}
          className={[
            'w-full h-28 p-2 font-mono text-xs leading-relaxed',
            'border rounded-lg resize-y',
            'bg-slate-50 dark:bg-slate-900/50',
            'text-slate-800 dark:text-slate-200',
            'focus:outline-none focus:ring-2 focus:ring-blue-500',
            isValidJson
              ? 'border-slate-300 dark:border-slate-600'
              : 'border-red-300 dark:border-red-700',
          ].join(' ')}
          spellCheck={false}
          placeholder='{"type": "motion_detected", ...}'
        />
        {!isValidJson && (
          <p className="text-xs text-red-500">Invalid JSON</p>
        )}
      </div>

      {/* ─── Results ──────────────────────────────────────────────── */}
      <div className="flex-1 overflow-y-auto p-4 space-y-2">
        <p className="text-xs font-semibold text-slate-500 dark:text-slate-400 uppercase tracking-wider">
          {t('results') || 'Results'}
          {results.length > 0 && (
            <span className="ml-2 text-slate-400 font-normal">
              ({results.filter((r) => r.status === 'success').length}/
              {results.length} passed)
            </span>
          )}
        </p>

        {results.length === 0 && !isRunning && (
          <div className="flex flex-col items-center justify-center py-8 text-slate-400">
            <Play className="w-8 h-8 mb-2 opacity-50" />
            <p className="text-xs text-center">
              {t('test_mode_hint') || 'Click Run to test this workflow with mock data'}
            </p>
          </div>
        )}

        {results.map((result, idx) => (
          <div
            key={result.nodeId}
            className={[
              'p-3 rounded-lg border',
              result.status === 'success'
                ? 'border-green-200 dark:border-green-800 bg-green-50 dark:bg-green-950/20'
                : result.status === 'error'
                  ? 'border-red-200 dark:border-red-800 bg-red-50 dark:bg-red-950/20'
                  : 'border-slate-200 dark:border-slate-700 bg-slate-50 dark:bg-slate-900/50',
            ].join(' ')}
          >
            <div className="flex items-center gap-2 mb-1">
              <StatusIcon status={result.status} />
              <span className="text-sm font-medium text-slate-800 dark:text-slate-200">
                {idx + 1}. {getNodeName(result.nodeId)}
              </span>
              <span className="ml-auto text-[10px] text-slate-400">
                {result.durationMs}ms
              </span>
            </div>

            {result.output && (
              <pre className="text-[10px] font-mono text-slate-600 dark:text-slate-400 mt-1 bg-white dark:bg-slate-900/50 p-1.5 rounded overflow-x-auto">
                {JSON.stringify(result.output, null, 2) as string}
              </pre>
            )}

            {result.error && (
              <p className="text-xs text-red-500 mt-1">{result.error}</p>
            )}
          </div>
        ))}
      </div>
    </div>
  );
}
