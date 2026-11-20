import { test, expect } from "@playwright/test";
import { start, stop, restart, baseUrl } from "./evcc";
import { expectModalVisible, expectModalHidden } from "./utils";

test.use({ baseURL: baseUrl() });

const flags = ["--disable-auth", "--disable-onboarding", "--template-type", "meter", "--template"];

test.afterEach(async () => {
  await stop();
});

test.describe("hidden template params", async () => {
  test("defaults not stored", async ({ page }) => {
    await start(undefined, undefined, [...flags, "tests/config-hidden-defaults-v1.tpl.yaml"]);
    await page.goto("/#/config");

    await page.getByRole("button", { name: "Add grid meter" }).click();
    const meterModal = page.getByTestId("meter-modal");
    await expectModalVisible(meterModal);
    await meterModal.getByLabel("Manufacturer").selectOption("Hidden Defaults Meter");
    await meterModal.getByLabel("Power").fill("5000");
    await meterModal.getByRole("button", { name: "Save" }).click();
    await expectModalHidden(meterModal);
    await expect(page.getByTestId("grid")).toBeVisible();

    await restart(undefined, [...flags, "tests/config-hidden-defaults-v2.tpl.yaml"]);
    await page.reload();

    await expect(page.getByTestId("grid")).toBeVisible();
    await expect(page.getByTestId("fatal-error")).not.toBeVisible();
  });
});
