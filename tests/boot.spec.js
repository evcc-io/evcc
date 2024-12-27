import { test, expect } from "@playwright/test";
import { start, stop, baseUrl } from "./evcc";
import { enableExperimental } from "./utils";
test.use({ baseURL: baseUrl() });

test.beforeAll(async () => {
  await start("battery-settings.evcc.yaml");
});
test.afterAll(async () => {
  await stop();
});

test.beforeEach(async ({ page }) => {
  await page.goto("/");
});

test.describe("boost", async () => {
  test("not visible when not experimental", async ({ page }) => {
    await expect(page.getByTestId("vehicle-status-batteryboost")).not.toBeVisible();
    await page.getByTestId("loadpoint-settings-button").nth(1).click();
    const modal = page.getByTestId("loadpoint-settings-modal");
    await expect(modal.getByTestId("battery-boost")).not.toBeVisible();
  });

  test("activate boost in solar mode", async ({ page }) => {
    await enableExperimental(page);
    await expect(page.getByTestId("vehicle-status-batteryboost")).not.toBeVisible();
    await page
      .getByTestId("mode")
      .first()
      .getByRole("button", { name: "Solar", exact: true })
      .click();
    await page.getByTestId("loadpoint-settings-button").nth(1).click();
    const modal = page.getByTestId("loadpoint-settings-modal");
    await modal.getByTestId("battery-boost").getByTestId("battery-boost-checkbox").click();
    await expect(modal.getByTestId("battery-boost")).toContainText(
      "Boost active for this charging session."
    );
    await modal.getByLabel("Close").click();
    await expect(modal).not.toBeVisible();
    await expect(page.getByTestId("vehicle-status-batteryboost")).toBeVisible();
  });

  test("disabled in fast mode", async ({ page }) => {
    await enableExperimental(page);
    await page.getByTestId("mode").first().getByRole("button", { name: "Fast" }).click();
    await page.getByTestId("loadpoint-settings-button").nth(1).click();
    const modal = page.getByTestId("loadpoint-settings-modal");
    await expect(modal.getByTestId("battery-boost-checkbox")).toBeDisabled();
    await expect(modal.getByTestId("battery-boost")).toContainText(
      "Only available in solar and min+solar mode."
    );
  });
});
