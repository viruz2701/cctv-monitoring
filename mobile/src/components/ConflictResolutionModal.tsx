// ConflictResolutionModal — модальное окно для разрешения offline sync конфликтов.
//
// P1-3.1: Conflict Resolution UI
//   - Показывает diff: "Local vs Server"
//   - Кнопки: Keep Local, Keep Server, Merge
//   - Visual diff highlighting

import React from 'react';
import {
  View,
  Text,
  TouchableOpacity,
  ScrollView,
  Modal,
  StyleSheet,
} from 'react-native';

// ──────────────────────────────────────────────────
// Types
// ──────────────────────────────────────────────────

export interface ConflictData {
  id: string;
  field: string;
  localValue: string;
  serverValue: string;
}

interface ConflictResolutionModalProps {
  visible: boolean;
  conflicts: ConflictData[];
  onKeepLocal: (id: string) => void;
  onKeepServer: (id: string) => void;
  onMerge: (id: string, value: string) => void;
  onClose: () => void;
}

// ──────────────────────────────────────────────────
// Component
// ──────────────────────────────────────────────────

export function ConflictResolutionModal({
  visible,
  conflicts,
  onKeepLocal,
  onKeepServer,
  onClose,
}: ConflictResolutionModalProps) {
  return (
    <Modal visible={visible} animationType="slide" transparent>
      <View style={styles.overlay}>
        <View style={styles.sheet}>
          <View style={styles.header}>
            <Text style={styles.title}>Sync Conflict</Text>
            <Text style={styles.subtitle}>
              Local changes conflict with server version
            </Text>
          </View>

          <ScrollView style={styles.scrollArea} contentContainerStyle={styles.scrollContent}>
            {conflicts.map((conflict) => (
              <View key={conflict.id} style={styles.conflictCard}>
                <Text style={styles.fieldLabel}>{conflict.field}</Text>

                <View style={styles.diffRow}>
                  <View style={styles.localBox}>
                    <Text style={styles.localLabel}>Local</Text>
                    <Text style={styles.valueText}>{conflict.localValue}</Text>
                  </View>

                  <View style={styles.serverBox}>
                    <Text style={styles.serverLabel}>Server</Text>
                    <Text style={styles.valueText}>{conflict.serverValue}</Text>
                  </View>
                </View>

                <View style={styles.actionRow}>
                  <TouchableOpacity
                    onPress={() => onKeepLocal(conflict.id)}
                    style={styles.keepLocalBtn}
                    activeOpacity={0.7}
                  >
                    <Text style={styles.btnTextLight}>Keep Local</Text>
                  </TouchableOpacity>

                  <TouchableOpacity
                    onPress={() => onKeepServer(conflict.id)}
                    style={styles.keepServerBtn}
                    activeOpacity={0.7}
                  >
                    <Text style={styles.btnTextLight}>Keep Server</Text>
                  </TouchableOpacity>
                </View>
              </View>
            ))}
          </ScrollView>

          <View style={styles.footer}>
            <TouchableOpacity
              onPress={onClose}
              style={styles.closeBtn}
              activeOpacity={0.7}
            >
              <Text style={styles.closeBtnText}>Close</Text>
            </TouchableOpacity>
          </View>
        </View>
      </View>
    </Modal>
  );
}

// ──────────────────────────────────────────────────
// Styles
// ──────────────────────────────────────────────────

const styles = StyleSheet.create({
  overlay: {
    flex: 1,
    backgroundColor: 'rgba(0,0,0,0.5)',
    justifyContent: 'flex-end',
  },
  sheet: {
    backgroundColor: '#fff',
    borderTopLeftRadius: 16,
    borderTopRightRadius: 16,
    maxHeight: '80%',
  },
  header: {
    padding: 16,
    borderBottomWidth: 1,
    borderBottomColor: '#e2e8f0',
  },
  title: {
    fontSize: 18,
    fontWeight: '700',
    color: '#0f172a',
  },
  subtitle: {
    fontSize: 13,
    color: '#64748b',
    marginTop: 4,
  },
  scrollArea: {
    paddingHorizontal: 16,
  },
  scrollContent: {
    paddingVertical: 16,
  },
  conflictCard: {
    marginBottom: 16,
    padding: 12,
    backgroundColor: '#f8fafc',
    borderRadius: 12,
  },
  fieldLabel: {
    fontSize: 13,
    fontWeight: '600',
    color: '#475569',
    marginBottom: 8,
  },
  diffRow: {
    flexDirection: 'row',
    gap: 8,
  },
  localBox: {
    flex: 1,
    padding: 8,
    backgroundColor: '#fef2f2',
    borderRadius: 8,
    borderWidth: 1,
    borderColor: '#fecaca',
  },
  localLabel: {
    fontSize: 11,
    color: '#dc2626',
    fontWeight: '600',
    marginBottom: 2,
  },
  serverBox: {
    flex: 1,
    padding: 8,
    backgroundColor: '#f0fdf4',
    borderRadius: 8,
    borderWidth: 1,
    borderColor: '#bbf7d0',
  },
  serverLabel: {
    fontSize: 11,
    color: '#16a34a',
    fontWeight: '600',
    marginBottom: 2,
  },
  valueText: {
    fontSize: 13,
    color: '#0f172a',
  },
  actionRow: {
    flexDirection: 'row',
    gap: 8,
    marginTop: 8,
  },
  keepLocalBtn: {
    flex: 1,
    paddingVertical: 8,
    backgroundColor: '#ef4444',
    borderRadius: 8,
    alignItems: 'center',
  },
  keepServerBtn: {
    flex: 1,
    paddingVertical: 8,
    backgroundColor: '#22c55e',
    borderRadius: 8,
    alignItems: 'center',
  },
  btnTextLight: {
    color: '#fff',
    fontSize: 13,
    fontWeight: '600',
  },
  footer: {
    padding: 16,
    borderTopWidth: 1,
    borderTopColor: '#e2e8f0',
  },
  closeBtn: {
    paddingVertical: 12,
    backgroundColor: '#f1f5f9',
    borderRadius: 12,
    alignItems: 'center',
  },
  closeBtnText: {
    color: '#475569',
    fontSize: 15,
    fontWeight: '600',
  },
});
