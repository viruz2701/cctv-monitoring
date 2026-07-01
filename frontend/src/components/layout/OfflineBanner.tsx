// OfflineBanner — persistent banner для offline режима.
//
// P1-2.3: "Offline mode. 3 operations queued."
// - Показывается при потере соединения
// - Счётчик операций в очереди
// - Анимация при появлении/исчезновении

import React, { useEffect, useState } from 'react';
import { useTranslation } from 'react-i18next';
import { WifiOff, Wifi } from '../ui/Icons';

interface OfflineBannerProps {
    queueCount?: number;
}

export function OfflineBanner({ queueCount = 0 }: OfflineBannerProps) {
    const { t } = useTranslation();
    const [isOnline, setIsOnline] = useState(navigator.onLine);

    useEffect(() => {
        const handleOnline = () => setIsOnline(true);
        const handleOffline = () => setIsOnline(false);
        window.addEventListener('online', handleOnline);
        window.addEventListener('offline', handleOffline);
        return () => {
            window.removeEventListener('online', handleOnline);
            window.removeEventListener('offline', handleOffline);
        };
    }, []);

    if (isOnline && queueCount === 0) return null;

    return (
        <div
            role="alert"
            aria-live="assertive"
            className={`fixed top-0 left-0 right-0 z-50 transition-[transform,opacity] duration-500 ${
                isOnline ? 'bg-emerald-600' : 'bg-amber-600'
            } text-white`}
        >
            <div className="flex items-center justify-center gap-2 px-4 py-2 text-sm font-medium">
                {isOnline ? (
                    <>
                        <Wifi className="w-4 h-4" />
                        <span>
                            {t('online_syncing') || 'Back online — syncing...'}
                        </span>
                        {queueCount > 0 && (
                            <span className="ml-1 px-2 py-0.5 bg-white/20 rounded-full text-xs">
                                {queueCount} {t('operations_left') || 'remaining'}
                            </span>
                        )}
                    </>
                ) : (
                    <>
                        <WifiOff className="w-4 h-4" />
                        <span>
                            {t('offline_mode') || 'Offline mode'}
                        </span>
                        {queueCount > 0 && (
                            <span className="ml-1 px-2 py-0.5 bg-white/20 rounded-full text-xs">
                                {queueCount} {t('operations_queued') || 'queued'}
                            </span>
                        )}
                    </>
                )}
            </div>
        </div>
    );
}
