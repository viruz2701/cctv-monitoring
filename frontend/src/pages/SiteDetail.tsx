import React, { useMemo } from 'react';
import { useParams, useNavigate } from 'react-router-dom';
import { useTranslation } from 'react-i18next';
import { Breadcrumbs } from '../components/ui/Breadcrumbs';
import {
    Building2, MapPin, ArrowLeft, Camera,
    Wifi, WifiOff, Users, Edit, Activity,
} from '../components/ui/Icons';
import {
    Card, CardHeader, CardBody, Button, Badge,
    StatusBadge, HealthBadge,
} from '../components/ui';
import { useSite, useDevices } from '../hooks/useApiQuery';
import { getArrayData } from '../utils/helpers';
import { SkeletonDetailPage } from '../components/layout';
import type { Site, Device } from '../types';

// ─── Helpers ──────────────────────────────────────────────────────────────
function formatDate(dateStr: string, locale: string): string {
    return new Date(dateStr).toLocaleDateString(locale, {
        month: 'short',
        day: 'numeric',
        year: 'numeric',
        hour: '2-digit',
        minute: '2-digit',
    });
}

// ─── Main Component ───────────────────────────────────────────────────────
export const SiteDetail: React.FC = () => {
    const { t, i18n } = useTranslation();
    const { siteId } = useParams<{ siteId: string }>();
    const navigate = useNavigate();

    const { data: rawSite, isLoading: siteLoading } = useSite(siteId || '');
    const { data: rawDevices } = useDevices();

    const site = rawSite as Site | undefined;
    const allDevices = getArrayData<Record<string, any>>(rawDevices);

    // Filter devices belonging to this site
    const siteDevices = useMemo(() => {
        if (!site || !allDevices.length) return [];
        return allDevices
            .filter((d: any) => d.site_id === site.id || d.location === site.name)
            .map((d: any) => ({
                id: d.device_id || d.id,
                name: d.name || d.device_id,
                status: (d.status || 'offline').toLowerCase(),
                type: d.vendor_type || 'camera',
                ipAddress: d.ip_address || '',
                lastSeen: d.last_seen || '',
            }));
    }, [site, allDevices]);

    const onlineCount = siteDevices.filter((d) => d.status === 'online').length;
    const offlineCount = siteDevices.filter((d) => d.status !== 'online').length;

    // ── Breadcrumb items ────────────────────────────────────────────────
    const breadcrumbItems = useMemo(() => {
        if (!site) return [{ label: 'sites', href: '/sites' }];
        return [
            { label: 'sites', href: '/sites' },
            { label: site.name, href: undefined },
        ];
    }, [site]);

    // ── Loading state ───────────────────────────────────────────────────
    if (siteLoading) {
        return <SkeletonDetailPage />;
    }

    if (!site) {
        return (
            <div className="space-y-6">
                <Breadcrumbs items={breadcrumbItems} className="mb-2" />
                <Card>
                    <CardBody>
                        <div className="text-center py-12 text-slate-500 dark:text-slate-400">
                            <Building2 className="w-12 h-12 mx-auto mb-4 opacity-50" />
                            <p className="text-lg font-medium">{t('site_not_found') || 'Site not found'}</p>
                            <Button
                                variant="outline"
                                className="mt-4"
                                onClick={() => navigate('/sites')}
                            >
                                <ArrowLeft className="w-4 h-4 mr-2" />
                                {t('back_to_sites') || 'Back to Sites'}
                            </Button>
                        </div>
                    </CardBody>
                </Card>
            </div>
        );
    }

    // ── Render ──────────────────────────────────────────────────────────
    return (
        <div className="space-y-6">
            <Breadcrumbs items={breadcrumbItems} className="mb-2" />

            {/* Site Header */}
            <div className="flex flex-col md:flex-row md:items-start justify-between gap-4">
                <div className="flex items-start gap-4">
                    <div className="p-4 bg-slate-100 dark:bg-slate-800/80 dark:border dark:border-slate-700/50 rounded-xl">
                        <Building2 className="w-8 h-8 text-slate-600 dark:text-slate-400" />
                    </div>
                    <div>
                        <div className="flex items-center gap-3 flex-wrap">
                            <h1 className="text-2xl font-bold text-slate-900 dark:text-white">
                                {site.name}
                            </h1>
                            <StatusBadge status={site.status} />
                        </div>
                        <p className="text-sm text-slate-500 dark:text-slate-400 mt-1 flex items-center gap-1">
                            <MapPin className="w-3.5 h-3.5" />
                            {[site.address, site.city].filter(Boolean).join(', ') || t('no_address') || 'No address'}
                        </p>
                        {site.organization && (
                            <p className="text-sm text-slate-500 dark:text-slate-400 mt-0.5">
                                {site.organization}
                            </p>
                        )}
                    </div>
                </div>
                <div className="flex gap-2">
                    <Button
                        variant="outline"
                        onClick={() => navigate('/sites')}
                        icon={<ArrowLeft className="w-4 h-4" />}
                    >
                        {t('back')}
                    </Button>
                </div>
            </div>

            {/* Stats Cards */}
            <div className="grid grid-cols-1 sm:grid-cols-2 lg:grid-cols-4 gap-4">
                <Card>
                    <CardBody>
                        <div className="flex items-center justify-between">
                            <div>
                                <p className="text-xs font-medium text-slate-500 dark:text-slate-400 uppercase tracking-wider">
                                    {t('total_devices') || 'Total Devices'}
                                </p>
                                <p className="text-2xl font-bold text-slate-900 dark:text-white mt-1">
                                    {siteDevices.length}
                                </p>
                            </div>
                            <div className="p-3 bg-blue-50 dark:bg-blue-900/20 rounded-lg">
                                <Camera className="w-6 h-6 text-blue-600 dark:text-blue-400" />
                            </div>
                        </div>
                    </CardBody>
                </Card>
                <Card>
                    <CardBody>
                        <div className="flex items-center justify-between">
                            <div>
                                <p className="text-xs font-medium text-slate-500 dark:text-slate-400 uppercase tracking-wider">
                                    {t('online')}
                                </p>
                                <p className="text-2xl font-bold text-emerald-600 dark:text-emerald-400 mt-1">
                                    {onlineCount}
                                </p>
                            </div>
                            <div className="p-3 bg-emerald-50 dark:bg-emerald-900/20 rounded-lg">
                                <Wifi className="w-6 h-6 text-emerald-600 dark:text-emerald-400" />
                            </div>
                        </div>
                    </CardBody>
                </Card>
                <Card>
                    <CardBody>
                        <div className="flex items-center justify-between">
                            <div>
                                <p className="text-xs font-medium text-slate-500 dark:text-slate-400 uppercase tracking-wider">
                                    {t('offline')}
                                </p>
                                <p className="text-2xl font-bold text-red-600 dark:text-red-400 mt-1">
                                    {offlineCount}
                                </p>
                            </div>
                            <div className="p-3 bg-red-50 dark:bg-red-900/20 rounded-lg">
                                <WifiOff className="w-6 h-6 text-red-600 dark:text-red-400" />
                            </div>
                        </div>
                    </CardBody>
                </Card>
                <Card>
                    <CardBody>
                        <div className="flex items-center justify-between">
                            <div>
                                <p className="text-xs font-medium text-slate-500 dark:text-slate-400 uppercase tracking-wider">
                                    {t('last_sync') || 'Last Sync'}
                                </p>
                                <p className="text-sm font-semibold text-slate-900 dark:text-white mt-1">
                                    {formatDate(site.lastSync, i18n.language)}
                                </p>
                            </div>
                            <div className="p-3 bg-purple-50 dark:bg-purple-900/20 rounded-lg">
                                <Activity className="w-6 h-6 text-purple-600 dark:text-purple-400" />
                            </div>
                        </div>
                    </CardBody>
                </Card>
            </div>

            {/* Device List */}
            <Card>
                <CardHeader>
                    <div className="flex items-center gap-2">
                        <Camera className="w-5 h-5 text-slate-500" />
                        <h2 className="text-lg font-semibold text-slate-900 dark:text-white">
                            {t('connected_devices') || 'Connected Devices'}
                        </h2>
                        <Badge variant="neutral">{siteDevices.length}</Badge>
                    </div>
                </CardHeader>
                <CardBody>
                    {siteDevices.length === 0 ? (
                        <div className="text-center py-8 text-slate-500 dark:text-slate-400">
                            <Camera className="w-10 h-10 mx-auto mb-3 opacity-50" />
                            <p>{t('no_devices_site') || 'No devices at this site'}</p>
                        </div>
                    ) : (
                        <div className="grid gap-3">
                            {siteDevices.map((device) => (
                                <button
                                    key={device.id}
                                    onClick={() => navigate(`/devices/${device.id}`)}
                                    className="flex items-center justify-between p-4 bg-slate-50 dark:bg-slate-800/50 rounded-xl border border-slate-200 dark:border-slate-700 hover:border-blue-300 dark:hover:border-blue-600 transition-colors text-left w-full focus-visible:outline-2 focus-visible:outline-blue-500"
                                >
                                    <div className="flex items-center gap-3">
                                        <div className={`w-2.5 h-2.5 rounded-full flex-shrink-0 ${
                                            device.status === 'online'
                                                ? 'bg-emerald-500'
                                                : 'bg-red-500'
                                        }`} />
                                        <div>
                                            <p className="text-sm font-medium text-slate-900 dark:text-white">
                                                {device.name}
                                            </p>
                                            <p className="text-xs text-slate-500 dark:text-slate-400">
                                                {device.type} — {device.ipAddress || 'N/A'}
                                            </p>
                                        </div>
                                    </div>
                                    <div className="flex items-center gap-2">
                                        <HealthBadge
                                            health={device.status === 'online' ? 'healthy' : 'faulty'}
                                        />
                                    </div>
                                </button>
                            ))}
                        </div>
                    )}
                </CardBody>
            </Card>
        </div>
    );
};

export default SiteDetail;
