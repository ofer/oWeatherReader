import { defineConfig, devices } from '@playwright/test';

/**
 * Playwright config for oWeatherReader end-to-end tests.
 *
 * Target deployment is overridable via E2E_BASE_URL:
 *   E2E_BASE_URL=http://localhost:6656/weather/ npx playwright test
 *
 * Defaults to the live deployment.
 */
const BASE_URL = process.env['E2E_BASE_URL'] ?? 'https://israeli.achler.org:4443/weather/';

export default defineConfig({
  testDir: './e2e',
  fullyParallel: true,
  forbidOnly: !!process.env['CI'],
  retries: process.env['CI'] ? 2 : 0,
  workers: process.env['CI'] ? 1 : undefined,
  reporter: [['list'], ['html', { open: 'never' }]],
  use: {
    baseURL: BASE_URL,
    trace: 'on-first-retry',
    screenshot: 'only-on-failure',
    video: 'retain-on-failure',
    // Deployment uses a self-signed / non-public CA cert chain.
    ignoreHTTPSErrors: true,
  },
  projects: [
    {
      name: 'chromium',
      use: { ...devices['Desktop Chrome'] },
    },
  ],
});
