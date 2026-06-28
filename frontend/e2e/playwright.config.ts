import { defineConfig, devices } from '@playwright/test';

// ═══════════════════════════════════════════════════════════════════════════
// CCTV Health Monitor — Playwright Configuration
// P1-QA.1: Parallel E2E + P1-QA.3: Accessibility Testing in CI
// ═══════════════════════════════════════════════════════════════════════════

const CI = !!process.env.CI;

export default defineConfig({
  testDir: './',
  timeout: CI ? 60_000 : 30_000,
  fullyParallel: true,
  retries: CI ? 2 : 1,
  workers: CI ? 4 : 2,
  forbidOnly: CI,
  reporter: [
    ['html', { outputFolder: '../playwright-report', open: CI ? 'never' : 'on-failure' }],
    ['list'],
    ...(CI ? ['github'] as any[] : []),
  ],
  use: {
    baseURL: 'http://localhost:5173',
    trace: 'on-first-retry',
    screenshot: 'only-on-failure',
    video: CI ? 'on-first-retry' : 'off',
  },
  webServer: {
    command: 'npx vite --port 5173',
    port: 5173,
    reuseExistingServer: !CI,
    timeout: CI ? 60_000 : 30_000,
  },
  projects: [
    {
      name: 'chromium',
      use: { ...devices['Desktop Chrome'] },
    },
    {
      name: 'a11y',
      testDir: '../tests/a11y',
      testMatch: '*.spec.ts',
      use: {
        ...devices['Desktop Chrome'],
        viewport: { width: 1280, height: 720 },
      },
    },
    {
      name: 'visual',
      testDir: '../tests/visual',
      testMatch: 'visual-regression.spec.ts',
      use: {
        ...devices['Desktop Chrome'],
        viewport: { width: 1280, height: 720 },
      },
      dependencies: ['chromium'],
    },
  ],
});
