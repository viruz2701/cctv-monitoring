// ═══════════════════════════════════════════════════════════════════════
// MaintenanceTab — Maintenance schedule dashboard placeholder
// Lazy-loaded via React.lazy from DashboardHub
// ═══════════════════════════════════════════════════════════════════════

import React from 'react';
import { Wrench } from '../ui/Icons';

export default function MaintenanceTab() {
    return (
        <div className="flex flex-col items-center justify-center py-20 text-center">
            <div className="p-4 bg-amber-50 dark:bg-amber-900/20 rounded-full mb-4">
                <Wrench className="w-12 h-12 text-amber-500 dark:text-amber-400" />
            </div>
            <h2 className="text-xl font-semibold text-slate-900 dark:text-white mb-2">
                Maintenance Schedule
            </h2>
            <p className="text-slate-500 dark:text-slate-400 max-w-md">
                Maintenance schedule dashboard coming soon
            </p>
        </div>
    );
}
