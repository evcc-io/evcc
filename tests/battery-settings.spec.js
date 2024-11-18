import { test, expect } from "@playwright/test";
import { start, stop, baseUrl } from "./evcc";
import { enableExperimental } from "./utils";
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
    await enableExperimental(page);
    await page.getByTestId("topnavigation-button").click();
    await page.getByTestId("topnavigation-battery").click();

    await page.getByRole("link", { name: "Grid charging 🧪" }).click();
    await page.getByLabel("Price limit").selectOption({ label: "≤ 50.0 ct/kWh" });
    await expect(page.getByTestId("battery-settings-modal")).toContainText("5.0 ct – 50.0 ct");
    await page.getByRole("button", { name: "Close" }).click();
    await expect(page.getByTestId("battery-settings-modal")).not.toBeVisible();
    await page.getByTestId("energyflow").click();
    await page.getByRole("button", { name: "cheap grid energy (≤ 50.0 ct)" }).click();
    await expect(page.getByTestId("battery-settings-modal")).toBeVisible();
  });
});
