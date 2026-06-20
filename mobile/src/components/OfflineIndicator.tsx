import React from 'react';
import { View, Text, StyleSheet } from 'react-native';
import { useSyncStore } from '../store/syncStore';

export default function OfflineIndicator() {
  const isOnline = useSyncStore((s) => s.isOnline);
  const queueLength = useSyncStore((s) => s.queue.length);

  if (isOnline && queueLength === 0) return null;

  return (
    <View style={[styles.container, !isOnline ? styles.offline : styles.pending]}>
      <Text style={styles.text}>
        {!isOnline
          ? 'Офлайн — данные синхронизируются при подключении'
          : `Ожидает синхронизации: ${queueLength} операций`}
      </Text>
    </View>
  );
}

const styles = StyleSheet.create({
  container: {
    paddingVertical: 6,
    paddingHorizontal: 16,
    alignItems: 'center',
  },
  offline: {
    backgroundColor: '#fef2f2',
  },
  pending: {
    backgroundColor: '#fef3c7',
  },
  text: {
    fontSize: 12,
    fontWeight: '500',
    color: '#1e293b',
  },
});