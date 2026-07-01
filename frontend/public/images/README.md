# Static Images

Директория для статических изображений, используемых в приложении.

## Формат

Все изображения должны быть в форматах PNG, JPEG или SVG. 
При сборке `vite-imagetools` автоматически конвертирует их в WebP.

## Использование

```tsx
import cameraImage from '@/public/images/camera.png';
// На production: cameraImage → /assets/camera-abc123.webp (через imagetools)

// Для динамических изображений используйте LazyImage:
<LazyImage src="/images/camera.png" alt="Camera" />
```

## Конвертация в WebP

- Установите изображения в PNG/JPEG
- Vite-imagetools + sharp конвертируют их в WebP при сборке
- Для инлайн-импортов используйте суффикс `?webp`:
  ```ts
  import img from './camera.png?webp';
  ```

## P3-LOW-01

- WebP conversion через vite-imagetools + sharp
- Качество: 80%
- Максимальная ширина: 800px
