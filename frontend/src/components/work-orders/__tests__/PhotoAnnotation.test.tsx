// ═══════════════════════════════════════════════════════════════════════
// PhotoAnnotation.test.tsx — Unit tests for PhotoAnnotation component
// P1-PHOTO: Freehand drawing, text labels, blur/redact, measurement,
//           layer management (undo/redo), and export as PNG.
// ═══════════════════════════════════════════════════════════════════════

import React from 'react';
import { render, screen } from '@testing-library/react';
import userEvent from '@testing-library/user-event';
import { describe, it, expect, vi, beforeEach } from 'vitest';
import { PhotoAnnotation } from '../PhotoAnnotation';

// ── Mocks ──────────────────────────────────────────────────────────────

/**
 * Mock для window.Image — класс, а не стрелочная функция,
 * чтобы работал вызов `new Image()`.
 * onload вызывается при установке src (как в реальном браузере).
 */
class MockImage {
  onload: (() => void) | null = null;
  naturalWidth = 800;
  naturalHeight = 600;
  crossOrigin = '';
  private _src = '';

  set src(url: string) {
    this._src = url;
    // Вызываем onload после установки src (имитация загрузки)
    setTimeout(() => {
      this.onload?.();
    }, 0);
  }

  get src(): string {
    return this._src;
  }
}

vi.stubGlobal('Image', MockImage);

// Мокаем canvas getContext с минимальным набором методов
const mockCanvasContext = {
  save: vi.fn(),
  restore: vi.fn(),
  beginPath: vi.fn(),
  moveTo: vi.fn(),
  lineTo: vi.fn(),
  stroke: vi.fn(),
  fill: vi.fn(),
  arc: vi.fn(),
  fillRect: vi.fn(),
  strokeRect: vi.fn(),
  scale: vi.fn(),
  drawImage: vi.fn(),
  measureText: vi.fn(() => ({ width: 50 })),
  fillText: vi.fn(),
  closePath: vi.fn(),
  setLineDash: vi.fn(),
  getLineDash: vi.fn(() => []),
  clearRect: vi.fn(),
  createLinearGradient: vi.fn(() => ({
    addColorStop: vi.fn(),
  })),
};

beforeEach(() => {
  vi.clearAllMocks();
  HTMLCanvasElement.prototype.getContext = vi.fn(
    () => mockCanvasContext as unknown as CanvasRenderingContext2D,
  ) as any;
  HTMLCanvasElement.prototype.toDataURL = vi.fn(
    () => 'data:image/png;base64,mock',
  ) as any;
  // jsdom не поддерживает setPointerCapture — добавляем заглушку
  HTMLCanvasElement.prototype.setPointerCapture = vi.fn() as any;
});

// ── Shared helpers ────────────────────────────────────────────────────

const defaultProps = {
  imageUrl: 'https://example.com/photo.jpg',
  onSave: vi.fn(),
};

// ── Tests ──────────────────────────────────────────────────────────────

describe('PhotoAnnotation', () => {
  // ── 1. Renders canvas and toolbar ───────────────────────────────────
  it('renders canvas and toolbar with image loaded', async () => {
    const { container } = render(<PhotoAnnotation {...defaultProps} />);

    // Ждём загрузки изображения — после неё тулбар становится доступен
    // toolbar отображается всегда при readOnly=false
    await screen.findByTitle('Стрелка');

    // Canvas должен быть в DOM
    const canvas = container.querySelector('canvas');
    expect(canvas).toBeInTheDocument();

    // Undo кнопка — часть toolbar
    const undoButton = screen.getByTitle('Отменить (Ctrl+Z)');
    expect(undoButton).toBeInTheDocument();
  });

  // ── 2. Toolbar shows all drawing tools ──────────────────────────────
  it('shows all 7 drawing tools in toolbar', () => {
    render(<PhotoAnnotation {...defaultProps} />);

    // Все инструменты должны быть видны (toolbar не зависит от загрузки изображения)
    expect(screen.getByTitle('Стрелка')).toBeInTheDocument();
    expect(screen.getByTitle('Рисование')).toBeInTheDocument();
    expect(screen.getByTitle('Текст')).toBeInTheDocument();
    expect(screen.getByTitle('Выделение')).toBeInTheDocument();
    expect(screen.getByTitle('Круг')).toBeInTheDocument();
    expect(screen.getByTitle('Размытие')).toBeInTheDocument();
    expect(screen.getByTitle('Линейка')).toBeInTheDocument();
  });

  // ── 3. Color picker changes color ───────────────────────────────────
  it('changes current color when color button is clicked', async () => {
    render(<PhotoAnnotation {...defaultProps} />);

    // Находим кнопку зелёного цвета (#22c55e)
    const colorButton = screen.getByTitle('#22c55e');
    expect(colorButton).toBeInTheDocument();

    // Кликаем по цвету
    const user = userEvent.setup();
    await user.click(colorButton);

    // Проверяем что цвет выбран (должен иметь класс scale-125)
    expect(colorButton.className).toContain('scale-125');
  });

  // ── 4. Undo button disabled when no elements ────────────────────────
  it('disables undo button when no elements exist', () => {
    render(<PhotoAnnotation {...defaultProps} />);

    // Undo кнопка — должна быть disabled при пустом elements
    const undoButton = screen.getByTitle('Отменить (Ctrl+Z)');
    expect(undoButton).toBeDisabled();

    // Redo кнопка
    const redoButton = screen.getByTitle('Повторить (Ctrl+Shift+Z)');
    expect(redoButton).toBeDisabled();

    // Clear кнопка
    const clearButton = screen.getByTitle('Очистить все');
    expect(clearButton).toBeDisabled();

    // Export кнопка
    const exportButton = screen.getByTitle('Экспорт PNG');
    expect(exportButton).toBeDisabled();
  });

  // ── 5. Undo enables after drawing ───────────────────────────────────
  it('enables undo after drawing an element via pointer events', async () => {
    const { container } = render(<PhotoAnnotation {...defaultProps} />);

    // Ждём загрузки изображения
    await screen.findByTitle('Стрелка');

    const canvas = container.querySelector('canvas')!;

    // Симулируем рисование стрелки:
    // [MouseLeft>] — нажать левую кнопку мыши (pointerdown)
    // [/MouseLeft] — отпустить левую кнопку мыши (pointerup)
    const user = userEvent.setup();
    await user.pointer([
      { keys: '[MouseLeft>]', target: canvas, coords: { x: 100, y: 100 } },
      { keys: '[/MouseLeft]', target: canvas, coords: { x: 200, y: 200 } },
    ]);

    // После рисования элемента, undo должен стать enabled
    await vi.waitFor(
      () => {
        const undoButton = screen.getByTitle('Отменить (Ctrl+Z)');
        expect(undoButton).not.toBeDisabled();
      },
      { timeout: 3000 },
    );
  });

  // ── 6. Export button triggers save ──────────────────────────────────
  it('calls onSave callback when export button is clicked after drawing', async () => {
    const onSaveMock = vi.fn();
    const { container } = render(<PhotoAnnotation {...defaultProps} onSave={onSaveMock} />);

    // Ждём загрузки изображения
    await screen.findByTitle('Стрелка');

    const canvas = container.querySelector('canvas')!;
    const user = userEvent.setup();

    // Рисуем элемент
    await user.pointer([
      { keys: '[MouseLeft>]', target: canvas, coords: { x: 100, y: 100 } },
      { keys: '[/MouseLeft]', target: canvas, coords: { x: 200, y: 200 } },
    ]);

    // Ждём пока undo станет активным (элемент добавлен)
    await vi.waitFor(
      () => {
        expect(screen.getByTitle('Отменить (Ctrl+Z)')).not.toBeDisabled();
      },
      { timeout: 3000 },
    );

    // Кликаем Export
    const exportButton = screen.getByTitle('Экспорт PNG');
    await user.click(exportButton);

    // Проверяем что onSave вызван с data URL
    expect(onSaveMock).toHaveBeenCalledWith('data:image/png;base64,mock');
  });
});
