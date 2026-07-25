import { test, expect } from "@playwright/test";
import { start, stop, baseUrl } from "./evcc";

test.use({ baseURL: baseUrl() });

// 2026-04-11: two grid entities from a swapped device ref, old "grid" with
// data 11:00-11:45, leftover "db:2" 12:00-12:45
test.beforeAll(async () => {
  await start("config-grid-only.evcc.yaml", "energy-history.sql");
});
test.afterAll(async () => {
  await stop();
});

test("tooltip shows one merged grid entity", async ({ page }) => {
  await page.goto("/#/history?period=day&year=2026&month=4&day=11");

  const chart = page.getByTestId("group-chart-grid");
  await expect(chart.locator("svg")).toBeVisible();

  // hover the center of the 12:00-12:15 slot (48 of 96, plot area spans
  // the chart width minus the 36px right margin)
  const box = await chart.boundingBox();
  if (!box) throw new Error("grid chart not visible");
  await chart.hover({ position: { x: ((box.width - 36) * 48.5) / 96, y: box.height / 2 } });

  const tooltip = chart.locator("table");
  await expect(tooltip).toBeVisible();
  // single unnamed row, no per-entity rows
  await expect(tooltip).toHaveText(
    ["12:00 – 12:15", "imported", "exported", "2.0 kW", "400 W"].join("")
  );
});
