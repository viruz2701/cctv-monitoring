import React from 'react';
import { Modal } from './Modal';
import { MapPin, ExternalLink } from 'lucide-react';

interface MapModalProps {
    isOpen: boolean;
    onClose: () => void;
    latitude: number;
    longitude: number;
    title?: string;
}

/**
 * MapModal — открывает OpenStreetMap во всплывающем iframe.
 *
 * Использует OSM Export/Embed API:
 * https://www.openstreetmap.org/export/embed.html?bbox=...&marker=...
 *
 * CSP: требуется frame-src https://www.openstreetmap.org
 */
export function MapModal({ isOpen, onClose, latitude, longitude, title }: MapModalProps) {
    // Вычисляем bounding box с отступом ~0.01° (~1км) для масштаба
    const lat = Number(latitude) || 0;
    const lon = Number(longitude) || 0;
    const padding = 0.01;

    const embedUrl = React.useMemo(() => {
        if (!lat && !lon) return '';
        const bbox = `${lon - padding},${lat - padding},${lon + padding},${lat + padding}`;
        return `https://www.openstreetmap.org/export/embed.html?bbox=${encodeURIComponent(bbox)}&layer=mapnik&marker=${lat},${lon}`;
    }, [lat, lon]);

    const externalUrl = React.useMemo(() => {
        if (!lat && !lon) return '#';
        return `https://www.openstreetmap.org/?mlat=${lat}&mlon=${lon}#map=15/${lat}/${lon}`;
    }, [lat, lon]);

    return (
        <Modal isOpen={isOpen} onClose={onClose} title={title || 'Map'} size="xl" showClose>
            {embedUrl ? (
                <div className="flex flex-col gap-3">
                    <div className="relative w-full h-[60vh] rounded-lg overflow-hidden border border-slate-200 dark:border-slate-700">
                        <iframe
                            title="OpenStreetMap"
                            src={embedUrl}
                            width="100%"
                            height="100%"
                            style={{ border: 0 }}
                            allowFullScreen
                            loading="lazy"
                            referrerPolicy="no-referrer"
                            className="rounded-lg"
                        />
                    </div>
                    <div className="flex items-center justify-between text-xs text-slate-500 dark:text-slate-400">
                        <span className="flex items-center gap-1">
                            <MapPin className="w-3.5 h-3.5" />
                            {lat.toFixed(5)}, {lon.toFixed(5)}
                        </span>
                        <a
                            href={externalUrl}
                            target="_blank"
                            rel="noopener noreferrer"
                            className="flex items-center gap-1 text-blue-600 dark:text-blue-400 hover:underline"
                        >
                            <ExternalLink className="w-3.5 h-3.5" />
                            Open in new tab
                        </a>
                    </div>
                </div>
            ) : (
                <div className="py-8 text-center text-slate-400 text-sm">
                    No coordinates available
                </div>
            )}
        </Modal>
    );
}
