// ═══════════════════════════════════════════════════════════════════════
// EdgeTerminal — WebSocket SSH Terminal (PROXY-02)
//
// React компонент терминала на xterm.js для SSH доступа к устройству
// через WireGuard VPN прокси.
//
// Flow:
//  1. WebSocket подключение к /api/v1/edge/ssh/{agent_id}/{device_ip}/{port}
//  2. Отправка SSH credentials (username, password)
//  3. Двусторонний прокси: xterm.js ↔ WebSocket ↔ SSH
//  4. Resize handler для синхронизации размеров терминала
//
// Соответствие:
//   - IEC 62443-3-3 SL-3: Zone separation
//   - OWASP ASVS L3 V2: Authentication (JWT + SSH credentials)
// ═══════════════════════════════════════════════════════════════════════

import React, { useEffect, useRef, useCallback, useState } from 'react';

// Динамический импорт xterm.js для избежания проблем с SSR
// В production xterm.css должен быть импортирован в index.html или через import
// eslint-disable-next-line @typescript-eslint/no-explicit-any
type ITerminal = any;

interface EdgeTerminalProps {
  /** ID edge-агента */
  agentId: string;
  /** IP устройства в LAN агента */
  deviceIp: string;
  /** SSH порт (по умолчанию 22) */
  port?: number;
  /** URL WebSocket (если не указан, генерируется автоматически) */
  wsUrl?: string;
  /** Имя пользователя SSH */
  username?: string;
  /** Пароль SSH */
  password?: string;
  /** Высота контейнера в пикселях */
  height?: number;
  /** Колбэк при ошибке */
  onError?: (error: string) => void;
  /** Колбэк при закрытии соединения */
  onDisconnect?: () => void;
}

export function EdgeTerminal({
  agentId,
  deviceIp,
  port = 22,
  wsUrl: externalWsUrl,
  username = 'root',
  password = '',
  height = 400,
  onError,
  onDisconnect,
}: EdgeTerminalProps) {
  const terminalRef = useRef<HTMLDivElement>(null);
  const wsRef = useRef<WebSocket | null>(null);
  const terminalInstanceRef = useRef<ITerminal | null>(null);
  const [connected, setConnected] = useState(false);
  const [statusText, setStatusText] = useState('Connecting...');
  const authSentRef = useRef(false);

  // Функция для получения URL WebSocket
  const getWsUrl = useCallback(() => {
    if (externalWsUrl) return externalWsUrl;
    const protocol = window.location.protocol === 'https:' ? 'wss:' : 'ws:';
    const host = window.location.host;
    return `${protocol}//${host}/api/v1/edge/ssh/${encodeURIComponent(agentId)}/${deviceIp}/${port}`;
  }, [agentId, deviceIp, port, externalWsUrl]);

  // Инициализация терминала
  useEffect(() => {
    let term: any = null;
    let ws: WebSocket | null = null;
    let disposed = false;

    async function initTerminal() {
      try {
        // Динамический импорт xterm.js
        const { Terminal } = await import('@xterm/xterm');
        const { FitAddon } = await import('@xterm/addon-fit');

        // Импорт CSS (если не глобальный)
        try {
          await import('@xterm/xterm/css/xterm.css');
        } catch {
          // CSS может быть уже импортирован глобально
        }

        if (disposed || !terminalRef.current) return;

        term = new Terminal({
          cursorBlink: true,
          cursorStyle: 'block',
          fontSize: 14,
          fontFamily: "'JetBrains Mono', 'Fira Code', 'Cascadia Code', monospace",
          theme: {
            background: '#1a1b26',
            foreground: '#a9b1d6',
            cursor: '#c0caf5',
            selectionBackground: '#33467c',
            black: '#1d202f',
            red: '#f7768e',
            green: '#9ece6a',
            yellow: '#e0af68',
            blue: '#7aa2f7',
            magenta: '#bb9af7',
            cyan: '#7dcfff',
            white: '#a9b1d6',
            brightBlack: '#414868',
            brightRed: '#f7768e',
            brightGreen: '#9ece6a',
            brightYellow: '#e0af68',
            brightBlue: '#7aa2f7',
            brightMagenta: '#bb9af7',
            brightCyan: '#7dcfff',
            brightWhite: '#c0caf5',
          },
          allowTransparency: false,
          rows: Math.floor(height / 20),
          cols: Math.floor((terminalRef.current?.clientWidth || 800) / 9),
        });

        const fitAddon = new FitAddon();
        term.loadAddon(fitAddon);

        if (terminalRef.current) {
          term.open(terminalRef.current);
          fitAddon.fit();
        }

        terminalInstanceRef.current = term;
        setStatusText('Connecting WebSocket...');

        // Подключение WebSocket
        ws = new WebSocket(getWsUrl());
        wsRef.current = ws;

        ws.onopen = () => {
          if (disposed) return;
          // Отправляем SSH credentials
          const authMsg = JSON.stringify({
            type: 'auth',
            username,
            password,
          });
          ws?.send(authMsg);
          authSentRef.current = true;
        };

        ws.onmessage = (event) => {
          if (disposed) return;
          try {
            const msg = JSON.parse(event.data);

            switch (msg.type) {
              case 'connected':
                setConnected(true);
                setStatusText('Connected');
                term?.write('\r\n\x1b[32mSSH session established\x1b[0m\r\n');
                break;

              case 'output':
                if (msg.data) {
                  term?.write(msg.data);
                }
                break;

              case 'pong':
                // keepalive ответ
                break;

              default:
                if (typeof event.data === 'string') {
                  term?.write(event.data);
                }
            }
          } catch {
            // Бинарные данные — пишем напрямую
            if (typeof event.data === 'string') {
              term?.write(event.data);
            }
          }
        };

        ws.onclose = (event) => {
          if (disposed) return;
          setConnected(false);
          setStatusText('Disconnected');
          term?.write(`\r\n\x1b[31mConnection closed (code: ${event.code})\x1b[0m\r\n`);
          onDisconnect?.();
        };

        ws.onerror = () => {
          if (disposed) return;
          setStatusText('Connection error');
          onError?.('WebSocket connection failed');
        };

        // Обработка ввода с терминала
        term.onData((data: string) => {
          if (ws?.readyState === WebSocket.OPEN) {
            const inputMsg = JSON.stringify({
              type: 'input',
              data: JSON.stringify(data),
            });
            ws.send(inputMsg);
          }
        });

        // Обработка resize
        term.onResize((cols: number, rows: number) => {
          if (ws?.readyState === WebSocket.OPEN) {
            const resizeMsg = JSON.stringify({
              type: 'resize',
              size: { cols, rows },
            });
            ws.send(resizeMsg);
          }
        });

        // Keepalive ping
        const pingInterval = setInterval(() => {
          if (ws?.readyState === WebSocket.OPEN) {
            ws.send(JSON.stringify({ type: 'ping' }));
          }
        }, 30000);

        // Window resize handler
        const handleWindowResize = () => {
          try {
            fitAddon.fit();
          } catch {
            // ignore
          }
        };
        window.addEventListener('resize', handleWindowResize);

        // Cleanup interval and listener on dispose
        const originalDispose = term.dispose.bind(term);
        term.dispose = () => {
          clearInterval(pingInterval);
          window.removeEventListener('resize', handleWindowResize);
          originalDispose();
        };

      } catch (err) {
        if (!disposed) {
          console.error('Failed to initialize terminal:', err);
          setStatusText('Failed to initialize');
          onError?.(`Terminal init failed: ${err instanceof Error ? err.message : 'Unknown error'}`);
        }
      }
    }

    initTerminal();

    return () => {
      disposed = true;
      if (ws) {
        ws.close();
        wsRef.current = null;
      }
      if (term) {
        try {
          term.dispose();
        } catch {
          // ignore
        }
        terminalInstanceRef.current = null;
      }
    };
  }, [agentId, deviceIp, port, getWsUrl, username, password, height, onError, onDisconnect]);

  return (
    <div className="edge-terminal-container">
      {!connected && (
        <div className="absolute top-0 left-0 right-0 z-10 px-4 py-2 text-sm bg-slate-800 border-b border-slate-700 text-slate-300 flex items-center gap-2">
          <span className={`inline-block w-2 h-2 rounded-full ${connected ? 'bg-green-500' : 'bg-yellow-500 animate-pulse'}`} />
          <span>{statusText}</span>
          <span className="text-slate-500 ml-auto">
            {agentId} @ {deviceIp}:{port}
          </span>
        </div>
      )}
      <div
        ref={terminalRef}
        className="w-full rounded-lg overflow-hidden border border-slate-700 bg-[#1a1b26]"
        style={{ height: `${height}px`, marginTop: connected ? 0 : '36px' }}
      />
    </div>
  );
}

export default EdgeTerminal;
