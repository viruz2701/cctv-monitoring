import React, { useCallback, useMemo } from 'react';
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
import { useQuery } from '@tanstack/react-query';
import { workOrdersApi } from '../api/workOrders';
import { useStartWorkOrder } from '../hooks/useWorkOrders';
import { RootStackParamList } from '../types';
import StatusBadge from '../components/StatusBadge';
import { formatWorkOrderDate, formatSLADeadline, isSLAPast } from '../utils/dateHelpers';

type Props = NativeStackScreenProps<RootStackParamList, 'WorkOrderDetail'>;

export default function WorkOrderDetailScreen({ route }: Props) {
  const { workOrderId } = route.params;
  const navigation = useNavigation<NativeStackNavigationProp<RootStackParamList>>();

  const { data: workOrder, isLoading, error } = useQuery({
    queryKey: ['workOrder', workOrderId],
    queryFn: () => workOrdersApi.getWorkOrder(workOrderId),
  });

  const startMutation = useStartWorkOrder();

  const handleStart = useCallback(() => {
    Alert.alert('Начать работу', 'Вы уверены, что хотите начать выполнение?', [
      { text: 'Отмена', style: 'cancel' },
      {
        text: 'Начать',
        onPress: () => startMutation.mutate(workOrderId),
      },
    ]);
  }, [startMutation, workOrderId]);

  const handleComplete = useCallback(() => {
    if (!workOrder) return;
    navigation.navigate('CompleteWorkOrder', { workOrder });
  }, [navigation, workOrder]);

  const handleScanQR = useCallback(() => {
    navigation.navigate('QRScanner');
  }, [navigation]);

  const isActive = useMemo(
    () => workOrder?.status === 'open' || workOrder?.status === 'in_progress',
    [workOrder?.status],
  );

  const slaOverdue = useMemo(
    () => (workOrder?.sla_deadline ? isSLAPast(workOrder.sla_deadline) : false),
    [workOrder?.sla_deadline],
  );

  const workOrderType = useMemo(() => {
    if (!workOrder) return '';
    switch (workOrder.type) {
      case 'preventive': return 'Плановое ТО';
      case 'corrective': return 'Ремонт';
      case 'emergency': return 'Аварийный';
      default: return workOrder.type;
    }
  }, [workOrder?.type]);

  if (isLoading) {
    return (
      <View style={styles.centered}>
        <ActivityIndicator size="large" color="#2563eb" />
      </View>
    );
  }

  if (error || !workOrder) {
    return (
      <View style={styles.centered}>
        <Text style={styles.errorText}>Не удалось загрузить наряд</Text>
      </View>
    );
  }

  return (
    <ScrollView style={styles.container} contentContainerStyle={styles.content}>
      <View style={styles.section}>
        <View style={styles.headerRow}>
          <StatusBadge status={workOrder.status} />
          <Text style={styles.type}>{workOrderType}</Text>
        </View>

        <Text style={styles.deviceName}>{workOrder.device_name || workOrder.device_id}</Text>
        {workOrder.site_name && <Text style={styles.siteName}>{workOrder.site_name}</Text>}
      </View>

      <View style={styles.section}>
        <Text style={styles.sectionTitle}>Информация</Text>
        <View style={styles.infoRow}>
          <Text style={styles.infoLabel}>Приоритет</Text>
          <Text style={styles.infoValue}>{workOrder.priority.toUpperCase()}</Text>
        </View>
        <View style={styles.infoRow}>
          <Text style={styles.infoLabel}>Создан</Text>
          <Text style={styles.infoValue}>{formatWorkOrderDate(workOrder.created_at)}</Text>
        </View>
        {workOrder.sla_deadline && (
          <View style={styles.infoRow}>
            <Text style={styles.infoLabel}>SLA Deadline</Text>
            <Text style={[styles.infoValue, slaOverdue && styles.slaOverdue]}>
              {formatSLADeadline(workOrder.sla_deadline)}
            </Text>
          </View>
        )}
        {workOrder.started_at && (
          <View style={styles.infoRow}>
            <Text style={styles.infoLabel}>Начат</Text>
            <Text style={styles.infoValue}>{formatWorkOrderDate(workOrder.started_at)}</Text>
          </View>
        )}
        {workOrder.assignee_name && (
          <View style={styles.infoRow}>
            <Text style={styles.infoLabel}>Исполнитель</Text>
            <Text style={styles.infoValue}>{workOrder.assignee_name}</Text>
          </View>
        )}
      </View>

      {workOrder.notes && (
        <View style={styles.section}>
          <Text style={styles.sectionTitle}>Заметки</Text>
          <Text style={styles.notes}>{workOrder.notes}</Text>
        </View>
      )}

      {isActive && (
        <View style={styles.actions}>
          {workOrder.status === 'open' && (
            <TouchableOpacity
              style={[styles.button, styles.primaryButton]}
              onPress={handleStart}
              disabled={startMutation.isPending}
            >
              {startMutation.isPending ? (
                <ActivityIndicator color="#fff" />
              ) : (
                <Text style={styles.buttonText}>Начать работу</Text>
              )}
            </TouchableOpacity>
          )}

          {workOrder.status === 'in_progress' && (
            <TouchableOpacity
              style={[styles.button, styles.completeButton]}
              onPress={handleComplete}
            >
              <Text style={styles.buttonText}>✅ Завершить наряд</Text>
            </TouchableOpacity>
          )}

          <TouchableOpacity
            style={[styles.button, styles.outlineButton]}
            onPress={handleScanQR}
          >
            <Text style={styles.outlineButtonText}>Сканировать QR устройства</Text>
          </TouchableOpacity>
        </View>
      )}

      {workOrder.status === 'completed' && (
        <View style={styles.completedBanner}>
          <Text style={styles.completedBannerText}>
            Завершён {workOrder.completed_at ? formatWorkOrderDate(workOrder.completed_at) : ''}
          </Text>
        </View>
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
  centered: {
    flex: 1,
    justifyContent: 'center',
    alignItems: 'center',
  },
  errorText: {
    fontSize: 16,
    color: '#dc2626',
  },
  section: {
    backgroundColor: '#fff',
    borderRadius: 12,
    padding: 16,
    marginBottom: 12,
  },
  headerRow: {
    flexDirection: 'row',
    justifyContent: 'space-between',
    alignItems: 'center',
    marginBottom: 12,
  },
  type: {
    fontSize: 12,
    color: '#64748b',
  },
  deviceName: {
    fontSize: 20,
    fontWeight: '700',
    color: '#1e293b',
    marginBottom: 4,
  },
  siteName: {
    fontSize: 14,
    color: '#64748b',
  },
  sectionTitle: {
    fontSize: 14,
    fontWeight: '600',
    color: '#64748b',
    textTransform: 'uppercase',
    marginBottom: 12,
  },
  infoRow: {
    flexDirection: 'row',
    justifyContent: 'space-between',
    paddingVertical: 8,
    borderBottomWidth: 1,
    borderBottomColor: '#f1f5f9',
  },
  infoLabel: {
    fontSize: 14,
    color: '#64748b',
  },
  infoValue: {
    fontSize: 14,
    fontWeight: '500',
    color: '#1e293b',
  },
  slaOverdue: {
    color: '#dc2626',
  },
  notes: {
    fontSize: 14,
    color: '#1e293b',
    lineHeight: 20,
  },
  actions: {
    gap: 10,
    marginTop: 8,
  },
  button: {
    borderRadius: 12,
    padding: 16,
    alignItems: 'center',
  },
  primaryButton: {
    backgroundColor: '#2563eb',
  },
  completeButton: {
    backgroundColor: '#16a34a',
  },
  outlineButton: {
    backgroundColor: '#fff',
    borderWidth: 1,
    borderColor: '#2563eb',
  },
  buttonText: {
    color: '#fff',
    fontSize: 16,
    fontWeight: '600',
  },
  outlineButtonText: {
    color: '#2563eb',
    fontSize: 16,
    fontWeight: '600',
  },
  completedBanner: {
    backgroundColor: '#d1fae5',
    borderRadius: 12,
    padding: 16,
    alignItems: 'center',
    marginTop: 8,
  },
  completedBannerText: {
    color: '#059669',
    fontSize: 14,
    fontWeight: '600',
  },
});
