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

test("both grid entities count as one grid", async ({ page }) => {
  await page.goto("/#/energy?period=day&year=2026&month=4&day=11");

  // 8 slots of 0.5 / 0.1 kWh across both entities
  const grid = page.getByTestId("energy-grid");
  await expect(grid).toContainText("Import");
  await expect(grid).toContainText("4.0 kWh");
  // no tariff, no prices in the chart
  await expect(grid).not.toContainText("Import price");
  await expect(grid.getByTestId("energy-stat-gridImport")).toContainText("4.0 kWh");
  // neither production nor battery: no overview
  await expect(page.getByTestId("energy-summary")).toHaveCount(0);

  // the slot of the leftover entity reads like any other slot
  const chart = grid.getByTestId("group-chart-grid");
  const box = await chart.boundingBox();
  if (!box) throw new Error("chart not visible");
  // plot area spans the chart width minus 36px on either side
  await chart.hover({ position: { x: 36 + ((box.width - 72) * 48.5) / 96, y: box.height / 2 } });
  // the tooltip renders outside the card
  await expect(page.getByRole("table")).toContainText("12:00 – 12:15");
  await expect(page.getByRole("table")).toContainText("2.0 kW");
});
