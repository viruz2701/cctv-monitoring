import React, { useCallback, useState } from 'react';
import { View, Modal, StyleSheet, TouchableOpacity, Text } from 'react-native';
import { useBackgroundSync } from '../hooks/useBackgroundSync';
import AppNavigator from '../navigation/AppNavigator';
import SyncStatusBar from './SyncStatusBar';

/**
 * Root component that initializes background sync and provides
 * sync management UI at the app level.
 *
 * - Calls useBackgroundSync() which registers expo-background-fetch (15 min)
 * - Initializes SyncService (NetInfo listener, SQLite, pending queue)
 * - Handles AppState transitions (sync on foreground)
 * - Shows SyncStatusBar for queue management, manual sync, status indicator
 *
 * Рендерит AppNavigator как дочерний с SyncStatusBar поверх.
 */
export default function BackgroundSyncApp() {
  // Mount background sync hook — side effects only
  const { pendingCount } = useBackgroundSync();

  const [showSyncPanel, setShowSyncPanel] = useState(false);

  const handleBadgePress = useCallback(() => {
    setShowSyncPanel((prev) => !prev);
  }, []);

  return (
    <View style={styles.root}>
      {/* Sync status bar — компактный режим, кликабельный */}
      <TouchableOpacity onPress={handleBadgePress} activeOpacity={0.8}>
        <SyncStatusBar />
      </TouchableOpacity>

      {/* Навигация */}
      <View style={styles.content}>
        <AppNavigator />
      </View>

      {/* Expanded sync panel — модалка с деталями */}
      <Modal
        visible={showSyncPanel}
        transparent
        animationType="slide"
        onRequestClose={() => setShowSyncPanel(false)}
      >
        <View style={styles.overlay}>
          <View style={styles.panel}>
            <View style={styles.panelHeader}>
              <Text style={styles.panelTitle}>Управление синхронизацией</Text>
              <TouchableOpacity
                onPress={() => setShowSyncPanel(false)}
                style={styles.closeButton}
              >
                <Text style={styles.closeText}>✕</Text>
              </TouchableOpacity>
            </View>

            <SyncStatusBar expanded showToggle />

            {pendingCount > 0 && (
              <Text style={styles.queueHint}>
                ⏳ {pendingCount} мутаций ожидают отправки на сервер.
                Они будут синхронизированы при восстановлении соединения.
              </Text>
            )}

            <TouchableOpacity
              style={styles.dismissButton}
              onPress={() => setShowSyncPanel(false)}
            >
              <Text style={styles.dismissText}>Закрыть</Text>
            </TouchableOpacity>
          </View>
        </View>
      </Modal>
    </View>
  );
}

// ──────────────────────────────────────────────────
// Styles
// ──────────────────────────────────────────────────

const styles = StyleSheet.create({
  root: {
    flex: 1,
  },
  content: {
    flex: 1,
  },
  overlay: {
    flex: 1,
    backgroundColor: 'rgba(0, 0, 0, 0.5)',
    justifyContent: 'flex-end',
  },
  panel: {
    backgroundColor: '#fff',
    borderTopLeftRadius: 20,
    borderTopRightRadius: 20,
    paddingTop: 16,
    paddingBottom: 32,
    paddingHorizontal: 8,
    gap: 12,
  },
  panelHeader: {
    flexDirection: 'row',
    justifyContent: 'space-between',
    alignItems: 'center',
    paddingHorizontal: 16,
    marginBottom: 4,
  },
  panelTitle: {
    fontSize: 18,
    fontWeight: '700',
    color: '#111827',
  },
  closeButton: {
    width: 32,
    height: 32,
    borderRadius: 16,
    backgroundColor: '#f3f4f6',
    alignItems: 'center',
    justifyContent: 'center',
  },
  closeText: {
    fontSize: 16,
    color: '#6b7280',
    fontWeight: '700',
  },
  queueHint: {
    fontSize: 13,
    color: '#6b7280',
    textAlign: 'center',
    paddingHorizontal: 24,
    lineHeight: 18,
  },
  dismissButton: {
    alignItems: 'center',
    paddingVertical: 10,
  },
  dismissText: {
    fontSize: 15,
    color: '#1e40af',
    fontWeight: '600',
  },
});
