/// <reference types="vitest/config" />
import { type Plugin } from 'vite';
import { defineConfig } from 'vitest/config';
import react from '@vitejs/plugin-react';
import tailwindcss from '@tailwindcss/vite';
import { visualizer } from 'rollup-plugin-visualizer';

// More info at: https://storybook.js.org/docs/next/writing-tests/integrations/vitest-addon
function cspPlugin(): Plugin {
  // Production CSP — без 'unsafe-inline' в script-src (OWASP ASVS V5.3.3)
  // strict-dynamic отключает fallback к 'self' в старых браузерах
  // Для SSR/SPA nonce будет вставляться сервером Go
  const prodCSP = ["default-src 'self'", "script-src 'self' 'strict-dynamic' blob:", "worker-src 'self' blob:", "style-src 'self' 'unsafe-inline' https://fonts.googleapis.com", "font-src 'self' data: https://fonts.gstatic.com", "img-src 'self' data: https:", "connect-src 'self' https://nominatim.openstreetmap.org", "frame-ancestors 'none'", "base-uri 'self'", "form-action 'self'"].join('; ');

  // Dev CSP — нужен 'unsafe-inline' для HMR и 'wasm-unsafe-eval' для Vite
  // Это приемлемо только для разработки
  const devCSP = ["default-src 'self'", "script-src 'self' 'unsafe-inline' 'wasm-unsafe-eval' blob:", "worker-src 'self' blob:", "style-src 'self' 'unsafe-inline' https://fonts.googleapis.com", "font-src 'self' data: https://fonts.gstatic.com", "img-src 'self' data: https:", "connect-src 'self' http://localhost:8080 https://nominatim.openstreetmap.org ws: wss:", "frame-ancestors 'none'", "base-uri 'self'", "form-action 'self'"].join('; ');
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
  plugins: [react(), tailwindcss(), cspPlugin(), visualizer({
    filename: 'dist/stats.html',
    open: true,
    gzipSize: true,
    brotliSize: true
  })],
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