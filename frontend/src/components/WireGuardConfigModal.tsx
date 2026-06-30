// ═══════════════════════════════════════════════════════════════════════
// WireGuardConfigModal (SELFSERV-03)
//
// Модальное окно для self-service скачивания WireGuard конфигурации.
// Отображает wg-quick конфиг, QR-код для мобильного клиента,
// кнопки копирования/скачивания и пошаговую инструкцию.
//
// Соответствие:
//   - IEC 62443-3-3 SL-3: Zone separation
//   - OWASP ASVS L3 V2.1: Session management
//   - Приказ ОАЦ №66 п. 7.18.2: Управление удалённым доступом
// ═══════════════════════════════════════════════════════════════════════

import React, { useState, useCallback } from 'react';
import { QRCodeSVG } from 'qrcode.react';
import { Modal } from './ui/Modal';
import {
    Download,
    Copy,
    Check,
    Smartphone,
    Monitor,
    Globe,
    Wifi,
    HelpCircle,
    ChevronDown,
    ChevronUp,
} from 'lucide-react';

// ─── Types ──────────────────────────────────────────────────────────

interface WireGuardConfigModalProps {
    isOpen: boolean;
    onClose: () => void;
    sessionId: string;
    configText: string;
    onDownload: () => void;
}

type OSType = 'windows' | 'macos' | 'linux' | 'android' | 'ios';

interface OSInstructions {
    title: string;
    icon: React.ReactNode;
    steps: string[];
}

// ─── Instructions ───────────────────────────────────────────────────

function getOSInstructions(): Record<OSType, OSInstructions> {
    return {
        windows: {
            title: 'Windows',
            icon: <Monitor className="w-5 h-5" />,
            steps: [
                'Скачайте и установите WireGuard с сайта wireguard.com/install',
                'Откройте WireGuard и нажмите "Import tunnel(s) from file"',
                'Выберите скачанный .conf файл',
                'Нажмите "Activate" для подключения',
                'Проверьте статус — должен быть "Connected"',
            ],
        },
        macos: {
            title: 'macOS',
            icon: <Monitor className="w-5 h-5" />,
            steps: [
                'Скачайте WireGuard из App Store',
                'Откройте WireGuard и нажмите "Import tunnel(s) from file"',
                'Выберите скачанный .conf файл',
                'Нажмите "Activate" для подключения',
            ],
        },
        linux: {
            title: 'Linux',
            icon: <Globe className="w-5 h-5" />,
            steps: [
                'Установите WireGuard: sudo apt install wireguard',
                'Сохраните конфиг как /etc/wireguard/cctv.conf',
                'Подключитесь: sudo wg-quick up cctv',
                'Для отключения: sudo wg-quick down cctv',
                'Опционально: sudo systemctl enable wg-quick@cctv',
            ],
        },
        android: {
            title: 'Android',
            icon: <Smartphone className="w-5 h-5" />,
            steps: [
                'Установите WireGuard из Google Play',
                'Откройте приложение и нажмите "+"',
                'Выберите "Scan from QR code"',
                'Отсканируйте QR-код ниже',
                'Нажмите "Activate" для подключения',
            ],
        },
        ios: {
            title: 'iOS',
            icon: <Smartphone className="w-5 h-5" />,
            steps: [
                'Установите WireGuard из App Store',
                'Откройте приложение и нажмите "+"',
                'Выберите "Create from QR code"',
                'Отсканируйте QR-код ниже',
                'Нажмите "Activate" для подключения',
            ],
        },
    };
}

// ─── OS Selector Tab ────────────────────────────────────────────────

function OSTab({
    os,
    isActive,
    onClick,
}: {
    os: { key: OSType; title: string; icon: React.ReactNode };
    isActive: boolean;
    onClick: () => void;
}) {
    return (
        <button
            onClick={onClick}
            className={`
                flex items-center gap-2 px-3 py-2 text-sm rounded-lg transition-colors
                ${isActive
                    ? 'bg-blue-100 text-blue-700 dark:bg-blue-900/40 dark:text-blue-300'
                    : 'text-slate-600 hover:bg-slate-100 dark:text-slate-400 dark:hover:bg-slate-800'
                }
            `}
        >
            {os.icon}
            <span className="hidden sm:inline">{os.title}</span>
        </button>
    );
}

// ─── Main Component ─────────────────────────────────────────────────

export function WireGuardConfigModal({
    isOpen,
    onClose,
    sessionId,
    configText,
    onDownload,
}: WireGuardConfigModalProps) {
    const [copied, setCopied] = useState(false);
    const [selectedOS, setSelectedOS] = useState<OSType>('android');
    const [showInstructions, setShowInstructions] = useState(false);
    const [showQR, setShowQR] = useState(true);

    const osList: { key: OSType; title: string; icon: React.ReactNode }[] = [
        { key: 'windows', title: 'Windows', icon: <Monitor className="w-4 h-4" /> },
        { key: 'macos', title: 'macOS', icon: <Monitor className="w-4 h-4" /> },
        { key: 'linux', title: 'Linux', icon: <Globe className="w-4 h-4" /> },
        { key: 'android', title: 'Android', icon: <Smartphone className="w-4 h-4" /> },
        { key: 'ios', title: 'iOS', icon: <Smartphone className="w-4 h-4" /> },
    ];

    const handleCopy = useCallback(async () => {
        try {
            await navigator.clipboard.writeText(configText);
            setCopied(true);
            setTimeout(() => setCopied(false), 2000);
        } catch {
            // Fallback for older browsers
            const textarea = document.createElement('textarea');
            textarea.value = configText;
            document.body.appendChild(textarea);
            textarea.select();
            document.execCommand('copy');
            document.body.removeChild(textarea);
            setCopied(true);
            setTimeout(() => setCopied(false), 2000);
        }
    }, [configText]);

    const instructions = getOSInstructions();
    const currentOS = instructions[selectedOS];

    // QR-код содержит WG quick-config в формате для мобильного клиента
    const qrValue = configText;

    const footer = (
        <div className="flex flex-wrap gap-3">
            <button
                onClick={handleCopy}
                className="inline-flex items-center gap-2 px-4 py-2 text-sm font-medium rounded-lg
                    bg-blue-600 text-white hover:bg-blue-700 active:bg-blue-800
                    transition-colors"
            >
                {copied ? (
                    <><Check className="w-4 h-4" /> Скопировано</>
                ) : (
                    <><Copy className="w-4 h-4" /> Скопировать</>
                )}
            </button>
            <button
                onClick={onDownload}
                className="inline-flex items-center gap-2 px-4 py-2 text-sm font-medium rounded-lg
                    bg-slate-100 text-slate-700 hover:bg-slate-200
                    dark:bg-slate-800 dark:text-slate-200 dark:hover:bg-slate-700
                    transition-colors"
            >
                <Download className="w-4 h-4" /> Скачать .conf
            </button>
        </div>
    );

    return (
        <Modal
            isOpen={isOpen}
            onClose={onClose}
            title="🔒 WireGuard VPN — Self-Service Config"
            size="xl"
            footer={footer}
        >
            <div className="space-y-6">
                {/* ═══ QR Code Section ═══ */}
                {showQR && (
                    <div className="flex flex-col items-center p-4 bg-slate-50 dark:bg-slate-900/50 rounded-xl">
                        <div className="bg-white dark:bg-slate-800 p-3 rounded-xl shadow-sm border border-slate-200 dark:border-slate-700">
                            <QRCodeSVG
                                value={qrValue}
                                size={180}
                                level="M"
                                includeMargin
                            />
                        </div>
                        <p className="mt-2 text-xs text-slate-500 dark:text-slate-400 text-center">
                            QR-код для мобильного WireGuard клиента
                        </p>
                        <button
                            onClick={() => setShowQR(false)}
                            className="mt-1 text-xs text-blue-600 hover:text-blue-700 dark:text-blue-400"
                        >
                            Скрыть QR-код
                        </button>
                    </div>
                )}

                {!showQR && (
                    <button
                        onClick={() => setShowQR(true)}
                        className="text-sm text-blue-600 hover:text-blue-700 dark:text-blue-400"
                    >
                        Показать QR-код
                    </button>
                )}

                {/* ═══ Config Preview ═══ */}
                <div>
                    <label className="block text-sm font-medium text-slate-700 dark:text-slate-300 mb-2">
                        WireGuard Configuration
                    </label>
                    <pre className="p-3 bg-slate-900 text-green-400 text-xs font-mono rounded-xl overflow-x-auto max-h-48 border border-slate-700">
                        {configText}
                    </pre>
                </div>

                {/* ═══ Instructions Section ═══ */}
                <div className="border-t border-slate-200 dark:border-slate-700 pt-4">
                    <button
                        onClick={() => setShowInstructions(!showInstructions)}
                        className="flex items-center justify-between w-full text-sm font-medium text-slate-700 dark:text-slate-300"
                    >
                        <span className="flex items-center gap-2">
                            <Wifi className="w-4 h-4" />
                            Инструкция по установке
                        </span>
                        {showInstructions ? (
                            <ChevronUp className="w-4 h-4" />
                        ) : (
                            <ChevronDown className="w-4 h-4" />
                        )}
                    </button>

                    {showInstructions && (
                        <div className="mt-4 space-y-4">
                            {/* OS Tabs */}
                            <div className="flex flex-wrap gap-2">
                                {osList.map((os) => (
                                    <OSTab
                                        key={os.key}
                                        os={os}
                                        isActive={selectedOS === os.key}
                                        onClick={() => setSelectedOS(os.key)}
                                    />
                                ))}
                            </div>

                            {/* Instructions Content */}
                            <div className="p-4 bg-slate-50 dark:bg-slate-900/50 rounded-xl">
                                <h4 className="text-sm font-semibold text-slate-800 dark:text-slate-200 mb-3 flex items-center gap-2">
                                    {currentOS.icon}
                                    {currentOS.title}
                                </h4>
                                <ol className="space-y-2">
                                    {currentOS.steps.map((step, index) => (
                                        <li
                                            key={index}
                                            className="flex items-start gap-2 text-sm text-slate-600 dark:text-slate-400"
                                        >
                                            <span className="flex-shrink-0 w-5 h-5 rounded-full bg-blue-100 dark:bg-blue-900/40 text-blue-600 dark:text-blue-400 flex items-center justify-center text-xs font-medium">
                                                {index + 1}
                                            </span>
                                            <span>{step}</span>
                                        </li>
                                    ))}
                                </ol>
                            </div>
                        </div>
                    )}
                </div>

                {/* ═══ Session Info ═══ */}
                <div className="border-t border-slate-200 dark:border-slate-700 pt-4">
                    <div className="flex items-center gap-2 text-xs text-slate-500 dark:text-slate-400">
                        <HelpCircle className="w-3 h-3" />
                        <span>
                            Session ID: {sessionId.slice(0, 8)}... | 
                            Config подписан HMAC для защиты целостности
                        </span>
                    </div>
                </div>
            </div>
        </Modal>
    );
}
