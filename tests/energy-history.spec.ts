import { test, expect, type Page, type Locator } from "@playwright/test";
import { start, stop, restart, baseUrl } from "./evcc";

test.use({ baseURL: baseUrl() });

const from = "2026-03-24T21:00:00+01:00";
const to = "2026-03-25T01:00:00+01:00";

test.beforeAll(async () => {
  await start(undefined, "energy-history.sql");
});
test.afterAll(async () => {
  await stop();
});

test.describe("api", () => {
  test("15-minute resolution", async ({ request }) => {
    const res = await request.get(`${baseUrl()}/api/history/energy`, {
      params: { from, to },
    });
    expect(res.ok()).toBeTruthy();

    const data = await res.json();
    expect(data).toHaveLength(2);

    const grid = data.find((s: { title: string }) => s.title === "grid");
    const home = data.find((s: { title: string }) => s.title === "home");
    expect(grid).toBeDefined();
    expect(home).toBeDefined();

    expect(grid.data).toHaveLength(6);
    expect(home.data).toHaveLength(6);

    expect(home.data[0].energy).toBeCloseTo(0.1, 4);
    expect(home.data[0].returnEnergy).toBeCloseTo(0, 4);
    expect(grid.data[0].energy).toBeCloseTo(0.5, 4);
    expect(grid.data[0].returnEnergy).toBeCloseTo(0.1, 4);
  });

  test("day aggregation", async ({ request }) => {
    const res = await request.get(`${baseUrl()}/api/history/energy`, {
      params: { from, to, aggregate: "day" },
    });
    expect(res.ok()).toBeTruthy();

    const data = await res.json();
    expect(data).toHaveLength(2);

    const grid = data.find((s: { title: string }) => s.title === "grid");
    const home = data.find((s: { title: string }) => s.title === "home");
    expect(grid).toBeDefined();
    expect(home).toBeDefined();

    expect(grid.data).toHaveLength(2);
    expect(home.data).toHaveLength(2);

    const [homeDay1, homeDay2] = home.data;
    const [gridDay1, gridDay2] = grid.data;

    // day 1 (2026-03-24): 4 slots
    expect(homeDay1.energy).toBeCloseTo(0.4, 4);
    expect(homeDay1.returnEnergy).toBeCloseTo(0, 4);
    expect(gridDay1.energy).toBeCloseTo(2.0, 4);
    expect(gridDay1.returnEnergy).toBeCloseTo(0.4, 4);

    // day 2 (2026-03-25): 2 slots
    expect(homeDay2.energy).toBeCloseTo(0.2, 4);
    expect(homeDay2.returnEnergy).toBeCloseTo(0, 4);
    expect(gridDay2.energy).toBeCloseTo(1.0, 4);
    expect(gridDay2.returnEnergy).toBeCloseTo(0.2, 4);
  });
});

async function gotoDay(page: Page, year: number, month: number, day: number): Promise<void> {
  await page.goto(`/#/history?period=day&year=${year}&month=${month}&day=${day}`);
  await expect(page.getByRole("heading", { name: /history/i }).first()).toBeVisible();
}

async function gotoMonth(page: Page, year: number, month: number): Promise<void> {
  await page.goto(`/#/history?period=month&year=${year}&month=${month}`);
  await expect(page.getByRole("heading", { name: /history/i }).first()).toBeVisible();
}

function chart(page: Page, group: string): Locator {
  return page.getByTestId(`group-chart-${group}`);
}

function section(page: Page, group: string): Locator {
  return page.getByTestId(`history-section-${group}`);
}

// Y-axis labels rendered by echarts: position=right uses text-anchor=start.
// DOM order: axis name first, then ticks bottom→top (min, ..., max).
async function yAxis(c: Locator): Promise<string[]> {
  const els = c.locator('svg text[text-anchor="start"]');
  await els.first().waitFor();
  return els.allTextContents();
}

test.describe("axis and units", () => {
  test("grid ±2 kW bidirectional, 1 decimal", async ({ page }) => {
    await gotoDay(page, 2026, 3, 24);
    expect(await yAxis(chart(page, "grid"))).toEqual(["kW", "-2.0", "-1.0", "0.0", "1.0", "2.0"]);
  });

  test("sub-1 kW switches to W, floored at 1000 W", async ({ page }) => {
    await gotoDay(page, 2026, 4, 1);
    expect(await yAxis(chart(page, "grid"))).toEqual(["W", "-1,000", "-500", "0", "500", "1,000"]);
  });

  test("import-only stays symmetric", async ({ page }) => {
    await gotoDay(page, 2026, 4, 2);
    expect(await yAxis(chart(page, "grid"))).toEqual(["kW", "-2.0", "-1.0", "0.0", "1.0", "2.0"]);
  });

  test("battery ±3 kW, 1 decimal", async ({ page }) => {
    await gotoDay(page, 2026, 4, 3);
    expect(await yAxis(chart(page, "battery"))).toEqual([
      "kW",
      "-3.0",
      "-1.5",
      "0.0",
      "1.5",
      "3.0",
    ]);
  });

  test("battery ±6 kW, integer labels", async ({ page }) => {
    await gotoDay(page, 2026, 4, 4);
    expect(await yAxis(chart(page, "battery"))).toEqual(["kW", "-6", "-3", "0", "3", "6"]);
  });

  test("stacked batteries: axis follows stacked sum, not per-entity max", async ({ page }) => {
    await gotoDay(page, 2026, 4, 8);
    expect(await yAxis(chart(page, "battery"))).toEqual([
      "kW",
      "-3.0",
      "-1.5",
      "0.0",
      "1.5",
      "3.0",
    ]);
  });

  test("unidirectional axis stays positive", async ({ page }) => {
    await gotoDay(page, 2026, 4, 5);
    // Unidirectional groups use echarts auto-scale (min: 0). Peak 1.6 kW → ticks 0..2.
    expect(await yAxis(chart(page, "pv"))).toEqual(["kW", "0.0", "0.5", "1.0", "1.5", "2.0"]);
  });

  test("all-zero data: W axis at 1000 W", async ({ page }) => {
    await gotoDay(page, 2026, 4, 6);
    await expect(section(page, "pv")).toBeVisible();
    expect(await yAxis(chart(page, "pv"))).toEqual(["W", "0", "250", "500", "750", "1,000"]);
  });

  test("stacked entities + overlay not clipped", async ({ page }) => {
    await gotoDay(page, 2026, 5, 2);
    // Stacked east+west peak = 1.8 kWh × 4 = 7.2 kW → niceCeil = 8.
    expect(await yAxis(chart(page, "pv"))).toEqual(["kW", "0", "2", "4", "6", "8"]);
  });

  test("month view uses kWh unit", async ({ page }) => {
    await gotoMonth(page, 2026, 6);
    expect(await yAxis(chart(page, "battery"))).toEqual([
      "kWh",
      "-2.0",
      "-1.0",
      "0.0",
      "1.0",
      "2.0",
    ]);
  });
});

test.describe("consumption breakdown", () => {
  // 2026-04-07: home = 1.0 kWh, Kitchen = 0.4 kWh, Office = 0.3 kWh,
  // virtual Others = home − meters = 0.3 kWh.
  test("home total, meter legend, virtual Others", async ({ page }) => {
    await gotoDay(page, 2026, 4, 7);
    const consumption = section(page, "consumer");
    await expect(consumption).toBeVisible();

    // Section total = home, not sum of meters.
    await expect(consumption.getByRole("heading")).toContainText("1.0 kWh");

    // Others (virtual) + explicit meters.
    await expect(consumption.getByRole("button", { name: "Others 300 Wh" })).toBeVisible();
    await expect(consumption.getByRole("button", { name: "Kitchen 400 Wh" })).toBeVisible();
    await expect(consumption.getByRole("button", { name: "Office 300 Wh" })).toBeVisible();
  });

  // 2026-03-24: home = 0.4 kWh, no meter entities with data.
  test("home without meters shows chart without legend", async ({ page }) => {
    await gotoDay(page, 2026, 3, 24);
    const consumption = section(page, "consumer");
    await expect(consumption).toBeVisible();

    await expect(consumption.getByRole("heading")).toContainText("0.4 kWh");
    // No explicit consumers, no entity legend.
    await expect(consumption.getByRole("button")).toHaveCount(0);
  });

  test("entity focus rescales axis and resets on unfocus", async ({ page }) => {
    await gotoDay(page, 2026, 4, 7);
    const consumption = section(page, "consumer");
    const meterChart = chart(page, "consumer");

    // Stacked total per slot 1.0 kW → echarts auto-axis 0..1.2 in 1-decimal.
    expect(await yAxis(meterChart)).toEqual(["kW", "0.0", "0.3", "0.6", "0.9", "1.2"]);

    // Focus Kitchen → peak 400 W → unit switches to W, floored at 1000 W.
    const kitchen = consumption.getByRole("button", { name: "Kitchen 400 Wh" });
    await kitchen.click();
    await expect.poll(() => yAxis(meterChart)).toEqual(["W", "0", "250", "500", "750", "1,000"]);

    // Unfocus → axis returns to kW range, not stuck on 1000 W cap.
    await kitchen.click();
    await expect.poll(() => yAxis(meterChart)).toEqual(["kW", "0.0", "0.3", "0.6", "0.9", "1.2"]);
  });
});

test.describe("additional meters", () => {
  // 2026-04-09: single ext meter "Submeter" = 1.2 kWh, no home data.
  test("standalone section without virtual Others", async ({ page }) => {
    await gotoDay(page, 2026, 4, 9);
    const additional = section(page, "meter");
    await expect(additional).toBeVisible();

    // No section total: additional meters can be import, export, or consumption.
    await expect(additional.getByRole("heading")).not.toContainText("kWh");

    // Explicit entity legend, no virtual "Others" (unlike the consumer group).
    await expect(additional.getByRole("button", { name: "Submeter 1.2 kWh" })).toBeVisible();
    await expect(additional.getByText("Others", { exact: true })).toBeHidden();
  });

  // 2026-04-10: same meter with export. Negatives flip the axis to symmetric.
  test("bidirectional axis when data contains exports", async ({ page }) => {
    await gotoDay(page, 2026, 4, 10);
    expect(await yAxis(chart(page, "meter"))).toEqual(["kW", "-2.0", "-1.0", "0.0", "1.0", "2.0"]);
  });
});

test.describe("reconnect", () => {
  test("history still shows content after backend restart", async ({ page }) => {
    await gotoDay(page, 2026, 3, 24);
    await expect(chart(page, "grid")).toBeVisible();

    await restart();

    await expect(chart(page, "grid")).toBeVisible();
  });
});
