import { test, expect } from "@playwright/test";
import { start, stop, baseUrl } from "./evcc";
import { expectModalVisible, expectModalHidden } from "./utils";

test.use({ baseURL: baseUrl() });

const from = "2026-09-15T00:00:00+02:00";
const to = "2026-09-16T00:00:00+02:00";

test.beforeAll(async () => {
  await start(undefined, "energy.sql");
});
test.afterAll(async () => {
  await stop();
});

test.describe("api", () => {
  test("flows and costs", async ({ request }) => {
    const res = await request.get(`${baseUrl()}/api/history/flow`, { params: { from, to } });
    expect(res.ok()).toBeTruthy();

    const data = await res.json();
    expect(data.flows).toEqual([
      { from: "pv", to: "home", energy: 1 },
      { from: "pv", to: "battery", energy: 1 },
      { from: "pv", to: "loadpoint", energy: 1 },
      { from: "pv", to: "export", energy: 1 },
      { from: "battery", to: "home", energy: 1 },
      { from: "grid", to: "loadpoint", energy: 2 },
    ]);
    // import 2 kWh × 0.30, export 1 kWh × 0.10, consumption 5 kWh × ø 0.30
    expect(data.cost.import).toBeCloseTo(0.6, 6);
    expect(data.cost.export).toBeCloseTo(0.1, 6);
    expect(data.cost.avgGrid).toBeCloseTo(0.3, 6);
    expect(data.cost.baseline).toBeCloseTo(1.5, 6);
    expect(data.cost.consumption).toBeCloseTo(0.9, 6);
    expect(data.co2.baseline).toBeCloseTo(2, 6);
  });

  test("tariffs", async ({ request }) => {
    const res = await request.get("/api/history/tariffs", {
      params: { from: "2026-09-15T00:00:00+02:00", to: "2026-09-16T00:00:00+02:00" },
    });
    expect(res.ok()).toBeTruthy();
    const data = await res.json();
    expect(data).toHaveLength(4);
    expect(data[3].grid).toBeCloseTo(0.35, 6);
    expect(data[3].feedin).toBeCloseTo(0.1, 6);
  });

  test("empty period", async ({ request }) => {
    const res = await request.get(`${baseUrl()}/api/history/flow`, {
      params: { from: "2026-01-01T00:00:00+01:00", to: "2026-01-02T00:00:00+01:00" },
    });
    expect(res.ok()).toBeTruthy();
    const data = await res.json();
    expect(data.flows ?? []).toHaveLength(0);
    expect(data.cost).toBeUndefined();
  });
});

test.describe("page", () => {
  test("flow chart and stats", async ({ page }) => {
    await page.goto("/#/energy?year=2026&month=9&day=15");
    const card = page.getByTestId("energy-flow");
    await expect(card).toBeVisible();

    // stacked areas with legend by default, usage and sankey behind the toggle
    await expect(card.getByText("Grid export")).toBeVisible();
    await card.getByTitle("Usage").click();
    await expect(card.getByText("Consumption")).toBeVisible();
    await expect(card.getByText("Carport")).toBeVisible();
    await expect(card.getByText("Battery")).toBeVisible();
    await card.getByTitle("Energy flow").click();
    const chart = card.getByTestId("flow-chart");
    await expect(chart.getByText("Production")).toBeVisible();
    await expect(chart.getByText("Consumption")).toBeVisible();
    await expect(chart.getByText("Heating")).toBeVisible();
    await expect(chart.getByText("Grid export")).toBeVisible();

    // autarky 1 - 2/5, grid import 2 kWh and export 1 kWh
    await expect(page.getByTestId("energy-stat-autarky")).toContainText("60%");
    const grid = page.getByTestId("energy-grid");
    await expect(grid).toContainText("Import");
    await expect(grid).toContainText("2.0 kWh");
    await expect(grid).toContainText("Export");
    await expect(grid).toContainText("1.0 kWh");
    // priced: cost and revenue with the average price below
    const gridImport = page.getByTestId("energy-stat-gridImport");
    await expect(gridImport).toContainText("Grid cost");
    await expect(gridImport).toContainText("0.60 €");
    await expect(gridImport).toContainText("ø 30.0 ct/kWh");
    await expect(page.getByTestId("energy-stat-gridExport")).toContainText("0.10 €");
    // the price overlay is off by default and the switch is remembered
    const prices = page.getByRole("switch", { name: "show prices" });
    await expect(prices).not.toBeChecked();
    await prices.check();
    await page.reload();
    await expect(page.getByRole("switch", { name: "show prices" })).toBeChecked();

    // pv 4 kWh vs forecast 3.2 kWh, self-consumption 1 - 1/4
    const production = page.getByTestId("energy-production");
    await expect(production).toContainText("4.0 kWh");
    await expect(production.getByText("Forecast", { exact: true })).toBeVisible();
    await expect(page.getByTestId("energy-stat-produced")).toContainText("75% self-consumed");
    const forecast = page.getByTestId("energy-stat-forecast");
    await expect(forecast).toContainText("Forecast accuracy");
    await expect(forecast).toContainText("75%");
    await expect(forecast).toContainText("800 Wh below");

    // battery 1 kWh charged, 1 kWh discharged
    const battery = page.getByTestId("energy-battery");
    await expect(battery).toContainText("Charged");
    await expect(battery).toContainText("Discharged");

    // loadpoint card with its 3 kWh
    const loadpoint = page.getByTestId("energy-loadpoint");
    await expect(loadpoint.getByRole("heading", { name: "Carport" })).toBeVisible();
    await expect(loadpoint).toContainText("Charged");
    await expect(loadpoint).toContainText("3.0 kWh");

    // additional meter outside the balance
    const meters = page.getByTestId("energy-meters");
    await expect(meters).toContainText("Pool");
    await expect(meters).toContainText("300 Wh");

    // fixture has no consumer meters
    await expect(page.getByTestId("energy-consumers")).toHaveCount(0);

    // savings 1.50 - 0.90
    await expect(page.getByTestId("energy-stat-savings")).toContainText("0.60 €");
    // co2 saved 2 kg - 0.8 kg
    await expect(page.getByTestId("energy-stat-co2")).toContainText("1 kg");
  });

  test("bottom tab with experimental, sessions moves to more on phones", async ({ page }) => {
    await page.goto("/#/config");
    const experimentalEntry = page.getByTestId("generalconfig-experimental");
    await experimentalEntry.getByRole("button", { name: "edit" }).click();
    const experimentalModal = page.getByTestId("experimental-modal");
    await expectModalVisible(experimentalModal);
    await experimentalModal.getByLabel("Enable experimental features.").click();
    await experimentalModal.getByRole("button", { name: "Close" }).click();
    await expectModalHidden(experimentalModal);

    const bar = page.getByTestId("bottom-tab-bar");
    await expect(bar.getByRole("link", { name: "Sessions" })).toBeVisible();
    await bar.getByRole("link", { name: "Energy" }).click();
    await expect(page.getByRole("heading", { name: "Energy" })).toBeVisible();

    await page.setViewportSize({ width: 390, height: 800 });
    await expect(bar.getByRole("link", { name: "Sessions" })).toBeHidden();
    await bar.getByTestId("tab-more").click();
    await expect(bar.getByTestId("tab-more").getByRole("link", { name: "Sessions" })).toBeVisible();
  });

  test("empty state", async ({ page }) => {
    await page.goto("/#/energy?year=2026&month=1&day=1");
    await expect(page.getByText("No energy data available")).toBeVisible();
  });
});
