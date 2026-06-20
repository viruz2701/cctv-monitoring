import React from 'react';
import { View, Text, StyleSheet } from 'react-native';
import { Ionicons } from '@expo/vector-icons';

interface Props {
  passed: boolean;
  similarity: number;
  changeDetected: boolean;
  summary?: string;
  skipped: boolean;
  error?: string;
}

export default function AIScore({ passed, similarity, changeDetected, summary, skipped, error }: Props) {
  if (skipped) {
    return (
      <View style={[styles.container, styles.skipped]}>
        <View style={styles.header}>
          <Ionicons name="analytics-outline" size={20} color="#64748b" />
          <Text style={styles.skippedTitle}>AI Сравнение (Phase 2)</Text>
        </View>
        <Text style={styles.skippedText}>Пропущено — AI не настроен</Text>
      </View>
    );
  }

  const percent = Math.round(similarity * 100);

  return (
    <View style={[styles.container, passed ? styles.passed : styles.failed]}>
      <View style={styles.header}>
        <Ionicons
          name={passed ? 'analytics' : 'analytics-outline'}
          size={20}
          color={passed ? '#16a34a' : '#dc2626'}
        />
        <Text style={[styles.title, passed ? styles.passedText : styles.failedText]}>
          AI Сравнение фото
        </Text>
        <Ionicons
          name={passed ? 'checkmark-circle' : 'close-circle'}
          size={20}
          color={passed ? '#16a34a' : '#dc2626'}
        />
      </View>

      <View style={styles.scoreRow}>
        <View style={styles.scoreCircle}>
          <Text style={[styles.scoreValue, passed ? styles.passedText : styles.failedText]}>
            {percent}%
          </Text>
          <Text style={styles.scoreLabel}>Сходство</Text>
        </View>

        <View style={styles.details}>
          <View style={styles.checkRow}>
            <Ionicons
              name={changeDetected ? 'checkmark' : 'close'}
              size={16}
              color={changeDetected ? '#16a34a' : '#dc2626'}
            />
            <Text style={styles.checkText}>Изменения обнаружены</Text>
          </View>
          {summary && <Text style={styles.summary}>{summary}</Text>}
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
  skipped: {
    backgroundColor: '#f8fafc',
    borderColor: '#e2e8f0',
  },
  header: {
    flexDirection: 'row',
    alignItems: 'center',
    gap: 8,
    marginBottom: 12,
  },
  title: {
    fontSize: 14,
    fontWeight: '600',
    flex: 1,
  },
  skippedTitle: {
    fontSize: 14,
    fontWeight: '600',
    color: '#64748b',
    flex: 1,
  },
  passedText: {
    color: '#16a34a',
  },
  failedText: {
    color: '#dc2626',
  },
  skippedText: {
    fontSize: 12,
    color: '#94a3b8',
    textAlign: 'center',
  },
  scoreRow: {
    flexDirection: 'row',
    alignItems: 'center',
    gap: 16,
  },
  scoreCircle: {
    width: 72,
    height: 72,
    borderRadius: 36,
    borderWidth: 3,
    borderColor: '#e2e8f0',
    justifyContent: 'center',
    alignItems: 'center',
    backgroundColor: '#fff',
  },
  scoreValue: {
    fontSize: 18,
    fontWeight: '800',
  },
  scoreLabel: {
    fontSize: 10,
    color: '#64748b',
  },
  details: {
    flex: 1,
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
  summary: {
    fontSize: 12,
    color: '#64748b',
    fontStyle: 'italic',
  },
  error: {
    marginTop: 8,
    fontSize: 12,
    color: '#dc2626',
    textAlign: 'center',
  },
});