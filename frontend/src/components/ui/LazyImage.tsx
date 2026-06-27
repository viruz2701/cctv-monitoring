// ═══════════════════════════════════════════════════════════════════════
// LazyImage — изображение с IntersectionObserver + loading="lazy" + WebP
// P1-PERF.2: Image Lazy Loading в DataGrid и других компонентах
//
// Особенности:
//   - WebP через <picture> с автоматическим выбором формата
//   - Intersection Observer для off-screen изображений
//   - Placeholder во время загрузки (skeleton / blur-hash)
//   - aspectRatio для предотвращения layout shift (CLS)
//   - aria-busy для accessibility
//   - fallback при ошибке загрузки
// ═══════════════════════════════════════════════════════════════════════

import React, { useState, useRef, useEffect } from 'react';

interface LazyImageProps extends Omit<React.ImgHTMLAttributes<HTMLImageElement>, 'loading'> {
  /** URL изображения (основной) */
  src: string;
  /** URL WebP версии (если отличается от src) */
  srcWebp?: string;
  /** Alt текст */
  alt: string;
  /** URL placeholder'а (опционально) */
  placeholder?: string;
  /** Цвет placeholder'а (Tailwind класс) */
  placeholderColor?: string;
  /** Размер placeholder'а */
  placeholderSize?: 'sm' | 'md' | 'lg';
  /** Показать скелетон вместо изображения до загрузки */
  showSkeleton?: boolean;
  /** Соотношение сторон для контейнера (напр. "16/9", "4/3", "1/1") */
  aspectRatio?: string;
  /** Принудительно отключить WebP (для тестов) */
  _disableWebp?: boolean;
}

const placeholderSizes = {
  sm: 'w-8 h-8',
  md: 'w-12 h-12',
  lg: 'w-20 h-20',
};

/**
 * LazyImage — компонент для ленивой загрузки изображений с WebP поддержкой.
 *
 * Использование:
 * ```tsx
 * <LazyImage
 *   src="/path/to/image.jpg"
 *   srcWebp="/path/to/image.webp"
 *   alt="Device thumbnail"
 *   className="w-16 h-16 rounded-lg"
 *   placeholderSize="sm"
 *   aspectRatio="1/1"
 * />
 * ```
 */
export function LazyImage({
  src,
  srcWebp,
  alt,
  placeholder,
  placeholderColor = 'bg-slate-200 dark:bg-slate-700',
  placeholderSize = 'md',
  showSkeleton = true,
  aspectRatio,
  className = '',
  style,
  _disableWebp = false,
  ...imgProps
}: LazyImageProps) {
  const imgRef = useRef<HTMLDivElement>(null);
  const [isInView, setIsInView] = useState(false);
  const [isLoaded, setIsLoaded] = useState(false);
  const [hasError, setHasError] = useState(false);
  const observerRef = useRef<IntersectionObserver | null>(null);

  useEffect(() => {
    const el = imgRef.current;
    if (!el) return;

    observerRef.current = new IntersectionObserver(
      ([entry]) => {
        if (entry.isIntersecting) {
          setIsInView(true);
          observerRef.current?.disconnect();
        }
      },
      {
        rootMargin: '200px', // начинаем загрузку за 200px до появления
        threshold: 0.01,
      },
    );

    observerRef.current.observe(el);

    return () => {
      observerRef.current?.disconnect();
    };
  }, []);

  const handleLoad = () => {
    setIsLoaded(true);
  };

  const handleError = () => {
    setHasError(true);
    setIsLoaded(true);
  };

  const supportsWebp = !_disableWebp && typeof window !== 'undefined' && !src.endsWith('.gif') && !src.endsWith('.svg');
  const useWebp = srcWebp || (supportsWebp ? src.replace(/\.(jpg|jpeg|png)$/i, '.webp') : undefined);
  const hasWebp = useWebp && useWebp !== src;

  const sizeClass = placeholderSizes[placeholderSize];
  const aspectPadding = aspectRatio ? aspectRatio.split('/').map(Number) : null;
  const aspectStyle = aspectPadding && aspectPadding[1] > 0
    ? { paddingBottom: `${(aspectPadding[1] / aspectPadding[0]) * 100}%` }
    : undefined;

  return (
    <div
      ref={imgRef}
      className={`relative overflow-hidden ${className}`}
      style={{
        ...style,
        ...(aspectStyle ? {} : {}),
      }}
      role="img"
      aria-label={alt || 'Lazy loaded image'}
      aria-busy={!isLoaded}
    >
      {/* Aspect ratio container */}
      {aspectStyle && (
        <div style={aspectStyle} aria-hidden="true" />
      )}

      {/* Placeholder / Skeleton */}
      {(!isLoaded || !isInView) && (
        <div
          className={`absolute inset-0 ${placeholderColor} ${
            showSkeleton ? 'animate-pulse' : ''
          } flex items-center justify-center`}
          style={aspectStyle ? { position: 'absolute', inset: 0 } : undefined}
        >
          {placeholder && !hasError && (
            <img
              src={placeholder}
              alt=""
              className={`opacity-50 ${sizeClass}`}
              aria-hidden="true"
            />
          )}
          {!placeholder && !hasError && (
            <svg
              className={`${sizeClass} text-slate-400 dark:text-slate-500`}
              fill="none"
              stroke="currentColor"
              viewBox="0 0 24 24"
              aria-hidden="true"
            >
              <path
                strokeLinecap="round"
                strokeLinejoin="round"
                strokeWidth={1.5}
                d="M4 16l4.586-4.586a2 2 0 012.828 0L16 16m-2-2l1.586-1.586a2 2 0 012.828 0L20 14m-6-6h.01M6 20h12a2 2 0 002-2V6a2 2 0 00-2-2H6a2 2 0 00-2 2v12a2 2 0 002 2z"
              />
            </svg>
          )}
          {hasError && (
            <svg
              className={`${sizeClass} text-red-400`}
              fill="none"
              stroke="currentColor"
              viewBox="0 0 24 24"
              aria-hidden="true"
            >
              <path
                strokeLinecap="round"
                strokeLinejoin="round"
                strokeWidth={1.5}
                d="M12 9v2m0 4h.01m-6.938 4h13.856c1.54 0 2.502-1.667 1.732-2.5L13.732 4.5c-.77-.833-2.694-.833-3.464 0L3.34 16.5c-.77.833.192 2.5 1.732 2.5z"
              />
            </svg>
          )}
        </div>
      )}

      {/* Actual image — показываем только когда в viewport */}
      {isInView && (
        <>
          {hasWebp ? (
            <picture>
              <source srcSet={useWebp!} type="image/webp" />
              <img
                src={src}
                alt={alt || ''}
                loading="lazy"
                onLoad={handleLoad}
                onError={handleError}
                className={`w-full h-full object-cover transition-opacity duration-300 ${
                  isLoaded ? 'opacity-100' : 'opacity-0 absolute inset-0'
                }`}
                style={aspectStyle ? { position: 'absolute', inset: 0, width: '100%', height: '100%' } : undefined}
                {...imgProps}
              />
            </picture>
          ) : (
            <img
              src={src}
              alt={alt || ''}
              loading="lazy"
              onLoad={handleLoad}
              onError={handleError}
              className={`w-full h-full object-cover transition-opacity duration-300 ${
                isLoaded ? 'opacity-100' : 'opacity-0 absolute inset-0'
              }`}
              style={aspectStyle ? { position: 'absolute', inset: 0, width: '100%', height: '100%' } : undefined}
              {...imgProps}
            />
          )}
        </>
      )}
    </div>
  );
}
