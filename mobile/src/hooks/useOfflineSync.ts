import { useEffect, useRef } from 'react';
import { AppState, AppStateStatus } from 'react-native';
import { useSyncStore } from '../store/syncStore';

export function useOfflineSync() {
  const { setOnline, processQueue, loadQueue } = useSyncStore();
  const appState = useRef<AppStateStatus>(AppState.currentState);

  useEffect(() => {
    loadQueue();
  }, []);

  useEffect(() => {
    const subscription = AppState.addEventListener('change', (nextAppState) => {
      if (appState.current.match(/inactive|background/) && nextAppState === 'active') {
        setOnline(true);
        processQueue();
      }
      appState.current = nextAppState;
    });

    return () => {
      subscription.remove();
    };
  }, [setOnline, processQueue]);

  return {
    addToQueue: useSyncStore((s) => s.addToQueue),
    processQueue,
    isOnline: useSyncStore((s) => s.isOnline),
  };
}