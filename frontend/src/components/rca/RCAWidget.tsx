// ═══════════════════════════════════════════════════════════════════════
// RCAWidget — Summary card for Root Cause Analysis
//
// P1-UX.7: RCA Widget in Device Overview
//   - Summary card: root cause, blast radius, affected devices
//   - "No RCA available" state
//   - Click to expand full RCA graph in modal
//   - Export RCA summary as PDF
//   - Real-time updates via WebSocket (optional, via wsHub)
// ═══════════════════════════════════════════════════════════════════════

import React, { useState, useEffect, useCallback, useRef } from 'react';
import { useTranslation } from 'react-i18next';
import {
    AlertTriangle,
    Loader2,
    AlertCircle,
    Info,
    Target,
    Radio,
    Expand,
    FileDown,
    X,
} from 'lucide-react';
import { api } from '../../services/api';
import { Card, CardHeader, CardBody, Badge, Modal } from '../ui';

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

interface RCAWidgetProps {
    deviceId: string;
}

// Лениво загружаем RCAGraph для модалки
const FullRCAGraph = React.lazy(() => import('./RCAGraph'));

// ── Helpers ──────────────────────────────────────────────────────────

const STATUS_BADGE_VARIANT: Record<string, 'success' | 'warning' | 'danger' | 'info' | 'neutral'> = {
    ONLINE: 'success',
    OFFLINE: 'danger',
    WARNING: 'warning',
    SUSPENDED: 'neutral',
    DEGRADED: 'warning',
    UNKNOWN: 'neutral',
};

function getRootCauseName(nodes: RCAGraphNode[], rootCauseId: string): string {
    const root = nodes.find(n => n.id === rootCauseId);
    return root?.data.label ?? 'Unknown';
}

function getAffectedDeviceTypes(nodes: RCAGraphNode[]): string[] {
    const types = new Set(nodes.map(n => n.data.device_type));
    return Array.from(types);
}

// ═══════════════════════════════════════════════════════════════════════
// Component
// ═══════════════════════════════════════════════════════════════════════

export function RCAWidget({ deviceId }: RCAWidgetProps) {
    const { t } = useTranslation();
    const [loading, setLoading] = useState(true);
    const [error, setError] = useState<string | null>(null);
    const [graphData, setGraphData] = useState<RCAGraphResponse | null>(null);
    const [modalOpen, setModalOpen] = useState(false);
    const printRef = useRef<HTMLDivElement>(null);

    useEffect(() => {
        loadRCAData();
    }, [deviceId]);

    const loadRCAData = useCallback(async () => {
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
    }, [deviceId]);

    // ── Export as PDF ─────────────────────────────────────────────

    const handleExportPDF = useCallback(() => {
        if (!graphData) return;
        // Формируем текстовый отчёт (нативный PDF — в будущем через jsPDF/report API)
        const lines: string[] = [
            `RCA Report — Device: ${deviceId}`,
            `Generated: ${new Date().toISOString()}`,
            '',
            `Root Cause: ${getRootCauseName(graphData.nodes, graphData.root_cause_id)}`,
            `Blast Radius: ${graphData.blast_radius} devices`,
            `Affected Types: ${getAffectedDeviceTypes(graphData.nodes).join(', ')}`,
            '',
            `Impact: ${graphData.impact_description}`,
            '',
            `Recommendation: ${graphData.recommendation}`,
            '',
            '── RCA Node Summary ──',
            ...graphData.nodes.map(n =>
                `  ${n.data.label} (${n.data.device_type}) — ${n.data.status}${n.data.is_root_cause ? ' [ROOT CAUSE]' : ''}${n.data.is_failed ? ' [FAILED]' : ''}`
            ),
        ];

        const blob = new Blob([lines.join('\n')], { type: 'text/plain' });
        const url = URL.createObjectURL(blob);
        const a = document.createElement('a');
        a.href = url;
        a.download = `rca-report-${deviceId}-${Date.now()}.txt`;
        a.click();
        URL.revokeObjectURL(url);
    }, [graphData, deviceId]);

    // ── Loading State ──────────────────────────────────────────────

    if (loading) {
        return (
            <Card>
                <CardHeader>
                    <div className="flex items-center gap-2">
                        <AlertTriangle className="w-4 h-4 text-amber-500" />
                        <span>{t('root_cause_analysis') || 'Root Cause Analysis'}</span>
                    </div>
                </CardHeader>
                <CardBody>
                    <div className="flex items-center justify-center py-6">
                        <Loader2 className="w-5 h-5 animate-spin text-blue-500" />
                        <span className="ml-2 text-sm text-slate-500">
                            {t('rca_loading') || 'Analysing...'}
                        </span>
                    </div>
                </CardBody>
            </Card>
        );
    }

    // ── Error State ────────────────────────────────────────────────

    if (error) {
        return (
            <Card>
                <CardHeader>
                    <div className="flex items-center gap-2">
                        <AlertTriangle className="w-4 h-4 text-amber-500" />
                        <span>{t('root_cause_analysis') || 'Root Cause Analysis'}</span>
                    </div>
                </CardHeader>
                <CardBody>
                    <div className="flex flex-col items-center py-6">
                        <AlertCircle className="w-8 h-8 text-red-400 mb-2" />
                        <p className="text-sm text-slate-600 dark:text-slate-400 mb-3">{error}</p>
                        <button
                            onClick={loadRCAData}
                            className="px-4 py-2 bg-blue-50 text-blue-600 rounded-lg hover:bg-blue-100 text-sm font-medium transition-colors"
                        >
                            {t('retry') || 'Retry'}
                        </button>
                    </div>
                </CardBody>
            </Card>
        );
    }

    // ── No RCA Available State ─────────────────────────────────────

    if (!graphData || !graphData.nodes.length) {
        return (
            <Card>
                <CardHeader>
                    <div className="flex items-center gap-2">
                        <AlertTriangle className="w-4 h-4 text-amber-500" />
                        <span>{t('root_cause_analysis') || 'Root Cause Analysis'}</span>
                    </div>
                </CardHeader>
                <CardBody>
                    <div className="flex flex-col items-center py-6">
                        <Info className="w-8 h-8 text-slate-300 dark:text-slate-600 mb-2" />
                        <p className="text-sm font-medium text-slate-600 dark:text-slate-400">
                            {t('rca_no_data') || 'No RCA data available'}
                        </p>
                        <p className="text-xs text-slate-400 dark:text-slate-500 mt-1 text-center max-w-sm">
                            {t('rca_no_data_description') || 'Root Cause Analysis will be available after an incident is detected on this device.'}
                        </p>
                    </div>
                </CardBody>
            </Card>
        );
    }

    // ── Summary Data ───────────────────────────────────────────────

    const rootCauseName = getRootCauseName(graphData.nodes, graphData.root_cause_id);
    const rootNode = graphData.nodes.find(n => n.id === graphData.root_cause_id);
    const rootStatus = rootNode?.data.status ?? 'UNKNOWN';
    const affectedTypes = getAffectedDeviceTypes(graphData.nodes);
    const failedCount = graphData.nodes.filter(n => n.data.is_failed).length;
    const isRoot = graphData.root_cause_id === graphData.failed_device_id;

    return (
        <>
            <Card>
                <CardHeader
                    action={
                        <div className="flex items-center gap-2">
                            {/* Expand button */}
                            <button
                                onClick={() => setModalOpen(true)}
                                className="flex items-center gap-1 text-xs text-blue-600 hover:text-blue-700 dark:text-blue-400 font-medium transition-colors"
                                aria-label={t('expand_rca_graph') || 'Expand RCA graph'}
                            >
                                <Expand className="w-3.5 h-3.5" />
                                <span className="hidden sm:inline">{t('expand') || 'Expand'}</span>
                            </button>

                            {/* Export PDF button */}
                            <button
                                onClick={handleExportPDF}
                                className="flex items-center gap-1 text-xs text-slate-500 hover:text-slate-700 dark:text-slate-400 dark:hover:text-slate-300 font-medium transition-colors"
                                aria-label={t('export_rca_pdf') || 'Export RCA as PDF'}
                            >
                                <FileDown className="w-3.5 h-3.5" />
                                <span className="hidden sm:inline">{t('export') || 'Export'}</span>
                            </button>
                        </div>
                    }
                >
                    <div className="flex items-center gap-2">
                        <AlertTriangle className={`w-4 h-4 ${isRoot ? 'text-red-500' : 'text-amber-500'}`} />
                        {t('root_cause_analysis') || 'Root Cause Analysis'}
                    </div>
                </CardHeader>

                <CardBody>
                    <div ref={printRef} className="space-y-4">
                        {/* Impact Banner */}
                        <div
                            className={`p-3 rounded-lg border ${
                                isRoot
                                    ? 'bg-red-50 dark:bg-red-900/20 border-red-200 dark:border-red-800/50'
                                    : 'bg-amber-50 dark:bg-amber-900/20 border-amber-200 dark:border-amber-800/50'
                            }`}
                        >
                            <div className="flex items-start gap-2">
                                <AlertTriangle
                                    className={`w-4 h-4 mt-0.5 shrink-0 ${
                                        isRoot ? 'text-red-600' : 'text-amber-600'
                                    }`}
                                />
                                <div className="flex-1 min-w-0">
                                    <p className="text-sm font-medium text-slate-900 dark:text-white">
                                        {graphData.impact_description}
                                    </p>
                                </div>
                            </div>
                        </div>

                        {/* Summary Grid */}
                        <div className="grid grid-cols-1 sm:grid-cols-3 gap-3">
                            {/* Root Cause */}
                            <div className="flex items-start gap-3 p-3 bg-slate-50 dark:bg-slate-900/30 rounded-lg border border-slate-100 dark:border-slate-800">
                                <div className="p-2 rounded-lg bg-red-100 dark:bg-red-900/30">
                                    <Target className="w-4 h-4 text-red-600 dark:text-red-400" />
                                </div>
                                <div className="min-w-0">
                                    <p className="text-[11px] font-medium text-slate-500 dark:text-slate-400 uppercase tracking-wider">
                                        {t('rca_root_cause') || 'Root Cause'}
                                    </p>
                                    <p className="text-sm font-semibold text-slate-900 dark:text-white truncate mt-0.5">
                                        {rootCauseName}
                                    </p>
                                    <Badge
                                        variant={STATUS_BADGE_VARIANT[rootStatus] ?? 'neutral'}
                                        size="sm"
                                        className="mt-1"
                                    >
                                        {rootStatus}
                                    </Badge>
                                </div>
                            </div>

                            {/* Blast Radius */}
                            <div className="flex items-start gap-3 p-3 bg-slate-50 dark:bg-slate-900/30 rounded-lg border border-slate-100 dark:border-slate-800">
                                <div className="p-2 rounded-lg bg-amber-100 dark:bg-amber-900/30">
                                    <Radio className="w-4 h-4 text-amber-600 dark:text-amber-400" />
                                </div>
                                <div className="min-w-0">
                                    <p className="text-[11px] font-medium text-slate-500 dark:text-slate-400 uppercase tracking-wider">
                                        {t('blast_radius') || 'Blast Radius'}
                                    </p>
                                    <p className="text-2xl font-bold text-slate-900 dark:text-white mt-0.5">
                                        {graphData.blast_radius}
                                    </p>
                                    <p className="text-xs text-slate-500 dark:text-slate-400 mt-0.5">
                                        {t('affected_devices') || 'affected devices'}
                                    </p>
                                </div>
                            </div>

                            {/* Affected Types */}
                            <div className="flex items-start gap-3 p-3 bg-slate-50 dark:bg-slate-900/30 rounded-lg border border-slate-100 dark:border-slate-800">
                                <div className="p-2 rounded-lg bg-blue-100 dark:bg-blue-900/30">
                                    <AlertTriangle className="w-4 h-4 text-blue-600 dark:text-blue-400" />
                                </div>
                                <div className="min-w-0">
                                    <p className="text-[11px] font-medium text-slate-500 dark:text-slate-400 uppercase tracking-wider">
                                        {t('device_types') || 'Device Types'}
                                    </p>
                                    <div className="flex flex-wrap gap-1 mt-1.5">
                                        {affectedTypes.map(type => (
                                            <Badge key={type} variant="info" size="sm">
                                                {type}
                                            </Badge>
                                        ))}
                                    </div>
                                    {failedCount > 0 && (
                                        <p className="text-xs text-red-500 mt-1.5">
                                            {failedCount} {t('failed') || 'failed'}
                                        </p>
                                    )}
                                </div>
                            </div>
                        </div>

                        {/* Recommendation */}
                        {graphData.recommendation && (
                            <div className="p-3 bg-blue-50 dark:bg-blue-900/20 rounded-lg border border-blue-200 dark:border-blue-800/50">
                                <div className="flex items-start gap-2">
                                    <Info className="w-4 h-4 text-blue-600 dark:text-blue-400 mt-0.5 shrink-0" />
                                    <div>
                                        <p className="text-xs font-medium text-slate-900 dark:text-white">
                                            {t('rca_recommendation') || 'Recommendation'}
                                        </p>
                                        <p className="text-xs text-slate-600 dark:text-slate-400 mt-0.5">
                                            {graphData.recommendation}
                                        </p>
                                    </div>
                                </div>
                            </div>
                        )}
                    </div>
                </CardBody>
            </Card>

            {/* ── Full RCA Graph Modal ───────────────────────────────── */}
            <Modal
                isOpen={modalOpen}
                onClose={() => setModalOpen(false)}
                title={t('root_cause_analysis') || 'Root Cause Analysis'}
                size="xl"
            >
                <React.Suspense
                    fallback={
                        <div className="flex items-center justify-center py-16">
                            <Loader2 className="w-8 h-8 animate-spin text-blue-500" />
                            <span className="ml-3 text-sm text-slate-500">
                                {t('loading_rca') || 'Loading RCA...'}
                            </span>
                        </div>
                    }
                >
                    <FullRCAGraph deviceId={deviceId} onClose={() => setModalOpen(false)} />
                </React.Suspense>
            </Modal>
        </>
    );
}

export default RCAWidget;
