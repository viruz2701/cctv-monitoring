import React from 'react';
import {
  View,
  Text,
  TouchableOpacity,
  ScrollView,
  StyleSheet,
  Alert,
} from 'react-native';
import { useAuthStore } from '../store/authStore';
import { useTechnicianProfile, useTechnicianStats } from '../hooks/useWorkOrders';
import { getGreeting } from '../utils/dateHelpers';

export default function ProfileScreen() {
  const { user, logout } = useAuthStore();
  const { data: profile } = useTechnicianProfile();
  const { data: stats } = useTechnicianStats();

  const handleLogout = () => {
    Alert.alert('Выход', 'Вы уверены, что хотите выйти?', [
      { text: 'Отмена', style: 'cancel' },
      {
        text: 'Выйти',
        style: 'destructive',
        onPress: () => logout(),
      },
    ]);
  };

  return (
    <ScrollView style={styles.container} contentContainerStyle={styles.content}>
      <View style={styles.header}>
        <View style={styles.avatar}>
          <Text style={styles.avatarText}>
            {user?.username?.charAt(0).toUpperCase() || 'Т'}
          </Text>
        </View>
        <Text style={styles.name}>{user?.username || 'Техник'}</Text>
        <Text style={styles.role}>{user?.role || 'technician'}</Text>
      </View>

      {stats && (
        <View style={styles.statsRow}>
          <View style={styles.statCard}>
            <Text style={styles.statNumber}>{stats.completed_this_month || 0}</Text>
            <Text style={styles.statLabel}>За месяц</Text>
          </View>
          <View style={styles.statCard}>
            <Text style={styles.statNumber}>{stats.total_work_orders || 0}</Text>
            <Text style={styles.statLabel}>Всего</Text>
          </View>
          <View style={styles.statCard}>
            <Text style={styles.statNumber}>{stats.on_time_percent || 0}%</Text>
            <Text style={styles.statLabel}>Вовремя</Text>
          </View>
        </View>
      )}

      {profile && (
        <View style={styles.section}>
          <Text style={styles.sectionTitle}>Информация</Text>
          <View style={styles.infoRow}>
            <Text style={styles.infoLabel}>Нагрузка</Text>
            <Text style={styles.infoValue}>
              {profile.current_workload}/{profile.max_workload}
            </Text>
          </View>
          <View style={styles.infoRow}>
            <Text style={styles.infoLabel}>Локация</Text>
            <Text style={styles.infoValue}>{profile.base_location || '—'}</Text>
          </View>
          {profile.skills && profile.skills.length > 0 && (
            <View style={styles.infoRow}>
              <Text style={styles.infoLabel}>Навыки</Text>
              <Text style={styles.infoValue}>{profile.skills.join(', ')}</Text>
            </View>
          )}
        </View>
      )}

      <TouchableOpacity style={styles.logoutButton} onPress={handleLogout}>
        <Text style={styles.logoutText}>Выйти</Text>
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
  header: {
    alignItems: 'center',
    paddingVertical: 24,
  },
  avatar: {
    width: 72,
    height: 72,
    borderRadius: 36,
    backgroundColor: '#2563eb',
    justifyContent: 'center',
    alignItems: 'center',
    marginBottom: 12,
  },
  avatarText: {
    fontSize: 28,
    fontWeight: 'bold',
    color: '#fff',
  },
  name: {
    fontSize: 20,
    fontWeight: '700',
    color: '#1e293b',
  },
  role: {
    fontSize: 14,
    color: '#64748b',
    marginTop: 4,
  },
  statsRow: {
    flexDirection: 'row',
    gap: 12,
    marginBottom: 16,
  },
  statCard: {
    flex: 1,
    backgroundColor: '#fff',
    borderRadius: 12,
    padding: 16,
    alignItems: 'center',
  },
  statNumber: {
    fontSize: 22,
    fontWeight: 'bold',
    color: '#2563eb',
  },
  statLabel: {
    fontSize: 12,
    color: '#64748b',
    marginTop: 4,
  },
  section: {
    backgroundColor: '#fff',
    borderRadius: 12,
    padding: 16,
    marginBottom: 16,
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
  logoutButton: {
    backgroundColor: '#fff',
    borderRadius: 12,
    padding: 16,
    alignItems: 'center',
    borderWidth: 1,
    borderColor: '#fecaca',
  },
  logoutText: {
    fontSize: 16,
    fontWeight: '600',
    color: '#dc2626',
  },
});