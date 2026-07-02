// ═══════════════════════════════════════════════════════════════════════
// types.ts — Print Template Builder shared types
//
// UX-3.5: Visual drag-n-drop редактор печатных форм
// ═══════════════════════════════════════════════════════════════════════

import type { LucideIcon } from 'lucide-react';

/** Категория блоков в библиотеке */
export interface BlockCategory {
  id: string;
  label: string;
  icon: LucideIcon;
}

/** Определение блока (из библиотеки) */
export interface BlockDefinition {
  id: string;
  type: BlockType;
  category: string;
  label: string;
  icon: LucideIcon;
  defaultProps: Record<string, unknown>;
  description: string;
}

/** Тип блока на канвасе */
export type BlockType = 'text' | 'field' | 'table' | 'image' | 'signature' | 'qr' | 'divider' | 'spacer';

/** Экземпляр блока на канвасе (с уникальным ID) */
export interface CanvasBlock {
  instanceId: string;
  definitionId: string;
  type: BlockType;
  label: string;
  props: Record<string, unknown>;
  position: {
    x: number;
    y: number;
  };
  size: {
    width: number;
    height: number;
  };
  locked?: boolean;
  visible?: boolean;
}

/** Шаблон печатной формы */
export interface PrintTemplate {
  id: string;
  tenant_id: string;
  name: string;
  description?: string;
  blocks: CanvasBlock[];
  pageSettings: PageSettings;
  created_by: string;
  created_at: string;
  updated_at: string;
  version: number;
}

/** Настройки страницы */
export interface PageSettings {
  pageSize: 'a4' | 'a5' | 'letter' | 'custom';
  orientation: 'portrait' | 'landscape';
  margins: {
    top: number;
    bottom: number;
    left: number;
    right: number;
  };
  customWidth?: number;
  customHeight?: number;
}

/** Колонка таблицы */
export interface TableColumn {
  key: string;
  label: string;
  width: number | 'auto';
  alignment?: 'left' | 'center' | 'right';
  format?: string;
}

/** Тип пропсов для каждого типа блока */
export interface TextBlockProps {
  content: string;
  fontSize: number;
  fontWeight: 'normal' | 'bold' | 'light';
  alignment: 'left' | 'center' | 'right';
  color: string;
}

export interface FieldBlockProps {
  field: string;
  label: string;
  fontSize: number;
  showLabel: boolean;
  format?: string;
}

export interface TableBlockProps {
  dataSource: string;
  columns: TableColumn[];
  showHeader: boolean;
  bordered: boolean;
  fontSize: number;
}

export interface ImageBlockProps {
  src: string;
  alt: string;
  width: number;
  height: number;
  fit: 'contain' | 'cover' | 'fill';
  dataSource?: string;
  columns?: number;
  maxPhotos?: number;
}

export interface SignatureBlockProps {
  label: string;
  showDate: boolean;
  lineWidth: number;
  showHash?: boolean;
  algorithm?: string;
}

export interface QRBlockProps {
  dataSource: string;
  size: number;
  label: string;
  includeHash?: boolean;
}

/** Состояние редактора */
export interface EditorState {
  blocks: CanvasBlock[];
  selectedBlockId: string | null;
  pageSettings: PageSettings;
  isDragging: boolean;
  zoom: number;
  unsavedChanges: boolean;
}

/** Действия редактора */
export type EditorAction =
  | { type: 'ADD_BLOCK'; block: CanvasBlock }
  | { type: 'REMOVE_BLOCK'; instanceId: string }
  | { type: 'SELECT_BLOCK'; instanceId: string | null }
  | { type: 'MOVE_BLOCK'; instanceId: string; x: number; y: number }
  | { type: 'RESIZE_BLOCK'; instanceId: string; width: number; height: number }
  | { type: 'UPDATE_BLOCK_PROPS'; instanceId: string; props: Record<string, unknown> }
  | { type: 'REORDER_BLOCKS'; blocks: CanvasBlock[] }
  | { type: 'SET_PAGE_SETTINGS'; settings: PageSettings }
  | { type: 'SET_ZOOM'; zoom: number }
  | { type: 'SET_DRAGGING'; isDragging: boolean }
  | { type: 'LOAD_TEMPLATE'; blocks: CanvasBlock[]; settings: PageSettings }
  | { type: 'CLEAR_SELECTION' };
