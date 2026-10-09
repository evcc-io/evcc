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
    // what reached home and loadpoint is priced: self-produced at feed-in 0.10, grid at ø 0.30
    const priced = (cost: number, pricedEnergy: number) => ({
      cost: expect.closeTo(cost, 6),
      pricedEnergy,
    });
    expect(data.flows).toEqual([
      { from: "pv", to: "home", energy: 1, ...priced(0.1, 1) },
      { from: "pv", to: "battery", energy: 1 },
      { from: "pv", to: "loadpoint", energy: 1, ...priced(0.1, 1) },
      { from: "pv", to: "export", energy: 1 },
      { from: "battery", to: "home", energy: 1, ...priced(0.1, 1) },
      { from: "grid", to: "loadpoint", energy: 2, ...priced(0.6, 2) },
    ]);
    // import 2 kWh × 0.30, export 1 kWh × 0.10
    expect(data.cost.import).toBeCloseTo(0.6, 6);
    expect(data.cost.export).toBeCloseTo(0.1, 6);
    expect(data.cost.avgGrid).toBeCloseTo(0.3, 6);
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
    await expect(chart.getByText("Charging")).toBeVisible();
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
    // every defined price is in the legend, the import price moves between 25 and 35 ct
    await expect(grid).toContainText("Import price");
    await expect(grid).toContainText("25.0 – 35.0 ct/kWh");
    await expect(grid).toContainText("Export price");
    await expect(grid).toContainText("10.0 ct/kWh");

    // pv 4 kWh vs forecast 3.2 kWh, self-consumption 1 - 1/4
    const production = page.getByTestId("energy-production");
    await expect(production).toContainText("4.0 kWh");
    await expect(production.getByText("Forecast", { exact: true })).toBeVisible();
    const selfConsumed = page.getByTestId("energy-stat-selfConsumed");
    await expect(selfConsumed).toContainText("Self-consumed");
    await expect(selfConsumed).toContainText("75%");
    await expect(selfConsumed).toContainText("25% exported");
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
    await expect(loadpoint.getByRole("heading", { name: "Carport 3.0 kWh" })).toBeVisible();

    // additional meter outside the balance
    const meters = page.getByTestId("energy-meters");
    await expect(meters).toContainText("Pool");
    await expect(meters).toContainText("300 Wh");

    // fixture has no consumer meters, the house consumption stands alone
    await expect(
      page.getByTestId("energy-consumers").getByRole("heading", { name: "Consumption 2.0 kWh" })
    ).toBeVisible();

    // savings 1.50 - 0.90
    await expect(page.getByTestId("energy-stat-savings")).toContainText("0.60 €");
    // co2 saved 2 kg - 0.8 kg
    await expect(page.getByTestId("energy-stat-co2")).toContainText("1 kg");
  });

  // plot area spans the chart width minus the 36px axes, 96 slots a day
  const slotX = (width: number, slot: number, left = 0) =>
    left + ((width - left - 36) * (slot + 0.5)) / 96;

  test("production tooltip with forecast", async ({ page }) => {
    await page.goto("/#/energy?year=2026&month=9&day=15");
    // 12:00: 2 kWh produced, 1.6 kWh forecast, as average power
    const chart = page.getByTestId("energy-production").getByTestId("group-chart-pv");
    const box = await chart.boundingBox();
    if (!box) throw new Error("chart not visible");
    await chart.hover({ position: { x: slotX(box.width, 48), y: box.height / 2 } });
    const tooltip = page.getByRole("table");
    await expect(tooltip).toContainText("12:00 – 12:15");
    await expect(tooltip.getByRole("row", { name: "Production" })).toContainText("8.0 kW");
    await expect(tooltip.getByRole("row", { name: "Forecast" })).toContainText("6.4 kW");
  });

  test("battery soc below the bars", async ({ page }) => {
    await page.goto("/#/energy?year=2026&month=9&day=15");
    // 12:00: charging 0.5 kWh at 50%, the soc axis takes 36px on the left
    const chart = page.getByTestId("energy-battery").getByTestId("group-chart-battery");
    const box = await chart.boundingBox();
    if (!box) throw new Error("chart not visible");
    const x = slotX(box.width, 48, 36);
    const tooltip = page.getByRole("table");
    // the bars and the soc panel share one tooltip
    for (const y of [box.height / 3, box.height - 40]) {
      await chart.hover({ position: { x, y } });
      await expect(tooltip.getByRole("row", { name: "charged 2.0 kW", exact: true })).toBeVisible();
      await expect(tooltip.getByRole("row", { name: "charge 50%", exact: true })).toBeVisible();
      await page.mouse.move(0, 0);
    }
  });

  test("grid tooltip with cost and effective price", async ({ page }) => {
    await page.goto("/#/energy?period=month&year=2026&month=9");
    // 15th: 2 kWh imported at 0.25 and 0.35, 1 kWh exported at 0.10
    const chart = page.getByTestId("energy-grid").getByTestId("group-chart-grid");
    const box = await chart.boundingBox();
    if (!box) throw new Error("chart not visible");
    // plot area spans the chart width minus the 36px axis on the right
    await chart.hover({ position: { x: ((box.width - 36) * 14.5) / 30, y: box.height / 2 } });
    const tooltip = page.getByRole("table");
    const amount = tooltip.getByRole("row", { name: "Amount" });
    await expect(amount).toContainText("0.60 €");
    await expect(amount).toContainText("0.10 €");
    const price = tooltip.getByRole("row", { name: "Price" });
    await expect(price).toContainText("30.0 ct/kWh");
    await expect(price).toContainText("10.0 ct/kWh");
  });

  test("flow chart loadpoint label", async ({ page }) => {
    const card = page.getByTestId("energy-flow");
    const chart = card.getByTestId("flow-chart");

    // charger only
    await page.goto("/#/energy?year=2026&month=9&day=15");
    await card.getByTitle("Energy flow").click();
    await expect(chart.getByText("Charging")).toBeVisible();
    await expect(chart.getByText("Heating")).toHaveCount(0);

    // heater only
    await page.goto("/#/energy?year=2026&month=9&day=16");
    await expect(chart.getByText("Heating")).toBeVisible();
    await expect(chart.getByText("Charging")).toHaveCount(0);

    // both
    await page.goto("/#/energy?period=month&year=2026&month=9");
    await expect(chart.getByText("Charging &")).toBeVisible();
    await expect(chart.getByText("Heating")).toBeVisible();
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

  test.describe("sources", () => {
    test("shares and cost", async ({ page }) => {
      // carport 3 kWh: 1 solar at feed-in 0.10, 2 grid at ø 0.30
      await page.goto("/#/energy?year=2026&month=9&day=15");
      const sources = page.getByTestId("energy-loadpoint").getByRole("button", { name: "Sources" });
      await expect(sources).toContainText("33% Solar");
      await expect(sources).toContainText("0% Battery, 67% Grid");

      await sources.click();
      const modal = page.getByTestId("energy-sources-modal");
      await expectModalVisible(modal);
      await expect(modal.getByRole("heading", { name: "Carport" })).toBeVisible();
      await expect(modal).toContainText("Sep 15, 2026");
      const solar = modal.getByRole("row", { name: "Solar" });
      await expect(solar).toContainText("33%");
      await expect(solar).toContainText("1.0 kWh");
      await expect(solar).toContainText("0.10 €");
      await expect(solar).toContainText("10.0 ct");
      const grid = modal.getByRole("row", { name: "Grid" });
      await expect(grid).toContainText("67%");
      await expect(grid).toContainText("2.0 kWh");
      await expect(grid).toContainText("0.60 €");
      await expect(grid).toContainText("30.0 ct");
      const total = modal.getByRole("row", { name: "Total" });
      await expect(total).toContainText("3.0 kWh");
      await expect(total).toContainText("0.70 €");
      await expect(modal.getByText("valued at feed-in price")).toBeVisible();
    });

    test("no price data", async ({ page }) => {
      // solar only, neither battery nor grid has data that day
      await page.goto("/#/energy?year=2026&month=9&day=17");
      const loadpoint = page.getByTestId("energy-loadpoint");
      await expect(loadpoint.getByRole("heading", { name: "Carport 1.0 kWh" })).toBeVisible();
      const sources = loadpoint.getByRole("button", { name: "Sources" });
      await expect(sources).toContainText("100% Solar");
      await expect(sources).not.toContainText("battery");
      await expect(sources).not.toContainText("grid");

      await sources.click();
      const modal = page.getByTestId("energy-sources-modal");
      await expectModalVisible(modal);
      await expect(modal.getByRole("row", { name: "Solar" })).toContainText("1.0 kWh");
      await expect(modal.getByRole("row", { name: "Battery" })).toHaveCount(0);
      await expect(modal.getByRole("row", { name: "Grid" })).toHaveCount(0);
      await expect(modal.getByRole("columnheader", { name: "Cost" })).toHaveCount(0);
      await expect(modal.getByText("valued at feed-in price")).toHaveCount(0);
    });

    test("battery leads without solar", async ({ page }) => {
      await page.goto("/#/energy?year=2026&month=9&day=18");
      const sources = page.getByTestId("energy-loadpoint").getByRole("button", { name: "Sources" });
      await expect(sources).toContainText("100% Battery");
      await expect(sources).not.toContainText("solar");
    });

    test("grid only", async ({ page }) => {
      await page.goto("/#/energy?year=2026&month=9&day=16");
      const loadpoint = page.getByTestId("energy-loadpoint");
      await expect(loadpoint.getByRole("heading", { name: "Heat pump 1.0 kWh" })).toBeVisible();
      await expect(loadpoint.getByRole("button", { name: "Sources" })).toHaveCount(0);
    });
  });

  test("empty state", async ({ page }) => {
    await page.goto("/#/energy?year=2026&month=1&day=1");
    await expect(page.getByText("No energy data available")).toBeVisible();
  });
});
