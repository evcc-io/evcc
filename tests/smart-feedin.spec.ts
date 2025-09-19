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
    const lp = page.getByTestId("loadpoint").first();

    // set limit
    await lp.getByTestId("loadpoint-settings-button").nth(1).click();
    const modal = page.getByTestId("loadpoint-settings-modal").first();
    await expectModalVisible(modal);
    await modal.getByLabel("Feed-in limit").selectOption("≥ 10.0 ct/kWh");
    await modal.getByLabel("Close").click();
    await expectModalHidden(modal);

    // check status
    await expect(lp.getByTestId("vehicle-status-smartfeedinpriority")).toBeVisible();
    await expect(lp.getByTestId("vehicle-status-smartfeedinpriority")).toHaveText(/≥ 10\.0 ct/);

    // remove limit
    await lp.getByTestId("loadpoint-settings-button").nth(1).click();
    await expectModalVisible(modal);
    await modal.getByLabel("Feed-in limit").selectOption("none");
    await modal.getByLabel("Close").click();
    await expectModalHidden(modal);

    // check status
    await expect(lp.getByTestId("vehicle-status-smartfeedinpriority")).not.toBeVisible();
  });

  test("feed-in apply to all loadpoints", async ({ page }) => {
    await page.goto("/");
    const lp1 = page.getByTestId("loadpoint").first();
    const lp2 = page.getByTestId("loadpoint").last();
    await lp1.getByTestId("loadpoint-settings-button").nth(1).click();
    const modal1 = page.getByTestId("loadpoint-settings-modal").first();
    await expectModalVisible(modal1);
    await modal1.getByLabel("Feed-in limit").selectOption("≥ 10.0 ct/kWh");
    await modal1.getByRole("button", { name: "Apply everywhere?" }).click();
    await modal1.getByLabel("Close").click();
    await expectModalHidden(modal1);

    // check lp1, lp2 status
    await expect(lp1.getByTestId("vehicle-status-smartfeedinpriority")).toBeVisible();
    await expect(lp1.getByTestId("vehicle-status-smartfeedinpriority")).toHaveText(/≥ 10\.0 ct/);
    await expect(lp2.getByTestId("vehicle-status-smartfeedinpriority")).toBeVisible();
    await expect(lp2.getByTestId("vehicle-status-smartfeedinpriority")).toHaveText(/≥ 10\.0 ct/);

    // check lp2 setting
    await lp2.getByTestId("loadpoint-settings-button").nth(1).click();
    const modal2 = page.getByTestId("loadpoint-settings-modal").last();
    await expectModalVisible(modal2);
    await expect(modal2.getByLabel("Feed-in limit")).toHaveValue("0.1");
  });
});
