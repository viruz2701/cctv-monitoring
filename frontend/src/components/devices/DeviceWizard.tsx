// ═══════════════════════════════════════════════════════════════════════
// DeviceWizard — Smart Device Onboarding (5-step wizard)
// P1-7: Smart Device Onboarding Wizard
//
// Steps:
//   1. IP/Domain → auto-detect model (ONVIF/Hikvision/Dahua)
//   2. Compatibility check (protocols: ONVIF, RTSP, HTTP API)
//   3. Capacity calculation (bandwidth, storage, PoE budget)
//   4. QR code generation for physical asset tag
//   5. Create WorkOrder for installation/connection
// ═══════════════════════════════════════════════════════════════════════

import React, { useState, useCallback, useMemo } from 'react';
import { useNavigate } from 'react-router-dom';
import {
    Search,
    CheckCircle,
    XCircle,
    Loader2,
    ArrowLeft,
    ArrowRight,
    Wifi,
    HardDrive,
    Zap,
    QrCode,
    ClipboardList,
    Monitor,
    Server,
    Camera,
    HelpCircle,
    AlertTriangle,
} from 'lucide-react';
import { api, type DeviceDetectionResult, type CapacityResult } from '../../services/api';
import { workOrdersApi } from '../../services/workOrdersApi';
import { useSites } from '../../hooks/useApiQuery';
import { useReducedMotion } from '../../hooks/useReducedMotion';
import { QRCode } from '../ui/QRCode';
import { ProgressBar } from '../ui/ProgressBar';
import { Button } from '../ui/Button';
import { Input, Select } from '../ui/Input';
import { Card, CardBody } from '../ui/Card';
import { useToast } from '../ui/Toast';

// ═══ Types ═══════════════════════════════════════════════════════════

interface WizardState {
    // Step 1
    ipOrDomain: string;
    username: string;
    password: string;
    // Step 1 — result
    detection: DeviceDetectionResult | null;
    detectionLoading: boolean;
    detectionError: string | null;
    // Step 2
    compatibilityChecked: boolean;
    // Step 3 — capacity
    resolution: string;
    fps: number;
    codec: 'H.264' | 'H.265' | 'MJPEG';
    retentionDays: number;
    camerasCount: number;
    poeWattage: number;
    capacity: CapacityResult | null;
    capacityLoading: boolean;
    // Step 5 — work order
    siteId: string;
    workType: 'installation' | 'maintenance' | 'repair' | 'inspection';
    priority: 'low' | 'medium' | 'high' | 'critical';
    description: string;
    scheduledDate: string;
}

const defaultWizardState = (): WizardState => ({
    ipOrDomain: '',
    username: '',
    password: '',
    detection: null,
    detectionLoading: false,
    detectionError: null,
    compatibilityChecked: false,
    resolution: '1080p',
    fps: 30,
    codec: 'H.265',
    retentionDays: 30,
    camerasCount: 1,
    poeWattage: 12.95,
    capacity: null,
    capacityLoading: false,
    siteId: '',
    workType: 'installation',
    priority: 'medium',
    description: '',
    scheduledDate: new Date(Date.now() + 7 * 24 * 60 * 60 * 1000).toISOString().split('T')[0],
});

// ═══ Step Config ═════════════════════════════════════════════════════

const STEPS = [
    { id: 0, label: 'IP / Auto-Detect', icon: Search },
    { id: 1, label: 'Compatibility', icon: CheckCircle },
    { id: 2, label: 'Capacity', icon: HardDrive },
    { id: 3, label: 'QR Code', icon: QrCode },
    { id: 4, label: 'Work Order', icon: ClipboardList },
] as const;

// ═══ Helper: resolution → Mbps ═══════════════════════════════════════

const RESOLUTION_BITRATES: Record<string, { h264: number; h265: number }> = {
    '4K':      { h264: 25, h265: 12 },
    '5MP':     { h264: 12, h265: 6 },
    '4MP':     { h264: 10, h265: 5 },
    '1080p':   { h264: 6,  h265: 3 },
    '720p':    { h264: 3,  h265: 1.5 },
    'D1':      { h264: 1.5, h265: 0.8 },
};

// ═══ Component ═══════════════════════════════════════════════════════

interface DeviceWizardProps {
    onClose: () => void;
    onComplete?: (deviceId?: string) => void;
}

export function DeviceWizard({ onClose, onComplete }: DeviceWizardProps) {
    const navigate = useNavigate();
    const prefersReduced = useReducedMotion();
    const { data: apiSites = [] } = useSites();
    const toast = useToast();

    const [step, setStep] = useState(0);
    const [wizard, setWizard] = useState<WizardState>(defaultWizardState);
    const [submitting, setSubmitting] = useState(false);

    // ── Update helper ────────────────────────────────────────────────
    const update = useCallback((patch: Partial<WizardState>) => {
        setWizard(prev => ({ ...prev, ...patch }));
    }, []);

    // ── Steps validation ─────────────────────────────────────────────

    const canProceed = useMemo(() => {
        switch (step) {
            case 0: return wizard.detection?.detected === true;
            case 1: return wizard.compatibilityChecked;
            case 2: return wizard.capacity !== null;
            case 3: return true; // QR step — always proceed
            case 4: return wizard.siteId !== '' && wizard.description.trim().length > 0;
            default: return false;
        }
    }, [step, wizard]);

    // ── Step 1: Auto-Detect ──────────────────────────────────────────

    const handleDetect = useCallback(async () => {
        if (!wizard.ipOrDomain.trim()) return;
        update({ detectionLoading: true, detectionError: null, detection: null });

        try {
            const result = await api.detectDevice(wizard.ipOrDomain.trim(), {
                username: wizard.username || undefined,
                password: wizard.password || undefined,
            });

            if (!result.detected) {
                update({
                    detection: result,
                    detectionLoading: false,
                    detectionError: result.error || 'Device not detected on this IP/domain',
                });
                return;
            }

            // Auto-fill capacity params from detection
            const resolution = result.stream_urls?.length
                ? (result.stream_urls[0].includes('1080') ? '1080p' : '4K')
                : '1080p';

            update({
                detection: result,
                detectionLoading: false,
                resolution,
            });
        } catch (err) {
            update({
                detectionLoading: false,
                detectionError: err instanceof Error ? err.message : 'Detection failed',
            });
        }
    }, [wizard.ipOrDomain, wizard.username, wizard.password, update]);

    // ── Step 2: Compatibility Check ──────────────────────────────────

    const handleRunCompatibility = useCallback(() => {
        update({ compatibilityChecked: true });
    }, [update]);

    // ── Step 3: Capacity Calculation ─────────────────────────────────

    const handleCalculateCapacity = useCallback(async () => {
        update({ capacityLoading: true });
        try {
            const result = await api.calculateDeviceCapacity({
                resolution: wizard.resolution,
                fps: wizard.fps,
                codec: wizard.codec,
                retention_days: wizard.retentionDays,
                cameras_count: wizard.camerasCount,
                poe_wattage: wizard.poeWattage,
            });
            update({ capacity: result, capacityLoading: false });
        } catch {
            // Fallback: local calculation
            const codecKey = wizard.codec === 'H.265' ? 'h265' : 'h264';
            const bitrateMap = RESOLUTION_BITRATES[wizard.resolution];
            const perCameraMbps = bitrateMap
                ? (bitrateMap[codecKey] || 6) * (wizard.fps / 30)
                : 6;

            const bandwidthMbps = perCameraMbps * wizard.camerasCount;
            const storageGbPerDay = (bandwidthMbps * 1000 * 1000 / 8) * 86400 / (1024 * 1024 * 1024);
            const totalStorage = storageGbPerDay * wizard.retentionDays;
            const poeBudget = wizard.camerasCount * wizard.poeWattage;

            update({
                capacity: {
                    bandwidth_mbps: Math.round(bandwidthMbps * 100) / 100,
                    storage_gb: Math.round(totalStorage),
                    poe_budget_watts: Math.round(poeBudget),
                    recommended_nvr: poeBudget > 200 ? 'Enterprise NVR' : 'Mid-range NVR',
                    warnings: bandwidthMbps > 100
                        ? ['High bandwidth — consider link aggregation']
                        : [],
                },
                capacityLoading: false,
            });
        }
    }, [wizard.resolution, wizard.fps, wizard.codec, wizard.retentionDays, wizard.camerasCount, wizard.poeWattage, update]);

    // ── Step 5: Create Work Order ────────────────────────────────────

    const handleCreateWorkOrder = useCallback(async () => {
        setSubmitting(true);
        try {
            const wo = await workOrdersApi.createWorkOrder({
                device_id: wizard.detection?.model || '',
                type: wizard.workType === 'installation' ? 'preventive'
                    : wizard.workType === 'repair' ? 'corrective'
                    : 'preventive',
                priority: wizard.priority,
                notes: [
                    `[Onboarding Wizard] ${wizard.description}`,
                    wizard.detection?.model ? `Model: ${wizard.detection.model}` : '',
                    wizard.detection?.vendor ? `Vendor: ${wizard.detection.vendor}` : '',
                    `IP: ${wizard.ipOrDomain}`,
                ].filter(Boolean).join('\n'),
                checklist: [
                    { task: 'Mount device at location', completed: false },
                    { task: 'Connect network cable', completed: false },
                    { task: 'Verify device powers on', completed: false },
                    { task: 'Configure network settings', completed: false },
                    { task: 'Verify video stream', completed: false },
                    { task: 'Tag device with asset QR code', completed: false },
                ],
            });

            toast.success('Work order created successfully');
            setSubmitting(false);
            onComplete?.(wo.id);
            navigate(`/work-orders/${wo.id}`);
        } catch (err) {
            toast.error(
                err instanceof Error ? err.message : 'Failed to create work order',
            );
            setSubmitting(false);
        }
    }, [wizard, navigate, onComplete, toast]);

    // ── Navigation ───────────────────────────────────────────────────

    const handleNext = useCallback(() => {
        if (step === 2 && !wizard.capacity) {
            handleCalculateCapacity();
            return;
        }
        if (step < STEPS.length - 1) setStep(s => s + 1);
    }, [step, wizard.capacity, handleCalculateCapacity]);

    const handleBack = useCallback(() => {
        if (step > 0) setStep(s => s - 1);
        else onClose();
    }, [step, onClose]);

    // ── QR Value ─────────────────────────────────────────────────────

    const qrValue = useMemo(() => {
        const base = wizard.detection?.model || wizard.ipOrDomain;
        return JSON.stringify({
            type: 'cctv-asset',
            model: wizard.detection?.model || '',
            vendor: wizard.detection?.vendor || '',
            ip: wizard.ipOrDomain,
            mac: wizard.detection?.mac_address || '',
            generated: new Date().toISOString(),
        });
    }, [wizard.detection, wizard.ipOrDomain]);

    // ═══ Render Steps ═════════════════════════════════════════════════

    const renderStep = () => {
        switch (step) {
            case 0: return renderStep1Detect();
            case 1: return renderStep2Compatibility();
            case 2: return renderStep3Capacity();
            case 3: return renderStep4QR();
            case 4: return renderStep5WorkOrder();
            default: return null;
        }
    };

    // ── Step 1: IP / Auto-Detect ─────────────────────────────────────

    const renderStep1Detect = () => (
        <div className="space-y-5">
            <div className="flex items-start gap-3 p-4 bg-blue-50 dark:bg-blue-900/20 rounded-lg border border-blue-200 dark:border-blue-800">
                <Search className="w-5 h-5 text-blue-600 dark:text-blue-400 mt-0.5 shrink-0" />
                <div className="text-sm text-blue-800 dark:text-blue-200">
                    <p className="font-medium mb-1">Enter device IP address or domain name</p>
                    <p className="text-blue-600 dark:text-blue-300">
                        The wizard will attempt to auto-detect the device model using
                        ONVIF Profile S/T, RTSP, and HTTP API (Hikvision/Dahua).
                    </p>
                </div>
            </div>

            <div>
                <label className="block text-sm font-medium text-slate-700 dark:text-slate-300 mb-1">
                    IP Address or Domain <span className="text-red-500">*</span>
                </label>
                <Input
                    value={wizard.ipOrDomain}
                    onChange={(e) => update({ ipOrDomain: e.target.value })}
                    placeholder="e.g. 192.168.1.100 or camera.example.com"
                    onKeyDown={(e: React.KeyboardEvent) => e.key === 'Enter' && handleDetect()}
                />
            </div>

            <div className="grid grid-cols-2 gap-4">
                <div>
                    <label className="block text-sm font-medium text-slate-700 dark:text-slate-300 mb-1">
                        Username <span className="text-slate-400">(optional)</span>
                    </label>
                    <Input
                        value={wizard.username}
                        onChange={(e) => update({ username: e.target.value })}
                        placeholder="admin"
                    />
                </div>
                <div>
                    <label className="block text-sm font-medium text-slate-700 dark:text-slate-300 mb-1">
                        Password <span className="text-slate-400">(optional)</span>
                    </label>
                    <Input
                        type="password"
                        value={wizard.password}
                        onChange={(e) => update({ password: e.target.value })}
                        placeholder="••••••••"
                    />
                </div>
            </div>

            <Button
                variant="primary"
                onClick={handleDetect}
                disabled={!wizard.ipOrDomain.trim() || wizard.detectionLoading}
                icon={wizard.detectionLoading ? <Loader2 className="w-4 h-4 animate-spin" /> : <Search className="w-4 h-4" />}
            >
                {wizard.detectionLoading ? 'Detecting...' : 'Detect Device'}
            </Button>

            {wizard.detectionError && (
                <div className="flex items-center gap-2 p-3 bg-red-50 dark:bg-red-900/20 rounded-lg border border-red-200 dark:border-red-800">
                    <XCircle className="w-5 h-5 text-red-500 shrink-0" />
                    <span className="text-sm text-red-700 dark:text-red-300">{wizard.detectionError}</span>
                </div>
            )}

            {wizard.detection?.detected && (
                <div className="space-y-3 p-4 bg-emerald-50 dark:bg-emerald-900/20 rounded-lg border border-emerald-200 dark:border-emerald-800">
                    <div className="flex items-center gap-2 text-emerald-700 dark:text-emerald-300">
                        <CheckCircle className="w-5 h-5" />
                        <span className="font-medium">Device detected successfully</span>
                    </div>
                    <div className="grid grid-cols-2 gap-3 text-sm">
                        <div><span className="text-slate-500">Model:</span> <span className="font-medium">{wizard.detection.model || 'Unknown'}</span></div>
                        <div><span className="text-slate-500">Vendor:</span> <span className="font-medium capitalize">{wizard.detection.vendor || 'Unknown'}</span></div>
                        {wizard.detection.firmware && (
                            <div><span className="text-slate-500">Firmware:</span> <span className="font-medium">{wizard.detection.firmware}</span></div>
                        )}
                        {wizard.detection.mac_address && (
                            <div><span className="text-slate-500">MAC:</span> <span className="font-medium font-mono text-xs">{wizard.detection.mac_address}</span></div>
                        )}
                    </div>
                    <div className="flex flex-wrap gap-2">
                        {wizard.detection.onvif_profile_s && (
                            <span className="px-2 py-1 text-xs font-medium bg-blue-100 dark:bg-blue-900/30 text-blue-700 dark:text-blue-300 rounded-full">ONVIF Profile S</span>
                        )}
                        {wizard.detection.onvif_profile_t && (
                            <span className="px-2 py-1 text-xs font-medium bg-purple-100 dark:bg-purple-900/30 text-purple-700 dark:text-purple-300 rounded-full">ONVIF Profile T</span>
                        )}
                        {wizard.detection.rtsp_supported && (
                            <span className="px-2 py-1 text-xs font-medium bg-emerald-100 dark:bg-emerald-900/30 text-emerald-700 dark:text-emerald-300 rounded-full">RTSP</span>
                        )}
                        {wizard.detection.http_api_supported && (
                            <span className="px-2 py-1 text-xs font-medium bg-amber-100 dark:bg-amber-900/30 text-amber-700 dark:text-amber-300 rounded-full">HTTP API</span>
                        )}
                    </div>
                </div>
            )}
        </div>
    );

    // ── Step 2: Compatibility Check ──────────────────────────────────

    const renderStep2Compatibility = () => {
        const protocols = wizard.detection?.protocols || [];
        const checks = [
            { label: 'ONVIF Profile S (streaming)', passed: wizard.detection?.onvif_profile_s ?? false },
            { label: 'ONVIF Profile T (advanced)', passed: wizard.detection?.onvif_profile_t ?? false },
            { label: 'RTSP (real-time streaming)', passed: wizard.detection?.rtsp_supported ?? false },
            { label: 'HTTP API (management)', passed: wizard.detection?.http_api_supported ?? false },
            { label: `Protocols detected: ${protocols.length > 0 ? protocols.join(', ') : 'None'}`, passed: protocols.length > 0 },
        ];

        const allPassed = checks.every(c => c.passed);

        return (
            <div className="space-y-5">
                <div className="flex items-start gap-3 p-4 bg-amber-50 dark:bg-amber-900/20 rounded-lg border border-amber-200 dark:border-amber-800">
                    <CheckCircle className="w-5 h-5 text-amber-600 dark:text-amber-400 mt-0.5 shrink-0" />
                    <div className="text-sm text-amber-800 dark:text-amber-200">
                        <p className="font-medium mb-1">Protocol Compatibility Check</p>
                        <p className="text-amber-600 dark:text-amber-300">
                            Verify which protocols this device supports for integration.
                        </p>
                    </div>
                </div>

                <div className="space-y-3">
                    {checks.map((check) => (
                        <div
                            key={check.label}
                            className={`flex items-center gap-3 p-3 rounded-lg border ${
                                check.passed
                                    ? 'bg-emerald-50 dark:bg-emerald-900/20 border-emerald-200 dark:border-emerald-800'
                                    : 'bg-slate-50 dark:bg-slate-800/50 border-slate-200 dark:border-slate-700'
                            }`}
                        >
                            {check.passed ? (
                                <CheckCircle className="w-5 h-5 text-emerald-500 shrink-0" />
                            ) : (
                                <XCircle className="w-5 h-5 text-slate-300 dark:text-slate-600 shrink-0" />
                            )}
                            <span className={`text-sm ${check.passed ? 'text-emerald-700 dark:text-emerald-300' : 'text-slate-500 dark:text-slate-400'}`}>
                                {check.label}
                            </span>
                        </div>
                    ))}
                </div>

                <div className={`p-3 rounded-lg text-sm ${
                    allPassed
                        ? 'bg-emerald-50 dark:bg-emerald-900/20 text-emerald-700 dark:text-emerald-300 border border-emerald-200 dark:border-emerald-800'
                        : 'bg-amber-50 dark:bg-amber-900/20 text-amber-700 dark:text-amber-300 border border-amber-200 dark:border-amber-800'
                }`}>
                    {allPassed
                        ? '✓ All protocols compatible — device is ready for integration.'
                        : '⚠ Some protocols are not supported. Basic functionality will still work.'}
                </div>

                {!wizard.compatibilityChecked && (
                    <Button variant="primary" onClick={handleRunCompatibility} icon={<CheckCircle className="w-4 h-4" />}>
                        Confirm Compatibility
                    </Button>
                )}

                {wizard.compatibilityChecked && (
                    <div className="flex items-center gap-2 text-emerald-600 dark:text-emerald-400 text-sm font-medium">
                        <CheckCircle className="w-4 h-4" />
                        Compatibility verified
                    </div>
                )}
            </div>
        );
    };

    // ── Step 3: Capacity Calculation ─────────────────────────────────

    const renderStep3Capacity = () => (
        <div className="space-y-5">
            <div className="flex items-start gap-3 p-4 bg-indigo-50 dark:bg-indigo-900/20 rounded-lg border border-indigo-200 dark:border-indigo-800">
                <HardDrive className="w-5 h-5 text-indigo-600 dark:text-indigo-400 mt-0.5 shrink-0" />
                <div className="text-sm text-indigo-800 dark:text-indigo-200">
                    <p className="font-medium mb-1">Capacity Planning</p>
                    <p className="text-indigo-600 dark:text-indigo-300">
                        Calculate bandwidth, storage requirements, and PoE budget.
                    </p>
                </div>
            </div>

            <div className="grid grid-cols-2 gap-4">
                <div>
                    <label className="block text-sm font-medium text-slate-700 dark:text-slate-300 mb-1">Resolution</label>
                    <Select
                        value={wizard.resolution}
                        onChange={(e) => update({ resolution: e.target.value })}
                        options={[
                            { value: '4K', label: '4K (3840×2160)' },
                            { value: '5MP', label: '5MP (2592×1944)' },
                            { value: '4MP', label: '4MP (2688×1520)' },
                            { value: '1080p', label: '1080p (1920×1080)' },
                            { value: '720p', label: '720p (1280×720)' },
                            { value: 'D1', label: 'D1 (704×576)' },
                        ]}
                    />
                </div>
                <div>
                    <label className="block text-sm font-medium text-slate-700 dark:text-slate-300 mb-1">FPS</label>
                    <Select
                        value={String(wizard.fps)}
                        onChange={(e) => update({ fps: Number(e.target.value) })}
                        options={[
                            { value: '30', label: '30 fps (smooth)' },
                            { value: '25', label: '25 fps (PAL)' },
                            { value: '15', label: '15 fps (balanced)' },
                            { value: '10', label: '10 fps (economy)' },
                            { value: '5', label: '5 fps (minimal)' },
                        ]}
                    />
                </div>
                <div>
                    <label className="block text-sm font-medium text-slate-700 dark:text-slate-300 mb-1">Codec</label>
                    <Select
                        value={wizard.codec}
                        onChange={(e) => update({ codec: e.target.value as 'H.264' | 'H.265' | 'MJPEG' })}
                        options={[
                            { value: 'H.265', label: 'H.265 / HEVC (recommended)' },
                            { value: 'H.264', label: 'H.264 / AVC' },
                            { value: 'MJPEG', label: 'MJPEG (legacy)' },
                        ]}
                    />
                </div>
                <div>
                    <label className="block text-sm font-medium text-slate-700 dark:text-slate-300 mb-1">
                        Retention Days
                    </label>
                    <Input
                        type="number"
                        min={1}
                        max={365}
                        value={wizard.retentionDays}
                        onChange={(e) => update({ retentionDays: Math.max(1, Number(e.target.value)) })}
                    />
                </div>
                <div>
                    <label className="block text-sm font-medium text-slate-700 dark:text-slate-300 mb-1">
                        Camera Count
                    </label>
                    <Input
                        type="number"
                        min={1}
                        max={256}
                        value={wizard.camerasCount}
                        onChange={(e) => update({ camerasCount: Math.max(1, Number(e.target.value)) })}
                    />
                </div>
                <div>
                    <label className="block text-sm font-medium text-slate-700 dark:text-slate-300 mb-1">
                        PoE Wattage <span className="text-slate-400">(per camera)</span>
                    </label>
                    <Select
                        value={String(wizard.poeWattage)}
                        onChange={(e) => update({ poeWattage: Number(e.target.value) })}
                        options={[
                            { value: '12.95', label: '12.95W (802.3af)' },
                            { value: '25.5', label: '25.5W (802.3at)' },
                            { value: '60', label: '60W (802.3bt)' },
                            { value: '100', label: '100W (802.3bt)' },
                        ]}
                    />
                </div>
            </div>

            {!wizard.capacity && (
                <Button
                    variant="primary"
                    onClick={handleCalculateCapacity}
                    disabled={wizard.capacityLoading}
                    icon={wizard.capacityLoading ? <Loader2 className="w-4 h-4 animate-spin" /> : <HardDrive className="w-4 h-4" />}
                >
                    {wizard.capacityLoading ? 'Calculating...' : 'Calculate Capacity'}
                </Button>
            )}

            {wizard.capacity && (
                <div className="space-y-3 p-4 bg-emerald-50 dark:bg-emerald-900/20 rounded-lg border border-emerald-200 dark:border-emerald-800">
                    <h4 className="text-sm font-semibold text-emerald-800 dark:text-emerald-200">Capacity Results</h4>
                    <div className="grid grid-cols-3 gap-4">
                        <Card>
                            <CardBody>
                                <div className="flex items-center gap-2">
                                    <Wifi className="w-4 h-4 text-blue-500" />
                                    <div>
                                        <p className="text-[10px] uppercase text-slate-500 tracking-wide">Bandwidth</p>
                                        <p className="text-lg font-bold text-slate-900 dark:text-white">
                                            {wizard.capacity.bandwidth_mbps} <span className="text-xs font-normal text-slate-400">Mbps</span>
                                        </p>
                                    </div>
                                </div>
                            </CardBody>
                        </Card>
                        <Card>
                            <CardBody>
                                <div className="flex items-center gap-2">
                                    <HardDrive className="w-4 h-4 text-indigo-500" />
                                    <div>
                                        <p className="text-[10px] uppercase text-slate-500 tracking-wide">Storage</p>
                                        <p className="text-lg font-bold text-slate-900 dark:text-white">
                                            {wizard.capacity.storage_gb} <span className="text-xs font-normal text-slate-400">GB</span>
                                        </p>
                                    </div>
                                </div>
                            </CardBody>
                        </Card>
                        <Card>
                            <CardBody>
                                <div className="flex items-center gap-2">
                                    <Zap className="w-4 h-4 text-amber-500" />
                                    <div>
                                        <p className="text-[10px] uppercase text-slate-500 tracking-wide">PoE Budget</p>
                                        <p className="text-lg font-bold text-slate-900 dark:text-white">
                                            {wizard.capacity.poe_budget_watts} <span className="text-xs font-normal text-slate-400">W</span>
                                        </p>
                                    </div>
                                </div>
                            </CardBody>
                        </Card>
                    </div>
                    <div className="text-xs text-slate-500">
                        <p>Recommended NVR: <span className="font-medium">{wizard.capacity.recommended_nvr}</span></p>
                        {wizard.capacity.warnings.map((w, i) => (
                            <p key={i} className="text-amber-600 dark:text-amber-400 flex items-center gap-1 mt-1">
                                <AlertTriangle className="w-3 h-3" /> {w}
                            </p>
                        ))}
                    </div>
                </div>
            )}
        </div>
    );

    // ── Step 4: QR Code ──────────────────────────────────────────────

    const renderStep4QR = () => (
        <div className="space-y-5">
            <div className="flex items-start gap-3 p-4 bg-purple-50 dark:bg-purple-900/20 rounded-lg border border-purple-200 dark:border-purple-800">
                <QrCode className="w-5 h-5 text-purple-600 dark:text-purple-400 mt-0.5 shrink-0" />
                <div className="text-sm text-purple-800 dark:text-purple-200">
                    <p className="font-medium mb-1">Asset Tag — QR Code</p>
                    <p className="text-purple-600 dark:text-purple-300">
                        Print and attach this QR code to the physical device for mobile inventory management.
                    </p>
                </div>
            </div>

            <div className="flex flex-col items-center py-4">
                <QRCode
                    value={qrValue}
                    size={200}
                    label="Scan for asset info"
                />
                <p className="mt-3 text-xs text-slate-400 font-mono max-w-sm break-all text-center">
                    {qrValue}
                </p>
            </div>

            <div className="p-3 bg-slate-50 dark:bg-slate-800/50 rounded-lg border border-slate-200 dark:border-slate-700">
                <h4 className="text-sm font-medium text-slate-700 dark:text-slate-300 mb-2">Asset Info</h4>
                <div className="grid grid-cols-2 gap-2 text-xs text-slate-500">
                    <div>Model: <span className="font-medium text-slate-700 dark:text-slate-300">{wizard.detection?.model || 'N/A'}</span></div>
                    <div>Vendor: <span className="font-medium text-slate-700 dark:text-slate-300 capitalize">{wizard.detection?.vendor || 'N/A'}</span></div>
                    <div>IP: <span className="font-medium text-slate-700 dark:text-slate-300 font-mono">{wizard.ipOrDomain}</span></div>
                    <div>MAC: <span className="font-medium text-slate-700 dark:text-slate-300 font-mono">{wizard.detection?.mac_address || 'N/A'}</span></div>
                </div>
            </div>
        </div>
    );

    // ── Step 5: Create Work Order ────────────────────────────────────

    const renderStep5WorkOrder = () => (
        <div className="space-y-5">
            <div className="flex items-start gap-3 p-4 bg-slate-50 dark:bg-slate-800/50 rounded-lg border border-slate-200 dark:border-slate-700">
                <ClipboardList className="w-5 h-5 text-slate-600 dark:text-slate-400 mt-0.5 shrink-0" />
                <div className="text-sm text-slate-700 dark:text-slate-300">
                    <p className="font-medium mb-1">Create Installation Work Order</p>
                    <p className="text-slate-500 dark:text-slate-400">
                        Generate a work order for physical installation and connection of this device.
                    </p>
                </div>
            </div>

            <div>
                <label className="block text-sm font-medium text-slate-700 dark:text-slate-300 mb-1">
                    Site <span className="text-red-500">*</span>
                </label>
                <Select
                    value={wizard.siteId}
                    onChange={(e) => update({ siteId: e.target.value })}
                    options={[
                        { value: '', label: 'Select site...' },
                        ...apiSites.map(s => ({ value: s.id, label: s.name })),
                    ]}
                />
            </div>

            <div className="grid grid-cols-2 gap-4">
                <div>
                    <label className="block text-sm font-medium text-slate-700 dark:text-slate-300 mb-1">Work Type</label>
                    <Select
                        value={wizard.workType}
                        onChange={(e) => update({ workType: e.target.value as WizardState['workType'] })}
                        options={[
                            { value: 'installation', label: 'Installation' },
                            { value: 'maintenance', label: 'Maintenance' },
                            { value: 'repair', label: 'Repair' },
                            { value: 'inspection', label: 'Inspection' },
                        ]}
                    />
                </div>
                <div>
                    <label className="block text-sm font-medium text-slate-700 dark:text-slate-300 mb-1">Priority</label>
                    <Select
                        value={wizard.priority}
                        onChange={(e) => update({ priority: e.target.value as WizardState['priority'] })}
                        options={[
                            { value: 'low', label: 'Low' },
                            { value: 'medium', label: 'Medium' },
                            { value: 'high', label: 'High' },
                            { value: 'critical', label: 'Critical' },
                        ]}
                    />
                </div>
            </div>

            <div>
                <label className="block text-sm font-medium text-slate-700 dark:text-slate-300 mb-1">
                    Description <span className="text-red-500">*</span>
                </label>
                <textarea
                    className="w-full rounded-lg border border-slate-300 dark:border-slate-600 bg-white dark:bg-slate-800 px-3 py-2 text-sm text-slate-900 dark:text-white placeholder:text-slate-400 focus:outline-none focus:ring-2 focus:ring-blue-500 min-h-[80px]"
                    value={wizard.description}
                    onChange={(e) => update({ description: e.target.value })}
                    placeholder="Describe the installation task..."
                />
            </div>

            <div>
                <label className="block text-sm font-medium text-slate-700 dark:text-slate-300 mb-1">Scheduled Date</label>
                <Input
                    type="date"
                    value={wizard.scheduledDate}
                    onChange={(e) => update({ scheduledDate: e.target.value })}
                />
            </div>

            <div className="p-3 bg-slate-50 dark:bg-slate-800/50 rounded-lg border border-slate-200 dark:border-slate-700">
                <h4 className="text-sm font-medium text-slate-700 dark:text-slate-300 mb-2">Installation Checklist</h4>
                <ul className="space-y-1.5">
                    {[
                        'Mount device at location',
                        'Connect network cable',
                        'Verify device powers on',
                        'Configure network settings',
                        'Verify video stream',
                        'Tag device with asset QR code',
                    ].map((item) => (
                        <li key={item} className="flex items-center gap-2 text-xs text-slate-500">
                            <span className="w-1.5 h-1.5 bg-slate-300 dark:bg-slate-600 rounded-full" />
                            {item}
                        </li>
                    ))}
                </ul>
            </div>
        </div>
    );

    // ═══ Main Render ═════════════════════════════════════════════════

    const totalSteps = STEPS.length;

    return (
        <div className="flex flex-col h-full" role="dialog" aria-label="Smart Device Onboarding Wizard" aria-describedby="wizard-description">
            {/* Progress bar */}
            <div className="mb-6">
                <div className="flex items-center justify-between mb-2">
                    <span className="text-xs font-medium text-slate-500">
                        Step {step + 1} of {totalSteps}
                    </span>
                    <span className="text-xs font-medium text-slate-500">
                        {Math.round(((step + 1) / totalSteps) * 100)}%
                    </span>
                </div>
                <ProgressBar
                    value={step + 1}
                    max={totalSteps}
                    variant="info"
                    size="sm"
                />
                <div className="flex justify-between mt-2">
                    {STEPS.map((s, i) => {
                        const StepIcon = s.icon;
                        const isActive = i === step;
                        const isDone = i < step;
                        return (
                            <button
                                key={s.id}
                                type="button"
                                disabled
                                className={`flex flex-col items-center gap-1 transition-opacity ${
                                    isActive ? 'opacity-100' : isDone ? 'opacity-70' : 'opacity-40'
                                }`}
                                aria-current={isActive ? 'step' : undefined}
                            >
                                <div className={`p-1.5 rounded-full ${
                                    isActive
                                        ? 'bg-blue-100 dark:bg-blue-900/30 text-blue-600 dark:text-blue-400'
                                        : isDone
                                            ? 'bg-emerald-100 dark:bg-emerald-900/30 text-emerald-600 dark:text-emerald-400'
                                            : 'bg-slate-100 dark:bg-slate-800 text-slate-400'
                                }`}>
                                    {isDone ? (
                                        <CheckCircle className="w-3.5 h-3.5" />
                                    ) : (
                                        <StepIcon className="w-3.5 h-3.5" />
                                    )}
                                </div>
                                <span className={`text-[10px] leading-tight text-center ${
                                    isActive ? 'font-medium text-blue-600 dark:text-blue-400' : 'text-slate-400'
                                }`}>
                                    {s.label}
                                </span>
                            </button>
                        );
                    })}
                </div>
            </div>

            {/* Step title */}
            <h3 id="wizard-description" className="text-lg font-semibold text-slate-900 dark:text-white mb-4 flex items-center gap-2">
                {step === 0 && <><Search className="w-5 h-5" /> IP / Auto-Detect</>}
                {step === 1 && <><CheckCircle className="w-5 h-5" /> Compatibility Check</>}
                {step === 2 && <><HardDrive className="w-5 h-5" /> Capacity Planning</>}
                {step === 3 && <><QrCode className="w-5 h-5" /> Asset QR Code</>}
                {step === 4 && <><ClipboardList className="w-5 h-5" /> Create Work Order</>}
            </h3>

            {/* Step content — scrollable */}
            <div
                className={`flex-1 overflow-y-auto ${prefersReduced ? '' : 'animate-in fade-in slide-in-from-bottom-4 duration-200'}`}
                style={{ minHeight: 0 }}
            >
                {renderStep()}
            </div>

            {/* Navigation footer */}
            <div className="flex items-center justify-between pt-6 border-t border-slate-200 dark:border-slate-700 mt-6">
                <Button
                    variant="outline"
                    onClick={handleBack}
                    icon={<ArrowLeft className="w-4 h-4" />}
                >
                    {step === 0 ? 'Cancel' : 'Back'}
                </Button>

                {step < totalSteps - 1 ? (
                    <Button
                        variant="primary"
                        onClick={handleNext}
                        disabled={!canProceed}
                        icon={<ArrowRight className="w-4 h-4" />}
                        iconPosition="right"
                    >
                        Next
                    </Button>
                ) : (
                    <Button
                        variant="primary"
                        onClick={handleCreateWorkOrder}
                        disabled={!canProceed || submitting}
                        icon={submitting ? <Loader2 className="w-4 h-4 animate-spin" /> : <ClipboardList className="w-4 h-4" />}
                    >
                        {submitting ? 'Creating...' : 'Create Work Order'}
                    </Button>
                )}
            </div>
        </div>
    );
}
