import React, { useState, useCallback, useEffect } from 'react';
import {
  View,
  Text,
  FlatList,
  StyleSheet,
  ActivityIndicator,
  TouchableOpacity,
} from 'react-native';
import AsyncStorage from '@react-native-async-storage/async-storage';
import { ChecklistItem, ChecklistItemData } from './ChecklistItem';
import { useSyncStore } from '../../store/syncStore';

// ── Types ────────────────────────────────────────────────

interface RegulatoryChecklistProps {
  workOrderId: string;
  regulationId: string;
  regionCode: string;
  initialItems?: ChecklistItemData[];
  onComplete?: (results: ChecklistResult) => void;
  readOnly?: boolean;
}

interface ChecklistResult {
  workOrderId: string;
  regulationId: string;
  regionCode: string;
  items: ChecklistItemData[];
  completedAt: string;
  passedCount: number;
  failedCount: number;
  skippedCount: number;
  totalCount: number;
  synced: boolean;
}

// ── Storage keys ─────────────────────────────────────────

const STORAGE_PREFIX = '@regulatory_checklist_';

// ── Component ────────────────────────────────────────────

export const RegulatoryChecklist: React.FC<RegulatoryChecklistProps> = ({
  workOrderId,
  regulationId,
  regionCode,
  initialItems,
  onComplete,
  readOnly = false,
}) => {
  const [items, setItems] = useState<ChecklistItemData[]>(initialItems || []);
  const [loading, setLoading] = useState(!initialItems);
  const [saving, setSaving] = useState(false);
  const [synced, setSynced] = useState(true);
  const addToQueue = useSyncStore((s) => s.addToQueue);

  // Load from local storage on mount (offline-first)
  useEffect(() => {
    loadLocalChecklist();
  }, [workOrderId]);

  const loadLocalChecklist = async () => {
    try {
      const stored = await AsyncStorage.getItem(`${STORAGE_PREFIX}${workOrderId}`);
      if (stored) {
        const parsed: ChecklistItemData[] = JSON.parse(stored);
        setItems(parsed);
        setSynced(false); // not synced with server yet
      } else if (initialItems) {
        setItems(initialItems);
        await saveLocalChecklist(initialItems);
      }
    } catch (error) {
      console.error('Failed to load local checklist:', error);
      if (initialItems) {
        setItems(initialItems);
      }
    } finally {
      setLoading(false);
    }
  };

  const saveLocalChecklist = async (updatedItems: ChecklistItemData[]) => {
    try {
      await AsyncStorage.setItem(
        `${STORAGE_PREFIX}${workOrderId}`,
        JSON.stringify(updatedItems),
      );
    } catch (error) {
      console.error('Failed to save local checklist:', error);
    }
  };

  // ── Status change handler ───────────────────────────

  const handleStatusChange = useCallback(
    (order: number, status: ChecklistItemData['status']) => {
      setItems((prev) => {
        const updated = prev.map((item) =>
          item.order === order ? { ...item, status, synced: false } : item,
        );
        saveLocalChecklist(updated);
        setSynced(false);
        return updated;
      });
    },
    [workOrderId],
  );

  // ── Save and sync handler ───────────────────────────

  const handleSave = useCallback(async () => {
    setSaving(true);

    try {
      const passedCount = items.filter((i) => i.status === 'passed').length;
      const failedCount = items.filter((i) => i.status === 'failed').length;
      const skippedCount = items.filter((i) => i.status === 'skipped').length;

      const result: ChecklistResult = {
        workOrderId,
        regulationId,
        regionCode,
        items,
        completedAt: new Date().toISOString(),
        passedCount,
        failedCount,
        skippedCount,
        totalCount: items.length,
        synced: false,
      };

      // Save locally for offline-first
      await AsyncStorage.setItem(
        `${STORAGE_PREFIX}${workOrderId}_result`,
        JSON.stringify(result),
      );

      // Queue for server sync
      await addToQueue({
        type: 'checklist_complete',
        workOrderId,
        payload: result,
      });

      setSynced(true);

      if (onComplete) {
        onComplete(result);
      }
    } catch (error) {
      console.error('Failed to save checklist result:', error);
    } finally {
      setSaving(false);
    }
  }, [items, workOrderId, regulationId, regionCode, addToQueue, onComplete]);

  // ── Stats ───────────────────────────────────────────

  const passedCount = items.filter((i) => i.status === 'passed').length;
  const failedCount = items.filter((i) => i.status === 'failed').length;
  const pendingCount = items.filter((i) => i.status === 'pending').length;

  // ── Render ──────────────────────────────────────────

  if (loading) {
    return (
      <View style={styles.center}>
        <ActivityIndicator size="large" color="#2563EB" />
        <Text style={styles.loadingText}>Загрузка чек-листа...</Text>
      </View>
    );
  }

  if (items.length === 0) {
    return (
      <View style={styles.center}>
        <Text style={styles.emptyText}>Нет пунктов для проверки</Text>
      </View>
    );
  }

  return (
    <View style={styles.container}>
      {/* Header */}
      <View style={styles.header}>
        <Text style={styles.title}>Регламент ТО: {regionCode}</Text>
        <View style={styles.stats}>
          <View style={styles.statBox}>
            <Text style={styles.statValue}>{passedCount}</Text>
            <Text style={styles.statLabel}>Пройдено</Text>
          </View>
          <View style={styles.statBox}>
            <Text style={[styles.statValue, { color: '#EF4444' }]}>
              {failedCount}
            </Text>
            <Text style={styles.statLabel}>Не пройдено</Text>
          </View>
          <View style={styles.statBox}>
            <Text style={[styles.statValue, { color: '#6B7280' }]}>
              {pendingCount}
            </Text>
            <Text style={styles.statLabel}>Ожидает</Text>
          </View>
        </View>
      </View>

      {/* Progress bar */}
      <View style={styles.progressBar}>
        <View
          style={[
            styles.progressFill,
            { width: `${(passedCount / items.length) * 100}%` },
          ]}
        />
      </View>

      {/* Checklist items */}
      <FlatList
        data={items}
        keyExtractor={(item) => `item-${item.order}`}
        renderItem={({ item }) => (
          <ChecklistItem
            item={item}
            workOrderId={workOrderId}
            onStatusChange={handleStatusChange}
            disabled={readOnly}
          />
        )}
        contentContainerStyle={styles.list}
        showsVerticalScrollIndicator={false}
      />

      {/* Sync status */}
      {!synced && (
        <View style={styles.syncWarning}>
          <Text style={styles.syncWarningText}>
            ⚠ Изменения не синхронизированы. Будет отправлено при подключении к сети.
          </Text>
        </View>
      )}

      {/* Save button */}
      {!readOnly && (
        <TouchableOpacity
          style={[
            styles.saveButton,
            (saving || pendingCount === items.length) && styles.saveButtonDisabled,
          ]}
          onPress={handleSave}
          disabled={saving || pendingCount === items.length}
        >
          {saving ? (
            <ActivityIndicator color="#FFFFFF" size="small" />
          ) : (
            <Text style={styles.saveButtonText}>
              {pendingCount === items.length
                ? 'Отметьте хотя бы один пункт'
                : 'Сохранить и синхронизировать'}
            </Text>
          )}
        </TouchableOpacity>
      )}
    </View>
  );
};

// ── Styles ──────────────────────────────────────────────

const styles = StyleSheet.create({
  container: {
    flex: 1,
    backgroundColor: '#F9FAFB',
  },
  center: {
    flex: 1,
    justifyContent: 'center',
    alignItems: 'center',
    padding: 32,
  },
  loadingText: {
    marginTop: 12,
    fontSize: 14,
    color: '#6B7280',
  },
  emptyText: {
    fontSize: 16,
    color: '#9CA3AF',
  },
  header: {
    backgroundColor: '#FFFFFF',
    padding: 16,
    borderBottomWidth: 1,
    borderBottomColor: '#E5E7EB',
  },
  title: {
    fontSize: 16,
    fontWeight: '700',
    color: '#1F2937',
    marginBottom: 12,
  },
  stats: {
    flexDirection: 'row',
    gap: 16,
  },
  statBox: {
    alignItems: 'center',
  },
  statValue: {
    fontSize: 24,
    fontWeight: '700',
    color: '#10B981',
  },
  statLabel: {
    fontSize: 11,
    color: '#6B7280',
    marginTop: 2,
  },
  progressBar: {
    height: 4,
    backgroundColor: '#E5E7EB',
    borderRadius: 2,
    marginHorizontal: 16,
    marginTop: 8,
    overflow: 'hidden',
  },
  progressFill: {
    height: '100%',
    backgroundColor: '#10B981',
    borderRadius: 2,
  },
  list: {
    padding: 16,
  },
  syncWarning: {
    marginHorizontal: 16,
    marginBottom: 8,
    padding: 12,
    backgroundColor: '#FFF7ED',
    borderRadius: 8,
    borderWidth: 1,
    borderColor: '#FED7AA',
  },
  syncWarningText: {
    fontSize: 12,
    color: '#C2410C',
    textAlign: 'center',
  },
  saveButton: {
    margin: 16,
    paddingVertical: 14,
    backgroundColor: '#2563EB',
    borderRadius: 10,
    alignItems: 'center',
  },
  saveButtonDisabled: {
    backgroundColor: '#93C5FD',
  },
  saveButtonText: {
    fontSize: 16,
    fontWeight: '600',
    color: '#FFFFFF',
  },
});
