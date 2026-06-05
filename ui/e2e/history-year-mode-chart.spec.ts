import { test, expect, Page, Route } from '@playwright/test';

/**
 * Regression test for the history page in YEAR mode (monthly aggregation).
 *
 * User-reported bug: "The history page in year mode does not show the outdoor
 * with the indoor — the outdoor stops in March and the indoor starts after
 * that."
 *
 * Root cause:
 *   - HistoryPageComponent builds a SINGLE `tempChartOption` / `humidityChartOption`
 *     that combines both sensors' series and uses a shared x-axis built from
 *     `[...indoor, ...outdoor].sort()`. Duplicated months produce split column
 *     indices in the ECharts category axis, so the two series get plotted at
 *     non-overlapping column positions even when their data overlap in time.
 *   - Both the "Indoor Temperature" and "Outdoor Temperature" chart cards bind
 *     to that same combined option, so each section's chart shows the blended
 *     (and broken) visualization instead of its own sensor.
 *
 * This spec mocks BOTH sensors with full-year monthly data and asserts:
 *   1. The Indoor Temperature card's chart contains the indoor series only,
 *      covering all 12 months with no nulls.
 *   2. The Outdoor Temperature card's chart contains the outdoor series only,
 *      covering all 12 months with no nulls.
 *   3. Each chart's x-axis has exactly 12 unique months (no duplicates).
 *
 * Run: npx playwright test history-year-mode-chart.spec.ts
 */

const HISTORY_PATH = 'history';
const INDOOR_MODEL = 'LaCrosse-TX141THBv2';
const OUTDOOR_MODEL = 'Bresser-3CH';
const INDOOR_NAME = 'Indoor Sensor';
const OUTDOOR_NAME = 'Outdoor Sensor';

const MONTHS_2025 = [
  '2025-01', '2025-02', '2025-03', '2025-04',
  '2025-05', '2025-06', '2025-07', '2025-08',
  '2025-09', '2025-10', '2025-11', '2025-12',
];

function makeMonthly(monthStr: string, model: string, modelName: string, temp: number) {
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

async function gotoHistory(page: Page) {
  await page.addInitScript(() => {
    localStorage.setItem('indoorDeviceModel', 'LaCrosse-TX141THBv2');
    localStorage.setItem('outdoorDeviceModel', 'Bresser-3CH');
  });
  await page.goto(HISTORY_PATH, { waitUntil: 'networkidle' });
  await expect(page.locator('.history-container')).toBeVisible();
}

async function selectTimeRange(page: Page, label: string) {
  const field = page.locator('mat-form-field', { hasText: 'Time Range' });
  await field.locator('mat-select').click();
  await page.locator('mat-option', { hasText: label }).click();
}

/**
 * Read a chart option field straight from the HistoryPageComponent instance
 * using Angular's dev-mode debug API (`window.ng.getComponent`).
 * Returns { xAxisData, series } in the same shape as ECharts options.
 */
async function extractChartOption(
  page: Page,
  field: 'indoorTempChartOption' | 'outdoorTempChartOption' | 'indoorHumidityChartOption' | 'outdoorHumidityChartOption',
): Promise<{ xAxisData: string[]; series: { name: string; data: (number | null)[] }[] }> {
  const host = page.locator('app-history-page').first();
  await expect(host).toBeVisible();

  return await host.evaluate((el, fieldName) => {
    const win = window as any;
    if (!win.ng || typeof win.ng.getComponent !== 'function') {
      throw new Error('Angular debug API (ng.getComponent) not available — is the app in dev mode?');
    }
    const cmp = win.ng.getComponent(el);
    if (!cmp) throw new Error('Could not locate HistoryPageComponent instance');
    const opt = cmp[fieldName];
    if (!opt || !opt.xAxis) {
      throw new Error(`Chart option "${fieldName}" not built yet: ${JSON.stringify(Object.keys(opt || {}))}`);
    }
    const xAxis = Array.isArray(opt.xAxis) ? opt.xAxis[0] : opt.xAxis;
    return {
      xAxisData: (xAxis?.data ?? []) as string[],
      series: (opt.series ?? []).map((s: any) => ({ name: s.name, data: s.data })),
    };
  }, field);
}

test.describe('History page — year mode chart per-section isolation', () => {
  test.beforeEach(async ({ page }) => {
    await page.route(/\/models($|\?)/, async (route: Route) => {
      await route.fulfill({
        status: 200,
        contentType: 'application/json',
        body: JSON.stringify([
          { DeviceModel: INDOOR_MODEL, ReportCount: 1200, Name: INDOOR_NAME },
          { DeviceModel: OUTDOOR_MODEL, ReportCount: 1200, Name: OUTDOOR_NAME },
        ]),
      });
    });

    // Both sensors have data for ALL 12 months — full overlap.
    await page.route(/\/reports\/history(\?|$)/, async (route: Route) => {
      const url = new URL(route.request().url());
      const modelsParam = url.searchParams.get('models') ?? '';
      const requested = modelsParam.split(',').map((s) => s.trim()).filter(Boolean);

      const payload: ReturnType<typeof makeMonthly>[] = [];
      for (const model of requested) {
        const isIndoor = model === INDOOR_MODEL;
        const name = isIndoor ? INDOOR_NAME : OUTDOOR_NAME;
        const base = isIndoor ? 70 : 60;
        for (const m of MONTHS_2025) {
          const monthNum = parseInt(m.split('-')[1], 10);
          payload.push(makeMonthly(m, model, name, base + monthNum));
        }
      }

      await route.fulfill({
        status: 200,
        contentType: 'application/json',
        body: JSON.stringify(payload),
      });
    });
  });

  test('Indoor Temperature chart shows ONLY indoor series across all 12 months', async ({ page }) => {
    await gotoHistory(page);
    await selectTimeRange(page, '1 Year');

    // Wait for record-count badge to appear so the charts have been built.
    await expect(page.locator('.record-count')).toBeVisible({ timeout: 10_000 });

    const indoor = await extractChartOption(page, 'indoorTempChartOption');

    // The indoor chart's x-axis must contain exactly the 12 unique months.
    expect(indoor.xAxisData).toEqual(MONTHS_2025);

    // Series must be indoor-only.
    const indoorSeriesNames = indoor.series.map((s) => s.name);
    expect(indoorSeriesNames).toEqual(expect.arrayContaining(['Indoor Avg']));
    expect(indoorSeriesNames).not.toEqual(expect.arrayContaining(['Outdoor Avg']));

    // Indoor Avg series must have a value for every month (no nulls, no holes).
    const indoorAvg = indoor.series.find((s) => s.name === 'Indoor Avg')!;
    expect(indoorAvg.data).toHaveLength(12);
    for (const v of indoorAvg.data) {
      expect(v).not.toBeNull();
      expect(typeof v).toBe('number');
    }
  });

  test('Outdoor Temperature chart shows ONLY outdoor series across all 12 months', async ({ page }) => {
    await gotoHistory(page);
    await selectTimeRange(page, '1 Year');
    await expect(page.locator('.record-count')).toBeVisible({ timeout: 10_000 });

    const outdoor = await extractChartOption(page, 'outdoorTempChartOption');

    expect(outdoor.xAxisData).toEqual(MONTHS_2025);

    const outdoorSeriesNames = outdoor.series.map((s) => s.name);
    expect(outdoorSeriesNames).toEqual(expect.arrayContaining(['Outdoor Avg']));
    expect(outdoorSeriesNames).not.toEqual(expect.arrayContaining(['Indoor Avg']));

    const outdoorAvg = outdoor.series.find((s) => s.name === 'Outdoor Avg')!;
    expect(outdoorAvg.data).toHaveLength(12);
    for (const v of outdoorAvg.data) {
      expect(v).not.toBeNull();
      expect(typeof v).toBe('number');
    }
  });

  test('chart x-axis has NO duplicated month entries (regression for shared-axis bug)', async ({ page }) => {
    await gotoHistory(page);
    await selectTimeRange(page, '1 Year');
    await expect(page.locator('.record-count')).toBeVisible({ timeout: 10_000 });

    const fields: Array<'indoorTempChartOption' | 'outdoorTempChartOption' | 'indoorHumidityChartOption' | 'outdoorHumidityChartOption'> = [
      'indoorTempChartOption', 'outdoorTempChartOption', 'indoorHumidityChartOption', 'outdoorHumidityChartOption',
    ];
    for (const field of fields) {
      const { xAxisData } = await extractChartOption(page, field);
      const unique = new Set(xAxisData);
      expect(
        unique.size,
        `${field} x-axis has duplicate categories: ${JSON.stringify(xAxisData)}`,
      ).toBe(xAxisData.length);
    }
  });

  // =========================================================================
  // Humidity chart — Avg High and Avg Low series
  // =========================================================================

  test('Indoor Humidity chart includes Avg High and Avg Low series', async ({ page }) => {
    await gotoHistory(page);
    await selectTimeRange(page, '1 Year');
    await expect(page.locator('.record-count')).toBeVisible({ timeout: 10_000 });

    const indoorHumidity = await extractChartOption(page, 'indoorHumidityChartOption');

    const seriesNames = indoorHumidity.series.map((s) => s.name);
    expect(seriesNames).toContain('Indoor Avg High');
    expect(seriesNames).toContain('Indoor Avg Low');
    expect(seriesNames).toContain('Indoor Avg');
    expect(seriesNames).toContain('Indoor High');
    expect(seriesNames).toContain('Indoor Low');

    // All 5 series must have exactly 12 data points
    for (const s of indoorHumidity.series) {
      expect(s.data).toHaveLength(12);
    }

    // Avg High and Avg Low must have numeric values (not null, not zero)
    const avgHigh = indoorHumidity.series.find((s) => s.name === 'Indoor Avg High')!;
    expect(avgHigh.data).not.toContain(null);
    for (const v of avgHigh.data) {
      expect(typeof v).toBe('number');
      expect(v).toBeGreaterThan(0);
    }

    const avgLow = indoorHumidity.series.find((s) => s.name === 'Indoor Avg Low')!;
    expect(avgLow.data).not.toContain(null);
    for (const v of avgLow.data) {
      expect(typeof v).toBe('number');
      expect(v).toBeGreaterThan(0);
    }
  });

  test('Outdoor Humidity chart includes Avg High and Avg Low series', async ({ page }) => {
    await gotoHistory(page);
    await selectTimeRange(page, '1 Year');
    await expect(page.locator('.record-count')).toBeVisible({ timeout: 10_000 });

    const outdoorHumidity = await extractChartOption(page, 'outdoorHumidityChartOption');

    const seriesNames = outdoorHumidity.series.map((s) => s.name);
    expect(seriesNames).toContain('Outdoor Avg High');
    expect(seriesNames).toContain('Outdoor Avg Low');
    expect(seriesNames).toContain('Outdoor Avg');
    expect(seriesNames).toContain('Outdoor High');
    expect(seriesNames).toContain('Outdoor Low');

    for (const s of outdoorHumidity.series) {
      expect(s.data).toHaveLength(12);
    }

    const avgHigh = outdoorHumidity.series.find((s) => s.name === 'Outdoor Avg High')!;
    expect(avgHigh.data).not.toContain(null);
    for (const v of avgHigh.data) {
      expect(typeof v).toBe('number');
      expect(v).toBeGreaterThan(0);
    }

    const avgLow = outdoorHumidity.series.find((s) => s.name === 'Outdoor Avg Low')!;
    expect(avgLow.data).not.toContain(null);
    for (const v of avgLow.data) {
      expect(typeof v).toBe('number');
      expect(v).toBeGreaterThan(0);
    }
  });

  test('Humidity chart series count is 5 (not 3) — regression for series addition', async ({ page }) => {
    await gotoHistory(page);
    await selectTimeRange(page, '1 Year');
    await expect(page.locator('.record-count')).toBeVisible({ timeout: 10_000 });

    const fields: Array<'indoorHumidityChartOption' | 'outdoorHumidityChartOption'> = [
      'indoorHumidityChartOption', 'outdoorHumidityChartOption',
    ];
    for (const field of fields) {
      const { series } = await extractChartOption(page, field);
      expect(series.length, `${field} should have 5 series`).toBe(5);
    }
  });
});
