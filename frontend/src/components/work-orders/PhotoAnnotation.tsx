// ═══════════════════════════════════════════════════════════════════════
// PhotoAnnotation — Advanced Photo Annotation Component
// P1-PHOTO: Freehand drawing, text labels, blur/redact, measurement,
//           layer management (undo/redo), and export as PNG.
// ═══════════════════════════════════════════════════════════════════════

import React, { useRef, useState, useEffect, useCallback } from 'react';
import {
  Camera,
  Save,
  RotateCcw,
  Trash2,
  Type,
  Minus,
  Square,
  Circle,
  EyeOff,
  Ruler,
  Pen,
} from '../ui/Icons';

// ── Types ──────────────────────────────────────────────────────────────

export type AnnotationTool =
  | 'arrow'
  | 'freehand'
  | 'text'
  | 'highlight'
  | 'circle'
  | 'blur'
  | 'measurement';

export interface Point {
  x: number;
  y: number;
}

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

export interface LayerState {
  elements: AnnotationElement[];
  currentIndex: number; // points to last applied element
}

// ── Constants ──────────────────────────────────────────────────────────

const COLORS = [
  '#ef4444', '#f97316', '#eab308', '#22c55e',
  '#3b82f6', '#8b5cf6', '#ec4899', '#ffffff', '#000000',
];

const STROKE_WIDTHS = [2, 4, 6, 10];

const DEFAULT_DPI = 96; // pixels per inch for measurement estimation
const MM_PER_INCH = 25.4;

const TOOL_ICONS: Record<AnnotationTool, React.ReactNode> = {
  arrow: <Minus className="w-4 h-4 rotate-45" />,
  freehand: <Pen className="w-4 h-4" />,
  text: <Type className="w-4 h-4" />,
  highlight: <Square className="w-4 h-4" />,
  circle: <Circle className="w-4 h-4" />,
  blur: <EyeOff className="w-4 h-4" />,
  measurement: <Ruler className="w-4 h-4" />,
};

const TOOL_LABELS: Record<AnnotationTool, string> = {
  arrow: 'Стрелка',
  freehand: 'Рисование',
  text: 'Текст',
  highlight: 'Выделение',
  circle: 'Круг',
  blur: 'Размытие',
  measurement: 'Линейка',
};

// ── Helpers ────────────────────────────────────────────────────────────

let elementIdCounter = 0;
function nextId(): string {
  elementIdCounter += 1;
  return `el-${elementIdCounter}-${Date.now()}`;
}

function distance(a: Point, b: Point): number {
  return Math.sqrt((b.x - a.x) ** 2 + (b.y - a.y) ** 2);
}

function pxToMm(px: number, dpi: number = DEFAULT_DPI): number {
  return (px / dpi) * MM_PER_INCH;
}

function drawArrowhead(
  ctx: CanvasRenderingContext2D,
  from: Point,
  to: Point,
  size: number,
): void {
  const angle = Math.atan2(to.y - from.y, to.x - from.x);
  ctx.beginPath();
  ctx.moveTo(to.x, to.y);
  ctx.lineTo(
    to.x - size * Math.cos(angle - Math.PI / 6),
    to.y - size * Math.sin(angle - Math.PI / 6),
  );
  ctx.lineTo(
    to.x - size * Math.cos(angle + Math.PI / 6),
    to.y - size * Math.sin(angle + Math.PI / 6),
  );
  ctx.closePath();
  ctx.fill();
}

// ── Props ──────────────────────────────────────────────────────────────

interface PhotoAnnotationProps {
  imageUrl: string;
  onSave?: (dataUrl: string) => void;
  readOnly?: boolean;
  className?: string;
}

// ── Component ──────────────────────────────────────────────────────────

export const PhotoAnnotation: React.FC<PhotoAnnotationProps> = ({
  imageUrl,
  onSave,
  readOnly = false,
  className = '',
}) => {
  const canvasRef = useRef<HTMLCanvasElement>(null);
  const containerRef = useRef<HTMLDivElement>(null);
  const imgRef = useRef<HTMLImageElement | null>(null);

  // ── State ──────────────────────────────────────────────────────────
  const [imageLoaded, setImageLoaded] = useState(false);
  const [imageSize, setImageSize] = useState({ w: 800, h: 600 });
  const [currentTool, setCurrentTool] = useState<AnnotationTool>('arrow');
  const [currentColor, setCurrentColor] = useState(COLORS[0]);
  const [strokeWidth, setStrokeWidth] = useState(4);
  const [fontSize, setFontSize] = useState(18);

  // Layer management (undo/redo stack)
  const [redoStack, setRedoStack] = useState<AnnotationElement[][]>([]);
  const [elements, setElements] = useState<AnnotationElement[]>([]);

  // Drawing state (not persisted)
  const [isDrawing, setIsDrawing] = useState(false);
  const [drawStart, setDrawStart] = useState<Point | null>(null);
  const [currentPoints, setCurrentPoints] = useState<Point[]>([]);
  const [previewPoint, setPreviewPoint] = useState<Point | null>(null);

  // Text input state
  const [textInput, setTextInput] = useState<{
    position: Point;
    value: string;
  } | null>(null);
  const textInputRef = useRef<HTMLInputElement>(null);

  // Canvas scale for high-DPI
  const [canvasScale, setCanvasScale] = useState(1);

  // ── Image loading ──────────────────────────────────────────────────
  useEffect(() => {
    const img = new Image();
    img.crossOrigin = 'anonymous';
    img.onload = () => {
      imgRef.current = img;
      const maxW = containerRef.current?.clientWidth ?? 800;
      const displayW = Math.min(img.naturalWidth, maxW);
      const ratio = displayW / img.naturalWidth;
      const displayH = img.naturalHeight * ratio;
      setImageSize({ w: displayW, h: displayH });
      setImageLoaded(true);
    };
    img.src = imageUrl;
    return () => {
      img.onload = null;
    };
  }, [imageUrl]);

  // ── Canvas drawing ─────────────────────────────────────────────────
  const renderCanvas = useCallback(() => {
    const canvas = canvasRef.current;
    const img = imgRef.current;
    if (!canvas || !img) return;

    const ctx = canvas.getContext('2d');
    if (!ctx) return;

    const dpr = window.devicePixelRatio || 1;
    const { w, h } = imageSize;
    canvas.width = w * dpr;
    canvas.height = h * dpr;
    canvas.style.width = `${w}px`;
    canvas.style.height = `${h}px`;
    ctx.scale(dpr, dpr);
    setCanvasScale(dpr);

    // Draw the image
    ctx.drawImage(img, 0, 0, w, h);

    // Draw all elements
    for (const el of elements) {
      drawElement(ctx, el, false);
    }

    // Draw preview (in-progress element)
    if (isDrawing && drawStart && previewPoint) {
      drawPreview(ctx, currentTool, currentColor, strokeWidth, drawStart, previewPoint, currentPoints);
    }
  }, [elements, imageSize, isDrawing, drawStart, previewPoint, currentTool, currentColor, strokeWidth, currentPoints]);

  useEffect(() => {
    if (imageLoaded) renderCanvas();
  }, [renderCanvas, imageLoaded]);

  // ── Coordinate conversion ──────────────────────────────────────────
  const getCanvasCoords = useCallback(
    (clientX: number, clientY: number): Point => {
      const canvas = canvasRef.current;
      if (!canvas) return { x: 0, y: 0 };
      const rect = canvas.getBoundingClientRect();
      return {
        x: clientX - rect.left,
        y: clientY - rect.top,
      };
    },
    [],
  );

  // ── Drawing helpers ────────────────────────────────────────────────
  const drawElement = (
    ctx: CanvasRenderingContext2D,
    el: AnnotationElement,
    isPreview: boolean,
  ) => {
    ctx.save();
    ctx.strokeStyle = el.color;
    ctx.fillStyle = el.color;
    ctx.lineWidth = el.strokeWidth;
    ctx.lineCap = 'round';
    ctx.lineJoin = 'round';

    switch (el.type) {
      case 'arrow': {
        const arrow = el as ArrowElement;
        ctx.beginPath();
        ctx.moveTo(arrow.start.x, arrow.start.y);
        ctx.lineTo(arrow.end.x, arrow.end.y);
        ctx.stroke();
        drawArrowhead(ctx, arrow.start, arrow.end, 12);
        break;
      }

      case 'freehand': {
        const fh = el as FreehandElement;
        if (fh.points.length < 2) break;
        ctx.beginPath();
        ctx.moveTo(fh.points[0].x, fh.points[0].y);
        for (let i = 1; i < fh.points.length; i++) {
          ctx.lineTo(fh.points[i].x, fh.points[i].y);
        }
        ctx.stroke();
        break;
      }

      case 'text': {
        const txt = el as TextElement;
        ctx.font = `bold ${txt.fontSize}px Inter, system-ui, sans-serif`;
        const metrics = ctx.measureText(txt.text);
        const pad = 6;
        // Background
        ctx.fillStyle = 'rgba(0,0,0,0.65)';
        const bgX = txt.position.x;
        const bgY = txt.position.y - txt.fontSize - pad;
        roundRect(ctx, bgX, bgY, metrics.width + pad * 2, txt.fontSize + pad * 2, 4);
        ctx.fill();
        // Text
        ctx.fillStyle = '#ffffff';
        ctx.fillText(txt.text, txt.position.x + pad, txt.position.y - pad);
        break;
      }

      case 'highlight': {
        const hl = el as HighlightElement;
        ctx.globalAlpha = 0.3;
        ctx.fillStyle = el.color;
        ctx.fillRect(
          Math.min(hl.start.x, hl.end.x),
          Math.min(hl.start.y, hl.end.y),
          Math.abs(hl.end.x - hl.start.x),
          Math.abs(hl.end.y - hl.start.y),
        );
        break;
      }

      case 'circle': {
        const cir = el as CircleElement;
        ctx.beginPath();
        ctx.arc(cir.center.x, cir.center.y, cir.radius, 0, Math.PI * 2);
        ctx.stroke();
        ctx.globalAlpha = 0.2;
        ctx.fillStyle = el.color;
        ctx.fill();
        break;
      }

      case 'blur': {
        const bl = el as BlurElement;
        const bx = Math.min(bl.start.x, bl.end.x);
        const by = Math.min(bl.start.y, bl.end.y);
        const bw = Math.abs(bl.end.x - bl.start.x);
        const bh = Math.abs(bl.end.y - bl.start.y);
        ctx.filter = `blur(${Math.max(4, el.strokeWidth * 3)}px)`;
        ctx.fillStyle = '#000000';
        ctx.globalAlpha = 0.85;
        ctx.fillRect(bx, by, bw, bh);
        ctx.filter = 'none';
        // Border
        ctx.strokeStyle = '#ffffff';
        ctx.lineWidth = 1.5;
        ctx.setLineDash([4, 4]);
        ctx.strokeRect(bx, by, bw, bh);
        ctx.setLineDash([]);
        // Label
        ctx.fillStyle = 'rgba(255,255,255,0.85)';
        ctx.font = '11px Inter, system-ui, sans-serif';
        ctx.fillText('⬡ redacted', bx + 4, by + 14);
        break;
      }

      case 'measurement': {
        const m = el as MeasurementElement;
        ctx.strokeStyle = '#22d3ee';
        ctx.lineWidth = el.strokeWidth;
        ctx.setLineDash([6, 4]);
        ctx.beginPath();
        ctx.moveTo(m.start.x, m.start.y);
        ctx.lineTo(m.end.x, m.end.y);
        ctx.stroke();
        ctx.setLineDash([]);
        // Endpoints
        ctx.fillStyle = '#22d3ee';
        ctx.beginPath();
        ctx.arc(m.start.x, m.start.y, 5, 0, Math.PI * 2);
        ctx.fill();
        ctx.beginPath();
        ctx.arc(m.end.x, m.end.y, 5, 0, Math.PI * 2);
        ctx.fill();
        // Length label
        const midX = (m.start.x + m.end.x) / 2;
        const midY = (m.start.y + m.end.y) / 2;
        const label = `${m.lengthPx.toFixed(0)}px (${pxToMm(m.lengthPx).toFixed(1)}mm)`;
        ctx.font = 'bold 12px Inter, system-ui, sans-serif';
        const mw = ctx.measureText(label).width;
        ctx.fillStyle = 'rgba(0,0,0,0.75)';
        roundRect(ctx, midX - mw / 2 - 6, midY - 12, mw + 12, 24, 4);
        ctx.fill();
        ctx.fillStyle = '#22d3ee';
        ctx.fillText(label, midX - mw / 2, midY + 4);
        break;
      }
    }

    ctx.restore();
  };

  const drawPreview = (
    ctx: CanvasRenderingContext2D,
    tool: AnnotationTool,
    color: string,
    sw: number,
    start: Point,
    end: Point,
    points: Point[],
  ) => {
    ctx.save();
    ctx.strokeStyle = color;
    ctx.fillStyle = color;
    ctx.lineWidth = sw;
    ctx.lineCap = 'round';
    ctx.lineJoin = 'round';
    ctx.globalAlpha = 0.7;

    switch (tool) {
      case 'arrow': {
        ctx.beginPath();
        ctx.moveTo(start.x, start.y);
        ctx.lineTo(end.x, end.y);
        ctx.stroke();
        drawArrowhead(ctx, start, end, 12);
        break;
      }
      case 'freehand': {
        if (points.length < 2) break;
        ctx.beginPath();
        ctx.moveTo(points[0].x, points[0].y);
        for (let i = 1; i < points.length; i++) {
          ctx.lineTo(points[i].x, points[i].y);
        }
        ctx.stroke();
        break;
      }
      case 'highlight': {
        ctx.globalAlpha = 0.25;
        ctx.fillStyle = color;
        ctx.fillRect(
          Math.min(start.x, end.x),
          Math.min(start.y, end.y),
          Math.abs(end.x - start.x),
          Math.abs(end.y - start.y),
        );
        break;
      }
      case 'circle': {
        const r = distance(start, end);
        ctx.beginPath();
        ctx.arc(start.x, start.y, r, 0, Math.PI * 2);
        ctx.stroke();
        ctx.globalAlpha = 0.15;
        ctx.fill();
        break;
      }
      case 'blur': {
        ctx.globalAlpha = 0.4;
        ctx.fillStyle = '#000';
        ctx.fillRect(
          Math.min(start.x, end.x),
          Math.min(start.y, end.y),
          Math.abs(end.x - start.x),
          Math.abs(end.y - start.y),
        );
        break;
      }
      case 'measurement': {
        ctx.strokeStyle = '#22d3ee';
        ctx.setLineDash([6, 4]);
        ctx.beginPath();
        ctx.moveTo(start.x, start.y);
        ctx.lineTo(end.x, end.y);
        ctx.stroke();
        ctx.setLineDash([]);
        ctx.fillStyle = '#22d3ee';
        ctx.beginPath();
        ctx.arc(start.x, start.y, 5, 0, Math.PI * 2);
        ctx.fill();
        ctx.beginPath();
        ctx.arc(end.x, end.y, 5, 0, Math.PI * 2);
        ctx.fill();
        break;
      }
    }

    ctx.restore();
  };

  // ── Pointer handlers ───────────────────────────────────────────────
  const handlePointerDown = useCallback(
    (e: React.PointerEvent<HTMLCanvasElement>) => {
      if (readOnly) return;
      const pos = getCanvasCoords(e.clientX, e.clientY);
      (e.target as HTMLCanvasElement).setPointerCapture(e.pointerId);

      if (currentTool === 'text') {
        setTextInput({ position: pos, value: '' });
        return;
      }

      setIsDrawing(true);
      setDrawStart(pos);
      setPreviewPoint(pos);
      setCurrentPoints(currentTool === 'freehand' ? [pos] : []);
    },
    [readOnly, currentTool, getCanvasCoords],
  );

  const handlePointerMove = useCallback(
    (e: React.PointerEvent<HTMLCanvasElement>) => {
      if (!isDrawing || readOnly) return;
      const pos = getCanvasCoords(e.clientX, e.clientY);

      if (currentTool === 'freehand') {
        setCurrentPoints((prev) => [...prev, pos]);
      }

      setPreviewPoint(pos);
    },
    [isDrawing, readOnly, currentTool, getCanvasCoords],
  );

  const commitElement = useCallback(
    (el: AnnotationElement) => {
      setElements((prev) => [...prev, el]);
      setRedoStack([]);
    },
    [],
  );

  const handlePointerUp = useCallback(
    (e: React.PointerEvent<HTMLCanvasElement>) => {
      if (!isDrawing || readOnly || !drawStart || !previewPoint) {
        setIsDrawing(false);
        setDrawStart(null);
        setPreviewPoint(null);
        setCurrentPoints([]);
        return;
      }

      const end = getCanvasCoords(e.clientX, e.clientY);
      const dist = distance(drawStart, end);

      // Ignore tiny clicks (< 3px)
      if (dist < 3 && currentTool !== 'freehand') {
        setIsDrawing(false);
        setDrawStart(null);
        setPreviewPoint(null);
        setCurrentPoints([]);
        return;
      }

      let el: AnnotationElement | null = null;

      switch (currentTool) {
        case 'arrow':
          el = {
            id: nextId(),
            type: 'arrow',
            color: currentColor,
            strokeWidth,
            start: drawStart,
            end,
          };
          break;
        case 'freehand':
          el = {
            id: nextId(),
            type: 'freehand',
            color: currentColor,
            strokeWidth,
            points: currentPoints.length > 0 ? currentPoints : [drawStart, end],
          };
          break;
        case 'highlight':
          el = {
            id: nextId(),
            type: 'highlight',
            color: currentColor,
            strokeWidth,
            start: drawStart,
            end,
          };
          break;
        case 'circle':
          el = {
            id: nextId(),
            type: 'circle',
            color: currentColor,
            strokeWidth,
            center: drawStart,
            radius: dist,
          };
          break;
        case 'blur':
          el = {
            id: nextId(),
            type: 'blur',
            color: currentColor,
            strokeWidth,
            start: drawStart,
            end,
          };
          break;
        case 'measurement':
          el = {
            id: nextId(),
            type: 'measurement',
            color: currentColor,
            strokeWidth,
            start: drawStart,
            end,
            lengthPx: dist,
          };
          break;
      }

      if (el) commitElement(el);

      setIsDrawing(false);
      setDrawStart(null);
      setPreviewPoint(null);
      setCurrentPoints([]);
    },
    [
      isDrawing, readOnly, drawStart, previewPoint, currentTool,
      currentColor, strokeWidth, currentPoints, getCanvasCoords, commitElement,
    ],
  );

  // ── Text input handlers ────────────────────────────────────────────
  const handleTextConfirm = useCallback(() => {
    if (!textInput || !textInput.value.trim()) {
      setTextInput(null);
      return;
    }

    const el: TextElement = {
      id: nextId(),
      type: 'text',
      color: currentColor,
      strokeWidth,
      position: textInput.position,
      text: textInput.value.trim(),
      fontSize,
    };

    commitElement(el);
    setTextInput(null);
  }, [textInput, currentColor, strokeWidth, fontSize, commitElement]);

  const handleTextKeyDown = useCallback(
    (e: React.KeyboardEvent<HTMLInputElement>) => {
      if (e.key === 'Enter') {
        e.preventDefault();
        handleTextConfirm();
      }
      if (e.key === 'Escape') {
        setTextInput(null);
      }
    },
    [handleTextConfirm],
  );

  // Focus text input when it appears
  useEffect(() => {
    if (textInput) {
      setTimeout(() => textInputRef.current?.focus(), 50);
    }
  }, [textInput]);

  // ── Actions ────────────────────────────────────────────────────────
  const handleUndo = useCallback(() => {
    if (elements.length === 0) return;
    const lastEl = elements[elements.length - 1];
    setRedoStack((prev) => [...prev, [lastEl]]);
    setElements((prev) => prev.slice(0, -1));
  }, [elements]);

  const handleRedo = useCallback(() => {
    if (redoStack.length === 0) return;
    const redoEls = redoStack[redoStack.length - 1];
    setRedoStack((prev) => prev.slice(0, -1));
    setElements((prev) => [...prev, ...redoEls]);
  }, [redoStack]);

  const handleClear = useCallback(() => {
    if (elements.length === 0) return;
    setRedoStack((prev) => [...prev, [...elements]]);
    setElements([]);
  }, [elements]);

  const handleExport = useCallback(() => {
    const canvas = canvasRef.current;
    if (!canvas) return;

    const dataUrl = canvas.toDataURL('image/png');
    const link = document.createElement('a');
    link.download = `annotation-${Date.now()}.png`;
    link.href = dataUrl;
    link.click();
    onSave?.(dataUrl);
  }, [onSave]);

  // ── JSX ────────────────────────────────────────────────────────────
  return (
    <div
      ref={containerRef}
      className={`space-y-3 ${className}`}
    >
      {/* Canvas area */}
      <div className="relative rounded-lg overflow-hidden border border-slate-200 dark:border-slate-700 bg-slate-100 dark:bg-slate-900">
        {!imageLoaded && (
          <div className="flex items-center justify-center h-64 text-slate-400">
            <Camera className="w-8 h-8 animate-pulse" />
          </div>
        )}

        <canvas
          ref={canvasRef}
          onPointerDown={handlePointerDown}
          onPointerMove={handlePointerMove}
          onPointerUp={handlePointerUp}
          onPointerLeave={() => {
            if (isDrawing) {
              setIsDrawing(false);
              setDrawStart(null);
              setPreviewPoint(null);
              setCurrentPoints([]);
            }
          }}
          className={`w-full h-auto ${
            readOnly ? 'cursor-default' : 'cursor-crosshair'
          }`}
          style={{
            display: imageLoaded ? 'block' : 'none',
            touchAction: 'none',
          }}
        />

        {/* Inline text input overlay */}
        {textInput && (
          <input
            ref={textInputRef}
            type="text"
            value={textInput.value}
            onChange={(e) =>
              setTextInput({ ...textInput, value: e.target.value })
            }
            onKeyDown={handleTextKeyDown}
            onBlur={handleTextConfirm}
            placeholder="Введите текст..."
            className="absolute z-10 px-2 py-1 text-sm text-white bg-slate-900/80 border border-blue-500 rounded outline-none placeholder:text-slate-400"
            style={{
              left: textInput.position.x,
              top: textInput.position.y - 28,
              minWidth: 120,
            }}
          />
        )}
      </div>

      {/* Toolbar — only shown when not readOnly */}
      {!readOnly && (
        <div className="flex flex-wrap items-center gap-2">
          {/* Tool selection */}
          <div className="flex items-center gap-1 p-1 bg-slate-100 dark:bg-slate-800 rounded-lg">
            {(Object.keys(TOOL_ICONS) as AnnotationTool[]).map((tool) => (
              <button
                key={tool}
                onClick={() => setCurrentTool(tool)}
                title={TOOL_LABELS[tool]}
                className={`p-1.5 rounded-md transition-colors ${
                  currentTool === tool
                    ? 'bg-blue-600 text-white shadow-sm'
                    : 'text-slate-500 dark:text-slate-400 hover:bg-slate-200 dark:hover:bg-slate-700'
                }`}
              >
                {TOOL_ICONS[tool]}
              </button>
            ))}
          </div>

          {/* Separator */}
          <div className="w-px h-6 bg-slate-300 dark:bg-slate-600" />

          {/* Colors */}
          <div className="flex items-center gap-1">
            {COLORS.map((color) => (
              <button
                key={color}
                onClick={() => setCurrentColor(color)}
                title={color}
                className={`w-5 h-5 rounded-full border-2 transition-all ${
                  currentColor === color
                    ? 'border-slate-900 dark:border-white scale-125'
                    : 'border-transparent'
                }`}
                style={{ backgroundColor: color }}
              />
            ))}
          </div>

          {/* Separator */}
          <div className="w-px h-6 bg-slate-300 dark:bg-slate-600" />

          {/* Stroke width */}
          <div className="flex items-center gap-1">
            {STROKE_WIDTHS.map((sw) => (
              <button
                key={sw}
                onClick={() => setStrokeWidth(sw)}
                title={`${sw}px`}
                className={`p-1.5 rounded-md transition-colors ${
                  strokeWidth === sw
                    ? 'bg-blue-100 dark:bg-blue-900/40'
                    : 'hover:bg-slate-100 dark:hover:bg-slate-700'
                }`}
              >
                <div
                  className="bg-slate-700 dark:bg-slate-300 rounded-full"
                  style={{ width: sw + 4, height: sw + 4 }}
                />
              </button>
            ))}
          </div>

          {/* Separator */}
          <div className="w-px h-6 bg-slate-300 dark:bg-slate-600" />

          {/* Font size (only for text tool) */}
          {currentTool === 'text' && (
            <>
              <div className="flex items-center gap-1">
                {[14, 18, 24, 32].map((fs) => (
                  <button
                    key={fs}
                    onClick={() => setFontSize(fs)}
                    title={`${fs}px`}
                    className={`px-2 py-1 text-xs rounded-md transition-colors ${
                      fontSize === fs
                        ? 'bg-blue-100 dark:bg-blue-900/40 font-bold'
                        : 'hover:bg-slate-100 dark:hover:bg-slate-700'
                    }`}
                  >
                    {fs}
                  </button>
                ))}
              </div>
              <div className="w-px h-6 bg-slate-300 dark:bg-slate-600" />
            </>
          )}

          {/* Undo / Redo */}
          <button
            onClick={handleUndo}
            disabled={elements.length === 0}
            title="Отменить (Ctrl+Z)"
            className="p-1.5 rounded-md text-slate-500 dark:text-slate-400 hover:bg-slate-100 dark:hover:bg-slate-700 disabled:opacity-30 disabled:cursor-not-allowed"
          >
            <RotateCcw className="w-4 h-4" />
          </button>

          <button
            onClick={handleRedo}
            disabled={redoStack.length === 0}
            title="Повторить (Ctrl+Shift+Z)"
            className="p-1.5 rounded-md text-slate-500 dark:text-slate-400 hover:bg-slate-100 dark:hover:bg-slate-700 disabled:opacity-30 disabled:cursor-not-allowed"
          >
            <RotateCcw className="w-4 h-4 scale-x-[-1]" />
          </button>

          {/* Clear */}
          <button
            onClick={handleClear}
            disabled={elements.length === 0}
            title="Очистить все"
            className="p-1.5 rounded-md text-red-500 hover:bg-red-50 dark:hover:bg-red-900/20 disabled:opacity-30 disabled:cursor-not-allowed"
          >
            <Trash2 className="w-4 h-4" />
          </button>

          {/* Spacer */}
          <div className="flex-1" />

          {/* Layer count */}
          <span className="text-xs text-slate-400 dark:text-slate-500">
            {elements.length} слой{elements.length !== 1 ? 'ев' : ''}
            {redoStack.length > 0 && ` (+${redoStack.length} отм.)`}
          </span>

          {/* Export */}
          <button
            onClick={handleExport}
            disabled={elements.length === 0}
            title="Экспорт PNG"
            className="px-3 py-1.5 text-xs font-medium rounded-lg bg-emerald-600 text-white hover:bg-emerald-700 disabled:opacity-40 disabled:cursor-not-allowed flex items-center gap-1"
          >
            <Save className="w-3.5 h-3.5" />
            Экспорт
          </button>
        </div>
      )}
    </div>
  );
};

// ── Utility: rounded rectangle ─────────────────────────────────────────

function roundRect(
  ctx: CanvasRenderingContext2D,
  x: number,
  y: number,
  w: number,
  h: number,
  r: number,
): void {
  ctx.beginPath();
  ctx.moveTo(x + r, y);
  ctx.lineTo(x + w - r, y);
  ctx.arcTo(x + w, y, x + w, y + r, r);
  ctx.lineTo(x + w, y + h - r);
  ctx.arcTo(x + w, y + h, x + w - r, y + h, r);
  ctx.lineTo(x + r, y + h);
  ctx.arcTo(x, y + h, x, y + h - r, r);
  ctx.lineTo(x, y + r);
  ctx.arcTo(x, y, x + r, y, r);
  ctx.closePath();
}
