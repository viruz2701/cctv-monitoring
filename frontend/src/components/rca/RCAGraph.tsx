import React, { useState, useEffect } from 'react';
import { api } from '../../services/api';
import { useTranslation } from 'react-i18next';
import { AlertTriangle, Loader2, AlertCircle, Info } from '../ui/Icons';

// ── Types ────────────────────────────────────────────────────────────

interface RCAGraphNodeData {
  label: string;
  device_type: string;
  status: string;
  is_root_cause: boolean;
  is_failed: boolean;
  is_healthy: boolean;
}

interface RCAGraphNode {
  id: string;
  data: RCAGraphNodeData;
  position: { x: number; y: number };
}

interface RCAGraphEdge {
  id: string;
  source: string;
  target: string;
  animated: boolean;
}

interface RCAGraphResponse {
  nodes: RCAGraphNode[];
  edges: RCAGraphEdge[];
  root_cause_id: string;
  failed_device_id: string;
  impact_description: string;
  recommendation: string;
  blast_radius: number;
}

interface Props {
  deviceId: string;
  onClose?: () => void;
}

const STATUS_CONFIG: Record<string, { color: string; bg: string; icon: string }> = {
  ONLINE: { color: '#059669', bg: '#d1fae5', icon: '✅' },
  OFFLINE: { color: '#dc2626', bg: '#fef2f2', icon: '🔴' },
  WARNING: { color: '#d97706', bg: '#fef3c7', icon: '⚠️' },
  SUSPENDED: { color: '#9333ea', bg: '#f3e8ff', icon: '⏸️' },
  DEGRADED: { color: '#ea580c', bg: '#fff7ed', icon: '🔻' },
  UNKNOWN: { color: '#64748b', bg: '#f1f5f9', icon: '❓' },
};

const TYPE_ICONS: Record<string, string> = {
  site: '🏢', switch: '🔀', nvr: '🖥️', dvr: '💾',
  camera: '📹', encoder: '🔌', server: '🖧', ups: '🔋', rack: '🗄️',
};

export default function RCAGraph({ deviceId, onClose }: Props) {
  const { t } = useTranslation();
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState<string | null>(null);
  const [graphData, setGraphData] = useState<RCAGraphResponse | null>(null);

  useEffect(() => { loadRCAData(); }, [deviceId]);

  const loadRCAData = async () => {
    setLoading(true);
    setError(null);
    try {
      const data = await api.getRCAGraph(deviceId);
      setGraphData(data);
    } catch (err: any) {
      setError(err.message || 'Failed to load RCA data');
    } finally {
      setLoading(false);
    }
  };

  if (loading) return (
    <div className="flex items-center justify-center py-16">
      <Loader2 className="w-8 h-8 animate-spin text-blue-500" />
      <span className="ml-3 text-slate-600">{t('rca_loading') || 'Анализ...'}</span>
    </div>
  );

  if (error) return (
    <div className="flex flex-col items-center py-16">
      <AlertCircle className="w-10 h-10 text-red-400 mb-3" />
      <p className="text-slate-600 mb-4">{error}</p>
      <button onClick={loadRCAData} className="px-4 py-2 bg-blue-50 text-blue-600 rounded-lg hover:bg-blue-100 text-sm font-medium">{t('retry') || 'Повтор'}</button>
    </div>
  );

  if (!graphData || !graphData.nodes.length) return (
    <div className="flex flex-col items-center py-16">
      <Info className="w-10 h-10 text-slate-400 mb-3" />
      <p className="text-slate-500">{t('rca_no_data') || 'Нет данных'}</p>
    </div>
  );

  const isRoot = graphData.root_cause_id === graphData.failed_device_id;

  return (
    <div className="space-y-6">
      {/* Impact Banner */}
      <div className={`p-4 rounded-lg border ${isRoot ? 'bg-red-50 border-red-200' : 'bg-amber-50 border-amber-200'}`}>
        <div className="flex items-start gap-3">
          <AlertTriangle className={`w-5 h-5 mt-0.5 ${isRoot ? 'text-red-600' : 'text-amber-600'}`} />
          <div className="flex-1">
            <p className="font-medium text-slate-900">{t('rca_impact') || 'Влияние'}</p>
            <p className="text-sm text-slate-600 mt-1">{graphData.impact_description}</p>
            {graphData.blast_radius > 0 && (
              <p className="text-xs text-slate-500 mt-1">{t('rca_blast_radius') || 'Затронуто'}: {graphData.blast_radius}</p>
            )}
          </div>
        </div>
      </div>

      {/* Graph */}
      <div className="overflow-x-auto bg-white rounded-lg border border-slate-200 p-4">
        <div className="relative" style={{ minWidth: Math.max(graphData.nodes.length * 160, 400), minHeight: Math.max((Math.max(...graphData.nodes.map(n => n.position.y)) || 100) + 160, 200) }}>
          {/* SVG Edges */}
          <svg className="absolute inset-0 w-full h-full pointer-events-none z-0">
            {graphData.edges.map(edge => {
              const src = graphData.nodes.find(n => n.id === edge.source);
              const tgt = graphData.nodes.find(n => n.id === edge.target);
              if (!src || !tgt) return null;
              const sx = src.position.x + 100, sy = src.position.y + 40;
              const tx = tgt.position.x + 100, ty = tgt.position.y;
              const my = (sy + ty) / 2;
              return (
                <path key={edge.id} d={`M ${sx} ${sy} C ${sx} ${my}, ${tx} ${my}, ${tx} ${ty}`}
                  fill="none" stroke={edge.animated ? '#dc2626' : '#94a3b8'}
                  strokeWidth={edge.animated ? 2.5 : 1.5}
                  strokeDasharray={edge.animated ? '6 3' : 'none'}
                  className={edge.animated ? 'animate-pulse' : ''}
                />
              );
            })}
          </svg>

          {/* Nodes */}
          {graphData.nodes.map(node => {
            const sc = STATUS_CONFIG[node.data.status] || STATUS_CONFIG.UNKNOWN;
            const ti = TYPE_ICONS[node.data.device_type] || '📡';
            const isRootCause = node.data.is_root_cause;
            const isFailedNode = node.data.is_failed;
            return (
              <div key={node.id} className="absolute flex flex-col items-center z-10 hover:scale-105 transition-transform"
                style={{ left: node.position.x, top: node.position.y, width: 200 }}>
                <div className={`w-full p-3 rounded-xl border-2 cursor-default ${isRootCause ? 'border-red-500 bg-red-50 shadow-lg shadow-red-200' : isFailedNode ? 'border-orange-400 bg-orange-50' : node.data.is_healthy ? 'border-emerald-300 bg-emerald-50' : 'border-slate-200 bg-white'}`}>
                  <div className="flex items-center justify-between mb-1">
                    <span className="text-lg">{ti}</span>
                    {isRootCause && <span className="px-1.5 py-0.5 text-[10px] font-bold text-white bg-red-500 rounded">ROOT</span>}
                    {isFailedNode && !isRootCause && <span className="px-1.5 py-0.5 text-[10px] font-bold text-white bg-orange-500 rounded">FAILED</span>}
                  </div>
                  <p className="text-sm font-semibold text-slate-900 truncate">{node.data.label}</p>
                  <div className="flex items-center gap-2 mt-1">
                    <span className="text-[10px] text-slate-500 uppercase">{node.data.device_type}</span>
                    <span className="inline-flex items-center gap-1 px-1.5 py-0.5 rounded text-[10px] font-medium" style={{ backgroundColor: sc.bg, color: sc.color }}>
                      {sc.icon} {node.data.status}
                    </span>
                  </div>
                </div>
                <div className="w-2 h-2 rounded-full mt-1" style={{ backgroundColor: sc.color }} />
              </div>
            );
          })}
        </div>
      </div>

      {/* Recommendation */}
      {graphData.recommendation && (
        <div className="p-4 bg-blue-50 rounded-lg border border-blue-200">
          <div className="flex items-start gap-3">
            <Info className="w-5 h-5 text-blue-600 mt-0.5" />
            <div>
              <p className="font-medium text-slate-900">{t('rca_recommendation') || 'Рекомендация'}</p>
              <p className="text-sm text-slate-600 mt-1">{graphData.recommendation}</p>
            </div>
          </div>
        </div>
      )}

      {onClose && (
        <div className="flex justify-end">
          <button onClick={onClose} className="px-4 py-2 text-sm font-medium text-slate-600 bg-slate-100 rounded-lg hover:bg-slate-200 transition-colors">
            {t('close') || 'Закрыть'}
          </button>
        </div>
      )}
    </div>
  );
}
