import { test, expect } from "@playwright/test";
import { start, stop, restart, baseUrl } from "./evcc";
import { expectModalHidden, expectModalVisible } from "./utils";

test.use({ baseURL: baseUrl() });

test.beforeAll(async () => {
  await start();
});
test.afterAll(async () => {
  await stop();
});

test.describe("meter only", async () => {
  test("run without loadpoints", async ({ page }) => {
    await page.goto("/#/config");

    await expect(page.getByTestId("welcome-banner")).toContainText(
      "Start with creating at least one"
    );

    await page.getByRole("button", { name: "Add grid meter" }).click();
    const gridModal = page.getByTestId("meter-modal");
    await expectModalVisible(gridModal);
    await gridModal.getByLabel("Manufacturer").selectOption("Demo meter");
    await gridModal.getByLabel("Power").fill("1000");
    await gridModal.getByRole("button", { name: "Save" }).click();
    await expectModalHidden(gridModal);

    await restart();

    await expect(page.getByTestId("welcome-banner")).not.toBeVisible();
    await expect(page.getByTestId("grid").getByTestId("device-tag-power")).toContainText("1.0 kW");

    await page.getByTestId("home-link").click();

    await expect(page.locator("body")).not.toContainText("Hello aboard!");
    await expect(page.getByTestId("energyflow")).toBeVisible();
    await expect(page.getByTestId("energyflow-entry-gridimport")).toContainText("1.0 kW");
    await expect(page.getByTestId("loadpoints")).toBeEmpty();
  });
});
