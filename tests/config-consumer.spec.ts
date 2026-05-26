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

    // usage selector hidden — pre-locked to charge
    await expect(modal.getByLabel("Usage")).toHaveCount(0);

    await modal.getByLabel("Title").fill("EV charger");
    await modal.getByLabel("Manufacturer").selectOption("Demo meter");
    await modal.getByLabel("Power").fill("3500");
    await modal.getByRole("button", { name: "Save" }).click();
    await expectModalHidden(modal);

    // appears in consumers section (rendered as ext meter card)
    await expect(page.getByTestId("ext")).toHaveCount(1);
    await expect(page.getByTestId("ext")).toContainText("EV charger");

    // restart and confirm persistence + classification
    await restart();
    await page.reload();
    await expect(page.getByTestId("ext")).toHaveCount(1);
    await expect(page.getByTestId("ext")).toContainText("EV charger");
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
    await modal.getByRole("button", { name: "Validate & save" }).click();
    await expectModalHidden(modal);

    await expect(page.getByTestId("aux")).toHaveCount(1);
    await expect(page.getByTestId("aux")).toContainText("Heat pump");
  });

  test("additional meter", async ({ page }) => {
    await page.goto("/#/config");

    await page.getByRole("button", { name: "Add additional meter" }).click();
    const modal = page.getByTestId("meter-modal");
    await expectModalVisible(modal);

    // usage selector visible without default; charge not offered
    const usage = modal.getByLabel("Usage");
    await expect(usage).toHaveValue("");
    const usageValues = await usage
      .locator("option")
      .evaluateAll((els) => els.map((e) => (e as HTMLOptionElement).value).filter((v) => v !== ""));
    expect(usageValues).toEqual(["grid", "pv", "battery", "aux"]);

    await modal.getByLabel("Usage").selectOption("grid");
    await modal.getByLabel("Title").fill("Garage submeter");
    await modal.getByLabel("Manufacturer").selectOption("Demo meter");
    await modal.getByLabel("Power").fill("500");
    await modal.getByRole("button", { name: "Save" }).click();
    await expectModalHidden(modal);

    // appears in additional section
    await expect(page.getByTestId("ext")).toHaveCount(1);
    await expect(page.getByTestId("ext")).toContainText("Garage submeter");
  });
});
