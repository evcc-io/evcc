import { test, expect } from "@playwright/test";
import { start, stop, baseUrl } from "./evcc";
import { expectModalVisible } from "./utils";

test.use({ baseURL: baseUrl() });

const templateFlags = [
  "--disable-auth",
  "--disable-onboarding",
  "--template-type",
  "meter",
  "--template",
  "tests/config-usage-filter.tpl.yaml",
];

test.beforeAll(async () => {
  await start(undefined, undefined, templateFlags);
});
test.afterAll(async () => {
  await stop();
});

test.describe("param usage filter", async () => {
  test("pv", async ({ page }) => {
    await page.goto("/#/config");
    await page.getByRole("button", { name: "Add solar or battery" }).click();
    const meterModal = page.getByTestId("meter-modal");
    await expectModalVisible(meterModal);
    await meterModal.getByRole("button", { name: "Add solar meter" }).click();
    await meterModal.getByLabel("Manufacturer").selectOption("Usage Filter Demo");
    await expect(meterModal.getByLabel("PV only")).toBeVisible();
    await expect(meterModal.getByLabel("Battery only")).toHaveCount(0);
  });

  test("battery", async ({ page }) => {
    await page.goto("/#/config");
    await page.getByRole("button", { name: "Add solar or battery" }).click();
    const meterModal = page.getByTestId("meter-modal");
    await expectModalVisible(meterModal);
    await meterModal.getByRole("button", { name: "Add battery meter" }).click();
    await meterModal.getByLabel("Manufacturer").selectOption("Usage Filter Demo");
    await expect(meterModal.getByLabel("Battery only")).toBeVisible();
    await expect(meterModal.getByLabel("PV only")).toHaveCount(0);
  });
});
