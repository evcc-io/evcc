import { test, expect } from "@playwright/test";
import { start, stop, baseUrl } from "./evcc";
test.use({ baseURL: baseUrl() });

test.beforeAll(async () => {
  await start("battery-settings-co2.evcc.yaml");
});
test.afterAll(async () => {
  await stop();
});

test.describe("battery settings co2", async () => {
  test("grid charging", async ({ page }) => {
    await page.goto("/#/battery");

    await page.getByLabel("Enable limit").check();
    await page.getByLabel("CO₂ limit").selectOption({ label: "≤ 150 g/kWh" });
    // CO2 demo tariff provides 72 hours, so active hours vary by test execution time
    await expect(page.getByTestId("active-hours").locator(".value")).toHaveText(/^\d+ hr/);
    await expect(page.locator("body")).toContainText("20 g – 150 g");

    // navigate back to main via bottom nav, open via energyflow deep-link
    await page.getByRole("link", { name: "Charge" }).click();
    await page.getByTestId("energyflow").click();
    await page.getByRole("button", { name: "Grid charging: active (≤ 150 g)" }).click();
    await expect(page).toHaveURL(/#\/battery/);

    await page.getByLabel("CO₂ limit").selectOption({ label: "≤ 10 g/kWh" });
    await expect(page.getByTestId("active-hours")).toHaveText("Active time");

    await page.getByRole("link", { name: "Charge" }).click();
    await expect(page.getByRole("button", { name: "Grid charging: when ≤ 10 g" })).toBeVisible();
  });
});
