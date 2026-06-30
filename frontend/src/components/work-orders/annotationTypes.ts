// ═══════════════════════════════════════════════════════════════════════
// annotationTypes.ts — Shared types for PhotoAnnotation components
// P1-PHOTO: Annotation element types, tool definitions, and layer state
// ═══════════════════════════════════════════════════════════════════════

// ── Point ───────────────────────────────────────────────────────────────

export interface Point {
  x: number;
  y: number;
}

// ── Tool ────────────────────────────────────────────────────────────────

/** Available annotation drawing tools */
export type AnnotationTool =
  | 'arrow'
  | 'freehand'
  | 'text'
  | 'highlight'
  | 'circle'
  | 'blur'
  | 'measurement';

// ── Element types ───────────────────────────────────────────────────────

interface BaseElement {
  id: string;
  type: AnnotationTool;
  color: string;
  strokeWidth: number;
}

export interface ArrowElement extends BaseElement {
  type: 'arrow';
  start: Point;
  end: Point;
}

export interface FreehandElement extends BaseElement {
  type: 'freehand';
  points: Point[];
}

export interface TextElement extends BaseElement {
  type: 'text';
  position: Point;
  text: string;
  fontSize: number;
}

export interface HighlightElement extends BaseElement {
  type: 'highlight';
  start: Point;
  end: Point;
}

export interface CircleElement extends BaseElement {
  type: 'circle';
  center: Point;
  radius: number;
}

export interface BlurElement extends BaseElement {
  type: 'blur';
  start: Point;
  end: Point;
}

export interface MeasurementElement extends BaseElement {
  type: 'measurement';
  start: Point;
  end: Point;
  lengthPx: number;
}

export type AnnotationElement =
  | ArrowElement
  | FreehandElement
  | TextElement
  | HighlightElement
  | CircleElement
  | BlurElement
  | MeasurementElement;

// ── Layer management ────────────────────────────────────────────────────

export interface LayerState {
  elements: AnnotationElement[];
  currentIndex: number;
}

// ── API types ───────────────────────────────────────────────────────────

export interface AnnotationSaveRequest {
  elements: AnnotationElement[];
}

export interface AnnotationResponse {
  id: string;
  work_order_id: string;
  photo_url: string;
  elements: AnnotationElement[];
  created_by: string;
  created_at: string;
  updated_at: string;
}

// ── Constants ───────────────────────────────────────────────────────────

export const COLORS = [
  '#ef4444', // Red — defect
  '#f97316', // Orange — warning
  '#eab308', // Yellow — caution
  '#22c55e', // Green — ok
  '#3b82f6', // Blue — note
  '#8b5cf6', // Purple
  '#ec4899', // Pink
  '#ffffff', // White
  '#000000', // Black
];

export const STROKE_WIDTHS = [2, 4, 6, 10];

export const DEFAULT_DPI = 96;
export const MM_PER_INCH = 25.4;

// ── Tool definitions ────────────────────────────────────────────────────

export interface ToolDef {
  key: AnnotationTool;
  label: string;
}

export const TOOLS: ToolDef[] = [
  { key: 'arrow', label: 'Стрелка' },
  { key: 'freehand', label: 'Рисование' },
  { key: 'text', label: 'Текст' },
  { key: 'highlight', label: 'Выделение' },
  { key: 'circle', label: 'Круг' },
  { key: 'blur', label: 'Размытие' },
  { key: 'measurement', label: 'Линейка' },
];

// ── Helpers ─────────────────────────────────────────────────────────────

let elementIdCounter = 0;

export function nextId(): string {
  elementIdCounter += 1;
  return `el-${elementIdCounter}-${Date.now()}`;
}

export function distance(a: Point, b: Point): number {
  return Math.sqrt((b.x - a.x) ** 2 + (b.y - a.y) ** 2);
}

export function pxToMm(px: number, dpi: number = DEFAULT_DPI): number {
  return (px / dpi) * MM_PER_INCH;
}
