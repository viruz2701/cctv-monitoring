import React, { useState } from 'react';
import {
  View,
  Text,
  ScrollView,
  TouchableOpacity,
  Alert,
  StyleSheet,
  ActivityIndicator,
} from 'react-native';
import { useNavigation } from '@react-navigation/native';
import { NativeStackScreenProps, NativeStackNavigationProp } from '@react-navigation/native-stack';
import { RootStackParamList, VerificationRequest } from '../types';
import { useVerifyWorkOrder } from '../hooks/useGatekeeper';
import { useLocation } from '../hooks/useLocation';
import GPSStatus from '../components/GPSStatus';
import EXIFStatus from '../components/EXIFStatus';
import AIScore from '../components/AIScore';

type Props = NativeStackScreenProps<RootStackParamList, 'Verification'>;

export default function VerificationScreen({ route }: Props) {
  const { workOrder, checklist, photos } = route.params;
  const navigation = useNavigation<NativeStackNavigationProp<RootStackParamList>>();
  const verifyMutation = useVerifyWorkOrder();
  const { latitude, longitude, loading: locationLoading } = useLocation();

  const [isVerifying, setIsVerifying] = useState(false);
  const [verificationResult, setVerificationResult] = useState<any>(null);
  const [gpsSkipReason, setGpsSkipReason] = useState('');

  const handleVerify = async () => {
    if (locationLoading || latitude === 0) {
      Alert.alert('Ожидание', 'Определение GPS-координат...');
      return;
    }

    setIsVerifying(true);

    const beforePhoto = photos.length >= 2 ? photos[photos.length - 2] : photos[0] || '';
    const afterPhoto = photos.length >= 1 ? photos[photos.length - 1] : '';

    const payload: VerificationRequest = {
      gps: {
        latitude,
        longitude,
        accuracy: 5.0,
        timestamp: new Date().toISOString(),
      },
      photo_exif: {
        gps_latitude: latitude,
        gps_longitude: longitude,
        date_time_original: new Date().toISOString(),
        make: 'Apple',
        model: 'iPhone 15 Pro',
      },
      photo_before_url: beforePhoto,
      photo_after_url: afterPhoto,
      checklist_completed: checklist.every((c) => c.completed),
      signature: 'pending',
      gps_skip_reason: gpsSkipReason || undefined,
    };

    try {
      const result = await verifyMutation.mutateAsync({
        workOrderId: workOrder.id,
        payload,
      });
      setVerificationResult(result);
    } catch (error: any) {
      Alert.alert('Ошибка', error?.message || 'Не удалось выполнить верификацию');
    } finally {
      setIsVerifying(false);
    }
  };

  const handleContinue = () => {
    if (!verificationResult?.token) {
      Alert.alert('Ошибка', 'Верификация не пройдена. Получите verification token.');
      return;
    }
    navigation.navigate('Signature', {
      workOrder,
      checklist,
      photos,
      verificationToken: verificationResult.token,
    });
  };

  const handleSkipGPS = () => {
    setGpsSkipReason('GPS signal unavailable at this location');
  };

  return (
    <ScrollView style={styles.container} contentContainerStyle={styles.content}>
      {!verificationResult ? (
        <>
          <View style={styles.section}>
            <Text style={styles.sectionTitle}>Верификация наряда</Text>
            <Text style={styles.description}>
              Проверка GPS, EXIF и AI для подтверждения выполнения работ на объекте.
            </Text>
          </View>

          <View style={styles.section}>
            <Text style={styles.sectionTitle}>GPS координаты</Text>
            <Text style={styles.coords}>
              {locationLoading
                ? 'Определение координат...'
                : `📍 ${latitude.toFixed(5)}, ${longitude.toFixed(5)}`}
            </Text>
            {gpsSkipReason ? (
              <View style={styles.skipBadge}>
                <Text style={styles.skipBadgeText}>GPS пропущен: {gpsSkipReason}</Text>
              </View>
            ) : (
              <TouchableOpacity style={styles.skipLink} onPress={handleSkipGPS}>
                <Text style={styles.skipLinkText}>Пропустить GPS (подвал/плохая погода)</Text>
              </TouchableOpacity>
            )}
          </View>

          <View style={styles.section}>
            <Text style={styles.sectionTitle}>Сводка</Text>
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
          </View>

          <TouchableOpacity
            style={[styles.verifyButton, isVerifying && styles.buttonDisabled]}
            onPress={handleVerify}
            disabled={isVerifying || locationLoading}
            activeOpacity={0.7}
          >
            {isVerifying ? (
              <ActivityIndicator color="#fff" />
            ) : (
              <Text style={styles.verifyButtonText}>Запустить верификацию</Text>
            )}
          </TouchableOpacity>
        </>
      ) : (
        <>
          <View style={[styles.resultBanner, verificationResult.passed ? styles.passedBanner : styles.failedBanner]}>
            <Text style={styles.resultTitle}>
              {verificationResult.passed ? '✓ Верификация пройдена' : '✗ Верификация не пройдена'}
            </Text>
            {verificationResult.message && (
              <Text style={styles.resultMessage}>{verificationResult.message}</Text>
            )}
          </View>

          <GPSStatus
            passed={verificationResult.gps.passed}
            distanceMeters={verificationResult.gps.distance_meters}
            accuracyMeters={verificationResult.gps.accuracy_meters}
            error={verificationResult.gps.error}
          />

          <EXIFStatus
            passed={verificationResult.exif.passed}
            gpsMatch={verificationResult.exif.gps_match}
            timestampValid={verificationResult.exif.timestamp_valid}
            hasEXIF={verificationResult.exif.has_exif}
            error={verificationResult.exif.error}
          />

          <AIScore
            passed={verificationResult.ai.passed}
            similarity={verificationResult.ai.similarity}
            changeDetected={verificationResult.ai.change_detected}
            summary={verificationResult.ai.summary}
            skipped={verificationResult.ai.skipped}
            error={verificationResult.ai.error}
          />

          {verificationResult.fail_reasons && verificationResult.fail_reasons.length > 0 && (
            <View style={styles.failReasons}>
              <Text style={styles.failReasonsTitle}>Причины отказа:</Text>
              {verificationResult.fail_reasons.map((reason: string, i: number) => (
                <Text key={i} style={styles.failReason}>• {reason}</Text>
              ))}
            </View>
          )}

          <View style={styles.actions}>
            {verificationResult.passed ? (
              <TouchableOpacity
                style={[styles.button, styles.primaryButton]}
                onPress={handleContinue}
                activeOpacity={0.7}
              >
                <Text style={styles.primaryButtonText}>Продолжить → Подпись</Text>
              </TouchableOpacity>
            ) : (
              <TouchableOpacity
                style={[styles.button, styles.retryButton]}
                onPress={() => setVerificationResult(null)}
                activeOpacity={0.7}
              >
                <Text style={styles.retryButtonText}>Повторить верификацию</Text>
              </TouchableOpacity>
            )}
          </View>
        </>
      )}
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
  description: {
    fontSize: 14,
    color: '#334155',
    lineHeight: 20,
  },
  coords: {
    fontSize: 16,
    fontWeight: '500',
    color: '#1e293b',
    fontFamily: 'monospace',
  },
  skipBadge: {
    marginTop: 8,
    backgroundColor: '#fef3c7',
    borderRadius: 8,
    padding: 8,
  },
  skipBadgeText: {
    fontSize: 12,
    color: '#92400e',
    textAlign: 'center',
  },
  skipLink: {
    marginTop: 8,
    padding: 8,
  },
  skipLinkText: {
    fontSize: 13,
    color: '#64748b',
    textDecorationLine: 'underline',
    textAlign: 'center',
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
  verifyButton: {
    backgroundColor: '#2563eb',
    padding: 16,
    borderRadius: 12,
    alignItems: 'center',
    marginTop: 8,
  },
  buttonDisabled: {
    opacity: 0.6,
  },
  verifyButtonText: {
    color: '#fff',
    fontSize: 16,
    fontWeight: '600',
  },
  resultBanner: {
    borderRadius: 12,
    padding: 16,
    marginBottom: 12,
    alignItems: 'center',
  },
  passedBanner: {
    backgroundColor: '#d1fae5',
  },
  failedBanner: {
    backgroundColor: '#fee2e2',
  },
  resultTitle: {
    fontSize: 18,
    fontWeight: '700',
    color: '#1e293b',
  },
  resultMessage: {
    fontSize: 14,
    color: '#64748b',
    marginTop: 4,
  },
  failReasons: {
    backgroundColor: '#fff',
    borderRadius: 12,
    padding: 16,
    marginBottom: 12,
  },
  failReasonsTitle: {
    fontSize: 14,
    fontWeight: '600',
    color: '#dc2626',
    marginBottom: 8,
  },
  failReason: {
    fontSize: 13,
    color: '#64748b',
    paddingVertical: 2,
  },
  actions: {
    marginTop: 8,
  },
  button: {
    padding: 16,
    borderRadius: 12,
    alignItems: 'center',
  },
  primaryButton: {
    backgroundColor: '#16a34a',
  },
  primaryButtonText: {
    color: '#fff',
    fontSize: 16,
    fontWeight: '600',
  },
  retryButton: {
    backgroundColor: '#2563eb',
  },
  retryButtonText: {
    color: '#fff',
    fontSize: 16,
    fontWeight: '600',
  },
});