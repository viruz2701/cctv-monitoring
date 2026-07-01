// ═══════════════════════════════════════════════════════════════════════
// EdgeVideoPlayer — HLS/RTSP Video Player (PROXY-04)
//
// React компонент для просмотра видеопотока с камеры через Edge Proxy.
// Использует HLS.js для воспроизведения HLS-потоков и проксирует RTSP.
//
// Flow:
//  1. Получение VPN сессии (Lazy VPN)
//  2. Прокси HLS/RTSP потока через Edge HTTP Proxy
//  3. Воспроизведение через HLS.js
//
// Соответствие:
//   - IEC 62443-3-3 SL-3: Zone separation
//   - OWASP ASVS L3 V5: Input validation
// ═══════════════════════════════════════════════════════════════════════

import React, { useEffect, useRef, useState, useCallback } from 'react';

interface EdgeVideoPlayerProps {
  /** ID edge-агента */
  agentId: string;
  /** IP камеры в LAN агента */
  cameraIp: string;
  /** RTSP порт (по умолчанию 554) */
  rtspPort?: number;
  /** HTTP порт для HLS (по умолчанию 80) */
  httpPort?: number;
  /** Путь к HLS потоку (например, /hls/stream.m3u8) */
  hlsPath?: string;
  /** RTSP URL (если известен, например rtsp://camera:554/stream1) */
  rtspUrl?: string;
  /** Тип потока: 'hls' | 'rtsp' | 'auto' */
  streamType?: 'hls' | 'rtsp' | 'auto';
  /** Ширина плеера */
  width?: number | string;
  /** Высота плеера */
  height?: number | string;
  /** Колбэк при ошибке */
  onError?: (error: string) => void;
}

export function EdgeVideoPlayer({
  agentId,
  cameraIp,
  rtspPort = 554,
  httpPort = 80,
  hlsPath = '/hls/stream.m3u8',
  rtspUrl: externalRtspUrl,
  streamType = 'auto',
  width = '100%',
  height = 360,
  onError,
}: EdgeVideoPlayerProps) {
  const videoRef = useRef<HTMLVideoElement>(null);
  const hlsRef = useRef<any>(null);
  const [status, setStatus] = useState<'loading' | 'playing' | 'error' | 'idle'>('idle');
  const [errorMsg, setErrorMsg] = useState('');

  // Генерация прокси URL для HLS
  const getHlsProxyUrl = useCallback(() => {
    const basePath = `/api/v1/edge/proxy/${encodeURIComponent(agentId)}/${cameraIp}:${httpPort}`;
    return `${basePath}/${hlsPath.replace(/^\//, '')}`;
  }, [agentId, cameraIp, httpPort, hlsPath]);

  // Инициализация HLS.js плеера
  useEffect(() => {
    let hls: any = null;
    const video = videoRef.current;
    if (!video) return;

    const v = video!;

    async function initPlayer() {
      try {
        setStatus('loading');

        if (streamType === 'hls' || streamType === 'auto') {
          // Динамический импорт HLS.js
          const HlsModule = await import('hls.js');
          const Hls = HlsModule.default;

          if (Hls.isSupported()) {
            const proxyUrl = getHlsProxyUrl();

            hls = new (Hls as any)({
              enableWorker: true,
              lowLatencyMode: true,
              backBufferLength: 30,
              maxBufferLength: 30,
              manifestLoadingTimeOut: 10000,
            });

            hls.loadSource(proxyUrl);
            hls.attachMedia(v);
            hlsRef.current = hls;

            hls.on(Hls.Events.MANIFEST_PARSED, () => {
              v.play().catch(() => {
                // Autoplay может быть заблокирован браузером
                setStatus('idle');
              });
              setStatus('playing');
            });

            hls.on(Hls.Events.ERROR, (_event: any, data: any) => {
              if (data.fatal) {
                setStatus('error');
                setErrorMsg(`HLS error: ${data.type}`);
                onError?.(`HLS fatal error: ${data.type}`);
              }
            });
          } else if (v.canPlayType('application/vnd.apple.mpegurl')) {
            // Native HLS support (Safari)
            v.src = getHlsProxyUrl();
            v.addEventListener('loadedmetadata', () => {
              v.play().catch(() => {});
              setStatus('playing');
            });
          } else {
            // HLS not supported, try MSE
            setStatus('error');
            setErrorMsg('HLS is not supported in this browser');
            onError?.('HLS not supported');
          }
        } else {
          // RTSP — показываем заглушку (RTSP不能在浏览器中直接播放)
          setStatus('idle');
          setErrorMsg('RTSP playback requires transcoding to HLS');
          onError?.('RTSP requires transcoding');
        }
      } catch (err) {
        setStatus('error');
        const msg = err instanceof Error ? err.message : 'Failed to initialize player';
        setErrorMsg(msg);
        onError?.(msg);
      }
    }

    initPlayer();

    return () => {
      if (hls) {
        hls.destroy();
        hlsRef.current = null;
      }
    };
  }, [getHlsProxyUrl, streamType, onError]);

  return (
    <div className="edge-video-player relative overflow-hidden rounded-lg bg-black" style={{ width, height }}>
      <video
        ref={videoRef}
        className="w-full h-full object-contain"
        controls
        playsInline
        muted
        autoPlay={false}
      />

      {/* Status overlay */}
      {status === 'loading' && (
        <div className="absolute inset-0 flex items-center justify-center bg-black/60">
          <div className="flex flex-col items-center gap-3">
            <div className="w-8 h-8 border-2 border-blue-500 border-t-transparent rounded-full animate-spin" />
            <span className="text-sm text-slate-300">Connecting to camera...</span>
          </div>
        </div>
      )}

      {status === 'error' && (
        <div className="absolute inset-0 flex items-center justify-center bg-black/60">
          <div className="flex flex-col items-center gap-2 px-4 text-center">
            <svg className="w-10 h-10 text-red-500" fill="none" viewBox="0 0 24 24" stroke="currentColor">
              <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M12 9v2m0 4h.01m-6.938 4h13.856c1.54 0 2.502-1.667 1.732-2.5L13.732 4c-.77-.833-1.964-.833-2.732 0L4.082 16.5c-.77.833.192 2.5 1.732 2.5z" />
            </svg>
            <span className="text-sm text-slate-300">{errorMsg}</span>
          </div>
        </div>
      )}

      {status === 'idle' && !errorMsg && (
        <div className="absolute inset-0 flex items-center justify-center bg-black/60">
          <div className="flex flex-col items-center gap-2">
            <svg className="w-10 h-10 text-slate-500" fill="none" viewBox="0 0 24 24" stroke="currentColor">
              <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M15 10l4.553-2.276A1 1 0 0121 8.618v6.764a1 1 0 01-1.447.894L15 14M5 18h8a2 2 0 002-2V8a2 2 0 00-2-2H5a2 2 0 00-2 2v8a2 2 0 002 2z" />
            </svg>
            <span className="text-sm text-slate-400">Click play to start</span>
          </div>
        </div>
      )}
    </div>
  );
}

export default EdgeVideoPlayer;
