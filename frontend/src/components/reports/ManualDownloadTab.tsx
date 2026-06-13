import React, { useState } from 'react';
import { Card, CardHeader, CardBody, CardFooter, Button, Select, Input, Badge } from '../ui';
import { FileText, Download, Calendar, Search, Filter } from 'lucide-react';
import { useToast } from '../ui';
import { generateExcelReport, triggerBlobDownload } from '../../utils/reportGenerator';
import { useDevicesSites, useTickets, useReports } from '../../context/DataContext';
import { useAuth } from '../../hooks/useAuth';
import DatePicker from 'react-datepicker';
import 'react-datepicker/dist/react-datepicker.css';
import { parseISO, format } from 'date-fns';
import { useTranslation } from 'react-i18next';

export function ManualDownloadTab() {
    const { t } = useTranslation();
    const toast = useToast();
    const { devices, sites } = useDevicesSites();
    const { tickets } = useTickets();
    const { user } = useAuth();
    const { addGeneratedReport } = useReports();

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

    const handleGenerate = () => {
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
            const result = generateExcelReport({
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
                data: { devices, sites, tickets },
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
                                    <div><label className="block text-sm font-medium text-slate-700 dark:text-slate-200 mb-1.5">{t('start_date')}</label><DatePicker selected={startDate ? parseISO(startDate) : null} onChange={(date: Date | null) => setStartDate(date ? format(date, 'yyyy-MM-dd') : '')} dateFormat="dd/MM/yyyy" maxDate={new Date()} placeholderText="dd/mm/yyyy" className="w-full px-3.5 py-2.5 text-sm ..." /></div>
                                    <div><label className="block text-sm font-medium text-slate-700 dark:text-slate-200 mb-1.5">{t('end_date')}</label><DatePicker selected={endDate ? parseISO(endDate) : null} onChange={(date: Date | null) => setEndDate(date ? format(date, 'yyyy-MM-dd') : '')} dateFormat="dd/MM/yyyy" maxDate={new Date()} placeholderText="dd/mm/yyyy" className="w-full px-3.5 py-2.5 text-sm ..." /></div>
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
                <div className="flex gap-3"><Button variant="outline" icon={<Download className="w-4 h-4" />} onClick={() => toast.info(t('pdf_export_soon'))}>{t('export_pdf')}</Button><Button variant="primary" icon={isGenerating ? <div className="w-4 h-4 border-2 border-white/30 border-t-white rounded-full animate-spin" /> : <Download className="w-4 h-4" />} onClick={handleGenerate} disabled={isGenerating} className="min-w-[140px]">{isGenerating ? t('generating') : t('download_excel')}</Button></div>
            </CardFooter>
        </Card>
    );
}