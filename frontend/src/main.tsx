import { createRoot } from 'react-dom/client';
import { StrictMode, Suspense } from 'react';

// ═══ PWA Service Worker (ISO 27001 A.12.4.1) ═══════════════════════
function registerServiceWorker(): void {
  if ('serviceWorker' in navigator) {
    window.addEventListener('load', () => {
      navigator.serviceWorker
        .register('/sw.js')
        .then((registration) => {
          registration.addEventListener('updatefound', () => {
            const installingWorker = registration.installing;
            if (installingWorker) {
              installingWorker.addEventListener('statechange', () => {
                if (installingWorker.state === 'installed' && navigator.serviceWorker.controller) {
                  // Новый SW доступен — будет уведомление через Toast
                }
              });
            }
          });
        })
        .catch((error) => {
          console.error('[SW] Registration failed:', error);
        });
    });
  }
}

registerServiceWorker();

// ═══ Render (P0-CR-06) ═════════════════════════════════════════════
// Всё, кроме createRoot и StrictMode — динамические импорты.
// Это гарантирует main chunk < 200KB.
async function renderApp(): Promise<void> {
  const [{ ErrorBoundary }, { default: AppProviders }] = await Promise.all([
    import('./components/ErrorBoundary'),
    import('./components/AppProviders'),
  ]);

  // i18n инициализация — fire-and-forget, не блокирует рендер
  import('./i18n');
  // CSS тоже динамический, не в main bundle
  import('./index.css');

  const root = document.getElementById('root')!;
  root.innerHTML = '';

  createRoot(root).render(
    <StrictMode>
      <ErrorBoundary>
        <Suspense fallback={null}>
          <AppProviders />
        </Suspense>
      </ErrorBoundary>
    </StrictMode>
  );
}

renderApp();
