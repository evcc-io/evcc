import { test, expect } from "@playwright/test";
import { start, stop, restart, baseUrl } from "./evcc";
import { expectModalHidden, expectModalVisible } from "./utils";

test.use({ baseURL: baseUrl() });

test.beforeEach(async () => {
  await start();
});
test.afterEach(async () => {
  await stop();
});

test.describe("consumer", () => {
  test("regular consumer", async ({ page }) => {
    await page.goto("/#/config");

    await page.getByRole("button", { name: "Add consumer" }).click();
    const modal = page.getByTestId("meter-modal");
    await expectModalVisible(modal);
    await modal.getByRole("button", { name: "Regular consumer" }).click();

    // usage selector hidden — consumers are always charge meters
    await expect(modal.getByLabel("Usage")).toHaveCount(0);

    await modal.getByLabel("Title").fill("EV charger");
    await modal.getByLabel("Manufacturer").selectOption("Demo meter");
    await modal.getByLabel("Power").fill("3500");
    await modal.getByRole("button", { name: "Save" }).click();
    await expectModalHidden(modal);

    // appears in consumers section as a consumer card
    await expect(page.getByTestId("consumer")).toHaveCount(1);
    await expect(page.getByTestId("consumer")).toContainText("EV charger");

    // restart and confirm persistence
    await restart();
    await page.reload();
    await expect(page.getByTestId("consumer")).toHaveCount(1);
    await expect(page.getByTestId("consumer")).toContainText("EV charger");
  });

  test("self-regulating consumer", async ({ page }) => {
    await page.goto("/#/config");

    await page.getByRole("button", { name: "Add consumer" }).click();
    const modal = page.getByTestId("meter-modal");
    await expectModalVisible(modal);
    await modal.getByRole("button", { name: "Self-regulating consumer" }).click();

    await modal.getByLabel("Title").fill("Heat pump");
    await modal.getByLabel("Manufacturer").selectOption("Demo meter");
    await modal.getByLabel("Power").fill("800");
    await modal.getByRole("button", { name: "Save" }).click();
    await expectModalHidden(modal);

    // self-regulating consumers are stored as aux meters
    await expect(page.getByTestId("aux")).toHaveCount(1);
    await expect(page.getByTestId("aux")).toContainText("Heat pump");
  });
});
