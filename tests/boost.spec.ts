import { test, expect } from "@playwright/test";
import { start, stop, restart, baseUrl } from "./evcc";
import {
  expectModalHidden,
  expectModalVisible,
  newLoadpoint,
  addDemoCharger,
  ChargerStatus,
} from "./utils";
test.use({ baseURL: baseUrl() });

const CONFIG_BATTERY = "battery-settings.evcc.yaml";

test.afterEach(async () => {
  await stop();
});

test.describe("boost", async () => {
  test("activate and deactivate boost in solar mode", async ({ page }) => {
    await start(CONFIG_BATTERY);
    await page.goto("/");
    const lp = page.getByTestId("loadpoint").first();
    const boostButton = lp.getByTestId("battery-boost-button");
    await expect(boostButton).not.toBeVisible();
    await lp.getByTestId("mode").getByRole("button", { name: "Solar", exact: true }).click();
    await lp.getByTestId("loadpoint-settings-button").last().click();
    const modal = page.getByTestId("loadpoint-settings-modal").first();
    await expectModalVisible(modal);
    await modal.getByLabel("Battery Boost").selectOption("20 %");
    await expect(modal.getByTestId("battery-boost")).toContainText(
      "Allow fast charging from home battery until it's drained to 20%."
    );
    await page.waitForLoadState("networkidle");
    await modal.getByLabel("Close").click();
    await expectModalHidden(modal);
    await expect(boostButton).toHaveAttribute("aria-label", "Battery boost ready");
    // activate boost
    await boostButton.click();
    await expect(boostButton).toHaveAttribute("aria-label", "Battery boost active");
    // deactivate boost
    await boostButton.click();
    await expect(boostButton).toHaveAttribute("aria-label", "Battery boost ready");
  });

  test("battery too low for boost when limit above soc", async ({ page }) => {
    await start(CONFIG_BATTERY);
    await page.goto("/");
    const lp = page.getByTestId("loadpoint").first();
    const boostButton = lp.getByTestId("battery-boost-button");
    await lp.getByTestId("mode").getByRole("button", { name: "Solar", exact: true }).click();
    await lp.getByTestId("loadpoint-settings-button").last().click();
    const modal = page.getByTestId("loadpoint-settings-modal").first();
    await expectModalVisible(modal);
    await modal.getByLabel("Battery Boost").selectOption("90 %");
    await expect(modal.getByTestId("battery-boost")).toContainText("drained to 90%");
    await page.waitForLoadState("networkidle");
    await modal.getByLabel("Close").click();
    await expectModalHidden(modal);
    // limit (90%) above battery soc (50%)
    await expect(boostButton).toHaveAttribute("aria-label", "Battery too low for boost");
  });

  test("boost button disabled in fast mode", async ({ page }) => {
    await start(CONFIG_BATTERY);
    await page.goto("/");
    const lp = page.getByTestId("loadpoint").first();
    const boostButton = lp.getByTestId("battery-boost-button");
    // set a boost limit in solar mode so the boost button appears
    await lp.getByTestId("mode").getByRole("button", { name: "Solar", exact: true }).click();
    await lp.getByTestId("loadpoint-settings-button").last().click();
    const modal = page.getByTestId("loadpoint-settings-modal").first();
    await expectModalVisible(modal);
    await modal.getByLabel("Battery Boost").selectOption("20 %");
    await expect(modal.getByTestId("battery-boost")).toContainText("drained to 20%");
    await page.waitForLoadState("networkidle");
    await modal.getByLabel("Close").click();
    await expectModalHidden(modal);
    // switch to fast mode and verify boost button is disabled
    await lp.getByTestId("mode").getByRole("button", { name: "Fast" }).click();
    await expect(boostButton).toBeDisabled();
  });

  test("boost button disabled when battery is on hold", async ({ page }) => {
    await start(CONFIG_BATTERY);
    await page.goto("/");

    // LP1: set solar mode, configure boost limit
    const lp1 = page.getByTestId("loadpoint").first();
    await lp1.getByTestId("mode").getByRole("button", { name: "Solar", exact: true }).click();
    await lp1.getByTestId("loadpoint-settings-button").last().click();
    const modal = page.getByTestId("loadpoint-settings-modal").first();
    await expectModalVisible(modal);
    await modal.getByLabel("Battery Boost").selectOption("0 %");
    await modal.getByLabel("Close").click();
    await expectModalHidden(modal);

    // enable "Prevent discharge in fast mode"
    await page.goto("/#/battery");
    await page.getByLabel("Prevent discharge in fast mode and planned charging.").check();
    await page.waitForLoadState("networkidle");
    await page.goto("/");

    // LP2: switch to fast mode → triggers global battery hold
    const lp2 = page.getByTestId("loadpoint").nth(1);
    await lp2.getByTestId("mode").getByRole("button", { name: "Fast" }).click();

    // LP1: boost button should show hold state
    const boostButton = lp1.getByTestId("battery-boost-button");
    await expect(boostButton).toHaveAttribute("aria-label", "Battery locked");

    // clicking should not change state
    await boostButton.click();
    await expect(boostButton).toHaveAttribute("aria-label", "Battery locked");
  });

  test("boost default for ui-created loadpoint", async ({ page }) => {
    await start(CONFIG_BATTERY);

    // create a third loadpoint
    await page.goto("/#/config");
    await newLoadpoint(page, "New Charger");
    await addDemoCharger(page, undefined, ChargerStatus.Connected);
    const lpModal = page.getByTestId("loadpoint-modal");
    await lpModal.getByRole("button", { name: "Save" }).click();
    await expectModalHidden(lpModal);

    // restart evcc to apply new loadpoint
    await restart(CONFIG_BATTERY);
    await page.goto("/");
    await expect(page.getByTestId("loadpoint")).toHaveCount(3);

    const lp = page.getByTestId("loadpoint").last();

    // not visible by default
    const boostButton = lp.getByTestId("battery-boost-button");
    await expect(boostButton).not.toBeVisible();

    // visible after setting limit
    await lp.getByTestId("loadpoint-settings-button").last().click();
    const modal = page.getByTestId("loadpoint-settings-modal").last();
    await expectModalVisible(modal);
    await modal.getByLabel("Battery Boost").selectOption("50 %");
    await page.waitForLoadState("networkidle");
    await modal.getByLabel("Close").click();
    await expectModalHidden(modal);
    await expect(boostButton).toBeVisible();
  });
});
