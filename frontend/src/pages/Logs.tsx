import React, { useState } from 'react';
import { Card, CardBody, Table, Input, Button, Select, Pagination, Badge } from '../components/ui';
import { api, ParsedLog } from '../services/api';
import { Search } from 'lucide-react';
import { useTranslation } from 'react-i18next';

export function Logs() {
    const { t } = useTranslation();
    const [logs, setLogs] = useState<ParsedLog[]>([]);
    const [loading, setLoading] = useState(false);
    const [filters, setFilters] = useState({ device_id: '', level: '', keyword: '' });
    const [page, setPage] = useState(1);
    const itemsPerPage = 20;

    const handleSearch = async () => {
        setLoading(true);
        try {
            const result = await api.searchLogs(filters);
            setLogs(result);
        } catch (err) {
            console.error(err);
        } finally {
            setLoading(false);
        }
    };

    const paginated = logs.slice((page-1)*itemsPerPage, page*itemsPerPage);
    const columns = [
        { header: t('time'), key: 'timestamp', render: (l: ParsedLog) => new Date(l.timestamp).toLocaleString() },
        { header: t('device_id'), key: 'device_id' },
        { header: t('log_level'), key: 'log_level', render: (l: ParsedLog) => <Badge variant={l.log_level === 'ERROR' ? 'danger' : l.log_level === 'WARN' ? 'warning' : 'info'}>{l.log_level}</Badge> },
        { header: t('message'), key: 'message' },
        { header: t('source'), key: 'source' },
    ];

    return (
        <div className="space-y-6">
            <h1 className="text-2xl font-bold text-slate-900 dark:text-white">{t('logs_title')}</h1>
            <Card>
                <CardBody className="space-y-4">
                    <div className="grid grid-cols-1 md:grid-cols-4 gap-4">
                        <Input placeholder={t('device_id')} value={filters.device_id} onChange={(e) => setFilters({...filters, device_id: e.target.value})} />
                        <Select options={[{value:'', label:t('all_levels')},{value:'ERROR',label:'ERROR'},{value:'WARN',label:'WARN'},{value:'INFO',label:'INFO'}]} value={filters.level} onChange={(e) => setFilters({...filters, level: e.target.value})} />
                        <Input placeholder={t('keyword')} value={filters.keyword} onChange={(e) => setFilters({...filters, keyword: e.target.value})} />
                        <Button icon={<Search className="w-4 h-4" />} onClick={handleSearch} loading={loading}>{t('search')}</Button>
                    </div>
                    <Table data={paginated} columns={columns} keyExtractor={(l: ParsedLog) => l.timestamp + l.device_id} emptyMessage={t('no_logs_found')} />
                    {logs.length > itemsPerPage && <Pagination currentPage={page} totalPages={Math.ceil(logs.length/itemsPerPage)} onPageChange={setPage} totalItems={logs.length} itemsPerPage={itemsPerPage} />}
                </CardBody>
            </Card>
        </div>
    );
}