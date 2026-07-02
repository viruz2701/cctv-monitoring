// ═══════════════════════════════════════════════════════════════════════════
// playwright.a11y.config.ts — Accessibility CI Gate Configuration
//
// UX-8.1: A11y Audit & CI Gate
//   - axe-core Playwright integration
//   - CI gate: violations → PR block
//
// Использование:
//   cd frontend && npx playwright test --config=../tests/a11y/playwright.a11y.config.ts
//   cd frontend && npm run test:a11y:ci
// ═══════════════════════════════════════════════════════════════════════════

import { defineConfig, devices } from '@playwright/test';

const CI = !!process.env.CI;

export default defineConfig({
  testDir: '.',
  testMatch: 'a11y-ci.spec.ts',
  timeout: CI ? 60_000 : 30_000,
  expect: {
    timeout: 10_000,
  },
  fullyParallel: true,
  retries: CI ? 1 : 0,
  workers: CI ? 2 : 1,
  forbidOnly: CI, // CI gate: не пропускать test.only
  reporter: [
    ['html', { outputFolder: '../../playwright-report-a11y', open: CI ? 'never' : 'on-failure' }],
    ['list'],
    ...(CI ? [['github']] : []),
  ],
  use: {
    baseURL: 'http://localhost:5173',
    trace: 'retain-on-failure',
    screenshot: 'only-on-failure',
    actionTimeout: 10_000,
    navigationTimeout: 15_000,
  },
  projects: [
    {
      name: 'a11y',
      use: {
        ...devices['Desktop Chrome'],
        viewport: { width: 1280, height: 720 },
        colorScheme: 'light',
      },
    },
  ],
  webServer: {
    command: 'npx vite --port 5173',
    port: 5173,
    reuseExistingServer: !CI,
    timeout: CI ? 60_000 : 30_000,
    cwd: '../../frontend',
  },
});
