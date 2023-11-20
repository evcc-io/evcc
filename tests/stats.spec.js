const { test, expect } = require("@playwright/test");
const { start, stop } = require("./evcc");

test.beforeAll(async () => {
  await start("stats.evcc.yaml", "stats.sql");
});
test.afterAll(async () => {
  await stop();
});

test.describe("footer", async () => {
  test("default 30d solar percentage in footer and change on period switch", async ({ page }) => {
    await page.goto("/");

    // last 30 days
    await expect(page.getByTestId("savings-button")).toContainText("60% solar energy");

    // last 365 days
    await page.getByTestId("savings-button").click();
    await page
      .getByTestId("savings-period-select")
      .getByRole("combobox")
      .selectOption("last 365 days");
    await page.getByRole("button", { name: "Close" }).click();
    await expect(page.getByTestId("savings-button")).toContainText("30% solar energy");

    // all time
    await page.getByTestId("savings-button").click();
    await page.getByTestId("savings-period-select").getByRole("combobox").selectOption("all time");
    await page.getByRole("button", { name: "Close" }).click();
    await expect(page.getByTestId("savings-button")).toContainText("50% solar energy");
  });
});

test.describe.skip("stats values", async () => {
  test("last 30 days", async ({ page }) => {
    await page.goto("/");
    await page.getByTestId("savings-button").click();
    await page
      .getByTestId("savings-period-select")
      .getByRole("combobox")
      .selectOption("last 30 days");

    const solar = await page.getByTestId("savings-tile-solar");
    expect(solar).toContainText("60.0%");
    expect(solar).toContainText("30 kWh solar");
    expect(solar).toContainText("20 kWh grid");

    const price = await page.getByTestId("savings-tile-price");
    expect(price).toContainText("18.0rp/kWh");
    expect(price).toContainText("6 CHF saved");

    const co2 = await page.getByTestId("savings-tile-co2");
    expect(co2).toContainText("8g/kWh");
    expect(co2).toContainText("19 kg saved");
  });

  test("last 365 days", async ({ page }) => {
    await page.goto("/");
    await page.getByTestId("savings-button").click();
    await page
      .getByTestId("savings-period-select")
      .getByRole("combobox")
      .selectOption("last 365 days");

    const solar = await page.getByTestId("savings-tile-solar");
    expect(solar).toContainText("30.0%");
    expect(solar).toContainText("30 kWh solar");
    expect(solar).toContainText("70 kWh grid");

    const price = await page.getByTestId("savings-tile-price");
    expect(price).toContainText("24.0rp/kWh");
    expect(price).toContainText("6 CHF saved");

    const co2 = await page.getByTestId("savings-tile-co2");
    expect(co2).toContainText("14g/kWh");
    expect(co2).toContainText("37 kg saved");
  });

  test("reference data", async ({ page }) => {
    await page.goto("/");
    await page.getByTestId("savings-button").click();

    const reference = await page.getByTestId("savings-reference");
    expect(reference).toContainText("Reference data:");
    expect(reference).toContainText("30.0 rp/kWh (grid)");
    expect(reference).toContainText("⌀ 385 g/kWh");
    expect(reference).toContainText("Germany");

    expect(await page.getByTestId("savings-tile-co2")).toContainText("19 kg saved");

    await page
      .getByTestId("savings-region-select")
      .getByRole("combobox")
      .selectOption("Switzerland");

    expect(reference).toContainText("⌀ 46 g/kWh");
    expect(reference).toContainText("Switzerland");

    expect(await page.getByTestId("savings-tile-co2")).toContainText("2 kg saved");
  });
});
