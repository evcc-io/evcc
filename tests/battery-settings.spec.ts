import { test, expect } from "@playwright/test";
import { start, stop, baseUrl } from "./evcc";

test.use({ baseURL: baseUrl() });
test.describe.configure({ mode: "parallel" });

test.beforeEach(async () => {
  await start("battery-settings.evcc.yaml");
});
test.afterEach(async () => {
  await stop();
});

test.describe("battery settings", async () => {
  test("battery view", async ({ page }) => {
    await page.goto("/#/battery");

    await expect(page.getByRole("heading", { name: "Home Battery" })).toBeVisible();
    await expect(page.getByRole("heading", { name: "Grid charging" })).toBeVisible();
    await expect(page.getByTestId("header")).toContainText("Home Battery");
    await expect(page.locator("body")).toContainText("Battery level: 50%");
    await expect(page.locator("body")).toContainText("10.0 kWh of 20.0 kWh");
  });

  test("battery usage", async ({ page }) => {
    await page.goto("/#/battery");

    await page.locator("#batterySettingsPriority").selectOption({ label: "50%" });
    await expect(page.locator("label[for=batterySettingsPriorityMiddle] span")).toHaveText("50%");
    await expect(page.locator("label[for=batterySettingsPriorityBottom] span")).toHaveText("50%");
    await page.locator("#batterySettingsBufferTop").selectOption({ label: "80%" });
    await page.locator("#batterySettingsBufferStart").selectOption({ label: "when above 90%." });
    await expect(page.locator("label[for=batterySettingsBuffer] span")).toHaveText("80%");
  });

  test("grid charging", async ({ page }) => {
    await page.goto("/#/battery");

    await page.getByLabel("Enable limit").check();
    await page.getByLabel("Price limit").selectOption({ label: "≤ 50.0 ct/kWh" });
    await expect(page.getByTestId("active-hours")).toHaveText(["Active time", "96 hr"].join(""));
    await expect(page.locator("body")).toContainText("5.0 ct – 50.0 ct");

    await page.getByRole("link", { name: "Charge" }).click();
    await page.getByTestId("energyflow").click();
    await page.getByRole("button", { name: "Grid charging: active (≤ 50.0 ct)" }).click();
    await expect(page).toHaveURL(/#\/battery/);

    await page.getByLabel("Price limit").selectOption({ label: "≤ -10.0 ct/kWh" });
    await expect(page.getByTestId("active-hours")).toHaveText("Active time");

    await page.getByRole("link", { name: "Charge" }).click();
    await expect(
      page.getByRole("button", { name: "Grid charging: when ≤ -10.0 ct" })
    ).toBeVisible();
  });

  test("hold mode display", async ({ page }) => {
    await page.goto("/");
    await page.getByTestId("energyflow").click();

    const discharge = page.getByTestId("energyflow-entry-batterydischarge");
    const charge = page.getByTestId("energyflow-entry-batterycharge");

    await expect(discharge).toContainText("Battery discharging");
    await expect(charge).toContainText("Battery charging");

    // enable discharge lock
    await page.goto("/#/battery");
    await page.getByLabel("Prevent discharge in fast mode and planned charging.").check();
    await page.waitForLoadState("networkidle");
    await page.getByRole("link", { name: "Charge" }).click();

    await page.getByTestId("energyflow").click();
    await expect(discharge).toContainText("Battery (locked)");
    await expect(charge).toContainText("Battery charging");
  });
});
