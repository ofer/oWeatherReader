import { test, expect, Page, Route } from '@playwright/test';

/**
 * End-to-end tests for the History page.
 *
 * Two suites:
 *   1. "live deployment"   — hits the real backend; tolerates empty data.
 *   2. "mocked backend"    — stubs /models and /reports/history so we can
 *      assert the populated-state UI deterministically.
 *
 * Run a single suite via grep:
 *   npx playwright test -g "mocked backend"
 */

const HISTORY_PATH = 'history';

// --- helpers ---------------------------------------------------------------

async function gotoHistory(page: Page): Promise<void> {
  await page.goto(HISTORY_PATH, { waitUntil: 'networkidle' });
  // History container is the page root marker.
  await expect(page.locator('.history-container')).toBeVisible();
}

async function openMatSelect(page: Page, label: string): Promise<void> {
  const field = page.locator('mat-form-field', { hasText: label });
  await field.locator('mat-select').click();
  await expect(page.locator('mat-option').first()).toBeVisible();
}

function makeAggregate(daysAgo: number, model: string, modelName: string) {
  const d = new Date();
  d.setUTCDate(d.getUTCDate() - daysAgo);
  const date = d.toISOString().slice(0, 10);
  // Vary values so charts/comfort ratings produce different categories.
  const base = 60 + (daysAgo % 30);
  return {
    date,
    model,
    modelName,
    avgTemp: base + 5,
    highTemp: base + 12,
    lowTemp: base - 2,
    avgHumidity: 40 + (daysAgo % 40),
    highHumidity: 55 + (daysAgo % 30),
    lowHumidity: 30 + (daysAgo % 20),
  };
}

// =========================================================================
// 1. Live deployment smoke tests
// =========================================================================

test.describe('History page — live deployment', () => {
  test('loads, shows controls and empty-state when no sensors configured', async ({ page }) => {
    await gotoHistory(page);

    // Time Range selector is always rendered.
    await expect(page.locator('mat-form-field', { hasText: 'Time Range' })).toBeVisible();

    // Without indoor/outdoor set in localStorage, the component shows the empty card.
    await expect(page.locator('.empty-card')).toBeVisible();
    await expect(page.locator('.empty-card')).toContainText(/No sensors selected/i);
  });

  test('Time Range select offers the documented presets', async ({ page }) => {
    await gotoHistory(page);
    await openMatSelect(page, 'Time Range');

    // 6-month option was removed; now only 7, 30, 90, 365 days.
    for (const label of ['7 Days', '30 Days', '90 Days', '1 Year']) {
      await expect(page.locator('mat-option', { hasText: label })).toBeVisible();
    }
    // Verify 6 Months is NOT present.
    await expect(page.locator('mat-option', { hasText: '6 Months' })).toHaveCount(0);
  });

  test('renders either populated content or the empty-state card, never both', async ({ page }) => {
    await gotoHistory(page);
    // Wait until the indeterminate progress bar disappears (or never appeared).
    await page
      .locator('mat-progress-bar')
      .waitFor({ state: 'detached', timeout: 15_000 })
      .catch(() => { /* ok if it was never there */ });

    const hasCharts = await page.locator('.chart-card').count();
    const hasEmpty = await page.locator('.empty-card').count();

    // Exactly one of: charts visible, empty card visible, OR neither (data
    // came through but charts haven't been asserted on). The XOR sanity
    // check is: we must NOT render the empty-card AND the chart-card together.
    expect(hasCharts === 0 || hasEmpty === 0).toBe(true);
  });
});

// =========================================================================
// 2. Mocked backend — deterministic populated-state assertions
// =========================================================================

test.describe('History page — mocked backend', () => {
  const INDOOR_MODEL = 'LaCrosse-TX141THBv2';
  const OUTDOOR_MODEL = 'Bresser-3CH';
  const MODELS_PAYLOAD = [
    { DeviceModel: INDOOR_MODEL, ReportCount: 1000, Name: 'Indoor Sensor' },
    { DeviceModel: OUTDOOR_MODEL, ReportCount: 50, Name: 'Outdoor Sensor' },
  ];

  test.beforeEach(async ({ page }) => {
    // The Angular app is served at /weather/ with base-href /weather/, so
    // its relative './models' resolves to /weather/models. Match by suffix
    // to be deployment-agnostic.
    await page.route(/\/models($|\?)/, async (route: Route) => {
      await route.fulfill({
        status: 200,
        contentType: 'application/json',
        body: JSON.stringify(MODELS_PAYLOAD),
      });
    });

    await page.route(/\/reports\/history(\?|$)/, async (route: Route) => {
      const url = new URL(route.request().url());
      const days = Number(url.searchParams.get('days') ?? '30');
      const modelsParam = url.searchParams.get('models');
      const selectedModels = modelsParam ? modelsParam.split(',').map(s => s.trim()) : [];

      const payload = [];
      // Component now calls /reports/history once per sensor (indoor + outdoor).
      // Mock returns data only for the models requested.
      for (const model of selectedModels) {
        for (let i = 0; i < Math.min(days, 14); i++) {
          const isIndoor = model === INDOOR_MODEL;
          payload.push(makeAggregate(i, model, isIndoor ? 'Indoor Sensor' : 'Outdoor Sensor'));
        }
      }
      await route.fulfill({
        status: 200,
        contentType: 'application/json',
        body: JSON.stringify(payload),
      });
    });
  });

  test('renders indoor and outdoor sections with separate data tables', async ({ page }) => {
    // Set indoor and outdoor sensors in localStorage so the component loads data.
    await page.evaluate(() => {
      localStorage.setItem('indoorDeviceModel', 'LaCrosse-TX141THBv2');
      localStorage.setItem('outdoorDeviceModel', 'Bresser-3CH');
    });

    await gotoHistory(page);

    // Record count badge confirms data loaded.
    const recordCount = page.locator('.record-count');
    await expect(recordCount).toBeVisible();
    await expect(recordCount).toContainText(/\d+ daily records/);

    // Both indoor and outdoor section headers must appear.
    await expect(page.locator('mat-card-title', { hasText: 'Indoor Sensor' })).toBeVisible();
    await expect(page.locator('mat-card-title', { hasText: 'Outdoor Sensor' })).toBeVisible();

    // Charts render (shared chart showing both sensors).
    const chartCards = page.locator('.chart-card');
    await expect(chartCards).toHaveCount(4); // 2 for indoor, 2 for outdoor

    // Data tables render for each sensor.
    const tableCards = page.locator('.table-card');
    await expect(tableCards).toHaveCount(2);

    // Table headers come straight from displayedColumns in the component.
    for (const table of await tableCards.all()) {
      const headerRow = table.locator('thead');
      for (const header of ['Date', 'Device', 'Avg Temp', 'High', 'Low', 'Avg Humidity', 'Comfort']) {
        await expect(headerRow).toContainText(header);
      }
    }

    // Indoor table has Indoor Sensor rows; outdoor has Outdoor Sensor rows.
    const indoorTable = page.locator('.table-card', { hasText: 'Indoor Sensor' }).first();
    const indoorRows = indoorTable.locator('tbody tr');
    expect(await indoorRows.count()).toBeGreaterThan(0);
    for (let i = 0; i < await indoorRows.count(); i++) {
      await expect(indoorRows.nth(i)).toContainText('Indoor Sensor');
    }

    const outdoorTable = page.locator('.table-card', { hasText: 'Outdoor Sensor' }).first();
    const outdoorRows = outdoorTable.locator('tbody tr');
    expect(await outdoorRows.count()).toBeGreaterThan(0);
    for (let i = 0; i < await outdoorRows.count(); i++) {
      await expect(outdoorRows.nth(i)).toContainText('Outdoor Sensor');
    }
  });

  test('changing the Time Range triggers a new request with the chosen days', async ({ page }) => {
    await page.evaluate(() => {
      localStorage.setItem('indoorDeviceModel', 'LaCrosse-TX141THBv2');
      localStorage.setItem('outdoorDeviceModel', 'Bresser-3CH');
    });

    await gotoHistory(page);
    await expect(page.locator('.record-count')).toBeVisible();

    const requestPromise = page.waitForRequest((req) =>
      /\/reports\/history\?days=7(\D|$)/.test(req.url()),
    );

    await openMatSelect(page, 'Time Range');
    await page.locator('mat-option', { hasText: '7 Days' }).click();

    const req = await requestPromise;
    expect(req.url()).toMatch(/days=7/);

    // UI should not be stuck in the loading state after the new fetch resolves.
    await expect(page.locator('mat-progress-bar')).toHaveCount(0);
    await expect(page.locator('.record-count')).toBeVisible();
  });

  test('falls back to the empty-state card when no sensors are configured', async ({ page }) => {
    // Clear localStorage so the component sees no sensors.
    await page.evaluate(() => {
      localStorage.removeItem('indoorDeviceModel');
      localStorage.removeItem('outdoorDeviceModel');
    });

    // Also override /models to return empty so no data comes through.
    await page.route(/\/models($|\?)/, async (route) => {
      await route.fulfill({
        status: 200,
        contentType: 'application/json',
        body: '[]',
      });
    });

    await gotoHistory(page);

    await expect(page.locator('.empty-card')).toBeVisible();
    await expect(page.locator('.empty-card')).toContainText(/No sensors selected/i);

    // None of the populated-state cards should render.
    await expect(page.locator('.comfort-card')).toHaveCount(0);
    await expect(page.locator('.chart-card')).toHaveCount(0);
    await expect(page.locator('.table-card')).toHaveCount(0);
  });

  test('handles a backend error on /reports/history without freezing the loading bar', async ({ page }) => {
    await page.evaluate(() => {
      localStorage.setItem('indoorDeviceModel', 'LaCrosse-TX141THBv2');
      localStorage.setItem('outdoorDeviceModel', 'Bresser-3CH');
    });

    // Override to return 500 for both indoor and outdoor requests.
    await page.route(/\/reports\/history(\?|$)/, async (route) => {
      await route.fulfill({
        status: 500,
        contentType: 'application/json',
        body: JSON.stringify({ error: 'Failed to retrieve daily aggregates' }),
      });
    });

    await gotoHistory(page);

    // The component clears `loading` in the error callback, so the progress
    // bar must not remain in the DOM.
    await expect(page.locator('mat-progress-bar')).toHaveCount(0, { timeout: 10_000 });

    // With zero aggregates we expect no populated cards and no record count.
    await expect(page.locator('.record-count')).toHaveCount(0);
    await expect(page.locator('.chart-card')).toHaveCount(0);
    await expect(page.locator('.table-card')).toHaveCount(0);
  });
});
