import React, { useState, useMemo, Suspense } from 'react';
import { Card, CardHeader, CardBody, CardFooter, Button, Select, Input, Badge, Modal, Table } from '../ui';
import { FileText, Download, Calendar, Search, Filter, X } from '../ui/Icons';
import { useToast } from '../ui';
import { generateExcelReport, triggerBlobDownload } from '../../utils/reportGenerator';
import { useDevices, useSites, useTickets } from '../../hooks/useApiQuery';
import { useReportsStore } from '../../store/reportsStore';
import { useAuth } from '../../hooks/useAuth';
import 'react-datepicker/dist/react-datepicker.css';
import { parseISO, format } from 'date-fns';
import { useTranslation } from 'react-i18next';
import { Device } from '../../types';

// Lazy-loaded: react-datepicker (~100KB)
const LazyDatePicker = React.lazy(() => import('react-datepicker'));

const StatCard: React.FC<{ label: string; value: number }> = ({ label, value }) => (
  <div className="p-3 bg-slate-50 dark:bg-slate-800 rounded-lg border border-slate-200 dark:border-slate-700">
    <p className="text-2xl font-bold text-slate-900 dark:text-white">{value}</p>
    <p className="text-xs text-slate-500 dark:text-slate-400">{label}</p>
  </div>
);

export function ManualDownloadTab() {
    const { t } = useTranslation();
    const toast = useToast();
    const { data: rawDevices = [] } = useDevices();
    const { data: rawSites = [] } = useSites();
    const { data: apiTickets = [] } = useTickets();
    const { user } = useAuth();
    const addGeneratedReport = useReportsStore((s) => s.addGeneratedReport);

    const mapAPIDeviceToUI = (d: Record<string, any>): { id: string; name: string; siteId: string; siteName: string; type: string; status: string; health: string; recordingStatus: string; lastSeen: string; ipAddress: string; model: string; firmware: string; owner_id: any } => ({
        id: d.device_id,
        name: d.name || d.device_id,
        siteId: d.site_id || 'site-default',
        siteName: d.location || 'Unknown',
        type: d.vendor_type === 'camera' ? 'camera' : 'nvr',
        status: (d.status || 'offline').toLowerCase(),
        health: d.status === 'online' ? 'healthy' : 'faulty',
        recordingStatus: 'recording',
        lastSeen: d.last_seen || new Date().toISOString(),
        ipAddress: '',
        model: d.vendor_type || '',
        firmware: '',
        owner_id: d.owner_id,
    });

    const devices = useMemo((): Array<Record<string, any>> => {
        const devsArray = Array.isArray(rawDevices) ? rawDevices : (rawDevices && typeof rawDevices === 'object' && 'devices' in rawDevices ? (rawDevices as any).devices : []);
        return devsArray.map(mapAPIDeviceToUI);
    }, [rawDevices]);

    const sites = useMemo((): Array<Record<string, any>> => rawSites.map((s: Record<string, any>) => ({
        id: s.id, name: s.name || 'Unnamed', address: s.address || '', city: s.city || '',
        organization: s.organization || '', latitude: s.latitude || 0, longitude: s.longitude || 0,
        status: s.status || 'active', lastSync: s.last_sync || new Date().toISOString(),
    })), [rawSites]);

    const tickets = useMemo((): Array<Record<string, any>> => apiTickets.map((t: Record<string, any>) => ({
        id: t.id, title: t.title, description: t.description, deviceId: t.device_id || '',
        deviceName: '', siteName: '', priority: (t.priority || 'medium') as any,
        status: (t.status || 'open') as any, assignee: t.assignee || '',
        createdAt: t.created_at, updatedAt: t.updated_at,
        comments: (t.comments || []).map((c: any) => ({ id: c.id, ticketId: c.ticket_id, userId: c.user_id, userName: c.user_name || '', content: c.content, createdAt: c.created_at })),
    })), [apiTickets]);

    const [reportType, setReportType] = useState('consolidated');
    const [duration, setDuration] = useState('last_3_months');
    const [startDate, setStartDate] = useState('');
    const [endDate, setEndDate] = useState('');
    const [filteredSites, setFilteredSites] = useState('all');
    const [deviceType, setDeviceType] = useState('all');
    const [statusFilter, setStatusFilter] = useState('all');
    const [issueTypeFilter, setIssueTypeFilter] = useState('all');
    const [deviceNameQuery, setDeviceNameQuery] = useState('');
    const [selectedDeviceId, setSelectedDeviceId] = useState('');
    const [isDeviceDropdownOpen, setIsDeviceDropdownOpen] = useState(false);
    const [isGenerating, setIsGenerating] = useState(false);
    const [showPreview, setShowPreview] = useState(false);

    const filteredDevices = useMemo(() => devices.filter(d => {
        if (filteredSites !== 'all' && d.siteId !== filteredSites) return false;
        if (deviceType !== 'all' && d.type !== deviceType) return false;
        if (statusFilter !== 'all' && d.status !== statusFilter) return false;
        return true;
    }), [devices, filteredSites, deviceType, statusFilter]);

    const previewColumns = [
        { key: 'name' as keyof Device, header: t('device') || 'Device' },
        { key: 'type' as keyof Device, header: t('type') || 'Type' },
        { key: 'status' as keyof Device, header: t('status') || 'Status' },
        { key: 'siteName' as keyof Device, header: t('site') || 'Site' },
    ];

    const handleExportPDF = async () => {
        try {
            setIsGenerating(true);
            const response = await fetch('/api/v1/reports/generate', {
                method: 'POST',
                headers: { 'Content-Type': 'application/json' },
                body: JSON.stringify({
                    type: reportType,
                    format: 'pdf',
                    filters: {
                        duration,
                        startDate: duration === 'custom' ? startDate : undefined,
                        endDate: duration === 'custom' ? endDate : undefined,
                        site: filteredSites,
                        deviceType,
                        status: statusFilter,
                        issueType: issueTypeFilter,
                        deviceId: selectedDeviceId,
                    },
                }),
            });
            if (!response.ok) throw new Error('Failed to generate PDF');

            const result = await response.json();
            // Redirect to download
            window.open(`/api/v1/reports/${result.report_id}/download`, '_blank');
            toast.success('PDF report generated');
        } catch (err) {
            console.error('PDF export failed:', err);
            toast.error('Failed to generate PDF report');
        } finally {
            setIsGenerating(false);
        }
    };

    const handleGenerate = async () => {
        if (duration === 'custom') {
            if (!startDate || !endDate) {
                toast.error(t('custom_date_required') || 'Please select both start and end dates for custom duration.');
                return;
            }
            if (new Date(startDate) > new Date(endDate)) {
                toast.error(t('date_start_after_end') || 'Start date cannot be after end date.');
                return;
            }
            if (new Date(endDate) > new Date()) {
                toast.error(t('date_future') || 'End date cannot be in the future.');
                return;
            }
            const msInYear = 365 * 24 * 60 * 60 * 1000;
            if (new Date(endDate).getTime() - new Date(startDate).getTime() > msInYear) {
                toast.error(t('date_range_too_long') || 'Custom date range cannot exceed 1 year of data retention.');
                return;
            }
        }

        setIsGenerating(true);
        toast.info(t('generating_report') || 'Generating report...');

        try {
            const result = await generateExcelReport({
                type: reportType,
                duration,
                startDate,
                endDate,
                filters: {
                    site: filteredSites,
                    deviceType,
                    status: statusFilter,
                    issueType: issueTypeFilter,
                    deviceId: selectedDeviceId,
                },
                data: { devices: devices as any, sites: sites as any, tickets: tickets as any },
                userSites: user?.role === 'admin' ? [] : user?.sites || []
            });

            const generatedAt = new Date().toISOString();
            let dateRangeStr = '';
            if (duration === 'custom') dateRangeStr = `${startDate} to ${endDate}`;
            else if (duration === 'all_data') dateRangeStr = t('all_available_data') || 'All Available Data';
            else dateRangeStr = duration.split('_').map(w => w.charAt(0).toUpperCase() + w.slice(1)).join(' ');

            const typeLabels: Record<string, string> = {
                'consolidated': t('consolidated_health'),
                'dvr_nvr_health': t('dvr_nvr_health'),
                'camera_health': t('camera_health_report'),
                'hdd_health': t('hdd_health_report'),
                'recording_availability': t('recording_availability_report'),
                'ticket_log': t('ticket_log_report')
            };

            const approximateSize = result?.excelBuffer?.length ? `${(result.excelBuffer.length / 1024).toFixed(1)} KB` : '1.2 MB';

            addGeneratedReport({
                id: `rep-${Date.now()}`,
                name: `${t('manual_export_prefix')}: ${typeLabels[reportType] || reportType}`,
                type: typeLabels[reportType] || reportType,
                format: 'xlsx',
                dateRange: dateRangeStr,
                generatedAt,
                generatedBy: user?.name || 'Manual Request',
                status: 'ready',
                size: approximateSize,
                excelBuffer: result?.excelBuffer,
                fileName: result?.fileName
            });

            if (result?.excelBuffer && result?.fileName) {
                triggerBlobDownload(result.excelBuffer, result.fileName);
            }

            toast.success(t('report_generation_complete') || 'Report generation complete.');
        } catch (error: any) {
            console.error('Report generation failed:', error);
            toast.error(error.message || t('report_generation_failed') || 'Failed to generate report.');
        } finally {
            setIsGenerating(false);
        }
    };

    return (
        <>
            <Card>
                <CardHeader className="border-b border-slate-200 dark:border-slate-700/50 pb-4">
                    <div className="flex items-center gap-3">
                        <div className="p-2 bg-blue-50 dark:bg-blue-900/30 text-blue-600 dark:text-blue-400 rounded-lg">
                            <Download className="w-5 h-5" />
                        </div>
                        <div>
                            <h3 className="text-lg font-medium text-slate-900 dark:text-white">{t('manual_download')}</h3>
                            <p className="text-sm text-slate-500 dark:text-slate-400 font-normal">{t('manual_download_desc')}</p>
                        </div>
                    </div>
                </CardHeader>
                <CardBody className="p-6 space-y-8 pb-8">
                    {/* Section 1: Report Options */}
                    <div>
                        <h4 className="text-sm font-semibold text-slate-700 dark:text-slate-300 uppercase tracking-wider mb-4 flex items-center gap-2">
                            <FileText className="w-4 h-4" />
                            {t('select_report_type')}
                        </h4>
                        <div className="grid grid-cols-1 md:grid-cols-2 lg:grid-cols-3 gap-4">
                            {[
                                { id: 'consolidated', label: t('consolidated_health'), desc: t('consolidated_desc') },
                                { id: 'dvr_nvr_health', label: t('dvr_nvr_health'), desc: t('dvr_nvr_desc') },
                                { id: 'camera_health', label: t('camera_health_report'), desc: t('camera_health_desc') },
                                { id: 'hdd_health', label: t('hdd_health_report'), desc: t('hdd_health_desc') },
                                { id: 'recording_availability', label: t('recording_availability_report'), desc: t('recording_availability_desc') },
                                { id: 'ticket_log', label: t('ticket_log_report'), desc: t('ticket_log_desc') }
                            ].map(type => (
                                <div
                                    key={type.id}
                                    onClick={() => setReportType(type.id)}
                                    className={`p-4 rounded-xl border-2 cursor-pointer transition-all ${reportType === type.id
                                        ? 'border-blue-500 bg-blue-50/50 dark:bg-blue-900/20 dark:border-blue-500 shadow-sm'
                                        : 'border-slate-200 dark:border-slate-700 hover:border-blue-300 dark:hover:border-slate-500 bg-white dark:bg-slate-800'
                                        }`}
                                >
                                    <div className="flex justify-between items-start mb-1">
                                        <span className={`font-medium ${reportType === type.id ? 'text-blue-700 dark:text-blue-300' : 'text-slate-900 dark:text-white'}`}>
                                            {type.label}
                                        </span>
                                        {reportType === type.id && (
                                            <div className="w-5 h-5 rounded-full bg-blue-500 flex items-center justify-center"><div className="w-2 h-2 rounded-full bg-white" /></div>
                                        )}
                                    </div>
                                    <p className="text-xs text-slate-500 dark:text-slate-400">{type.desc}</p>
                                </div>
                            ))}
                        </div>
                    </div>

                    <div className="grid grid-cols-1 lg:grid-cols-2 gap-8 pt-4 border-t border-slate-100 dark:border-slate-800">
                        {/* Section 2: Duration Options */}
                        <div>
                            <h4 className="text-sm font-semibold text-slate-700 dark:text-slate-300 uppercase tracking-wider mb-4 flex items-center gap-2">
                                <Calendar className="w-4 h-4" />
                                {t('select_duration')}
                            </h4>
                            <div className="space-y-4 max-w-sm">
                                <Select
                                    label={t('time_range_preset')}
                                    options={[
                                        { value: 'last_7_days', label: t('last_7_days') },
                                        { value: 'last_30_days', label: t('last_30_days') },
                                        { value: 'last_3_months', label: t('last_3_months') },
                                        { value: 'last_6_months', label: t('last_6_months') },
                                        { value: 'all_data', label: t('all_available_data') },
                                        { value: 'custom', label: t('custom_date_range') },
                                    ]}
                                    value={duration}
                                    onChange={(e) => setDuration(e.target.value)}
                                />
                                {duration === 'custom' && (
                                    <div className="grid grid-cols-2 gap-3 p-4 bg-slate-50 dark:bg-slate-800/50 rounded-lg border border-slate-200 dark:border-slate-700">
                                        <Suspense fallback={<div className="h-[38px] bg-slate-100 dark:bg-slate-700 rounded animate-pulse" />}>
                                            <div><label className="block text-sm font-medium text-slate-700 dark:text-slate-200 mb-1.5">{t('start_date')}</label><LazyDatePicker selected={startDate ? parseISO(startDate) : null} onChange={(date: Date | null) => setStartDate(date ? format(date, 'yyyy-MM-dd') : '')} dateFormat="dd/MM/yyyy" maxDate={new Date()} placeholderText="dd/mm/yyyy" className="w-full px-3.5 py-2.5 text-sm ..." /></div>
                                        </Suspense>
                                        <Suspense fallback={<div className="h-[38px] bg-slate-100 dark:bg-slate-700 rounded animate-pulse" />}>
                                            <div><label className="block text-sm font-medium text-slate-700 dark:text-slate-200 mb-1.5">{t('end_date')}</label><LazyDatePicker selected={endDate ? parseISO(endDate) : null} onChange={(date: Date | null) => setEndDate(date ? format(date, 'yyyy-MM-dd') : '')} dateFormat="dd/MM/yyyy" maxDate={new Date()} placeholderText="dd/mm/yyyy" className="w-full px-3.5 py-2.5 text-sm ..." /></div>
                                        </Suspense>
                                    </div>
                                )}
                            </div>
                        </div>

                        {/* Section 3: Filters */}
                        <div>
                            <h4 className="text-sm font-semibold text-slate-700 dark:text-slate-300 uppercase tracking-wider mb-4 flex items-center gap-2">
                                <Filter className="w-4 h-4" />
                                {t('apply_filters')}
                            </h4>
                            <div className="grid grid-cols-1 sm:grid-cols-2 gap-4">
                                <Select label={t('region_site')} options={[{ value: 'all', label: t('all_accessible_sites') }, ...sites.filter(s => user?.role === 'admin' || user?.sites?.includes(s.id)).map(s => ({ value: s.id, label: s.name }))]} value={filteredSites} onChange={(e) => setFilteredSites(e.target.value)} />
                                <Select label={t('device_type_filter')} options={[{ value: 'all', label: t('all_device_types') }, { value: 'camera', label: t('camera') }, { value: 'nvr', label: 'NVR' }, { value: 'dvr', label: 'DVR' }]} value={deviceType} onChange={(e) => setDeviceType(e.target.value)} disabled={reportType === 'dvr_nvr_health' || reportType === 'camera_health'} />
                                <Select label={t('current_status')} options={[{ value: 'all', label: t('all_statuses_filter') }, { value: 'online', label: t('online') }, { value: 'warning', label: t('warning') }, { value: 'offline', label: t('offline') }]} value={statusFilter} onChange={(e) => setStatusFilter(e.target.value)} />
                                <Select label={t('issue_type')} options={[{ value: 'all', label: t('all_issues') }, { value: 'offline', label: t('issue_offline') }, { value: 'storage', label: t('issue_storage') }, { value: 'recording', label: t('issue_recording') }]} value={issueTypeFilter} onChange={(e) => setIssueTypeFilter(e.target.value)} />
                                <div className="relative">
                                    <Input label={t('device_name_search')} placeholder={t('type_to_search')} value={deviceNameQuery} onChange={(e) => { setDeviceNameQuery(e.target.value); setSelectedDeviceId(''); setIsDeviceDropdownOpen(true); }} onFocus={() => setIsDeviceDropdownOpen(true)} onBlur={() => setTimeout(() => setIsDeviceDropdownOpen(false), 200)} />
                                    {isDeviceDropdownOpen && deviceNameQuery && (
                                        <div className="absolute z-10 w-full mt-1 bg-white dark:bg-slate-800 border border-slate-200 dark:border-slate-700 rounded-lg shadow-lg max-h-48 overflow-y-auto">
                                            {devices.filter(d => (user?.role === 'admin' || user?.sites?.includes(d.siteId)) && d.name.toLowerCase().includes(deviceNameQuery.toLowerCase())).map(device => (
                                                <div key={device.id} className="px-4 py-2 hover:bg-slate-50 dark:hover:bg-slate-700/50 cursor-pointer flex flex-col border-b border-slate-100 dark:border-slate-700/50 last:border-0" onMouseDown={(e) => { e.preventDefault(); setDeviceNameQuery(device.name); setSelectedDeviceId(device.id); setIsDeviceDropdownOpen(false); }}><span className="text-sm font-medium text-slate-900 dark:text-white">{device.name}</span><span className="text-xs text-slate-500">{device.siteName} • {device.type.toUpperCase()}</span></div>
                                            ))}
                                            {devices.filter(d => (user?.role === 'admin' || user?.sites?.includes(d.siteId)) && d.name.toLowerCase().includes(deviceNameQuery.toLowerCase())).length === 0 && <div className="px-4 py-3 text-sm text-slate-500 text-center">{t('no_matching_devices')}</div>}
                                        </div>
                                    )}
                                </div>
                            </div>
                        </div>
                    </div>
                </CardBody>
                <CardFooter className="bg-slate-50 dark:bg-slate-800/50 border-t border-slate-200 dark:border-slate-700/50 flex justify-between items-center py-4">
                    <span className="text-sm text-slate-500 dark:text-slate-400 flex items-center gap-2"><div className="w-2 h-2 rounded-full bg-blue-500" />{t('ready_to_generate')} {t(reportType.replace(/_/g, ' '))}</span>
                    <div className="flex gap-3">
                        <Button variant="outline" icon={<Search className="w-4 h-4" />} onClick={() => setShowPreview(true)}>
                            {t('preview') || 'Preview'}
                        </Button>
                        <Button variant="outline" icon={<FileText className="w-4 h-4" />} onClick={handleExportPDF}>{t('export_pdf')}</Button>
                        <Button variant="primary" icon={isGenerating ? <div className="w-4 h-4 border-2 border-white/30 border-t-white rounded-full animate-spin" /> : <Download className="w-4 h-4" />} onClick={handleGenerate} disabled={isGenerating} className="min-w-[140px]">{isGenerating ? t('generating') : t('download_excel')}</Button>
                    </div>
                </CardFooter>
            </Card>

            {/* Preview Modal */}
            <Modal isOpen={showPreview} onClose={() => setShowPreview(false)} title={t('report_preview') || 'Report Preview'} size="xl">
                <div className="space-y-4 max-h-[70vh] overflow-y-auto">
                    <div className="grid grid-cols-2 md:grid-cols-4 gap-3">
                        <StatCard label={t('total_devices')} value={filteredDevices.length} />
                        <StatCard label={t('online')} value={filteredDevices.filter(d => d.status === 'online').length} />
                        <StatCard label={t('offline')} value={filteredDevices.filter(d => d.status === 'offline').length} />
                        <StatCard label={t('sites')} value={sites.length} />
                    </div>
                    <Table data={filteredDevices.slice(0, 50)} columns={previewColumns} keyExtractor={d => d.id} />
                    <p className="text-xs text-slate-400">{filteredDevices.length > 50 ? `Showing 50 of ${filteredDevices.length} devices` : ''}</p>
                    <div className="flex gap-2 justify-end pt-2">
                        <Button variant="secondary" onClick={() => setShowPreview(false)}>{t('close') || 'Close'}</Button>
                        <Button icon={<Download size={16} />} onClick={() => { setShowPreview(false); handleGenerate(); }}>{t('download_excel')}</Button>
                        <Button icon={<FileText size={16} />} onClick={() => { setShowPreview(false); handleExportPDF(); }}>{t('export_pdf') || 'Export PDF'}</Button>
                    </div>
                </div>
            </Modal>
        </>
    );
}
