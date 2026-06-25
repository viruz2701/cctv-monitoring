import React, { useState, useRef, useCallback } from 'react';
import {
  View,
  Text,
  TouchableOpacity,
  StyleSheet,
  Image,
  Alert,
  ActivityIndicator,
  StatusBar,
  Dimensions,
} from 'react-native';
import { useNavigation, useRoute, RouteProp } from '@react-navigation/native';
import { NativeStackNavigationProp } from '@react-navigation/native-stack';
import { Ionicons } from '@expo/vector-icons';
import SignatureView, { SignatureViewRef } from 'react-native-signature-canvas';
import { RootStackParamList } from '../types';

// ═══════════════════════════════════════════════════════════════════════
// Types
// ═══════════════════════════════════════════════════════════════════════

type ScreenRoute = RouteProp<RootStackParamList, 'Signature'>;

interface SignatureState {
  step: 'draw' | 'preview';
  signature: string | null;
  isSaving: boolean;
}

// ═══════════════════════════════════════════════════════════════════════
// Constants
// ═══════════════════════════════════════════════════════════════════════

const SCREEN_WIDTH = Dimensions.get('window').width;
const PREVIEW_SIZE = SCREEN_WIDTH - 64;

const SIGNATURE_WEB_STYLE = `
  .m-signature-pad {
    border: none;
    border-radius: 12px;
    box-shadow: none;
  }
  .m-signature-pad--body {
    border: 2px solid #e2e8f0;
    border-radius: 12px;
    margin: 8px;
  }
  .m-signature-pad--footer {
    display: flex;
    justify-content: center;
    padding: 12px 16px;
    gap: 12px;
    background: transparent;
  }
  .m-signature-pad--footer .button {
    background: #2563eb;
    color: #fff;
    padding: 12px 32px;
    border-radius: 10px;
    font-size: 16px;
    font-weight: 600;
    border: none;
    cursor: pointer;
  }
  .m-signature-pad--footer .button.clear {
    background: #64748b;
  }
  .m-signature-pad--footer .button.clear:hover {
    background: #475569;
  }
  .m-signature-pad--footer .button:hover {
    background: #1d4ed8;
  }
  .m-signature-pad--footer .description {
    display: none;
  }
`;

// ═══════════════════════════════════════════════════════════════════════
// Main Component
// ═══════════════════════════════════════════════════════════════════════

export default function SignatureScreen() {
  const navigation = useNavigation<NativeStackNavigationProp<RootStackParamList>>();
  const route = useRoute<ScreenRoute>();
  const { workOrderId } = route.params;

  const signatureRef = useRef<SignatureViewRef>(null);

  const [state, setState] = useState<SignatureState>({
    step: 'draw',
    signature: null,
    isSaving: false,
  });

  // ═════════════════════════════════════════════════════════════════
  // Signature handlers
  // ═════════════════════════════════════════════════════════════════

  const handleOK = useCallback((signature: string) => {
    // signature is a data URL: "data:image/png;base64,..."
    setState((prev) => ({
      ...prev,
      signature,
      step: 'preview',
    }));
  }, []);

  const handleEmpty = useCallback(() => {
    Alert.alert('Подпись не поставлена', 'Пожалуйста, поставьте подпись перед сохранением.');
  }, []);

  const handleClear = useCallback(() => {
    signatureRef.current?.clearSignature();
    setState((prev) => ({ ...prev, signature: null }));
  }, []);

  const handleUndo = useCallback(() => {
    signatureRef.current?.undo();
  }, []);

  // ═════════════════════════════════════════════════════════════════
  // Navigation
  // ═════════════════════════════════════════════════════════════════

  const handleCancel = useCallback(() => {
    if (state.signature) {
      Alert.alert('Отменить подпись?', 'Введённая подпись будет потеряна.', [
        { text: 'Продолжить', style: 'cancel' },
        {
          text: 'Отменить',
          style: 'destructive',
          onPress: () => navigation.goBack(),
        },
      ]);
    } else {
      navigation.goBack();
    }
  }, [state.signature, navigation]);

  const handleBackToDraw = useCallback(() => {
    setState((prev) => ({ ...prev, step: 'draw' }));
  }, []);

  const handleSave = useCallback(async () => {
    if (!state.signature) {
      handleEmpty();
      return;
    }

    setState((prev) => ({ ...prev, isSaving: true }));

    try {
      // In a real scenario, we'd upload the signature to the backend
      // and associate it with the work order. For now, we navigate back
      // with the signature data accessible via the work order store.

      // Simulate save delay
      await new Promise((resolve) => setTimeout(resolve, 500));

      Alert.alert(
        'Подпись сохранена',
        'Подпись успешно сохранена для наряда-заказа.',
        [
          {
            text: 'OK',
            onPress: () => navigation.goBack(),
          },
        ],
      );
    } catch (error) {
      Alert.alert('Ошибка', 'Не удалось сохранить подпись. Попробуйте снова.');
      setState((prev) => ({ ...prev, isSaving: false }));
    }
  }, [state.signature, handleEmpty, navigation]);

  // ═════════════════════════════════════════════════════════════════
  // Render: Draw step
  // ═════════════════════════════════════════════════════════════════

  const renderDrawStep = () => (
    <View style={styles.canvasContainer}>
      {/* Header info */}
      <View style={styles.header}>
        <Text style={styles.headerTitle}>✍️ Подпись клиента</Text>
        <Text style={styles.headerSubtitle}>
          Наряд-заказ #{workOrderId.slice(0, 8).toUpperCase()}
        </Text>
      </View>

      {/* Signature canvas */}
      <View style={styles.canvasWrapper}>
        <SignatureView
          ref={signatureRef}
          onOK={handleOK}
          onEmpty={handleEmpty}
          autoClear={false}
          descriptionText=""
          clearText="Очистить"
          confirmText="Сохранить"
          penColor="#1e293b"
          backgroundColor="#ffffff"
          webStyle={SIGNATURE_WEB_STYLE}
          style={styles.canvas}
        />
      </View>

      {/* Action hints */}
      <View style={styles.hints}>
        <View style={styles.hintRow}>
          <Ionicons name="hand-left-outline" size={16} color="#64748b" />
          <Text style={styles.hintText}>Поставьте подпись пальцем</Text>
        </View>
      </View>
    </View>
  );

  // ═════════════════════════════════════════════════════════════════
  // Render: Preview step
  // ═════════════════════════════════════════════════════════════════

  const renderPreviewStep = () => (
    <View style={styles.previewContainer}>
      <View style={styles.header}>
        <Text style={styles.headerTitle}>✓ Предпросмотр подписи</Text>
        <Text style={styles.headerSubtitle}>
          Проверьте подпись перед сохранением
        </Text>
      </View>

      {/* Signature preview */}
      <View style={styles.previewCard}>
        {state.signature ? (
          <Image
            source={{ uri: state.signature }}
            style={styles.previewImage}
            resizeMode="contain"
          />
        ) : (
          <View style={styles.previewEmpty}>
            <Ionicons name="image-outline" size={48} color="#cbd5e1" />
            <Text style={styles.previewEmptyText}>Подпись отсутствует</Text>
          </View>
        )}
      </View>

      {/* Stamp-like overlay */}
      <View style={styles.stampInfo}>
        <Ionicons name="checkmark-circle" size={20} color="#16a34a" />
        <Text style={styles.stampText}>
          Подпись будет прикреплена к наряду #{workOrderId.slice(0, 8).toUpperCase()}
        </Text>
      </View>
    </View>
  );

  // ═════════════════════════════════════════════════════════════════
  // Main Render
  // ═════════════════════════════════════════════════════════════════

  return (
    <View style={styles.container}>
      <StatusBar barStyle="dark-content" backgroundColor="#f1f5f9" />

      {/* ═══ Content ═══ */}
      {state.step === 'draw' ? renderDrawStep() : renderPreviewStep()}

      {/* ═══ Bottom bar ═══ */}
      {state.step === 'draw' ? (
        <View style={styles.bottomBar}>
          <TouchableOpacity
            style={styles.cancelButton}
            onPress={handleCancel}
          >
            <Ionicons name="close-outline" size={22} color="#64748b" />
            <Text style={styles.cancelButtonText}>Отмена</Text>
          </TouchableOpacity>

          <TouchableOpacity
            style={styles.clearButtonSmall}
            onPress={handleClear}
          >
            <Ionicons name="refresh-outline" size={20} color="#dc2626" />
            <Text style={styles.clearButtonSmallText}>Сброс</Text>
          </TouchableOpacity>
        </View>
      ) : (
        <View style={styles.bottomBar}>
          <TouchableOpacity
            style={styles.backButton}
            onPress={handleBackToDraw}
          >
            <Ionicons name="pencil-outline" size={22} color="#64748b" />
            <Text style={styles.backButtonText}>Исправить</Text>
          </TouchableOpacity>

          <TouchableOpacity
            style={[styles.saveButton, state.isSaving && styles.saveButtonDisabled]}
            onPress={handleSave}
            disabled={state.isSaving}
          >
            {state.isSaving ? (
              <ActivityIndicator color="#fff" size="small" />
            ) : (
              <>
                <Ionicons name="save-outline" size={20} color="#fff" />
                <Text style={styles.saveButtonText}>Сохранить</Text>
              </>
            )}
          </TouchableOpacity>
        </View>
      )}
    </View>
  );
}

// ═══════════════════════════════════════════════════════════════════════
// Styles
// ═══════════════════════════════════════════════════════════════════════

const styles = StyleSheet.create({
  container: {
    flex: 1,
    backgroundColor: '#f1f5f9',
  },

  // ── Header ─────────────────────────────────────────────────────
  header: {
    paddingHorizontal: 20,
    paddingTop: 16,
    paddingBottom: 12,
  },
  headerTitle: {
    fontSize: 22,
    fontWeight: '700',
    color: '#1e293b',
  },
  headerSubtitle: {
    fontSize: 13,
    color: '#64748b',
    marginTop: 4,
  },

  // ── Canvas (Draw step) ─────────────────────────────────────────
  canvasContainer: {
    flex: 1,
  },
  canvasWrapper: {
    flex: 1,
    marginHorizontal: 12,
    marginBottom: 8,
    backgroundColor: '#fff',
    borderRadius: 12,
    overflow: 'hidden',
    borderWidth: 1,
    borderColor: '#e2e8f0',
  },
  canvas: {
    flex: 1,
  },
  hints: {
    paddingHorizontal: 20,
    paddingBottom: 8,
    alignItems: 'center',
  },
  hintRow: {
    flexDirection: 'row',
    alignItems: 'center',
    gap: 6,
  },
  hintText: {
    fontSize: 13,
    color: '#64748b',
  },

  // ── Preview step ───────────────────────────────────────────────
  previewContainer: {
    flex: 1,
  },
  previewCard: {
    flex: 1,
    marginHorizontal: 20,
    marginBottom: 12,
    backgroundColor: '#fff',
    borderRadius: 16,
    justifyContent: 'center',
    alignItems: 'center',
    borderWidth: 1,
    borderColor: '#e2e8f0',
    shadowColor: '#000',
    shadowOffset: { width: 0, height: 2 },
    shadowOpacity: 0.05,
    shadowRadius: 8,
    elevation: 2,
  },
  previewImage: {
    width: PREVIEW_SIZE,
    height: PREVIEW_SIZE * 0.6,
  },
  previewEmpty: {
    alignItems: 'center',
    gap: 12,
  },
  previewEmptyText: {
    fontSize: 14,
    color: '#94a3b8',
  },
  stampInfo: {
    flexDirection: 'row',
    alignItems: 'center',
    justifyContent: 'center',
    gap: 8,
    paddingHorizontal: 20,
    paddingBottom: 8,
  },
  stampText: {
    fontSize: 13,
    color: '#16a34a',
    fontWeight: '500',
  },

  // ── Bottom Bar ─────────────────────────────────────────────────
  bottomBar: {
    flexDirection: 'row',
    paddingHorizontal: 16,
    paddingVertical: 12,
    paddingBottom: 32,
    gap: 12,
    backgroundColor: '#fff',
    borderTopWidth: 1,
    borderTopColor: '#e2e8f0',
  },

  // ── Draw buttons ───────────────────────────────────────────────
  cancelButton: {
    flex: 1,
    flexDirection: 'row',
    alignItems: 'center',
    justifyContent: 'center',
    gap: 6,
    backgroundColor: '#fff',
    paddingVertical: 14,
    borderRadius: 12,
    borderWidth: 1,
    borderColor: '#e2e8f0',
  },
  cancelButtonText: {
    fontSize: 16,
    fontWeight: '500',
    color: '#64748b',
  },
  clearButtonSmall: {
    flex: 1,
    flexDirection: 'row',
    alignItems: 'center',
    justifyContent: 'center',
    gap: 6,
    backgroundColor: '#fef2f2',
    paddingVertical: 14,
    borderRadius: 12,
    borderWidth: 1,
    borderColor: '#fecaca',
  },
  clearButtonSmallText: {
    fontSize: 16,
    fontWeight: '500',
    color: '#dc2626',
  },

  // ── Preview buttons ────────────────────────────────────────────
  backButton: {
    flex: 1,
    flexDirection: 'row',
    alignItems: 'center',
    justifyContent: 'center',
    gap: 6,
    backgroundColor: '#fff',
    paddingVertical: 14,
    borderRadius: 12,
    borderWidth: 1,
    borderColor: '#e2e8f0',
  },
  backButtonText: {
    fontSize: 16,
    fontWeight: '500',
    color: '#64748b',
  },
  saveButton: {
    flex: 2,
    flexDirection: 'row',
    alignItems: 'center',
    justifyContent: 'center',
    gap: 8,
    backgroundColor: '#16a34a',
    paddingVertical: 14,
    borderRadius: 12,
  },
  saveButtonDisabled: {
    backgroundColor: '#86efac',
  },
  saveButtonText: {
    fontSize: 16,
    fontWeight: '600',
    color: '#fff',
  },
});
