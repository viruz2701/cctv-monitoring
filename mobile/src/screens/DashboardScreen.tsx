import React from 'react';
import { View, Text, FlatList, TouchableOpacity, RefreshControl, StyleSheet } from 'react-native';
import { useQuery } from '@tanstack/react-query';
import { useNavigation } from '@react-navigation/native';
import { NativeStackNavigationProp } from '@react-navigation/native-stack';
import { workOrdersApi } from '../api/workOrders';
import { RootStackParamList } from '../types';
import WorkOrderCard from '../components/WorkOrderCard';
import OfflineIndicator from '../components/OfflineIndicator';
import { getGreeting } from '../utils/dateHelpers';
import { useAuthStore } from '../store/authStore';

export default function DashboardScreen() {
  const navigation = useNavigation<NativeStackNavigationProp<RootStackParamList>>();
  const user = useAuthStore((s) => s.user);

  const {
    data: workOrders,
    isLoading,
    refetch,
    isRefetching,
  } = useQuery({
    queryKey: ['myWorkOrders'],
    queryFn: workOrdersApi.getMyWorkOrders,
  });

  const openWorkOrder = (id: string) => {
    navigation.navigate('WorkOrderDetail', { workOrderId: id });
  };

  const getStatusCounts = () => {
    if (!workOrders) return { open: 0, inProgress: 0, completed: 0 };
    return {
      open: workOrders.filter((wo) => wo.status === 'open').length,
      inProgress: workOrders.filter((wo) => wo.status === 'in_progress').length,
      completed: workOrders.filter((wo) => wo.status === 'completed').length,
    };
  };

  const counts = getStatusCounts();

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
        renderItem={({ item }) => (
          <WorkOrderCard workOrder={item} onPress={() => openWorkOrder(item.id)} />
        )}
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