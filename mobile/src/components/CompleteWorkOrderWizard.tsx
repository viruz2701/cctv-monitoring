import React, { useState, useRef } from 'react';
import {
  View,
  Text,
  ScrollView,
  TouchableOpacity,
  FlatList,
  Image,
  TextInput,
  Alert,
  StyleSheet,
  ActivityIndicator,
} from 'react-native';
import { useNavigation, CommonActions } from '@react-navigation/native';
import { NativeStackScreenProps } from '@react-navigation/native-stack';
import * as ImagePicker from 'expo-image-picker';
import SignatureCanvas from 'react-native-signature-canvas';

import { useCompleteWorkOrder, useUploadPhoto } from '../hooks/useWorkOrders';
import { useLocation } from '../hooks/useLocation';
import { useVerifyWorkOrder } from '../hooks/useGatekeeper';
import { useSyncStore } from '../store/syncStore';
import { RootStackParamList, ChecklistItem, VerificationRequest } from '../types';

// ═══════════════════════════════════════════════════════════════════════
// Types
// ═══════════════════════════════════════════════════════════════════════

type WizardStep = 1 | 2 | 3 | 4;

interface WizardState {
  step: WizardStep;
  checklist: ChecklistItem[];
  photos: string[];
  gpsSkipped: string | null;
  verificationToken: string | null;
  signature: string | null;
  notes: string;
  isSubmitting: boolean;
}

type Props = NativeStackScreenProps<RootStackParamList, 'CompleteWorkOrder'>;

// ═══════════════════════════════════════════════════════════════════════
// Step Indicator
// ═══════════════════════════════════════════════════════════════════════

function StepIndicator({ current, total }: { current: number; total: number }) {
  return (
    <View style={styles.indicator}>
      {Array.from({ length: total }, (_, i) => {
        const stepNum = i + 1;
        const isActive = stepNum === current;
        const isCompleted = stepNum < current;
        return (
          <View key={i} style={styles.indicatorItem}>
            <View
              style={[
                styles.dot,
                isActive && styles.dotActive,
                isCompleted && styles.dotCompleted,
              ]}
            >
              <Text style={[styles.dotText, (isActive || isCompleted) && styles.dotTextActive]}>
                {isCompleted ? '✓' : stepNum}
              </Text>
            </View>
            {i < total - 1 && (
              <View
                style={[
                  styles.dotLine,
                  isCompleted && styles.dotLineCompleted,
                ]}
              />
            )}
          </View>
        );
      })}
    </View>
  );
}

// ═══════════════════════════════════════════════════════════════════════
// Main Component
// ═══════════════════════════════════════════════════════════════════════

export default function CompleteWorkOrderWizard({ route }: Props) {
  const { workOrder } = route.params;
  const navigation = useNavigation();
  const completeMutation = useCompleteWorkOrder();
  const uploadMutation = useUploadPhoto();
  const verifyMutation = useVerifyWorkOrder();
  const { latitude, longitude, loading: locationLoading } = useLocation();
  const addToQueue = useSyncStore((s) => s.addToQueue);
  const signatureRef = useRef<any>(null);

  const [state, setState] = useState<WizardState>({
    step: 1,
    checklist: workOrder.checklist || [],
    photos: [],
    gpsSkipped: null,
    verificationToken: null,
    signature: null,
    notes: '',
    isSubmitting: false,
  });

  // ── Step 1: Checklist ────────────────────────────────────────────

  const toggleChecklist = (index: number) => {
    const updated = [...state.checklist];
    updated[index] = { ...updated[index], completed: !updated[index].completed };
    setState((s) => ({ ...s, checklist: updated }));
  };

  const checklistProgress =
    state.checklist.length > 0
      ? (state.checklist.filter((c) => c.completed).length / state.checklist.length) * 100
      : 0;

  // ── Step 2: Photo + GPS ─────────────────────────────────────────

  const takePhoto = async () => {
    const permission = await ImagePicker.requestCameraPermissionsAsync();
    if (!permission.granted) {
      Alert.alert('Ошибка', 'Необходимо разрешение на использование камеры');
      return;
    }

    const result = await ImagePicker.launchCameraAsync({
      quality: 0.8,
      allowsEditing: false,
    });

    if (!result.canceled && result.assets.length > 0) {
      setState((s) => ({ ...s, photos: [...s.photos, result.assets[0].uri] }));
    }
  };

  const pickFromGallery = async () => {
    const permission = await ImagePicker.requestMediaLibraryPermissionsAsync();
    if (!permission.granted) {
      Alert.alert('Ошибка', 'Необходимо разрешение на доступ к галерее');
      return;
    }

    const result = await ImagePicker.launchImageLibraryAsync({
      quality: 0.8,
      allowsEditing: false,
    });

    if (!result.canceled && result.assets.length > 0) {
      setState((s) => ({ ...s, photos: [...s.photos, result.assets[0].uri] }));
    }
  };

  const removePhoto = (index: number) => {
    setState((s) => ({ ...s, photos: s.photos.filter((_, i) => i !== index) }));
  };

  const handleSkipGPS = () => {
    setState((s) => ({ ...s, gpsSkipped: 'GPS signal unavailable at this location' }));
  };

  // ═══════════════════════════════════════════════════════════════════
  // Step 3: Signature + Submit
  // ═══════════════════════════════════════════════════════════════════

  const handleSignature = (sig: string) => {
    setState((s) => ({ ...s, signature: sig }));
  };

  const handleClearSignature = () => {
    signatureRef.current?.clearSignature();
    setState((s) => ({ ...s, signature: null }));
  };

  const handleSkipSignature = () => {
    setState((s) => ({ ...s, signature: 'skipped' }));
  };

  const handleSubmit = async () => {
    setState((s) => ({ ...s, isSubmitting: true }));

    // Upload photos first (best effort)
    const uploadedUrls: string[] = [];
    for (const photoUri of state.photos) {
      try {
        const result = await uploadMutation.mutateAsync({
          workOrderId: workOrder.id,
          photoUri,
        });
        uploadedUrls.push(result.url);
      } catch {
        uploadedUrls.push(photoUri); // fallback to local URI
      }
    }

    const payload = {
      notes: state.notes,
      checklist: state.checklist,
      photos: uploadedUrls,
      parts_used: [] as any[],
      signature: state.signature || undefined,
      verification_token: state.verificationToken || undefined,
      location:
        latitude || longitude
          ? { latitude, longitude }
          : undefined,
    };

    try {
      await completeMutation.mutateAsync({ id: workOrder.id, payload });
      setState((s) => ({ ...s, step: 4, isSubmitting: false }));
    } catch {
      // Offline fallback
      addToQueue({
        type: 'complete_work_order' as any,
        workOrderId: workOrder.id,
        payload,
      });
      setState((s) => ({ ...s, step: 4, isSubmitting: false }));
    }
  };

  // ═══════════════════════════════════════════════════════════════════
  // Navigation
  // ═══════════════════════════════════════════════════════════════════

  const goToStep = (step: WizardStep) => {
    setState((s) => ({ ...s, step }));
  };

  const goBack = () => {
    if (state.step > 1) {
      goToStep((state.step - 1) as WizardStep);
    } else {
      navigation.goBack();
    }
  };

  const goToDashboard = () => {
    navigation.dispatch(
      CommonActions.reset({
        index: 0,
        routes: [{ name: 'Main' }],
      })
    );
  };

  // ═══════════════════════════════════════════════════════════════════
  // Render — Step 1: Checklist
  // ═══════════════════════════════════════════════════════════════════

  const renderChecklistStep = () => (
    <View style={styles.stepContainer}>
      <Text style={styles.stepTitle}>📋 Чек-лист</Text>
      <Text style={styles.stepDescription}>
        Отметьте выполненные пункты
      </Text>

      {/* Progress */}
      <View style={styles.progressContainer}>
        <View style={styles.progressHeader}>
          <Text style={styles.progressText}>
            Прогресс: {state.checklist.filter((c) => c.completed).length}/{state.checklist.length}
          </Text>
          <Text style={styles.progressPercent}>{Math.round(checklistProgress)}%</Text>
        </View>
        <View style={styles.progressBar}>
          <View style={[styles.progressFill, { width: `${checklistProgress}%` }]} />
        </View>
      </View>

      {/* Checklist items */}
      <ScrollView style={styles.checklistScroll}>
        {state.checklist.length === 0 ? (
          <View style={styles.emptyState}>
            <Text style={styles.emptyText}>Нет пунктов чек-листа</Text>
          </View>
        ) : (
          state.checklist.map((item, index) => (
            <TouchableOpacity
              key={index}
              style={[styles.checklistItem, item.completed && styles.checklistItemDone]}
              onPress={() => toggleChecklist(index)}
              activeOpacity={0.7}
            >
              <View
                style={[
                  styles.checkbox,
                  item.completed && styles.checkboxChecked,
                ]}
              >
                {item.completed && <Text style={styles.checkmark}>✓</Text>}
              </View>
              <Text
                style={[
                  styles.checklistText,
                  item.completed && styles.checklistTextDone,
                ]}
              >
                {item.task}
              </Text>
            </TouchableOpacity>
          ))
        )}
      </ScrollView>

      {/* Navigation */}
      <View style={styles.stepNav}>
        <TouchableOpacity style={styles.navButtonSecondary} onPress={goBack}>
          <Text style={styles.navButtonSecondaryText}>← Назад</Text>
        </TouchableOpacity>
        <TouchableOpacity
          style={[styles.navButton, checklistProgress < 100 && styles.navButtonDisabled]}
          onPress={() => goToStep(2)}
          disabled={checklistProgress < 100}
        >
          <Text style={styles.navButtonText}>
            Фото и GPS →
          </Text>
        </TouchableOpacity>
      </View>
    </View>
  );

  // ═══════════════════════════════════════════════════════════════════
  // Render — Step 2: Photo + GPS
  // ═══════════════════════════════════════════════════════════════════

  const renderPhotoGPSStep = () => (
    <View style={styles.stepContainer}>
      <Text style={styles.stepTitle}>📸 Фотофиксация</Text>
      <Text style={styles.stepDescription}>
        Сделайте фото объекта
      </Text>

      {/* GPS status */}
      <View style={styles.gpsBar}>
        <Text style={styles.gpsText}>
          {state.gpsSkipped
            ? '⚠ GPS пропущен'
            : locationLoading
              ? '📍 Определение координат...'
              : `📍 ${latitude.toFixed(5)}, ${longitude.toFixed(5)}`}
        </Text>
        {!state.gpsSkipped && !locationLoading && (
          <TouchableOpacity onPress={handleSkipGPS}>
            <Text style={styles.gpsSkipLink}>Пропустить</Text>
          </TouchableOpacity>
        )}
      </View>

      {/* Photo gallery */}
      <ScrollView horizontal style={styles.photoGallery} contentContainerStyle={styles.photoGalleryContent}>
        {state.photos.length === 0 ? (
          <View style={styles.photoPlaceholder}>
            <Text style={styles.photoPlaceholderText}>Нет фото</Text>
          </View>
        ) : (
          state.photos.map((uri, index) => (
            <View key={index} style={styles.photoWrapper}>
              <Image source={{ uri }} style={styles.photo} />
              <TouchableOpacity
                style={styles.photoRemove}
                onPress={() => removePhoto(index)}
              >
                <Text style={styles.photoRemoveText}>✕</Text>
              </TouchableOpacity>
            </View>
          ))
        )}
      </ScrollView>

      {/* Capture buttons */}
      <View style={styles.captureRow}>
        <TouchableOpacity style={styles.captureButton} onPress={takePhoto}>
          <Text style={styles.captureButtonText}>📷 Камера</Text>
        </TouchableOpacity>
        <TouchableOpacity style={styles.captureButton} onPress={pickFromGallery}>
          <Text style={styles.captureButtonText}>🖼 Галерея</Text>
        </TouchableOpacity>
      </View>

      {/* Navigation */}
      <View style={styles.stepNav}>
        <TouchableOpacity style={styles.navButtonSecondary} onPress={goBack}>
          <Text style={styles.navButtonSecondaryText}>← Назад</Text>
        </TouchableOpacity>
        <TouchableOpacity
          style={[styles.navButton, state.photos.length === 0 && styles.navButtonDisabled]}
          onPress={() => goToStep(3)}
          disabled={state.photos.length === 0}
        >
          <Text style={styles.navButtonText}>
            Подпись →
          </Text>
        </TouchableOpacity>
      </View>
    </View>
  );

  // ═══════════════════════════════════════════════════════════════════
  // Render — Step 3: Signature + Submit
  // ═══════════════════════════════════════════════════════════════════

  const renderSignatureStep = () => (
    <View style={styles.stepContainer}>
      <Text style={styles.stepTitle}>✍️ Подпись</Text>
      <Text style={styles.stepDescription}>
        Получите подпись клиента
      </Text>

      {/* Signature pad */}
      <View style={styles.signatureContainer}>
        {!state.signature ? (
          <SignatureCanvas
            ref={signatureRef}
            onOK={handleSignature}
            autoClear={false}
            descriptionText="Подпись клиента"
            clearText="Очистить"
            confirmText="Сохранить"
            webStyle={`.m-signature-pad { border: 1px solid #e2e8f0; border-radius: 8px; }
              .m-signature-pad--body { border: none; }
              .m-signature-pad--footer { display: flex; justify-content: space-between; padding: 8px; }
              .m-signature-pad--footer .button { background: #2563eb; color: #fff; padding: 8px 16px; border-radius: 6px; }
              .m-signature-pad--footer .button.clear { background: #64748b; }`}
          />
        ) : (
          <View style={styles.signatureReceived}>
            <Text style={styles.signatureReceivedText}>
              {state.signature === 'skipped' ? '⏭ Подпись пропущена' : '✓ Подпись получена'}
            </Text>
            <TouchableOpacity onPress={handleClearSignature}>
              <Text style={styles.signatureClearLink}>Очистить</Text>
            </TouchableOpacity>
          </View>
        )}
      </View>

      {/* Notes */}
      <View style={styles.notesContainer}>
        <Text style={styles.notesLabel}>Комментарий</Text>
        <TextInput
          style={styles.notesInput}
          placeholder="Опишите выполненные работы..."
          placeholderTextColor="#94a3b8"
          value={state.notes}
          onChangeText={(text) => setState((s) => ({ ...s, notes: text }))}
          multiline
          numberOfLines={3}
          textAlignVertical="top"
        />
      </View>

      {/* Summary */}
      <View style={styles.summary}>
        <Text style={styles.summaryTitle}>Сводка</Text>
        <View style={styles.summaryRow}>
          <Text style={styles.summaryLabel}>Чек-лист</Text>
          <Text style={styles.summaryValue}>
            {state.checklist.filter((c) => c.completed).length}/{state.checklist.length}
          </Text>
        </View>
        <View style={styles.summaryRow}>
          <Text style={styles.summaryLabel}>Фото</Text>
          <Text style={styles.summaryValue}>{state.photos.length}</Text>
        </View>
        <View style={styles.summaryRow}>
          <Text style={styles.summaryLabel}>Подпись</Text>
          <Text style={styles.summaryValue}>
            {state.signature ? (state.signature === 'skipped' ? '⏭' : '✓') : '✗'}
          </Text>
        </View>
      </View>

      {/* Navigation */}
      <View style={styles.stepNav}>
        <TouchableOpacity style={styles.navButtonSecondary} onPress={goBack}>
          <Text style={styles.navButtonSecondaryText}>← Назад</Text>
        </TouchableOpacity>
        <TouchableOpacity
          style={[styles.submitButton, state.isSubmitting && styles.navButtonDisabled]}
          onPress={handleSubmit}
          disabled={state.isSubmitting}
        >
          {state.isSubmitting ? (
            <ActivityIndicator color="#fff" />
          ) : (
            <Text style={styles.submitButtonText}>✅ Завершить наряд</Text>
          )}
        </TouchableOpacity>
      </View>

      {!state.signature && (
        <TouchableOpacity style={styles.skipButton} onPress={handleSkipSignature}>
          <Text style={styles.skipButtonText}>Пропустить подпись</Text>
        </TouchableOpacity>
      )}
    </View>
  );

  // ═══════════════════════════════════════════════════════════════════
  // Render — Step 4: Complete
  // ═══════════════════════════════════════════════════════════════════

  const renderCompleteStep = () => (
    <View style={styles.completeContainer}>
      <View style={styles.completeIcon}>
        <Text style={styles.completeIconText}>✅</Text>
      </View>
      <Text style={styles.completeTitle}>Наряд завершён!</Text>
      <Text style={styles.completeSubtitle}>
        #{workOrder.id} — {workOrder.device_name || workOrder.device_id}
      </Text>
      <TouchableOpacity style={styles.dashboardButton} onPress={goToDashboard}>
        <Text style={styles.dashboardButtonText}>← К списку заданий</Text>
      </TouchableOpacity>
    </View>
  );

  // ═══════════════════════════════════════════════════════════════════
  // Main Render
  // ═══════════════════════════════════════════════════════════════════

  return (
    <View style={styles.container}>
      {/* Header with step indicator */}
      {state.step < 4 && (
        <View style={styles.header}>
          <StepIndicator current={state.step} total={3} />
        </View>
      )}

      {/* Step content */}
      {state.step === 1 && renderChecklistStep()}
      {state.step === 2 && renderPhotoGPSStep()}
      {state.step === 3 && renderSignatureStep()}
      {state.step === 4 && renderCompleteStep()}
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
  header: {
    backgroundColor: '#fff',
    paddingVertical: 16,
    paddingHorizontal: 24,
    borderBottomWidth: 1,
    borderBottomColor: '#e2e8f0',
  },

  // ── Step Indicator ─────────────────────────────────────────────
  indicator: {
    flexDirection: 'row',
    alignItems: 'center',
    justifyContent: 'center',
  },
  indicatorItem: {
    flexDirection: 'row',
    alignItems: 'center',
  },
  dot: {
    width: 32,
    height: 32,
    borderRadius: 16,
    backgroundColor: '#e2e8f0',
    justifyContent: 'center',
    alignItems: 'center',
  },
  dotActive: {
    backgroundColor: '#2563eb',
  },
  dotCompleted: {
    backgroundColor: '#16a34a',
  },
  dotText: {
    fontSize: 14,
    fontWeight: '600',
    color: '#64748b',
  },
  dotTextActive: {
    color: '#fff',
  },
  dotLine: {
    width: 48,
    height: 2,
    backgroundColor: '#e2e8f0',
    marginHorizontal: 8,
  },
  dotLineCompleted: {
    backgroundColor: '#16a34a',
  },

  // ── Step Container ─────────────────────────────────────────────
  stepContainer: {
    flex: 1,
    padding: 16,
  },
  stepTitle: {
    fontSize: 22,
    fontWeight: '700',
    color: '#1e293b',
    marginBottom: 4,
  },
  stepDescription: {
    fontSize: 14,
    color: '#64748b',
    marginBottom: 16,
  },

  // ── Checklist ──────────────────────────────────────────────────
  progressContainer: {
    backgroundColor: '#fff',
    borderRadius: 12,
    padding: 12,
    marginBottom: 12,
  },
  progressHeader: {
    flexDirection: 'row',
    justifyContent: 'space-between',
    marginBottom: 8,
  },
  progressText: {
    fontSize: 14,
    fontWeight: '500',
    color: '#1e293b',
  },
  progressPercent: {
    fontSize: 14,
    fontWeight: '600',
    color: '#2563eb',
  },
  progressBar: {
    height: 8,
    backgroundColor: '#e2e8f0',
    borderRadius: 4,
    overflow: 'hidden',
  },
  progressFill: {
    height: '100%',
    backgroundColor: '#2563eb',
    borderRadius: 4,
  },
  checklistScroll: {
    flex: 1,
    marginBottom: 12,
  },
  checklistItem: {
    flexDirection: 'row',
    alignItems: 'center',
    backgroundColor: '#fff',
    padding: 14,
    borderRadius: 10,
    marginBottom: 8,
  },
  checklistItemDone: {
    backgroundColor: '#f0fdf4',
  },
  checkbox: {
    width: 24,
    height: 24,
    borderRadius: 6,
    borderWidth: 2,
    borderColor: '#cbd5e1',
    marginRight: 12,
    justifyContent: 'center',
    alignItems: 'center',
  },
  checkboxChecked: {
    backgroundColor: '#16a34a',
    borderColor: '#16a34a',
  },
  checkmark: {
    color: '#fff',
    fontSize: 16,
    fontWeight: 'bold',
  },
  checklistText: {
    flex: 1,
    fontSize: 15,
    color: '#1e293b',
  },
  checklistTextDone: {
    textDecorationLine: 'line-through',
    color: '#64748b',
  },
  emptyState: {
    padding: 40,
    alignItems: 'center',
  },
  emptyText: {
    fontSize: 14,
    color: '#94a3b8',
  },

  // ── Photo + GPS ────────────────────────────────────────────────
  gpsBar: {
    flexDirection: 'row',
    justifyContent: 'space-between',
    alignItems: 'center',
    backgroundColor: '#fff',
    borderRadius: 12,
    padding: 12,
    marginBottom: 12,
  },
  gpsText: {
    fontSize: 12,
    color: '#64748b',
    fontFamily: 'monospace',
  },
  gpsSkipLink: {
    fontSize: 12,
    color: '#2563eb',
    fontWeight: '500',
  },
  photoGallery: {
    maxHeight: 160,
    marginBottom: 12,
  },
  photoGalleryContent: {
    gap: 10,
  },
  photoPlaceholder: {
    width: 140,
    height: 140,
    borderRadius: 12,
    borderWidth: 2,
    borderColor: '#e2e8f0',
    borderStyle: 'dashed',
    justifyContent: 'center',
    alignItems: 'center',
  },
  photoPlaceholderText: {
    fontSize: 13,
    color: '#94a3b8',
  },
  photoWrapper: {
    position: 'relative',
  },
  photo: {
    width: 140,
    height: 140,
    borderRadius: 12,
    backgroundColor: '#e2e8f0',
  },
  photoRemove: {
    position: 'absolute',
    top: -6,
    right: -6,
    backgroundColor: '#dc2626',
    width: 22,
    height: 22,
    borderRadius: 11,
    justifyContent: 'center',
    alignItems: 'center',
  },
  photoRemoveText: {
    color: '#fff',
    fontSize: 11,
    fontWeight: 'bold',
  },
  captureRow: {
    flexDirection: 'row',
    gap: 12,
    marginBottom: 12,
  },
  captureButton: {
    flex: 1,
    backgroundColor: '#fff',
    borderRadius: 12,
    padding: 14,
    alignItems: 'center',
    borderWidth: 1,
    borderColor: '#e2e8f0',
  },
  captureButtonText: {
    fontSize: 15,
    fontWeight: '500',
    color: '#1e293b',
  },

  // ── Signature ──────────────────────────────────────────────────
  signatureContainer: {
    backgroundColor: '#fff',
    borderRadius: 12,
    padding: 12,
    marginBottom: 12,
    minHeight: 180,
    justifyContent: 'center',
  },
  signatureReceived: {
    alignItems: 'center',
    padding: 24,
  },
  signatureReceivedText: {
    fontSize: 18,
    fontWeight: '600',
    color: '#16a34a',
    marginBottom: 8,
  },
  signatureClearLink: {
    fontSize: 14,
    color: '#dc2626',
    fontWeight: '500',
  },

  // ── Notes ──────────────────────────────────────────────────────
  notesContainer: {
    backgroundColor: '#fff',
    borderRadius: 12,
    padding: 12,
    marginBottom: 12,
  },
  notesLabel: {
    fontSize: 12,
    fontWeight: '600',
    color: '#64748b',
    textTransform: 'uppercase',
    marginBottom: 8,
  },
  notesInput: {
    borderWidth: 1,
    borderColor: '#e2e8f0',
    borderRadius: 8,
    padding: 12,
    fontSize: 14,
    minHeight: 80,
    color: '#1e293b',
  },

  // ── Summary ────────────────────────────────────────────────────
  summary: {
    backgroundColor: '#fff',
    borderRadius: 12,
    padding: 12,
    marginBottom: 12,
  },
  summaryTitle: {
    fontSize: 12,
    fontWeight: '600',
    color: '#64748b',
    textTransform: 'uppercase',
    marginBottom: 8,
  },
  summaryRow: {
    flexDirection: 'row',
    justifyContent: 'space-between',
    paddingVertical: 6,
  },
  summaryLabel: {
    fontSize: 14,
    color: '#64748b',
  },
  summaryValue: {
    fontSize: 14,
    fontWeight: '500',
    color: '#1e293b',
  },

  // ── Navigation ─────────────────────────────────────────────────
  stepNav: {
    flexDirection: 'row',
    gap: 12,
    marginTop: 8,
  },
  navButton: {
    flex: 1,
    backgroundColor: '#2563eb',
    padding: 16,
    borderRadius: 12,
    alignItems: 'center',
  },
  navButtonDisabled: {
    backgroundColor: '#cbd5e1',
  },
  navButtonText: {
    color: '#fff',
    fontSize: 16,
    fontWeight: '600',
  },
  navButtonSecondary: {
    flex: 1,
    backgroundColor: '#fff',
    padding: 16,
    borderRadius: 12,
    alignItems: 'center',
    borderWidth: 1,
    borderColor: '#e2e8f0',
  },
  navButtonSecondaryText: {
    color: '#64748b',
    fontSize: 16,
    fontWeight: '500',
  },
  submitButton: {
    flex: 1,
    backgroundColor: '#16a34a',
    padding: 16,
    borderRadius: 12,
    alignItems: 'center',
  },
  submitButtonText: {
    color: '#fff',
    fontSize: 16,
    fontWeight: '600',
  },
  skipButton: {
    padding: 12,
    alignItems: 'center',
    marginTop: 4,
  },
  skipButtonText: {
    fontSize: 14,
    color: '#64748b',
  },

  // ── Complete Screen ────────────────────────────────────────────
  completeContainer: {
    flex: 1,
    justifyContent: 'center',
    alignItems: 'center',
    padding: 32,
    backgroundColor: '#f0fdf4',
  },
  completeIcon: {
    width: 80,
    height: 80,
    borderRadius: 40,
    backgroundColor: '#d1fae5',
    justifyContent: 'center',
    alignItems: 'center',
    marginBottom: 20,
  },
  completeIconText: {
    fontSize: 40,
  },
  completeTitle: {
    fontSize: 24,
    fontWeight: '700',
    color: '#16a34a',
    marginBottom: 8,
  },
  completeSubtitle: {
    fontSize: 14,
    color: '#64748b',
    marginBottom: 32,
    textAlign: 'center',
  },
  dashboardButton: {
    backgroundColor: '#2563eb',
    paddingHorizontal: 32,
    paddingVertical: 16,
    borderRadius: 12,
  },
  dashboardButtonText: {
    color: '#fff',
    fontSize: 16,
    fontWeight: '600',
  },
});
