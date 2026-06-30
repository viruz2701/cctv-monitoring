// ═══════════════════════════════════════════════════════════════════════
// PhotoAnnotationToolbar — Toolbar for PhotoAnnotation component
// P1-PHOTO: Tool selection, colors, stroke width, undo/redo, clear, export
// ═══════════════════════════════════════════════════════════════════════

import React from 'react';
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
import {
  type AnnotationTool,
  COLORS,
  STROKE_WIDTHS,
  TOOLS,
} from './annotationTypes';

// ── Tool icons map ──────────────────────────────────────────────────────

const TOOL_ICONS: Record<AnnotationTool, React.ReactNode> = {
  arrow: <Minus className="w-4 h-4 rotate-45" />,
  freehand: <Pen className="w-4 h-4" />,
  text: <Type className="w-4 h-4" />,
  highlight: <Square className="w-4 h-4" />,
  circle: <Circle className="w-4 h-4" />,
  blur: <EyeOff className="w-4 h-4" />,
  measurement: <Ruler className="w-4 h-4" />,
};

// ── Props ───────────────────────────────────────────────────────────────

interface PhotoAnnotationToolbarProps {
  currentTool: AnnotationTool;
  onToolChange: (tool: AnnotationTool) => void;
  currentColor: string;
  onColorChange: (color: string) => void;
  strokeWidth: number;
  onStrokeWidthChange: (width: number) => void;
  fontSize: number;
  onFontSizeChange: (size: number) => void;
  elementCount: number;
  redoCount: number;
  onUndo: () => void;
  onRedo: () => void;
  onClear: () => void;
  onExport: () => void;
  isExporting?: boolean;
}

// ── Component ───────────────────────────────────────────────────────────

export const PhotoAnnotationToolbar: React.FC<PhotoAnnotationToolbarProps> = ({
  currentTool,
  onToolChange,
  currentColor,
  onColorChange,
  strokeWidth,
  onStrokeWidthChange,
  fontSize,
  onFontSizeChange,
  elementCount,
  redoCount,
  onUndo,
  onRedo,
  onClear,
  onExport,
  isExporting = false,
}) => {
  const hasElements = elementCount > 0;

  return (
    <div
      className="flex flex-wrap items-center gap-2"
      role="toolbar"
      aria-label="Панель инструментов аннотации"
    >
      {/* Tool selection */}
      <div className="flex items-center gap-1 p-1 bg-slate-100 dark:bg-slate-800 rounded-lg">
        {TOOLS.map((tool) => (
          <button
            key={tool.key}
            onClick={() => onToolChange(tool.key)}
            title={tool.label}
            aria-label={tool.label}
            aria-pressed={currentTool === tool.key}
            className={`p-1.5 rounded-md transition-colors ${
              currentTool === tool.key
                ? 'bg-blue-600 text-white shadow-sm'
                : 'text-slate-500 dark:text-slate-400 hover:bg-slate-200 dark:hover:bg-slate-700'
            }`}
          >
            {TOOL_ICONS[tool.key]}
          </button>
        ))}
      </div>

      {/* Separator */}
      <div className="w-px h-6 bg-slate-300 dark:bg-slate-600" />

      {/* Colors */}
      <div className="flex items-center gap-1" role="radiogroup" aria-label="Цвет">
        {COLORS.map((color) => (
          <button
            key={color}
            onClick={() => onColorChange(color)}
            title={color}
            aria-label={`Цвет ${color}`}
            aria-checked={currentColor === color}
            role="radio"
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
      <div className="flex items-center gap-1" role="radiogroup" aria-label="Толщина линии">
        {STROKE_WIDTHS.map((sw) => (
          <button
            key={sw}
            onClick={() => onStrokeWidthChange(sw)}
            title={`${sw}px`}
            aria-label={`Толщина ${sw}px`}
            aria-checked={strokeWidth === sw}
            role="radio"
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

      {/* Font size (only for text tool) */}
      {currentTool === 'text' && (
        <>
          <div className="w-px h-6 bg-slate-300 dark:bg-slate-600" />
          <div className="flex items-center gap-1" role="radiogroup" aria-label="Размер шрифта">
            {[14, 18, 24, 32].map((fs) => (
              <button
                key={fs}
                onClick={() => onFontSizeChange(fs)}
                title={`${fs}px`}
                aria-label={`Шрифт ${fs}px`}
                aria-checked={fontSize === fs}
                role="radio"
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
        </>
      )}

      {/* Separator */}
      <div className="w-px h-6 bg-slate-300 dark:bg-slate-600" />

      {/* Undo / Redo */}
      <button
        onClick={onUndo}
        disabled={!hasElements}
        title="Отменить (Ctrl+Z)"
        aria-label="Отменить"
        className="p-1.5 rounded-md text-slate-500 dark:text-slate-400 hover:bg-slate-100 dark:hover:bg-slate-700 disabled:opacity-30 disabled:cursor-not-allowed"
      >
        <RotateCcw className="w-4 h-4" />
      </button>

      <button
        onClick={onRedo}
        disabled={redoCount === 0}
        title="Повторить (Ctrl+Shift+Z)"
        aria-label="Повторить"
        className="p-1.5 rounded-md text-slate-500 dark:text-slate-400 hover:bg-slate-100 dark:hover:bg-slate-700 disabled:opacity-30 disabled:cursor-not-allowed"
      >
        <RotateCcw className="w-4 h-4 scale-x-[-1]" />
      </button>

      {/* Clear */}
      <button
        onClick={onClear}
        disabled={!hasElements}
        title="Очистить все (Delete)"
        aria-label="Очистить все"
        className="p-1.5 rounded-md text-red-500 hover:bg-red-50 dark:hover:bg-red-900/20 disabled:opacity-30 disabled:cursor-not-allowed"
      >
        <Trash2 className="w-4 h-4" />
      </button>

      {/* Spacer */}
      <div className="flex-1" />

      {/* Layer count */}
      <span className="text-xs text-slate-400 dark:text-slate-500" aria-live="polite">
        {elementCount} слой{elementCount !== 1 ? 'ев' : ''}
        {redoCount > 0 && ` (+${redoCount} отм.)`}
      </span>

      {/* Export */}
      <button
        onClick={onExport}
        disabled={!hasElements || isExporting}
        title="Экспорт PNG"
        aria-label="Экспорт PNG"
        className="px-3 py-1.5 text-xs font-medium rounded-lg bg-emerald-600 text-white hover:bg-emerald-700 disabled:opacity-40 disabled:cursor-not-allowed flex items-center gap-1"
      >
        <Save className="w-3.5 h-3.5" />
        {isExporting ? 'Сохранение...' : 'Экспорт'}
      </button>
    </div>
  );
};

export default PhotoAnnotationToolbar;
