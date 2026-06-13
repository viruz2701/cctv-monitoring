import React, { createContext, useContext, useState, ReactNode, useEffect, useRef } from 'react';

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
    fileUrl?: string; // Replaces data
    fileName?: string;
    excelBuffer?: ArrayBuffer; // Temporary passing mechanism 
}

interface ReportsContextType {
    generatedReports: ReportHistoryItem[];
    addGeneratedReport: (report: ReportHistoryItem) => void;
    clearReportUrl: (id: string) => void;
}

const ReportsContext = createContext<ReportsContextType | undefined>(undefined);

export function ReportsProvider({ children }: { children: ReactNode }) {
    const [generatedReports, setGeneratedReports] = useState<ReportHistoryItem[]>([]);
    const pendingRevokes = useRef<string[]>([]);

    // Process safely queued memory revocations generated during state transitions
    useEffect(() => {
        if (pendingRevokes.current.length > 0) {
            pendingRevokes.current.forEach(url => URL.revokeObjectURL(url));
            pendingRevokes.current = [];
        }
    }, [generatedReports]);

    const generatedReportsRef = useRef(generatedReports);

    // Keep the ref strictly synchronized with the latest reports state without triggering effect runs
    useEffect(() => {
        generatedReportsRef.current = generatedReports;
    }, [generatedReports]);

    // On unmount (or provider destruction), safely revoke any dangling Blob URLs
    useEffect(() => {
        return () => {
            generatedReportsRef.current.forEach(report => {
                if (report.fileUrl) {
                    URL.revokeObjectURL(report.fileUrl);
                }
            });
        };
    }, []);

    // Periodically sweep and expire reports older than 30 days
    useEffect(() => {
        const checkExpirations = () => {
            const now = new Date().getTime();
            const THIRTY_DAYS_MS = 30 * 24 * 60 * 60 * 1000;

            setGeneratedReports(prev => {
                let changed = false;
                const next = prev.map(report => {
                    if (report.status === 'ready') {
                        const age = now - new Date(report.generatedAt).getTime();
                        if (age > THIRTY_DAYS_MS) {
                            changed = true;
                            if (report.fileUrl) pendingRevokes.current.push(report.fileUrl);
                            return { ...report, status: 'expired' as const, fileUrl: undefined };
                        }
                    }
                    return report;
                });
                return changed ? next : prev;
            });
        };

        const interval = setInterval(checkExpirations, 60000); // Check every minute
        checkExpirations(); // Check on mount

        return () => clearInterval(interval);
    }, []);

    const addGeneratedReport = (report: ReportHistoryItem) => {
        let newReport = { ...report };

        // Convert the excelBuffer into a Blob URL to avoid keeping raw data in memory
        if (newReport.excelBuffer) {
            const blob = new Blob([newReport.excelBuffer], { type: 'application/vnd.openxmlformats-officedocument.spreadsheetml.sheet' });
            newReport.fileUrl = URL.createObjectURL(blob);
            delete newReport.excelBuffer; // Don't save the raw buffer in state
        }

        setGeneratedReports((prev) => {
            const MAX_REPORTS = 50;
            let updatedList = [newReport, ...prev];

            // Memory Management Eviction Policy
            if (updatedList.length > MAX_REPORTS) {
                // Drop the oldest items and queue them to safely free their Blob URLs
                const evicted = updatedList.slice(MAX_REPORTS);
                evicted.forEach(item => {
                    if (item.fileUrl) pendingRevokes.current.push(item.fileUrl);
                });
                updatedList = updatedList.slice(0, MAX_REPORTS);
            }
            return updatedList;
        });
    };

    const clearReportUrl = (id: string) => {
        setGeneratedReports((prev) =>
            prev.map(item => {
                if (item.id === id) {
                    if (item.fileUrl) pendingRevokes.current.push(item.fileUrl);
                    return { ...item, fileUrl: undefined };
                }
                return item;
            })
        );
    };

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
