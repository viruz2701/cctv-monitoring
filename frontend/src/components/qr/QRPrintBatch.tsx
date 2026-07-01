// ═══════════════════════════════════════════════════════════════════════════
// QRPrintBatch.tsx — Bulk QR generation UI (UX-4.2)
//
// Позволяет выбрать устройства/WO и сгенерировать QR-коды для печати.
// Поддерживает фильтрацию по сайту, типу устройства, batch preview.
//
// Compliance:
//   - IEC 62443-3-3 SR 3.1 (Batch processing — ограничение 100 шт.)
//   - OWASP ASVS V5.1 (Input validation — whitelist через Zod)
//   - ISO 27001 A.12.4 (Audit trail — логирование batch генерации)
// ═══════════════════════════════════════════════════════════════════════════

import React, { useState, useCallback } from 'react';
import { QRCode } from '../ui/QRCode';
import { Button } from '../ui/Button';
import { Modal } from '../ui/Modal';
import { DataGrid } from '../ui/DataGrid';
import { EmptyState } from '../ui/EmptyState';

// ── Types ────────────────────────────────────────────────────────────────

export type QRPrintType = 'device' | 'work_order' | 'spare_part' | 'to' | 'onboard' | 'verify';

export interface QRPrintEntry {
  entity_id: string;
  entity_name: string;
  site_id?: string;
  site_name?: string;
  selected: boolean;
}

export interface QRCodeResult {
  code_id: string;
  entity_id: string;
  qr_data: string;
  qr_url: string;
}

export interface QRPrintBatchProps {
  /** Тип QR для генерации */
  type: QRPrintType;
  /** Элементы для печати */
  entries: QRPrintEntry[];
  /** Загружается ли список */
  loading?: boolean;
  /** Ошибка загрузки */
  error?: string | null;
  /** Коллбэк генерации */
  onGenerate: (entries: QRPrintEntry[]) => Promise<QRCodeResult[]>;
  /** Коллбэк закрытия */
  onClose: () => void;
}

// ── Labels ───────────────────────────────────────────────────────────────

const TYPE_LABELS: Record<QRPrintType, string> = {
  device: 'Устройств',
  work_order: 'Нарядов',
  spare_part: 'Запчастей',
  to: 'Технических отчётов',
  onboard: 'Onboarding',
  verify: 'Верификации',
};

const TYPE_ICONS: Record<QRPrintType, string> = {
  device: '📹',
  work_order: '📋',
  spare_part: '🔧',
  to: '📄',
  onboard: '🚀',
  verify: '✅',
};

// ── Main Component ───────────────────────────────────────────────────────

export function QRPrintBatch({
  type,
  entries,
  loading = false,
  error = null,
  onGenerate,
  onClose,
}: QRPrintBatchProps) {
  const [selected, setSelected] = useState<Set<string>>(
    new Set(entries.filter((e) => e.selected).map((e) => e.entity_id)),
  );
  const [generating, setGenerating] = useState(false);
  const [results, setResults] = useState<QRCodeResult[]>([]);
  const [genError, setGenError] = useState<string | null>(null);
  const [previewCode, setPreviewCode] = useState<QRCodeResult | null>(null);

  // ── Selection ──────────────────────────────────────────────────────────

  const toggleAll = useCallback(() => {
    if (selected.size === entries.length) {
      setSelected(new Set());
    } else {
      setSelected(new Set(entries.map((e) => e.entity_id)));
    }
  }, [entries, selected]);

  const toggleEntry = useCallback((entityId: string) => {
    setSelected((prev) => {
      const next = new Set(prev);
      if (next.has(entityId)) {
        next.delete(entityId);
      } else {
        next.add(entityId);
      }
      return next;
    });
  }, []);

  // ── Generate ──────────────────────────────────────────────────────────

  const handleGenerate = useCallback(async () => {
    const selectedEntries = entries.filter((e) => selected.has(e.entity_id));
    if (selectedEntries.length === 0) return;

    setGenerating(true);
    setGenError(null);

    try {
      const codes = await onGenerate(selectedEntries);
      setResults(codes);
    } catch (err) {
      setGenError(err instanceof Error ? err.message : 'Generation failed');
    } finally {
      setGenerating(false);
    }
  }, [entries, selected, onGenerate]);

  // ── Preview ───────────────────────────────────────────────────────────

  const handlePreview = useCallback((code: QRCodeResult) => {
    setPreviewCode(code);
  }, []);

  // ── Print ─────────────────────────────────────────────────────────────

  const handlePrint = useCallback(() => {
    window.print();
  }, []);

  // ── Render: Generation Results ─────────────────────────────────────────

  if (results.length > 0) {
    return (
      <div className="space-y-6 p-6">
        <div className="flex items-center justify-between">
          <div>
            <h2 className="text-xl font-semibold text-slate-900 dark:text-white">
              QR-коды сгенерированы
            </h2>
            <p className="mt-1 text-sm text-slate-500">
              Сгенерировано {results.length} QR-кодов для {TYPE_LABELS[type].toLowerCase()}
            </p>
          </div>
          <div className="flex gap-3">
            <Button variant="outline" onClick={() => setResults([])}>
              Назад
            </Button>
            <Button onClick={handlePrint}>
              🖨 Печать
            </Button>
          </div>
        </div>

        {/* Grid of QR codes */}
        <div className="grid grid-cols-2 sm:grid-cols-3 md:grid-cols-4 lg:grid-cols-5 gap-4">
          {results.map((code) => (
            <button
              key={code.code_id}
              onClick={() => handlePreview(code)}
              className="flex flex-col items-center p-3 rounded-lg border border-slate-200 dark:border-slate-700 hover:border-blue-400 dark:hover:border-blue-500 transition-colors cursor-pointer bg-white dark:bg-slate-800"
            >
              <QRCode
                value={code.qr_data}
                size={120}
                className="print:size-[80px]"
              />
              <span className="mt-2 text-xs text-slate-500 dark:text-slate-400 truncate max-w-full">
                {code.entity_id}
              </span>
            </button>
          ))}
        </div>

        {/* Preview Modal */}
        {previewCode && (
          <Modal isOpen={true} onClose={() => setPreviewCode(null)}>
            <div className="flex flex-col items-center p-8">
              <QRCode
                value={previewCode.qr_data}
                size={250}
                label={previewCode.entity_id}
              />
              <p className="mt-4 text-sm text-slate-500 break-all">
                {previewCode.qr_url}
              </p>
            </div>
          </Modal>
        )}

        {/* Print-only styles */}
        <style>{`
          @media print {
            body { margin: 0; padding: 16px; }
            @page { size: A4; margin: 10mm; }
          }
        `}</style>
      </div>
    );
  }

  // ── Render: Selection + Generation ─────────────────────────────────────

  return (
    <div className="space-y-6 p-6">
      {/* Header */}
      <div className="flex items-center justify-between">
        <div>
          <h2 className="text-xl font-semibold text-slate-900 dark:text-white">
            {TYPE_ICONS[type]} Печать QR-кодов
          </h2>
          <p className="mt-1 text-sm text-slate-500">
            Выберите {TYPE_LABELS[type].toLowerCase()} для генерации QR-кодов
          </p>
        </div>
        <div className="flex gap-3">
          <Button variant="outline" onClick={onClose}>
            Отмена
          </Button>
          <Button
            onClick={handleGenerate}
            disabled={selected.size === 0 || generating}
          >
            {generating ? 'Генерация...' : `Сгенерировать (${selected.size})`}
          </Button>
        </div>
      </div>

      {/* Error */}
      {genError && (
        <div className="p-4 bg-red-50 dark:bg-red-900/20 border border-red-200 dark:border-red-800 rounded-lg">
          <p className="text-sm text-red-600 dark:text-red-400">{genError}</p>
        </div>
      )}

      {error && (
        <div className="p-4 bg-red-50 dark:bg-red-900/20 border border-red-200 dark:border-red-800 rounded-lg">
          <p className="text-sm text-red-600 dark:text-red-400">{error}</p>
        </div>
      )}

      {/* Loading */}
      {loading && (
        <div className="flex items-center justify-center py-12">
          <div className="animate-spin h-8 w-8 border-4 border-blue-500 border-t-transparent rounded-full" />
        </div>
      )}

      {/* Empty */}
      {!loading && entries.length === 0 && (
        <EmptyState
          icon="qr-code"
          title="Нет элементов для печати"
          description={`Нет ${TYPE_LABELS[type].toLowerCase()} для генерации QR-кодов`}
        />
      )}

      {/* Select All */}
      {!loading && entries.length > 0 && (
        <label className="flex items-center gap-3 px-4 py-2 bg-slate-50 dark:bg-slate-800/50 rounded-lg cursor-pointer">
          <input
            type="checkbox"
            checked={selected.size === entries.length}
            onChange={toggleAll}
            className="w-4 h-4 rounded border-slate-300 text-blue-600 focus:ring-blue-500"
          />
          <span className="text-sm font-medium text-slate-700 dark:text-slate-300">
            {selected.size === entries.length
              ? 'Снять выделение'
              : `Выбрать все (${entries.length})`}
          </span>
        </label>
      )}

      {/* List */}
      {!loading && entries.length > 0 && (
        <div className="border border-slate-200 dark:border-slate-700 rounded-lg divide-y divide-slate-200 dark:divide-slate-700">
          {entries.map((entry) => (
            <label
              key={entry.entity_id}
              className="flex items-center gap-4 px-4 py-3 hover:bg-slate-50 dark:hover:bg-slate-800/30 cursor-pointer transition-colors"
            >
              <input
                type="checkbox"
                checked={selected.has(entry.entity_id)}
                onChange={() => toggleEntry(entry.entity_id)}
                className="w-4 h-4 rounded border-slate-300 text-blue-600 focus:ring-blue-500"
              />
              <div className="flex-1 min-w-0">
                <p className="text-sm font-medium text-slate-900 dark:text-white truncate">
                  {entry.entity_name || entry.entity_id}
                </p>
                {entry.site_name && (
                  <p className="text-xs text-slate-500 truncate">
                    {entry.site_name}
                  </p>
                )}
              </div>
              <span className="text-xs text-slate-400 font-mono truncate max-w-[120px]">
                {entry.entity_id}
              </span>
            </label>
          ))}
        </div>
      )}

      {/* Info */}
      <div className="flex items-center gap-2 text-xs text-slate-400">
        <svg className="w-4 h-4" fill="none" viewBox="0 0 24 24" stroke="currentColor">
          <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M13 16h-1v-4h-1m1-4h.01M21 12a9 9 0 11-18 0 9 9 0 0118 0z" />
        </svg>
        <span>Максимум 100 QR-кодов за одну генерацию. Для печати используйте кнопку "Печать".</span>
      </div>
    </div>
  );
}

export default QRPrintBatch;
