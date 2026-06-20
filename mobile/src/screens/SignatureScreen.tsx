import React, { useRef, useState } from 'react';
import {
  View,
  Text,
  TouchableOpacity,
  TextInput,
  Alert,
  StyleSheet,
  ActivityIndicator,
  ScrollView,
} from 'react-native';
import { useNavigation } from '@react-navigation/native';
import { NativeStackScreenProps, NativeStackNavigationProp } from '@react-navigation/native-stack';
import SignatureCanvas from 'react-native-signature-canvas';
import { RootStackParamList, PartUsage } from '../types';
import { useCompleteWorkOrder } from '../hooks/useWorkOrders';
import { useLocation } from '../hooks/useLocation';
import { useSyncStore } from '../store/syncStore';

type Props = NativeStackScreenProps<RootStackParamList, 'Signature'>;

export default function SignatureScreen({ route }: Props) {
  const { workOrder, checklist, photos, verificationToken } = route.params;
  const navigation = useNavigation<NativeStackNavigationProp<RootStackParamList>>();
  const completeMutation = useCompleteWorkOrder();
  const { latitude, longitude } = useLocation();
  const addToQueue = useSyncStore((s) => s.addToQueue);

  const [signature, setSignature] = useState<string | null>(null);
  const [notes, setNotes] = useState('');
  const [isSubmitting, setIsSubmitting] = useState(false);
  const signatureRef = useRef<any>(null);

  const handleSignature = (sig: string) => {
    setSignature(sig);
  };

  const handleClear = () => {
    signatureRef.current?.clearSignature();
    setSignature(null);
  };

  const handleSubmit = async () => {
    if (!signature) {
      Alert.alert('Предупреждение', 'Получите подпись клиента');
      return;
    }

    setIsSubmitting(true);

    const payload = {
      notes,
      checklist,
      photos,
      parts_used: [] as PartUsage[],
      signature,
      location: { latitude, longitude },
      verification_token: verificationToken,
    };

    try {
      await completeMutation.mutateAsync({ id: workOrder.id, payload });
      Alert.alert('Успех', 'Наряд завершён!', [
        { text: 'OK', onPress: () => navigation.navigate('Main') },
      ]);
    } catch (error) {
      Alert.alert(
        'Офлайн-режим',
        'Наряд сохранён локально. Данные отправятся при подключении к сети.',
        [
          {
            text: 'OK',
            onPress: () => {
              addToQueue({
                type: 'complete_work_order',
                workOrderId: workOrder.id,
                payload,
              });
              navigation.navigate('Main');
            },
          },
        ],
      );
    } finally {
      setIsSubmitting(false);
    }
  };

  const handleSkip = () => {
    setSignature('skipped');
  };

  return (
    <ScrollView style={styles.container} contentContainerStyle={styles.content}>
      <View style={styles.section}>
        <Text style={styles.sectionTitle}>Подпись клиента</Text>
        <View style={styles.signatureContainer}>
          <SignatureCanvas
            ref={signatureRef}
            onOK={handleSignature}
            autoClear={false}
            descriptionText="Подпись клиента"
            clearText="Очистить"
            confirmText="Сохранить"
            webStyle={`.m-signature-pad { border: 1px solid #e2e8f0; border-radius: 8px; }
              .m-signature-pad--body { border: none; }
              .m-signature-pad--footer { display: flex; justify-content: space-between; }
              .m-signature-pad--footer .button { background: #2563eb; color: #fff; padding: 8px 16px; border-radius: 6px; }
              .m-signature-pad--footer .button.clear { background: #64748b; }`}
          />
        </View>

        {signature && (
          <View style={styles.signatureStatus}>
            <Text style={styles.signatureStatusText}>✓ Подпись получена</Text>
            <TouchableOpacity onPress={handleClear}>
              <Text style={styles.clearLink}>Очистить</Text>
            </TouchableOpacity>
          </View>
        )}
      </View>

      <View style={styles.section}>
        <Text style={styles.sectionTitle}>Комментарий</Text>
        <TextInput
          style={styles.textInput}
          placeholder="Опишите выполненные работы..."
          placeholderTextColor="#94a3b8"
          value={notes}
          onChangeText={setNotes}
          multiline
          numberOfLines={4}
          textAlignVertical="top"
        />
      </View>

      <View style={styles.summary}>
        <Text style={styles.summaryTitle}>Сводка</Text>
        <View style={styles.summaryRow}>
          <Text style={styles.summaryLabel}>Чек-лист</Text>
          <Text style={styles.summaryValue}>
            {checklist.filter((c) => c.completed).length}/{checklist.length}
          </Text>
        </View>
        <View style={styles.summaryRow}>
          <Text style={styles.summaryLabel}>Фото</Text>
          <Text style={styles.summaryValue}>{photos.length}</Text>
        </View>
        <View style={styles.summaryRow}>
          <Text style={styles.summaryLabel}>Подпись</Text>
          <Text style={styles.summaryValue}>{signature ? '✓' : '✗'}</Text>
        </View>
      </View>

      <TouchableOpacity
        style={[styles.submitButton, isSubmitting && styles.buttonDisabled]}
        onPress={handleSubmit}
        disabled={isSubmitting}
        activeOpacity={0.7}
      >
        {isSubmitting ? (
          <ActivityIndicator color="#fff" />
        ) : (
          <Text style={styles.submitButtonText}>Завершить наряд</Text>
        )}
      </TouchableOpacity>

      <TouchableOpacity style={styles.skipButton} onPress={handleSkip}>
        <Text style={styles.skipButtonText}>Пропустить подпись</Text>
      </TouchableOpacity>
    </ScrollView>
  );
}

const styles = StyleSheet.create({
  container: {
    flex: 1,
    backgroundColor: '#f1f5f9',
  },
  content: {
    padding: 16,
  },
  section: {
    backgroundColor: '#fff',
    borderRadius: 12,
    padding: 16,
    marginBottom: 12,
  },
  sectionTitle: {
    fontSize: 14,
    fontWeight: '600',
    color: '#64748b',
    textTransform: 'uppercase',
    marginBottom: 12,
  },
  signatureContainer: {
    height: 250,
    borderRadius: 8,
    overflow: 'hidden',
  },
  signatureStatus: {
    flexDirection: 'row',
    justifyContent: 'space-between',
    alignItems: 'center',
    marginTop: 8,
  },
  signatureStatusText: {
    fontSize: 14,
    color: '#16a34a',
    fontWeight: '500',
  },
  clearLink: {
    fontSize: 14,
    color: '#dc2626',
    fontWeight: '500',
  },
  textInput: {
    borderWidth: 1,
    borderColor: '#e2e8f0',
    borderRadius: 8,
    padding: 12,
    fontSize: 14,
    minHeight: 100,
    color: '#1e293b',
  },
  summary: {
    backgroundColor: '#fff',
    borderRadius: 12,
    padding: 16,
    marginBottom: 12,
  },
  summaryTitle: {
    fontSize: 14,
    fontWeight: '600',
    color: '#64748b',
    textTransform: 'uppercase',
    marginBottom: 12,
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
  submitButton: {
    backgroundColor: '#16a34a',
    marginBottom: 12,
    padding: 16,
    borderRadius: 12,
    alignItems: 'center',
  },
  buttonDisabled: {
    opacity: 0.6,
  },
  submitButtonText: {
    color: '#fff',
    fontSize: 16,
    fontWeight: '600',
  },
  skipButton: {
    padding: 12,
    alignItems: 'center',
  },
  skipButtonText: {
    fontSize: 14,
    color: '#64748b',
  },
});