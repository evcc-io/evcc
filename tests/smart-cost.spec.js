import { test, expect } from "@playwright/test";
import { start, stop, baseUrl } from "./evcc";
import { startSimulator, stopSimulator, simulatorUrl, simulatorConfig } from "./simulator";

test.use({ baseURL: baseUrl() });

test.beforeAll(async () => {
  await start(simulatorConfig(), "password.sql");
  await startSimulator();
});
test.afterAll(async () => {
  await stopSimulator();
  await stop();
});

test.beforeEach(async ({ page }) => {
  await page.goto(simulatorUrl());
  await page.getByLabel("PV Power").fill("6000");
  await page.getByTestId("loadpoint0").getByLabel("Power").fill("6000");
  await page.getByTestId("loadpoint0").getByText("C (charging)").click();
  await page.getByTestId("loadpoint0").getByText("Enabled").check();
  await page.getByRole("button", { name: "Apply changes" }).click();
});

test.describe("smart cost limit", async () => {
  test("no limit, normal charging", async ({ page }) => {
    await page.goto("/");
    await expect(page.getByTestId("vehicle-status")).toHaveText("Charging…");
  });
  test("price below limit", async ({ page }) => {
    await page.goto("/");
    await page.getByTestId("loadpoint-settings-button").nth(1).click();
    await expect(page.getByTestId("loadpoint-settings-modal")).toBeVisible();
    await page
      .getByTestId("loadpoint-settings-modal")
      .getByLabel("Price limit")
      .selectOption("≤ 50.0 ct/kWh");
    await page.getByTestId("loadpoint-settings-modal").getByLabel("Close").click();
    await expect(page.getByTestId("loadpoint-settings-modal")).not.toBeVisible();
    await expect(page.getByTestId("vehicle-status")).toContainText("Charging cheap energy");
    await expect(page.getByTestId("vehicle-status")).toContainText("(limit 50.0 ct)");
  });
  test("price above limit", async ({ page }) => {
    await page.goto("/");
    await page.getByTestId("loadpoint-settings-button").nth(1).click();
    await expect(page.getByTestId("loadpoint-settings-modal")).toBeVisible();
    await page
      .getByTestId("loadpoint-settings-modal")
      .getByLabel("Price limit")
      .selectOption("≤ 10.0 ct/kWh");
    await page.getByTestId("loadpoint-settings-modal").getByLabel("Close").click();
    await expect(page.getByTestId("loadpoint-settings-modal")).not.toBeVisible();
    await expect(page.getByTestId("vehicle-status")).toHaveText("Charging…");
  });
});
