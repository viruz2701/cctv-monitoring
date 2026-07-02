// ═══════════════════════════════════════════════════════════════════════
// BlockLibrary.tsx — Библиотека блоков для Print Template Builder
//
// UX-3.5: Visual drag-n-drop редактор печатных форм
//   - 10+ готовых блоков (text, table, image, signature, QR)
//   - Drag-n-drop из библиотеки на канвас
//
// Feature Flag: print_template_builder (default: false)
//
// Compliance:
//   - IEC 62443 SR 3.1 (RBAC — feature flag controls access)
//   - OWASP ASVS V1.8 (Feature flags не раскрывают sensitive functionality)
//   - ISO 27001 A.12.4 (Audit trail — каждое изменение логируется)
// ═══════════════════════════════════════════════════════════════════════

import React from 'react';
import {
  Type, Table, Image, PenSquare, QrCode,
  FileText, List, Grid3X3, Signature, Hash,
  Calendar, Clock, User,
} from 'lucide-react';
import type { BlockDefinition, BlockCategory } from './types';

// ── Categories ────────────────────────────────────────────────────────

export const BLOCK_CATEGORIES: BlockCategory[] = [
  { id: 'text', label: 'Text', icon: Type },
  { id: 'data', label: 'Data Fields', icon: FileText },
  { id: 'tables', label: 'Tables', icon: Table },
  { id: 'media', label: 'Media', icon: Image },
  { id: 'signatures', label: 'Signatures & QR', icon: Signature },
  { id: 'layout', label: 'Layout', icon: Grid3X3 },
];

// ── Block Definitions ─────────────────────────────────────────────────

export const BUILT_IN_BLOCKS: BlockDefinition[] = [
  // ── Text Blocks ──────────────────────────────────────────────────
  {
    id: 'heading',
    type: 'text',
    category: 'text',
    label: 'Heading',
    icon: Type,
    defaultProps: {
      content: '{{title}}',
      fontSize: 18,
      fontWeight: 'bold',
      alignment: 'left',
      color: '#1e293b',
    },
    description: 'Заголовок документа (поддерживает переменные)',
  },
  {
    id: 'paragraph',
    type: 'text',
    category: 'text',
    label: 'Paragraph',
    icon: FileText,
    defaultProps: {
      content: '{{description}}',
      fontSize: 12,
      fontWeight: 'normal',
      alignment: 'left',
      color: '#475569',
    },
    description: 'Текстовый абзац',
  },

  // ── Data Field Blocks ────────────────────────────────────────────
  {
    id: 'field-device',
    type: 'field',
    category: 'data',
    label: 'Device Name',
    icon: User,
    defaultProps: {
      field: 'device_name',
      label: 'Device:',
      fontSize: 11,
      showLabel: true,
    },
    description: 'Название устройства',
  },
  {
    id: 'field-date',
    type: 'field',
    category: 'data',
    label: 'Date',
    icon: Calendar,
    defaultProps: {
      field: 'completed_at',
      label: 'Date:',
      format: 'dd.MM.yyyy',
      fontSize: 11,
      showLabel: true,
    },
    description: 'Дата завершения',
  },
  {
    id: 'field-technician',
    type: 'field',
    category: 'data',
    label: 'Technician',
    icon: Clock,
    defaultProps: {
      field: 'technician_name',
      label: 'Technician:',
      fontSize: 11,
      showLabel: true,
    },
    description: 'ФИО техника',
  },
  {
    id: 'field-duration',
    type: 'field',
    category: 'data',
    label: 'Duration',
    icon: Clock,
    defaultProps: {
      field: 'duration_minutes',
      label: 'Duration:',
      format: '{{value}} min',
      fontSize: 11,
      showLabel: true,
    },
    description: 'Длительность работ',
  },

  // ── Table Blocks ─────────────────────────────────────────────────
  {
    id: 'checklist-table',
    type: 'table',
    category: 'tables',
    label: 'Checklist Table',
    icon: List,
    defaultProps: {
      dataSource: 'checklist',
      columns: [
        { key: 'task', label: 'Task', width: 'auto' },
        { key: 'completed', label: 'Done', width: 60 },
      ],
      showHeader: true,
      bordered: true,
      fontSize: 10,
    },
    description: 'Чеклист работ (из WorkOrder)',
  },
  {
    id: 'parts-table',
    type: 'table',
    category: 'tables',
    label: 'Parts Table',
    icon: Table,
    defaultProps: {
      dataSource: 'parts_used',
      columns: [
        { key: 'name', label: 'Part Name', width: 'auto' },
        { key: 'sku', label: 'SKU', width: 80 },
        { key: 'quantity', label: 'Qty', width: 50 },
        { key: 'unit_price', label: 'Price', width: 70 },
        { key: 'total_price', label: 'Total', width: 70 },
      ],
      showHeader: true,
      bordered: true,
      fontSize: 9,
    },
    description: 'Таблица запчастей',
  },
  {
    id: 'labor-table',
    type: 'table',
    category: 'tables',
    label: 'Labor Table',
    icon: Grid3X3,
    defaultProps: {
      dataSource: 'labor',
      columns: [
        { key: 'description', label: 'Description', width: 'auto' },
        { key: 'hours', label: 'Hours', width: 60 },
        { key: 'rate', label: 'Rate', width: 60 },
        { key: 'total', label: 'Total', width: 70 },
      ],
      showHeader: true,
      bordered: true,
      fontSize: 9,
    },
    description: 'Таблица трудозатрат',
  },

  // ── Media Blocks ─────────────────────────────────────────────────
  {
    id: 'image',
    type: 'image',
    category: 'media',
    label: 'Image',
    icon: Image,
    defaultProps: {
      src: '',
      alt: 'Image',
      width: 200,
      height: 150,
      fit: 'contain',
    },
    description: 'Изображение (логотип, схема)',
  },
  {
    id: 'photo-grid',
    type: 'image',
    category: 'media',
    label: 'Photo Grid',
    icon: Image,
    defaultProps: {
      dataSource: 'photos',
      columns: 2,
      maxPhotos: 4,
      width: 100,
      height: 100,
      fit: 'cover',
    },
    description: 'Сетка фото (из WorkOrder)',
  },

  // ── Signature & QR Blocks ────────────────────────────────────────
  {
    id: 'signature-line',
    type: 'signature',
    category: 'signatures',
    label: 'Signature Line',
    icon: PenSquare,
    defaultProps: {
      label: 'Technician Signature',
      showDate: true,
      lineWidth: 200,
    },
    description: 'Строка для подписи',
  },
  {
    id: 'signature-digital',
    type: 'signature',
    category: 'signatures',
    label: 'Digital Signature',
    icon: Signature,
    defaultProps: {
      label: 'Digital Signature',
      showHash: true,
      algorithm: 'bash-256',
    },
    description: 'Цифровая подпись (СТБ bash-256)',
  },
  {
    id: 'qr-code',
    type: 'qr',
    category: 'signatures',
    label: 'QR Code',
    icon: QrCode,
    defaultProps: {
      dataSource: 'work_order_id',
      size: 80,
      label: 'Scan to verify',
    },
    description: 'QR-код для верификации',
  },
  {
    id: 'qr-verification',
    type: 'qr',
    category: 'signatures',
    label: 'Verification QR',
    icon: Hash,
    defaultProps: {
      dataSource: 'verification_url',
      size: 100,
      label: 'Verify Document',
      includeHash: true,
    },
    description: 'QR для верификации документа',
  },
];

// ── BlockLibrary Component ────────────────────────────────────────────

interface BlockLibraryProps {
  /** Выбранная категория */
  selectedCategory: string;
  /** Смена категории */
  onCategoryChange: (categoryId: string) => void;
  /** Добавление блока на канвас */
  onAddBlock: (block: BlockDefinition) => void;
  /** Поисковый запрос */
  searchQuery?: string;
  /** Смена поискового запроса */
  onSearchChange?: (query: string) => void;
}

export function BlockLibrary({
  selectedCategory,
  onCategoryChange,
  onAddBlock,
  searchQuery = '',
  onSearchChange,
}: BlockLibraryProps) {
  const filteredBlocks = BUILT_IN_BLOCKS.filter((block) => {
    const matchesCategory = selectedCategory === 'all' || block.category === selectedCategory;
    const matchesSearch = !searchQuery || 
      block.label.toLowerCase().includes(searchQuery.toLowerCase()) ||
      block.description.toLowerCase().includes(searchQuery.toLowerCase());
    return matchesCategory && matchesSearch;
  });

  return (
    <div className="flex flex-col h-full">
      {/* Search */}
      {onSearchChange && (
        <div className="px-3 py-2 border-b border-slate-200 dark:border-slate-700">
          <input
            type="text"
            value={searchQuery}
            onChange={(e) => onSearchChange(e.target.value)}
            placeholder="Search blocks..."
            className="w-full px-3 py-1.5 text-sm border border-slate-300 dark:border-slate-600 rounded-lg bg-white dark:bg-slate-800 text-slate-900 dark:text-slate-100 placeholder-slate-400 focus:outline-none focus:ring-2 focus:ring-blue-500"
            aria-label="Search blocks"
          />
        </div>
      )}

      {/* Categories */}
      <div className="px-3 py-2 border-b border-slate-200 dark:border-slate-700">
        <div className="flex flex-wrap gap-1">
          <button
            onClick={() => onCategoryChange('all')}
            className={`px-2.5 py-1 rounded-md text-xs font-medium transition-colors ${
              selectedCategory === 'all'
                ? 'bg-blue-100 text-blue-700 dark:bg-blue-900/40 dark:text-blue-400'
                : 'text-slate-600 dark:text-slate-400 hover:bg-slate-100 dark:hover:bg-slate-800'
            }`}
            aria-pressed={selectedCategory === 'all'}
          >
            All
          </button>
          {BLOCK_CATEGORIES.map((cat) => {
            const Icon = cat.icon;
            return (
              <button
                key={cat.id}
                onClick={() => onCategoryChange(cat.id)}
                className={`inline-flex items-center gap-1 px-2.5 py-1 rounded-md text-xs font-medium transition-colors ${
                  selectedCategory === cat.id
                    ? 'bg-blue-100 text-blue-700 dark:bg-blue-900/40 dark:text-blue-400'
                    : 'text-slate-600 dark:text-slate-400 hover:bg-slate-100 dark:hover:bg-slate-800'
                }`}
                aria-pressed={selectedCategory === cat.id}
              >
                <Icon className="w-3.5 h-3.5" aria-hidden="true" />
                {cat.label}
              </button>
            );
          })}
        </div>
      </div>

      {/* Blocks Grid */}
      <div className="flex-1 overflow-y-auto p-3">
        {filteredBlocks.length === 0 ? (
          <div className="flex flex-col items-center justify-center py-8 text-center">
            <Type className="w-8 h-8 text-slate-300 dark:text-slate-600 mb-2" aria-hidden="true" />
            <p className="text-sm text-slate-500 dark:text-slate-400">
              {searchQuery ? 'No blocks match your search' : 'No blocks in this category'}
            </p>
          </div>
        ) : (
          <div className="grid grid-cols-1 gap-2">
            {filteredBlocks.map((block) => {
              const Icon = block.icon;
              return (
                <button
                  key={block.id}
                  onClick={() => onAddBlock(block)}
                  draggable
                  onDragStart={(e) => {
                    e.dataTransfer.setData('application/json', JSON.stringify(block));
                    e.dataTransfer.effectAllowed = 'copy';
                  }}
                  className="flex items-start gap-3 p-3 rounded-lg border border-slate-200 dark:border-slate-700 bg-white dark:bg-slate-800 hover:border-blue-300 dark:hover:border-blue-600 hover:shadow-sm transition-all cursor-grab active:cursor-grabbing text-left group"
                  aria-label={`Add ${block.label} block: ${block.description}`}
                  title={block.description}
                >
                  <div className="flex-shrink-0 w-8 h-8 rounded-md bg-blue-50 dark:bg-blue-900/30 flex items-center justify-center">
                    <Icon className="w-4 h-4 text-blue-600 dark:text-blue-400" aria-hidden="true" />
                  </div>
                  <div className="flex-1 min-w-0">
                    <div className="text-sm font-medium text-slate-900 dark:text-slate-100 truncate">
                      {block.label}
                    </div>
                    <div className="text-xs text-slate-500 dark:text-slate-400 line-clamp-2 mt-0.5">
                      {block.description}
                    </div>
                  </div>
                  <span className="flex-shrink-0 text-xs text-slate-400 group-hover:text-blue-500 transition-colors mt-1">
                    +
                  </span>
                </button>
              );
            })}
          </div>
        )}
      </div>

      {/* Summary */}
      <div className="px-3 py-2 border-t border-slate-200 dark:border-slate-700 bg-slate-50 dark:bg-slate-800/50">
        <p className="text-xs text-slate-500 dark:text-slate-400">
          {filteredBlocks.length} block{filteredBlocks.length !== 1 ? 's' : ''} available
        </p>
      </div>
    </div>
  );
}

export default BlockLibrary;
