import { test, expect, Page, Route } from '@playwright/test';

/**
 * Regression test for year mode — non-overlapping sensors displayed correctly.
 *
 * Scenario: outdoor sensor reports Jan–Jun, indoor sensor reports Apr–Dec.
 * In year mode the history page should show BOTH sensors' monthly data
 * in their respective sections, not blended together.
 *
 * Run: npx playwright test -g "year mode — non-overlapping sensors"
 */

const HISTORY_PATH = 'history';

// --- helpers ----------------------------------------------------------------

async function gotoHistory(page: Page): Promise<void> {
  await page.goto(HISTORY_PATH, { waitUntil: 'networkidle' });
  await expect(page.locator('.history-container')).toBeVisible();
}

async function selectTimeRange(page: Page, label: string): Promise<void> {
  const field = page.locator('mat-form-field', { hasText: 'Time Range' });
  await field.locator('mat-select').click();
  await page.locator('mat-option', { hasText: label }).click();
}

/**
 * Build a mock monthly aggregate.
 *
 * @param monthStr  e.g. "2025-01"
 * @param model     device_model string
 * @param modelName display name
 * @param temp      average temperature for that month
 */
function makeMonthly(monthStr: string, model: string, modelName: string, temp: number): Record<string, unknown> {
  return {
    date: monthStr,
    model,
    modelName,
    avgTemp: temp + 5,
    highTemp: temp + 12,
    lowTemp: temp - 2,
    avgHumidity: 50,
    avgHighHumidity: 60,
    avgLowHumidity: 40,
    highHumidity: 65,
    lowHumidity: 35,
  };
}

// =========================================================================
// Year mode — non-overlapping sensors
// =========================================================================

test.describe('History page — year mode, non-overlapping sensors', () => {
  const INDOOR_MODEL = 'LaCrosse-TX141THBv2';
  const OUTDOOR_MODEL = 'Bresser-3CH';

  test.beforeEach(async ({ page }) => {
    // Set indoor and outdoor sensors in localStorage so the component loads both.
    await page.evaluate(() => {
      localStorage.setItem('indoorDeviceModel', 'LaCrosse-TX141THBv2');
      localStorage.setItem('outdoorDeviceModel', 'Bresser-3CH');
    });

    // Mock the models endpoint.
    await page.route(/\/models($|\?)/, async (route: Route) => {
      await route.fulfill({
        status: 200,
        contentType: 'application/json',
        body: JSON.stringify([
          { DeviceModel: INDOOR_MODEL, ReportCount: 800, Name: 'Indoor Sensor' },
          { DeviceModel: OUTDOOR_MODEL, ReportCount: 600, Name: 'Outdoor Sensor' },
        ]),
      });
    });

    // Intercept /reports/history and return mock data where:
    //   - outdoor sensor reports Jan–Jun 2025
    //   - indoor sensor reports Apr–Dec 2025
    // This mirrors the real-world scenario the user reported:
    // "outdoor stops in march and indoor starts after that".
    await page.route(/\/reports\/history(\?|$)/, async (route: Route) => {
      const url = new URL(route.request().url());
      const days = Number(url.searchParams.get('days') ?? '365');
      const modelsParam = url.searchParams.get('models');
      const selectedModels = modelsParam ? modelsParam.split(',').map(s => s.trim()) : [];

      const payload: Record<string, unknown>[] = [];
      const months2025 = [
        '2025-01', '2025-02', '2025-03', '2025-04',
        '2025-05', '2025-06', '2025-07', '2025-08',
        '2025-09', '2025-10', '2025-11', '2025-12',
      ];

      // The component makes separate requests for indoor and outdoor models.
      // Return data per the models requested.
      for (const model of selectedModels) {
        const isIndoor = model === INDOOR_MODEL;
        if (isIndoor) {
          // Indoor: Apr–Dec (9 months)
          const indoorMonths = months2025.slice(3, 12);
          for (const m of indoorMonths) {
            payload.push(makeMonthly(m, model, 'Indoor Sensor', 70));
          }
        } else {
          // Outdoor: Jan–Jun (6 months, warmer temps)
          const outdoorMonths = months2025.slice(0, 6);
          for (const m of outdoorMonths) {
            const monthNum = parseInt(m.split('-')[1], 10);
            payload.push(makeMonthly(m, model, 'Outdoor Sensor', 60 + monthNum * 3));
          }
        }
      }

      await route.fulfill({
        status: 200,
        contentType: 'application/json',
        body: JSON.stringify(payload),
      });
    });
  });

  test('year mode shows separate monthly data in indoor and outdoor sections', async ({ page }) => {
    await gotoHistory(page);

    // Select "1 Year" — this triggers monthly aggregation.
    await selectTimeRange(page, '1 Year');

    // Wait for data to load.
    await expect(page.locator('.record-count')).toBeVisible({ timeout: 10_000 });

    // The record-count badge should reflect the combined monthly rows.
    // Outdoor (6 months) + Indoor (9 months) = 15 total monthly rows
    const recordCount = page.locator('.record-count');
    await expect(recordCount).toBeVisible();
    const countText = await recordCount.textContent();
    expect(countText).toMatch(/15 monthly records/);

    // Both indoor and outdoor section headers must appear.
    await expect(page.locator('mat-card-title', { hasText: 'Indoor Sensor' })).toBeVisible();
    await expect(page.locator('mat-card-title', { hasText: 'Outdoor Sensor' })).toBeVisible();

    // Each section has its own data table.
    const tableCards = page.locator('.table-card');
    await expect(tableCards).toHaveCount(2);

    // Indoor table has 9 monthly rows (Apr–Dec).
    const indoorTable = page.locator('.table-card', { hasText: 'Indoor Sensor' }).first();
    await expect(indoorTable).toBeVisible();
    const indoorRows = indoorTable.locator('tbody tr');
    expect(await indoorRows.count()).toBe(9);

    // Every indoor row shows "Indoor Sensor".
    for (let i = 0; i < 9; i++) {
      await expect(indoorRows.nth(i)).toContainText('Indoor Sensor');
    }

    // Outdoor table has 6 monthly rows (Jan–Jun).
    const outdoorTable = page.locator('.table-card', { hasText: 'Outdoor Sensor' }).first();
    await expect(outdoorTable).toBeVisible();
    const outdoorRows = outdoorTable.locator('tbody tr');
    expect(await outdoorRows.count()).toBe(6);

    // Every outdoor row shows "Outdoor Sensor".
    for (let i = 0; i < 6; i++) {
      await expect(outdoorRows.nth(i)).toContainText('Outdoor Sensor');
    }

    // Charts render with data (4 chart cards total: temp + humidity × 2 sections).
    const chartCards = page.locator('.chart-card');
    await expect(chartCards).toHaveCount(4);
  });

  test('year mode does NOT blend sensors — each section keeps its identity', async ({ page }) => {
    await gotoHistory(page);
    await selectTimeRange(page, '1 Year');

    // Indoor table should NOT contain any "Outdoor Sensor" text.
    const indoorTable = page.locator('.table-card', { hasText: 'Indoor Sensor' }).first();
    await expect(indoorTable.locator('tbody').locator('tr', { hasText: 'Outdoor Sensor' })).toHaveCount(0);

    // Outdoor table should NOT contain any "Indoor Sensor" text.
    const outdoorTable = page.locator('.table-card', { hasText: 'Outdoor Sensor' }).first();
    await expect(outdoorTable.locator('tbody').locator('tr', { hasText: 'Indoor Sensor' })).toHaveCount(0);

    // There must NOT be a row with "All Selected Sensors" — that's the old blended behavior.
    await expect(page.locator('tbody').locator('tr', { hasText: 'All Selected Sensors' })).toHaveCount(0);

    // Verify month ranges:
    // Indoor should have months Apr–Dec (2025-04 through 2025-12).
    const indoorTable2 = page.locator('.table-card', { hasText: 'Indoor Sensor' }).first();
    const indoorRows2 = indoorTable2.locator('tbody tr');
    // First indoor row should be April (index 3 of 12 months).
    await expect(indoorRows2.nth(0)).toContainText('2025-04');
    // Last indoor row should be December.
    await expect(indoorRows2.last()).toContainText('2025-12');

    // Outdoor should have months Jan–Jun (2025-01 through 2025-06).
    const outdoorTable2 = page.locator('.table-card', { hasText: 'Outdoor Sensor' }).first();
    const outdoorRows2 = outdoorTable2.locator('tbody tr');
    // First outdoor row should be January.
    await expect(outdoorRows2.nth(0)).toContainText('2025-01');
    // Last outdoor row should be June.
    await expect(outdoorRows2.last()).toContainText('2025-06');
  });

  test('30-day mode still works with daily aggregation', async ({ page }) => {
    await gotoHistory(page);

    // Select 30 days — should use daily aggregation.
    await selectTimeRange(page, '30 Days');

    await expect(page.locator('.record-count')).toBeVisible();

    const recordCount = page.locator('.record-count');
    const countText = await recordCount.textContent();
    expect(countText).toMatch(/\d+ daily records/);

    // Both indoor and outdoor sections should still appear.
    await expect(page.locator('mat-card-title', { hasText: 'Indoor Sensor' })).toBeVisible();
    await expect(page.locator('mat-card-title', { hasText: 'Outdoor Sensor' })).toBeVisible();

    // Each section has its own data table with indoor/outdoor rows.
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

  test('year mode indoor section shows Apr–Dec (not Jan–Mar)', async ({ page }) => {
    await gotoHistory(page);
    await selectTimeRange(page, '1 Year');

    const indoorTable = page.locator('.table-card', { hasText: 'Indoor Sensor' }).first();

    // No indoor rows should have months January through March.
    const janToMarRows = indoorTable.locator('tbody tr', {
      hasText: /^(2025-01|2025-02|2025-03)/
    });
    await expect(janToMarRows).toHaveCount(0);

    // All indoor rows should have months April through December.
    const aprToDecRows = indoorTable.locator('tbody tr', {
      hasText: /^(2025-04|2025-05|2025-06|2025-07|2025-08|2025-09|2025-10|2025-11|2025-12)/
    });
    expect(await aprToDecRows.count()).toBe(9);
  });

  test('year mode outdoor section shows Jan–Jun (not Jul–Dec)', async ({ page }) => {
    await gotoHistory(page);
    await selectTimeRange(page, '1 Year');

    const outdoorTable = page.locator('.table-card', { hasText: 'Outdoor Sensor' }).first();

    // No outdoor rows should have months July through December.
    const julToDecRows = outdoorTable.locator('tbody tr', {
      hasText: /^(2025-07|2025-08|2025-09|2025-10|2025-11|2025-12)/
    });
    await expect(julToDecRows).toHaveCount(0);

    // All outdoor rows should have months January through June.
    const janToJunRows = outdoorTable.locator('tbody tr', {
      hasText: /^(2025-01|2025-02|2025-03|2025-04|2025-05|2025-06)/
    });
    expect(await janToJunRows.count()).toBe(6);
  });
});
