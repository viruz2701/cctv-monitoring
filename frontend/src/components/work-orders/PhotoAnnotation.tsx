// ═══════════════════════════════════════════════════════════════════════
// PhotoAnnotation — Advanced Photo Annotation Component
// P1-PHOTO: Canvas-based annotation with Rectangle, Circle, Arrow,
//           Text, Freehand, measurement tools, undo/redo, zoom, export
// ═══════════════════════════════════════════════════════════════════════

import React, { useRef, useState, useEffect, useCallback } from 'react';
import { Camera } from '../ui/Icons';
import { PhotoAnnotationToolbar } from './PhotoAnnotationToolbar';
import { useAnnotationKeyboard } from './useAnnotationKeyboard';
import {
  type AnnotationElement,
  type AnnotationTool,
  type Point,
  type ArrowElement,
  type FreehandElement,
  type HighlightElement,
  type CircleElement,
  type BlurElement,
  type MeasurementElement,
  type TextElement,
  nextId,
  distance,
} from './annotationTypes';
import { drawElement, drawPreview, exportToPng } from './annotationCanvas';

// ── Props ───────────────────────────────────────────────────────────────

interface PhotoAnnotationProps {
  imageUrl: string;
  onSave?: (dataUrl: string) => void;
  onElementsChange?: (elements: AnnotationElement[]) => void;
  readOnly?: boolean;
  className?: string;
  username?: string;
  initialElements?: AnnotationElement[];
}

// ── Component ───────────────────────────────────────────────────────────

export const PhotoAnnotation: React.FC<PhotoAnnotationProps> = ({
  imageUrl,
  onSave,
  onElementsChange,
  readOnly = false,
  className = '',
  username,
  initialElements,
}) => {
  const canvasRef = useRef<HTMLCanvasElement>(null);
  const containerRef = useRef<HTMLDivElement>(null);
  const imgRef = useRef<HTMLImageElement | null>(null);

  // ── State ──────────────────────────────────────────────────────────
  const [imageLoaded, setImageLoaded] = useState(false);
  const [imageSize, setImageSize] = useState({ w: 800, h: 600 });
  const [currentTool, setCurrentTool] = useState<AnnotationTool>('arrow');
  const [currentColor, setCurrentColor] = useState('#ef4444');
  const [strokeWidth, setStrokeWidth] = useState(4);
  const [fontSize, setFontSize] = useState(18);
  const [isExporting, setIsExporting] = useState(false);

  // Layer management (undo/redo stack)
  const [redoStack, setRedoStack] = useState<AnnotationElement[][]>([]);
  const [elements, setElements] = useState<AnnotationElement[]>(initialElements ?? []);

  // Drawing state
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

  // Zoom state
  const [zoom, setZoom] = useState(1);
  const [panOffset, setPanOffset] = useState({ x: 0, y: 0 });
  const [isPanning, setIsPanning] = useState(false);
  const [panStart, setPanStart] = useState<Point | null>(null);

  // Notification elements change
  useEffect(() => {
    onElementsChange?.(elements);
  }, [elements, onElementsChange]);

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

    // Guard: некоторые методы Canvas могут отсутствовать в тестовом окружении (jsdom)
    if (typeof ctx.save !== 'function') return;

    const dpr = window.devicePixelRatio || 1;
    const { w, h } = imageSize;
    canvas.width = w * dpr;
    canvas.height = h * dpr;
    canvas.style.width = `${w}px`;
    canvas.style.height = `${h}px`;

    try {
      ctx.scale(dpr, dpr);

      // Apply zoom and pan
      ctx.save();
      ctx.translate(panOffset.x, panOffset.y);
      ctx.scale(zoom, zoom);

      // Draw the image
      ctx.drawImage(img, 0, 0, w, h);

      // Draw all elements
      for (const el of elements) {
        drawElement(ctx, el);
      }

      // Draw preview (in-progress element)
      if (isDrawing && drawStart && previewPoint) {
        drawPreview(
          ctx,
          currentTool,
          currentColor,
          strokeWidth,
          drawStart,
          previewPoint,
          currentPoints,
        );
      }

      ctx.restore();
    } catch {
      // Canvas rendering error in test environment — silently ignore
    }
  }, [
    elements, imageSize, isDrawing, drawStart, previewPoint,
    currentTool, currentColor, strokeWidth, currentPoints, zoom, panOffset,
  ]);

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
        x: (clientX - rect.left - panOffset.x) / zoom,
        y: (clientY - rect.top - panOffset.y) / zoom,
      };
    },
    [panOffset, zoom],
  );

  // ── Pointer handlers ───────────────────────────────────────────────
  const handlePointerDown = useCallback(
    (e: React.PointerEvent<HTMLCanvasElement>) => {
      if (readOnly) return;

      // Middle mouse button or Space+click for panning
      if (e.button === 1) {
        setIsPanning(true);
        setPanStart({ x: e.clientX, y: e.clientY });
        return;
      }

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
      if (isPanning && panStart) {
        const dx = e.clientX - panStart.x;
        const dy = e.clientY - panStart.y;
        setPanOffset((prev) => ({ x: prev.x + dx, y: prev.y + dy }));
        setPanStart({ x: e.clientX, y: e.clientY });
        return;
      }

      if (!isDrawing || readOnly) return;
      const pos = getCanvasCoords(e.clientX, e.clientY);

      if (currentTool === 'freehand') {
        setCurrentPoints((prev) => [...prev, pos]);
      }

      setPreviewPoint(pos);
    },
    [isDrawing, readOnly, currentTool, getCanvasCoords, isPanning, panStart],
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
      if (isPanning) {
        setIsPanning(false);
        setPanStart(null);
        return;
      }

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
          } as ArrowElement;
          break;
        case 'freehand':
          el = {
            id: nextId(),
            type: 'freehand',
            color: currentColor,
            strokeWidth,
            points: currentPoints.length > 0 ? currentPoints : [drawStart, end],
          } as FreehandElement;
          break;
        case 'highlight':
          el = {
            id: nextId(),
            type: 'highlight',
            color: currentColor,
            strokeWidth,
            start: drawStart,
            end,
          } as HighlightElement;
          break;
        case 'circle':
          el = {
            id: nextId(),
            type: 'circle',
            color: currentColor,
            strokeWidth,
            center: drawStart,
            radius: dist,
          } as CircleElement;
          break;
        case 'blur':
          el = {
            id: nextId(),
            type: 'blur',
            color: currentColor,
            strokeWidth,
            start: drawStart,
            end,
          } as BlurElement;
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
          } as MeasurementElement;
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
      currentColor, strokeWidth, currentPoints, getCanvasCoords,
      commitElement, isPanning,
    ],
  );

  // ── Wheel handler for zoom ─────────────────────────────────────────
  const handleWheel = useCallback(
    (e: React.WheelEvent<HTMLCanvasElement>) => {
      if (!e.ctrlKey && !e.metaKey) return;
      e.preventDefault();

      const delta = e.deltaY > 0 ? -0.1 : 0.1;
      setZoom((prev) => Math.max(0.1, Math.min(5, prev + delta)));
    },
    [],
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

  const handleEscape = useCallback(() => {
    if (textInput) {
      setTextInput(null);
    }
    if (isDrawing) {
      setIsDrawing(false);
      setDrawStart(null);
      setPreviewPoint(null);
      setCurrentPoints([]);
    }
  }, [textInput, isDrawing]);

  const handleExport = useCallback(async () => {
    const canvas = canvasRef.current;
    if (!canvas) return;

    setIsExporting(true);
    try {
      const dataUrl = await exportToPng(canvas, username);
      const link = document.createElement('a');
      link.download = `annotation-${Date.now()}.png`;
      link.href = dataUrl;
      link.click();
      onSave?.(dataUrl);
    } finally {
      setIsExporting(false);
    }
  }, [onSave, username]);

  // ── Keyboard shortcuts ─────────────────────────────────────────────
  useAnnotationKeyboard({
    onUndo: handleUndo,
    onRedo: handleRedo,
    onClear: handleClear,
    onEscape: handleEscape,
  });

  // ── Reset zoom ─────────────────────────────────────────────────────
  const handleResetZoom = useCallback(() => {
    setZoom(1);
    setPanOffset({ x: 0, y: 0 });
  }, []);

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

        {/* Zoom indicator */}
        {zoom !== 1 && (
          <button
            onClick={handleResetZoom}
            className="absolute top-2 left-2 z-10 px-2 py-1 text-xs font-medium bg-slate-900/70 text-white rounded-md hover:bg-slate-900/90 transition-colors"
            title="Сбросить zoom"
            aria-label="Сбросить zoom"
          >
            {Math.round(zoom * 100)}%
          </button>
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
            if (isPanning) {
              setIsPanning(false);
              setPanStart(null);
            }
          }}
          onWheel={handleWheel}
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
              left: textInput.position.x * zoom + panOffset.x,
              top: textInput.position.y * zoom + panOffset.y - 28,
              minWidth: 120,
            }}
            aria-label="Ввод текста аннотации"
          />
        )}
      </div>

      {/* Zoom info */}
      {!readOnly && (
        <div className="flex items-center justify-between">
          <span className="text-xs text-slate-400 dark:text-slate-500">
            Ctrl+Scroll — zoom, Ctrl+Z — отменить, Ctrl+Shift+Z — повторить
          </span>
        </div>
      )}

      {/* Toolbar */}
      {!readOnly && (
        <PhotoAnnotationToolbar
          currentTool={currentTool}
          onToolChange={setCurrentTool}
          currentColor={currentColor}
          onColorChange={setCurrentColor}
          strokeWidth={strokeWidth}
          onStrokeWidthChange={setStrokeWidth}
          fontSize={fontSize}
          onFontSizeChange={setFontSize}
          elementCount={elements.length}
          redoCount={redoStack.length}
          onUndo={handleUndo}
          onRedo={handleRedo}
          onClear={handleClear}
          onExport={handleExport}
          isExporting={isExporting}
        />
      )}
    </div>
  );
};

export default PhotoAnnotation;
