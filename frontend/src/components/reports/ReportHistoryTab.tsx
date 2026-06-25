import React, { useState, useMemo } from 'react';
import { Card, CardBody, Button, Badge, Input, Select } from '../ui';
import { Download, FileText, Calendar, Clock, FileSpreadsheet, Loader2 } from 'lucide-react';
import { useToast, SearchInput } from '../ui';
import { useReportsStore } from '../../context/ReportsContext';
import * as XLSX from 'xlsx';
import { useTranslation } from 'react-i18next';

export function ReportHistoryTab() {
    const { t } = useTranslation();
    const toast = useToast();
    const generatedReports = useReportsStore((s) => s.generatedReports);
    const [downloadingId, setDownloadingId] = useState<string | null>(null);

    // Filters
    const [searchQuery, setSearchQuery] = useState('');
    const [typeFilter, setTypeFilter] = useState('all');
    const [formatFilter, setFormatFilter] = useState('all');
    const [dateFilter, setDateFilter] = useState('all');

    const reportHistory = useMemo(() => {
        let filtered = [...generatedReports];

        // Search by name
        if (searchQuery.trim()) {
            const q = searchQuery.toLowerCase();
            filtered = filtered.filter(r => r.name.toLowerCase().includes(q) || r.type.toLowerCase().includes(q));
        }

        // Filter by type
        if (typeFilter !== 'all') {
            filtered = filtered.filter(r => r.type === typeFilter);
        }

        // Filter by format
        if (formatFilter !== 'all') {
            filtered = filtered.filter(r => r.format === formatFilter);
        }

        // Filter by date
        if (dateFilter !== 'all') {
            const now = new Date();
            const filterDate = new Date();
            switch (dateFilter) {
                case 'today':
                    filterDate.setHours(0, 0, 0, 0);
                    filtered = filtered.filter(r => new Date(r.generatedAt) >= filterDate);
                    break;
                case 'week':
                    filterDate.setDate(now.getDate() - 7);
                    filtered = filtered.filter(r => new Date(r.generatedAt) >= filterDate);
                    break;
                case 'month':
                    filterDate.setMonth(now.getMonth() - 1);
                    filtered = filtered.filter(r => new Date(r.generatedAt) >= filterDate);
                    break;
            }
        }

        return filtered;
    }, [generatedReports, searchQuery, typeFilter, formatFilter, dateFilter]);

    // Extract unique types for the type filter dropdown
    const typeOptions = useMemo(() => {
        const types = new Set(generatedReports.map(r => r.type));
        return [{ value: 'all', label: t('all_types') || 'All Types' }, ...Array.from(types).map(type => ({ value: type, label: type }))];
    }, [generatedReports, t]);

    return (
        <div className="space-y-6">
            <div className="flex justify-between items-center">
                <div>
                    <h3 className="text-lg font-medium text-slate-900 dark:text-white">{t('report_history')}</h3>
                    <p className="text-sm text-slate-500 dark:text-slate-400">{t('report_history_desc')}</p>
                </div>
            </div>

            {/* Filters */}
            <div className="grid grid-cols-1 sm:grid-cols-2 lg:grid-cols-4 gap-3">
                <SearchInput
                    placeholder={t('search') + '...'}
                    value={searchQuery}
                    onSearch={(value) => setSearchQuery(value)}
                    onChange={(e) => setSearchQuery(e.target.value)}
                />
                <Select
                    options={typeOptions}
                    value={typeFilter}
                    onChange={(e) => setTypeFilter(e.target.value)}
                />
                <Select
                    options={[
                        { value: 'all', label: t('all_formats') || 'All Formats' },
                        { value: 'xlsx', label: 'XLSX' },
                        { value: 'pdf', label: 'PDF' },
                    ]}
                    value={formatFilter}
                    onChange={(e) => setFormatFilter(e.target.value)}
                />
                <Select
                    options={[
                        { value: 'all', label: t('all_time') || 'All Time' },
                        { value: 'today', label: t('today') },
                        { value: 'week', label: t('last_7_days') },
                        { value: 'month', label: t('last_30_days') },
                    ]}
                    value={dateFilter}
                    onChange={(e) => setDateFilter(e.target.value)}
                />
            </div>

            <div className="grid grid-cols-1 gap-4">
                {reportHistory.length > 0 ? reportHistory.map((report) => (
                    <Card key={report.id} variant="bordered" className="hover:border-blue-500/50 transition-colors">
                        <CardBody className="p-4 md:p-5">
                            <div className="flex flex-col md:flex-row gap-4 justify-between items-start md:items-center">
                                <div className="flex items-start gap-4">
                                    <div className={`p-3 rounded-xl shrink-0 ${report.format === 'xlsx' ? 'bg-green-50 dark:bg-green-900/20 text-green-600 dark:text-green-400' : 'bg-red-50 dark:bg-red-900/20 text-red-600 dark:text-red-400'}`}>
                                        {report.format === 'xlsx' ? <FileSpreadsheet className="w-6 h-6" /> : <FileText className="w-6 h-6" />}
                                    </div>
                                    <div className="space-y-1">
                                        <div className="flex flex-wrap items-center gap-2">
                                            <h4 className="font-semibold text-slate-900 dark:text-white text-base leading-tight">{report.name}</h4>
                                            {report.status === 'expired' && <Badge variant="neutral" size="sm">{t('expired')}</Badge>}
                                        </div>
                                        <div className="flex flex-wrap gap-x-4 gap-y-1 text-sm text-slate-500 dark:text-slate-400">
                                            <span className="flex items-center gap-1.5"><Calendar className="w-3.5 h-3.5" />{report.dateRange}</span>
                                            <span className="flex items-center gap-1.5"><Clock className="w-3.5 h-3.5" />{t('generated_at')}: {new Date(report.generatedAt).toLocaleString()}</span>
                                            <span className="hidden md:inline">•</span><span>{t('by')}: {report.generatedBy}</span>
                                        </div>
                                        <div className="text-xs text-slate-400 dark:text-slate-500 font-medium">{t('type')}: {report.type} • {t('size')}: {report.size}</div>
                                    </div>
                                </div>
                                <div className="w-full md:w-auto flex justify-end mt-2 md:mt-0 border-t md:border-t-0 pt-4 md:pt-0 border-slate-200 dark:border-slate-700/50">
                                    <Button
                                        variant={report.status === 'ready' ? 'outline' : 'ghost'}
                                        size="sm"
                                        icon={downloadingId === report.id ? <Loader2 className="w-4 h-4 animate-spin" /> : <Download className="w-4 h-4" />}
                                        className={`${report.status === 'expired' ? 'opacity-50 cursor-not-allowed' : ''}`}
                                        disabled={report.status === 'expired' || downloadingId === report.id}
                                        onClick={() => {
                                            if (report.status === 'expired') { toast.error(t('report_expired')); return; }
                                            setDownloadingId(report.id);
                                            toast.info(t('downloading') + report.name);
                                            setTimeout(() => {
                                                try {
                                                    if (report.fileUrl) {
                                                        const dlFileName = report.fileName || `${report.name.replace(/[^a-z0-9]/gi, '_').toLowerCase()}_history.xlsx`;
                                                        const link = document.createElement('a');
                                                        link.href = report.fileUrl;
                                                        link.download = dlFileName;
                                                        document.body.appendChild(link);
                                                        link.click();
                                                        document.body.removeChild(link);
                                                    } else {
                                                        const worksheet = XLSX.utils.json_to_sheet([{ 'Report ID': report.id, 'Report Name': report.name, 'Type': report.type, 'Date Range': report.dateRange, 'Generated At': report.generatedAt, 'Generated By': report.generatedBy, 'Original Size': report.size }]);
                                                        const workbook = XLSX.utils.book_new();
                                                        XLSX.utils.book_append_sheet(workbook, worksheet, 'Report Meta');
                                                        const dlFileName = report.fileName || `${report.name.replace(/[^a-z0-9]/gi, '_').toLowerCase()}_history.xlsx`;
                                                        XLSX.writeFile(workbook, dlFileName);
                                                    }
                                                    toast.success(t('download_complete'));
                                                } catch (error) { console.error(error); toast.error(t('download_failed')); }
                                                finally { setDownloadingId(null); }
                                            }, 1500);
                                        }}
                                    >{downloadingId === report.id ? t('downloading') : t('download')} {report.format.toUpperCase()}</Button>
                                </div>
                            </div>
                        </CardBody>
                    </Card>
                )) : (
                    <div className="text-center py-12 bg-slate-50 dark:bg-slate-800/50 rounded-xl border border-dashed border-slate-200 dark:border-slate-700">
                        <FileText className="w-12 h-12 text-slate-400 mx-auto mb-4" />
                        <h4 className="text-base font-medium text-slate-900 dark:text-white mb-1">{t('no_report_history')}</h4>
                        <p className="text-sm text-slate-500 dark:text-slate-400">{t('reports_will_appear')}</p>
                    </div>
                )}
            </div>
        </div>
    );
}
