// ═══════════════════════════════════════════════════════════════════════
// PerformanceTab — Performance metrics dashboard placeholder
// Lazy-loaded via React.lazy from DashboardHub
// ═══════════════════════════════════════════════════════════════════════

import React from 'react';
import { Gauge } from '../ui/Icons';

export default function PerformanceTab() {
    return (
        <div className="flex flex-col items-center justify-center py-20 text-center">
            <div className="p-4 bg-emerald-50 dark:bg-emerald-900/20 rounded-full mb-4">
                <Gauge className="w-12 h-12 text-emerald-500 dark:text-emerald-400" />
            </div>
            <h2 className="text-xl font-semibold text-slate-900 dark:text-white mb-2">
                Performance Metrics
            </h2>
            <p className="text-slate-500 dark:text-slate-400 max-w-md">
                Performance metrics dashboard coming soon
            </p>
        </div>
    );
}
