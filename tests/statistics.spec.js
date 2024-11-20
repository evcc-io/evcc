import { test, expect } from "@playwright/test";
import { start, stop, baseUrl } from "./evcc";

test.use({ baseURL: baseUrl() });

test.beforeAll(async () => {
  await start("statistics.evcc.yaml", "statistics.sql");
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
    const savingsModal = await page.getByTestId("savings-modal");
    await expect(savingsModal).toBeVisible();
    await page
      .getByTestId("savings-period-select")
      .getByRole("combobox")
      .selectOption("last 365 days");
    await page.getByRole("button", { name: "Close" }).click();
    await expect(savingsModal).not.toBeVisible();
    await expect(page.getByTestId("savings-button")).toContainText("30% solar energy");

    // all time
    await page.getByTestId("savings-button").click();
    await expect(savingsModal).toBeVisible();
    await page.getByTestId("savings-period-select").getByRole("combobox").selectOption("all time");
    await page.getByRole("button", { name: "Close" }).click();
    await expect(savingsModal).not.toBeVisible();
    await expect(page.getByTestId("savings-button")).toContainText("50% solar energy");
  });
});

test.describe("statistics values", async () => {
  test("last 30 days", async ({ page }) => {
    await page.goto("/");
    await page.getByRole("button", { name: "60% solar energy" }).click();
    await expect(page.getByTestId("savings-modal")).toBeVisible();

    await page
      .getByTestId("savings-period-select")
      .getByRole("combobox")
      .selectOption("last 30 days");

    await expect(page.getByTestId("savings-tile-solar")).toContainText("60.0%");
    await expect(page.getByTestId("savings-tile-solar")).toContainText("30 kWh solar");
    await expect(page.getByTestId("savings-tile-solar")).toContainText("20 kWh grid");

    await expect(page.getByTestId("savings-tile-price")).toContainText("18.0rp/kWh");
    await expect(page.getByTestId("savings-tile-price")).toContainText("6 CHF saved");

    await expect(page.getByTestId("savings-tile-co2")).toContainText("8g/kWh");
    await expect(page.getByTestId("savings-tile-co2")).toContainText("19 kg saved");
  });

  test("last 365 days", async ({ page }) => {
    await page.goto("/");
    await page.getByRole("button", { name: "60% solar energy" }).click();
    await expect(page.getByTestId("savings-modal")).toBeVisible();
    await page
      .getByTestId("savings-period-select")
      .getByRole("combobox")
      .selectOption("last 365 days");

    await expect(page.getByTestId("savings-tile-solar")).toContainText("30.0%");
    await expect(page.getByTestId("savings-tile-solar")).toContainText("30 kWh solar");
    await expect(page.getByTestId("savings-tile-solar")).toContainText("70 kWh grid");

    await expect(page.getByTestId("savings-tile-price")).toContainText("24.0rp/kWh");
    await expect(page.getByTestId("savings-tile-price")).toContainText("6 CHF saved");

    await expect(page.getByTestId("savings-tile-co2")).toContainText("14g/kWh");
    await expect(page.getByTestId("savings-tile-co2")).toContainText("37 kg saved");
  });

  test("reference data", async ({ page }) => {
    await page.goto("/");
    await page.getByRole("button", { name: "60% solar energy" }).click();
    await expect(page.getByTestId("savings-modal")).toBeVisible();

    await expect(page.getByTestId("savings-reference")).toContainText("Reference data:");
    await expect(page.getByTestId("savings-reference")).toContainText("30.0 rp/kWh (grid)");
    await expect(page.getByTestId("savings-reference")).toContainText("⌀ 385 g/kWh");
    await expect(page.getByTestId("savings-reference")).toContainText("Germany");

    await expect(page.getByTestId("savings-tile-co2")).toContainText("19 kg saved");

    await page
      .getByTestId("savings-region-select")
      .getByRole("combobox")
      .selectOption("Switzerland");

    await expect(page.getByTestId("savings-reference")).toContainText("⌀ 46 g/kWh");
    await expect(page.getByTestId("savings-reference")).toContainText("Switzerland");

    await expect(page.getByTestId("savings-tile-co2")).toContainText("2 kg saved");
  });
});
