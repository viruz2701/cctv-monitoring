import { useState, useEffect, useMemo } from 'react';
import { useAlarms } from '../../hooks/useApiQuery';
import { AlertCircle, X, ChevronRight } from 'lucide-react';
import { useNavigate } from 'react-router-dom';
import { useTranslation } from 'react-i18next';

export function AlertBanner() {
    const { t } = useTranslation();
    const { data: apiAlarms = [] } = useAlarms();

    const alerts = useMemo(() => apiAlarms.map(a => ({
        id: a.device_id + '-' + a.timestamp,
        deviceId: a.device_id,
        deviceName: a.device_id,
        type: a.priority >= 3 ? 'error' : a.priority >= 2 ? 'warning' : 'info',
        message: a.description,
        timestamp: a.timestamp,
        status: 'active',
        priority: a.priority >= 4 ? 'critical' : a.priority >= 3 ? 'high' : a.priority >= 2 ? 'medium' : 'low',
        source: a.device_id,
        siteName: '',
    })), [apiAlarms]);
    const navigate = useNavigate();
    const activeAlerts = alerts.filter(a => a.status === 'active' && (a.type === 'error' || a.type === 'warning'));
    const [dismissed, setDismissed] = useState(false);
    const [lastCount, setLastCount] = useState(activeAlerts.length);

    useEffect(() => {
        if (activeAlerts.length > lastCount) {
            setDismissed(false);
        }
        setLastCount(activeAlerts.length);
    }, [activeAlerts.length, lastCount]);

    if (activeAlerts.length === 0 || dismissed) return null;

    const criticalCount = activeAlerts.filter(a => a.type === 'error').length;
    const warningCount = activeAlerts.filter(a => a.type === 'warning').length;

    return (
        <div className="bg-red-50 dark:bg-red-900/20 border border-red-200 dark:border-red-800 rounded-lg p-4 mb-6 relative overflow-hidden">
            <div className="absolute top-0 left-0 w-1 h-full bg-red-500 animate-pulse"></div>
            <div className="flex items-center justify-between flex-wrap gap-4">
                <div className="flex items-center gap-3">
                    <div className="p-2 bg-red-100 dark:bg-red-800/30 rounded-full">
                        <AlertCircle className="w-5 h-5 text-red-600 dark:text-red-400" />
                    </div>
                    <div>
                        {criticalCount > 0 ? (
                            <h3 className="text-sm font-semibold text-red-900 dark:text-red-200">
                                {t('critical_issues_detected', { count: criticalCount })}
                            </h3>
                        ) : (
                            <h3 className="text-sm font-semibold text-red-900 dark:text-red-200">
                                {t('warnings_detected')}
                            </h3>
                        )}
                        <p className="text-sm text-red-700 dark:text-red-300 mt-0.5">
                            {t('critical_warning_summary', { critical: criticalCount, warning: warningCount })}
                        </p>
                    </div>
                </div>
                <div className="flex items-center gap-2">
                    <button
                        onClick={() => navigate('/alerts')}
                        className="flex items-center gap-2 px-4 py-2 bg-white dark:bg-slate-800 border border-red-100 dark:border-red-900/50 rounded-lg text-sm font-medium text-red-700 dark:text-red-300 hover:bg-red-50 dark:hover:bg-red-900/30 transition-colors shadow-sm"
                    >
                        {t('view_alerts')} <ChevronRight className="w-4 h-4" />
                    </button>
                    <button
                        onClick={() => setDismissed(true)}
                        className="p-2 rounded-lg text-red-400 hover:text-red-600 dark:hover:text-red-300 hover:bg-red-100 dark:hover:bg-red-800/30 transition-colors"
                        aria-label="Dismiss alert banner"
                    >
                        <X className="w-4 h-4" />
                    </button>
                </div>
            </div>
        </div>
    );
}