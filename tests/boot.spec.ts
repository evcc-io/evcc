import { test, expect } from "@playwright/test";
import { start, stop, baseUrl } from "./evcc";
import { expectModalHidden } from "./utils";
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
  test("activate and deactivate boost in solar mode", async ({ page }) => {
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
    await modal.getByLabel("Close").click();
    await expectModalHidden(modal);
    const status = page.getByTestId("vehicle-status");
    // activate boost
    await boostButton.click();
    await expect(status).toContainText("Boost until battery at 20%.");
    // deactivate boost
    await boostButton.click();
    await expect(status).toContainText("Battery boost disabled.");
  });

  test("battery too low for boost when limit above soc", async ({ page }) => {
    const boostButton = page.getByTestId("battery-boost-button");
    await page
      .getByTestId("mode")
      .first()
      .getByRole("button", { name: "Solar", exact: true })
      .click();
    await page.getByTestId("loadpoint-settings-button").nth(1).click();
    const modal = page.getByTestId("loadpoint-settings-modal");
    await modal.getByTestId("battery-boost-limit").selectOption("90 %");
    await modal.getByLabel("Close").click();
    await expectModalHidden(modal);
    // try to activate boost with limit (90%) above battery soc (50%)
    await boostButton.click();
    await expect(page.getByTestId("vehicle-status")).toContainText("Battery too low for boost.");
  });

  test("boost button disabled in fast mode", async ({ page }) => {
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
    await modal.getByLabel("Close").click();
    await expectModalHidden(modal);
    // switch to fast mode and verify boost button is disabled
    await page.getByTestId("mode").first().getByRole("button", { name: "Fast" }).click();
    await expect(boostButton).toBeDisabled();
  });
});
