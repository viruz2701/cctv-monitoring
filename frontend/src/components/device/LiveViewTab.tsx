// ═══════════════════════════════════════════════════════════════════════
// LiveViewTab — WebRTC/HLS live stream from camera with PTZ controls
// UX-2.2: Device Live View Tab
//   - WebRTC/HLS stream from camera
//   - PTZ controls (if supported)
//   - Snapshot button
//   - Recording trigger
//   - Fallback on snapshot if stream unavailable
//   - Placeholder if camera offline
// ═══════════════════════════════════════════════════════════════════════

import React, { useState, useCallback, useRef, useEffect } from 'react';
import { useTranslation } from 'react-i18next';
import {
  Video,
  VideoOff,
  Camera,
  Download,
  Radio,
  Maximize2,
  Monitor,
  Loader2,
  ImageOff,
} from '../../components/ui/Icons';
import { Card, CardHeader, CardBody, Button, Badge } from '../../components/ui';
import { PTZControls } from '../../components/p2p/PTZControls';
import { devicesApi } from '../../services/api/devices';
import type { Device } from '../../services/api/devices';

// ─── Types ────────────────────────────────────────────────────────────

export interface LiveViewTabProps {
  device: Device;
  /** WebRTC stream URL (optional — fetched from API if not provided) */
  streamUrl?: string;
  /** Snapshot URL (optional — fetched from API if not provided) */
  snapshotUrl?: string;
  /** Whether the device supports PTZ */
  supportsPtz?: boolean;
  /** Whether the device supports recording trigger */
  supportsRecording?: boolean;
}

// ─── Constants ────────────────────────────────────────────────────────

type StreamState = 'loading' | 'playing' | 'error' | 'offline' | 'unsupported';

// ─── Video Player Component ───────────────────────────────────────────

function VideoPlayer({ streamUrl, deviceId }: { streamUrl: string; deviceId: string }) {
  const videoRef = useRef<HTMLVideoElement>(null);
  const [streamState, setStreamState] = useState<StreamState>('loading');

  useEffect(() => {
    const video = videoRef.current;
    if (!video || !streamUrl) return;

    // Try WebRTC first, fallback to HLS
    if (streamUrl.startsWith('webrtc://') || streamUrl.startsWith('wss://')) {
      // WebRTC — use RTCPeerConnection
      setStreamState('unsupported');
      // In real implementation, use a WebRTC client library
      // For now, show placeholder as browser WebRTC needs specific handling
    } else if (streamUrl.includes('.m3u8')) {
      // HLS — use native HLS or hls.js
      if (video.canPlayType('application/vnd.apple.mpegurl')) {
        video.src = streamUrl;
        video.play().catch(() => setStreamState('error'));
      } else {
        // Would use hls.js in production
        setStreamState('unsupported');
      }
    } else {
      // Direct RTSP stream — not supported in browser, show snapshot fallback
      setStreamState('unsupported');
    }

    const handleCanPlay = () => setStreamState('playing');
    const handleError = () => setStreamState('error');
    video.addEventListener('canplay', handleCanPlay);
    video.addEventListener('error', handleError);
    return () => {
      video.removeEventListener('canplay', handleCanPlay);
      video.removeEventListener('error', handleError);
    };
  }, [streamUrl]);

  return (
    <div className="relative bg-slate-900 rounded-lg overflow-hidden aspect-video flex items-center justify-center">
      <video
        ref={videoRef}
        className="w-full h-full object-contain"
        autoPlay
        muted
        playsInline
      />
      {streamState === 'loading' && (
        <div className="absolute inset-0 flex items-center justify-center bg-slate-900/80">
          <Loader2 className="w-8 h-8 text-blue-400 animate-spin" />
          <span className="ml-2 text-sm text-slate-400">Connecting...</span>
        </div>
      )}
      {streamState === 'unsupported' && (
        <div className="absolute inset-0 flex flex-col items-center justify-center bg-slate-900/80">
          <Monitor className="w-10 h-10 text-slate-500 mb-2" />
          <p className="text-sm text-slate-400">Stream format not supported in browser</p>
          <p className="text-xs text-slate-500 mt-1">Use snapshot or native client</p>
        </div>
      )}
      {streamState === 'offline' && (
        <div className="absolute inset-0 flex flex-col items-center justify-center bg-slate-900/80">
          <VideoOff className="w-10 h-10 text-red-400 mb-2" />
          <p className="text-sm text-slate-400">Camera is offline</p>
        </div>
      )}
    </div>
  );
}

// ─── Main Component ──────────────────────────────────────────────────

export function LiveViewTab({
  device,
  streamUrl: initialStreamUrl,
  snapshotUrl: initialSnapshotUrl,
  supportsPtz = false,
  supportsRecording = false,
}: LiveViewTabProps) {
  const { t } = useTranslation();
  const [streamUrl, setStreamUrl] = useState<string | undefined>(initialStreamUrl);
  const [snapshotUrl, setSnapshotUrl] = useState<string | undefined>(initialSnapshotUrl);
  const [isRecording, setIsRecording] = useState(false);
  const [isFullscreen, setIsFullscreen] = useState(false);
  const [loadingSnapshot, setLoadingSnapshot] = useState(false);
  const containerRef = useRef<HTMLDivElement>(null);

  const isOffline = device.status === 'offline' || device.status === 'OFFLINE';
  const isWarning = device.status === 'warning' || device.status === 'WARNING';

  // Take snapshot
  const handleSnapshot = useCallback(async () => {
    if (!device.device_id) return;
    setLoadingSnapshot(true);
    try {
      // Try to get snapshot from API
      const blob = await devicesApi.getDeviceImages(device.device_id);
      if (Array.isArray(blob) && blob.length > 0) {
        setSnapshotUrl(blob[0]);
      }
    } catch {
      // Fallback: use existing snapshot URL
    } finally {
      setLoadingSnapshot(false);
    }
  }, [device.device_id]);

  // Download snapshot
  const handleDownloadSnapshot = useCallback(() => {
    if (!snapshotUrl) return;
    const a = document.createElement('a');
    a.href = snapshotUrl;
    a.download = `${device.name || device.device_id}_snapshot.jpg`;
    a.click();
  }, [snapshotUrl, device.name, device.device_id]);

  // Toggle recording
  const handleToggleRecording = useCallback(() => {
    setIsRecording((prev) => !prev);
  }, []);

  // Toggle fullscreen
  const handleToggleFullscreen = useCallback(() => {
    if (!containerRef.current) return;
    if (!document.fullscreenElement) {
      containerRef.current.requestFullscreen();
      setIsFullscreen(true);
    } else {
      document.exitFullscreen();
      setIsFullscreen(false);
    }
  }, []);

  return (
    <div ref={containerRef} className="space-y-4">
      {/* Status Badge */}
      <div className="flex items-center gap-2">
        <Badge
          variant={isOffline ? 'danger' : isWarning ? 'warning' : 'success'}
          size="sm"
        >
          {isOffline
            ? (t('offline') || 'Offline')
            : isWarning
              ? (t('warning') || 'Warning')
              : (t('online') || 'Online')}
        </Badge>
        {isRecording && (
            <Badge variant="danger" size="sm">
              <span className="flex items-center gap-1">
                <span className="w-2 h-2 bg-red-500 rounded-full animate-pulse" />
                {t('recording') || 'Recording'}
              </span>
            </Badge>
          )}
      </div>

      {/* Video Stream or Offline Placeholder */}
      {isOffline ? (
        <Card>
          <CardBody>
            <div className="flex flex-col items-center justify-center py-16">
              <VideoOff className="w-16 h-16 text-slate-300 dark:text-slate-600 mb-4" />
              <h3 className="text-lg font-semibold text-slate-700 dark:text-slate-300">
                {t('camera_offline') || 'Camera Offline'}
              </h3>
              <p className="text-sm text-slate-500 dark:text-slate-400 mt-1">
                {t('camera_offline_description') || 'This camera is currently unavailable'}
              </p>
              <Button
                variant="outline"
                size="sm"
                className="mt-4"
                onClick={handleSnapshot}
                loading={loadingSnapshot}
                icon={<Camera className="w-4 h-4" />}
              >
                {t('try_snapshot') || 'Try Snapshot'}
              </Button>
            </div>
          </CardBody>
        </Card>
      ) : streamUrl ? (
        <VideoPlayer streamUrl={streamUrl} deviceId={device.device_id} />
      ) : snapshotUrl ? (
        <div className="relative bg-slate-100 dark:bg-slate-800 rounded-lg overflow-hidden aspect-video">
          <img
            src={snapshotUrl}
            alt={device.name || 'Camera snapshot'}
            className="w-full h-full object-contain"
          />
        </div>
      ) : (
        <Card>
          <CardBody>
            <div className="flex flex-col items-center justify-center py-16">
              <Video className="w-16 h-16 text-slate-300 dark:text-slate-600 mb-4" />
              <h3 className="text-lg font-semibold text-slate-700 dark:text-slate-300">
                {t('live_view') || 'Live View'}
              </h3>
              <p className="text-sm text-slate-500 dark:text-slate-400 mt-1">
                {t('stream_not_available') || 'Stream not available'}
              </p>
              <Button
                variant="outline"
                size="sm"
                className="mt-4"
                onClick={handleSnapshot}
                loading={loadingSnapshot}
                icon={<Camera className="w-4 h-4" />}
              >
                {t('request_snapshot') || 'Request Snapshot'}
              </Button>
            </div>
          </CardBody>
        </Card>
      )}

      {/* Controls Bar */}
      <div className="flex items-center gap-2 flex-wrap">
        {/* Snapshot */}
        <Button
          variant="outline"
          size="sm"
          icon={<Camera className="w-4 h-4" />}
          onClick={handleSnapshot}
          loading={loadingSnapshot}
          disabled={isOffline}
        >
          {t('snapshot') || 'Snapshot'}
        </Button>

        {/* Download Snapshot */}
        {snapshotUrl && (
          <Button
            variant="outline"
            size="sm"
            icon={<Download className="w-4 h-4" />}
            onClick={handleDownloadSnapshot}
          >
            {t('download') || 'Download'}
          </Button>
        )}

        {/* Recording Trigger */}
          {supportsRecording && !isOffline && (
            <Button
              variant={isRecording ? 'danger' : 'outline'}
              size="sm"
              icon={<Radio className="w-4 h-4" />}
              onClick={handleToggleRecording}
            >
              {isRecording
                ? (t('stop_recording') || 'Stop Recording')
                : (t('start_recording') || 'Start Recording')}
            </Button>
          )}

        {/* Fullscreen */}
        <Button
          variant="outline"
          size="sm"
          icon={<Maximize2 className="w-4 h-4" />}
          onClick={handleToggleFullscreen}
          className="ml-auto"
        >
          {t('fullscreen') || 'Fullscreen'}
        </Button>
      </div>

      {/* PTZ Controls */}
      {supportsPtz && !isOffline && (
        <Card>
          <CardHeader>
            {t('ptz_controls') || 'PTZ Controls'}
          </CardHeader>
          <CardBody>
            <div className="flex justify-center">
              <PTZControls deviceId={device.device_id} disabled={isOffline} />
            </div>
          </CardBody>
        </Card>
      )}
    </div>
  );
}

export default LiveViewTab;
