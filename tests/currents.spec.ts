import { test, expect } from "@playwright/test";
import { start, stop, baseUrl } from "./evcc";
import { expectModalVisible } from "./utils";
test.use({ baseURL: baseUrl() });

test.beforeAll(async () => {
  await start("basics.evcc.yaml");
});
test.afterAll(async () => {
  await stop();
});

test.beforeEach(async ({ page }) => {
  await page.goto("/");
});

test.describe("currents", async () => {
  test("change min and max current", async ({ page }) => {
    // Open loadpoint settings modal
    await page.getByTestId("loadpoint-settings-button").nth(1).click();
    const modal = page.getByTestId("loadpoint-settings-modal");
    await expectModalVisible(modal);

    const minCurrent = modal.getByLabel("Min. current");
    const maxCurrent = modal.getByLabel("Max. current");
    const minPower = modal.getByTestId("min-power");
    const maxPower = modal.getByTestId("max-power");

    // initial values
    await expect(maxCurrent).toHaveValue("16");
    await expect(maxPower).toContainText("~ 11.0 kW");
    await expect(minCurrent).toHaveValue("6");
    await expect(minPower).toContainText("~ 4.1 kW");

    // change min current
    await minCurrent.selectOption("0.125 A");
    await expect(minPower).toContainText("~ 0.1 kW");

    // change max current
    await maxCurrent.selectOption("32 A");
    await expect(maxPower).toContainText("~ 22.1 kW");
  });
});
