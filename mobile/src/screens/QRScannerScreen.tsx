import React, { useState, useCallback, useRef, useEffect } from 'react';
import {
  View,
  Text,
  TouchableOpacity,
  StyleSheet,
  Animated,
  Vibration,
  Alert,
  Dimensions,
  StatusBar,
} from 'react-native';
import { CameraView, useCameraPermissions } from 'expo-camera';
import { useNavigation, useRoute, RouteProp } from '@react-navigation/native';
import { NativeStackNavigationProp } from '@react-navigation/native-stack';
import { Ionicons } from '@expo/vector-icons';
import { RootStackParamList, QRScanMode, QRScanResult } from '../types';

// ═══════════════════════════════════════════════════════════════════════
// Constants
// ═══════════════════════════════════════════════════════════════════════

const SCREEN_WIDTH = Dimensions.get('window').width;
const SCAN_FRAME_SIZE = SCREEN_WIDTH * 0.75;
const MIN_ZOOM = 0;
const MAX_ZOOM = 1;
const VIBRATION_DURATION = 100;

type ScreenRoute = RouteProp<RootStackParamList, 'QRScanner'>;

// ═══════════════════════════════════════════════════════════════════════
// Scan Modes Config
// ═══════════════════════════════════════════════════════════════════════

interface ScanModeConfig {
  key: QRScanMode;
  label: string;
  icon: keyof typeof Ionicons.glyphMap;
  description: string;
}

const SCAN_MODES: ScanModeConfig[] = [
  {
    key: 'device',
    label: 'Устройство',
    icon: 'videocam',
    description: 'Сканируйте QR-код устройства',
  },
  {
    key: 'part',
    label: 'Запчасть',
    icon: 'construct',
    description: 'Сканируйте QR-код запчасти',
  },
  {
    key: 'work_order',
    label: 'Наряд',
    icon: 'document-text',
    description: 'Сканируйте QR-код наряда',
  },
];

// ═══════════════════════════════════════════════════════════════════════
// QR Parsing
// ═══════════════════════════════════════════════════════════════════════

function parseQRData(data: string, mode: QRScanMode): QRScanResult | null {
  // Try JSON format first
  try {
    const parsed = JSON.parse(data);
    if (mode === 'device' && parsed.device_id) {
      return { type: 'device', id: parsed.device_id, label: parsed.device_name, raw: data };
    }
    if (mode === 'part' && parsed.part_id) {
      return { type: 'part', id: parsed.part_id, label: parsed.part_name, raw: data };
    }
    if (mode === 'work_order' && parsed.work_order_id) {
      return { type: 'work_order', id: parsed.work_order_id, label: parsed.work_order_ref, raw: data };
    }
  } catch {
    // Not JSON — try plain text formats
  }

  // Plain text formats: "DEVICE:abc123", "PART:xyz", "WO:456"
  const deviceMatch = data.match(/^DEVICE[:_](\S+)/i);
  if (deviceMatch && mode === 'device') {
    return { type: 'device', id: deviceMatch[1], raw: data };
  }

  const partMatch = data.match(/^PART[:_](\S+)/i);
  if (partMatch && mode === 'part') {
    return { type: 'part', id: partMatch[1], raw: data };
  }

  const woMatch = data.match(/^(?:WO|WORK_ORDER)[:_](\S+)/i);
  if (woMatch && mode === 'work_order') {
    return { type: 'work_order', id: woMatch[1], raw: data };
  }

  // Fallback: treat raw string as ID for the current mode
  if (data.length > 0 && data.length < 128) {
    return { type: mode, id: data, raw: data };
  }

  return null;
}

// ═══════════════════════════════════════════════════════════════════════
// Scan Animation — Scanning Line
// ═══════════════════════════════════════════════════════════════════════

function ScanAnimation({ visible }: { visible: boolean }) {
  const scanAnim = useRef(new Animated.Value(0)).current;

  useEffect(() => {
    if (!visible) {
      scanAnim.setValue(0);
      return;
    }

    const animation = Animated.loop(
      Animated.sequence([
        Animated.timing(scanAnim, {
          toValue: 1,
          duration: 2000,
          useNativeDriver: true,
        }),
        Animated.timing(scanAnim, {
          toValue: 0,
          duration: 2000,
          useNativeDriver: true,
        }),
      ]),
    );
    animation.start();
    return () => animation.stop();
  }, [visible, scanAnim]);

  if (!visible) return null;

  const translateY = scanAnim.interpolate({
    inputRange: [0, 1],
    outputRange: [0, SCAN_FRAME_SIZE],
  });

  return (
    <Animated.View
      style={[
        styles.scanLine,
        { transform: [{ translateY }] },
      ]}
    >
      <View style={styles.scanLineInner} />
    </Animated.View>
  );
}

// ═══════════════════════════════════════════════════════════════════════
// Flash Animation — Success Overlay
// ═══════════════════════════════════════════════════════════════════════

function FlashOverlay({ visible }: { visible: boolean }) {
  const opacity = useRef(new Animated.Value(0)).current;

  useEffect(() => {
    if (!visible) {
      opacity.setValue(0);
      return;
    }

    Animated.sequence([
      Animated.timing(opacity, { toValue: 0.6, duration: 100, useNativeDriver: true }),
      Animated.timing(opacity, { toValue: 0, duration: 300, useNativeDriver: true }),
    ]).start();
  }, [visible, opacity]);

  return (
    <Animated.View
      style={[styles.flashOverlay, { opacity }]}
      pointerEvents="none"
    />
  );
}

// ═══════════════════════════════════════════════════════════════════════
// Main Component
// ═══════════════════════════════════════════════════════════════════════

export default function QRScannerScreen() {
  const navigation = useNavigation<NativeStackNavigationProp<RootStackParamList>>();
  const route = useRoute<ScreenRoute>();
  const [permission, requestPermission] = useCameraPermissions();

  // State
  const [mode, setMode] = useState<QRScanMode>(route.params?.defaultMode || 'device');
  const [enableTorch, setEnableTorch] = useState(false);
  const [zoom, setZoom] = useState(0);
  const [scanned, setScanned] = useState(false);
  const [flashVisible, setFlashVisible] = useState(false);
  const [lastPinchDistance, setLastPinchDistance] = useState<number | null>(null);

  const pendingScanRef = useRef(false);

  // ═════════════════════════════════════════════════════════════════
  // Permission handling
  // ═════════════════════════════════════════════════════════════════

  useEffect(() => {
    if (!permission?.granted && !permission?.canAskAgain) {
      Alert.alert(
        'Доступ к камере',
        'Разрешите доступ к камере в настройках устройства для сканирования QR-кодов.',
        [{ text: 'OK', onPress: () => navigation.goBack() }],
      );
    }
  }, [permission, navigation]);

  // ═════════════════════════════════════════════════════════════════
  // Barcode handler
  // ═════════════════════════════════════════════════════════════════

  const handleBarCodeScanned = useCallback(
    ({ data }: { type: string; data: string }) => {
      if (scanned || pendingScanRef.current) return;
      pendingScanRef.current = true;

      // Vibration feedback
      Vibration.vibrate(VIBRATION_DURATION);

      // Flash animation
      setFlashVisible(true);
      setTimeout(() => setFlashVisible(false), 400);

      const result = parseQRData(data, mode);

      if (!result) {
        Alert.alert(
          'Неверный QR-код',
          `Не удалось распознать данные для режима "${mode}".`,
          [
            {
              text: 'Повторить',
              onPress: () => {
                setScanned(false);
                pendingScanRef.current = false;
              },
            },
          ],
        );
        return;
      }

      setScanned(true);

      // Deep link navigation
      handleScanNavigation(result);
    },
    [scanned, mode, navigation],
  );

  const handleScanNavigation = (result: QRScanResult) => {
    switch (result.type) {
      case 'work_order':
        navigation.navigate('WorkOrderDetail', { workOrderId: result.id });
        break;

      case 'device':
        Alert.alert(
          'Устройство найдено',
          `${result.label ? `📹 ${result.label}\n` : ''}ID: ${result.id}`,
          [
            {
              text: 'Сканировать снова',
              onPress: () => {
                setScanned(false);
                pendingScanRef.current = false;
              },
            },
            { text: 'Закрыть', onPress: () => navigation.goBack(), style: 'cancel' },
          ],
        );
        break;

      case 'part':
        Alert.alert(
          'Запчасть найдена',
          `${result.label ? `🔧 ${result.label}\n` : ''}ID: ${result.id}`,
          [
            {
              text: 'Сканировать снова',
              onPress: () => {
                setScanned(false);
                pendingScanRef.current = false;
              },
            },
            { text: 'Закрыть', onPress: () => navigation.goBack(), style: 'cancel' },
          ],
        );
        break;
    }
  };

  // ═════════════════════════════════════════════════════════════════
  // Pinch-to-zoom
  // ═════════════════════════════════════════════════════════════════

  const handleTouchStart = (e: any) => {
    if (e.nativeEvent.touches.length === 2) {
      const [t1, t2] = e.nativeEvent.touches;
      const distance = Math.sqrt(
        Math.pow(t2.pageX - t1.pageX, 2) + Math.pow(t2.pageY - t1.pageY, 2),
      );
      setLastPinchDistance(distance);
    }
  };

  const handleTouchMove = (e: any) => {
    if (e.nativeEvent.touches.length === 2 && lastPinchDistance !== null) {
      const [t1, t2] = e.nativeEvent.touches;
      const distance = Math.sqrt(
        Math.pow(t2.pageX - t1.pageX, 2) + Math.pow(t2.pageY - t1.pageY, 2),
      );

      const scale = distance / lastPinchDistance;
      const newZoom = Math.min(MAX_ZOOM, Math.max(MIN_ZOOM, zoom * (scale > 1 ? 1.05 : 0.95)));
      setZoom(newZoom);
      setLastPinchDistance(distance);
    }
  };

  const handleTouchEnd = () => {
    setLastPinchDistance(null);
  };

  // ═════════════════════════════════════════════════════════════════
  // Mode switch
  // ═════════════════════════════════════════════════════════════════

  const switchMode = (newMode: QRScanMode) => {
    setMode(newMode);
    setScanned(false);
    pendingScanRef.current = false;
  };

  // ═════════════════════════════════════════════════════════════════
  // Actions
  // ═════════════════════════════════════════════════════════════════

  const toggleTorch = () => setEnableTorch((prev) => !prev);

  const handleCancel = () => navigation.goBack();

  const handleRescan = () => {
    setScanned(false);
    pendingScanRef.current = false;
  };

  // ═════════════════════════════════════════════════════════════════
  // Render: Permission states
  // ═════════════════════════════════════════════════════════════════

  if (!permission) {
    return (
      <View style={styles.centered}>
        <Text style={styles.infoText}>Запрос разрешения камеры...</Text>
      </View>
    );
  }

  if (!permission.granted) {
    return (
      <View style={styles.centered}>
        <Ionicons name="camera-outline" size={64} color="#94a3b8" />
        <Text style={styles.errorText}>Нет доступа к камере</Text>
        <Text style={styles.errorSubtext}>
          Разрешите доступ к камере для сканирования QR-кодов
        </Text>
        <TouchableOpacity
          style={styles.permissionButton}
          onPress={requestPermission}
        >
          <Text style={styles.permissionButtonText}>Разрешить доступ</Text>
        </TouchableOpacity>
      </View>
    );
  }

  // ═════════════════════════════════════════════════════════════════
  // Render: Main scanner
  // ═════════════════════════════════════════════════════════════════

  return (
    <View style={styles.container}>
      <StatusBar barStyle="light-content" backgroundColor="#000" />

      {/* ═══ Camera ═══ */}
      <CameraView
        style={StyleSheet.absoluteFill}
        facing="back"
        enableTorch={enableTorch}
        zoom={zoom}
        onBarcodeScanned={scanned ? undefined : handleBarCodeScanned}
        barcodeScannerSettings={{ barcodeTypes: ['qr'] }}
        onTouchStart={handleTouchStart}
        onTouchMove={handleTouchMove}
        onTouchEnd={handleTouchEnd}
      />

      {/* ═══ Overlay ═══ */}
      <View style={styles.overlay} pointerEvents="box-none">
        {/* Darkened borders around scan frame */}
        <View style={styles.overlayTop} />
        <View style={styles.overlayMiddle}>
          <View style={styles.overlaySide} />
          <View style={styles.scanFrame}>
            {/* Corner markers */}
            <View style={[styles.corner, styles.cornerTL]} />
            <View style={[styles.corner, styles.cornerTR]} />
            <View style={[styles.corner, styles.cornerBL]} />
            <View style={[styles.corner, styles.cornerBR]} />

            {/* Animated scan line */}
            <ScanAnimation visible={!scanned} />
          </View>
          <View style={styles.overlaySide} />
        </View>
        <View style={styles.overlayBottom} />

        {/* ═══ Flash overlay ═══ */}
        <FlashOverlay visible={flashVisible} />

        {/* ═══ Top bar — Torch + Cancel ═══ */}
        <View style={styles.topBar}>
          <TouchableOpacity
            style={styles.topButton}
            onPress={handleCancel}
            hitSlop={{ top: 12, bottom: 12, left: 12, right: 12 }}
          >
            <Ionicons name="close" size={28} color="#fff" />
          </TouchableOpacity>

          <Text style={styles.topTitle}>Сканер QR</Text>

          <TouchableOpacity
            style={[styles.topButton, enableTorch && styles.topButtonActive]}
            onPress={toggleTorch}
            hitSlop={{ top: 12, bottom: 12, left: 12, right: 12 }}
          >
            <Ionicons
              name={enableTorch ? 'flashlight' : 'flashlight-outline'}
              size={24}
              color={enableTorch ? '#fbbf24' : '#fff'}
            />
          </TouchableOpacity>
        </View>

        {/* ═══ Mode selector ═══ */}
        <View style={styles.modeSelector}>
          {SCAN_MODES.map((scanMode) => (
            <TouchableOpacity
              key={scanMode.key}
              style={[
                styles.modeButton,
                mode === scanMode.key && styles.modeButtonActive,
              ]}
              onPress={() => switchMode(scanMode.key)}
              activeOpacity={0.7}
            >
              <Ionicons
                name={scanMode.icon}
                size={18}
                color={mode === scanMode.key ? '#fff' : '#94a3b8'}
              />
              <Text
                style={[
                  styles.modeButtonText,
                  mode === scanMode.key && styles.modeButtonTextActive,
                ]}
              >
                {scanMode.label}
              </Text>
            </TouchableOpacity>
          ))}
        </View>

        {/* ═══ Hint text ═══ */}
        <View style={styles.hintContainer}>
          <Text style={styles.hintText}>
            {scanned
              ? 'QR-код распознан'
              : SCAN_MODES.find((m) => m.key === mode)?.description || 'Наведите на QR-код'}
          </Text>
        </View>

        {/* ═══ Rescan button ═══ */}
        {scanned && (
          <TouchableOpacity style={styles.rescanButton} onPress={handleRescan}>
            <Ionicons name="scan-outline" size={20} color="#2563eb" />
            <Text style={styles.rescanText}>Сканировать снова</Text>
          </TouchableOpacity>
        )}
      </View>
    </View>
  );
}

// ═══════════════════════════════════════════════════════════════════════
// Styles
// ═══════════════════════════════════════════════════════════════════════

const styles = StyleSheet.create({
  container: {
    flex: 1,
    backgroundColor: '#000',
  },
  centered: {
    flex: 1,
    justifyContent: 'center',
    alignItems: 'center',
    backgroundColor: '#f1f5f9',
    padding: 24,
  },
  infoText: {
    fontSize: 16,
    color: '#64748b',
  },
  errorText: {
    fontSize: 18,
    fontWeight: '600',
    color: '#dc2626',
    marginTop: 16,
    marginBottom: 8,
  },
  errorSubtext: {
    fontSize: 14,
    color: '#64748b',
    textAlign: 'center',
    marginBottom: 24,
  },
  permissionButton: {
    backgroundColor: '#2563eb',
    paddingHorizontal: 24,
    paddingVertical: 14,
    borderRadius: 12,
  },
  permissionButtonText: {
    color: '#fff',
    fontSize: 16,
    fontWeight: '600',
  },

  // ── Overlay ────────────────────────────────────────────────────
  overlay: {
    flex: 1,
  },
  overlayTop: {
    flex: 1,
    backgroundColor: 'rgba(0,0,0,0.5)',
  },
  overlayMiddle: {
    flexDirection: 'row',
    height: SCAN_FRAME_SIZE,
  },
  overlaySide: {
    flex: 1,
    backgroundColor: 'rgba(0,0,0,0.5)',
  },
  overlayBottom: {
    flex: 2,
    backgroundColor: 'rgba(0,0,0,0.5)',
  },

  // ── Scan Frame ─────────────────────────────────────────────────
  scanFrame: {
    width: SCAN_FRAME_SIZE,
    height: SCAN_FRAME_SIZE,
    justifyContent: 'center',
    alignItems: 'center',
    overflow: 'hidden',
  },
  corner: {
    position: 'absolute',
    width: 24,
    height: 24,
    borderColor: '#60a5fa',
  },
  cornerTL: {
    top: 0,
    left: 0,
    borderTopWidth: 3,
    borderLeftWidth: 3,
    borderTopLeftRadius: 4,
  },
  cornerTR: {
    top: 0,
    right: 0,
    borderTopWidth: 3,
    borderRightWidth: 3,
    borderTopRightRadius: 4,
  },
  cornerBL: {
    bottom: 0,
    left: 0,
    borderBottomWidth: 3,
    borderLeftWidth: 3,
    borderBottomLeftRadius: 4,
  },
  cornerBR: {
    bottom: 0,
    right: 0,
    borderBottomWidth: 3,
    borderRightWidth: 3,
    borderBottomRightRadius: 4,
  },

  // ── Scan Line ──────────────────────────────────────────────────
  scanLine: {
    position: 'absolute',
    left: 4,
    right: 4,
    height: 3,
    zIndex: 10,
  },
  scanLineInner: {
    flex: 1,
    backgroundColor: '#60a5fa',
    opacity: 0.8,
    borderRadius: 2,
  },

  // ── Flash Overlay ──────────────────────────────────────────────
  flashOverlay: {
    ...StyleSheet.absoluteFillObject,
    backgroundColor: '#fff',
    zIndex: 20,
  },

  // ── Top Bar ────────────────────────────────────────────────────
  topBar: {
    position: 'absolute',
    top: 48,
    left: 0,
    right: 0,
    flexDirection: 'row',
    justifyContent: 'space-between',
    alignItems: 'center',
    paddingHorizontal: 16,
    zIndex: 30,
  },
  topButton: {
    width: 44,
    height: 44,
    borderRadius: 22,
    backgroundColor: 'rgba(0,0,0,0.4)',
    justifyContent: 'center',
    alignItems: 'center',
  },
  topButtonActive: {
    backgroundColor: 'rgba(251,191,36,0.2)',
  },
  topTitle: {
    fontSize: 18,
    fontWeight: '600',
    color: '#fff',
  },

  // ── Mode Selector ──────────────────────────────────────────────
  modeSelector: {
    position: 'absolute',
    bottom: 120,
    left: 16,
    right: 16,
    flexDirection: 'row',
    gap: 8,
    justifyContent: 'center',
    zIndex: 30,
  },
  modeButton: {
    flex: 1,
    flexDirection: 'row',
    alignItems: 'center',
    justifyContent: 'center',
    gap: 6,
    backgroundColor: 'rgba(0,0,0,0.5)',
    paddingVertical: 10,
    paddingHorizontal: 12,
    borderRadius: 10,
    borderWidth: 1,
    borderColor: 'rgba(255,255,255,0.15)',
  },
  modeButtonActive: {
    backgroundColor: '#2563eb',
    borderColor: '#2563eb',
  },
  modeButtonText: {
    fontSize: 13,
    fontWeight: '500',
    color: '#94a3b8',
  },
  modeButtonTextActive: {
    color: '#fff',
  },

  // ── Hint ───────────────────────────────────────────────────────
  hintContainer: {
    position: 'absolute',
    bottom: 84,
    left: 0,
    right: 0,
    alignItems: 'center',
    zIndex: 30,
  },
  hintText: {
    color: '#fff',
    fontSize: 14,
    backgroundColor: 'rgba(0,0,0,0.6)',
    paddingHorizontal: 16,
    paddingVertical: 8,
    borderRadius: 8,
    overflow: 'hidden',
  },

  // ── Rescan ─────────────────────────────────────────────────────
  rescanButton: {
    position: 'absolute',
    bottom: 48,
    left: 0,
    right: 0,
    alignItems: 'center',
    zIndex: 30,
    flexDirection: 'row',
    justifyContent: 'center',
    gap: 8,
  },
  rescanText: {
    color: '#60a5fa',
    fontSize: 16,
    fontWeight: '600',
    backgroundColor: 'rgba(255,255,255,0.9)',
    paddingHorizontal: 20,
    paddingVertical: 12,
    borderRadius: 8,
    overflow: 'hidden',
  },
});
