// ConflictResolutionModal — модальное окно для ручного разрешения offline sync конфликтов.
//
// P0-3.1: Conflict Resolution UI
//   - Diff-view: "Local vs Server" с подсветкой изменений
//   - Keep Local / Keep Server / Merge
//   - Timestamp comparison (formatRelativeTime)
//   - Telemetry logging resolved conflicts

import React, { useState, useCallback, useMemo } from 'react';
import {
  View,
  Text,
  TouchableOpacity,
  ScrollView,
  Modal,
  TextInput,
  StyleSheet,
} from 'react-native';
import { formatDistanceToNow } from 'date-fns';
import { ru } from 'date-fns/locale';

// ──────────────────────────────────────────────────
// Types
// ──────────────────────────────────────────────────

export interface ConflictField {
  /** Название поля (e.g. "status", "notes", "checklist") */
  name: string;
  /** Значение в локальной версии */
  localValue: string;
  /** Значение в серверной версии */
  serverValue: string;
  /** true — значение было изменено (отличается от сервера) */
  isChanged: boolean;
}

export interface ConflictData {
  /** ID сущности (work_order_id) */
  id: string;
  /** Человекочитаемое название сущности */
  label: string;
  /** Локальный timestamp последнего изменения */
  localTimestamp: number;
  /** Серверный timestamp последнего изменения */
  serverTimestamp: number;
  /** Список конфликтующих полей */
  fields: ConflictField[];
}

export type ConflictResolutionAction =
  | { type: 'keep_local'; conflictId: string }
  | { type: 'keep_server'; conflictId: string }
  | { type: 'merge'; conflictId: string; mergedFields: Record<string, string> };

interface ConflictResolutionModalProps {
  visible: boolean;
  conflicts: ConflictData[];
  onResolve: (action: ConflictResolutionAction) => void;
  onClose: () => void;
}

// ──────────────────────────────────────────────────
// Helpers
// ──────────────────────────────────────────────────

/**
 * Форматирует timestamp в относительное время на русском.
 * Примеры: "5 минут назад", "только что", "2 часа назад"
 */
function formatTimestamp(ts: number): string {
  try {
    return formatDistanceToNow(ts, { addSuffix: true, locale: ru });
  } catch {
    return 'неизвестно';
  }
}

/**
 * Определяет, какая версия новее.
 */
function getNewerLabel(
  localTs: number,
  serverTs: number,
): { label: string; isLocalNewer: boolean } {
  if (localTs > serverTs) {
    return { label: 'Локальная версия новее', isLocalNewer: true };
  }
  if (serverTs > localTs) {
    return { label: 'Серверная версия новее', isLocalNewer: false };
  }
  return { label: 'Одновременные изменения', isLocalNewer: false };
}

// ──────────────────────────────────────────────────
// Telemetry (структурированный лог; заменить на API вызов при интеграции)
// ──────────────────────────────────────────────────

function logTelemetry(
  event: string,
  payload: Record<string, unknown>,
): void {
  console.log(
    JSON.stringify({
      event: `conflict_resolution.${event}`,
      timestamp: Date.now(),
      payload,
    }),
  );
}

// ──────────────────────────────────────────────────
// MergeRow — компонент для слияния одного поля
// ──────────────────────────────────────────────────

interface MergeRowProps {
  field: ConflictField;
  mergedValue: string;
  onMergedValueChange: (value: string) => void;
}

const MergeRow = React.memo(function MergeRow({
  field,
  mergedValue,
  onMergedValueChange,
}: MergeRowProps) {
  return (
    <View style={styles.mergeRow}>
      <Text style={styles.mergeFieldLabel}>{field.name}</Text>

      <View style={styles.mergeDiffRow}>
        {/* Local — красный фон, если изменено */}
        <View
          style={[
            styles.mergeBox,
            field.isChanged ? styles.localChangedBox : styles.localUnchangedBox,
          ]}
        >
          <Text style={styles.mergeBoxLabel}>Local</Text>
          <Text
            style={[
              styles.mergeBoxValue,
              field.isChanged && styles.changedText,
            ]}
            numberOfLines={3}
          >
            {field.localValue}
          </Text>
        </View>

        {/* Server — зелёный фон */}
        <View style={[styles.mergeBox, styles.serverBox]}>
          <Text style={styles.mergeBoxLabel}>Server</Text>
          <Text style={styles.mergeBoxValue} numberOfLines={3}>
            {field.serverValue}
          </Text>
        </View>
      </View>

      {/* Поле ввода для ручного слияния */}
      <View style={styles.mergeInputWrapper}>
        <Text style={styles.mergeInputLabel}>Merged value</Text>
        <TextInput
          style={styles.mergeInput}
          value={mergedValue}
          onChangeText={onMergedValueChange}
          multiline
          placeholder="Введите объединённое значение..."
          placeholderTextColor="#94a3b8"
        />
      </View>
    </View>
  );
});

// ──────────────────────────────────────────────────
// ConflictCard — карточка одного конфликта
// ──────────────────────────────────────────────────

interface ConflictCardProps {
  conflict: ConflictData;
  mergeValues: Record<string, string>;
  onMergeValueChange: (fieldName: string, value: string) => void;
  onKeepLocal: () => void;
  onKeepServer: () => void;
  onStartMerge: () => void;
  isMerging: boolean;
}

const ConflictCard = React.memo(function ConflictCard({
  conflict,
  mergeValues,
  onMergeValueChange,
  onKeepLocal,
  onKeepServer,
  onStartMerge,
  isMerging,
}: ConflictCardProps) {
  const newerInfo = getNewerLabel(conflict.localTimestamp, conflict.serverTimestamp);

  return (
    <View style={styles.conflictCard}>
      {/* Заголовок сущности */}
      <Text style={styles.conflictTitle}>{conflict.label}</Text>

      {/* Timestamp comparison */}
      <View style={styles.timestampRow}>
        <View style={styles.timestampBadge}>
          <Text style={styles.timestampLabel}>Local</Text>
          <Text style={styles.timestampValue}>
            {formatTimestamp(conflict.localTimestamp)}
          </Text>
        </View>
        <Text style={styles.timestampVs}>vs</Text>
        <View style={styles.timestampBadge}>
          <Text style={styles.timestampLabel}>Server</Text>
          <Text style={styles.timestampValue}>
            {formatTimestamp(conflict.serverTimestamp)}
          </Text>
        </View>
      </View>

      {/* Индикатор новизны */}
      <View
        style={[
          styles.newerBadge,
          newerInfo.isLocalNewer
            ? styles.newerBadgeLocal
            : styles.newerBadgeServer,
        ]}
      >
        <Text style={styles.newerBadgeText}>{newerInfo.label}</Text>
      </View>

      {/* Diff поля */}
      {!isMerging ? (
        <>
          {conflict.fields.map((field) => (
            <View key={field.name} style={styles.diffFieldRow}>
              <Text style={styles.diffFieldName}>{field.name}</Text>
              <View style={styles.diffFieldValues}>
                {/* Local value — highlighted if changed */}
                <View
                  style={[
                    styles.diffValueBox,
                    field.isChanged
                      ? styles.diffLocalChanged
                      : styles.diffUnchanged,
                  ]}
                >
                  <Text style={styles.diffValueLabel}>Local</Text>
                  <Text
                    style={[
                      styles.diffValueText,
                      field.isChanged && styles.diffChangedText,
                    ]}
                    numberOfLines={2}
                  >
                    {field.localValue}
                  </Text>
                </View>

                {/* Arrow */}
                <Text style={styles.diffArrow}>→</Text>

                {/* Server value — green highlight */}
                <View
                  style={[
                    styles.diffValueBox,
                    field.isChanged
                      ? styles.diffServerChanged
                      : styles.diffUnchanged,
                  ]}
                >
                  <Text style={styles.diffValueLabel}>Server</Text>
                  <Text
                    style={[
                      styles.diffValueText,
                      field.isChanged && styles.diffServerText,
                    ]}
                    numberOfLines={2}
                  >
                    {field.serverValue}
                  </Text>
                </View>
              </View>
            </View>
          ))}

          {/* Action buttons */}
          <View style={styles.actionRow}>
            <TouchableOpacity
              onPress={onKeepLocal}
              style={styles.keepLocalBtn}
              activeOpacity={0.7}
            >
              <Text style={styles.btnTextLight}>Keep Local</Text>
            </TouchableOpacity>

            <TouchableOpacity
              onPress={onKeepServer}
              style={styles.keepServerBtn}
              activeOpacity={0.7}
            >
              <Text style={styles.btnTextLight}>Keep Server</Text>
            </TouchableOpacity>

            <TouchableOpacity
              onPress={onStartMerge}
              style={styles.mergeBtn}
              activeOpacity={0.7}
            >
              <Text style={styles.btnTextLight}>Merge</Text>
            </TouchableOpacity>
          </View>
        </>
      ) : (
        <>
          {conflict.fields.map((field) => (
            <MergeRow
              key={field.name}
              field={field}
              mergedValue={mergeValues[field.name] ?? field.localValue}
              onMergedValueChange={(val) => onMergeValueChange(field.name, val)}
            />
          ))}

          {/* Merge action buttons */}
          <View style={styles.actionRow}>
            <TouchableOpacity
              onPress={onKeepLocal}
              style={styles.keepLocalBtn}
              activeOpacity={0.7}
            >
              <Text style={styles.btnTextLight}>Keep Local</Text>
            </TouchableOpacity>

            <TouchableOpacity
              onPress={onKeepServer}
              style={styles.keepServerBtn}
              activeOpacity={0.7}
            >
              <Text style={styles.btnTextLight}>Keep Server</Text>
            </TouchableOpacity>
          </View>
        </>
      )}
    </View>
  );
});

// ──────────────────────────────────────────────────
// Main Component
// ──────────────────────────────────────────────────

export function ConflictResolutionModal({
  visible,
  conflicts,
  onResolve,
  onClose,
}: ConflictResolutionModalProps) {
  // Состояние: какой конфликт сейчас в режиме merge (по id или null)
  const [mergingId, setMergingId] = useState<string | null>(null);
  // Значения полей для merge: { [fieldName]: string }
  const [mergeValues, setMergeValues] = useState<Record<string, string>>({});

  // Сбрасываем состояние при скрытии/показе
  const handleClose = useCallback(() => {
    setMergingId(null);
    setMergeValues({});
    onClose();
  }, [onClose]);

  const handleKeepLocal = useCallback(
    (conflict: ConflictData) => {
      logTelemetry('keep_local', {
        conflictId: conflict.id,
        label: conflict.label,
        fields: conflict.fields.map((f) => f.name),
      });
      setMergingId(null);
      setMergeValues({});
      onResolve({ type: 'keep_local', conflictId: conflict.id });
    },
    [onResolve],
  );

  const handleKeepServer = useCallback(
    (conflict: ConflictData) => {
      logTelemetry('keep_server', {
        conflictId: conflict.id,
        label: conflict.label,
        fields: conflict.fields.map((f) => f.name),
      });
      setMergingId(null);
      setMergeValues({});
      onResolve({ type: 'keep_server', conflictId: conflict.id });
    },
    [onResolve],
  );

  const handleStartMerge = useCallback(
    (conflict: ConflictData) => {
      logTelemetry('merge_started', {
        conflictId: conflict.id,
        label: conflict.label,
      });
      // Инициализируем mergeValues значениями из local (как baseline)
      const initial: Record<string, string> = {};
      conflict.fields.forEach((f) => {
        initial[f.name] = f.localValue;
      });
      setMergeValues(initial);
      setMergingId(conflict.id);
    },
    [],
  );

  const handleMergeValueChange = useCallback(
    (fieldName: string, value: string) => {
      setMergeValues((prev) => ({ ...prev, [fieldName]: value }));
    },
    [],
  );

  const handleApplyMerge = useCallback(
    (conflict: ConflictData) => {
      logTelemetry('merge_applied', {
        conflictId: conflict.id,
        label: conflict.label,
        mergedFields: mergeValues,
      });
      setMergingId(null);
      setMergeValues({});
      onResolve({
        type: 'merge',
        conflictId: conflict.id,
        mergedFields: { ...mergeValues },
      });
    },
    [mergeValues, onResolve],
  );

  const handleCancelMerge = useCallback(() => {
    setMergingId(null);
    setMergeValues({});
  }, []);

  // Сортируем конфликты: самые старые по localTimestamp первыми
  const sortedConflicts = useMemo(
    () =>
      [...conflicts].sort((a, b) => a.localTimestamp - b.localTimestamp),
    [conflicts],
  );

  const isAnyMerging = mergingId !== null;

  return (
    <Modal visible={visible} animationType="slide" transparent>
      <View style={styles.overlay}>
        <View style={styles.sheet}>
          {/* Header */}
          <View style={styles.header}>
            <View style={styles.headerTop}>
              <Text style={styles.title}>Sync Conflicts</Text>
              <Text style={styles.badge}>
                {conflicts.length}
              </Text>
            </View>
            <Text style={styles.subtitle}>
              Локальные изменения конфликтуют с серверной версией.
              Выберите способ разрешения для каждого конфликта.
            </Text>
          </View>

          {/* Conflict list */}
          <ScrollView
            style={styles.scrollArea}
            contentContainerStyle={styles.scrollContent}
            showsVerticalScrollIndicator={false}
          >
            {sortedConflicts.length === 0 ? (
              <View style={styles.emptyState}>
                <Text style={styles.emptyText}>Нет конфликтов</Text>
              </View>
            ) : (
              sortedConflicts.map((conflict) => (
                <ConflictCard
                  key={conflict.id}
                  conflict={conflict}
                  mergeValues={mergeValues}
                  onMergeValueChange={handleMergeValueChange}
                  onKeepLocal={() => handleKeepLocal(conflict)}
                  onKeepServer={() => handleKeepServer(conflict)}
                  onStartMerge={() => handleStartMerge(conflict)}
                  isMerging={mergingId === conflict.id}
                />
              ))
            )}
          </ScrollView>

          {/* Footer */}
          <View style={styles.footer}>
            {isAnyMerging ? (
              <View style={styles.footerMergeRow}>
                <TouchableOpacity
                  onPress={handleCancelMerge}
                  style={styles.cancelMergeBtn}
                  activeOpacity={0.7}
                >
                  <Text style={styles.cancelMergeText}>Cancel Merge</Text>
                </TouchableOpacity>

                <TouchableOpacity
                  onPress={() => {
                    const conflict = conflicts.find((c) => c.id === mergingId);
                    if (conflict) handleApplyMerge(conflict);
                  }}
                  style={styles.applyMergeBtn}
                  activeOpacity={0.7}
                >
                  <Text style={styles.btnTextLight}>Apply Merge</Text>
                </TouchableOpacity>
              </View>
            ) : (
              <TouchableOpacity
                onPress={handleClose}
                style={styles.closeBtn}
                activeOpacity={0.7}
              >
                <Text style={styles.closeBtnText}>Close</Text>
              </TouchableOpacity>
            )}
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
    maxHeight: '90%',
  },

  // ── Header ──────────────────────────────────────
  header: {
    padding: 16,
    borderBottomWidth: 1,
    borderBottomColor: '#e2e8f0',
  },
  headerTop: {
    flexDirection: 'row',
    alignItems: 'center',
    gap: 8,
  },
  title: {
    fontSize: 18,
    fontWeight: '700',
    color: '#0f172a',
  },
  badge: {
    fontSize: 12,
    fontWeight: '700',
    color: '#fff',
    backgroundColor: '#ef4444',
    paddingHorizontal: 8,
    paddingVertical: 2,
    borderRadius: 10,
    overflow: 'hidden',
  },
  subtitle: {
    fontSize: 13,
    color: '#64748b',
    marginTop: 4,
    lineHeight: 18,
  },

  // ── Scroll ──────────────────────────────────────
  scrollArea: {
    paddingHorizontal: 16,
  },
  scrollContent: {
    paddingVertical: 16,
    gap: 16,
  },

  // ── Empty state ─────────────────────────────────
  emptyState: {
    paddingVertical: 32,
    alignItems: 'center',
  },
  emptyText: {
    fontSize: 15,
    color: '#94a3b8',
  },

  // ── Conflict Card ───────────────────────────────
  conflictCard: {
    padding: 14,
    backgroundColor: '#f8fafc',
    borderRadius: 12,
    borderWidth: 1,
    borderColor: '#e2e8f0',
  },
  conflictTitle: {
    fontSize: 15,
    fontWeight: '700',
    color: '#0f172a',
    marginBottom: 8,
  },

  // ── Timestamp row ───────────────────────────────
  timestampRow: {
    flexDirection: 'row',
    alignItems: 'center',
    gap: 8,
    marginBottom: 8,
  },
  timestampBadge: {
    flex: 1,
    paddingVertical: 6,
    paddingHorizontal: 10,
    borderRadius: 8,
    backgroundColor: '#f1f5f9',
  },
  timestampLabel: {
    fontSize: 10,
    fontWeight: '700',
    color: '#64748b',
    textTransform: 'uppercase',
    marginBottom: 2,
  },
  timestampValue: {
    fontSize: 12,
    color: '#334155',
    fontWeight: '500',
  },
  timestampVs: {
    fontSize: 11,
    color: '#94a3b8',
    fontWeight: '600',
  },

  // ── Newer badge ─────────────────────────────────
  newerBadge: {
    alignSelf: 'flex-start',
    paddingVertical: 3,
    paddingHorizontal: 8,
    borderRadius: 6,
    marginBottom: 10,
  },
  newerBadgeLocal: {
    backgroundColor: '#fef3c7',
  },
  newerBadgeServer: {
    backgroundColor: '#dbeafe',
  },
  newerBadgeText: {
    fontSize: 11,
    fontWeight: '600',
    color: '#475569',
  },

  // ── Diff field rows ─────────────────────────────
  diffFieldRow: {
    marginBottom: 10,
  },
  diffFieldName: {
    fontSize: 12,
    fontWeight: '600',
    color: '#475569',
    marginBottom: 4,
  },
  diffFieldValues: {
    flexDirection: 'row',
    alignItems: 'center',
    gap: 6,
  },
  diffValueBox: {
    flex: 1,
    padding: 8,
    borderRadius: 8,
    borderWidth: 1,
  },
  diffUnchanged: {
    backgroundColor: '#f8fafc',
    borderColor: '#e2e8f0',
  },
  diffLocalChanged: {
    backgroundColor: '#fef2f2',
    borderColor: '#fecaca',
  },
  diffServerChanged: {
    backgroundColor: '#f0fdf4',
    borderColor: '#bbf7d0',
  },
  diffValueLabel: {
    fontSize: 9,
    fontWeight: '700',
    color: '#94a3b8',
    textTransform: 'uppercase',
    marginBottom: 2,
  },
  diffValueText: {
    fontSize: 12,
    color: '#334155',
  },
  diffChangedText: {
    color: '#dc2626',
    fontWeight: '600',
  },
  diffServerText: {
    color: '#16a34a',
    fontWeight: '600',
  },
  diffArrow: {
    fontSize: 14,
    color: '#94a3b8',
  },

  // ── Action buttons ──────────────────────────────
  actionRow: {
    flexDirection: 'row',
    gap: 8,
    marginTop: 4,
  },
  keepLocalBtn: {
    flex: 1,
    paddingVertical: 10,
    backgroundColor: '#ef4444',
    borderRadius: 8,
    alignItems: 'center',
  },
  keepServerBtn: {
    flex: 1,
    paddingVertical: 10,
    backgroundColor: '#22c55e',
    borderRadius: 8,
    alignItems: 'center',
  },
  mergeBtn: {
    flex: 1,
    paddingVertical: 10,
    backgroundColor: '#6366f1',
    borderRadius: 8,
    alignItems: 'center',
  },
  btnTextLight: {
    color: '#fff',
    fontSize: 13,
    fontWeight: '600',
  },

  // ── Merge Row ───────────────────────────────────
  mergeRow: {
    marginBottom: 12,
    padding: 10,
    backgroundColor: '#f1f5f9',
    borderRadius: 10,
  },
  mergeFieldLabel: {
    fontSize: 12,
    fontWeight: '700',
    color: '#334155',
    marginBottom: 6,
  },
  mergeDiffRow: {
    flexDirection: 'row',
    gap: 6,
    marginBottom: 8,
  },
  mergeBox: {
    flex: 1,
    padding: 8,
    borderRadius: 8,
    borderWidth: 1,
  },
  localChangedBox: {
    backgroundColor: '#fef2f2',
    borderColor: '#fecaca',
  },
  localUnchangedBox: {
    backgroundColor: '#f8fafc',
    borderColor: '#e2e8f0',
  },
  serverBox: {
    backgroundColor: '#f0fdf4',
    borderColor: '#bbf7d0',
  },
  mergeBoxLabel: {
    fontSize: 9,
    fontWeight: '700',
    color: '#94a3b8',
    textTransform: 'uppercase',
    marginBottom: 2,
  },
  mergeBoxValue: {
    fontSize: 12,
    color: '#334155',
  },
  changedText: {
    color: '#dc2626',
    fontWeight: '600',
  },
  mergeInputWrapper: {
    marginTop: 4,
  },
  mergeInputLabel: {
    fontSize: 10,
    fontWeight: '600',
    color: '#64748b',
    marginBottom: 4,
    textTransform: 'uppercase',
  },
  mergeInput: {
    backgroundColor: '#fff',
    borderWidth: 1,
    borderColor: '#cbd5e1',
    borderRadius: 8,
    padding: 10,
    fontSize: 13,
    color: '#0f172a',
    minHeight: 44,
    textAlignVertical: 'top',
  },

  // ── Footer ──────────────────────────────────────
  footer: {
    padding: 16,
    borderTopWidth: 1,
    borderTopColor: '#e2e8f0',
  },
  footerMergeRow: {
    flexDirection: 'row',
    gap: 8,
  },
  cancelMergeBtn: {
    flex: 1,
    paddingVertical: 12,
    backgroundColor: '#f1f5f9',
    borderRadius: 12,
    alignItems: 'center',
  },
  cancelMergeText: {
    color: '#475569',
    fontSize: 15,
    fontWeight: '600',
  },
  applyMergeBtn: {
    flex: 1,
    paddingVertical: 12,
    backgroundColor: '#6366f1',
    borderRadius: 12,
    alignItems: 'center',
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
