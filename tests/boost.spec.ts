import { test, expect } from "@playwright/test";
import { start, stop, restart, baseUrl } from "./evcc";
import { expectModalHidden, newLoadpoint, addDemoCharger } from "./utils";
test.use({ baseURL: baseUrl() });

const CONFIG_BATTERY = "battery-settings.evcc.yaml";

test.afterEach(async () => {
  await stop();
});

test.describe("boost", async () => {
  test("activate and deactivate boost in solar mode", async ({ page }) => {
    await start(CONFIG_BATTERY);
    await page.goto("/");
    const boostButton = page.getByTestId("battery-boost-button");
    await expect(boostButton).not.toBeVisible();
    await page
      .getByTestId("mode")
      .first()
      .getByRole("button", { name: "Solar", exact: true })
      .click();
    await page.getByTestId("loadpoint-settings-button").nth(1).click();
    const modal = page.getByTestId("loadpoint-settings-modal");
    await modal.getByTestId("battery-boost-limit").selectOption("20 %");
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
    const boostButton = page.getByTestId("battery-boost-button");
    await page
      .getByTestId("mode")
      .first()
      .getByRole("button", { name: "Solar", exact: true })
      .click();
    await page.getByTestId("loadpoint-settings-button").nth(1).click();
    const modal = page.getByTestId("loadpoint-settings-modal");
    await modal.getByTestId("battery-boost-limit").selectOption("90 %");
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
    const boostButton = page.getByTestId("battery-boost-button");
    // set a boost limit in solar mode so the boost button appears
    await page
      .getByTestId("mode")
      .first()
      .getByRole("button", { name: "Solar", exact: true })
      .click();
    await page.getByTestId("loadpoint-settings-button").nth(1).click();
    const modal = page.getByTestId("loadpoint-settings-modal");
    await modal.getByTestId("battery-boost-limit").selectOption("20 %");
    await expect(modal.getByTestId("battery-boost")).toContainText("drained to 20%");
    await page.waitForLoadState("networkidle");
    await modal.getByLabel("Close").click();
    await expectModalHidden(modal);
    // switch to fast mode and verify boost button is disabled
    await page.getByTestId("mode").first().getByRole("button", { name: "Fast" }).click();
    await expect(boostButton).toBeDisabled();
  });

  test("boost default for ui-created loadpoint", async ({ page }) => {
    await start(CONFIG_BATTERY);

    // create a second loadpoint via config UI
    await page.goto("/#/config");
    await newLoadpoint(page, "New Charger");
    await addDemoCharger(page);
    const lpModal = page.getByTestId("loadpoint-modal");
    await lpModal.getByRole("button", { name: "Save" }).click();
    await expectModalHidden(lpModal);

    // restart evcc to apply new loadpoint
    await restart(CONFIG_BATTERY);
    await page.goto("/");
    await expect(page.getByTestId("loadpoint")).toHaveCount(2);

    // boost button on new loadpoint should not be visible (batteryBoostLimit defaults to 100 = disabled)
    const boostButton = page.getByTestId("battery-boost-button");
    await expect(boostButton).not.toBeVisible();

    // set boost limit to 50% on new loadpoint and verify button appears
    await page.getByTestId("loadpoint-settings-button").last().click();
    const modal = page.getByTestId("loadpoint-settings-modal").last();
    await modal.getByTestId("battery-boost-limit").selectOption("50 %");
    await page.waitForLoadState("networkidle");
    await modal.getByLabel("Close").click();
    await expectModalHidden(modal);
    await expect(boostButton).toBeVisible();
  });
});
