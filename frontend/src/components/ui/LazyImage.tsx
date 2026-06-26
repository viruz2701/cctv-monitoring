// ═══════════════════════════════════════════════════════════════════════
// LazyImage — изображение с IntersectionObserver + loading="lazy"
// P1-2.2: Image Lazy Loading в DataGrid и других компонентах
//
// Особенности:
//   - Intersection Observer для off-screen изображений
//   - Placeholder во время загрузки
//   - aria-busy для accessibility
//   - fallback при ошибке загрузки
// ═══════════════════════════════════════════════════════════════════════

import React, { useState, useRef, useEffect } from 'react';

interface LazyImageProps extends React.ImgHTMLAttributes<HTMLImageElement> {
  /** URL placeholder'а (опционально) */
  placeholder?: string;
  /** Цвет placeholder'а (Tailwind класс) */
  placeholderColor?: string;
  /** Размер placeholder'а */
  placeholderSize?: 'sm' | 'md' | 'lg';
  /** Показать скелетон вместо изображения до загрузки */
  showSkeleton?: boolean;
}

const placeholderSizes = {
  sm: 'w-8 h-8',
  md: 'w-12 h-12',
  lg: 'w-20 h-20',
};

/**
 * LazyImage — компонент для ленивой загрузки изображений.
 *
 * Использование:
 * ```tsx
 * <LazyImage
 *   src="/path/to/image.jpg"
 *   alt="Device thumbnail"
 *   className="w-16 h-16 rounded-lg"
 *   placeholderSize="sm"
 * />
 * ```
 */
export function LazyImage({
  src,
  alt,
  placeholder,
  placeholderColor = 'bg-slate-200 dark:bg-slate-700',
  placeholderSize = 'md',
  showSkeleton = true,
  className = '',
  style,
  ...imgProps
}: LazyImageProps) {
  const imgRef = useRef<HTMLImageElement>(null);
  const [isInView, setIsInView] = useState(false);
  const [isLoaded, setIsLoaded] = useState(false);
  const [hasError, setHasError] = useState(false);
  const observerRef = useRef<IntersectionObserver | null>(null);

  useEffect(() => {
    const imgEl = imgRef.current;
    if (!imgEl) return;

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

    observerRef.current.observe(imgEl);

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

  const sizeClass = placeholderSizes[placeholderSize];

  return (
    <div
      ref={imgRef}
      className={`relative overflow-hidden ${className}`}
      role="img"
      aria-label={alt || 'Lazy loaded image'}
      aria-busy={!isLoaded}
    >
      {/* Placeholder / Skeleton */}
      {(!isLoaded || !isInView) && (
        <div
          className={`absolute inset-0 ${placeholderColor} ${
            showSkeleton ? 'animate-pulse' : ''
          } flex items-center justify-center`}
        >
          {placeholder && !hasError && (
            <img
              src={placeholder}
              alt=""
              className="opacity-50 w-8 h-8"
              aria-hidden="true"
            />
          )}
          {hasError && (
            <svg
              className="w-6 h-6 text-slate-400"
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
        </div>
      )}

      {/* Actual image — показываем только когда в viewport */}
      {isInView && (
        <img
          src={src}
          alt={alt || ''}
          loading="lazy"
          onLoad={handleLoad}
          onError={handleError}
          className={`w-full h-full object-cover transition-opacity duration-300 ${
            isLoaded ? 'opacity-100' : 'opacity-0 absolute'
          }`}
          style={style}
          {...imgProps}
        />
      )}
    </div>
  );
}
