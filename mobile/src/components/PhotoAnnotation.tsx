import React, { useRef, useState, useCallback } from 'react';
import {
  View,
  Text,
  TouchableOpacity,
  StyleSheet,
  Alert,
  ActivityIndicator,
  Dimensions,
  Platform,
} from 'react-native';
import { WebView, WebViewMessageEvent } from 'react-native-webview';
import ViewShot from 'react-native-view-shot';
import * as FileSystem from 'expo-file-system';
import { workOrdersApi } from '../api/workOrders';

// ──────────────────────────────────────────────────
// Types
// ──────────────────────────────────────────────────

export type AnnotationTool = 'arrow' | 'text' | 'highlight' | 'freehand' | 'none';

export interface AnnotationResult {
  /** URI аннотированного изображения */
  annotatedUri: string;
  /** Метаданные аннотации (для воспроизведения) */
  metadata: AnnotationMetadata;
}

export interface AnnotationMetadata {
  tool: AnnotationTool;
  color: string;
  strokeWidth: number;
  /** JSON-строка с данными о нарисованных элементах */
  elements: string;
  /** Размеры оригинального изображения */
  imageWidth: number;
  imageHeight: number;
}

interface PhotoAnnotationProps {
  /** URI исходного фото */
  photoUri: string;
  /** ID work order для интеграции */
  workOrderId?: string;
  /** Callback после сохранения */
  onSave?: (result: AnnotationResult) => void;
  /** Callback при закрытии без сохранения */
  onClose?: () => void;
  /** Цвета инструментов по умолчанию */
  colors?: string[];
}

// ──────────────────────────────────────────────────
// Constants
// ──────────────────────────────────────────────────

const DEFAULT_COLORS = ['#ef4444', '#2563eb', '#22c55e', '#f59e0b', '#8b5cf6', '#000000', '#ffffff'];
const { width: SCREEN_WIDTH } = Dimensions.get('window');
const CANVAS_HEIGHT = SCREEN_WIDTH * 0.75; // 4:3 aspect ratio

// HTML-шаблон для WebView с HTML5 Canvas
const ANNOTATION_HTML = `
<!DOCTYPE html>
<html>
<head>
<meta name="viewport" content="width=device-width, initial-scale=1.0, maximum-scale=1.0, user-scalable=no">
<style>
  * { margin: 0; padding: 0; box-sizing: border-box; }
  body { 
    background: #1e293b; 
    display: flex; 
    justify-content: center; 
    align-items: center; 
    height: 100vh; 
    overflow: hidden;
    font-family: -apple-system, sans-serif;
  }
  .container { position: relative; max-width: 100%; max-height: 100%; }
  img { display: block; max-width: 100%; max-height: 80vh; }
  canvas { 
    position: absolute; 
    top: 0; 
    left: 0; 
    width: 100%; 
    height: 100%; 
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
<div class="container">
  <img id="sourceImage" crossorigin="anonymous" />
  <canvas id="canvas"></canvas>
  <input type="text" id="textInput" class="text-input" />
</div>

<script>
// ── Состояние ────────────────────────────────────
let currentTool = 'arrow';
let currentColor = '#ef4444';
let strokeWidth = 3;
let isDrawing = false;
let elements = [];
let startX = 0, startY = 0;
let imageLoaded = false;
let imageWidth = 0, imageHeight = 0;
let scaleX = 1, scaleY = 1;
let currentText = '';

const canvas = document.getElementById('canvas');
const ctx = canvas.getContext('2d');
const img = document.getElementById('sourceImage');
const textInput = document.getElementById('textInput');
const container = document.querySelector('.container');

// ── Загрузка изображения ─────────────────────────
function loadImage(uri) {
  img.onload = function() {
    imageWidth = img.naturalWidth;
    imageHeight = img.naturalHeight;

    // Рассчитываем масштаб
    const maxWidth = container.clientWidth;
    const maxHeight = window.innerHeight * 0.8;
    const scale = Math.min(maxWidth / imageWidth, maxHeight / imageHeight, 1);

    canvas.width = imageWidth * scale;
    canvas.height = imageHeight * scale;
    canvas.style.width = canvas.width + 'px';
    canvas.style.height = canvas.height + 'px';

    scaleX = canvas.width / imageWidth;
    scaleY = canvas.height / imageHeight;

    imageLoaded = true;
    redraw();
  };
  img.src = uri;
}

// ── Отрисовка ────────────────────────────────────
function redraw() {
  ctx.clearRect(0, 0, canvas.width, canvas.height);
  for (const el of elements) {
    ctx.save();
    ctx.strokeStyle = el.color;
    ctx.fillStyle = el.color;
    ctx.lineWidth = el.strokeWidth;
    ctx.lineCap = 'round';
    ctx.lineJoin = 'round';
    ctx.font = el.strokeWidth * 5 + 'px sans-serif';

    switch (el.type) {
      case 'arrow': {
        const angle = Math.atan2(el.ey - el.sy, el.ex - el.sx);
        const headLen = 15;
        ctx.beginPath();
        ctx.moveTo(el.sx, el.sy);
        ctx.lineTo(el.ex, el.ey);
        ctx.stroke();
        // Наконечник стрелки
        ctx.beginPath();
        ctx.moveTo(el.ex, el.ey);
        ctx.lineTo(el.ex - headLen * Math.cos(angle - Math.PI / 6), el.ey - headLen * Math.sin(angle - Math.PI / 6));
        ctx.lineTo(el.ex - headLen * Math.cos(angle + Math.PI / 6), el.ey - headLen * Math.sin(angle + Math.PI / 6));
        ctx.closePath();
        ctx.fill();
        break;
      }
      case 'text':
        ctx.fillStyle = el.color;
        ctx.font = 'bold ' + el.strokeWidth * 5 + 'px sans-serif';
        ctx.fillText(el.text || '', el.x, el.y);
        break;
      case 'highlight':
        ctx.globalAlpha = 0.3;
        ctx.fillStyle = el.color;
        ctx.fillRect(el.sx, el.sy, el.ex - el.sx, el.ey - el.sy);
        break;
      case 'freehand':
        if (el.points && el.points.length > 1) {
          ctx.beginPath();
          ctx.moveTo(el.points[0].x, el.points[0].y);
          for (let i = 1; i < el.points.length; i++) {
            ctx.lineTo(el.points[i].x, el.points[i].y);
          }
          ctx.stroke();
        }
        break;
    }
    ctx.restore();
  }
}

// ── Конвертация координат ────────────────────────
function getPos(e) {
  const rect = canvas.getBoundingClientRect();
  const touch = e.touches ? e.touches[0] : e;
  return {
    x: (touch.clientX - rect.left) * (canvas.width / rect.width),
    y: (touch.clientY - rect.top) * (canvas.height / rect.height),
  };
}

// ── События мыши/тач ────────────────────────────
function onPointerDown(e) {
  if (!imageLoaded) return;
  const pos = getPos(e);
  isDrawing = true;
  startX = pos.x;
  startY = pos.y;

  if (currentTool === 'text') {
    // Показываем текстовый инпут
    textInput.style.display = 'block';
    textInput.style.left = pos.x + 'px';
    textInput.style.top = pos.y + 'px';
    textInput.value = '';
    textInput.focus();
    return;
  }

  if (currentTool === 'freehand') {
    elements.push({
      type: 'freehand',
      color: currentColor,
      strokeWidth,
      points: [{ x: pos.x, y: pos.y }],
    });
  }
}

function onPointerMove(e) {
  if (!isDrawing || !imageLoaded) return;
  e.preventDefault();
  const pos = getPos(e);

  if (currentTool === 'freehand') {
    const lastEl = elements[elements.length - 1];
    if (lastEl && lastEl.type === 'freehand') {
      lastEl.points.push({ x: pos.x, y: pos.y });
      redraw();
    }
    return;
  }

  // Preview для arrow и highlight
  redraw();
  ctx.save();
  ctx.strokeStyle = currentColor;
  ctx.fillStyle = currentColor;
  ctx.lineWidth = strokeWidth;
  ctx.lineCap = 'round';

  if (currentTool === 'arrow') {
    const angle = Math.atan2(pos.y - startY, pos.x - startX);
    const headLen = 15;
    ctx.beginPath();
    ctx.moveTo(startX, startY);
    ctx.lineTo(pos.x, pos.y);
    ctx.stroke();
    ctx.beginPath();
    ctx.moveTo(pos.x, pos.y);
    ctx.lineTo(pos.x - headLen * Math.cos(angle - Math.PI / 6), pos.y - headLen * Math.sin(angle - Math.PI / 6));
    ctx.lineTo(pos.x - headLen * Math.cos(angle + Math.PI / 6), pos.y - headLen * Math.sin(angle + Math.PI / 6));
    ctx.closePath();
    ctx.fill();
  } else if (currentTool === 'highlight') {
    ctx.globalAlpha = 0.3;
    ctx.fillStyle = currentColor;
    ctx.fillRect(startX, startY, pos.x - startX, pos.y - startY);
  }
  ctx.restore();
}

function onPointerUp(e) {
  if (!isDrawing || !imageLoaded) return;
  isDrawing = false;
  const pos = getPos(e);

  if (currentTool === 'arrow') {
    elements.push({
      type: 'arrow',
      color: currentColor,
      strokeWidth,
      sx: startX, sy: startY,
      ex: pos.x, ey: pos.y,
    });
    redraw();
  } else if (currentTool === 'highlight') {
    elements.push({
      type: 'highlight',
      color: currentColor,
      strokeWidth,
      sx: startX, sy: startY,
      ex: pos.x, ey: pos.y,
    });
    redraw();
  }
}

// ── Текстовый ввод ───────────────────────────────
textInput.addEventListener('blur', function() {
  if (this.value.trim()) {
    elements.push({
      type: 'text',
      color: currentColor,
      strokeWidth,
      x: parseFloat(this.style.left),
      y: parseFloat(this.style.top),
      text: this.value,
    });
    redraw();
  }
  this.style.display = 'none';
});

textInput.addEventListener('keydown', function(e) {
  if (e.key === 'Enter') {
    this.blur();
  }
});

// ── Touch/Mouse события ──────────────────────────
canvas.addEventListener('mousedown', onPointerDown);
canvas.addEventListener('mousemove', onPointerMove);
canvas.addEventListener('mouseup', onPointerUp);
canvas.addEventListener('mouseleave', () => { isDrawing = false; });

canvas.addEventListener('touchstart', (e) => { e.preventDefault(); onPointerDown(e); });
canvas.addEventListener('touchmove', (e) => { e.preventDefault(); onPointerMove(e); });
canvas.addEventListener('touchend', (e) => { e.preventDefault(); onPointerUp(e); });

// ── API для React Native ─────────────────────────
window.addEventListener('message', function(event) {
  const data = JSON.parse(event.data);

  switch (data.type) {
    case 'setTool':
      currentTool = data.tool;
      currentText = '';
      textInput.style.display = 'none';
      break;
    case 'setColor':
      currentColor = data.color;
      break;
    case 'setStrokeWidth':
      strokeWidth = data.width;
      break;
    case 'loadImage':
      loadImage(data.uri);
      break;
    case 'getElements':
      // Отправляем данные об элементах обратно
      window.ReactNativeWebView.postMessage(JSON.stringify({
        type: 'elements',
        elements: JSON.stringify(elements),
        imageWidth: imageWidth,
        imageHeight: imageHeight,
        scaleX: scaleX,
        scaleY: scaleY,
      }));
      break;
    case 'clearAll':
      elements = [];
      redraw();
      break;
    case 'undo':
      elements.pop();
      redraw();
      break;
  }
});

// Сообщаем о готовности
window.ReactNativeWebView.postMessage(JSON.stringify({ type: 'ready' }));
</script>
</body>
</html>
`;

// ──────────────────────────────────────────────────
// Component
// ──────────────────────────────────────────────────

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

  // ── Инициализация WebView ───────────────────────

  const handleWebViewMessage = useCallback((event: WebViewMessageEvent) => {
    try {
      const data = JSON.parse(event.nativeEvent.data);
      if (data.type === 'ready') {
        setIsReady(true);
        // Загружаем изображение после готовности
        webViewRef.current?.postMessage(JSON.stringify({
          type: 'loadImage',
          uri: photoUri,
        }));
      }
    } catch {
      // ignore
    }
  }, [photoUri]);

  // ── Отправка команд в WebView ────────────────────

  const sendCommand = useCallback((command: Record<string, unknown>) => {
    webViewRef.current?.postMessage(JSON.stringify(command));
  }, []);

  const handleToolChange = useCallback((tool: AnnotationTool) => {
    setCurrentTool(tool);
    sendCommand({ type: 'setTool', tool });
  }, [sendCommand]);

  const handleColorChange = useCallback((color: string) => {
    setCurrentColor(color);
    sendCommand({ type: 'setColor', color });
  }, [sendCommand]);

  const handleUndo = useCallback(() => {
    sendCommand({ type: 'undo' });
  }, [sendCommand]);

  const handleClear = useCallback(() => {
    Alert.alert('Очистить всё', 'Удалить все аннотации?', [
      { text: 'Отмена', style: 'cancel' },
      {
        text: 'Очистить',
        style: 'destructive',
        onPress: () => sendCommand({ type: 'clearAll' }),
      },
    ]);
  }, [sendCommand]);

  // ── Сохранение аннотированного фото ────────────

  const handleSave = useCallback(async () => {
    if (!viewShotRef.current || isSaving) return;

    setIsSaving(true);
    try {
      // Захватываем WebView через ViewShot
      const uri = await (viewShotRef.current as any).capture?.();

      if (!uri) {
        throw new Error('Failed to capture annotated image');
      }

      // Копируем в постоянное хранилище
      const fileName = `annotated_${Date.now()}.jpg`;
      const dest = `${FileSystem.documentDirectory}annotations/${fileName}`;

      await FileSystem.makeDirectoryAsync(
        `${FileSystem.documentDirectory}annotations/`,
        { intermediates: true },
      );

      await FileSystem.copyAsync({
        from: uri,
        to: dest,
      });

      // Загружаем на сервер, если есть workOrderId
      let uploadedUrl: string | undefined;
      if (workOrderId) {
        try {
          const result = await workOrdersApi.uploadPhoto(workOrderId, dest);
          uploadedUrl = result.url;
        } catch (uploadError) {
          console.error('[PhotoAnnotation] Upload failed:', uploadError);
          // Сохраняем локально даже если upload не удался
        }
      }

      const result: AnnotationResult = {
        annotatedUri: uploadedUrl || dest,
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
  }, [isSaving, workOrderId, currentTool, currentColor, strokeWidth, onSave]);

  // ── Инструменты ─────────────────────────────────

  const tools: { key: AnnotationTool; icon: string; label: string }[] = [
    { key: 'arrow', icon: '➡️', label: 'Стрелка' },
    { key: 'text', icon: 'Aa', label: 'Текст' },
    { key: 'highlight', icon: '🖍️', label: 'Выделение' },
    { key: 'freehand', icon: '✏️', label: 'Рисование' },
  ];

  // ── Render ──────────────────────────────────────

  return (
    <View style={styles.container}>
      {/* Header */}
      <View style={styles.header}>
        <TouchableOpacity onPress={onClose} style={styles.headerBtn}>
          <Text style={styles.headerBtnText}>✕</Text>
        </TouchableOpacity>
        <Text style={styles.title}>Аннотация фото</Text>
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
          source={{ html: ANNOTATION_HTML }}
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
              {tool.key === 'text' ? (
                <Text
                  style={[
                    styles.toolIcon,
                    { fontSize: 14, fontWeight: 'bold' },
                  ]}
                >
                  {tool.icon}
                </Text>
              ) : (
                <Text style={styles.toolIcon}>{tool.icon}</Text>
              )}
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

        {/* Actions */}
        <View style={styles.actionsRow}>
          <TouchableOpacity style={styles.actionBtn} onPress={handleUndo}>
            <Text style={styles.actionBtnText}>↩ Отменить</Text>
          </TouchableOpacity>
          <TouchableOpacity
            style={[styles.actionBtn, styles.clearBtn]}
            onPress={handleClear}
          >
            <Text style={[styles.actionBtnText, { color: '#ef4444' }]}>
              ✕ Очистить
            </Text>
          </TouchableOpacity>
        </View>
      </View>
    </View>
  );
}

// ──────────────────────────────────────────────────
// Styles
// ──────────────────────────────────────────────────

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
  // ── Tools ──────────────────────────────────────
  toolsRow: {
    flexDirection: 'row',
    justifyContent: 'center',
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
  // ── Stroke width ───────────────────────────────
  strokeRow: {
    flexDirection: 'row',
    alignItems: 'center',
    justifyContent: 'center',
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
  // ── Colors ─────────────────────────────────────
  colorsRow: {
    flexDirection: 'row',
    justifyContent: 'center',
    gap: 8,
    paddingVertical: 8,
    paddingHorizontal: 16,
    borderBottomWidth: 1,
    borderBottomColor: '#334155',
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
  // ── Actions ────────────────────────────────────
  actionsRow: {
    flexDirection: 'row',
    justifyContent: 'center',
    gap: 12,
    paddingVertical: 8,
    paddingHorizontal: 16,
  },
  actionBtn: {
    paddingHorizontal: 20,
    paddingVertical: 8,
    borderRadius: 8,
    backgroundColor: '#334155',
  },
  clearBtn: {
    backgroundColor: '#450a0a',
  },
  actionBtnText: {
    fontSize: 13,
    color: '#cbd5e1',
    fontWeight: '600',
  },
});
