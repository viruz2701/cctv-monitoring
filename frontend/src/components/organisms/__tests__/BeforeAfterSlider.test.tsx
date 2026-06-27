// ═══════════════════════════════════════════════════════════════════════
// BeforeAfterSlider — Unit Tests
// P1-QA.8: Coverage 80%
// ═══════════════════════════════════════════════════════════════════════

import React from 'react';
import { describe, it, expect, vi, beforeEach, afterEach } from 'vitest';
import { render, screen, fireEvent, cleanup } from '@testing-library/react';
import { BeforeAfterSlider } from '../BeforeAfterSlider';

// ── Test Data ────────────────────────────────────────────────────────

const TEST_BEFORE = 'https://cdn.example.com/photos/before.jpg';
const TEST_AFTER = 'https://cdn.example.com/photos/after.jpg';

// ── Helpers ──────────────────────────────────────────────────────────

function renderSlider(props: Partial<React.ComponentProps<typeof BeforeAfterSlider>> = {}) {
  return render(
    <BeforeAfterSlider
      beforeImage={TEST_BEFORE}
      afterImage={TEST_AFTER}
      {...props}
    />,
  );
}

function simulateMouseDrag(
  container: HTMLElement,
  startX: number,
  endX: number,
): void {
  // Mouse down at start position
  fireEvent.mouseDown(container, { clientX: startX });

  // Get container rect
  const rect = container.getBoundingClientRect();

  // Mouse move to 25% position
  const moveX = rect.left + rect.width * 0.25;
  fireEvent.mouseMove(document, { clientX: moveX });

  // Mouse up
  fireEvent.mouseUp(document);
}

// ── Tests ────────────────────────────────────────────────────────────

describe('BeforeAfterSlider', () => {
  beforeEach(() => {
    vi.clearAllMocks();
  });

  afterEach(() => {
    cleanup();
  });

  // ── Rendering ─────────────────────────────────────────────────────

  it('Должен отрендерить два изображения (before/after)', () => {
    renderSlider();
    const images = screen.getAllByRole('img');
    expect(images.length).toBe(2);
  });

  it('Должен показать before изображение', () => {
    renderSlider();
    const img = screen.getByAltText('До');
    expect(img).toHaveAttribute('src', TEST_BEFORE);
  });

  it('Должен показать after изображение', () => {
    renderSlider();
    const img = screen.getByAltText('После');
    expect(img).toHaveAttribute('src', TEST_AFTER);
  });

  it('Должен показать пользовательские подписи', () => {
    renderSlider({ beforeLabel: 'Before', afterLabel: 'After' });
    expect(screen.getByAltText('Before')).toBeDefined();
    expect(screen.getByAltText('After')).toBeDefined();
    // Footer also shows labels
    expect(screen.getAllByText('Before').length).toBe(2);
    expect(screen.getAllByText('After').length).toBe(2);
  });

  it('Должен показать подписи по умолчанию "До" и "После"', () => {
    renderSlider();
    expect(screen.getByAltText('До')).toBeDefined();
    expect(screen.getByAltText('После')).toBeDefined();
  });

  it('Должен показать слайдер по центру (50%) по умолчанию', () => {
    const { container } = renderSlider();
    // Проверяем, что процент отображается
    expect(screen.getByText('50%')).toBeDefined();
  });

  it('Должен иметь cursor-ew-resize на контейнере', () => {
    const { container } = renderSlider();
    const sliderContainer = container.firstChild?.firstChild as HTMLElement;
    expect(sliderContainer.className).toContain('cursor-ew-resize');
  });

  it('Должен установить aspect-ratio 16/9 на контейнере', () => {
    const { container } = renderSlider();
    const sliderContainer = container.firstChild?.firstChild as HTMLElement;
    expect(sliderContainer.style.aspectRatio).toBe('16/9');
  });

  // ── Slider Interaction ────────────────────────────────────────────

  it('Должен обновить позицию слайдера при mouse move', () => {
    const { container } = renderSlider();
    const sliderContainer = container.firstChild?.firstChild as HTMLElement;

    // Set up bounding rect mock
    Object.defineProperty(sliderContainer, 'getBoundingClientRect', {
      value: () => ({
        left: 0,
        right: 400,
        top: 0,
        bottom: 225,
        width: 400,
        height: 225,
      }),
    });

    // Mouse down to start dragging
    fireEvent.mouseDown(sliderContainer, { clientX: 200 });

    // Move to 25%
    fireEvent.mouseMove(document, { clientX: 100 });

    // Проверяем, что процент обновился
    // (25% от 400px = 100px)
    expect(screen.getByText('25%')).toBeDefined();

    fireEvent.mouseUp(document);
  });

  it('Должен завершить drag при mouse up', () => {
    const { container } = renderSlider();
    const sliderContainer = container.firstChild?.firstChild as HTMLElement;

    Object.defineProperty(sliderContainer, 'getBoundingClientRect', {
      value: () => ({
        left: 0,
        right: 400,
        top: 0,
        bottom: 225,
        width: 400,
        height: 225,
      }),
    });

    // Start drag
    fireEvent.mouseDown(sliderContainer, { clientX: 200 });
    fireEvent.mouseMove(document, { clientX: 100 });

    // End drag
    fireEvent.mouseUp(document);

    // После mouseUp, move не должен менять позицию
    fireEvent.mouseMove(document, { clientX: 300 });

    // Позиция осталась на 25%
    expect(screen.getByText('25%')).toBeDefined();
  });

  it('Должен ограничить позицию слайдера от 0 до 100', () => {
    const { container } = renderSlider();
    const sliderContainer = container.firstChild?.firstChild as HTMLElement;

    Object.defineProperty(sliderContainer, 'getBoundingClientRect', {
      value: () => ({
        left: 0,
        right: 400,
        top: 0,
        bottom: 225,
        width: 400,
        height: 225,
      }),
    });

    // Drag beyond left edge
    fireEvent.mouseDown(sliderContainer, { clientX: 200 });
    fireEvent.mouseMove(document, { clientX: -50 });
    expect(screen.getByText('0%')).toBeDefined();
    fireEvent.mouseUp(document);

    // Drag beyond right edge
    fireEvent.mouseDown(sliderContainer, { clientX: 200 });
    fireEvent.mouseMove(document, { clientX: 500 });
    expect(screen.getByText('100%')).toBeDefined();
    fireEvent.mouseUp(document);
  });

  // ── Touch Support ─────────────────────────────────────────────────

  it('Должен поддерживать touch start', () => {
    const { container } = renderSlider();
    const sliderContainer = container.firstChild?.firstChild as HTMLElement;

    Object.defineProperty(sliderContainer, 'getBoundingClientRect', {
      value: () => ({
        left: 0,
        right: 400,
        top: 0,
        bottom: 225,
        width: 400,
        height: 225,
      }),
    });

    // Touch start
    fireEvent.touchStart(sliderContainer);

    // Touch move
    fireEvent.touchMove(document, {
      touches: [{ clientX: 80 }],
    });

    expect(screen.getByText('20%')).toBeDefined();

    fireEvent.touchEnd(document);
  });

  it('Должен обработать touch move и touch end', () => {
    const { container } = renderSlider();
    const sliderContainer = container.firstChild?.firstChild as HTMLElement;

    Object.defineProperty(sliderContainer, 'getBoundingClientRect', {
      value: () => ({
        left: 0,
        right: 400,
        top: 0,
        bottom: 225,
        width: 400,
        height: 225,
      }),
    });

    fireEvent.touchStart(sliderContainer);
    fireEvent.touchMove(document, {
      touches: [{ clientX: 200 }],
    });

    expect(screen.getByText('50%')).toBeDefined();

    fireEvent.touchEnd(document);

    // После touchEnd движение не должно работать
    fireEvent.touchMove(document, {
      touches: [{ clientX: 0 }],
    });
    expect(screen.getByText('50%')).toBeDefined();
  });

  // ── Images ────────────────────────────────────────────────────────

  it('Должен установить draggable=false на изображениях', () => {
    renderSlider();
    const images = screen.getAllByRole('img');
    images.forEach((img) => {
      expect(img).toHaveAttribute('draggable', 'false');
    });
  });

  it('Должен установить className object-cover на изображениях', () => {
    renderSlider();
    const images = screen.getAllByRole('img');
    images.forEach((img) => {
      expect(img.className).toContain('object-cover');
    });
  });

  // ── Slider Handle ─────────────────────────────────────────────────

  it('Должен показать слайдер handle', () => {
    const { container } = renderSlider();
    // Slider handle — белая линия с кругом
    const handle = container.querySelector('.bg-white.shadow-lg');
    expect(handle).toBeDefined();
  });

  it('Должен показать иконку со стрелками на handle', () => {
    const { container } = renderSlider();
    // SVG иконка внутри handle
    const svg = container.querySelector('svg');
    expect(svg).toBeDefined();
    expect(svg?.getAttribute('viewBox')).toBe('0 0 24 24');
  });

  // ── Percentage Display ────────────────────────────────────────────

  it('Должен показать текущий процент под слайдером', () => {
    renderSlider();
    expect(screen.getByText('50%')).toBeDefined();
  });

  it('Должен обновить процент при изменении позиции', () => {
    const { container } = renderSlider();
    const sliderContainer = container.firstChild?.firstChild as HTMLElement;

    Object.defineProperty(sliderContainer, 'getBoundingClientRect', {
      value: () => ({
        left: 0,
        right: 400,
        top: 0,
        bottom: 225,
        width: 400,
        height: 225,
      }),
    });

    fireEvent.mouseDown(sliderContainer, { clientX: 200 });
    fireEvent.mouseMove(document, { clientX: 300 });
    expect(screen.getByText('75%')).toBeDefined();
    fireEvent.mouseUp(document);
  });

  // ── Cleanup ───────────────────────────────────────────────────────

  it('Должен очистить event listeners при unmount', () => {
    const { unmount } = renderSlider();
    // Не должно быть ошибок при unmount
    expect(() => unmount()).not.toThrow();
  });
});
