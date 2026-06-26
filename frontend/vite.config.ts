/// <reference types="vitest/config" />
/// <reference types="vite-plugin-pwa/client" />
import { type Plugin } from 'vite';
import { defineConfig } from 'vitest/config';
import react from '@vitejs/plugin-react';
import tailwindcss from '@tailwindcss/vite';
import { visualizer } from 'rollup-plugin-visualizer';
import { VitePWA } from 'vite-plugin-pwa';

// More info at: https://storybook.js.org/docs/next/writing-tests/integrations/vitest-addon
function cspPlugin(): Plugin {
  // Production CSP — без 'unsafe-inline' в script-src (OWASP ASVS V5.3.3)
  // strict-dynamic отключает fallback к 'self' в старых браузерах
  // Для SSR/SPA nonce будет вставляться сервером Go
  const prodCSP = ["default-src 'self'", "script-src 'self' 'strict-dynamic' blob:", "worker-src 'self' blob:", "style-src 'self' 'unsafe-inline' https://fonts.googleapis.com", "font-src 'self' data: https://fonts.gstatic.com", "img-src 'self' data: https:", "connect-src 'self' https://nominatim.openstreetmap.org", "frame-src https://www.openstreetmap.org", "base-uri 'self'", "form-action 'self'"].join('; ');

  // Dev CSP — нужен 'unsafe-inline' для HMR и 'wasm-unsafe-eval' для Vite
  // Это приемлемо только для разработки
  const devCSP = ["default-src 'self'", "script-src 'self' 'unsafe-inline' 'wasm-unsafe-eval' blob:", "worker-src 'self' blob:", "style-src 'self' 'unsafe-inline' https://fonts.googleapis.com", "font-src 'self' data: https://fonts.gstatic.com", "img-src 'self' data: https:", "connect-src 'self' http://localhost:8080 https://nominatim.openstreetmap.org ws: wss:", "base-uri 'self'", "form-action 'self'"].join('; ');
  return {
    name: 'vite-plugin-csp',
    transformIndexHtml: {
      order: 'pre',
      handler(_html, ctx) {
        const csp = ctx.server ? devCSP : prodCSP;
        return [{
          tag: 'meta',
          attrs: {
            'http-equiv': 'Content-Security-Policy',
            content: csp
          },
          injectTo: 'head'
        }];
      }
    }
  };
}
export default defineConfig({
  plugins: [react(), tailwindcss(), cspPlugin(), VitePWA({
    registerType: 'autoUpdate',
    includeAssets: ['vite.svg'],
    manifest: {
      name: 'CCTV Health Monitor',
      short_name: 'CCTV Monitor',
      description: 'CCTV Health Monitoring & CMMS — управление CCTV инфраструктурой',
      theme_color: '#1e3a5f',
      background_color: '#f8fafc',
      display: 'standalone',
      orientation: 'any',
      start_url: '/',
      icons: [
        { src: '/vite.svg', sizes: '192x192', type: 'image/svg+xml' },
        { src: '/vite.svg', sizes: '512x512', type: 'image/svg+xml' },
      ],
    },
    workbox: {
      // Cache-first для статики (JS, CSS, изображения, шрифты)
      globPatterns: ['**/*.{js,css,html,svg,png,ico,woff2}'],
      globIgnores: ['**/stats.html'],
      runtimeCaching: [
        {
          // Network-first для API-запросов
          urlPattern: /^\/api\/.*/i,
          handler: 'NetworkFirst',
          options: {
            cacheName: 'api-cache',
            expiration: {
              maxEntries: 200,
              maxAgeSeconds: 60 * 60 * 24, // 24 часа
            },
            networkTimeoutSeconds: 10,
          },
        },
        {
          // Cache-first для Google Fonts
          urlPattern: /^https:\/\/fonts\.(googleapis|gstatic)\.com\/.*/i,
          handler: 'CacheFirst',
          options: {
            cacheName: 'google-fonts-cache',
            expiration: {
              maxEntries: 10,
              maxAgeSeconds: 60 * 60 * 24 * 365, // 1 год
            },
          },
        },
      ],
    },
  }), visualizer({
    filename: 'dist/stats.html',
    open: false,
    gzipSize: true,
    brotliSize: true
  })],
  build: {
    rollupOptions: {
      output: {
        // P3-2.3: Code splitting — выделение вендоров в отдельные чанки
        manualChunks(id: string) {
          if (id.includes('node_modules/react-dom') || id.includes('node_modules/react/') || id.includes('node_modules/react-router')) {
            return 'vendor-react';
          }
          if (id.includes('node_modules/recharts')) {
            return 'vendor-charts';
          }
          if (id.includes('node_modules/jspdf') || id.includes('node_modules/html2canvas')) {
            return 'vendor-pdf';
          }
          if (id.includes('node_modules/i18next')) {
            return 'vendor-i18n';
          }
        },
      },
    },
    chunkSizeWarningLimit: 500,
  },
  server: {
    host: '0.0.0.0',
    port: 3000,
    proxy: {
      '/api': {
        target: 'http://localhost:8080',
        changeOrigin: true,
        secure: false,
        ws: true
      }
    }
  },
  test: {
    globals: true,
    environment: 'jsdom',
    setupFiles: ['./src/test-setup.ts'],
    exclude: ['e2e/**', 'node_modules/**']
  }
});