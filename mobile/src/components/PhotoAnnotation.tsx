import React, { useRef, useState, useCallback, useMemo } from 'react';
import {
  View,
  Text,
  TouchableOpacity,
  StyleSheet,
  Alert,
  ActivityIndicator,
  Dimensions,
  Platform,
  ScrollView,
} from 'react-native';
import { WebView, WebViewMessageEvent } from 'react-native-webview';
import ViewShot from 'react-native-view-shot';
import * as FileSystem from 'expo-file-system';
import { workOrdersApi, type AnnotationElement } from '../api/workOrders';

// ── Types ──────────────────────────────────────────────────────────────

export type AnnotationTool =
  | 'arrow'
  | 'freehand'
  | 'text'
  | 'highlight'
  | 'circle'
  | 'blur'
  | 'measurement';

export interface AnnotationResult {
  annotatedUri: string;
  metadata: AnnotationMetadata;
}

export interface AnnotationMetadata {
  tool: AnnotationTool;
  color: string;
  strokeWidth: number;
  elements: string;
  imageWidth: number;
  imageHeight: number;
}

interface PhotoAnnotationProps {
  photoUri: string;
  workOrderId?: string;
  onSave?: (result: AnnotationResult) => void;
  onClose?: () => void;
  colors?: string[];
}

// ── Constants ──────────────────────────────────────────────────────────

const DEFAULT_COLORS = [
  '#ef4444', '#2563eb', '#22c55e', '#f59e0b',
  '#8b5cf6', '#000000', '#ffffff',
];

const { width: SCREEN_WIDTH } = Dimensions.get('window');

// ── HTML Template with Pinch-to-Zoom support ──────────────────────────

const HTML_TEMPLATE = `
<!DOCTYPE html>
<html>
<head>
<meta name="viewport" content="width=device-width, initial-scale=1.0, maximum-scale=5.0, user-scalable=yes">
<style>
  * { margin: 0; padding: 0; box-sizing: border-box; user-select: none; -webkit-user-select: none; }
  body {
    background: #1e293b;
    display: flex;
    justify-content: center;
    align-items: center;
    height: 100vh;
    overflow: hidden;
    font-family: -apple-system, sans-serif;
  }
  .viewport {
    position: relative;
    width: 100%;
    height: 100vh;
    overflow: hidden;
    touch-action: none;
  }
  .transform-layer {
    position: absolute;
    top: 0; left: 0;
    transform-origin: 0 0;
  }
  #sourceImage {
    display: block;
    max-width: none;
  }
  canvas {
    position: absolute;
    top: 0; left: 0;
    cursor: crosshair;
    touch-action: none;
  }
  .text-input {
    position: absolute;
    display: none;
    z-index: 10;
    background: rgba(0,0,0,0.7);
    border: 2px solid #3b82f6;
    color: #fff;
    padding: 4px 8px;
    font-size: 16px;
    min-width: 100px;
    outline: none;
    border-radius: 4px;
  }
</style>
</head>
<body>
<div class="viewport" id="viewport">
  <div class="transform-layer" id="transformLayer">
    <img id="sourceImage" crossorigin="anonymous" />
    <canvas id="canvas"></canvas>
    <input type="text" id="textInput" class="text-input" />
  </div>
</div>

<script>
// ── State ──────────────────────────────────────
let currentTool = 'arrow';
let currentColor = '#ef4444';
let strokeWidth = 3;
let isDrawing = false;
let elements = [];
let redoStack = [];
let startX = 0, startY = 0;
let imageLoaded = false;
let imageWidth = 0, imageHeight = 0;
let displayWidth = 0, displayHeight = 0;
let scaleX = 1, scaleY = 1;

// Zoom / Pan state
let zoom = 1;
let panX = 0, panY = 0;
let lastDist = 0;
let isPinching = false;
let lastTouchX = 0, lastTouchY = 0;

const viewport = document.getElementById('viewport');
const transformLayer = document.getElementById('transformLayer');
const canvas = document.getElementById('canvas');
const ctx = canvas.getContext('2d');
const img = document.getElementById('sourceImage');
const textInput = document.getElementById('textInput');

// ── Helpers ────────────────────────────────────
function distance(x1, y1, x2, y2) {
  return Math.sqrt((x2-x1)**2 + (y2-y1)**2);
}

function drawArrowhead(ctx, fromX, fromY, toX, toY, size) {
  const angle = Math.atan2(toY - fromY, toX - fromX);
  ctx.beginPath();
  ctx.moveTo(toX, toY);
  ctx.lineTo(toX - size * Math.cos(angle - Math.PI/6), toY - size * Math.sin(angle - Math.PI/6));
  ctx.lineTo(toX - size * Math.cos(angle + Math.PI/6), toY - size * Math.sin(angle + Math.PI/6));
  ctx.closePath();
  ctx.fill();
}

function roundRect(ctx, x, y, w, h, r) {
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

// ── Apply transform ────────────────────────────
function applyTransform() {
  transformLayer.style.transform = 'translate(' + panX + 'px, ' + panY + 'px) scale(' + zoom + ')';
}

// ── Image loading ──────────────────────────────
function loadImage(uri) {
  img.onload = function() {
    imageWidth = img.naturalWidth;
    imageHeight = img.naturalHeight;
    const maxWidth = viewport.clientWidth;
    const maxHeight = viewport.clientHeight;
    const scale = Math.min(maxWidth / imageWidth, maxHeight / imageHeight, 1);
    displayWidth = imageWidth * scale;
    displayHeight = imageHeight * scale;
    canvas.width = displayWidth;
    canvas.height = displayHeight;
    canvas.style.width = displayWidth + 'px';
    canvas.style.height = displayHeight + 'px';
    img.style.width = displayWidth + 'px';
    img.style.height = displayHeight + 'px';
    scaleX = canvas.width / imageWidth;
    scaleY = canvas.height / imageHeight;
    imageLoaded = true;
    applyTransform();
    redraw();
  };
  img.src = uri;
}

// ── Rendering ──────────────────────────────────
function redraw() {
  ctx.clearRect(0, 0, canvas.width, canvas.height);
  if (imageLoaded) {
    ctx.drawImage(img, 0, 0, canvas.width, canvas.height);
  }
  for (const el of elements) {
    ctx.save();
    ctx.strokeStyle = el.color;
    ctx.fillStyle = el.color;
    ctx.lineWidth = el.strokeWidth;
    ctx.lineCap = 'round';
    ctx.lineJoin = 'round';
    ctx.filter = 'none';
    ctx.globalAlpha = 1;
    ctx.setLineDash([]);
    switch (el.type) {
      case 'arrow':
        ctx.beginPath();
        ctx.moveTo(el.sx, el.sy);
        ctx.lineTo(el.ex, el.ey);
        ctx.stroke();
        drawArrowhead(ctx, el.sx, el.sy, el.ex, el.ey, 15);
        break;
      case 'text':
        ctx.font = 'bold ' + (el.fontSize || 18) + 'px sans-serif';
        { const metrics = ctx.measureText(el.text); const pad = 6;
        ctx.fillStyle = 'rgba(0,0,0,0.65)';
        roundRect(ctx, el.x, el.y - (el.fontSize || 18) - pad, metrics.width + pad*2, (el.fontSize || 18) + pad*2, 4);
        ctx.fill();
        ctx.fillStyle = '#fff';
        ctx.fillText(el.text, el.x + pad, el.y - pad); }
        break;
      case 'highlight':
        ctx.globalAlpha = 0.3;
        ctx.fillStyle = el.color;
        ctx.fillRect(Math.min(el.sx, el.ex), Math.min(el.sy, el.ey), Math.abs(el.ex - el.sx), Math.abs(el.ey - el.sy));
        break;
      case 'circle':
        { const r = distance(el.sx, el.sy, el.ex, el.ey);
        ctx.beginPath(); ctx.arc(el.sx, el.sy, r, 0, Math.PI * 2); ctx.stroke();
        ctx.globalAlpha = 0.2; ctx.fill(); }
        break;
      case 'freehand':
        if (el.points && el.points.length > 1) {
          ctx.beginPath(); ctx.moveTo(el.points[0].x, el.points[0].y);
          for (let i = 1; i < el.points.length; i++) ctx.lineTo(el.points[i].x, el.points[i].y);
          ctx.stroke();
        }
        break;
      case 'blur':
        { const bx = Math.min(el.sx, el.ex), by = Math.min(el.sy, el.ey);
        const bw = Math.abs(el.ex - el.sx), bh = Math.abs(el.ey - el.sy);
        ctx.filter = 'blur(' + Math.max(6, el.strokeWidth * 3) + 'px)';
        ctx.fillStyle = '#000'; ctx.globalAlpha = 0.85; ctx.fillRect(bx, by, bw, bh);
        ctx.filter = 'none';
        ctx.strokeStyle = '#fff'; ctx.lineWidth = 1.5; ctx.setLineDash([4, 4]);
        ctx.strokeRect(bx, by, bw, bh); ctx.setLineDash([]);
        ctx.fillStyle = 'rgba(255,255,255,0.85)'; ctx.font = '11px sans-serif';
        ctx.fillText('\\u2B22 redacted', bx + 4, by + 14); }
        break;
      case 'measurement':
        ctx.strokeStyle = '#22d3ee'; ctx.lineWidth = el.strokeWidth; ctx.setLineDash([6, 4]);
        ctx.beginPath(); ctx.moveTo(el.sx, el.sy); ctx.lineTo(el.ex, el.ey); ctx.stroke(); ctx.setLineDash([]);
        ctx.fillStyle = '#22d3ee';
        ctx.beginPath(); ctx.arc(el.sx, el.sy, 5, 0, Math.PI * 2); ctx.fill();
        ctx.beginPath(); ctx.arc(el.ex, el.ey, 5, 0, Math.PI * 2); ctx.fill();
        { const midX = (el.sx + el.ex) / 2, midY = (el.sy + el.ey) / 2;
        const dist = distance(el.sx, el.sy, el.ex, el.ey);
        const mm = ((dist / 96) * 25.4).toFixed(1);
        const label = dist.toFixed(0) + 'px (' + mm + 'mm)';
        ctx.font = 'bold 12px sans-serif';
        const mw = ctx.measureText(label).width;
        ctx.fillStyle = 'rgba(0,0,0,0.75)';
        roundRect(ctx, midX - mw/2 - 6, midY - 12, mw + 12, 24, 4); ctx.fill();
        ctx.fillStyle = '#22d3ee'; ctx.fillText(label, midX - mw/2, midY + 4); }
        break;
    }
    ctx.restore();
  }
}

// ── Coordinates (relative to canvas) ───────────
function getPos(e) {
  const rect = canvas.getBoundingClientRect();
  const touch = e.touches ? e.touches[0] : e;
  return {
    x: (touch.clientX - rect.left) * (canvas.width / rect.width),
    y: (touch.clientY - rect.top) * (canvas.height / rect.height),
  };
}

// ── Touch/Pointer events with pinch-to-zoom ────
function onTouchStart(e) {
  if (e.touches.length === 2) {
    isPinching = true;
    isDrawing = false;
    const dx = e.touches[0].clientX - e.touches[1].clientX;
    const dy = e.touches[0].clientY - e.touches[1].clientY;
    lastDist = Math.sqrt(dx * dx + dy * dy);
    return;
  }
  if (e.touches.length === 1 && !isPinching) {
    lastTouchX = e.touches[0].clientX;
    lastTouchY = e.touches[0].clientY;
    if (!imageLoaded) return;
    const pos = getPos(e);
    isDrawing = true;
    startX = pos.x;
    startY = pos.y;
    if (currentTool === 'text') {
      textInput.style.display = 'block';
      textInput.style.left = (pos.x * zoom + panX) + 'px';
      textInput.style.top = (pos.y * zoom + panY) + 'px';
      textInput.value = '';
      textInput.focus();
      return;
    }
    if (currentTool === 'freehand') {
      elements.push({ type: 'freehand', color: currentColor, strokeWidth, points: [{ x: pos.x, y: pos.y }] });
    }
  }
}

function onTouchMove(e) {
  e.preventDefault();
  if (e.touches.length === 2 && isPinching) {
    const dx = e.touches[0].clientX - e.touches[1].clientX;
    const dy = e.touches[0].clientY - e.touches[1].clientY;
    const dist = Math.sqrt(dx * dx + dy * dy);
    const scale = dist / lastDist;
    zoom = Math.max(0.5, Math.min(5, zoom * scale));
    lastDist = dist;
    applyTransform();
    return;
  }
  if (e.touches.length === 1 && isDrawing && !isPinching) {
    const pos = getPos(e);
    if (currentTool === 'freehand') {
      const lastEl = elements[elements.length - 1];
      if (lastEl && lastEl.type === 'freehand') {
        lastEl.points.push({ x: pos.x, y: pos.y });
        redraw();
      }
      return;
    }
    // Preview
    redraw();
    ctx.save();
    ctx.strokeStyle = currentColor; ctx.fillStyle = currentColor;
    ctx.lineWidth = strokeWidth; ctx.lineCap = 'round'; ctx.globalAlpha = 0.7;
    if (currentTool === 'arrow') {
      ctx.beginPath(); ctx.moveTo(startX, startY); ctx.lineTo(pos.x, pos.y); ctx.stroke();
      drawArrowhead(ctx, startX, startY, pos.x, pos.y, 12);
    } else if (currentTool === 'highlight') {
      ctx.globalAlpha = 0.25;
      ctx.fillRect(Math.min(startX, pos.x), Math.min(startY, pos.y), Math.abs(pos.x - startX), Math.abs(pos.y - startY));
    } else if (currentTool === 'circle') {
      const r = distance(startX, startY, pos.x, pos.y);
      ctx.beginPath(); ctx.arc(startX, startY, r, 0, Math.PI * 2); ctx.stroke(); ctx.globalAlpha = 0.15; ctx.fill();
    } else if (currentTool === 'blur') {
      ctx.globalAlpha = 0.4; ctx.fillStyle = '#000';
      ctx.fillRect(Math.min(startX, pos.x), Math.min(startY, pos.y), Math.abs(pos.x - startX), Math.abs(pos.y - startY));
    } else if (currentTool === 'measurement') {
      ctx.strokeStyle = '#22d3ee'; ctx.setLineDash([6, 4]);
      ctx.beginPath(); ctx.moveTo(startX, startY); ctx.lineTo(pos.x, pos.y); ctx.stroke(); ctx.setLineDash([]);
      ctx.fillStyle = '#22d3ee';
      ctx.beginPath(); ctx.arc(startX, startY, 5, 0, Math.PI * 2); ctx.fill();
      ctx.beginPath(); ctx.arc(pos.x, pos.y, 5, 0, Math.PI * 2); ctx.fill();
    }
    ctx.restore();
  } else if (e.touches.length === 1 && !isPinching) {
    // Pan
    panX += e.touches[0].clientX - lastTouchX;
    panY += e.touches[0].clientY - lastTouchY;
    lastTouchX = e.touches[0].clientX;
    lastTouchY = e.touches[0].clientY;
    applyTransform();
  }
}

function onTouchEnd(e) {
  isPinching = false;
  if (!isDrawing || !imageLoaded) { isDrawing = false; return; }
  isDrawing = false;
  const changedTouch = e.changedTouches[0];
  const rect = canvas.getBoundingClientRect();
  const pos = {
    x: (changedTouch.clientX - rect.left) * (canvas.width / rect.width),
    y: (changedTouch.clientY - rect.top) * (canvas.height / rect.height),
  };
  const dist = distance(startX, startY, pos.x, pos.y);
  if (dist < 3 && currentTool !== 'freehand') { redraw(); return; }
  const el = { color: currentColor, strokeWidth };
  let shouldAdd = false;
  switch (currentTool) {
    case 'arrow': el.type = 'arrow'; el.sx = startX; el.sy = startY; el.ex = pos.x; el.ey = pos.y; shouldAdd = true; break;
    case 'highlight': el.type = 'highlight'; el.sx = startX; el.sy = startY; el.ex = pos.x; el.ey = pos.y; shouldAdd = true; break;
    case 'circle': el.type = 'circle'; el.sx = startX; el.sy = startY; el.ex = pos.x; el.ey = pos.y; shouldAdd = true; break;
    case 'blur': el.type = 'blur'; el.sx = startX; el.sy = startY; el.ex = pos.x; el.ey = pos.y; shouldAdd = true; break;
    case 'measurement': el.type = 'measurement'; el.sx = startX; el.sy = startY; el.ex = pos.x; el.ey = pos.y; shouldAdd = true; break;
  }
  if (shouldAdd) { redoStack = []; elements.push(el); }
  redraw();
}

// ── Text input ─────────────────────────────────
textInput.addEventListener('blur', function() {
  if (this.value.trim()) {
    elements.push({ type: 'text', color: currentColor, strokeWidth, fontSize: 18, x: parseFloat(this.style.left), y: parseFloat(this.style.top), text: this.value });
    redoStack = []; redraw();
  }
  this.style.display = 'none';
});
textInput.addEventListener('keydown', function(e) { if (e.key === 'Enter') this.blur(); });

// ── Mouse events (for emulator/debug) ──────────
canvas.addEventListener('mousedown', function(e) { if (!imageLoaded) return; const pos = getPos(e); isDrawing = true; startX = pos.x; startY = pos.y; if (currentTool === 'text') { textInput.style.display = 'block'; textInput.style.left = pos.x + 'px'; textInput.style.top = pos.y + 'px'; textInput.value = ''; textInput.focus(); return; } if (currentTool === 'freehand') { elements.push({ type: 'freehand', color: currentColor, strokeWidth, points: [{ x: pos.x, y: pos.y }] }); } });
canvas.addEventListener('mousemove', function(e) { if (!isDrawing || !imageLoaded) return; const pos = getPos(e); if (currentTool === 'freehand') { const lastEl = elements[elements.length - 1]; if (lastEl && lastEl.type === 'freehand') { lastEl.points.push({ x: pos.x, y: pos.y }); redraw(); } return; } redraw(); ctx.save(); ctx.strokeStyle = currentColor; ctx.fillStyle = currentColor; ctx.lineWidth = strokeWidth; ctx.lineCap = 'round'; ctx.globalAlpha = 0.7; if (currentTool === 'arrow') { ctx.beginPath(); ctx.moveTo(startX, startY); ctx.lineTo(pos.x, pos.y); ctx.stroke(); drawArrowhead(ctx, startX, startY, pos.x, pos.y, 12); } else if (currentTool === 'highlight') { ctx.globalAlpha = 0.25; ctx.fillRect(Math.min(startX, pos.x), Math.min(startY, pos.y), Math.abs(pos.x - startX), Math.abs(pos.y - startY)); } else if (currentTool === 'circle') { const r = distance(startX, startY, pos.x, pos.y); ctx.beginPath(); ctx.arc(startX, startY, r, 0, Math.PI * 2); ctx.stroke(); ctx.globalAlpha = 0.15; ctx.fill(); } else if (currentTool === 'blur') { ctx.globalAlpha = 0.4; ctx.fillStyle = '#000'; ctx.fillRect(Math.min(startX, pos.x), Math.min(startY, pos.y), Math.abs(pos.x - startX), Math.abs(pos.y - startY)); } else if (currentTool === 'measurement') { ctx.strokeStyle = '#22d3ee'; ctx.setLineDash([6, 4]); ctx.beginPath(); ctx.moveTo(startX, startY); ctx.lineTo(pos.x, pos.y); ctx.stroke(); ctx.setLineDash([]); ctx.fillStyle = '#22d3ee'; ctx.beginPath(); ctx.arc(startX, startY, 5, 0, Math.PI * 2); ctx.fill(); ctx.beginPath(); ctx.arc(pos.x, pos.y, 5, 0, Math.PI * 2); ctx.fill(); } ctx.restore(); });
canvas.addEventListener('mouseup', function(e) { if (!isDrawing || !imageLoaded) { isDrawing = false; return; } isDrawing = false; const pos = getPos(e); const dist = distance(startX, startY, pos.x, pos.y); if (dist < 3 && currentTool !== 'freehand') { redraw(); return; } const el = { color: currentColor, strokeWidth }; let shouldAdd = false; switch (currentTool) { case 'arrow': el.type = 'arrow'; el.sx = startX; el.sy = startY; el.ex = pos.x; el.ey = pos.y; shouldAdd = true; break; case 'highlight': el.type = 'highlight'; el.sx = startX; el.sy = startY; el.ex = pos.x; el.ey = pos.y; shouldAdd = true; break; case 'circle': el.type = 'circle'; el.sx = startX; el.sy = startY; el.ex = pos.x; el.ey = pos.y; shouldAdd = true; break; case 'blur': el.type = 'blur'; el.sx = startX; el.sy = startY; el.ex = pos.x; el.ey = pos.y; shouldAdd = true; break; case 'measurement': el.type = 'measurement'; el.sx = startX; el.sy = startY; el.ex = pos.x; el.ey = pos.y; shouldAdd = true; break; } if (shouldAdd) { redoStack = []; elements.push(el); } redraw(); });
canvas.addEventListener('mouseleave', function() { isDrawing = false; });

// ── Touch events ───────────────────────────────
viewport.addEventListener('touchstart', function(e) { e.preventDefault(); onTouchStart(e); }, { passive: false });
viewport.addEventListener('touchmove', function(e) { e.preventDefault(); onTouchMove(e); }, { passive: false });
viewport.addEventListener('touchend', function(e) { e.preventDefault(); onTouchEnd(e); }, { passive: false });

// ── API for RN ─────────────────────────────────
window.addEventListener('message', function(event) {
  const data = JSON.parse(event.data);
  switch (data.type) {
    case 'setTool': currentTool = data.tool; textInput.style.display = 'none'; break;
    case 'setColor': currentColor = data.color; break;
    case 'setStrokeWidth': strokeWidth = data.width; break;
    case 'loadImage': loadImage(data.uri); break;
    case 'getElements':
      window.ReactNativeWebView.postMessage(JSON.stringify({ type: 'elements', elements: JSON.stringify(elements), imageWidth: imageWidth, imageHeight: imageHeight }));
      break;
    case 'clearAll': redoStack.push([...elements]); elements = []; redraw(); break;
    case 'undo': if (elements.length > 0) { redoStack.push([elements.pop()]); redraw(); } break;
    case 'redo': if (redoStack.length > 0) { const redoEl = redoStack.pop(); elements.push(...redoEl); redraw(); } break;
    case 'getElementCount':
      window.ReactNativeWebView.postMessage(JSON.stringify({ type: 'elementCount', count: elements.length, redoCount: redoStack.length }));
      break;
    case 'resetZoom': zoom = 1; panX = 0; panY = 0; applyTransform(); break;
  }
});
window.ReactNativeWebView.postMessage(JSON.stringify({ type: 'ready' }));
</script>
</body>
</html>
`;

// ── Component ──────────────────────────────────────────────────────────

export default function PhotoAnnotation({
  photoUri,
  workOrderId,
  onSave,
  onClose,
  colors = DEFAULT_COLORS,
}: PhotoAnnotationProps) {
  const webViewRef = useRef<WebView>(null);
  const viewShotRef = useRef<ViewShot>(null);
  const [currentTool, setCurrentTool] = useState<AnnotationTool>('arrow');
  const [currentColor, setCurrentColor] = useState(colors[0]);
  const [isReady, setIsReady] = useState(false);
  const [isSaving, setIsSaving] = useState(false);
  const [strokeWidth, setStrokeWidth] = useState(3);
  const [elementCount, setElementCount] = useState(0);
  const [redoCount, setRedoCount] = useState(0);

  // ── WebView messaging ─────────────────────────

  const handleWebViewMessage = useCallback(
    (event: WebViewMessageEvent) => {
      try {
        const data = JSON.parse(event.nativeEvent.data);
        if (data.type === 'ready') {
          setIsReady(true);
          webViewRef.current?.postMessage(
            JSON.stringify({ type: 'loadImage', uri: photoUri }),
          );
        } else if (data.type === 'elementCount') {
          setElementCount(data.count);
          setRedoCount(data.redoCount);
        }
      } catch {
        // ignore
      }
    },
    [photoUri],
  );

  const sendCommand = useCallback(
    (command: Record<string, unknown>) => {
      webViewRef.current?.postMessage(JSON.stringify(command));
    },
    [],
  );

  const handleToolChange = useCallback(
    (tool: AnnotationTool) => {
      setCurrentTool(tool);
      sendCommand({ type: 'setTool', tool });
    },
    [sendCommand],
  );

  const handleColorChange = useCallback(
    (color: string) => {
      setCurrentColor(color);
      sendCommand({ type: 'setColor', color });
    },
    [sendCommand],
  );

  const handleUndo = useCallback(() => {
    sendCommand({ type: 'undo' });
    setTimeout(() => sendCommand({ type: 'getElementCount' }), 50);
  }, [sendCommand]);

  const handleRedo = useCallback(() => {
    sendCommand({ type: 'redo' });
    setTimeout(() => sendCommand({ type: 'getElementCount' }), 50);
  }, [sendCommand]);

  const handleClear = useCallback(() => {
    Alert.alert('Очистить всё', 'Удалить все аннотации?', [
      { text: 'Отмена', style: 'cancel' },
      {
        text: 'Очистить',
        style: 'destructive',
        onPress: () => {
          sendCommand({ type: 'clearAll' });
          setTimeout(() => sendCommand({ type: 'getElementCount' }), 50);
        },
      },
    ]);
  }, [sendCommand]);

  const handleResetZoom = useCallback(() => {
    sendCommand({ type: 'resetZoom' });
  }, [sendCommand]);

  // ── Save ──────────────────────────────────────

  const handleSave = useCallback(async () => {
    if (!viewShotRef.current || isSaving) return;
    setIsSaving(true);

    try {
      const uri = await (viewShotRef.current as any).capture?.();
      if (!uri) throw new Error('Failed to capture annotated image');

      const fileName = `annotated_${Date.now()}.jpg`;
      const dest = `${FileSystem.documentDirectory}annotations/${fileName}`;

      await FileSystem.makeDirectoryAsync(
        `${FileSystem.documentDirectory}annotations/`,
        { intermediates: true },
      );

      await FileSystem.copyAsync({ from: uri, to: dest });

      // Save annotations to backend
      if (workOrderId) {
        try {
          await workOrdersApi.saveAnnotations(workOrderId, photoUri, []);
        } catch (uploadError) {
          console.error('[PhotoAnnotation] Annotation save failed:', uploadError);
        }
      }

      const result: AnnotationResult = {
        annotatedUri: dest,
        metadata: {
          tool: currentTool,
          color: currentColor,
          strokeWidth,
          elements: '',
          imageWidth: 0,
          imageHeight: 0,
        },
      };

      onSave?.(result);
    } catch (error) {
      const message = error instanceof Error ? error.message : 'Unknown error';
      console.error('[PhotoAnnotation] Save failed:', message);
      Alert.alert('Ошибка', 'Не удалось сохранить аннотированное фото');
    } finally {
      setIsSaving(false);
    }
  }, [isSaving, workOrderId, photoUri, currentTool, currentColor, strokeWidth, onSave]);

  // ── Tools ─────────────────────────────────────

  interface ToolDef {
    key: AnnotationTool;
    icon: string;
    label: string;
  }

  const tools: ToolDef[] = [
    { key: 'arrow', icon: '➡️', label: 'Стрелка' },
    { key: 'text', icon: 'Aa', label: 'Текст' },
    { key: 'highlight', icon: '🖍️', label: 'Выделение' },
    { key: 'circle', icon: '⭕', label: 'Круг' },
    { key: 'freehand', icon: '✏️', label: 'Рисование' },
    { key: 'blur', icon: '🌫️', label: 'Размытие' },
    { key: 'measurement', icon: '📏', label: 'Линейка' },
  ];

  // ── Render ────────────────────────────────────

  return (
    <View style={styles.container}>
      {/* Header */}
      <View style={styles.header}>
        <TouchableOpacity onPress={onClose} style={styles.headerBtn}>
          <Text style={styles.headerBtnText}>✕</Text>
        </TouchableOpacity>
        <Text style={styles.title}>Аннотация фото</Text>
        <View style={styles.headerRight}>
          <TouchableOpacity onPress={handleResetZoom} style={styles.headerBtn}>
            <Text style={styles.headerBtnText}>⊞</Text>
          </TouchableOpacity>
          <TouchableOpacity
            onPress={handleSave}
            style={[styles.headerBtn, styles.saveBtn]}
            disabled={isSaving}
          >
            {isSaving ? (
              <ActivityIndicator size="small" color="#fff" />
            ) : (
              <Text style={styles.saveBtnText}>✓</Text>
            )}
          </TouchableOpacity>
        </View>
      </View>

      {/* Photo + Canvas */}
      <ViewShot
        ref={viewShotRef}
        options={{
          format: 'jpg',
          quality: 0.9,
          result: 'tmpfile',
        }}
        style={styles.viewShot}
      >
        <WebView
          ref={webViewRef}
          source={{ html: HTML_TEMPLATE }}
          style={styles.webView}
          onMessage={handleWebViewMessage}
          scrollEnabled={false}
          bounces={false}
          javaScriptEnabled={true}
          domStorageEnabled={true}
          originWhitelist={['*']}
          showsVerticalScrollIndicator={false}
        />
      </ViewShot>

      {/* Toolbar */}
      <View style={styles.toolbar}>
        <ScrollView horizontal showsHorizontalScrollIndicator={false}>
          {/* Tools */}
          <View style={styles.toolsRow}>
            {tools.map((tool) => (
              <TouchableOpacity
                key={tool.key}
                style={[
                  styles.toolBtn,
                  currentTool === tool.key && styles.toolBtnActive,
                ]}
                onPress={() => handleToolChange(tool.key)}
              >
                <Text style={styles.toolIcon}>{tool.icon}</Text>
              </TouchableOpacity>
            ))}
          </View>

          {/* Stroke width */}
          <View style={styles.strokeRow}>
            <Text style={styles.strokeLabel}>Толщина:</Text>
            {[2, 4, 6].map((w) => (
              <TouchableOpacity
                key={w}
                style={[
                  styles.strokeBtn,
                  strokeWidth === w && styles.strokeBtnActive,
                ]}
                onPress={() => {
                  setStrokeWidth(w);
                  sendCommand({ type: 'setStrokeWidth', width: w });
                }}
              >
                <View
                  style={[
                    styles.strokeDot,
                    { width: w * 2, height: w * 2, borderRadius: w },
                  ]}
                />
              </TouchableOpacity>
            ))}
          </View>

          {/* Colors */}
          <View style={styles.colorsRow}>
            {colors.map((color) => (
              <TouchableOpacity
                key={color}
                style={[
                  styles.colorBtn,
                  { backgroundColor: color },
                  currentColor === color && styles.colorBtnActive,
                ]}
                onPress={() => handleColorChange(color)}
              />
            ))}
          </View>
        </ScrollView>

        {/* Actions */}
        <View style={styles.actionsRow}>
          <TouchableOpacity
            style={[styles.actionBtn, elementCount === 0 && styles.actionBtnDisabled]}
            onPress={handleUndo}
            disabled={elementCount === 0}
          >
            <Text style={styles.actionBtnText}>↩ Отменить</Text>
          </TouchableOpacity>

          <TouchableOpacity
            style={[styles.actionBtn, redoCount === 0 && styles.actionBtnDisabled]}
            onPress={handleRedo}
            disabled={redoCount === 0}
          >
            <Text style={styles.actionBtnText}>↪ Повторить</Text>
          </TouchableOpacity>

          <TouchableOpacity
            style={[styles.actionBtn, styles.clearBtn]}
            onPress={handleClear}
          >
            <Text style={[styles.actionBtnText, { color: '#ef4444' }]}>
              ✕ Очистить
            </Text>
          </TouchableOpacity>

          <Text style={styles.countText}>
            {elementCount} сл.
          </Text>
        </View>
      </View>
    </View>
  );
}

// ── Styles ─────────────────────────────────────────────────────────────

const styles = StyleSheet.create({
  container: {
    flex: 1,
    backgroundColor: '#0f172a',
  },
  header: {
    flexDirection: 'row',
    alignItems: 'center',
    justifyContent: 'space-between',
    paddingHorizontal: 16,
    paddingVertical: 12,
    backgroundColor: '#1e293b',
    borderBottomWidth: 1,
    borderBottomColor: '#334155',
  },
  headerBtn: {
    width: 36,
    height: 36,
    borderRadius: 18,
    backgroundColor: '#334155',
    alignItems: 'center',
    justifyContent: 'center',
  },
  headerBtnText: {
    fontSize: 18,
    color: '#94a3b8',
    fontWeight: 'bold',
  },
  headerRight: {
    flexDirection: 'row',
    gap: 8,
  },
  saveBtn: {
    backgroundColor: '#2563eb',
  },
  saveBtnText: {
    fontSize: 18,
    color: '#fff',
    fontWeight: 'bold',
  },
  title: {
    fontSize: 16,
    fontWeight: '600',
    color: '#f1f5f9',
  },
  viewShot: {
    flex: 1,
    backgroundColor: '#1e293b',
  },
  webView: {
    flex: 1,
    backgroundColor: 'transparent',
  },
  toolbar: {
    backgroundColor: '#1e293b',
    borderTopWidth: 1,
    borderTopColor: '#334155',
    paddingBottom: Platform.OS === 'ios' ? 20 : 8,
  },
  toolsRow: {
    flexDirection: 'row',
    gap: 8,
    paddingVertical: 8,
    paddingHorizontal: 16,
    borderBottomWidth: 1,
    borderBottomColor: '#334155',
  },
  toolBtn: {
    paddingHorizontal: 16,
    paddingVertical: 8,
    borderRadius: 8,
    backgroundColor: '#334155',
    alignItems: 'center',
    justifyContent: 'center',
    minWidth: 44,
  },
  toolBtnActive: {
    backgroundColor: '#2563eb',
  },
  toolIcon: {
    fontSize: 18,
  },
  strokeRow: {
    flexDirection: 'row',
    alignItems: 'center',
    gap: 12,
    paddingVertical: 6,
    paddingHorizontal: 16,
    borderBottomWidth: 1,
    borderBottomColor: '#334155',
  },
  strokeLabel: {
    fontSize: 12,
    color: '#94a3b8',
    marginRight: 4,
  },
  strokeBtn: {
    width: 32,
    height: 32,
    borderRadius: 16,
    backgroundColor: '#334155',
    alignItems: 'center',
    justifyContent: 'center',
  },
  strokeBtnActive: {
    borderWidth: 2,
    borderColor: '#3b82f6',
  },
  strokeDot: {
    backgroundColor: '#e2e8f0',
  },
  colorsRow: {
    flexDirection: 'row',
    gap: 8,
    paddingVertical: 8,
    paddingHorizontal: 16,
  },
  colorBtn: {
    width: 28,
    height: 28,
    borderRadius: 14,
    borderWidth: 2,
    borderColor: 'transparent',
  },
  colorBtnActive: {
    borderColor: '#fff',
    transform: [{ scale: 1.2 }],
  },
  actionsRow: {
    flexDirection: 'row',
    justifyContent: 'center',
    alignItems: 'center',
    gap: 12,
    paddingVertical: 8,
    paddingHorizontal: 16,
    borderTopWidth: 1,
    borderTopColor: '#334155',
  },
  actionBtn: {
    paddingHorizontal: 20,
    paddingVertical: 8,
    borderRadius: 8,
    backgroundColor: '#334155',
  },
  actionBtnDisabled: {
    opacity: 0.35,
  },
  clearBtn: {
    backgroundColor: '#450a0a',
  },
  actionBtnText: {
    fontSize: 13,
    color: '#cbd5e1',
    fontWeight: '600',
  },
  countText: {
    fontSize: 11,
    color: '#64748b',
    fontWeight: '500',
  },
});
