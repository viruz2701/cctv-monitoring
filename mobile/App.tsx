import React from 'react';
import { QueryClient, QueryClientProvider } from '@tanstack/react-query';
import { SafeAreaProvider } from 'react-native-safe-area-context';
import { StatusBar } from 'expo-status-bar';
import BackgroundSyncApp from './src/components/BackgroundSyncApp';

// ── Sentry (QA.4) ─────────────────────────────────────────────────────────
import { initSentry, SentryErrorBoundary } from './src/lib/sentry';

// Инициализация Sentry на уровне модуля
initSentry(process.env.EXPO_PUBLIC_SENTRY_DSN, {
  environment: process.env.NODE_ENV,
  tracesSampleRate: process.env.NODE_ENV === 'production' ? 0.2 : 0.0,
});

const queryClient = new QueryClient({
  defaultOptions: {
    queries: {
      retry: 2,
      staleTime: 5 * 60 * 1000,
    },
    mutations: {
      retry: 1,
    },
  },
});

export default function App() {
  return (
    <SentryErrorBoundary context={{ layer: 'app-root' }}>
      <QueryClientProvider client={queryClient}>
        <SafeAreaProvider>
          <StatusBar style="light" />
          <BackgroundSyncApp />
        </SafeAreaProvider>
      </QueryClientProvider>
    </SentryErrorBoundary>
  );
}
