import React from 'react';
import { View, Text, StyleSheet } from 'react-native';

interface Props {
  status: string;
}

const STATUS_CONFIG: Record<string, { bg: string; text: string; label: string }> = {
  open: { bg: '#fef2f2', text: '#dc2626', label: 'Открыт' },
  in_progress: { bg: '#fef3c7', text: '#d97706', label: 'В работе' },
  completed: { bg: '#d1fae5', text: '#059669', label: 'Завершён' },
  cancelled: { bg: '#f1f5f9', text: '#64748b', label: 'Отменён' },
};

export default function StatusBadge({ status }: Props) {
  const config = STATUS_CONFIG[status] || STATUS_CONFIG.open;

  return (
    <View style={[styles.badge, { backgroundColor: config.bg }]}>
      <Text style={[styles.text, { color: config.text }]}>{config.label}</Text>
    </View>
  );
}

const styles = StyleSheet.create({
  badge: {
    paddingHorizontal: 10,
    paddingVertical: 4,
    borderRadius: 12,
    alignSelf: 'flex-start',
  },
  text: {
    fontSize: 12,
    fontWeight: '600',
  },
});