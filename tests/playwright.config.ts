// ═══════════════════════════════════════════════════════════════════════════
// CCTV Health Monitor — Visual Regression Playwright Configuration
// P2-MED-17: CI gate with --forbid-only, visual regression baseline snapshots
//
// Использование:
//   cd frontend && NODE_PATH=./node_modules npx playwright test --config=../tests/playwright.config.ts
//   cd frontend && NODE_PATH=./node_modules CI=true npx playwright test --config=../tests/playwright.config.ts
//
// Альтернатива (через существующий конфиг):
//   cd frontend && npm run test:visual
//   cd frontend && npx playwright test --project=visual --forbid-only
// ═══════════════════════════════════════════════════════════════════════════

import { defineConfig, devices } from '@playwright/test';

const CI = !!process.env.CI;

export default defineConfig({
  // ── Test Discovery ─────────────────────────────────────────────────────
  testDir: './visual',
  testMatch: 'visual-regression.spec.ts',

  // ── Timeouts ──────────────────────────────────────────────────────────
  timeout: 30_000,
  expect: {
    timeout: 10_000,
    toHaveScreenshot: {
      maxDiffPixels: 100,
      maxDiffPixelRatio: 0.01,
      threshold: 0.2,
      animations: 'disabled',
      caret: 'hide',
    },
  },

  // ── Parallelism & Retries ──────────────────────────────────────────────
  fullyParallel: true,
  retries: CI ? 1 : 0,
  workers: CI ? 4 : 2,
  forbidOnly: CI, // CI gate — не пропускать test.only

  // ── Reporting ──────────────────────────────────────────────────────────
  reporter: [
    ['html', { outputFolder: '../playwright-report-visual', open: CI ? 'never' : 'on-failure' }],
    ['list'],
    ...(CI ? [['github']] : []),
  ],

  // ── Global Options ─────────────────────────────────────────────────────
  use: {
    baseURL: 'http://localhost:5173',
    trace: 'retain-on-failure',
    screenshot: 'only-on-failure',
    video: CI ? 'on-first-retry' : 'off',
    actionTimeout: 10_000,
    navigationTimeout: 15_000,
  },

  // ── Snapshot Path Template ─────────────────────────────────────────────
  // Сохраняет скриншоты в tests/visual/__screenshots__/<test-file>/<name>.png
  snapshotPathTemplate:
    '{testDir}/{testFileDir}/__screenshots__/{testFilePath}/{arg}{ext}',

  // ── Web Server ─────────────────────────────────────────────────────────
  webServer: {
    command: 'npx vite --port 5173',
    port: 5173,
    reuseExistingServer: !CI,
    timeout: CI ? 60_000 : 30_000,
    cwd: '../frontend',
  },

  // ── Projects ───────────────────────────────────────────────────────────
  projects: [
    {
      name: 'chromium',
      use: {
        ...devices['Desktop Chrome'],
        viewport: { width: 1280, height: 720 },
        colorScheme: 'light',
      },
    },
  ],
});
