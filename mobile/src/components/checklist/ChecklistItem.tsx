import React, { useCallback } from 'react';
import { View, Text, TouchableOpacity, StyleSheet } from 'react-native';
import { useSyncStore } from '../../store/syncStore';

// ── Types ────────────────────────────────────────────────

export interface ChecklistItemData {
  order: number;
  description: string;
  category: string;
  status: 'pending' | 'passed' | 'failed' | 'skipped';
  comment?: string;
  isRequired: boolean;
  synced?: boolean; // offline-first sync status
}

interface ChecklistItemProps {
  item: ChecklistItemData;
  workOrderId: string;
  onStatusChange: (order: number, status: ChecklistItemData['status'], comment?: string) => void;
  disabled?: boolean;
}

// ── Item status colors ──────────────────────────────────

const STATUS_COLORS: Record<string, string> = {
  pending: '#6B7280',   // gray
  passed: '#10B981',    // green
  failed: '#EF4444',    // red
  skipped: '#F59E0B',   // amber
};

const STATUS_LABELS: Record<string, string> = {
  pending: '⏳ Ожидание',
  passed: '✅ Пройден',
  failed: '❌ Не пройден',
  skipped: '⏭ Пропущен',
};

// ── Component ───────────────────────────────────────────

export const ChecklistItem: React.FC<ChecklistItemProps> = ({
  item,
  workOrderId,
  onStatusChange,
  disabled = false,
}) => {
  const addToQueue = useSyncStore((s) => s.addToQueue);

  const handleStatusChange = useCallback(
    (newStatus: ChecklistItemData['status']) => {
      // Не сохраняем 'pending' в sync queue — только реальные действия
      if (newStatus === 'pending') {
        onStatusChange(item.order, newStatus);
        return;
      }

      onStatusChange(item.order, newStatus);

      // Offline-first: сохраняем в sync queue
      addToQueue({
        type: 'checklist_update',
        workOrderId,
        payload: {
          itemOrder: item.order,
          status: newStatus as 'passed' | 'failed' | 'skipped',
          timestamp: Date.now(),
        },
      }).catch((err: Error) => {
        console.error('Failed to queue checklist update:', err);
      });
    },
    [item.order, workOrderId, onStatusChange, addToQueue],
  );

  const statusColor = STATUS_COLORS[item.status] || STATUS_COLORS.pending;

  return (
    <View style={[styles.container, { borderLeftColor: statusColor }]}>
      {/* Header */}
      <View style={styles.header}>
        <View style={styles.titleRow}>
          <Text style={styles.order}>#{item.order}</Text>
          <Text style={styles.category}>{item.category}</Text>
          {item.isRequired && <Text style={styles.required}>*</Text>}
        </View>
        <Text style={[styles.status, { color: statusColor }]}>
          {STATUS_LABELS[item.status]}
        </Text>
      </View>

      {/* Description */}
      <Text style={styles.description}>{item.description}</Text>

      {/* Actions */}
      {!disabled && (
        <View style={styles.actions}>
          <TouchableOpacity
            style={[styles.actionBtn, styles.passBtn]}
            onPress={() => handleStatusChange('passed')}
          >
            <Text style={styles.actionBtnText}>Пройден</Text>
          </TouchableOpacity>

          <TouchableOpacity
            style={[styles.actionBtn, styles.failBtn]}
            onPress={() => handleStatusChange('failed')}
          >
            <Text style={styles.actionBtnText}>Не пройден</Text>
          </TouchableOpacity>

          <TouchableOpacity
            style={[styles.actionBtn, styles.skipBtn]}
            onPress={() => handleStatusChange('skipped')}
          >
            <Text style={styles.actionBtnText}>Пропустить</Text>
          </TouchableOpacity>
        </View>
      )}

      {/* Sync status */}
      {item.synced === false && (
        <View style={styles.syncBadge}>
          <Text style={styles.syncBadgeText}>⏳ Ожидание синхронизации</Text>
        </View>
      )}
    </View>
  );
};

// ── Styles ──────────────────────────────────────────────

const styles = StyleSheet.create({
  container: {
    backgroundColor: '#FFFFFF',
    borderRadius: 8,
    borderLeftWidth: 4,
    padding: 12,
    marginBottom: 8,
    shadowColor: '#000',
    shadowOffset: { width: 0, height: 1 },
    shadowOpacity: 0.05,
    shadowRadius: 2,
    elevation: 1,
  },
  header: {
    flexDirection: 'row',
    justifyContent: 'space-between',
    alignItems: 'center',
    marginBottom: 6,
  },
  titleRow: {
    flexDirection: 'row',
    alignItems: 'center',
    gap: 6,
  },
  order: {
    fontSize: 12,
    fontWeight: '700',
    color: '#374151',
  },
  category: {
    fontSize: 11,
    color: '#6B7280',
    backgroundColor: '#F3F4F6',
    paddingHorizontal: 6,
    paddingVertical: 2,
    borderRadius: 4,
    overflow: 'hidden',
  },
  required: {
    fontSize: 16,
    color: '#EF4444',
    fontWeight: '700',
  },
  status: {
    fontSize: 12,
    fontWeight: '600',
  },
  description: {
    fontSize: 14,
    color: '#1F2937',
    lineHeight: 20,
    marginBottom: 8,
  },
  actions: {
    flexDirection: 'row',
    gap: 8,
  },
  actionBtn: {
    flex: 1,
    paddingVertical: 8,
    borderRadius: 6,
    alignItems: 'center',
  },
  passBtn: {
    backgroundColor: '#D1FAE5',
  },
  failBtn: {
    backgroundColor: '#FEE2E2',
  },
  skipBtn: {
    backgroundColor: '#FEF3C7',
  },
  actionBtnText: {
    fontSize: 12,
    fontWeight: '600',
    color: '#374151',
  },
  syncBadge: {
    marginTop: 8,
    paddingVertical: 4,
    paddingHorizontal: 8,
    backgroundColor: '#FFF7ED',
    borderRadius: 4,
  },
  syncBadgeText: {
    fontSize: 11,
    color: '#C2410C',
  },
});
