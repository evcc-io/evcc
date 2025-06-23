import { test, expect } from "@playwright/test";
import { start, stop, baseUrl } from "./evcc";
import { expectModalHidden, expectModalVisible } from "./utils";

const CONFIG = "smart-feedin.evcc.yaml";

test.use({ baseURL: baseUrl() });

test.beforeAll(async () => {
  await start(CONFIG);
});

test.afterAll(async () => {
  await stop();
});

test.describe("smart feed-in priority", async () => {
  test("no limit, normal charging", async ({ page }) => {
    await page.goto("/");
    await expect(page.getByTestId("vehicle-status-smartfeedinpriority")).not.toBeVisible();
  });

  test("feed-in above threshold, pause charging", async ({ page }) => {
    await page.goto("/");
    await page.getByTestId("loadpoint-settings-button").nth(1).click();
    const modal = page.getByTestId("loadpoint-settings-modal");
    expectModalVisible(modal);
    await modal.getByLabel("Feed-in limit").selectOption("≥ 10.0 ct/kWh");
    await modal.getByLabel("Close").click();
    expectModalHidden(modal);
    await expect(page.getByTestId("vehicle-status-smartfeedinpriority")).toBeVisible();
    await expect(page.getByTestId("vehicle-status-smartfeedinpriority")).toHaveText(/≥ 10\.0 ct/);
  });
});
