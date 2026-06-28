// ═══════════════════════════════════════════════════════════════════════
// LazyImage — Unit Tests
// P1-PERF.2: Image Lazy Loading
// ═══════════════════════════════════════════════════════════════════════

import React from 'react';
import { render, screen } from '@testing-library/react';
import { describe, it, expect } from 'vitest';
import { LazyImage } from '../LazyImage';

describe('LazyImage', () => {
  it('renders placeholder when image is not yet loaded', () => {
    const { container } = render(
      <LazyImage src="/test.jpg" alt="Test image" />
    );

    // Плейсхолдер с анимацией должен присутствовать
    const placeholder = container.querySelector('.animate-pulse');
    expect(placeholder).toBeTruthy();

    // Атрибут aria-busy должен быть true
    const wrapper = container.querySelector('[role="img"]');
    expect(wrapper?.getAttribute('aria-busy')).toBe('true');
  });

  it('renders with custom alt text', () => {
    render(<LazyImage src="/test.jpg" alt="Camera thumbnail" />);
    expect(screen.getByLabelText('Camera thumbnail')).toBeTruthy();
  });

  it('renders with aspect ratio', () => {
    const { container } = render(
      <LazyImage src="/test.jpg" alt="Test" aspectRatio="16/9" />
    );

    // Проверяем, что контейнер для aspect ratio создан
    const aspectContainer = container.querySelector('[aria-hidden="true"]');
    expect(aspectContainer).toBeTruthy();
  });

  it('renders without skeleton when showSkeleton=false', () => {
    const { container } = render(
      <LazyImage src="/test.jpg" alt="Test" showSkeleton={false} />
    );

    const skeleton = container.querySelector('.animate-pulse');
    expect(skeleton).toBeFalsy();
  });
});
