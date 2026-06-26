// ═══════════════════════════════════════════════════════════════════════
// Reports Store (Zustand)
// ARCH-02: Client-side state для отчётов. Blob URL management.
// ═══════════════════════════════════════════════════════════════════════

import { create } from 'zustand';

export interface ReportHistoryItem {
  id: string;
  name: string;
  type: string;
  format: string;
  dateRange: string;
  generatedAt: string;
  generatedBy: string;
  status: 'ready' | 'expired';
  size: string;
  fileUrl?: string;
  fileName?: string;
  excelBuffer?: ArrayBuffer;
}

interface ReportsState {
  generatedReports: ReportHistoryItem[];
  addGeneratedReport: (report: ReportHistoryItem) => void;
  clearReportUrl: (id: string) => void;
}

const THIRTY_DAYS_MS = 30 * 24 * 60 * 60 * 1000;
const MAX_REPORTS = 50;

// Pending Blob URLs to revoke (processed after state updates)
let pendingRevokes: string[] = [];

function processPendingRevokes() {
  if (pendingRevokes.length > 0) {
    pendingRevokes.forEach(url => URL.revokeObjectURL(url));
    pendingRevokes = [];
  }
}

function queueRevoke(url?: string) {
  if (url) pendingRevokes.push(url);
}

export const useReportsStore = create<ReportsState>()((set) => ({
  generatedReports: [],

  addGeneratedReport: (report) => {
    const newReport = { ...report };

    // Convert excelBuffer into Blob URL
    if (newReport.excelBuffer) {
      const blob = new Blob(
        [newReport.excelBuffer],
        { type: 'application/vnd.openxmlformats-officedocument.spreadsheetml.sheet' }
      );
      newReport.fileUrl = URL.createObjectURL(blob);
      delete newReport.excelBuffer;
    }

    set((state) => {
      processPendingRevokes();
      const updatedList = [newReport, ...state.generatedReports];

      // Evict excess reports
      if (updatedList.length > MAX_REPORTS) {
        const evicted = updatedList.slice(MAX_REPORTS);
        evicted.forEach(item => queueRevoke(item.fileUrl));
        return { generatedReports: updatedList.slice(0, MAX_REPORTS) };
      }
      return { generatedReports: updatedList };
    });
  },

  clearReportUrl: (id) => {
    set((state) => ({
      generatedReports: state.generatedReports.map(item => {
        if (item.id === id) {
          queueRevoke(item.fileUrl);
          return { ...item, fileUrl: undefined };
        }
        return item;
      }),
    }));
  },
}));

// ── Periodically expire reports older than 30 days ──────────────────

let expirationInterval: ReturnType<typeof setInterval> | null = null;

function runExpirationSweep() {
  const now = Date.now();
  const state = useReportsStore.getState();
  let changed = false;

  const next = state.generatedReports.map(report => {
    if (report.status === 'ready') {
      const age = now - new Date(report.generatedAt).getTime();
      if (age > THIRTY_DAYS_MS) {
        changed = true;
        queueRevoke(report.fileUrl);
        return { ...report, status: 'expired' as const, fileUrl: undefined };
      }
    }
    return report;
  });

  if (changed) {
    useReportsStore.setState({ generatedReports: next });
    processPendingRevokes();
  }
}

export function startReportExpirationSweep() {
  if (expirationInterval) return;
  expirationInterval = setInterval(runExpirationSweep, 60000);
  // Run immediately on start
  setTimeout(runExpirationSweep, 0);
}

export function stopReportExpirationSweep() {
  if (expirationInterval) {
    clearInterval(expirationInterval);
    expirationInterval = null;
  }
}
