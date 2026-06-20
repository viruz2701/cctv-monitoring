import React, { useEffect, useRef } from 'react';

interface QRCodeProps {
  value: string;
  size?: number;
  bgColor?: string;
  fgColor?: string;
  className?: string;
  label?: string;
}

export function QRCode({
  value,
  size = 200,
  bgColor = '#ffffff',
  fgColor = '#000000',
  className = '',
  label,
}: QRCodeProps) {
  const canvasRef = useRef<HTMLCanvasElement>(null);

  useEffect(() => {
    const canvas = canvasRef.current;
    if (!canvas || !value) return;

    const ctx = canvas.getContext('2d');
    if (!ctx) return;

    canvas.width = size;
    canvas.height = size;

    // Simple QR-like pattern (in production, use qrcode library)
    drawQRPattern(ctx, value, size, bgColor, fgColor);
  }, [value, size, bgColor, fgColor]);

  return (
    <div className={`flex flex-col items-center ${className}`}>
      <canvas
        ref={canvasRef}
        className="rounded-lg border border-slate-200 dark:border-slate-700"
        style={{ width: size, height: size }}
      />
      {label && (
        <p className="mt-2 text-sm text-slate-500 dark:text-slate-400 text-center">{label}</p>
      )}
    </div>
  );
}

function drawQRPattern(
  ctx: CanvasRenderingContext2D,
  value: string,
  size: number,
  bgColor: string,
  fgColor: string,
) {
  // Background
  ctx.fillStyle = bgColor;
  ctx.fillRect(0, 0, size, size);

  // Generate deterministic pattern from value
  const hash = djb2Hash(value);
  const moduleCount = 21;
  const moduleSize = size / moduleCount;
  const modules: boolean[][] = Array.from({ length: moduleCount }, () =>
    Array(moduleCount).fill(false),
  );

  // Finder patterns (three corners)
  drawFinderPattern(modules, 0, 0);
  drawFinderPattern(modules, moduleCount - 7, 0);
  drawFinderPattern(modules, 0, moduleCount - 7);

  // Timing patterns
  for (let i = 8; i < moduleCount - 8; i++) {
    modules[i][6] = i % 2 === 0;
    modules[6][i] = i % 2 === 0;
  }

  // Data modules from hash
  let bitIndex = 0;
  for (let col = moduleCount - 1; col >= 0; col -= 2) {
    if (col === 6) col = 5;
    for (let row = 0; row < moduleCount; row++) {
      for (let c = col; c >= col - 1 && c >= 0; c--) {
        if (modules[row][c] === false && row !== 6 && c !== 6) {
          const bit = (hash >> bitIndex) & 1;
          if (!isFinderArea(row, c, moduleCount)) {
            modules[row][c] = bit === 1;
          }
          bitIndex++;
          if (bitIndex >= 32) bitIndex = 0;
        }
      }
    }
  }

  // Render
  ctx.fillStyle = fgColor;
  for (let row = 0; row < moduleCount; row++) {
    for (let col = 0; col < moduleCount; col++) {
      if (modules[row][col]) {
        ctx.fillRect(
          Math.round(col * moduleSize),
          Math.round(row * moduleSize),
          Math.ceil(moduleSize),
          Math.ceil(moduleSize),
        );
      }
    }
  }
}

function drawFinderPattern(modules: boolean[][], startRow: number, startCol: number) {
  for (let row = 0; row < 7; row++) {
    for (let col = 0; col < 7; col++) {
      const isBorder = row === 0 || row === 6 || col === 0 || col === 6;
      const isInner = row >= 2 && row <= 4 && col >= 2 && col <= 4;
      modules[startRow + row][startCol + col] = isBorder || isInner;
    }
  }
}

function isFinderArea(row: number, col: number, moduleCount: number): boolean {
  return (
    (row < 8 && col < 8) ||
    (row < 8 && col >= moduleCount - 8) ||
    (row >= moduleCount - 8 && col < 8)
  );
}

function djb2Hash(str: string): number {
  let hash = 5381;
  for (let i = 0; i < str.length; i++) {
    hash = ((hash << 5) + hash + str.charCodeAt(i)) | 0;
  }
  return hash >>> 0;
}