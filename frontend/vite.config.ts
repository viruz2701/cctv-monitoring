/// <reference types="vitest/config" />
/// <reference types="vite-plugin-pwa/client" />
import { type Plugin } from 'vite';
import { defineConfig } from 'vitest/config';
import react from '@vitejs/plugin-react';
import tailwindcss from '@tailwindcss/vite';
import { visualizer } from 'rollup-plugin-visualizer';
import { VitePWA } from 'vite-plugin-pwa';
import { sentryVitePlugin } from '@sentry/vite-plugin';
import { imagetools } from 'vite-imagetools';

// More info at: https://storybook.js.org/docs/next/writing-tests/integrations/vitest-addon
function cspPlugin(): Plugin {
  // Production CSP — без 'unsafe-inline' в script-src (OWASP ASVS V5.3.3)
  // strict-dynamic отключает fallback к 'self' в старых браузерах
  // Для SSR/SPA nonce будет вставляться сервером Go
  const prodCSP = ["default-src 'self'", "script-src 'self' 'strict-dynamic' blob:", "worker-src 'self' blob:", "style-src 'self' 'unsafe-inline' https://fonts.googleapis.com", "font-src 'self' data: https://fonts.gstatic.com", "img-src 'self' data: https:", "connect-src 'self' https://nominatim.openstreetmap.org", "frame-src https://www.openstreetmap.org", "base-uri 'self'", "form-action 'self'"].join('; ');

  // Dev CSP — нужен 'unsafe-inline' для HMR и 'wasm-unsafe-eval' для Vite
  // Это приемлемо только для разработки
  const devCSP = ["default-src 'self'", "script-src 'self' 'unsafe-inline' 'wasm-unsafe-eval' blob:", "worker-src 'self' blob:", "style-src 'self' 'unsafe-inline' https://fonts.googleapis.com", "font-src 'self' data: https://fonts.gstatic.com", "img-src 'self' data: https:", "connect-src 'self' http://localhost:8080 https://nominatim.openstreetmap.org ws: wss:", "frame-src https://www.openstreetmap.org", "base-uri 'self'", "form-action 'self'"].join('; ');
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
    },
    // P2-MED-21: CSP HTTP response headers (defense-in-depth: meta + headers)
    configureServer(server) {
      server.middlewares.use((_req, res, next) => {
        const csp = prodCSP;
        res.setHeader('Content-Security-Policy', csp);
        res.setHeader('X-Content-Type-Options', 'nosniff');
        res.setHeader('X-Frame-Options', 'DENY');
        res.setHeader('Referrer-Policy', 'strict-origin-when-cross-origin');
        res.setHeader('Permissions-Policy', 'camera=(), microphone=(), geolocation=(self)');
        next();
      });
    },
  };
}
// Sentry source maps upload (только при наличии SENTRY_AUTH_TOKEN)
const sentryPlugin = process.env.SENTRY_AUTH_TOKEN
  ? sentryVitePlugin({
      authToken: process.env.SENTRY_AUTH_TOKEN,
      org: process.env.SENTRY_ORG || 'cctv-monitor',
      project: process.env.SENTRY_PROJECT || 'frontend',
      telemetry: false,
      sourcemaps: {
        assets: './dist/assets/**',
        filesToDeleteAfterUpload: ['./dist/assets/*.map'],
      },
    })
  : null;

export default defineConfig({
  plugins: [
    react(),
    tailwindcss(),
    imagetools({
      // P3-LOW-01: Автоматическая WebP конвертация изображений
      // Использует sharp (уже в devDependencies) для преобразования
      defaultDirectives: () => new URLSearchParams([
        ['webp', 'true'],        // Конвертация в WebP
        ['format', 'webp'],      // Принудительный формат
        ['quality', '80'],       // Качество 80% (P3-LOW-01)
        ['w', '800'],            // Максимальная ширина 800px
      ]),
    }),
    cspPlugin(),
    VitePWA({
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
        // P2-MED-18: увеличен лимит для Storybook preview (sb-manager ~3.2MB)
        maximumFileSizeToCacheInBytes: 5 * 1024 * 1024, // 5 MB
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
    }),
    sentryPlugin,
    visualizer({
      filename: 'dist/stats.html',
      open: false,
      gzipSize: true,
      brotliSize: true,
    }),
  ].filter(Boolean),
  build: {
    target: 'es2020', // P2-OPT.3: современные браузеры — меньше полифиллов
    minify: 'esbuild', // P2-OPT.4: esbuild быстрее terser, достаточно для production
    cssMinify: 'lightningcss', // P2-OPT.5: LightningCSS быстрее esbuild для CSS
    cssCodeSplit: true, // P2-OPT.6: раздельный CSS для каждого чанка
    sourcemap: Boolean(process.env.SENTRY_AUTH_TOKEN), // P2-OPT.7: sourcemap только для Sentry
    reportCompressedSize: false, // P2-OPT.8: ускоряет сборку, не влияет на деплой
    assetsInlineLimit: 4096, // P2-OPT.9: инлайн SVG/изображений <4KB как data URI
    emptyOutDir: true, // P2-OPT.14: очистка dist перед сборкой
    rollupOptions: {
      output: {
        // P3-2.3: Code splitting — выделение вендоров в отдельные чанки
        // P1-2.1: Bundle size reduction — выделение тяжёлых библиотек в отдельные чанки
        // P0-CR-06: Route-based code splitting + динамический main.tsx
        manualChunks(id: string) {
          // Core React + scheduler (часть React)
          if (id.includes('node_modules/react-dom') || id.includes('node_modules/react/') || id.includes('node_modules/react-router') || id.includes('node_modules/react-hook-form') || id.includes('node_modules/scheduler')) {
            return 'vendor-react';
          }
          // Charts & visualization (Nivo — tree-shakeable, ~180KB)
          if (id.includes('node_modules/@nivo') || id.includes('node_modules/chart')) {
            return 'vendor-nivo';
          }
          // i18n
          if (id.includes('node_modules/i18next')) {
            return 'vendor-i18n';
          }
          // Calendar (Schedule-X — ~80KB, replaces FullCalendar ~328KB)
          if (id.includes('node_modules/@schedule-x')) {
            return 'vendor-schedule-x';
          }
          // Excel (ExcelJS — MIT, ~350KB)
          if (id.includes('node_modules/exceljs')) {
            return 'vendor-excel';
          }
          // Drag & Drop
          if (id.includes('node_modules/@hello-pangea')) {
            return 'vendor-dnd';
          }
          // Workflow builder (@xyflow/react ~300KB)
          if (id.includes('node_modules/@xyflow') || id.includes('node_modules/react-flow')) {
            return 'vendor-workflow';
          }
          // Grid layout (react-grid-layout ~150KB)
          if (id.includes('node_modules/react-grid-layout')) {
            return 'vendor-grid';
          }
          // Tutorials & onboarding (react-joyride ~200KB)
          if (id.includes('node_modules/react-joyride')) {
            return 'vendor-joyride';
          }
          // Markdown (react-markdown + remark/rehype ~150KB)
          if (id.includes('node_modules/react-markdown') || id.includes('node_modules/remark-') || id.includes('node_modules/rehype-') || id.includes('node_modules/unified') || id.includes('node_modules/mdast')) {
            return 'vendor-markdown';
          }
          // Date picker (react-datepicker ~100KB)
          if (id.includes('node_modules/react-datepicker')) {
            return 'vendor-datepicker';
          }
          // Image comparison (react-compare-image ~50KB) — P2-OPT.10: выделяем отдельно
          if (id.includes('node_modules/react-compare-image')) {
            return 'vendor-compare-image';
          }
          // Query & state management (@tanstack/react-query, query-core, react-virtual, zustand)
          if (id.includes('node_modules/@tanstack') || id.includes('node_modules/zustand')) {
            return 'vendor-state';
          }
          // Form handling (hookform + zod)
          if (id.includes('node_modules/zod') || id.includes('node_modules/@hookform')) {
            return 'vendor-forms';
          }
          // Sentry
          if (id.includes('node_modules/@sentry')) {
            return 'vendor-sentry';
          }
          // Icons (lucide-react — tree-shakeable, ~30KB shared)
          if (id.includes('node_modules/lucide-react')) {
            return 'vendor-icons';
          }
          // Everything else from node_modules
          if (id.includes('node_modules')) {
            return 'vendor-other';
          }
        },
        // P2-OPT.11: assetFileNames для кэширования
        assetFileNames: 'assets/[name]-[hash:8][extname]',
        chunkFileNames: 'assets/[name]-[hash:8].js',
        entryFileNames: 'assets/[name]-[hash:8].js',
      },
    },
    chunkSizeWarningLimit: 350, // P2-OPT.12: уменьшаем порог для раннего обнаружения
    rolldownOptions: {
      external: ['preact', '@preact/signals'], // @schedule-x/calendar использует preact как transitive dep
    },
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
    exclude: ['e2e/**', 'tests/**', 'node_modules/**', '**/*.stories.*'],
    coverage: {
      provider: 'v8',
      reporter: ['text', 'json', 'html', 'lcov'],
      reportsDirectory: '../coverage',
      include: [
        'src/components/**/*.{ts,tsx}',
        'src/hooks/**/*.{ts,tsx}',
        'src/store/**/*.{ts,tsx}',
        'src/services/**/*.{ts,tsx}',
        'src/utils/**/*.{ts,tsx}',
      ],
      exclude: [
        '**/*.stories.{ts,tsx}',
        '**/*.test.{ts,tsx}',
        '**/__tests__/**',
        '**/index.ts',
        'src/types/**',
        'src/stories/**',
      ],
      // P2-MED-16: поднято с 82% → 85% через новые тесты critical paths
      thresholds: {
        statements: 85,
        branches: 80,
        functions: 85,
        lines: 85,
      },
    },
  }
});