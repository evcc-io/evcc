import { test, expect } from "@playwright/test";
import { start, stop, baseUrl } from "./evcc";
import { expectModalVisible, expectModalHidden, openTopNavigation } from "./utils";
test.use({ baseURL: baseUrl() });

test.beforeAll(async () => {
  await start("battery-settings-co2.evcc.yaml");
});
test.afterAll(async () => {
  await stop();
});

test.describe("battery settings co2", async () => {
  test("grid charging", async ({ page }) => {
    await page.goto("/");
    await openTopNavigation(page);
    await page.getByTestId("topnavigation-battery").click();
    const modal = page.getByTestId("battery-settings-modal");
    await expectModalVisible(modal);

    await modal.getByRole("link", { name: "Grid charging" }).click();
    await modal.getByLabel("CO₂ limit").selectOption({ label: "≤ 150 g/kWh" });
    await expect(modal).toContainText("20 g – 150 g");
    await page.getByRole("button", { name: "Close" }).click();
    await expectModalHidden(modal);
    await page.getByTestId("energyflow").click();
    await page.getByRole("button", { name: "grid charging active (≤ 150 g)" }).click();
    await expectModalVisible(modal);
    await modal.getByLabel("CO₂ limit").selectOption({ label: "≤ 10 g/kWh" });
    await modal.getByRole("button", { name: "Close" }).click();
    await expectModalHidden(modal);
    await expect(modal).not.toBeVisible();
    await expect(page.getByRole("button", { name: "grid charging when ≤ 10 g" })).toBeVisible();
  });
});
