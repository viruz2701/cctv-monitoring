import React from 'react';
import { View, Text, StyleSheet } from 'react-native';
import { Ionicons } from '@expo/vector-icons';

interface Props {
  passed: boolean;
  gpsMatch: boolean;
  timestampValid: boolean;
  hasEXIF: boolean;
  error?: string;
}

export default function EXIFStatus({ passed, gpsMatch, timestampValid, hasEXIF, error }: Props) {
  return (
    <View style={[styles.container, passed ? styles.passed : styles.failed]}>
      <View style={styles.header}>
        <Ionicons
          name={passed ? 'camera' : 'camera-outline'}
          size={20}
          color={passed ? '#16a34a' : '#dc2626'}
        />
        <Text style={[styles.title, passed ? styles.passedText : styles.failedText]}>
          EXIF Метаданные
        </Text>
        <Ionicons
          name={passed ? 'checkmark-circle' : 'close-circle'}
          size={20}
          color={passed ? '#16a34a' : '#dc2626'}
        />
      </View>

      <View style={styles.checks}>
        <View style={styles.checkRow}>
          <Ionicons
            name={hasEXIF ? 'checkmark' : 'close'}
            size={16}
            color={hasEXIF ? '#16a34a' : '#dc2626'}
          />
          <Text style={styles.checkText}>EXIF присутствует</Text>
        </View>
        <View style={styles.checkRow}>
          <Ionicons
            name={gpsMatch ? 'checkmark' : 'close'}
            size={16}
            color={gpsMatch ? '#16a34a' : '#dc2626'}
          />
          <Text style={styles.checkText}>GPS в EXIF совпадает</Text>
        </View>
        <View style={styles.checkRow}>
          <Ionicons
            name={timestampValid ? 'checkmark' : 'close'}
            size={16}
            color={timestampValid ? '#16a34a' : '#dc2626'}
          />
          <Text style={styles.checkText}>Время съёмки актуально</Text>
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
  checks: {
    gap: 8,
  },
  checkRow: {
    flexDirection: 'row',
    alignItems: 'center',
    gap: 8,
  },
  checkText: {
    fontSize: 13,
    color: '#334155',
  },
  error: {
    marginTop: 8,
    fontSize: 12,
    color: '#dc2626',
    textAlign: 'center',
  },
});