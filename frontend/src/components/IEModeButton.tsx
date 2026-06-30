// ═══════════════════════════════════════════════════════════════════════
// IEModeButton (DESKTOP-03)
//
// Кнопка "🔧 Открыть в IE-mode" для desktop-версии (Tauri).
// Отображается только в Tauri desktop окружении.
// Вызывает Tauri command для открытия Edge в IE-mode и
// автоматической авторизации на камере.
//
// Соответствие:
//   - IEC 62443-3-3 SL-3: Zone separation
//   - OWASP ASVS L3 V3.3: Access control
//   - ISO 27001 A.12.4: Audit trail
// ═══════════════════════════════════════════════════════════════════════

import React, { useState, useEffect } from 'react';
import { Button } from './ui/Button';
import { Globe, ExternalLink, AlertTriangle } from 'lucide-react';

// ─── Types ──────────────────────────────────────────────────────────

interface IEModeButtonProps {
    deviceId: string;
    deviceName?: string;
    deviceIp?: string;
    /** URL веб-интерфейса камеры */
    cameraUrl?: string;
    /** Показывать всегда (для отладки) */
    forceShow?: boolean;
    onError?: (error: string) => void;
}

// ─── Desktop Detection ──────────────────────────────────────────────

/**
 * Проверяет, запущено ли приложение в Tauri desktop окружении.
 * Использует window.__TAURI__ для определения.
 */
function isTauriDesktop(): boolean {
    return typeof window !== 'undefined' && 
        (window as unknown as Record<string, unknown>).__TAURI__ !== undefined;
}

// ─── Main Component ─────────────────────────────────────────────────

export function IEModeButton({
    deviceId,
    deviceName,
    deviceIp,
    cameraUrl,
    forceShow = false,
    onError,
}: IEModeButtonProps) {
    const [isDesktop, setIsDesktop] = useState(false);
    const [loading, setLoading] = useState(false);
    const [error, setError] = useState<string | null>(null);

    useEffect(() => {
        setIsDesktop(isTauriDesktop() || forceShow);
    }, [forceShow]);

    if (!isDesktop) {
        return null;
    }

    const handleOpenIEMode = async () => {
        setLoading(true);
        setError(null);

        try {
            const targetUrl = cameraUrl || (deviceIp ? `http://${deviceIp}` : '');

            if (!targetUrl) {
                throw new Error('Camera URL or IP is required');
            }

            // DESKTOP-03: Вызов Tauri command
            // @ts-expect-error - Tauri API доступен только в desktop
            if (window.__TAURI__) {
                // DESKTOP-01: invoke Tauri command для открытия в IE-mode
                // @ts-expect-error - Tauri invoke
                await window.__TAURI__.invoke('open_camera_web_ui', {
                    deviceId,
                    url: targetUrl,
                });
            } else {
                // Dev fallback: открываем в новой вкладке
                window.open(targetUrl, '_blank');
            }
        } catch (err) {
            const message = err instanceof Error ? err.message : 'Failed to open in IE-mode';
            setError(message);
            onError?.(message);
        } finally {
            setLoading(false);
        }
    };

    return (
        <div className="inline-flex flex-col gap-1">
            <Button
                variant="outline"
                size="sm"
                loading={loading}
                icon={<Globe className="w-4 h-4" />}
                onClick={handleOpenIEMode}
                className="border-amber-300 text-amber-700 hover:bg-amber-50
                    dark:border-amber-700 dark:text-amber-400 dark:hover:bg-amber-900/20"
                title={`Open ${deviceName || deviceId} in IE-mode`}
            >
                🔧 IE-mode
            </Button>

            {error && (
                <div className="flex items-center gap-1.5 text-xs text-red-600 dark:text-red-400">
                    <AlertTriangle className="w-3 h-3" />
                    <span>{error}</span>
                </div>
            )}
        </div>
    );
}
