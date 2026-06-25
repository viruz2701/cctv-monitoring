// ══════════════════════════════════════════════════════════════
// CCTV Health Monitor — Service Worker
// Версия: 1.0.0
// Стратегия: Cache-first для статики, Network-first для API
// ══════════════════════════════════════════════════════════════

const CACHE_VERSION = 'cctv-cache-v1';
const STATIC_CACHE = `${CACHE_VERSION}-static`;
const API_CACHE = `${CACHE_VERSION}-api`;
const DYNAMIC_CACHE = `${CACHE_VERSION}-dynamic`;

// ── Ресурсы для pre-cache при установке ──────────────────────
const PRECACHE_URLS = [
  '/',
  '/offline.html',
  '/vite.svg',
];

// ── Установка ────────────────────────────────────────────────
self.addEventListener('install', (event) => {
  console.log(`[SW] Installing ${CACHE_VERSION}`);

  event.waitUntil(
    caches.open(STATIC_CACHE).then((cache) => {
      return cache.addAll(PRECACHE_URLS);
    }).then(() => {
      // Активируем сразу, не ждём закрытия страницы
      return self.skipWaiting();
    }),
  );
});

// ── Активация — очистка старых кэшей ─────────────────────────
self.addEventListener('activate', (event) => {
  console.log(`[SW] Activating ${CACHE_VERSION}`);

  const validCaches = [STATIC_CACHE, API_CACHE, DYNAMIC_CACHE];

  event.waitUntil(
    caches.keys().then((cacheNames) => {
      return Promise.all(
        cacheNames
          .filter((name) => !validCaches.includes(name))
          .map((name) => {
            console.log(`[SW] Deleting old cache: ${name}`);
            return caches.delete(name);
          }),
      );
    }).then(() => {
      // Контролируем все открытые страницы
      return self.clients.claim();
    }),
  );
});

// ── Перехват запросов ────────────────────────────────────────
self.addEventListener('fetch', (event) => {
  const { request } = event;
  const url = new URL(request.url);

  // Пропускаем не-GET запросы
  if (request.method !== 'GET') return;

  // Пропускаем запросы к расширениям браузера и analytics
  if (
    !url.protocol.startsWith('http') ||
    url.origin === 'chrome-extension' ||
    url.hostname === 'analytics.google.com'
  ) {
    return;
  }

  // ── API запросы (/api/v1/) — Network-first ──────────────
  if (url.pathname.startsWith('/api/v1/')) {
    event.respondWith(networkFirstWithFallback(request, API_CACHE));
    return;
  }

  // ── Навигационные запросы (HTML-страницы) ───────────────
  if (request.mode === 'navigate') {
    event.respondWith(networkFirstWithFallback(request, DYNAMIC_CACHE, '/offline.html'));
    return;
  }

  // ── Статические ресурсы (CSS, JS, fonts, images) — Cache-first ──
  if (isStaticAsset(url)) {
    event.respondWith(cacheFirstWithRefresh(request, STATIC_CACHE));
    return;
  }

  // ── Всё остальное — Network-first ───────────────────────
  event.respondWith(networkFirstWithFallback(request, DYNAMIC_CACHE));
});

// ══════════════════════════════════════════════════════════════
// Стратегии кэширования
// ══════════════════════════════════════════════════════════════

/**
 * Cache-first: отдаём из кэша, фоново обновляем.
 * Идеально для статики (chunked CSS/JS, fonts, иконки).
 */
async function cacheFirstWithRefresh(request, cacheName) {
  const cachedResponse = await caches.match(request);

  if (cachedResponse) {
    // Фоново обновляем кэш (не блокируем ответ)
    fetchAndCache(request, cacheName).catch(() => {});
    return cachedResponse;
  }

  // Нет в кэше — загружаем из сети
  try {
    const networkResponse = await fetch(request);
    if (isCacheable(networkResponse)) {
      await putInCache(cacheName, request, networkResponse.clone());
    }
    return networkResponse;
  } catch {
    // Если нет ни кэша, ни сети — fallback
    return caches.match('/offline.html');
  }
}

/**
 * Network-first: пытаемся загрузить из сети, при ошибке — кэш.
 * Идеально для API и навигации.
 */
async function networkFirstWithFallback(request, cacheName, fallbackUrl) {
  try {
    const networkResponse = await fetch(request);

    if (isCacheable(networkResponse)) {
      await putInCache(cacheName, request, networkResponse.clone());
    }

    return networkResponse;
  } catch {
    // Сеть недоступна — пробуем кэш
    const cachedResponse = await caches.match(request);
    if (cachedResponse) return cachedResponse;

    // Ничего нет — offline fallback
    if (fallbackUrl) {
      return caches.match(fallbackUrl);
    }

    // Нет fallback — возвращаем 503
    return new Response('Service Unavailable', { status: 503 });
  }
}

/**
 * Stale-while-revalidate: отдаём кэш, параллельно обновляем.
 */
async function staleWhileRevalidate(request, cacheName) {
  const cachedResponse = await caches.match(request);

  const fetchPromise = fetchAndCache(request, cacheName).catch(() => {});

  if (cachedResponse) {
    // Не ждём обновления
    return cachedResponse;
  }

  // Нет кэша — ждём сеть
  const networkResponse = await fetchPromise;
  return networkResponse || caches.match('/offline.html');
}

// ══════════════════════════════════════════════════════════════
// Утилиты
// ══════════════════════════════════════════════════════════════

function isStaticAsset(url) {
  const extensions = [
    '.css', '.js', '.mjs', '.woff', '.woff2', '.ttf', '.eot',
    '.svg', '.png', '.jpg', '.jpeg', '.gif', '.webp', '.ico',
    '.webmanifest',
  ];
  return extensions.some((ext) => url.pathname.endsWith(ext));
}

function isCacheable(response) {
  if (!response || !response.ok) return false;
  if (response.status !== 200) return false;

  // Не кэшируем Server-Sent Events
  const contentType = response.headers.get('Content-Type') || '';
  if (contentType.includes('text/event-stream')) return false;

  return true;
}

async function fetchAndCache(request, cacheName) {
  const response = await fetch(request);
  if (isCacheable(response)) {
    await putInCache(cacheName, request, response.clone());
  }
  return response;
}

async function putInCache(cacheName, request, response) {
  const cache = await caches.open(cacheName);
  await cache.put(request, response);
}

// ── Обработка сообщений от основного потока ──────────────────
self.addEventListener('message', (event) => {
  if (event.data && event.data.type === 'SKIP_WAITING') {
    self.skipWaiting();
  }

  if (event.data && event.data.type === 'CLEAR_CACHE') {
    const cacheNames = [STATIC_CACHE, API_CACHE, DYNAMIC_CACHE];
    Promise.all(
      cacheNames.map((name) => caches.delete(name)),
    ).then(() => {
      self.clients.matchAll().then((clients) => {
        clients.forEach((client) => {
          client.postMessage({ type: 'CACHE_CLEARED' });
        });
      });
    });
  }
});
