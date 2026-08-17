import { defineConfig, devices } from '@playwright/test';

/**
 * Playwright configuration for Compliance Probe Web UI E2E tests.
 */
export default defineConfig({
  testDir: './test-e2e',
  timeout: 30000,
  expect: {
    timeout: 5000,
  },
  fullyParallel: false,
  workers: 1, // Run sequentially to avoid port/resource contention with binary servers
  reporter: [
    ['list'],
    ['html', { open: 'never', outputFolder: 'playwright-report' }],
  ],
  use: {
    headless: true,
    trace: 'on-first-retry',
    screenshot: 'only-on-failure',
    video: 'retain-on-failure',
  },
  projects: [
    {
      name: 'chromium',
      use: { ...devices['Desktop Chrome'] },
    },
  ],
});
