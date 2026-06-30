// ═══════════════════════════════════════════════════════════════════════
// EdgeFileManager — SFTP File Manager (PROXY-04)
//
// React компонент для управления файлами на устройстве через SFTP
// через WireGuard VPN прокси.
//
// Flow:
//  1. Получение VPN сессии (Lazy VPN)
//  2. SFTP подключение через Edge SSH Proxy
//  3. Список файлов, upload, download
//
// Соответствие:
//   - IEC 62443-3-3 SL-3: Zone separation
//   - OWASP ASVS L3 V5: Input validation
// ═══════════════════════════════════════════════════════════════════════

import React, { useState, useCallback } from 'react';
import { edgeProxyApi } from '../services/api/edgeProxy';
import { request } from '../services/api/client';

// ─── Types ──────────────────────────────────────────────────────────

interface FileEntry {
  name: string;
  path: string;
  size: number;
  mode: string;
  modTime: string;
  isDir: boolean;
}

interface EdgeFileManagerProps {
  /** ID edge-агента */
  agentId: string;
  /** IP устройства в LAN агента */
  deviceIp: string;
  /** SSH порт */
  port?: number;
  /** Имя пользователя SSH */
  username?: string;
  /** Пароль SSH */
  password?: string;
  /** Начальная директория */
  initialPath?: string;
  /** Колбэк при ошибке */
  onError?: (error: string) => void;
}

export function EdgeFileManager({
  agentId,
  deviceIp,
  port = 22,
  username = 'root',
  password = '',
  initialPath = '/',
  onError,
}: EdgeFileManagerProps) {
  const [currentPath, setCurrentPath] = useState(initialPath);
  const [files, setFiles] = useState<FileEntry[]>([]);
  const [loading, setLoading] = useState(false);
  const [uploading, setUploading] = useState(false);
  const [error, setError] = useState('');
  const [selectedFile, setSelectedFile] = useState<string | null>(null);

  // Загрузка списка файлов
  const loadDirectory = useCallback(async (path: string) => {
    setLoading(true);
    setError('');
    try {
      // Используем HTTP прокси для SFTP-over-HTTP API
      const response = await request<{ files: FileEntry[]; path: string }>(`/edge/proxy/${encodeURIComponent(agentId)}/${deviceIp}:${port}/files`, {
        method: 'POST',
        body: JSON.stringify({ action: 'list', path, username, password }),
      });
      setFiles(response.files);
      setCurrentPath(response.path);
    } catch (err) {
      const msg = err instanceof Error ? err.message : 'Failed to load directory';
      setError(msg);
      onError?.(msg);
    } finally {
      setLoading(false);
    }
  }, [agentId, deviceIp, port, username, password, onError]);

  // Загрузка файла
  const handleUpload = useCallback(async (event: React.ChangeEvent<HTMLInputElement>) => {
    const file = event.target.files?.[0];
    if (!file) return;

    setUploading(true);
    setError('');
    try {
      const formData = new FormData();
      formData.append('file', file);
      formData.append('path', currentPath);
      formData.append('username', username);
      formData.append('password', password);

      await fetch(`/api/v1/edge/proxy/${encodeURIComponent(agentId)}/${deviceIp}:${port}/files/upload`, {
        method: 'POST',
        body: formData,
        credentials: 'include',
      });

      // Перезагружаем директорию
      await loadDirectory(currentPath);
    } catch (err) {
      const msg = err instanceof Error ? err.message : 'Failed to upload file';
      setError(msg);
      onError?.(msg);
    } finally {
      setUploading(false);
    }
  }, [agentId, deviceIp, port, currentPath, username, password, loadDirectory, onError]);

  // Скачивание файла
  const handleDownload = useCallback(async (filePath: string, fileName: string) => {
    try {
      const response = await fetch(`/api/v1/edge/proxy/${encodeURIComponent(agentId)}/${deviceIp}:${port}/files/download`, {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify({ path: filePath, username, password }),
        credentials: 'include',
      });

      if (!response.ok) throw new Error('Download failed');

      const blob = await response.blob();
      const url = URL.createObjectURL(blob);
      const a = document.createElement('a');
      a.href = url;
      a.download = fileName;
      a.click();
      URL.revokeObjectURL(url);
    } catch (err) {
      const msg = err instanceof Error ? err.message : 'Failed to download file';
      setError(msg);
      onError?.(msg);
    }
  }, [agentId, deviceIp, port, username, password, onError]);

  // Навигация по директориям
  const navigateTo = useCallback((path: string) => {
    loadDirectory(path);
  }, [loadDirectory]);

  // Форматирование размера файла
  const formatSize = (bytes: number): string => {
    if (bytes === 0) return '-';
    const units = ['B', 'KB', 'MB', 'GB'];
    const i = Math.floor(Math.log(bytes) / Math.log(1024));
    return `${(bytes / Math.pow(1024, i)).toFixed(1)} ${units[i]}`;
  };

  // Инициализация
  React.useEffect(() => {
    loadDirectory(initialPath);
  }, [initialPath, loadDirectory]);

  return (
    <div className="edge-file-manager rounded-lg border border-slate-700 bg-slate-900 overflow-hidden">
      {/* Header */}
      <div className="px-4 py-3 bg-slate-800 border-b border-slate-700 flex items-center gap-2">
        <svg className="w-4 h-4 text-slate-400" fill="none" viewBox="0 0 24 24" stroke="currentColor">
          <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M3 7v10a2 2 0 002 2h14a2 2 0 002-2V9a2 2 0 00-2-2h-6l-2-2H5a2 2 0 00-2 2z" />
        </svg>
        <span className="text-sm text-slate-300 font-mono truncate">{currentPath}</span>

        <div className="ml-auto flex items-center gap-2">
          {/* Upload button */}
          <label className="cursor-pointer px-3 py-1.5 text-xs font-medium bg-blue-600 hover:bg-blue-700 text-white rounded-md transition-colors">
            Upload
            <input
              type="file"
              className="hidden"
              onChange={handleUpload}
              disabled={uploading}
            />
          </label>

          {/* Refresh button */}
          <button
            onClick={() => loadDirectory(currentPath)}
            className="px-3 py-1.5 text-xs font-medium bg-slate-700 hover:bg-slate-600 text-slate-200 rounded-md transition-colors"
            disabled={loading}
          >
            Refresh
          </button>
        </div>
      </div>

      {/* Error */}
      {error && (
        <div className="px-4 py-2 bg-red-900/30 border-b border-red-800 text-sm text-red-400">
          {error}
        </div>
      )}

      {/* Loading */}
      {loading && (
        <div className="flex items-center justify-center py-12">
          <div className="w-6 h-6 border-2 border-blue-500 border-t-transparent rounded-full animate-spin" />
        </div>
      )}

      {/* File list */}
      {!loading && (
        <div className="divide-y divide-slate-800">
          {/* Parent directory */}
          {currentPath !== '/' && (
            <button
              onClick={() => navigateTo(currentPath.split('/').slice(0, -1).join('/') || '/')}
              className="w-full flex items-center gap-3 px-4 py-2.5 hover:bg-slate-800/50 transition-colors text-left"
            >
              <svg className="w-4 h-4 text-slate-500" fill="none" viewBox="0 0 24 24" stroke="currentColor">
                <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M15 19l-7-7 7-7" />
              </svg>
              <span className="text-sm text-slate-400">..</span>
            </button>
          )}

          {files.map((file) => (
            <div
              key={file.path}
              className={`flex items-center gap-3 px-4 py-2.5 hover:bg-slate-800/50 transition-colors ${selectedFile === file.path ? 'bg-slate-800' : ''}`}
              onClick={() => setSelectedFile(file.path)}
              onDoubleClick={() => file.isDir && navigateTo(file.path)}
            >
              {/* Icon */}
              {file.isDir ? (
                <svg className="w-4 h-4 text-amber-500 shrink-0" fill="none" viewBox="0 0 24 24" stroke="currentColor">
                  <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M3 7v10a2 2 0 002 2h14a2 2 0 002-2V9a2 2 0 00-2-2h-6l-2-2H5a2 2 0 00-2 2z" />
                </svg>
              ) : (
                <svg className="w-4 h-4 text-slate-500 shrink-0" fill="none" viewBox="0 0 24 24" stroke="currentColor">
                  <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M7 21h10a2 2 0 002-2V9.414a1 1 0 00-.293-.707l-5.414-5.414A1 1 0 0012.586 3H7a2 2 0 00-2 2v14a2 2 0 002 2z" />
                </svg>
              )}

              {/* Name */}
              <span
                className={`text-sm flex-1 truncate ${file.isDir ? 'text-slate-200 font-medium cursor-pointer' : 'text-slate-300'}`}
                onClick={() => file.isDir && navigateTo(file.path)}
              >
                {file.name}
              </span>

              {/* Size */}
              <span className="text-xs text-slate-500 w-20 text-right shrink-0">
                {formatSize(file.size)}
              </span>

              {/* Modified time */}
              <span className="text-xs text-slate-500 w-32 text-right shrink-0 hidden md:block">
                {file.modTime}
              </span>

              {/* Download button */}
              {!file.isDir && (
                <button
                  onClick={(e) => {
                    e.stopPropagation();
                    handleDownload(file.path, file.name);
                  }}
                  className="p-1 text-slate-500 hover:text-blue-400 transition-colors"
                  title="Download"
                >
                  <svg className="w-4 h-4" fill="none" viewBox="0 0 24 24" stroke="currentColor">
                    <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M12 10v6m0 0l-3-3m3 3l3-3m2 8H7a2 2 0 01-2-2V5a2 2 0 012-2h5.586a1 1 0 01.707.293l5.414 5.414a1 1 0 01.293.707V19a2 2 0 01-2 2z" />
                  </svg>
                </button>
              )}
            </div>
          ))}

          {files.length === 0 && (
            <div className="px-4 py-8 text-center text-slate-500 text-sm">
              Directory is empty
            </div>
          )}
        </div>
      )}
    </div>
  );
}

export default EdgeFileManager;
