import React from 'react';
import { View, Text, StyleSheet } from 'react-native';
import { Ionicons } from '@expo/vector-icons';

interface Props {
  passed: boolean;
  distanceMeters: number;
  accuracyMeters: number;
  error?: string;
}

export default function GPSStatus({ passed, distanceMeters, accuracyMeters, error }: Props) {
  return (
    <View style={[styles.container, passed ? styles.passed : styles.failed]}>
      <View style={styles.header}>
        <Ionicons
          name={passed ? 'location' : 'location-outline'}
          size={20}
          color={passed ? '#16a34a' : '#dc2626'}
        />
        <Text style={[styles.title, passed ? styles.passedText : styles.failedText]}>
          GPS Верификация
        </Text>
        <Ionicons
          name={passed ? 'checkmark-circle' : 'close-circle'}
          size={20}
          color={passed ? '#16a34a' : '#dc2626'}
        />
      </View>

      <View style={styles.metrics}>
        <View style={styles.metric}>
          <Text style={styles.label}>Расстояние</Text>
          <Text style={styles.value}>{distanceMeters.toFixed(0)} м</Text>
        </View>
        <View style={styles.metric}>
          <Text style={styles.label}>Точность</Text>
          <Text style={styles.value}>{accuracyMeters.toFixed(1)} м</Text>
        </View>
        <View style={styles.metric}>
          <Text style={styles.label}>Статус</Text>
          <Text style={[styles.value, passed ? styles.passedText : styles.failedText]}>
            {passed ? 'OK' : 'FAIL'}
          </Text>
        </View>
      </View>

      {error && <Text style={styles.error}>{error}</Text>}
    </View>
  );
}

const styles = StyleSheet.create({
  container: {
    borderRadius: 12,
    padding: 16,
    marginBottom: 12,
    borderWidth: 1,
  },
  passed: {
    backgroundColor: '#f0fdf4',
    borderColor: '#bbf7d0',
  },
  failed: {
    backgroundColor: '#fef2f2',
    borderColor: '#fecaca',
  },
  header: {
    flexDirection: 'row',
    alignItems: 'center',
    justifyContent: 'space-between',
    marginBottom: 12,
  },
  title: {
    fontSize: 14,
    fontWeight: '600',
    flex: 1,
    marginLeft: 8,
  },
  passedText: {
    color: '#16a34a',
  },
  failedText: {
    color: '#dc2626',
  },
  metrics: {
    flexDirection: 'row',
    justifyContent: 'space-between',
  },
  metric: {
    alignItems: 'center',
    flex: 1,
  },
  label: {
    fontSize: 11,
    color: '#64748b',
    marginBottom: 4,
  },
  value: {
    fontSize: 16,
    fontWeight: '700',
    color: '#1e293b',
  },
  error: {
    marginTop: 8,
    fontSize: 12,
    color: '#dc2626',
    textAlign: 'center',
  },
});