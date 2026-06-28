import { generateUUID } from '../utils/uuid';
import { getArrayData } from '../utils/helpers';
import React, { useState, useMemo, useCallback } from 'react';
import { useParams, useNavigate } from 'react-router-dom';
import { useTickets, useDevices, useSites, useCreateTicket } from '../hooks/useApiQuery';
import { SkeletonDetailPage } from '../components/layout';
import type { Ticket as APITicket, Device as APIDevice, TicketComment } from '../services/api';
import { Breadcrumbs } from '../components/ui/Breadcrumbs';
import {
    ArrowLeft,
    Camera,
    Wifi,
    WifiOff,
    MapPin,
    Clock,
    Settings,
    RefreshCw,
    AlertTriangle,
    CheckCircle,
    XCircle,
    Video,
    HardDrive,
    Cpu,
    Activity,
    Thermometer,
    Info,
    Wrench,
    RotateCcw,
    Download,
    ChevronLeft,
    ChevronRight,
    Loader2,
    Tag,
    QrCode,
    Shield,
} from 'lucide-react';
import {
    Card,
    CardHeader,
    CardBody,
    CardFooter,
    Button,
    Badge,
    StatusBadge,
    HealthBadge,
    Modal,
    Input,
    Select,
    Tabs,
    InfoTooltip,
} from '../components/ui';
import type { RecordingDay, HealthTimelineEvent as TimelineEvent, DeviceStats, DeviceCamera } from '../types';
import { DeviceAuditLog } from '../components/devices/DeviceAuditLog';
import { useTranslation } from 'react-i18next';

// P1-UX.7: RCA Widget — summary card with full graph in modal
import { RCAWidget } from '../components/rca/RCAWidget';

// ─── Helpers with locale support ─────────────────────────────────────
function formatDate(dateStr: string, locale: string): string {
    return new Date(dateStr).toLocaleDateString(locale, {
        month: 'short',
        day: 'numeric',
        year: 'numeric',
    });
}

function formatTime(dateStr: string, locale: string): string {
    return new Date(dateStr).toLocaleTimeString(locale, {
        hour: '2-digit',
        minute: '2-digit',
    });
}

function formatDateTime(dateStr: string, locale: string): string {
    return `${formatDate(dateStr, locale)} at ${formatTime(dateStr, locale)}`;
}

// ─── Timeline Icon (unchanged) ───────────────────────────────────────
function TimelineIcon({ event }: { event: TimelineEvent }) {
    const baseClass = 'w-4 h-4';
    switch (event.type) {
        case 'status_change':
            return event.severity === 'error'
                ? <XCircle className={`${baseClass} text-red-500`} />
                : <CheckCircle className={`${baseClass} text-emerald-500`} />;
        case 'alert':
            return <AlertTriangle className={`${baseClass} text-amber-500`} />;
        case 'maintenance':
            return <Wrench className={`${baseClass} text-blue-500`} />;
        case 'firmware':
            return <Download className={`${baseClass} text-purple-500`} />;
        case 'restart':
            return <RotateCcw className={`${baseClass} text-orange-500`} />;
        default:
            return <Info className={`${baseClass} text-slate-500`} />;
    }
}

const severityBorder: Record<string, string> = {
    success: 'border-emerald-500',
    info: 'border-blue-500',
    warning: 'border-amber-500',
    error: 'border-red-500',
};

// ─── Memoized Calendar Sub-components (now receive t) ────────────────
interface RecordingCellProps {
    camId: string;
    camName: string;
    date: string;
    status: string;
    entry: RecordingDay | undefined;
    onSelect: (entry: RecordingDay) => void;
    t: (key: string) => string;
}

const RecordingCell = React.memo(function RecordingCell({
    camId, camName, date, status, entry, onSelect, t,
}: RecordingCellProps) {
    const cellColor =
        status === 'available'
            ? 'bg-emerald-500 hover:bg-emerald-600'
            : status === 'missing'
                ? 'bg-red-500 hover:bg-red-600 cursor-pointer'
                : 'bg-slate-200 dark:bg-slate-700 hover:bg-slate-300 dark:hover:bg-slate-600';
    const statusText = status === 'available' ? t('available') : status === 'missing' ? t('missing') : t('no_data');
    return (
        <button
            className={`h-5 rounded-sm ${cellColor} transition-colors`}
            onClick={status === 'missing' && entry ? () => onSelect(entry) : undefined}
            title={`${camName} — ${formatDate(date, 'en-US')} — ${statusText}`}
        />
    );
});

interface CameraRowProps {
    camId: string;
    camName: string;
    dateRange: string[];
    recordingDataMap: Map<string, RecordingDay>;
    onSelect: (entry: RecordingDay) => void;
    t: (key: string) => string;
}

const CameraRow = React.memo(function CameraRow({
    camId, camName, dateRange, recordingDataMap, onSelect, t,
}: CameraRowProps) {
    return (
        <div className="flex items-center mt-1">
            <div className="w-32 flex-shrink-0 pr-2">
                <p className="text-xs font-medium text-slate-700 dark:text-slate-300 truncate" title={camName}>
                    {camName}
                </p>
            </div>
            <div className="flex-1 grid gap-px" style={{ gridTemplateColumns: `repeat(${dateRange.length}, 1fr)` }}>
                {dateRange.map((date) => {
                    const entry = recordingDataMap.get(`${camId}-${date}`);
                    const status = entry?.status ?? 'no_data';
                    return (
                        <RecordingCell
                            key={`${camId}-${date}`}
                            camId={camId}
                            camName={camName}
                            date={date}
                            status={status}
                            entry={entry}
                            onSelect={onSelect}
                            t={t}
                        />
                    );
                })}
            </div>
        </div>
    );
});

// ─── Main Component ──────────────────────────────────────────────────
// ═══ API→UI mapping helpers ═══
function mapAPITicketToUI(t: APITicket): import('../types').Ticket {
    return {
        id: t.id,
        title: t.title,
        description: t.description,
        deviceId: t.device_id || '',
        deviceName: '',
        siteName: '',
        priority: (t.priority as import('../types').Ticket['priority']) || 'medium',
        status: (t.status as import('../types').Ticket['status']) || 'open',
        assignee: t.assignee || '',
        createdAt: t.created_at,
        updatedAt: t.updated_at,
        comments: (t.comments || []).map((c: TicketComment) => ({
            id: c.id,
            ticketId: c.ticket_id,
            userId: c.user_id || '',
            userName: c.user_name || '',
            content: c.content,
            createdAt: c.created_at,
        })),
    };
}

function mapAPIDeviceToUI(d: APIDevice): import('../types').Device {
    return {
        id: d.device_id,
        name: d.name || d.device_id,
        siteId: (d as any).site_id || 'site-default',
        siteName: (d as any).location || 'Unknown',
        type: ((d as any).vendor_type === 'camera' ? 'camera' : 'nvr') as 'camera' | 'nvr' | 'dvr' | 'switch',
        status: (d.status || 'offline').toLowerCase() as 'online' | 'offline' | 'warning',
        health: d.status === 'online' ? 'healthy' : 'faulty',
        recordingStatus: 'recording' as const,
        lastSeen: d.last_seen || new Date().toISOString(),
        ipAddress: '',
        model: (d as any).vendor_type || '',
        firmware: '',
        owner_id: d.owner_id,
    };
}

export function DeviceDetail() {
    const { t, i18n } = useTranslation();
    const { deviceId } = useParams();
    const navigate = useNavigate();
    const { data: apiTickets, isLoading: ticketsLoading } = useTickets();
    const { data: apiDevices, isLoading: devicesLoading } = useDevices();
    const apiTicketsData = getArrayData<APITicket>(apiTickets);
    const apiDevicesData = getArrayData<APIDevice>(apiDevices);
    const createTicketMut = useCreateTicket();

    const tickets = useMemo(() => apiTicketsData.map(mapAPITicketToUI), [apiTicketsData]);
    const devices = useMemo(() => apiDevicesData.map(mapAPIDeviceToUI), [apiDevicesData]);

    const [selectedRecording, setSelectedRecording] = useState<RecordingDay | null>(null);
    const [ticketCreated, setTicketCreated] = useState(false);
    const [showCreateTicketModal, setShowCreateTicketModal] = useState(false);
    const [newTicket, setNewTicket] = useState({
        title: '',
        description: '',
        priority: 'medium',
    });
    const [selectedMonth, setSelectedMonth] = useState(() => new Date(2026, 1, 1));

    const handleSelectRecording = useCallback((entry: RecordingDay) => {
        setSelectedRecording(entry);
        setTicketCreated(false);
    }, []);

    const device = devices.find((d) => d.id === deviceId);
    const deviceTickets = tickets.filter((t) => t.deviceId === deviceId);

    // ARCH-03: mockData заменён на пустые структуры.
    // При появлении API для cameras/stats/timeline/recording — заменить на React Query хуки.
    const cameras: DeviceCamera[] = ([] as DeviceCamera[]);
    const stats = (undefined as DeviceStats | undefined);
    const timeline = ([] as TimelineEvent[]);

    const recordingData = useMemo(() => {
        if (!deviceId) return [];
        const allData: RecordingDay[] = [];
        const oneYearAgo = new Date();
        oneYearAgo.setFullYear(oneYearAgo.getFullYear() - 1);
        oneYearAgo.setHours(0, 0, 0, 0);
        return allData.map(entry => {
            const entryDate = new Date(entry.date);
            if (entryDate < oneYearAgo) {
                return { ...entry, status: 'no_data' as const };
            }
            return entry;
        });
    }, [deviceId]);

    const recordingDataMap = useMemo(() => {
        const map = new Map<string, RecordingDay>();
        recordingData.forEach((r) => {
            map.set(`${r.cameraId}-${r.date}`, r);
        });
        return map;
    }, [recordingData]);

    const dateRange = useMemo(() => {
        const dates: string[] = [];
        const year = selectedMonth.getFullYear();
        const month = selectedMonth.getMonth();
        const daysInMonth = new Date(year, month + 1, 0).getDate();
        for (let d = 1; d <= daysInMonth; d++) {
            const date = new Date(year, month, d);
            dates.push(date.toISOString().split('T')[0]);
        }
        return dates;
    }, [selectedMonth]);

    const calendarCameras = useMemo(() => {
        const map = new Map<string, string>();
        recordingData.forEach((r) => map.set(r.cameraId, r.cameraName));
        return Array.from(map.entries());
    }, [recordingData]);

    const currentDate = new Date();
    const oneYearAgo = new Date();
    oneYearAgo.setFullYear(oneYearAgo.getFullYear() - 1);

    const selectedYM = selectedMonth.getFullYear() * 12 + selectedMonth.getMonth();
    const earliestYM = oneYearAgo.getFullYear() * 12 + oneYearAgo.getMonth();
    const latestYM = currentDate.getFullYear() * 12 + currentDate.getMonth();

    const canGoPrev = selectedYM > earliestYM;
    const canGoNext = selectedYM < latestYM;

    const goToPrevMonth = () => {
        if (!canGoPrev) return;
        setSelectedMonth((prev) => new Date(prev.getFullYear(), prev.getMonth() - 1, 1));
    };
    const goToNextMonth = () => {
        if (!canGoNext) return;
        setSelectedMonth((prev) => new Date(prev.getFullYear(), prev.getMonth() + 1, 1));
    };

    const monthLabel = selectedMonth.toLocaleDateString(i18n.language, { month: 'long', year: 'numeric' });

    // ── Loading state ──────────────────────────────────────────────
    const isLoading = devicesLoading || ticketsLoading;

    if (isLoading) {
      return <SkeletonDetailPage />;
    }

    if (!device) {
        return (
            <div className="text-center py-12">
                <h2 className="text-xl font-semibold text-slate-900 dark:text-white">{t('device_not_found')}</h2>
                <Button variant="outline" onClick={() => navigate('/devices')} className="mt-4">
                    {t('back_to_devices')}
                </Button>
            </div>
        );
    }

    const uptimeColor = (stats?.uptimePercent ?? 0) >= 99 ? 'text-emerald-600 dark:text-emerald-400'
        : (stats?.uptimePercent ?? 0) >= 90 ? 'text-amber-600 dark:text-amber-400'
            : 'text-red-600 dark:text-red-400';

    const hddColor = (stats?.hddFreePercent ?? 0) >= 50 ? 'bg-emerald-500'
        : (stats?.hddFreePercent ?? 0) >= 20 ? 'bg-amber-500'
            : 'bg-red-500';

    const handleCreateTicket = () => {
        if (!selectedRecording || !device) return;
        createTicketMut.mutateAsync({
            title: `${t('missing_recording')}: ${selectedRecording.cameraName}`,
            description: `${t('no_recording_data')} ${selectedRecording.cameraName} ${t('on')} ${formatDate(selectedRecording.date, i18n.language)}.`,
            priority: 'medium',
            status: 'open',
            device_id: device.id,
        });
        setTicketCreated(true);
        setTimeout(() => {
            setTicketCreated(false);
            setSelectedRecording(null);
        }, 2000);
    };

    const handleCreateGeneralTicket = (e: React.FormEvent) => {
        e.preventDefault();
        if (!device) return;
        createTicketMut.mutateAsync({
            title: newTicket.title,
            description: newTicket.description,
            priority: newTicket.priority,
            status: 'open',
            device_id: device.id,
        });
        setShowCreateTicketModal(false);
        setNewTicket({ title: '', description: '', priority: 'medium' });
    };

    const lastSeenFormatted = formatDateTime(device.lastSeen, i18n.language);

    // ── Breadcrumb items ────────────────────────────────────────────────
    const breadcrumbItems = device
        ? [
            { label: 'devices', href: '/devices' },
            { label: device.name, href: undefined },
        ]
        : [{ label: 'devices', href: '/devices' }];

    return (
        <div className="space-y-6">
            <Breadcrumbs items={breadcrumbItems} className="mb-2" />

            {/* Device Header */}
            <div className="flex flex-col md:flex-row md:items-start justify-between gap-4">
                <div className="flex items-start gap-4">
                    <div className="p-4 bg-slate-100 dark:bg-slate-800/80 dark:border dark:border-slate-700/50 rounded-xl">
                        <Camera className="w-8 h-8 text-slate-600 dark:text-slate-400" />
                    </div>
                    <div>
                        <div className="flex items-center gap-3">
                            <h1 className="text-2xl font-bold text-slate-900 dark:text-white">{device.name}</h1>
                            <StatusBadge status={device.status} />
                            <HealthBadge health={device.health} />
                        </div>
                        <p className="text-slate-500 dark:text-slate-400 mt-1">{device.siteName}</p>
                        <div className="flex items-center gap-4 mt-2 text-sm text-slate-500 dark:text-slate-400">
                            <span className="flex items-center gap-1"><MapPin className="w-4 h-4" /> {device.ipAddress}</span>
                            <span className="flex items-center gap-1"><Clock className="w-4 h-4" /> {t('last_seen')}: {lastSeenFormatted}</span>
                        </div>
                    </div>
                </div>
                <div className="flex gap-3">
                    <Button variant="outline" icon={<RefreshCw className="w-4 h-4" />}>{t('refresh')}</Button>
                    <Button variant="outline" icon={<QrCode className="w-4 h-4" />}
                      onClick={() => window.open(`/request?device_id=${deviceId}`, '_blank')}>
                      {t('qr_request') || 'QR Заявка'}
                    </Button>
                    <Button variant="outline" icon={<Settings className="w-4 h-4" />}>{t('configure')}</Button>
                </div>
            </div>

            {/* Summary Cards */}
            <div className="grid grid-cols-2 lg:grid-cols-4 gap-4">
                <Card><CardBody><div className="flex items-center gap-3"><div className={`p-2.5 rounded-xl ${device.status === 'online' ? 'bg-emerald-50 dark:bg-emerald-900/30' : device.status === 'warning' ? 'bg-amber-50 dark:bg-amber-900/30' : 'bg-red-50 dark:bg-red-900/30'}`}>{device.status === 'online' ? <Wifi className="w-5 h-5 text-emerald-600 dark:text-emerald-400" /> : device.status === 'warning' ? <AlertTriangle className="w-5 h-5 text-amber-600 dark:text-amber-400" /> : <WifiOff className="w-5 h-5 text-red-600 dark:text-red-400" />}</div><div><p className="text-sm text-slate-500 dark:text-slate-400">{t('status')}</p><p className="text-lg font-bold text-slate-900 dark:text-white capitalize">{device.status}</p></div></div></CardBody></Card>
                <Card><CardBody><div className="flex items-center gap-3"><div className="p-2.5 bg-blue-50 dark:bg-blue-900/30 rounded-xl"><Activity className="w-5 h-5 text-blue-600 dark:text-blue-400" /></div><div><p className="text-sm text-slate-500 dark:text-slate-400">{t('uptime')} <InfoTooltip text="Mean Time Between Failures (MTBF) — среднее время наработки на отказ. Показатель надёжности оборудования." glossaryTerm="MTBF" /></p><p className={`text-lg font-bold ${uptimeColor}`}>{stats?.uptimePercent.toFixed(1)}%</p></div></div></CardBody></Card>
                <Card><CardBody><div className="space-y-2"><div className="flex items-center gap-3"><div className="p-2.5 bg-purple-50 dark:bg-purple-900/30 rounded-xl"><HardDrive className="w-5 h-5 text-purple-600 dark:text-purple-400" /></div><div><p className="text-sm text-slate-500 dark:text-slate-400">{t('hdd_free')}</p><p className="text-lg font-bold text-slate-900 dark:text-white">{stats?.hddFreePercent}%</p></div></div><div className="w-full bg-slate-200 dark:bg-slate-700 rounded-full h-1.5"><div className={`${hddColor} h-1.5 rounded-full transition-all`} style={{ width: `${stats?.hddFreePercent ?? 0}%` }} /></div></div></CardBody></Card>
                <Card><CardBody><div className="flex items-center gap-3"><div className="p-2.5 bg-orange-50 dark:bg-orange-900/30 rounded-xl"><Thermometer className="w-5 h-5 text-orange-600 dark:text-orange-400" /></div><div><p className="text-sm text-slate-500 dark:text-slate-400">{t('temperature')}</p><p className={`text-lg font-bold ${(stats?.temperature ?? 0) > 50 ? 'text-red-600 dark:text-red-400' : (stats?.temperature ?? 0) > 40 ? 'text-amber-600 dark:text-amber-400' : 'text-slate-900 dark:text-white'}`}>{stats?.temperature}°C</p></div></div></CardBody></Card>
            </div>

            {/* Device Info & Camera List */}
            <div className="grid grid-cols-1 lg:grid-cols-2 gap-6">
                <Card><CardHeader>{t('device_information')}</CardHeader><CardBody><div className="space-y-3">
                    {[
                        [t('device_id'), device.id, true, false],
                        [t('type'), device.type.toUpperCase(), false, false],
                        [t('model'), device.model, false, false],
                        [t('firmware'), device.firmware, false, false],
                        [t('ip_address'), device.ipAddress, true, false],
                        [t('site'), device.siteName, false, false],
                        ['recording_status', device.recordingStatus.replace('_', ' '), false, true],
                        [t('assigned_technicians') || 'Assigned Technicians', t('view_in_site_settings') || 'Manage in site settings', false, false],
                    ].map(([label, value, mono, isRecording], idx, arr) => (
                        <div key={label as string} className={`flex justify-between py-2 ${idx < arr.length - 1 ? 'border-b border-slate-100 dark:border-slate-800/50' : ''}`}>
                            <span className="text-sm text-slate-500 dark:text-slate-400">
                                {isRecording ? t('recording_status') : label}
                                {isRecording && (
                                    <InfoTooltip
                                        text="Recording status — текущее состояние записи видео с устройства. Возможные значения: recording, stopped, error."
                                        glossaryTerm="NVR"
                                    />
                                )}
                            </span>
                            <span className={`text-sm font-medium text-slate-900 dark:text-white ${mono ? 'font-mono' : ''} capitalize`}>{value as string}</span>
                        </div>
                    ))}
                </div></CardBody></Card>
                <Card><CardHeader>{t('connected_cameras')} ({cameras.length})</CardHeader><CardBody>
                    {cameras.length === 0 ? (
                        <div className="text-center py-8"><Camera className="w-10 h-10 text-slate-300 dark:text-slate-600 mx-auto mb-3" /><p className="text-sm text-slate-500 dark:text-slate-400">{t('no_cameras_connected')}</p></div>
                    ) : (
                        <div className="space-y-3">
                            {cameras.map((cam) => (
                                <div key={cam.id} className="flex items-center justify-between p-3 bg-slate-50 dark:bg-slate-900/30 rounded-lg border border-transparent dark:border-slate-800">
                                    <div className="flex items-center gap-3"><div className={`w-2 h-2 rounded-full ${cam.status === 'online' ? 'bg-emerald-500' : cam.status === 'warning' ? 'bg-amber-500' : 'bg-red-500'}`} /><div><p className="text-sm font-medium text-slate-900 dark:text-white">{cam.name}</p><p className="text-xs text-slate-500 dark:text-slate-400">{t('channel')} {cam.channel} • {cam.type.toUpperCase()} • {cam.resolution}</p></div></div>
                                    <StatusBadge status={cam.status} />
                                </div>
                            ))}
                        </div>
                    )}
                </CardBody></Card>
            </div>

            {/* Health Timeline */}
            <Card><CardHeader>{t('health_timeline')} <InfoTooltip text="Health Score — комплексная оценка состояния устройства на основе uptime, температуры, свободного места на диске, частоты ошибок и статуса записи." glossaryTerm="health-score" /></CardHeader><CardBody>
                {timeline.length === 0 ? <p className="text-center text-sm text-slate-500 dark:text-slate-400 py-6">{t('no_events_recorded')}</p> : (
                    <div className="relative"><div className="absolute left-[17px] top-3 bottom-3 w-px bg-slate-200 dark:bg-slate-700" />
                        <div className="space-y-4">
                            {timeline.map((event) => (
                                <div key={event.id} className="flex items-start gap-4 relative">
                                    <div className={`relative z-10 flex items-center justify-center w-9 h-9 rounded-full border-2 bg-white dark:bg-slate-800 ${severityBorder[event.severity]}`}><TimelineIcon event={event} /></div>
                                    <div className="flex-1 pt-1">
                                        <p className="text-sm font-medium text-slate-900 dark:text-white">{event.message}</p>
                                        <div className="flex items-center gap-3 mt-1">
                                            <span className="text-xs text-slate-500 dark:text-slate-400">{formatDateTime(event.timestamp, i18n.language)}</span>
                                            <Badge variant={event.severity === 'error' ? 'danger' : event.severity === 'warning' ? 'warning' : event.severity === 'success' ? 'success' : 'info'} size="sm">{event.type.replace('_', ' ')}</Badge>
                                        </div>
                                    </div>
                                </div>
                            ))}
                        </div>
                    </div>
                )}
            </CardBody></Card>

            {/* ═══ P1-UX.7: RCA Widget — Summary card with full graph modal ═══ */}
            <RCAWidget deviceId={deviceId || ''} />

            {/* Recording Availability Calendar */}
            <Card><CardHeader>{t('recording_availability')}</CardHeader><CardBody>
                {calendarCameras.length === 0 ? <p className="text-center text-sm text-slate-500 dark:text-slate-400 py-6">{t('no_camera_data')}</p> : (
                    <>
                        <div className="flex flex-col sm:flex-row items-start sm:items-center justify-between gap-3 mb-5">
                            <div className="flex items-center gap-3">
                                <button onClick={goToPrevMonth} disabled={!canGoPrev} className="p-1.5 rounded-lg border border-slate-200 dark:border-slate-700 hover:bg-slate-100 dark:hover:bg-slate-800 disabled:opacity-30 disabled:cursor-not-allowed transition-colors" aria-label={t('previous_month')}><ChevronLeft className="w-4 h-4 text-slate-600 dark:text-slate-400" /></button>
                                <span className="text-sm font-semibold text-slate-900 dark:text-white min-w-[140px] text-center">{monthLabel}</span>
                                <button onClick={goToNextMonth} disabled={!canGoNext} className="p-1.5 rounded-lg border border-slate-200 dark:border-slate-700 hover:bg-slate-100 dark:hover:bg-slate-800 disabled:opacity-30 disabled:cursor-not-allowed transition-colors" aria-label={t('next_month')}><ChevronRight className="w-4 h-4 text-slate-600 dark:text-slate-400" /></button>
                            </div>
                            <div className="flex items-center gap-4">
                                <div className="flex items-center gap-1.5"><span className="w-3 h-3 rounded-sm bg-emerald-500" /><span className="text-xs text-slate-600 dark:text-slate-400">{t('available')}</span></div>
                                <div className="flex items-center gap-1.5"><span className="w-3 h-3 rounded-sm bg-red-500" /><span className="text-xs text-slate-600 dark:text-slate-400">{t('missing')}</span></div>
                                <div className="flex items-center gap-1.5"><span className="w-3 h-3 rounded-sm bg-slate-300 dark:bg-slate-600" /><span className="text-xs text-slate-600 dark:text-slate-400">{t('no_data')}</span></div>
                            </div>
                        </div>
                        <div className="overflow-x-auto">
                            <div className="min-w-[700px]">
                                <div className="flex"><div className="w-32 flex-shrink-0" /><div className="flex-1 grid" style={{ gridTemplateColumns: `repeat(${dateRange.length}, 1fr)` }}>{dateRange.map((date) => { const d = new Date(date); return <div key={date} className="text-center px-0.5"><span className="text-[10px] text-slate-400 dark:text-slate-500">{d.getDate()}</span></div>; })}</div></div>
                                {calendarCameras.map(([camId, camName]) => (
                                    <CameraRow key={camId} camId={camId} camName={camName} dateRange={dateRange} recordingDataMap={recordingDataMap} onSelect={handleSelectRecording} t={t} />
                                ))}
                            </div>
                        </div>
                        {(() => {
                            const now = new Date(); now.setHours(23,59,59,999);
                            const oneYearAgo = new Date(); oneYearAgo.setFullYear(now.getFullYear() - 1); oneYearAgo.setHours(0,0,0,0);
                            const monthStart = new Date(dateRange[0]);
                            const monthEnd = new Date(dateRange[dateRange.length - 1]); monthEnd.setHours(23,59,59,999);
                            const start = monthStart > oneYearAgo ? monthStart : oneYearAgo;
                            const end = monthEnd < now ? monthEnd : now;
                            if (start > end) return null;
                            const validDaysCount = Math.floor((end.getTime() - start.getTime()) / (1000*60*60*24)) + 1;
                            const totalCapacity = validDaysCount * cameras.length;
                            if (totalCapacity <= 0) return null;
                            let available = 0, missing = 0;
                            for (let d = new Date(start); d <= end; d.setDate(d.getDate() + 1)) {
                                const dateStr = d.toISOString().split('T')[0];
                                cameras.forEach(cam => {
                                    const entry = recordingDataMap.get(`${cam.id}-${dateStr}`);
                                    if (entry?.status === 'available') available++;
                                    else missing++;
                                });
                            }
                            const pctAvail = Math.round((available / totalCapacity) * 100);
                            const pctMissing = Math.round((missing / totalCapacity) * 100);
                            return (
                                <div className="mt-5 pt-4 border-t border-slate-100 dark:border-slate-800">
                                    <p className="text-xs font-medium text-slate-500 dark:text-slate-400 mb-2">{t('monthly_summary')}</p>
                                    <div className="flex h-1.5 rounded-full overflow-hidden bg-slate-200 dark:bg-slate-700"><div className="bg-emerald-500" style={{ width: `${pctAvail}%` }} /><div className="bg-red-500" style={{ width: `${pctMissing}%` }} /></div>
                                    <div className="flex justify-between mt-1 text-[10px] text-slate-500"><span>{pctAvail}% {t('available')}</span><span>{pctMissing}% {t('missing')}</span></div>
                                </div>
                            );
                        })()}
                    </>
                )}
            </CardBody></Card>

            {/* Related Tickets */}
            <Card>
                <CardHeader action={<Button variant="ghost" size="sm" onClick={() => setShowCreateTicketModal(true)}>{t('create_ticket')}</Button>}>{t('related_tickets')}</CardHeader>
                <CardBody>
                    {deviceTickets.length === 0 ? (
                        <div className="text-center py-8"><CheckCircle className="w-10 h-10 text-emerald-500 mx-auto mb-3" /><p className="text-sm font-medium text-slate-700 dark:text-slate-300">{t('no_open_tickets')}</p><p className="text-xs text-slate-500 dark:text-slate-400 mt-1">{t('this_device_has_no_tickets')}</p></div>
                    ) : (
                        <div className="space-y-3">
                            {deviceTickets.map((ticket) => (
                                <div key={ticket.id} className="p-3 bg-slate-50 dark:bg-slate-900/30 border border-transparent dark:border-slate-800 rounded-lg hover:bg-slate-100 dark:hover:bg-slate-800/70 cursor-pointer transition-all">
                                    <div className="flex items-start justify-between gap-2"><div><p className="text-sm font-medium text-slate-900 dark:text-white">{ticket.title}</p><p className="text-xs text-slate-500 dark:text-slate-400 mt-1">{ticket.id} • {ticket.assignee}</p></div><Badge variant={ticket.priority === 'critical' ? 'danger' : ticket.priority === 'high' ? 'warning' : ticket.priority === 'medium' ? 'info' : 'success'} size="sm">{ticket.priority}</Badge></div>
                                </div>
                            ))}
                        </div>
                    )}
                </CardBody>
            </Card>

            {/* ═══ Audit Log Tab ═══ */}
            <DeviceAuditLog deviceId={deviceId || ''} />

            {/* Missing Recording Modal */}
            <Modal isOpen={!!selectedRecording} onClose={() => setSelectedRecording(null)} title={t('missing_recording')} size="sm" footer={
                <div className="flex justify-end gap-3">
                    <button onClick={() => setSelectedRecording(null)} className="px-4 py-2 text-sm font-medium text-slate-700 dark:text-slate-300 bg-white dark:bg-slate-800 border border-slate-300 dark:border-slate-600 rounded-lg hover:bg-slate-50 dark:hover:bg-slate-700 transition-colors">{t('close')}</button>
                    <button onClick={handleCreateTicket} disabled={ticketCreated} className={`px-4 py-2 text-sm font-medium rounded-lg flex items-center gap-2 transition-colors ${ticketCreated ? 'bg-emerald-600 text-white cursor-default' : 'bg-blue-600 hover:bg-blue-700 text-white'}`}>{ticketCreated ? <><CheckCircle className="w-4 h-4" /> {t('ticket_created')}</> : t('create_ticket')}</button>
                </div>
            }>
                {selectedRecording && (
                    <div className="space-y-4">
                        <div className="p-4 bg-red-50 dark:bg-red-900/20 border border-red-200 dark:border-red-800/50 rounded-xl"><div className="flex items-center gap-2 mb-2"><AlertTriangle className="w-5 h-5 text-red-600 dark:text-red-400" /><span className="text-sm font-semibold text-red-700 dark:text-red-400">{t('recording_gap_detected')}</span></div><p className="text-sm text-red-600 dark:text-red-300">{t('no_recording_data')}</p></div>
                        <div className="space-y-3"><div className="flex justify-between py-2 border-b border-slate-100 dark:border-slate-800/50"><span className="text-sm text-slate-500 dark:text-slate-400">{t('date')}</span><span className="text-sm font-medium text-slate-900 dark:text-white">{formatDate(selectedRecording.date, i18n.language)}</span></div><div className="flex justify-between py-2 border-b border-slate-100 dark:border-slate-800/50"><span className="text-sm text-slate-500 dark:text-slate-400">{t('camera')}</span><span className="text-sm font-medium text-slate-900 dark:text-white">{selectedRecording.cameraName}</span></div><div className="flex justify-between py-2 border-b border-slate-100 dark:border-slate-800/50"><span className="text-sm text-slate-500 dark:text-slate-400">{t('device')}</span><span className="text-sm font-medium text-slate-900 dark:text-white">{device.name}</span></div><div className="flex justify-between py-2"><span className="text-sm text-slate-500 dark:text-slate-400">{t('site')}</span><span className="text-sm font-medium text-slate-900 dark:text-white">{device.siteName}</span></div></div>
                    </div>
                )}
            </Modal>

            {/* General Ticket Modal */}
            <Modal isOpen={showCreateTicketModal} onClose={() => setShowCreateTicketModal(false)} title={t('create_new_ticket')} size="md" footer={
                <div className="flex justify-end gap-3"><Button variant="outline" onClick={() => setShowCreateTicketModal(false)}>{t('cancel')}</Button><Button variant="primary" onClick={(e) => handleCreateGeneralTicket(e as any)}>{t('create_ticket_button')}</Button></div>
            }>
                <form onSubmit={handleCreateGeneralTicket} className="space-y-4">
                    <div><label className="block text-sm font-medium mb-1 text-slate-700 dark:text-slate-200">{t('title')}</label><Input value={newTicket.title} onChange={e => setNewTicket({ ...newTicket, title: e.target.value })} placeholder={t('title_placeholder')} required /></div>
                    <div><label className="block text-sm font-medium mb-1 text-slate-700 dark:text-slate-200">{t('description')}</label><Input value={newTicket.description} onChange={e => setNewTicket({ ...newTicket, description: e.target.value })} placeholder={t('description_placeholder')} required /></div>
                    <div><label className="block text-sm font-medium mb-1 text-slate-700 dark:text-slate-200">{t('priority')}</label><Select value={newTicket.priority} onChange={e => setNewTicket({ ...newTicket, priority: e.target.value })} options={[{ value: 'low', label: t('low') }, { value: 'medium', label: t('medium') }, { value: 'high', label: t('high') }, { value: 'critical', label: t('critical') }]} /></div>
                </form>
            </Modal>
        </div>
    );
}