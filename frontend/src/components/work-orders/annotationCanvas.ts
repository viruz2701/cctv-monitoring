// ═══════════════════════════════════════════════════════════════════════
// annotationCanvas.ts — Canvas drawing utilities for PhotoAnnotation
// P1-PHOTO: Shared drawing functions for all annotation element types
// ═══════════════════════════════════════════════════════════════════════

import {
  type AnnotationElement,
  type ArrowElement,
  type FreehandElement,
  type TextElement,
  type HighlightElement,
  type CircleElement,
  type BlurElement,
  type MeasurementElement,
  type Point,
  type AnnotationTool,
  distance,
  pxToMm,
} from './annotationTypes';

// ── Arrowhead ───────────────────────────────────────────────────────────

function drawArrowhead(
  ctx: CanvasRenderingContext2D,
  from: Point,
  to: Point,
  size: number = 12,
): void {
  const angle = Math.atan2(to.y - from.y, to.x - from.x);
  ctx.beginPath();
  ctx.moveTo(to.x, to.y);
  ctx.lineTo(
    to.x - size * Math.cos(angle - Math.PI / 6),
    to.y - size * Math.sin(angle - Math.PI / 6),
  );
  ctx.lineTo(
    to.x - size * Math.cos(angle + Math.PI / 6),
    to.y - size * Math.sin(angle + Math.PI / 6),
  );
  ctx.closePath();
  ctx.fill();
}

// ── Rounded rectangle ───────────────────────────────────────────────────

function roundRect(
  ctx: CanvasRenderingContext2D,
  x: number,
  y: number,
  w: number,
  h: number,
  r: number,
): void {
  ctx.beginPath();
  ctx.moveTo(x + r, y);
  ctx.lineTo(x + w - r, y);
  ctx.arcTo(x + w, y, x + w, y + r, r);
  ctx.lineTo(x + w, y + h - r);
  ctx.arcTo(x + w, y + h, x + w - r, y + h, r);
  ctx.lineTo(x + r, y + h);
  ctx.arcTo(x, y + h, x, y + h - r, r);
  ctx.lineTo(x, y + r);
  ctx.arcTo(x, y, x + r, y, r);
  ctx.closePath();
}

// ── Element drawing ─────────────────────────────────────────────────────

export function drawElement(
  ctx: CanvasRenderingContext2D,
  el: AnnotationElement,
): void {
  ctx.save();
  ctx.strokeStyle = el.color;
  ctx.fillStyle = el.color;
  ctx.lineWidth = el.strokeWidth;
  ctx.lineCap = 'round';
  ctx.lineJoin = 'round';

  switch (el.type) {
    case 'arrow': {
      const arrow = el as ArrowElement;
      ctx.beginPath();
      ctx.moveTo(arrow.start.x, arrow.start.y);
      ctx.lineTo(arrow.end.x, arrow.end.y);
      ctx.stroke();
      drawArrowhead(ctx, arrow.start, arrow.end);
      break;
    }

    case 'freehand': {
      const fh = el as FreehandElement;
      if (fh.points.length < 2) break;
      ctx.beginPath();
      ctx.moveTo(fh.points[0].x, fh.points[0].y);
      for (let i = 1; i < fh.points.length; i++) {
        ctx.lineTo(fh.points[i].x, fh.points[i].y);
      }
      ctx.stroke();
      break;
    }

    case 'text': {
      const txt = el as TextElement;
      ctx.font = `bold ${txt.fontSize}px Inter, system-ui, sans-serif`;
      const metrics = ctx.measureText(txt.text);
      const pad = 6;
      ctx.fillStyle = 'rgba(0,0,0,0.65)';
      const bgX = txt.position.x;
      const bgY = txt.position.y - txt.fontSize - pad;
      roundRect(ctx, bgX, bgY, metrics.width + pad * 2, txt.fontSize + pad * 2, 4);
      ctx.fill();
      ctx.fillStyle = '#ffffff';
      ctx.fillText(txt.text, txt.position.x + pad, txt.position.y - pad);
      break;
    }

    case 'highlight': {
      const hl = el as HighlightElement;
      ctx.globalAlpha = 0.3;
      ctx.fillStyle = el.color;
      ctx.fillRect(
        Math.min(hl.start.x, hl.end.x),
        Math.min(hl.start.y, hl.end.y),
        Math.abs(hl.end.x - hl.start.x),
        Math.abs(hl.end.y - hl.start.y),
      );
      break;
    }

    case 'circle': {
      const cir = el as CircleElement;
      ctx.beginPath();
      ctx.arc(cir.center.x, cir.center.y, cir.radius, 0, Math.PI * 2);
      ctx.stroke();
      ctx.globalAlpha = 0.2;
      ctx.fillStyle = el.color;
      ctx.fill();
      break;
    }

    case 'blur': {
      const bl = el as BlurElement;
      const bx = Math.min(bl.start.x, bl.end.x);
      const by = Math.min(bl.start.y, bl.end.y);
      const bw = Math.abs(bl.end.x - bl.start.x);
      const bh = Math.abs(bl.end.y - bl.start.y);
      ctx.filter = `blur(${Math.max(4, el.strokeWidth * 3)}px)`;
      ctx.fillStyle = '#000000';
      ctx.globalAlpha = 0.85;
      ctx.fillRect(bx, by, bw, bh);
      ctx.filter = 'none';
      ctx.strokeStyle = '#ffffff';
      ctx.lineWidth = 1.5;
      ctx.setLineDash([4, 4]);
      ctx.strokeRect(bx, by, bw, bh);
      ctx.setLineDash([]);
      ctx.fillStyle = 'rgba(255,255,255,0.85)';
      ctx.font = '11px Inter, system-ui, sans-serif';
      ctx.fillText('⬡ redacted', bx + 4, by + 14);
      break;
    }

    case 'measurement': {
      const m = el as MeasurementElement;
      ctx.strokeStyle = '#22d3ee';
      ctx.lineWidth = el.strokeWidth;
      ctx.setLineDash([6, 4]);
      ctx.beginPath();
      ctx.moveTo(m.start.x, m.start.y);
      ctx.lineTo(m.end.x, m.end.y);
      ctx.stroke();
      ctx.setLineDash([]);
      ctx.fillStyle = '#22d3ee';
      ctx.beginPath();
      ctx.arc(m.start.x, m.start.y, 5, 0, Math.PI * 2);
      ctx.fill();
      ctx.beginPath();
      ctx.arc(m.end.x, m.end.y, 5, 0, Math.PI * 2);
      ctx.fill();
      const midX = (m.start.x + m.end.x) / 2;
      const midY = (m.start.y + m.end.y) / 2;
      const label = `${m.lengthPx.toFixed(0)}px (${pxToMm(m.lengthPx).toFixed(1)}mm)`;
      ctx.font = 'bold 12px Inter, system-ui, sans-serif';
      const mw = ctx.measureText(label).width;
      ctx.fillStyle = 'rgba(0,0,0,0.75)';
      roundRect(ctx, midX - mw / 2 - 6, midY - 12, mw + 12, 24, 4);
      ctx.fill();
      ctx.fillStyle = '#22d3ee';
      ctx.fillText(label, midX - mw / 2, midY + 4);
      break;
    }
  }

  ctx.restore();
}

// ── Preview drawing (in-progress element) ───────────────────────────────

export function drawPreview(
  ctx: CanvasRenderingContext2D,
  tool: AnnotationTool,
  color: string,
  sw: number,
  start: Point,
  end: Point,
  points: Point[],
): void {
  ctx.save();
  ctx.strokeStyle = color;
  ctx.fillStyle = color;
  ctx.lineWidth = sw;
  ctx.lineCap = 'round';
  ctx.lineJoin = 'round';
  ctx.globalAlpha = 0.7;

  switch (tool) {
    case 'arrow': {
      ctx.beginPath();
      ctx.moveTo(start.x, start.y);
      ctx.lineTo(end.x, end.y);
      ctx.stroke();
      drawArrowhead(ctx, start, end);
      break;
    }
    case 'freehand': {
      if (points.length < 2) break;
      ctx.beginPath();
      ctx.moveTo(points[0].x, points[0].y);
      for (let i = 1; i < points.length; i++) {
        ctx.lineTo(points[i].x, points[i].y);
      }
      ctx.stroke();
      break;
    }
    case 'highlight': {
      ctx.globalAlpha = 0.25;
      ctx.fillStyle = color;
      ctx.fillRect(
        Math.min(start.x, end.x),
        Math.min(start.y, end.y),
        Math.abs(end.x - start.x),
        Math.abs(end.y - start.y),
      );
      break;
    }
    case 'circle': {
      const r = distance(start, end);
      ctx.beginPath();
      ctx.arc(start.x, start.y, r, 0, Math.PI * 2);
      ctx.stroke();
      ctx.globalAlpha = 0.15;
      ctx.fill();
      break;
    }
    case 'blur': {
      ctx.globalAlpha = 0.4;
      ctx.fillStyle = '#000';
      ctx.fillRect(
        Math.min(start.x, end.x),
        Math.min(start.y, end.y),
        Math.abs(end.x - start.x),
        Math.abs(end.y - start.y),
      );
      break;
    }
    case 'measurement': {
      ctx.strokeStyle = '#22d3ee';
      ctx.setLineDash([6, 4]);
      ctx.beginPath();
      ctx.moveTo(start.x, start.y);
      ctx.lineTo(end.x, end.y);
      ctx.stroke();
      ctx.setLineDash([]);
      ctx.fillStyle = '#22d3ee';
      ctx.beginPath();
      ctx.arc(start.x, start.y, 5, 0, Math.PI * 2);
      ctx.fill();
      ctx.beginPath();
      ctx.arc(end.x, end.y, 5, 0, Math.PI * 2);
      ctx.fill();
      break;
    }
  }

  ctx.restore();
}

// ── Export canvas to PNG with watermark ─────────────────────────────────

export async function exportToPng(
  canvas: HTMLCanvasElement,
  username?: string,
): Promise<string> {
  const dataUrl = canvas.toDataURL('image/png');

  if (!username) return dataUrl;

  // Add watermark with username
  const img = new Image();
  img.src = dataUrl;
  await new Promise((resolve) => { img.onload = resolve; });

  const offscreen = document.createElement('canvas');
  offscreen.width = canvas.width;
  offscreen.height = canvas.height;
  const octx = offscreen.getContext('2d')!;

  octx.drawImage(img, 0, 0, canvas.width, canvas.height);

  // Watermark
  octx.save();
  octx.fillStyle = 'rgba(255,255,255,0.15)';
  octx.font = '14px Inter, system-ui, sans-serif';
  octx.fillText(
    `© ${new Date().getFullYear()} CCTV Health Monitor — ${username}`,
    12,
    canvas.height - 12,
  );
  octx.restore();

  return offscreen.toDataURL('image/png');
}
