// ═══════════════════════════════════════════════════════════════════════
// DeviceActions — Edge Device Action Buttons (PROXY-04)
//
// React компонент с кнопками действий для устройства:
//   - "🌐 Web UI" — открывает новую вкладку с HTTP прокси
//   - "🔧 Terminal" — модальное окно с EdgeTerminal (SSH)
//   - "📹 Live View" — модальное окно с EdgeVideoPlayer (HLS/RTSP)
//   - "📁 Files" — модальное окно с EdgeFileManager (SFTP)
//
// Автоматическое создание VPN-сессии при клике (Lazy VPN — PROXY-03).
//
// Соответствие:
//   - IEC 62443-3-3 SL-3: Zone separation
//   - OWASP ASVS L3 V5: Input validation
//   - ISO 27001 A.12.4: Audit trail
// ═══════════════════════════════════════════════════════════════════════

import React, { useState, useCallback } from 'react';
import { Modal } from './ui/Modal';
import { EdgeTerminal } from './EdgeTerminal';
import { EdgeVideoPlayer } from './EdgeVideoPlayer';
import { EdgeFileManager } from './EdgeFileManager';
import { edgeProxyApi } from '../services/api/edgeProxy';

// ─── Types ──────────────────────────────────────────────────────────

type ModalType = 'terminal' | 'video' | 'files' | null;

interface DeviceActionsProps {
  /** ID edge-агента */
  agentId: string;
  /** IP устройства в LAN агента */
  deviceIp: string;
  /** HTTP порт Web UI устройства */
  httpPort?: number;
  /** SSH порт */
  sshPort?: number;
  /** RTSP порт */
  rtspPort?: number;
  /** HLS путь */
  hlsPath?: string;
  /** Имя пользователя SSH для терминала/SFTP */
  sshUsername?: string;
  /** Пароль SSH */
  sshPassword?: string;
  /** SSH username для файлового менеджера */
  fileUsername?: string;
  /** SSH password для файлового менеджера */
  filePassword?: string;
  /** Отключить автоматическое создание VPN */
  disableAutoVPN?: boolean;
  /** Колбэк при ошибке */
  onError?: (error: string) => void;
}

// ═══ Action Button Config ═══════════════════════════════════════════

interface ActionButton {
  id: string;
  label: string;
  icon: string;
  description: string;
  modalType: ModalType | 'webui';
  color: string;
  hoverColor: string;
}

const defaultActions: ActionButton[] = [
  {
    id: 'webui',
    label: 'Web UI',
    icon: '🌐',
    description: 'Open device web interface',
    modalType: 'webui',
    color: 'bg-blue-600',
    hoverColor: 'hover:bg-blue-700',
  },
  {
    id: 'terminal',
    label: 'Terminal',
    icon: '🔧',
    description: 'SSH terminal access',
    modalType: 'terminal',
    color: 'bg-emerald-600',
    hoverColor: 'hover:bg-emerald-700',
  },
  {
    id: 'video',
    label: 'Live View',
    icon: '📹',
    description: 'View camera stream',
    modalType: 'video',
    color: 'bg-purple-600',
    hoverColor: 'hover:bg-purple-700',
  },
  {
    id: 'files',
    label: 'Files',
    icon: '📁',
    description: 'Browse device files',
    modalType: 'files',
    color: 'bg-amber-600',
    hoverColor: 'hover:bg-amber-700',
  },
];

export function DeviceActions({
  agentId,
  deviceIp,
  httpPort = 80,
  sshPort = 22,
  rtspPort = 554,
  hlsPath = '/hls/stream.m3u8',
  sshUsername = 'root',
  sshPassword = '',
  fileUsername,
  filePassword,
  disableAutoVPN = false,
  onError,
}: DeviceActionsProps) {
  const [activeModal, setActiveModal] = useState<ModalType>(null);
  const [creatingVPN, setCreatingVPN] = useState(false);
  const [vpnReady, setVPNReady] = useState(disableAutoVPN);

  // Автоматическое создание VPN сессии (PROXY-03)
  const ensureVPNSession = useCallback(async () => {
    if (vpnReady || disableAutoVPN) return true;

    setCreatingVPN(true);
    try {
      // Разрешённые IP для сессии
      const allowedIPs = [
        `${deviceIp}/32`,
        '192.168.0.0/16',
      ];

      await edgeProxyApi.ensureVPNSession(agentId, allowedIPs);
      setVPNReady(true);
      return true;
    } catch (err) {
      const msg = err instanceof Error ? err.message : 'Failed to create VPN session';
      onError?.(msg);
      return false;
    } finally {
      setCreatingVPN(false);
    }
  }, [agentId, deviceIp, vpnReady, disableAutoVPN, onError]);

  // Обработчик для Web UI (открывает новую вкладку)
  const handleWebUI = useCallback(async () => {
    const sessionReady = await ensureVPNSession();
    if (!sessionReady) return;

    const proxyUrl = edgeProxyApi.getProxyUrl(agentId, deviceIp, httpPort);
    window.open(proxyUrl, '_blank', 'noopener,noreferrer');
  }, [agentId, deviceIp, httpPort, ensureVPNSession]);

  // Обработчик для модальных окон
  const handleOpenModal = useCallback(async (type: ModalType) => {
    if (!type) return;

    const sessionReady = await ensureVPNSession();
    if (!sessionReady) return;

    setActiveModal(type);
  }, [ensureVPNSession]);

  // Закрытие модального окна
  const handleCloseModal = useCallback(() => {
    setActiveModal(null);
  }, []);

  // Рендер модального окна
  const renderModal = () => {
    switch (activeModal) {
      case 'terminal':
        return (
          <Modal
            isOpen={true}
            onClose={handleCloseModal}
            title={`Terminal — ${deviceIp}:${sshPort}`}
            size="xl"
          >
            <EdgeTerminal
              agentId={agentId}
              deviceIp={deviceIp}
              port={sshPort}
              username={sshUsername}
              password={sshPassword}
              height={500}
              onError={onError}
              onDisconnect={() => {}}
            />
          </Modal>
        );

      case 'video':
        return (
          <Modal
            isOpen={true}
            onClose={handleCloseModal}
            title={`Live View — ${deviceIp}`}
            size="xl"
          >
            <EdgeVideoPlayer
              agentId={agentId}
              cameraIp={deviceIp}
              httpPort={httpPort}
              rtspPort={rtspPort}
              hlsPath={hlsPath}
              height={480}
              onError={onError}
            />
          </Modal>
        );

      case 'files':
        return (
          <Modal
            isOpen={true}
            onClose={handleCloseModal}
            title={`File Manager — ${deviceIp}`}
            size="xl"
          >
            <EdgeFileManager
              agentId={agentId}
              deviceIp={deviceIp}
              port={sshPort}
              username={fileUsername || sshUsername}
              password={filePassword || sshPassword}
              onError={onError}
            />
          </Modal>
        );

      default:
        return null;
    }
  };

  return (
    <div className="device-actions">
      {/* Action buttons grid */}
      <div className="grid grid-cols-2 sm:grid-cols-4 gap-3">
        {defaultActions.map((action) => (
          <button
            key={action.id}
            onClick={() => {
              if (action.id === 'webui') {
                handleWebUI();
              } else {
                handleOpenModal(action.modalType as ModalType);
              }
            }}
            disabled={creatingVPN}
            className={`
              flex flex-col items-center gap-2 p-4 rounded-xl
              ${action.color} ${action.hoverColor}
              text-white transition-transform duration-200
              disabled:opacity-50 disabled:cursor-not-allowed
              active:scale-[0.96] hover:shadow-lg
              focus:outline-none focus:ring-2 focus:ring-white/30
            `}
            title={action.description}
          >
            <span className="text-2xl">{action.icon}</span>
            <span className="text-sm font-medium">{action.label}</span>
          </button>
        ))}
      </div>

      {/* VPN status indicator */}
      {creatingVPN && (
        <div className="mt-3 flex items-center gap-2 px-3 py-2 bg-blue-900/30 border border-blue-800 rounded-lg">
          <div className="w-3 h-3 border-2 border-blue-500 border-t-transparent rounded-full animate-spin" />
          <span className="text-xs text-blue-300">Creating secure tunnel...</span>
        </div>
      )}

      {vpnReady && !creatingVPN && (
        <div className="mt-3 flex items-center gap-2 px-3 py-2 bg-green-900/30 border border-green-800 rounded-lg">
          <div className="w-2 h-2 rounded-full bg-green-500" />
          <span className="text-xs text-green-300">Secure tunnel active</span>
        </div>
      )}

      {/* Modal */}
      {renderModal()}
    </div>
  );
}

export default DeviceActions;
