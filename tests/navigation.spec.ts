import { test, expect } from "@playwright/test";
import { start, stop, restart, baseUrl } from "./evcc";
import { expectModalHidden, expectModalVisible } from "./utils";

test.use({ baseURL: baseUrl() });

test.describe("bottom navigation", async () => {
  test.afterAll(async () => {
    await stop();
  });

  test("tabs, navigation, and forecast empty state", async ({ page }) => {
    // start with blank install
    await start();
    await page.goto("/");

    const bottomNav = page.getByTestId("bottom-tab-bar");
    await expect(bottomNav).toBeVisible();

    const tabCharge = bottomNav.getByRole("link", { name: "Charge" });
    const tabBattery = bottomNav.getByRole("link", { name: "Battery" });
    const tabForecast = bottomNav.getByRole("link", { name: "Forecast" });
    const tabSessions = bottomNav.getByRole("link", { name: "Sessions" });
    const tabMore = bottomNav.getByTestId("tab-more");

    // verify bottom nav tabs (no battery)
    await expect(tabCharge).toBeVisible();
    await expect(tabForecast).toBeVisible();
    await expect(tabSessions).toBeVisible();
    await expect(tabMore).toBeVisible();
    await expect(tabBattery).toHaveCount(0);

    // navigate to config via More menu
    await tabMore.click();
    await tabMore.getByRole("link", { name: "Configuration" }).click();
    await expect(page.getByRole("heading", { name: "Configuration" })).toBeVisible();

    // create battery meter
    await page.getByRole("button", { name: "Add solar or battery" }).click();
    const meterModal = page.getByTestId("meter-modal");
    await meterModal.getByRole("button", { name: "Add battery meter" }).click();
    await meterModal.getByLabel("Title").fill("Demo Battery");
    await meterModal.getByLabel("Manufacturer").selectOption("Demo battery");
    await meterModal.getByLabel("Power").fill("1000");
    await meterModal.getByLabel("Charge").fill("50");
    await meterModal.getByRole("button", { name: "Save" }).click();
    await expectModalHidden(meterModal);
    await expect(page.getByTestId("battery")).toBeVisible();

    // restart and verify battery tab appears
    const restartButton = page
      .getByTestId("bottom-banner")
      .getByRole("button", { name: "Restart" });
    await expect(restartButton).toBeVisible();
    await restart();
    await expect(restartButton).not.toBeVisible();

    await page.goto("/");
    await expect(tabBattery).toBeVisible();

    // click all tabs, verify headlines
    await tabCharge.click();
    await expect(page.getByRole("heading", { name: "evcc" })).toBeVisible();

    await tabBattery.click();
    await expect(page.getByRole("heading", { name: "Home Battery" })).toBeVisible();

    await tabForecast.click();
    await expect(page.getByRole("heading", { name: "Forecast" })).toBeVisible();

    await tabSessions.click();
    await expect(page.getByRole("heading", { name: "Sessions" })).toBeVisible();

    // forecast empty state → create solar forecast
    await tabForecast.click();
    await page.getByRole("link", { name: "Set up tariffs and forecasts" }).click();
    await expect(page.getByRole("heading", { name: "Configuration" })).toBeVisible();

    const tariffModal = page.getByTestId("tariff-modal");
    await page.getByRole("button", { name: "Add forecast" }).click();
    await expectModalVisible(tariffModal);
    await tariffModal.getByRole("button", { name: "Add solar forecast" }).click();
    await tariffModal.getByLabel("Title").fill("Demo Forecast");
    await tariffModal.getByLabel("Provider").selectOption("Demo PV Forecast");
    await tariffModal.getByRole("button", { name: "Validate & save" }).click();
    await expectModalHidden(tariffModal);
    await expect(page.getByTestId("tariff-solar")).toBeVisible();

    // restart, navigate to forecast, verify headline
    await expect(restartButton).toBeVisible();
    await restart();
    await expect(restartButton).not.toBeVisible();

    await tabForecast.click();
    await expect(page.getByRole("heading", { name: "Forecast" })).toBeVisible();
  });
});
