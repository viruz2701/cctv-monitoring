import React, { useRef, useState, useEffect } from 'react';

interface Annotation {
  x: number;
  y: number;
  text: string;
  color: string;
}

interface PhotoAnnotationProps {
  imageUrl: string;
  onSave?: (annotations: Annotation[], dataUrl: string) => void;
  readOnly?: boolean;
}

const COLORS = ['#ef4444', '#f97316', '#3b82f6', '#22c55e', '#a855f7', '#ec4899'];

export const PhotoAnnotation: React.FC<PhotoAnnotationProps> = ({
  imageUrl,
  onSave,
  readOnly = false,
}) => {
  const canvasRef = useRef<HTMLCanvasElement>(null);
  const imgRef = useRef<HTMLImageElement>(null);
  const [annotations, setAnnotations] = useState<Annotation[]>([]);
  const [currentColor, setCurrentColor] = useState(COLORS[0]);
  const [isPlacing, setIsPlacing] = useState(false);
  const [imageLoaded, setImageLoaded] = useState(false);

  useEffect(() => {
    const img = new Image();
    img.crossOrigin = 'anonymous';
    img.onload = () => {
      setImageLoaded(true);
      drawCanvas();
    };
    img.src = imageUrl;
    imgRef.current = img;
  }, [imageUrl]);

  const drawCanvas = () => {
    const canvas = canvasRef.current;
    const img = imgRef.current;
    if (!canvas || !img) return;

    const ctx = canvas.getContext('2d');
    if (!ctx) return;

    canvas.width = img.naturalWidth || 800;
    canvas.height = img.naturalHeight || 600;

    ctx.drawImage(img, 0, 0, canvas.width, canvas.height);

    // Draw annotations
    annotations.forEach((ann) => {
      ctx.beginPath();
      ctx.arc(ann.x, ann.y, 6, 0, Math.PI * 2);
      ctx.fillStyle = ann.color;
      ctx.fill();
      ctx.strokeStyle = '#ffffff';
      ctx.lineWidth = 2;
      ctx.stroke();

      ctx.font = '14px Inter, system-ui, sans-serif';
      ctx.fillStyle = ann.color;
      const metrics = ctx.measureText(ann.text);
      const padding = 4;
      ctx.fillStyle = 'rgba(0,0,0,0.7)';
      ctx.fillRect(
        ann.x + 10,
        ann.y - 10,
        metrics.width + padding * 2,
        24
      );
      ctx.fillStyle = '#ffffff';
      ctx.fillText(ann.text, ann.x + 10 + padding, ann.y + 6);
    });
  };

  useEffect(() => {
    if (imageLoaded) drawCanvas();
  }, [annotations, imageLoaded]);

  const handleCanvasClick = (e: React.MouseEvent<HTMLCanvasElement>) => {
    if (readOnly || !isPlacing) return;
    const canvas = canvasRef.current;
    if (!canvas) return;

    const rect = canvas.getBoundingClientRect();
    const scaleX = canvas.width / rect.width;
    const scaleY = canvas.height / rect.height;
    const x = (e.clientX - rect.left) * scaleX;
    const y = (e.clientY - rect.top) * scaleY;

    const text = prompt('Введите текст аннотации:');
    if (!text) return;

    const newAnnotations = [...annotations, { x, y, text, color: currentColor }];
    setAnnotations(newAnnotations);
    setIsPlacing(false);
  };

  const handleSave = () => {
    const canvas = canvasRef.current;
    if (!canvas) return;
    const dataUrl = canvas.toDataURL('image/png');
    onSave?.(annotations, dataUrl);
  };

  const clearAll = () => {
    setAnnotations([]);
  };

  return (
    <div className="space-y-3">
      <div className="relative rounded-lg overflow-hidden border border-slate-200 dark:border-slate-700 bg-slate-50 dark:bg-slate-900">
        <canvas
          ref={canvasRef}
          onClick={handleCanvasClick}
          className="w-full h-auto cursor-crosshair max-h-96 object-contain"
          style={{ aspectRatio: '16/9' }}
        />
      </div>

      {!readOnly && (
        <div className="flex flex-wrap items-center gap-2">
          <button
            onClick={() => { setIsPlacing(!isPlacing); }}
            className={`px-3 py-1.5 text-xs font-medium rounded-lg border transition-colors ${
              isPlacing
                ? 'bg-blue-600 text-white border-blue-600'
                : 'bg-white dark:bg-slate-800 text-slate-700 dark:text-slate-300 border-slate-300 dark:border-slate-600 hover:bg-slate-50 dark:hover:bg-slate-700'
            }`}
          >
            {isPlacing ? '✏️ Разместить...' : 'Добавить аннотацию'}
          </button>

          <div className="flex gap-1">
            {COLORS.map((color) => (
              <button
                key={color}
                onClick={() => setCurrentColor(color)}
                className={`w-6 h-6 rounded-full border-2 transition-all ${
                  currentColor === color
                    ? 'border-slate-900 dark:border-white scale-110'
                    : 'border-transparent'
                }`}
                style={{ backgroundColor: color }}
              />
            ))}
          </div>

          <button
            onClick={clearAll}
            className="px-3 py-1.5 text-xs font-medium rounded-lg border border-red-300 dark:border-red-700 text-red-600 dark:text-red-400 hover:bg-red-50 dark:hover:bg-red-900/20"
          >
            Очистить
          </button>

          <button
            onClick={handleSave}
            className="px-3 py-1.5 text-xs font-medium rounded-lg bg-emerald-600 text-white hover:bg-emerald-700"
          >
            Сохранить
          </button>

          <span className="text-xs text-slate-400 dark:text-slate-500 ml-auto">
            {annotations.length} аннотаций
          </span>
        </div>
      )}
    </div>
  );
};
