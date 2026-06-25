import React from 'react';
import ReactDOM from 'react-dom/client';
import App from './App';
import { ErrorBoundary } from './components/ErrorBoundary';
import './index.css';
import './i18n';

// ── Регистрация Service Worker для PWA ─────────────────────

function registerServiceWorker(): void {
  if ('serviceWorker' in navigator) {
    // Откладываем регистрацию до полной загрузки страницы
    window.addEventListener('load', () => {
      navigator.serviceWorker
        .register('/sw.js')
        .then((registration) => {
          console.log(
            '[SW] Registered successfully, scope:',
            registration.scope,
          );

          // Проверяем обновления
          registration.addEventListener('updatefound', () => {
            const installingWorker = registration.installing;
            if (installingWorker) {
              installingWorker.addEventListener('statechange', () => {
                if (installingWorker.state === 'installed') {
                  if (navigator.serviceWorker.controller) {
                    // Новый SW доступен — уведомляем пользователя
                    console.log(
                      '[SW] New version available. Reload to update.',
                    );
                  }
                }
              });
            }
          });
        })
        .catch((error) => {
          console.error('[SW] Registration failed:', error);
        });

      // Слушаем сообщения от Service Worker
      navigator.serviceWorker.addEventListener('message', (event) => {
        if (event.data?.type === 'CACHE_CLEARED') {
          console.log('[SW] Cache cleared');
        }
      });
    });
  }
}

registerServiceWorker();

// ── Рендер приложения ───────────────────────────────────────

ReactDOM.createRoot(document.getElementById('root')!).render(
  <React.StrictMode>
    <ErrorBoundary>
      <App />
    </ErrorBoundary>
  </React.StrictMode>
);