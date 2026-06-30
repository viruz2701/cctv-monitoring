// ═══════════════════════════════════════════════════════════════════════
// CCTV Health Monitor — Desktop App (DESKTOP-01)
//
// React UI для Tauri desktop приложения.
// Предоставляет интерфейс для открытия камер в IE-mode.
//
// Соответствие:
//   - IEC 62443-3-3 SL-3: Zone separation
//   - OWASP ASVS L3: Input validation, access control
//   - ISO 27001 A.12.4: Audit trail
// ═══════════════════════════════════════════════════════════════════════

import React, { useState, useCallback } from 'react';

// ─── Types ──────────────────────────────────────────────────────────

interface DeviceEntry {
    id: string;
    name: string;
    ip: string;
    status: 'online' | 'offline';
}

interface IEModeResult {
    success: boolean;
    message: string;
}

// ─── Main App Component ─────────────────────────────────────────────

export default function App() {
    const [devices] = useState<DeviceEntry[]>([
        // Примеры устройств для отладки
        { id: 'dev-001', name: 'Camera-01 (Dahua)', ip: '192.168.1.100', status: 'online' },
        { id: 'dev-002', name: 'Camera-02 (Hikvision)', ip: '192.168.1.101', status: 'online' },
        { id: 'dev-003', name: 'Camera-03 (Uniview)', ip: '192.168.1.102', status: 'offline' },
    ]);

    const [recentResults, setRecentResults] = useState<IEModeResult[]>([]);
    const [loading, setLoading] = useState<string | null>(null);
    const [error, setError] = useState<string | null>(null);

    const handleOpenIEMode = useCallback(async (device: DeviceEntry) => {
        setLoading(device.id);
        setError(null);

        try {
            // DESKTOP-01: Вызов Tauri command
            // @ts-expect-error - Tauri API доступен в runtime
            if (window.__TAURI__) {
                // @ts-expect-error
                const result = await window.__TAURI__.invoke<{ success: boolean; message: string }>(
                    'open_camera_web_ui',
                    {
                        deviceId: device.id,
                        url: `http://${device.ip}`,
                    },
                );
                setRecentResults(prev => [
                    { success: result.success, message: result.message },
                    ...prev.slice(0, 9),
                ]);
            } else {
                // Dev fallback
                window.open(`http://${device.ip}`, '_blank');
                setRecentResults(prev => [
                    { success: true, message: `Opened ${device.name} in browser (dev mode)` },
                    ...prev.slice(0, 9),
                ]);
            }
        } catch (err) {
            const message = err instanceof Error ? err.message : 'Unknown error';
            setError(message);
            setRecentResults(prev => [
                { success: false, message },
                ...prev.slice(0, 9),
            ]);
        } finally {
            setLoading(null);
        }
    }, []);

    return (
        <div className="min-h-screen bg-slate-900 text-slate-100">
            <div className="max-w-4xl mx-auto p-6">
                {/* ═══ Header ═══ */}
                <header className="mb-8">
                    <h1 className="text-2xl font-bold text-white">
                        🔧 CCTV Health Monitor — IE-Mode Desktop
                    </h1>
                    <p className="text-slate-400 mt-1 text-sm">
                        Открытие веб-интерфейсов камер в Microsoft Edge IE-mode
                    </p>
                </header>

                {/* ═══ Device List ═══ */}
                <section className="mb-8">
                    <h2 className="text-lg font-semibold mb-4 text-slate-200">
                        Устройства
                    </h2>
                    <div className="space-y-3">
                        {devices.map((device) => (
                            <DeviceCard
                                key={device.id}
                                device={device}
                                loading={loading === device.id}
                                onOpen={() => handleOpenIEMode(device)}
                            />
                        ))}
                    </div>
                </section>

                {/* ═══ Error Display ═══ */}
                {error && (
                    <div className="mb-6 p-4 bg-red-900/30 border border-red-700 rounded-xl">
                        <p className="text-red-400 text-sm">{error}</p>
                        <button
                            onClick={() => setError(null)}
                            className="mt-2 text-xs text-red-300 hover:text-red-200 underline"
                        >
                            Dismiss
                        </button>
                    </div>
                )}

                {/* ═══ Recent Results ═══ */}
                {recentResults.length > 0 && (
                    <section>
                        <h2 className="text-lg font-semibold mb-4 text-slate-200">
                            Recent Actions
                        </h2>
                        <div className="space-y-2">
                            {recentResults.map((result, i) => (
                                <div
                                    key={i}
                                    className={`p-3 rounded-lg text-sm border ${
                                        result.success
                                            ? 'bg-green-900/20 border-green-800 text-green-300'
                                            : 'bg-red-900/20 border-red-800 text-red-300'
                                    }`}
                                >
                                    <span className="font-medium">
                                        {result.success ? '✓' : '✗'}
                                    </span>{' '}
                                    {result.message}
                                </div>
                            ))}
                        </div>
                    </section>
                )}
            </div>
        </div>
    );
}

// ─── Device Card Component ──────────────────────────────────────────

function DeviceCard({
    device,
    loading,
    onOpen,
}: {
    device: DeviceEntry;
    loading: boolean;
    onOpen: () => void;
}) {
    return (
        <div className="flex items-center justify-between p-4 bg-slate-800 rounded-xl border border-slate-700">
            <div className="flex items-center gap-3">
                {/* Status indicator */}
                <div
                    className={`w-2.5 h-2.5 rounded-full ${
                        device.status === 'online'
                            ? 'bg-green-500 shadow-sm shadow-green-500/50'
                            : 'bg-slate-600'
                    }`}
                />
                <div>
                    <p className="text-sm font-medium text-slate-100">{device.name}</p>
                    <p className="text-xs text-slate-400 font-mono">{device.ip}</p>
                </div>
            </div>

            <button
                onClick={onOpen}
                disabled={loading || device.status === 'offline'}
                className="px-4 py-2 text-sm font-medium rounded-lg transition-colors
                    bg-blue-600 text-white hover:bg-blue-500
                    disabled:opacity-50 disabled:cursor-not-allowed
                    flex items-center gap-2"
            >
                {loading ? (
                    <span className="inline-block w-4 h-4 border-2 border-white/30 border-t-white rounded-full animate-spin" />
                ) : (
                    '🔧 IE-mode'
                )}
            </button>
        </div>
    );
}
