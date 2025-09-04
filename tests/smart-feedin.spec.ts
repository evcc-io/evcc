import { test, expect } from "@playwright/test";
import { start, stop, baseUrl } from "./evcc";
import { expectModalHidden, expectModalVisible, openTopNavigation } from "./utils";

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

test.describe("smart feed-in disable (zero feed-in)", async () => {
  test("configure feed-in limit and verify production reduced indicator", async ({ page }) => {
    await page.goto("/");

    // open energy flow
    const energyflow = page.getByTestId("energyflow");
    await energyflow.click();
    await expect(energyflow).not.toContainText("Production (reduced)");

    // open export in forecast modal
    await openTopNavigation(page);
    await page.getByTestId("topnavigation-forecast").click();
    const forecastModal = page.getByTestId("forecast-modal");
    await expectModalVisible(forecastModal);
    await forecastModal.getByRole("button", { name: "Export" }).click();

    // verify content sections
    await expect(forecastModal.getByText("Average")).toBeVisible();
    await expect(forecastModal.getByText("2.5 ct/kWh")).toBeVisible();
    await expect(forecastModal.getByText("Range")).toBeVisible();
    await expect(forecastModal.getByText("-2.0 – 10.0 ct/kWh")).toBeVisible();
    await expect(forecastModal.getByText("Highest hour")).toBeVisible();
    await expect(forecastModal.getByText("10.0 ct/kWh")).toBeVisible();

    // enable limit
    const smartFeedInLimit = forecastModal.getByTestId("smart-feed-in-disable-limit");
    await expect(smartFeedInLimit).toContainText("low");
    await smartFeedInLimit.getByRole("switch").click();
    await expect(smartFeedInLimit).toContainText("≤ 0.0 ct");

    // change limit
    const options = smartFeedInLimit.locator("#smartFeedInDisableLimit");
    await options.selectOption({ label: "≤ -2.0 ct" });
    await expect(smartFeedInLimit).toContainText("≤ -2.0 ct");

    // set high limit
    await options.selectOption({ label: "≤ 10.0 ct" });
    await forecastModal.getByLabel("Close").click();
    await expectModalHidden(forecastModal);

    // verify limit
    await expect(energyflow).toContainText("Production (reduced)");

    // reset to normal
    await openTopNavigation(page);
    await page.getByTestId("topnavigation-forecast").click();
    await expectModalVisible(forecastModal);
    await forecastModal.getByRole("button", { name: "Export" }).click();
    await smartFeedInLimit.getByRole("switch").click();
    await forecastModal.getByLabel("Close").click();
    await expectModalHidden(forecastModal);

    // verify reset
    await expect(energyflow).not.toContainText("Production (reduced)");
  });
});
