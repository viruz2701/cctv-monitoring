import React, { useCallback } from 'react';
import { View, Text, FlatList, RefreshControl, StyleSheet, Alert } from 'react-native';
import { useQuery, useMutation, useQueryClient } from '@tanstack/react-query';
import { useNavigation } from '@react-navigation/native';
import { NativeStackNavigationProp } from '@react-navigation/native-stack';
import { workOrdersApi } from '../api/workOrders';
import { useSyncStore } from '../store/syncStore';
import { RootStackParamList, WorkOrder } from '../types';
import WorkOrderCard from '../components/WorkOrderCard';
import SwipeableCard, { SwipeAction } from '../components/SwipeableCard';
import OfflineIndicator from '../components/OfflineIndicator';
import { getGreeting } from '../utils/dateHelpers';
import { useAuthStore } from '../store/authStore';

export default function DashboardScreen() {
  const navigation = useNavigation<NativeStackNavigationProp<RootStackParamList>>();
  const user = useAuthStore((s) => s.user);
  const queryClient = useQueryClient();
  const addToQueue = useSyncStore((s) => s.addToQueue);
  const isOnline = useSyncStore((s) => s.isOnline);

  const {
    data: workOrders,
    isLoading,
    refetch,
    isRefetching,
  } = useQuery({
    queryKey: ['myWorkOrders'],
    queryFn: workOrdersApi.getMyWorkOrders,
  });

  // Mutations для inline-действий
  const startMutation = useMutation({
    mutationFn: (id: string) => workOrdersApi.startWorkOrder(id),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ['myWorkOrders'] });
    },
  });

  const completeMutation = useMutation({
    mutationFn: (id: string) =>
      workOrdersApi.completeWorkOrder(id, {
        notes: '',
        checklist: [],
        photos: [],
        parts_used: [],
      }),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ['myWorkOrders'] });
    },
  });

  const openWorkOrder = useCallback(
    (id: string) => {
      navigation.navigate('WorkOrderDetail', { workOrderId: id });
    },
    [navigation],
  );

  // ── Inline actions (UX-03) ──────────────────────────────────────────

  const handleStart = useCallback(
    (wo: WorkOrder) => {
      if (isOnline) {
        startMutation.mutate(wo.id);
      } else {
        // Offline: ставим в очередь синхронизации
        addToQueue({ type: 'start_work_order', workOrderId: wo.id });
        // Оптимистично обновляем кеш
        queryClient.setQueryData<WorkOrder[]>(['myWorkOrders'], (old) =>
          old?.map((o) => (o.id === wo.id ? { ...o, status: 'in_progress' as const } : o)),
        );
      }
    },
    [isOnline, startMutation, addToQueue, queryClient],
  );

  const handleComplete = useCallback(
    (wo: WorkOrder) => {
      Alert.alert(
        'Завершить наряд',
        `Завершить наряд #${wo.id}?\n\nДля полного завершения откройте наряд и заполните данные.`,
        [
          { text: 'Отмена', style: 'cancel' },
          {
            text: 'Быстрое завершение',
            onPress: () => {
              if (isOnline) {
                completeMutation.mutate(wo.id);
              } else {
                addToQueue({
                  type: 'complete_work_order',
                  workOrderId: wo.id,
                  payload: { notes: '', checklist: [], photos: [], parts_used: [] },
                });
                queryClient.setQueryData<WorkOrder[]>(['myWorkOrders'], (old) =>
                  old?.map((o) =>
                    o.id === wo.id ? { ...o, status: 'completed' as const } : o,
                  ),
                );
              }
            },
          },
        ],
      );
    },
    [isOnline, completeMutation, addToQueue, queryClient],
  );

  const handleCancel = useCallback(
    (wo: WorkOrder) => {
      Alert.alert('Отменить наряд', `Отменить наряд #${wo.id}?`, [
        { text: 'Нет', style: 'cancel' },
        {
          text: 'Да, отменить',
          style: 'destructive',
          onPress: () => {
            // TODO: добавить API endpoint для отмены inline
            addToQueue({
              type: 'start_work_order', // замена: будет отдельный тип cancel
              workOrderId: wo.id,
            });
            queryClient.setQueryData<WorkOrder[]>(['myWorkOrders'], (old) =>
              old?.map((o) =>
                o.id === wo.id ? { ...o, status: 'cancelled' as const } : o,
              ),
            );
          },
        },
      ]);
    },
    [addToQueue, queryClient],
  );

  // Получение действий для свайпа на основе статуса
  const getSwipeActions = useCallback(
    (wo: WorkOrder): { right?: SwipeAction[]; left?: SwipeAction[] } => {
      switch (wo.status) {
        case 'open':
          return {
            right: [
              {
                key: 'start',
                label: 'В работу',
                color: '#2563eb',
                icon: '▶️',
                onPress: () => handleStart(wo),
              },
            ],
          };
        case 'in_progress':
          return {
            right: [
              {
                key: 'complete',
                label: 'Завершить',
                color: '#059669',
                icon: '✅',
                onPress: () => handleComplete(wo),
              },
            ],
            left: [
              {
                key: 'cancel',
                label: 'Отмена',
                color: '#dc2626',
                icon: '⏹️',
                onPress: () => handleCancel(wo),
              },
            ],
          };
        default:
          // completed / cancelled — без действий
          return {};
      }
    },
    [handleStart, handleComplete, handleCancel],
  );

  const getStatusCounts = () => {
    if (!workOrders) return { open: 0, inProgress: 0, completed: 0 };
    return {
      open: workOrders.filter((wo) => wo.status === 'open').length,
      inProgress: workOrders.filter((wo) => wo.status === 'in_progress').length,
      completed: workOrders.filter((wo) => wo.status === 'completed').length,
    };
  };

  const counts = getStatusCounts();

  const renderItem = useCallback(
    ({ item }: { item: WorkOrder }) => {
      const actions = getSwipeActions(item);
      return (
        <SwipeableCard
          rightActions={actions.right}
          leftActions={actions.left}
          disabled={item.status === 'completed' || item.status === 'cancelled'}
        >
          <WorkOrderCard workOrder={item} onPress={() => openWorkOrder(item.id)} />
        </SwipeableCard>
      );
    },
    [getSwipeActions, openWorkOrder],
  );

  return (
    <View style={styles.container}>
      <OfflineIndicator />

      <View style={styles.statsRow}>
        <View style={[styles.statCard, { backgroundColor: '#fef2f2' }]}>
          <Text style={[styles.statNumber, { color: '#dc2626' }]}>{counts.open}</Text>
          <Text style={styles.statLabel}>Открытые</Text>
        </View>
        <View style={[styles.statCard, { backgroundColor: '#fef3c7' }]}>
          <Text style={[styles.statNumber, { color: '#d97706' }]}>
            {counts.inProgress}
          </Text>
          <Text style={styles.statLabel}>В работе</Text>
        </View>
        <View style={[styles.statCard, { backgroundColor: '#d1fae5' }]}>
          <Text style={[styles.statNumber, { color: '#059669' }]}>
            {counts.completed}
          </Text>
          <Text style={styles.statLabel}>Завершено</Text>
        </View>
      </View>

      <FlatList
        data={workOrders || []}
        keyExtractor={(item) => item.id}
        renderItem={renderItem}
        contentContainerStyle={styles.list}
        refreshControl={
          <RefreshControl refreshing={isRefetching} onRefresh={refetch} />
        }
        ListHeaderComponent={
          <Text style={styles.greeting}>
            {getGreeting()}, {user?.username || 'Техник'}
          </Text>
        }
        ListEmptyComponent={
          !isLoading ? (
            <View style={styles.empty}>
              <Text style={styles.emptyText}>Нет назначенных заданий</Text>
            </View>
          ) : null
        }
      />
    </View>
  );
}

const styles = StyleSheet.create({
  container: {
    flex: 1,
    backgroundColor: '#f1f5f9',
  },
  greeting: {
    fontSize: 18,
    fontWeight: '600',
    color: '#1e293b',
    marginBottom: 12,
  },
  statsRow: {
    flexDirection: 'row',
    paddingHorizontal: 16,
    paddingBottom: 8,
    gap: 12,
  },
  statCard: {
    flex: 1,
    borderRadius: 12,
    padding: 12,
    alignItems: 'center',
  },
  statNumber: {
    fontSize: 24,
    fontWeight: 'bold',
  },
  statLabel: {
    fontSize: 12,
    color: '#64748b',
    marginTop: 4,
  },
  list: {
    padding: 16,
    paddingTop: 8,
  },
  empty: {
    padding: 40,
    alignItems: 'center',
  },
  emptyText: {
    fontSize: 16,
    color: '#94a3b8',
  },
});
