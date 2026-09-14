import { test, expect } from "@playwright/test";
import { start, stop, baseUrl } from "./evcc";

test.use({ baseURL: baseUrl() });

test.beforeEach(async () => {
  await start("vehicle-asleep.evcc.yaml");
});

test.afterEach(async () => {
  await stop();
});

test.describe("asleep vehicle", async () => {
  test("config: values show sleeping state instead of errors", async ({ page }) => {
    await page.goto("/#/config");

    const vehicle = page.getByTestId("vehicle");
    await expect(vehicle).toHaveCount(1);
    await expect(vehicle).toContainText("Sleepy Car");
    await expect(vehicle.getByTestId("device-tag-capacity")).toContainText("68.0 kWh");
    await expect(vehicle.getByTestId("device-tag-soc")).toContainText("sleeping");
    await expect(page.getByTestId("header")).toBeVisible();
  });
});
