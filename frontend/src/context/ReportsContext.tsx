// ═══════════════════════════════════════════════════════════════════════
// Reports Store (Zustand)
// ARCH-02: Миграция на Zustand для client-side state.
// Purely client-side state — report history with Blob URL management.
// ═══════════════════════════════════════════════════════════════════════

import { create } from 'zustand';
import React, { createContext, useContext, ReactNode } from 'react';

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

export const useReportsStore = create<ReportsState>()((set, get) => ({
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
// Auto-starts when the first ReportsProvider mounts
let expirationInterval: ReturnType<typeof setInterval> | null = null;

function startExpirationSweep() {
  if (expirationInterval) return;
  expirationInterval = setInterval(() => {
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
  }, 60000);

  // Check on start
  setTimeout(() => {
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
  }, 0);
}

function stopExpirationSweep() {
  if (expirationInterval) {
    clearInterval(expirationInterval);
    expirationInterval = null;
  }
}

// ── React Context wrapper for backward compatibility ─────────────────
// Provides the Zustand store through the same context API

interface ReportsContextType {
  generatedReports: ReportHistoryItem[];
  addGeneratedReport: (report: ReportHistoryItem) => void;
  clearReportUrl: (id: string) => void;
}

const ReportsContext = createContext<ReportsContextType | undefined>(undefined);

export function ReportsProvider({ children }: { children: ReactNode }) {
  const generatedReports = useReportsStore((s) => s.generatedReports);
  const addGeneratedReport = useReportsStore((s) => s.addGeneratedReport);
  const clearReportUrl = useReportsStore((s) => s.clearReportUrl);

  // Start/stop expiration sweep on mount/unmount
  React.useEffect(() => {
    startExpirationSweep();
    return () => stopExpirationSweep();
  }, []);

  // Revoke dangling Blob URLs on unmount
  React.useEffect(() => {
    return () => {
      const state = useReportsStore.getState();
      state.generatedReports.forEach(report => {
        if (report.fileUrl) {
          URL.revokeObjectURL(report.fileUrl);
        }
      });
    };
  }, []);

  return (
    <ReportsContext.Provider value={{ generatedReports, addGeneratedReport, clearReportUrl }}>
      {children}
    </ReportsContext.Provider>
  );
}

export function useReports() {
  const context = useContext(ReportsContext);
  if (context === undefined) {
    throw new Error('useReports must be used within a ReportsProvider');
  }
  return context;
}
