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
  await page.goto(`/#/energy?period=day&year=${year}&month=${month}&day=${day}`);
  await expect(page.getByTestId("energy-flow")).toBeVisible();
}

async function gotoMonth(page: Page, year: number, month: number): Promise<void> {
  await page.goto(`/#/energy?period=month&year=${year}&month=${month}`);
  await expect(page.getByTestId("energy-flow")).toBeVisible();
}

const CARDS: Record<string, string> = {
  pv: "energy-production",
  battery: "energy-battery",
  consumer: "energy-consumers",
  meter: "energy-meters",
};

function card(page: Page, group: string): Locator {
  return page.getByTestId(CARDS[group] ?? "");
}

function chart(page: Page, group: string): Locator {
  return card(page, group).getByTestId(`group-chart-${group}`).first();
}

// Y-axis labels rendered by echarts: position=right uses text-anchor=start.
// DOM order: axis name first, then ticks bottom→top (min, ..., max).
async function yAxis(c: Locator): Promise<string[]> {
  const els = c.locator('svg text[text-anchor="start"]');
  await els.first().waitFor();
  return els.allTextContents();
}

test.describe("axis and units", () => {
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

  test("unidirectional axis stays positive", async ({ page }) => {
    await gotoDay(page, 2026, 4, 5);
    // Unidirectional groups use echarts auto-scale (min: 0). Peak 1.6 kW → ticks 0..2.
    expect(await yAxis(chart(page, "pv"))).toEqual(["kW", "0.0", "0.5", "1.0", "1.5", "2.0"]);
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
  test("home total, tiles, virtual Others", async ({ page }) => {
    await gotoDay(page, 2026, 4, 7);
    const consumption = card(page, "consumer");
    await expect(consumption).toBeVisible();

    // Card total = home, not sum of meters.
    await expect(consumption.getByRole("heading")).toContainText("1.0 kWh");

    // Others (virtual) + explicit meters, as tiles and legend rows.
    for (const name of ["Others 300 Wh", "Kitchen 400 Wh", "Office 300 Wh"]) {
      await expect(consumption.getByRole("button", { name })).toHaveCount(2);
    }
  });

  // 2026-03-24: home = 0.4 kWh, no meter entities with data.
  test("home without meters has no consumers card", async ({ page }) => {
    await gotoDay(page, 2026, 3, 24);
    await expect(card(page, "consumer")).toHaveCount(0);
  });

  // 2026-04-12: consumers are not a bidirectional group. Return energy is
  // ignored, not netted or split: home 1.0, Kitchen 0.4.
  test("return energy is ignored", async ({ page }) => {
    await gotoDay(page, 2026, 4, 12);
    const consumption = card(page, "consumer");
    await expect(consumption.getByRole("heading")).toContainText("1.0 kWh");
    await expect(consumption.getByRole("button", { name: "Kitchen 400 Wh" }).first()).toBeVisible();
    await expect(consumption.getByRole("button", { name: "Others 600 Wh" }).first()).toBeVisible();
  });

  test("tile opens the detail with its own axis, second click closes", async ({ page }) => {
    await gotoDay(page, 2026, 4, 7);
    const consumption = card(page, "consumer");
    const kitchen = consumption.getByRole("button", { name: "Kitchen 400 Wh" }).first();
    await kitchen.click();

    // Kitchen alone: peak 400 W → unit switches to W, floored at 1000 W.
    const detail = page.getByTestId("consumer-detail");
    await expect(detail).toBeVisible();
    await expect
      .poll(() => yAxis(chart(page, "consumer")))
      .toEqual(["W", "0", "250", "500", "750", "1,000"]);

    await kitchen.click();
    await expect(detail).toHaveCount(0);
  });
});

test.describe("battery card", () => {
  // 2026-07: Battery 6.0/3.0 kWh, Battery 2 balanced at 4.5/4.5 kWh. A net sum
  // would collapse the balanced battery to zero, so both directions are shown.
  test("month keeps charge and discharge apart per battery", async ({ page }) => {
    await gotoMonth(page, 2026, 7);
    // one card per battery
    const cards = page.getByTestId("energy-battery");
    await expect(cards).toHaveCount(2);
    await expect(cards.first().getByRole("heading", { name: "Battery" })).toBeVisible();
    await expect(cards.first()).toContainText("6.0 kWh");
    await expect(cards.first()).toContainText("3.0 kWh");
    await expect(cards.last().getByRole("heading", { name: "Battery 2" })).toBeVisible();
    await expect(cards.last()).toContainText("4.5 kWh");
  });
});

test.describe("additional meters", () => {
  // 2026-04-09: single ext meter "Submeter" = 1.2 kWh, no home data.
  test("standalone card without virtual Others", async ({ page }) => {
    await gotoDay(page, 2026, 4, 9);
    const additional = card(page, "meter");
    await expect(additional).toBeVisible();
    await expect(additional).toContainText("Submeter");
    await expect(additional).toContainText("1.2 kWh");
    await expect(additional.getByText("Others", { exact: true })).toBeHidden();
  });

  // 2026-04-10: same meter with export. Negatives flip the axis to symmetric.
  test("bidirectional axis when data contains exports", async ({ page }) => {
    await gotoDay(page, 2026, 4, 10);
    expect(await yAxis(chart(page, "meter"))).toEqual(["kW", "-2.0", "-1.0", "0.0", "1.0", "2.0"]);
  });

  // 2026-04-10: "Submeter" 2.0/0.4 kWh, both directions as stats.
  test("bidirectional meter shows both directions", async ({ page }) => {
    await gotoDay(page, 2026, 4, 10);
    const additional = card(page, "meter");
    await expect(additional).toContainText("Energy2.0 kWh");
    await expect(additional).toContainText("Energy (reverse)400 Wh");
  });

  // 2026-04-13: export-only meter. Return energy alone shows the reverse stat.
  test("export-only meter shows the reverse direction", async ({ page }) => {
    await gotoDay(page, 2026, 4, 13);
    const additional = card(page, "meter");
    await expect(additional).toContainText("Feed-in meter");
    await expect(additional).toContainText("Energy (reverse)1.2 kWh");
  });
});

test.describe("reconnect", () => {
  test("page still shows content after backend restart", async ({ page }) => {
    await gotoDay(page, 2026, 3, 24);
    await restart();
    await expect(page.getByTestId("energy-flow")).toBeVisible();
  });
});
