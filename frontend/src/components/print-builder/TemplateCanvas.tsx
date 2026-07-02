// ═══════════════════════════════════════════════════════════════════════
// TemplateCanvas.tsx — Drag-n-drop canvas for Print Template Builder
//
// UX-3.5: Visual drag-n-drop редактор печатных форм
//   - Canvas с сеткой и направляющими
//   - Drop zone для блоков из библиотеки
//   - Выделение, перемещение, ресайз блоков
//   - Preview в реальном времени
//
// Feature Flag: print_template_builder (default: false)
//
// Compliance:
//   - IEC 62443 SR 3.1 (RBAC — feature flag controls access)
//   - OWASP ASVS V1.8 (Feature flags)
//   - WCAG 2.1 AA (Keyboard navigation, focus management)
// ═══════════════════════════════════════════════════════════════════════

import React, { useCallback, useRef, useState, useEffect } from 'react';
import { isFeatureEnabled } from '../../config/featureFlags';
import type {
  CanvasBlock,
  BlockDefinition,
  BlockType,
  PageSettings,
  TextBlockProps,
  FieldBlockProps,
  TableBlockProps,
  ImageBlockProps,
  SignatureBlockProps,
  QRBlockProps,
} from './types';
import { Trash2, Lock, Unlock, Eye, EyeOff, Grid3X3 } from 'lucide-react';

// ── Constants ─────────────────────────────────────────────────────────

const SNAP_TO_GRID = 5;
const MIN_BLOCK_WIDTH = 40;
const MIN_BLOCK_HEIGHT = 20;
const CANVAS_WIDTH = 595; // A4 in px at 72dpi
const CANVAS_HEIGHT = 842;

const PAGE_SIZE_MAP: Record<string, { width: number; height: number }> = {
  a4: { width: 595, height: 842 },
  a5: { width: 420, height: 595 },
  letter: { width: 612, height: 792 },
  custom: { width: 595, height: 842 },
};

// ── Block Renderers ───────────────────────────────────────────────────

function renderBlockPreview(block: CanvasBlock): React.ReactNode {
  switch (block.type) {
    case 'text': {
      const p = block.props as unknown as TextBlockProps;
      return (
        <div
          style={{
            fontSize: p.fontSize || 14,
            fontWeight: p.fontWeight || 'normal',
            textAlign: p.alignment || 'left',
            color: p.color || '#1e293b',
          }}
        >
          {p.content || 'Text block'}
        </div>
      );
    }
    case 'field': {
      const p = block.props as unknown as FieldBlockProps;
      return (
        <div className="flex items-center gap-1" style={{ fontSize: p.fontSize || 11 }}>
          {p.showLabel && (
            <span className="text-slate-500 font-medium">{p.label}</span>
          )}
          <span className="text-slate-800">{`{{${p.field}}}`}</span>
        </div>
      );
    }
    case 'table': {
      const p = block.props as unknown as TableBlockProps;
      return (
        <div className="text-xs text-slate-500">
          [{p.dataSource}] {p.columns.length} columns
        </div>
      );
    }
    case 'image': {
      const p = block.props as unknown as ImageBlockProps;
      return (
        <div className="flex items-center justify-center h-full bg-slate-100 dark:bg-slate-700 text-slate-400 text-xs">
          {p.dataSource ? `[${p.dataSource}]` : 'Image'}
        </div>
      );
    }
    case 'signature': {
      const p = block.props as unknown as SignatureBlockProps;
      return (
        <div className="text-xs text-slate-500">
          <div className="border-b border-slate-300 h-6 mb-1" />
          <span>{p.label}</span>
        </div>
      );
    }
    case 'qr': {
      const p = block.props as unknown as QRBlockProps;
      return (
        <div className="flex flex-col items-center justify-center gap-1">
          <div className="w-12 h-12 bg-slate-200 dark:bg-slate-600 rounded" />
          <span className="text-[8px] text-slate-500">{p.label}</span>
        </div>
      );
    }
    default:
      return <div className="text-sm text-slate-500">{block.label}</div>;
  }
}

// ── Props ─────────────────────────────────────────────────────────────

interface TemplateCanvasProps {
  blocks: CanvasBlock[];
  selectedBlockId: string | null;
  pageSettings: PageSettings;
  zoom: number;
  onSelectBlock: (id: string | null) => void;
  onMoveBlock: (id: string, x: number, y: number) => void;
  onResizeBlock: (id: string, w: number, h: number) => void;
  onRemoveBlock: (id: string) => void;
  onToggleLock: (id: string) => void;
  onToggleVisibility: (id: string) => void;
  onDropBlock: (block: BlockDefinition, x: number, y: number) => void;
  onReorderBlocks: (blocks: CanvasBlock[]) => void;
}

// ── TemplateCanvas Component ──────────────────────────────────────────

export function TemplateCanvas({
  blocks,
  selectedBlockId,
  pageSettings,
  zoom,
  onSelectBlock,
  onMoveBlock,
  onResizeBlock,
  onRemoveBlock,
  onToggleLock,
  onToggleVisibility,
  onDropBlock,
  onReorderBlocks,
}: TemplateCanvasProps) {
  const canvasRef = useRef<HTMLDivElement>(null);
  const [isDragging, setIsDragging] = useState(false);
  const [dragOffset, setDragOffset] = useState({ x: 0, y: 0 });
  const [dragBlockId, setDragBlockId] = useState<string | null>(null);
  const [showGrid, setShowGrid] = useState(true);

  const isEnabled = isFeatureEnabled('print_template_builder');
  const pageSize = PAGE_SIZE_MAP[pageSettings.pageSize] || PAGE_SIZE_MAP.a4;
  const scaledWidth = pageSize.width * (zoom / 100);
  const scaledHeight = pageSize.height * (zoom / 100);

  // ── Drag over (from library) ─────────────────────────────────────
  const handleDragOver = useCallback((e: React.DragEvent) => {
    e.preventDefault();
    e.dataTransfer.dropEffect = 'copy';
  }, []);

  const handleDrop = useCallback(
    (e: React.DragEvent) => {
      e.preventDefault();
      const data = e.dataTransfer.getData('application/json');
      if (!data) return;

      try {
        const blockDef: BlockDefinition = JSON.parse(data);
        const rect = canvasRef.current?.getBoundingClientRect();
        if (!rect) return;

        const x = Math.round((e.clientX - rect.left) / (zoom / 100) / SNAP_TO_GRID) * SNAP_TO_GRID;
        const y = Math.round((e.clientY - rect.top) / (zoom / 100) / SNAP_TO_GRID) * SNAP_TO_GRID;

        onDropBlock(blockDef, Math.max(0, x), Math.max(0, y));
      } catch {
        // ignore invalid data
      }
    },
    [onDropBlock, zoom],
  );

  // ── Block dragging on canvas ─────────────────────────────────────
  const handleBlockMouseDown = useCallback(
    (e: React.MouseEvent, block: CanvasBlock) => {
      if (block.locked) return;
      e.stopPropagation();
      onSelectBlock(block.instanceId);

      const rect = canvasRef.current?.getBoundingClientRect();
      if (!rect) return;

      setIsDragging(true);
      setDragBlockId(block.instanceId);
      setDragOffset({
        x: e.clientX - rect.left - block.position.x * (zoom / 100),
        y: e.clientY - rect.top - block.position.y * (zoom / 100),
      });
    },
    [onSelectBlock, zoom],
  );

  useEffect(() => {
    if (!isDragging || !dragBlockId) return;

    const handleMouseMove = (e: MouseEvent) => {
      const rect = canvasRef.current?.getBoundingClientRect();
      if (!rect) return;

      const rawX = (e.clientX - rect.left - dragOffset.x) / (zoom / 100);
      const rawY = (e.clientY - rect.top - dragOffset.y) / (zoom / 100);

      const x = Math.max(0, Math.round(rawX / SNAP_TO_GRID) * SNAP_TO_GRID);
      const y = Math.max(0, Math.round(rawY / SNAP_TO_GRID) * SNAP_TO_GRID);

      onMoveBlock(dragBlockId, x, y);
    };

    const handleMouseUp = () => {
      setIsDragging(false);
      setDragBlockId(null);
    };

    window.addEventListener('mousemove', handleMouseMove);
    window.addEventListener('mouseup', handleMouseUp);

    return () => {
      window.removeEventListener('mousemove', handleMouseMove);
      window.removeEventListener('mouseup', handleMouseUp);
    };
  }, [isDragging, dragBlockId, dragOffset, onMoveBlock, zoom]);

  // ── Resize handlers ─────────────────────────────────────────────
  const handleResizeStart = useCallback(
    (e: React.MouseEvent, blockId: string) => {
      e.stopPropagation();
      e.preventDefault();

      const startX = e.clientX;
      const startY = e.clientY;
      const block = blocks.find((b) => b.instanceId === blockId);
      if (!block) return;

      const startWidth = block.size.width;
      const startHeight = block.size.height;

      const handleResizeMove = (ev: MouseEvent) => {
        const dx = (ev.clientX - startX) / (zoom / 100);
        const dy = (ev.clientY - startY) / (zoom / 100);
        const newW = Math.max(MIN_BLOCK_WIDTH, startWidth + dx);
        const newH = Math.max(MIN_BLOCK_HEIGHT, startHeight + dy);
        onResizeBlock(blockId, Math.round(newW), Math.round(newH));
      };

      const handleResizeUp = () => {
        window.removeEventListener('mousemove', handleResizeMove);
        window.removeEventListener('mouseup', handleResizeUp);
      };

      window.addEventListener('mousemove', handleResizeMove);
      window.addEventListener('mouseup', handleResizeUp);
    },
    [blocks, onResizeBlock, zoom],
  );

  // ── Keyboard handlers ────────────────────────────────────────────
  useEffect(() => {
    if (!selectedBlockId) return;

    const handleKeyDown = (e: KeyboardEvent) => {
      const block = blocks.find((b) => b.instanceId === selectedBlockId);
      if (!block || block.locked) return;

      const step = e.shiftKey ? 10 : 1;

      switch (e.key) {
        case 'Delete':
        case 'Backspace':
          if (!e.ctrlKey && !e.metaKey) {
            e.preventDefault();
            onRemoveBlock(selectedBlockId);
          }
          break;
        case 'ArrowUp':
          e.preventDefault();
          onMoveBlock(selectedBlockId, block.position.x, block.position.y - step);
          break;
        case 'ArrowDown':
          e.preventDefault();
          onMoveBlock(selectedBlockId, block.position.x, block.position.y + step);
          break;
        case 'ArrowLeft':
          e.preventDefault();
          onMoveBlock(selectedBlockId, block.position.x - step, block.position.y);
          break;
        case 'ArrowRight':
          e.preventDefault();
          onMoveBlock(selectedBlockId, block.position.x + step, block.position.y);
          break;
      }
    };

    window.addEventListener('keydown', handleKeyDown);
    return () => window.removeEventListener('keydown', handleKeyDown);
  }, [selectedBlockId, blocks, onMoveBlock, onRemoveBlock]);

  // ── Canvas click (deselect) ──────────────────────────────────────
  const handleCanvasClick = useCallback(
    (e: React.MouseEvent) => {
      if (e.target === canvasRef.current || (e.target as HTMLElement).dataset?.canvas) {
        onSelectBlock(null);
      }
    },
    [onSelectBlock],
  );

  if (!isEnabled) {
    return (
      <div className="flex items-center justify-center h-64 bg-slate-50 dark:bg-slate-900 rounded-lg border-2 border-dashed border-slate-300 dark:border-slate-600">
        <div className="text-center">
          <Grid3X3 className="w-8 h-8 text-slate-400 mx-auto mb-2" aria-hidden="true" />
          <p className="text-sm text-slate-500">
            Print Template Builder is disabled
          </p>
          <p className="text-xs text-slate-400 mt-1">
            Enable `print_template_builder` feature flag
          </p>
        </div>
      </div>
    );
  }

  return (
    <div className="flex flex-col items-center">
      {/* Canvas Toolbar */}
      <div className="flex items-center gap-2 mb-3 px-3 py-1.5 bg-white dark:bg-slate-800 rounded-lg border border-slate-200 dark:border-slate-700">
        <button
          onClick={() => setShowGrid(!showGrid)}
          className={`p-1.5 rounded-md transition-colors ${
            showGrid ? 'bg-blue-100 text-blue-700' : 'text-slate-500 hover:bg-slate-100'
          }`}
          aria-label={showGrid ? 'Hide grid' : 'Show grid'}
          title={showGrid ? 'Hide grid' : 'Show grid'}
        >
          <Grid3X3 className="w-4 h-4" aria-hidden="true" />
        </button>
        <span className="text-xs text-slate-500 ml-2">
          {blocks.length} block{blocks.length !== 1 ? 's' : ''}
        </span>
        <span className="text-xs text-slate-400 ml-auto">
          {pageSettings.pageSize.toUpperCase()} · {zoom}%
        </span>
      </div>

      {/* Canvas */}
      <div
        ref={canvasRef}
        data-canvas="true"
        className="relative bg-white dark:bg-slate-800 shadow-lg border border-slate-200 dark:border-slate-700"
        style={{
          width: scaledWidth,
          height: scaledHeight,
          backgroundImage: showGrid
            ? `radial-gradient(circle, #cbd5e1 0.5px, transparent 0.5px)`
            : undefined,
          backgroundSize: `${SNAP_TO_GRID * (zoom / 100)}px ${SNAP_TO_GRID * (zoom / 100)}px`,
        }}
        onDragOver={handleDragOver}
        onDrop={handleDrop}
        onClick={handleCanvasClick}
        role="region"
        aria-label="Print template canvas"
        aria-roledescription="Drop zone for template blocks"
      >
        {/* Empty state */}
        {blocks.length === 0 && (
          <div
            className="absolute inset-0 flex items-center justify-center"
            data-canvas="true"
          >
            <div className="text-center" data-canvas="true">
              <Grid3X3 className="w-10 h-10 text-slate-300 dark:text-slate-600 mx-auto mb-3" data-canvas="true" aria-hidden="true" />
              <p className="text-sm text-slate-400 dark:text-slate-500" data-canvas="true">
                Drag blocks from the library to start building
              </p>
              <p className="text-xs text-slate-300 dark:text-slate-600 mt-1" data-canvas="true">
                Or click a block in the library to add it
              </p>
            </div>
          </div>
        )}

        {/* Blocks */}
        {blocks.map((block) => {
          const isSelected = block.instanceId === selectedBlockId;
          const scale = zoom / 100;

          return (
            <div
              key={block.instanceId}
              className={`absolute cursor-move group ${
                !block.visible ? 'opacity-30' : ''
              }`}
              style={{
                left: block.position.x * scale,
                top: block.position.y * scale,
                width: block.size.width * scale,
                height: block.size.height * scale,
              }}
              onMouseDown={(e) => handleBlockMouseDown(e, block)}
              role="button"
              aria-label={`${block.label} block${block.locked ? ' (locked)' : ''}`}
              aria-selected={isSelected}
              tabIndex={0}
              onFocus={() => onSelectBlock(block.instanceId)}
            >
              {/* Block Content */}
              <div
                className={`w-full h-full p-2 overflow-hidden rounded border-2 transition-all ${
                  isSelected
                    ? 'border-blue-500 bg-blue-50/50 dark:bg-blue-900/20 shadow-md'
                    : 'border-slate-200 dark:border-slate-700 bg-white dark:bg-slate-800 hover:border-slate-300'
                } ${block.locked ? 'border-dashed' : ''}`}
              >
                {renderBlockPreview(block)}
              </div>

              {/* Selected overlay controls */}
              {isSelected && !block.locked && (
                <>
                  {/* Resize handle (bottom-right) */}
                  <div
                    className="absolute bottom-0 right-0 w-3 h-3 bg-blue-500 border-2 border-white rounded-sm cursor-se-resize"
                    onMouseDown={(e) => handleResizeStart(e, block.instanceId)}
                    aria-label="Resize block"
                  />
                  {/* Block toolbar */}
                  <div className="absolute -top-8 right-0 flex items-center gap-0.5 bg-white dark:bg-slate-800 rounded-md shadow-md border border-slate-200 dark:border-slate-700 opacity-0 group-hover:opacity-100 transition-opacity">
                    <button
                      onClick={(e) => { e.stopPropagation(); onToggleVisibility(block.instanceId); }}
                      className="p-1 hover:bg-slate-100 dark:hover:bg-slate-700 rounded-l-md"
                      aria-label={block.visible ? 'Hide block' : 'Show block'}
                      title={block.visible ? 'Hide' : 'Show'}
                    >
                      {block.visible ? <EyeOff className="w-3 h-3 text-slate-500" /> : <Eye className="w-3 h-3 text-slate-500" />}
                    </button>
                    <button
                      onClick={(e) => { e.stopPropagation(); onToggleLock(block.instanceId); }}
                      className="p-1 hover:bg-slate-100 dark:hover:bg-slate-700"
                      aria-label={block.locked ? 'Unlock block' : 'Lock block'}
                      title={block.locked ? 'Unlock' : 'Lock'}
                    >
                      {block.locked ? <Lock className="w-3 h-3 text-slate-500" /> : <Unlock className="w-3 h-3 text-slate-500" />}
                    </button>
                    <button
                      onClick={(e) => { e.stopPropagation(); onRemoveBlock(block.instanceId); }}
                      className="p-1 hover:bg-red-100 dark:hover:bg-red-900/30 rounded-r-md"
                      aria-label="Remove block"
                      title="Remove"
                    >
                      <Trash2 className="w-3 h-3 text-red-500" />
                    </button>
                  </div>
                </>
              )}
            </div>
          );
        })}
      </div>
    </div>
  );
}

export default TemplateCanvas;
