import React, { useEffect, useState, useCallback } from 'react';
import { useTranslation } from 'react-i18next';
import { request } from '../../services/api';
import {
  Card,
  Badge,
  Button,
  Input,
  Modal,
  EmptyState,
  Tabs,
} from '../../components/ui';
import {
  AlertTriangle,
  Shield,
  FileText,
  Download,
  Eye,
  Clock,
  CheckCircle,
  XCircle,
  Activity,
  Search,
  FileSpreadsheet,
  ChevronRight,
  AlertOctagon,
  AlertCircle,
  Info,
  Server,
  Camera,
  Wifi,
  UserX,
} from '../../components/ui/Icons';

// ═══════════════════════════════════════════════════════════════════════
// Types
// ═══════════════════════════════════════════════════════════════════════

type IncidentSeverity = 'low' | 'medium' | 'high' | 'significant' | 'critical';
type IncidentType =
  | 'unauthorized_access'
  | 'data_breach'
  | 'system_failure'
  | 'malware'
  | 'physical_tampering'
  | 'denial_of_service'
  | 'configuration_change'
  | 'network_breach'
  | 'insider_threat'
  | 'third_party_breach';
type ReportPhase = 'early_warning' | 'notification' | 'final' | 'progress';
type IncidentStatus = 'open' | 'investigating' | 'contained' | 'resolved' | 'closed';

interface IncidentClassification {
  severity: IncidentSeverity;
  type: IncidentType;
  impact: string[];
  confidence: number;
  description: string;
  rationale: string;
}

interface TimelineEntry {
  id: string;
  timestamp: string;
  event: string;
  source: string;
  description: string;
  actor?: string;
  severity?: string;
}

interface IncidentAction {
  id: string;
  action: string;
  owner: string;
  status: string;
  performed_at: string;
  notes?: string;
}

interface Incident {
  id: string;
  classification: IncidentClassification;
  status: IncidentStatus;
  detected_at: string;
  reported_at?: string;
  resolved_at?: string;
  asset_id: string;
  zone: string;
  description: string;
  actions?: IncidentAction[];
  timeline: TimelineEntry[];
  reports?: NIS2Report[];
  created_at: string;
  updated_at: string;
}

interface NIS2Report {
  id: string;
  incident_id: string;
  phase: ReportPhase;
  generated_at: string;
  format: string;
  enisa_compliant: boolean;
}

// ═══════════════════════════════════════════════════════════════════════
// Sub-components
// ═══════════════════════════════════════════════════════════════════════

function SeverityBadge({ severity }: { severity: IncidentSeverity }) {
  const config: Record<IncidentSeverity, { bg: string; text: string; label: string; icon: React.ReactNode }> = {
    low: { bg: 'bg-emerald-100 dark:bg-emerald-900/30', text: 'text-emerald-700 dark:text-emerald-400', label: 'Low', icon: <Info className="w-3 h-3" /> },
    medium: { bg: 'bg-blue-100 dark:bg-blue-900/30', text: 'text-blue-700 dark:text-blue-400', label: 'Medium', icon: <AlertCircle className="w-3 h-3" /> },
    high: { bg: 'bg-amber-100 dark:bg-amber-900/30', text: 'text-amber-700 dark:text-amber-400', label: 'High', icon: <AlertTriangle className="w-3 h-3" /> },
    significant: { bg: 'bg-orange-100 dark:bg-orange-900/30', text: 'text-orange-700 dark:text-orange-400', label: 'Significant', icon: <AlertOctagon className="w-3 h-3" /> },
    critical: { bg: 'bg-red-100 dark:bg-red-900/30', text: 'text-red-700 dark:text-red-400', label: 'Critical', icon: <AlertTriangle className="w-3 h-3" /> },
  };
  const c = config[severity] || config.low;
  return (
    <span className={`inline-flex items-center gap-1 px-2 py-0.5 rounded text-xs font-medium ${c.bg} ${c.text}`}>
      {c.icon}{c.label}
    </span>
  );
}

function IncidentTypeBadge({ type }: { type: IncidentType }) {
  const config: Record<IncidentType, { bg: string; text: string; label: string; icon: React.ReactNode }> = {
    unauthorized_access: { bg: 'bg-red-100 dark:bg-red-900/30', text: 'text-red-700 dark:text-red-400', label: 'Unauthorized Access', icon: <UserX className="w-3 h-3" /> },
    data_breach: { bg: 'bg-purple-100 dark:bg-purple-900/30', text: 'text-purple-700 dark:text-purple-400', label: 'Data Breach', icon: <Shield className="w-3 h-3" /> },
    system_failure: { bg: 'bg-slate-100 dark:bg-slate-700', text: 'text-slate-700 dark:text-slate-300', label: 'System Failure', icon: <Server className="w-3 h-3" /> },
    malware: { bg: 'bg-rose-100 dark:bg-rose-900/30', text: 'text-rose-700 dark:text-rose-400', label: 'Malware', icon: <AlertTriangle className="w-3 h-3" /> },
    physical_tampering: { bg: 'bg-amber-100 dark:bg-amber-900/30', text: 'text-amber-700 dark:text-amber-400', label: 'Physical Tampering', icon: <Camera className="w-3 h-3" /> },
    denial_of_service: { bg: 'bg-orange-100 dark:bg-orange-900/30', text: 'text-orange-700 dark:text-orange-400', label: 'DoS/DDoS', icon: <Wifi className="w-3 h-3" /> },
    configuration_change: { bg: 'bg-cyan-100 dark:bg-cyan-900/30', text: 'text-cyan-700 dark:text-cyan-400', label: 'Config Change', icon: <Activity className="w-3 h-3" /> },
    network_breach: { bg: 'bg-indigo-100 dark:bg-indigo-900/30', text: 'text-indigo-700 dark:text-indigo-400', label: 'Network Breach', icon: <Wifi className="w-3 h-3" /> },
    insider_threat: { bg: 'bg-pink-100 dark:bg-pink-900/30', text: 'text-pink-700 dark:text-pink-400', label: 'Insider Threat', icon: <UserX className="w-3 h-3" /> },
    third_party_breach: { bg: 'bg-violet-100 dark:bg-violet-900/30', text: 'text-violet-700 dark:text-violet-400', label: 'Third Party', icon: <AlertTriangle className="w-3 h-3" /> },
  };
  const c = config[type] || config.system_failure;
  return (
    <span className={`inline-flex items-center gap-1 px-2 py-0.5 rounded text-xs font-medium ${c.bg} ${c.text}`}>
      {c.icon}{c.label}
    </span>
  );
}

function StatusBadge({ status }: { status: IncidentStatus }) {
  const config: Record<IncidentStatus, { bg: string; text: string; label: string; icon: React.ReactNode }> = {
    open: { bg: 'bg-blue-100 dark:bg-blue-900/30', text: 'text-blue-700 dark:text-blue-400', label: 'Open', icon: <AlertCircle className="w-3 h-3" /> },
    investigating: { bg: 'bg-amber-100 dark:bg-amber-900/30', text: 'text-amber-700 dark:text-amber-400', label: 'Investigating', icon: <Search className="w-3 h-3" /> },
    contained: { bg: 'bg-indigo-100 dark:bg-indigo-900/30', text: 'text-indigo-700 dark:text-indigo-400', label: 'Contained', icon: <Shield className="w-3 h-3" /> },
    resolved: { bg: 'bg-emerald-100 dark:bg-emerald-900/30', text: 'text-emerald-700 dark:text-emerald-400', label: 'Resolved', icon: <CheckCircle className="w-3 h-3" /> },
    closed: { bg: 'bg-slate-100 dark:bg-slate-700', text: 'text-slate-500 dark:text-slate-400', label: 'Closed', icon: <XCircle className="w-3 h-3" /> },
  };
  const c = config[status] || config.open;
  return (
    <span className={`inline-flex items-center gap-1 px-2 py-0.5 rounded text-xs font-medium ${c.bg} ${c.text}`}>
      {c.icon}{c.label}
    </span>
  );
}

function PhaseBadge({ phase }: { phase: ReportPhase }) {
  const config: Record<ReportPhase, { bg: string; text: string; label: string }> = {
    early_warning: { bg: 'bg-red-100 dark:bg-red-900/30', text: 'text-red-700 dark:text-red-400', label: '24h Early Warning' },
    notification: { bg: 'bg-orange-100 dark:bg-orange-900/30', text: 'text-orange-700 dark:text-orange-400', label: '72h Notification' },
    final: { bg: 'bg-blue-100 dark:bg-blue-900/30', text: 'text-blue-700 dark:text-blue-400', label: 'Final Report' },
    progress: { bg: 'bg-purple-100 dark:bg-purple-900/30', text: 'text-purple-700 dark:text-purple-400', label: 'Progress Update' },
  };
  const c = config[phase] || config.notification;
  return (
    <span className={`inline-flex items-center px-2 py-0.5 rounded text-xs font-medium ${c.bg} ${c.text}`}>{c.label}</span>
  );
}

// ═══════════════════════════════════════════════════════════════════════
// Incident List Tab
// ═══════════════════════════════════════════════════════════════════════

function IncidentListTab() {
  const [incidents, setIncidents] = useState<Incident[]>([]);
  const [selectedIncident, setSelectedIncident] = useState<Incident | null>(null);
  const [loading, setLoading] = useState(true);
  const [searchQuery, setSearchQuery] = useState('');
  const [severityFilter, setSeverityFilter] = useState<string>('');

  const fetchData = useCallback(async () => {
    setLoading(true);
    try {
      const params = new URLSearchParams();
      if (severityFilter) params.set('severity', severityFilter);
      params.set('limit', '50');
      params.set('offset', '0');
      const data = await request<Incident[]>(`/compliance/nis2/incidents?${params.toString()}`);
      setIncidents(data || []);
    } catch { /* ignore */ } finally { setLoading(false); }
  }, [severityFilter]);

  useEffect(() => { fetchData(); }, [fetchData]);

  const filteredIncidents = incidents.filter((inc) =>
    !searchQuery || inc.id.toLowerCase().includes(searchQuery.toLowerCase()) ||
    inc.description.toLowerCase().includes(searchQuery.toLowerCase()) ||
    inc.classification.type.toLowerCase().includes(searchQuery.toLowerCase())
  );

  if (loading) {
    return <div className="h-48 animate-pulse bg-slate-100 dark:bg-slate-800 rounded-lg" />;
  }

  return (
    <div className="space-y-4">
      {/* Filters */}
      <div className="flex items-center gap-3 flex-wrap">
        <div className="flex-1 min-w-[200px]">
          <Input
            placeholder="Search incidents..."
            value={searchQuery}
            onChange={(e) => setSearchQuery(e.target.value)}
          />
        </div>
        <select
          value={severityFilter}
          onChange={(e) => setSeverityFilter(e.target.value)}
          className="rounded-lg border border-slate-300 dark:border-slate-600 bg-white dark:bg-slate-800 px-3 py-2 text-sm"
        >
          <option value="">All Severities</option>
          <option value="critical">Critical</option>
          <option value="significant">Significant</option>
          <option value="high">High</option>
          <option value="medium">Medium</option>
          <option value="low">Low</option>
        </select>
      </div>

      {/* Incident List */}
      {filteredIncidents.length === 0 ? (
        <EmptyState
          icon={<AlertTriangle className="w-12 h-12" />}
          title="No Incidents"
          description="No NIS2 incidents found. All systems operational."
          action={{ label: 'Refresh', onClick: fetchData }}
        />
      ) : (
        <div className="space-y-3">
          {filteredIncidents.map((inc) => (
            <div
              key={inc.id}
              onClick={() => setSelectedIncident(inc)}
              className="cursor-pointer hover:shadow-md transition-shadow"
            >
              <Card>
                <div className="p-4">
                  <div className="flex items-start justify-between mb-2">
                    <div className="flex items-center gap-2 flex-wrap">
                      <SeverityBadge severity={inc.classification.severity} />
                      <IncidentTypeBadge type={inc.classification.type} />
                      <StatusBadge status={inc.status} />
                    </div>
                    <span className="text-xs text-slate-400 font-mono">{inc.id.slice(0, 16)}...</span>
                  </div>
                  <p className="text-sm font-medium mb-1">{inc.description}</p>
                  <div className="flex items-center gap-4 text-xs text-slate-500">
                    <span className="flex items-center gap-1">
                      <Clock className="w-3 h-3" />
                      {new Date(inc.detected_at).toLocaleString()}
                    </span>
                    <span>Zone: {inc.zone}</span>
                    <span>Impact: {inc.classification.impact.join(', ')}</span>
                    {inc.classification.confidence > 0 && (
                      <span>Confidence: {(inc.classification.confidence * 100).toFixed(0)}%</span>
                    )}
                  </div>
                </div>
              </Card>
            </div>
          ))}
        </div>
      )}

      {/* Incident Detail Modal */}
      <Modal
        isOpen={!!selectedIncident}
        onClose={() => setSelectedIncident(null)}
        title={`Incident: ${selectedIncident?.id.slice(0, 24) || ''}...`}
        size="lg"
      >
        {selectedIncident && (
          <IncidentDetailView
            incident={selectedIncident}
            onClose={() => setSelectedIncident(null)}
            onRefresh={fetchData}
          />
        )}
      </Modal>
    </div>
  );
}

// ═══════════════════════════════════════════════════════════════════════
// Incident Detail View
// ═══════════════════════════════════════════════════════════════════════

function IncidentDetailView({ incident, onClose, onRefresh }: {
  incident: Incident;
  onClose: () => void;
  onRefresh: () => void;
}) {
  const [activeSubTab, setActiveSubTab] = useState('overview');

  const handleExport = async (format: 'pdf' | 'xml', phase: ReportPhase) => {
    try {
      const response = await request<Blob>(`/compliance/nis2/incidents/${incident.id}/report`, {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify({ phase, format }),
        // Assume response is blob for download
      });

      // Handle blob download
      if (response instanceof Blob) {
        const url = URL.createObjectURL(response);
        const a = document.createElement('a');
        a.href = url;
        a.download = `NIS2-${incident.id.slice(0, 8)}-${phase}.${format}`;
        a.click();
        URL.revokeObjectURL(url);
      } else {
        // Fallback: open in new tab
        const data = response as unknown as string;
        const blob = new Blob([data], { type: format === 'pdf' ? 'application/pdf' : 'application/xml' });
        const url = URL.createObjectURL(blob);
        window.open(url, '_blank');
      }
    } catch (e) {
      console.error('Export failed:', e);
      alert('Failed to export report');
    }
  };

  const handleStatusUpdate = async (newStatus: IncidentStatus) => {
    try {
      await request(`/compliance/nis2/incidents/${incident.id}/status`, {
        method: 'PUT',
        body: JSON.stringify({ status: newStatus }),
      });
      onRefresh();
      onClose();
    } catch {
      alert('Failed to update status');
    }
  };

  const subTabs = [
    { id: 'overview', label: 'Overview', icon: <Eye className="w-4 h-4" /> },
    { id: 'timeline', label: `Timeline (${incident.timeline.length})`, icon: <Activity className="w-4 h-4" /> },
    { id: 'reports', label: `Reports (${incident.reports?.length || 0})`, icon: <FileText className="w-4 h-4" /> },
    { id: 'actions', label: `Actions (${incident.actions?.length || 0})`, icon: <CheckCircle className="w-4 h-4" /> },
  ];

  return (
    <div className="space-y-4">
      {/* Header Info */}
      <div className="flex items-start justify-between">
        <div>
          <div className="flex items-center gap-2 mb-2">
            <SeverityBadge severity={incident.classification.severity} />
            <IncidentTypeBadge type={incident.classification.type} />
            <StatusBadge status={incident.status} />
          </div>
          <p className="text-sm text-slate-600 dark:text-slate-400">{incident.description}</p>
        </div>
      </div>

      {/* Status Update Buttons */}
      {incident.status !== 'closed' && (
        <div className="flex gap-2 flex-wrap">
          {incident.status === 'open' && (
            <Button size="sm" onClick={() => handleStatusUpdate('investigating')}>
              <Search className="w-3 h-3 mr-1" /> Start Investigation
            </Button>
          )}
          {incident.status === 'investigating' && (
            <Button size="sm" onClick={() => handleStatusUpdate('contained')}>
              <Shield className="w-3 h-3 mr-1" /> Mark Contained
            </Button>
          )}
          {incident.status === 'contained' && (
            <Button size="sm" variant="primary" onClick={() => handleStatusUpdate('resolved')}>
              <CheckCircle className="w-3 h-3 mr-1" /> Mark Resolved
            </Button>
          )}
        </div>
      )}

      {/* Sub Tabs */}
      <Tabs tabs={subTabs} activeTab={activeSubTab} onChange={setActiveSubTab}>
        {activeSubTab === 'overview' && <IncidentOverviewTab incident={incident} onExport={handleExport} />}
        {activeSubTab === 'timeline' && <IncidentTimelineTab timeline={incident.timeline} />}
        {activeSubTab === 'reports' && <IncidentReportsTab reports={incident.reports} onExport={handleExport} />}
        {activeSubTab === 'actions' && <IncidentActionsTab actions={incident.actions} />}
      </Tabs>
    </div>
  );
}

// ═══════════════════════════════════════════════════════════════════════
// Overview Sub-tab
// ═══════════════════════════════════════════════════════════════════════

function IncidentOverviewTab({ incident, onExport }: {
  incident: Incident;
  onExport: (format: 'pdf' | 'xml', phase: ReportPhase) => void;
}) {
  const [exportPhase, setExportPhase] = useState<ReportPhase>('notification');
  const [exportFormat, setExportFormat] = useState<'pdf' | 'xml'>('xml');

  return (
    <div className="space-y-4">
      {/* Classification */}
      <div>
        <h4 className="text-sm font-semibold mb-2">Classification</h4>
        <div className="grid grid-cols-2 gap-3 text-sm">
          <div>
            <span className="text-xs text-slate-500">Severity</span>
            <p className="font-medium">{incident.classification.severity}</p>
          </div>
          <div>
            <span className="text-xs text-slate-500">Type</span>
            <p className="font-medium">{incident.classification.type.replace(/_/g, ' ')}</p>
          </div>
          <div>
            <span className="text-xs text-slate-500">Confidence</span>
            <p className="font-medium">{(incident.classification.confidence * 100).toFixed(0)}%</p>
          </div>
          <div>
            <span className="text-xs text-slate-500">Impact</span>
            <p className="font-medium">{incident.classification.impact.join(', ')}</p>
          </div>
        </div>
        <div className="mt-2">
          <span className="text-xs text-slate-500">Rationale</span>
          <p className="text-sm mt-0.5 text-slate-600 dark:text-slate-400">{incident.classification.rationale}</p>
        </div>
      </div>

      <hr className="border-slate-200 dark:border-slate-700" />

      {/* Details */}
      <div>
        <h4 className="text-sm font-semibold mb-2">Incident Details</h4>
        <div className="grid grid-cols-2 gap-3 text-sm">
          <div>
            <span className="text-xs text-slate-500">Incident ID</span>
            <p className="font-mono text-xs">{incident.id}</p>
          </div>
          <div>
            <span className="text-xs text-slate-500">Zone (IEC 62443)</span>
            <p className="font-medium">{incident.zone}</p>
          </div>
          <div>
            <span className="text-xs text-slate-500">Detected At</span>
            <p>{new Date(incident.detected_at).toLocaleString()}</p>
          </div>
          <div>
            <span className="text-xs text-slate-500">Status</span>
            <p className="font-medium capitalize">{incident.status}</p>
          </div>
          {incident.resolved_at && (
            <div>
              <span className="text-xs text-slate-500">Resolved At</span>
              <p>{new Date(incident.resolved_at).toLocaleString()}</p>
            </div>
          )}
        </div>
      </div>

      <hr className="border-slate-200 dark:border-slate-700" />

      {/* Export Section */}
      <div>
        <h4 className="text-sm font-semibold mb-2">NIS2 Report Export</h4>
        <p className="text-xs text-slate-500 mb-3">
          Generate NIS2 Directive Article 23 compliant reports. ENISA-compatible format.
        </p>
        <div className="flex items-center gap-3 mb-3">
          <select
            value={exportPhase}
            onChange={(e) => setExportPhase(e.target.value as ReportPhase)}
            className="rounded-lg border border-slate-300 dark:border-slate-600 bg-white dark:bg-slate-800 px-3 py-2 text-sm flex-1"
          >
            <option value="early_warning">24h Early Warning</option>
            <option value="notification">72h Notification</option>
            <option value="final">Final Report (1 month)</option>
            <option value="progress">Progress Update</option>
          </select>
          <select
            value={exportFormat}
            onChange={(e) => setExportFormat(e.target.value as 'pdf' | 'xml')}
            className="rounded-lg border border-slate-300 dark:border-slate-600 bg-white dark:bg-slate-800 px-3 py-2 text-sm"
          >
            <option value="xml">XML (ENISA)</option>
            <option value="pdf">PDF</option>
          </select>
        </div>
        <div className="flex gap-2">
          <Button onClick={() => onExport(exportFormat, exportPhase)}>
            <Download className="w-4 h-4 mr-1" /> Export Report
          </Button>
        </div>
      </div>
    </div>
  );
}

// ═══════════════════════════════════════════════════════════════════════
// Timeline Sub-tab
// ═══════════════════════════════════════════════════════════════════════

function IncidentTimelineTab({ timeline }: { timeline: TimelineEntry[] }) {
  if (!timeline || timeline.length === 0) {
    return (
      <EmptyState
        icon={<Activity className="w-12 h-12" />}
        title="Empty Timeline"
        description="No timeline entries recorded for this incident."
      />
    );
  }

  // Sort by timestamp ascending
  const sorted = [...timeline].sort((a, b) =>
    new Date(a.timestamp).getTime() - new Date(b.timestamp).getTime()
  );

  return (
    <div className="relative">
      {/* Timeline vertical line */}
      <div className="absolute left-[11px] top-2 bottom-2 w-0.5 bg-slate-200 dark:bg-slate-700" />

      <div className="space-y-4">
        {sorted.map((entry, idx) => (
          <div key={entry.id || idx} className="relative flex gap-4 pl-8">
            {/* Timeline dot */}
            <div className={`absolute left-0 top-1 w-6 h-6 rounded-full border-2 flex items-center justify-center
              ${entry.severity === 'critical' || entry.severity === 'high'
                ? 'border-red-400 bg-red-50 dark:bg-red-900/20'
                : 'border-blue-400 bg-blue-50 dark:bg-blue-900/20'
              }`}
            >
              <div className={`w-2 h-2 rounded-full
                ${entry.severity === 'critical' || entry.severity === 'high'
                  ? 'bg-red-500'
                  : 'bg-blue-500'
                }`}
              />
            </div>

            {/* Content */}
            <div className="flex-1 min-w-0">
              <div className="flex items-center gap-2 mb-0.5">
                <span className="text-xs font-medium text-slate-900 dark:text-white">{entry.event}</span>
                {(entry.severity === 'critical' || entry.severity === 'high') && (
                  <SeverityBadge severity={entry.severity as IncidentSeverity} />
                )}
              </div>
              <p className="text-xs text-slate-500 mb-0.5">{entry.description}</p>
              <div className="flex items-center gap-3 text-xs text-slate-400">
                <span className="flex items-center gap-1">
                  <Clock className="w-3 h-3" />
                  {new Date(entry.timestamp).toLocaleString()}
                </span>
                <span>Source: {entry.source}</span>
                {entry.actor && <span>Actor: {entry.actor}</span>}
              </div>
            </div>
          </div>
        ))}
      </div>
    </div>
  );
}

// ═══════════════════════════════════════════════════════════════════════
// Reports Sub-tab
// ═══════════════════════════════════════════════════════════════════════

function IncidentReportsTab({ reports, onExport }: {
  reports?: NIS2Report[];
  onExport: (format: 'pdf' | 'xml', phase: ReportPhase) => void;
}) {
  if (!reports || reports.length === 0) {
    return (
      <EmptyState
        icon={<FileText className="w-12 h-12" />}
        title="No Reports Generated"
        description="Generate NIS2 reports from the Overview tab."
      />
    );
  }

  return (
    <div className="space-y-3">
      {reports.map((report) => (
        <div key={report.id} className="border border-slate-200 dark:border-slate-700 rounded-lg p-3">
          <div className="flex items-center justify-between mb-2">
            <div className="flex items-center gap-2">
              <PhaseBadge phase={report.phase} />
              {report.enisa_compliant && (
                <span className="text-xs bg-green-100 dark:bg-green-900/30 text-green-700 dark:text-green-400 px-1.5 py-0.5 rounded">
                  ENISA Compliant
                </span>
              )}
            </div>
            <span className="text-xs text-slate-400 font-mono">{report.id.slice(0, 20)}...</span>
          </div>
          <div className="flex items-center justify-between text-xs text-slate-500">
            <span>Generated: {new Date(report.generated_at).toLocaleString()}</span>
            <div className="flex gap-2">
              <button
                onClick={() => onExport('xml', report.phase)}
                className="text-blue-600 hover:text-blue-800 font-medium"
              >
                <FileSpreadsheet className="w-3 h-3 inline mr-1" /> XML
              </button>
              <button
                onClick={() => onExport('pdf', report.phase)}
                className="text-blue-600 hover:text-blue-800 font-medium"
              >
                <FileText className="w-3 h-3 inline mr-1" /> PDF
              </button>
            </div>
          </div>
        </div>
      ))}
    </div>
  );
}

// ═══════════════════════════════════════════════════════════════════════
// Actions Sub-tab
// ═══════════════════════════════════════════════════════════════════════

function IncidentActionsTab({ actions }: { actions?: IncidentAction[] }) {
  if (!actions || actions.length === 0) {
    return (
      <EmptyState
        icon={<CheckCircle className="w-12 h-12" />}
        title="No Actions Taken"
        description="No response actions have been recorded for this incident."
      />
    );
  }

  return (
    <div className="space-y-2">
      {actions.map((action) => (
        <div key={action.id} className="border border-slate-200 dark:border-slate-700 rounded-lg p-3">
          <div className="flex items-center justify-between mb-1">
            <span className="text-sm font-medium">{action.action}</span>
            <span className={`text-xs px-1.5 py-0.5 rounded ${
              action.status === 'completed'
                ? 'bg-emerald-100 text-emerald-700 dark:bg-emerald-900/30 dark:text-emerald-400'
                : action.status === 'in_progress'
                  ? 'bg-amber-100 text-amber-700 dark:bg-amber-900/30 dark:text-amber-400'
                  : 'bg-slate-100 text-slate-600 dark:bg-slate-700 dark:text-slate-400'
            }`}>
              {action.status}
            </span>
          </div>
          <div className="flex items-center gap-3 text-xs text-slate-500">
            <span>Owner: {action.owner}</span>
            <span>Performed: {new Date(action.performed_at).toLocaleString()}</span>
          </div>
          {action.notes && (
            <p className="text-xs text-slate-500 mt-1">{action.notes}</p>
          )}
        </div>
      ))}
    </div>
  );
}

// ═══════════════════════════════════════════════════════════════════════
// Analytics Tab
// ═══════════════════════════════════════════════════════════════════════

function AnalyticsTab() {
  const [loading, setLoading] = useState(true);
  const [summary, setSummary] = useState<{
    total: number;
    by_severity: Record<string, number>;
    by_type: Record<string, number>;
    open: number;
    resolved: number;
  } | null>(null);

  useEffect(() => {
    const fetchSummary = async () => {
      try {
        const data = await request<{
          total: number;
          by_severity: Record<string, number>;
          by_type: Record<string, number>;
          open: number;
          resolved: number;
        }>('/compliance/nis2/summary');
        setSummary(data);
      } catch { /* ignore */ } finally { setLoading(false); }
    };
    fetchSummary();
  }, []);

  if (loading) {
    return <div className="h-48 animate-pulse bg-slate-100 dark:bg-slate-800 rounded-lg" />;
  }

  if (!summary || summary.total === 0) {
    return (
      <EmptyState
        icon={<Activity className="w-12 h-12" />}
        title="No Data"
        description="Incident analytics will appear here once incidents are recorded."
      />
    );
  }

  return (
    <div className="space-y-4">
      {/* Summary Cards */}
      <div className="grid grid-cols-2 md:grid-cols-4 gap-4">
        <Card>
          <div className="p-4 text-center">
            <div className="text-2xl font-bold text-slate-900 dark:text-white">{summary.total}</div>
            <div className="text-xs text-slate-500 mt-1">Total Incidents</div>
          </div>
        </Card>
        <Card>
          <div className="p-4 text-center">
            <div className="text-2xl font-bold text-amber-600">{summary.open}</div>
            <div className="text-xs text-slate-500 mt-1">Open</div>
          </div>
        </Card>
        <Card>
          <div className="p-4 text-center">
            <div className="text-2xl font-bold text-emerald-600">{summary.resolved}</div>
            <div className="text-xs text-slate-500 mt-1">Resolved</div>
          </div>
        </Card>
        <Card>
          <div className="p-4 text-center">
            <div className="text-2xl font-bold text-red-600">{summary.by_severity?.critical || 0}</div>
            <div className="text-xs text-slate-500 mt-1">Critical</div>
          </div>
        </Card>
      </div>

      {/* By Severity */}
      <Card>
        <div className="p-4">
          <h4 className="text-sm font-semibold mb-3">By Severity</h4>
          <div className="space-y-2">
            {Object.entries(summary.by_severity || {}).map(([sev, count]) => (
              <div key={sev} className="flex items-center gap-3">
                <SeverityBadge severity={sev as IncidentSeverity} />
                <div className="flex-1 h-4 bg-slate-100 dark:bg-slate-700 rounded-full overflow-hidden">
                  <div
                    className="h-full bg-current rounded-full transition-all"
                    style={{
                      width: `${(count / summary.total) * 100}%`,
                      opacity: 0.6,
                    }}
                  />
                </div>
                <span className="text-xs font-medium w-8 text-right">{count}</span>
              </div>
            ))}
          </div>
        </div>
      </Card>

      {/* By Type */}
      <Card>
        <div className="p-4">
          <h4 className="text-sm font-semibold mb-3">By Type</h4>
          <div className="grid grid-cols-2 gap-2">
            {Object.entries(summary.by_type || {}).map(([type, count]) => (
              <div key={type} className="flex items-center justify-between text-sm">
                <span className="text-slate-600 dark:text-slate-400">{type.replace(/_/g, ' ')}</span>
                <span className="font-medium">{count}</span>
              </div>
            ))}
          </div>
        </div>
      </Card>
    </div>
  );
}

// ═══════════════════════════════════════════════════════════════════════
// Main NIS2 Component
// ═══════════════════════════════════════════════════════════════════════

export default function NIS2() {
  const { t } = useTranslation();
  const [activeTab, setActiveTab] = useState('incidents');

  const tabs = [
    { id: 'incidents', label: 'Incidents', icon: <AlertTriangle className="w-4 h-4" /> },
    { id: 'analytics', label: 'Analytics', icon: <Activity className="w-4 h-4" /> },
  ];

  return (
    <div className="p-6 space-y-6">
      <div className="flex items-start justify-between">
        <div>
          <h1 className="text-2xl font-bold text-slate-900 dark:text-white">NIS2 Incident Reporting</h1>
          <p className="text-sm text-slate-500 dark:text-slate-400 mt-1">
            Directive (EU) 2022/2555 — Article 23 Incident Reporting. Automated classification,
            24h/72h reporting, and ENISA-compatible exports.
          </p>
        </div>
        <div className="flex items-center gap-2">
          <span className="text-xs bg-blue-100 dark:bg-blue-900/30 text-blue-700 dark:text-blue-400 px-2 py-1 rounded font-medium">
            ENISA Compliant
          </span>
        </div>
      </div>

      <Tabs tabs={tabs} activeTab={activeTab} onChange={setActiveTab}>
        {activeTab === 'incidents' && <IncidentListTab />}
        {activeTab === 'analytics' && <AnalyticsTab />}
      </Tabs>
    </div>
  );
}
