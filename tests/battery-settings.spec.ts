import { test, expect } from "@playwright/test";
import { start, stop, baseUrl } from "./evcc";
import { expectModalVisible, expectModalHidden } from "./utils";
test.use({ baseURL: baseUrl() });

test.beforeAll(async () => {
  await start("battery-settings.evcc.yaml");
});
test.afterAll(async () => {
  await stop();
});

test.describe("battery settings", async () => {
  test("open modal", async ({ page }) => {
    await page.goto("/");
    await page.getByTestId("topnavigation-button").click();
    await page.getByTestId("topnavigation-battery").click();

    const modal = page.getByTestId("battery-settings-modal");
    await expect(modal.getByRole("heading", { name: "Home Battery" })).toBeVisible();
    await expect(modal.getByRole("link", { name: "Grid charging 🧪" })).not.toBeVisible();
    await expect(modal).toContainText("Battery level: 50%");
    await expect(modal).toContainText("10.0 kWh of 20.0 kWh");
  });

  test("battery usage", async ({ page }) => {
    await page.goto("/");
    await page.getByTestId("topnavigation-button").click();
    await page.getByTestId("topnavigation-battery").click();

    await page.locator("#batterySettingsPriority").selectOption({ label: "50%" });
    await expect(page.locator("label[for=batterySettingsPriorityMiddle] span")).toHaveText("50%");
    await expect(page.locator("label[for=batterySettingsPriorityBottom] span")).toHaveText("50%");
    await page.locator("#batterySettingsBufferTop").selectOption({ label: "80%" });
    await page.locator("#batterySettingsBufferStart").selectOption({ label: "when above 90%." });
    await expect(page.locator("label[for=batterySettingsBuffer] span")).toHaveText("80%");
  });

  test("grid charging", async ({ page }) => {
    await page.goto("/");
    await page.getByTestId("topnavigation-button").click();
    await page.getByTestId("topnavigation-battery").click();
    const modal = page.getByTestId("battery-settings-modal");
    await expectModalVisible(modal);

    await modal.getByRole("link", { name: "Grid charging" }).click();
    await modal.getByLabel("Price limit").selectOption({ label: "≤ 50.0 ct/kWh" });
    await expect(modal).toContainText("5.0 ct – 50.0 ct");
    await page.getByRole("button", { name: "Close" }).click();
    await expectModalHidden(modal);
    await page.getByTestId("energyflow").click();
    await page.getByRole("button", { name: "grid charging active (≤ 50.0 ct)" }).click();
    await expectModalVisible(modal);
    await modal.getByLabel("Price limit").selectOption({ label: "≤ -10.0 ct/kWh" });
    await modal.getByRole("button", { name: "Close" }).click();
    await expectModalHidden(modal);
    await expect(modal).not.toBeVisible();
    await expect(page.getByRole("button", { name: "grid charging when ≤ -10.0 ct" })).toBeVisible();
  });
});
