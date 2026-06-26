import React from 'react';
import { useBackgroundSync } from '../hooks/useBackgroundSync';
import AppNavigator from '../navigation/AppNavigator';

/**
 * Root component that initializes background sync at the app level.
 *
 * - Calls useBackgroundSync() which registers expo-background-fetch (15 min)
 * - Initializes SyncService (NetInfo listener, SQLite, pending queue)
 * - Handles AppState transitions (sync on foreground)
 *
 * Рендерит AppNavigator как дочерний — не добавляет лишних обёрток.
 */
export default function BackgroundSyncApp() {
  // Mount background sync hook — side effects only, no UI
  useBackgroundSync();

  return <AppNavigator />;
}
