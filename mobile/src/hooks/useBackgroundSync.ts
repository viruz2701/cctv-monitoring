import { useEffect, useRef } from 'react';
import { AppState, AppStateStatus } from 'react-native';
import * as BackgroundFetch from 'expo-background-fetch';
import * as TaskManager from 'expo-task-manager';
import { useSyncStore } from '../store/syncStore';

const BACKGROUND_SYNC_TASK = 'cctv-background-sync';

// Register background fetch task
TaskManager.defineTask(BACKGROUND_SYNC_TASK, async () => {
  try {
    const state = useSyncStore.getState();
    if (!state.isOnline) {
      return BackgroundFetch.BackgroundFetchResult.NoData;
    }
    await state.processQueue();
    return BackgroundFetch.BackgroundFetchResult.NewData;
  } catch (error) {
    console.error('Background sync failed:', error);
    return BackgroundFetch.BackgroundFetchResult.Failed;
  }
});

export function useBackgroundSync() {
  const appState = useRef<AppStateStatus>(AppState.currentState);
  const { setOnline, processQueue } = useSyncStore();

  useEffect(() => {
    let isMounted = true;

    const setup = async () => {
      try {
        // Register background fetch task
        const status = await BackgroundFetch.getStatusAsync();
        if (status === BackgroundFetch.BackgroundFetchStatus.Denied) {
          console.warn('Background fetch is denied');
          return;
        }

        const isRegistered = await TaskManager.isTaskRegisteredAsync(BACKGROUND_SYNC_TASK);
        if (!isRegistered) {
          await BackgroundFetch.registerTaskAsync(BACKGROUND_SYNC_TASK, {
            minimumInterval: 15 * 60, // 15 minutes
            stopOnTerminate: false,
            startOnBoot: true,
          });
        }
      } catch (error) {
        console.error('Failed to setup background sync:', error);
      }
    };

    setup();

    // App state listener for foreground sync
    const subscription = AppState.addEventListener('change', (nextAppState) => {
      if (!isMounted) return;

      if (appState.current.match(/inactive|background/) && nextAppState === 'active') {
        setOnline(true);
        processQueue();
      } else if (nextAppState.match(/inactive|background/)) {
        setOnline(false);
      }
      appState.current = nextAppState;
    });

    return () => {
      isMounted = false;
      subscription.remove();
    };
  }, [setOnline, processQueue]);

  return {
    isRegistered: TaskManager.isTaskRegisteredAsync(BACKGROUND_SYNC_TASK),
  };
}
