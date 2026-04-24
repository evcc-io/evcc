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
    await expect(page.getByTestId("savings-button")).toContainText("60%");

    // last 365 days
    await page.getByTestId("savings-button").click();
    const savingsModal = page.getByTestId("savings-modal");
    await expect(savingsModal).toBeVisible();
    await page
      .getByTestId("savings-period-select")
      .getByRole("combobox")
      .selectOption("last 365 days");
    await page.getByRole("button", { name: "Close" }).click();
    await expect(savingsModal).not.toBeVisible();
    await expect(page.getByTestId("savings-button")).toContainText("30%");

    // all time
    await page.getByTestId("savings-button").click();
    await expect(savingsModal).toBeVisible();
    await page.getByTestId("savings-period-select").getByRole("combobox").selectOption("all time");
    await page.getByRole("button", { name: "Close" }).click();
    await expect(savingsModal).not.toBeVisible();
    await expect(page.getByTestId("savings-button")).toContainText("50%");
  });
});

test.describe("statistics values", async () => {
  test("last 30 days", async ({ page }) => {
    await page.goto("/");
    await page.getByTestId("savings-button").click();
    await expect(page.getByTestId("savings-modal")).toBeVisible();

    await page
      .getByTestId("savings-period-select")
      .getByRole("combobox")
      .selectOption("last 30 days");

    await expect(page.getByTestId("savings-tile-solar")).toContainText("60.0%");
    await expect(page.getByTestId("savings-tile-solar")).toContainText("30 kWh solar");
    await expect(page.getByTestId("savings-tile-solar")).toContainText("20 kWh grid");

    await expect(page.getByTestId("savings-tile-price")).toContainText("18.0ct./kWh");
    await expect(page.getByTestId("savings-tile-price")).toContainText("6 Fr. saved");

    await expect(page.getByTestId("savings-tile-co2")).toContainText("8g/kWh");
    await expect(page.getByTestId("savings-tile-co2")).toContainText("17 kg saved");
  });

  test("last 365 days", async ({ page }) => {
    await page.goto("/");
    await page.getByTestId("savings-button").click();
    await expect(page.getByTestId("savings-modal")).toBeVisible();
    await page
      .getByTestId("savings-period-select")
      .getByRole("combobox")
      .selectOption("last 365 days");

    await expect(page.getByTestId("savings-tile-solar")).toContainText("30.0%");
    await expect(page.getByTestId("savings-tile-solar")).toContainText("30 kWh solar");
    await expect(page.getByTestId("savings-tile-solar")).toContainText("70 kWh grid");

    await expect(page.getByTestId("savings-tile-price")).toContainText("24.0ct./kWh");
    await expect(page.getByTestId("savings-tile-price")).toContainText("6 Fr. saved");

    await expect(page.getByTestId("savings-tile-co2")).toContainText("14g/kWh");
    await expect(page.getByTestId("savings-tile-co2")).toContainText("33 kg saved");
  });

  test("reference data", async ({ page }) => {
    await page.goto("/");
    await page.getByTestId("savings-button").click();
    await expect(page.getByTestId("savings-modal")).toBeVisible();

    await expect(page.getByTestId("savings-reference")).toContainText("Reference data");
    await expect(page.getByTestId("savings-reference")).toContainText("30.0 ct./kWh");
    await expect(page.getByTestId("savings-reference")).toContainText("⌀ 344 g/kWh");
    await expect(page.getByTestId("savings-reference")).toContainText("Germany");

    await expect(page.getByTestId("savings-tile-co2")).toContainText("17 kg saved");

    await page
      .getByTestId("savings-region-select")
      .getByRole("combobox")
      .selectOption("Switzerland");

    await expect(page.getByTestId("savings-reference")).toContainText("⌀ 37 g/kWh");
    await expect(page.getByTestId("savings-reference")).toContainText("Switzerland");

    await expect(page.getByTestId("savings-tile-co2")).toContainText("1 kg saved");
  });
});

test.describe("header savings", async () => {
  test("savings info in header and indicator persistence", async ({ page }) => {
    await page.goto("/");

    // savings button visible in header
    await expect(page.getByTestId("savings-button")).toBeVisible();
    // default indicator is solar
    await expect(page.getByTestId("savings-button")).toContainText("60%");

    // open modal, verify values still work
    await page.getByTestId("savings-button").click();
    const savingsModal = page.getByTestId("savings-modal");
    await expect(savingsModal).toBeVisible();

    await expect(page.getByTestId("savings-tile-solar")).toContainText("60.0%");
    await expect(page.getByTestId("savings-tile-price")).toContainText("18.0ct./kWh");
    await expect(page.getByTestId("savings-tile-co2")).toContainText("8g/kWh");

    // switch indicator to "price"
    await page.getByTestId("savings-indicator-select").getByRole("combobox").selectOption("price");
    await page.getByRole("button", { name: "Close" }).click();
    await expect(savingsModal).not.toBeVisible();

    // verify button now shows price
    await expect(page.getByTestId("savings-button")).toContainText("18.0");

    // reload and verify persistence
    await page.reload();
    await expect(page.getByTestId("savings-button")).toContainText("18.0");
  });
});
