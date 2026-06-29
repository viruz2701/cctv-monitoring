// ═══════════════════════════════════════════════════════════════════════
// WOModals — модальные окна для Complete и Cancel
// ═══════════════════════════════════════════════════════════════════════

import React from 'react';
import {
  XCircle, CheckCircle,
} from '../components/ui/Icons';
import { Modal, Button, FileUpload } from '../../components/ui';
import type { SparePart } from '../../services/sparePartsApi';
import type { PartUsage } from '../../services/workOrdersApi';

interface WOModalsProps {
  completeModal: boolean;
  cancelModal: boolean;
  completeNotes: string;
  cancelReason: string;
  completePhotos: string[];
  completeParts: PartUsage[];
  spareParts: SparePart[];
  submitting: boolean;
  onCompleteClose: () => void;
  onCancelClose: () => void;
  onCompleteNotesChange: (value: string) => void;
  onCancelReasonChange: (value: string) => void;
  onFileUpload: (files: File[]) => Promise<void>;
  onRemovePhoto: (index: number) => void;
  onTogglePart: (partId: string, quantity: number) => void;
  onComplete: () => void;
  onCancel: () => void;
}

export const WOModals: React.FC<WOModalsProps> = ({
  completeModal,
  cancelModal,
  completeNotes,
  cancelReason,
  completePhotos,
  completeParts,
  spareParts,
  submitting,
  onCompleteClose,
  onCancelClose,
  onCompleteNotesChange,
  onCancelReasonChange,
  onFileUpload,
  onRemovePhoto,
  onTogglePart,
  onComplete,
  onCancel,
}) => {
  return (
    <>
      {/* Complete Modal */}
      <Modal
        isOpen={completeModal}
        onClose={onCompleteClose}
        title="Завершить наряд-заказ"
        size="lg"
      >
        <div className="space-y-6">
          {/* Notes */}
          <div>
            <label className="block text-sm font-medium text-slate-700 dark:text-slate-200 mb-2">
              Заметки о выполнении
            </label>
            <textarea
              value={completeNotes}
              onChange={(e) => onCompleteNotesChange(e.target.value)}
              rows={3}
              className="w-full px-3 py-2 border border-slate-300 dark:border-slate-600 rounded-lg bg-white dark:bg-slate-800 text-slate-900 dark:text-white focus:ring-2 focus:ring-blue-500 focus:border-transparent"
              placeholder="Опишите выполненные работы..."
            />
          </div>

          {/* File Upload */}
          <div>
            <label className="block text-sm font-medium text-slate-700 dark:text-slate-200 mb-2">
              Фотографии
            </label>
            <FileUpload
              onUpload={onFileUpload}
              accept="image/*"
              maxFiles={10}
              maxSizeMB={20}
              label="Перетащите фото или нажмите для выбора"
            />
            {completePhotos.length > 0 && (
              <div className="grid grid-cols-3 gap-2 mt-2">
                {completePhotos.map((p, i) => (
                  <div key={i} className="relative aspect-video rounded-lg overflow-hidden bg-slate-100 dark:bg-slate-800">
                    <img src={p} alt="" className="w-full h-full object-cover" />
                    <button
                      onClick={() => onRemovePhoto(i)}
                      className="absolute top-1 right-1 p-0.5 bg-red-500 text-white rounded-full"
                    >
                      <XCircle className="w-3.5 h-3.5" />
                    </button>
                  </div>
                ))}
              </div>
            )}
          </div>

          {/* Parts Selection */}
          <div>
            <label className="block text-sm font-medium text-slate-700 dark:text-slate-200 mb-2">
              Использованные запчасти
            </label>
            <div className="grid grid-cols-1 sm:grid-cols-2 gap-2 max-h-48 overflow-y-auto">
              {spareParts.map(part => {
                const used = completeParts.find(p => p.part_id === part.id);
                return (
                  <div
                    key={part.id}
                    className="flex items-center justify-between p-2 bg-slate-50 dark:bg-slate-800/50 rounded-lg text-sm"
                  >
                    <span className="text-slate-700 dark:text-slate-300 truncate flex-1 mr-2">
                      {part.name}
                    </span>
                    <div className="flex items-center gap-1">
                      <button
                        onClick={() => onTogglePart(part.id, Math.max(0, (used?.quantity || 0) - 1))}
                        className="w-6 h-6 rounded bg-slate-200 dark:bg-slate-700 flex items-center justify-center text-slate-600 dark:text-slate-400 hover:bg-slate-300 dark:hover:bg-slate-600"
                      >
                        −
                      </button>
                      <span className="w-6 text-center font-mono text-sm">
                        {used?.quantity || 0}
                      </span>
                      <button
                        onClick={() => onTogglePart(part.id, (used?.quantity || 0) + 1)}
                        className="w-6 h-6 rounded bg-slate-200 dark:bg-slate-700 flex items-center justify-center text-slate-600 dark:text-slate-400 hover:bg-slate-300 dark:hover:bg-slate-600"
                      >
                        +
                      </button>
                    </div>
                  </div>
                );
              })}
            </div>
          </div>

          <div className="flex justify-end gap-3 pt-4 border-t border-slate-200 dark:border-slate-700">
            <Button variant="outline" onClick={onCompleteClose}>
              Отмена
            </Button>
            <Button
              icon={<CheckCircle className="w-4 h-4" />}
              onClick={onComplete}
              loading={submitting}
            >
              Завершить
            </Button>
          </div>
        </div>
      </Modal>

      {/* Cancel Modal */}
      <Modal
        isOpen={cancelModal}
        onClose={onCancelClose}
        title="Отменить наряд-заказ"
      >
        <div className="space-y-4">
          <div>
            <label className="block text-sm font-medium text-slate-700 dark:text-slate-200 mb-2">
              Причина отмены
            </label>
            <textarea
              value={cancelReason}
              onChange={(e) => onCancelReasonChange(e.target.value)}
              rows={3}
              className="w-full px-3 py-2 border border-slate-300 dark:border-slate-600 rounded-lg bg-white dark:bg-slate-800 text-slate-900 dark:text-white focus:ring-2 focus:ring-red-500 focus:border-transparent"
              placeholder="Укажите причину отмены..."
            />
          </div>
          <div className="flex justify-end gap-3 pt-4 border-t border-slate-200 dark:border-slate-700">
            <Button variant="outline" onClick={onCancelClose}>
              Назад
            </Button>
            <Button
              variant="danger"
              icon={<XCircle className="w-4 h-4" />}
              onClick={onCancel}
              loading={submitting}
              disabled={!cancelReason.trim()}
            >
              Отменить наряд
            </Button>
          </div>
        </div>
      </Modal>
    </>
  );
};
