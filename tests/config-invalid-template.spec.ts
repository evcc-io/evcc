import { test, expect } from "@playwright/test";
import { start, stop, restart, baseUrl } from "./evcc";
import { expectModalVisible, expectModalHidden, enableExperimental } from "./utils";

const CONFIG_INVALID_TEMPLATE = "config-invalid-template.sql";

test.use({ baseURL: baseUrl() });

test.afterEach(async () => {
  await stop();
});

test.describe("invalid template migration", async () => {
  test("fix broken grid meter with outdated field", async ({ page }) => {
    await start(undefined, CONFIG_INVALID_TEMPLATE);

    await page.goto("/#/config");
    await enableExperimental(page, false);

    // startup error
    await expect(page.getByTestId("fatal-error")).toBeVisible();
    await expect(page.getByTestId("fatal-error")).toContainText("invalid key: power_old");
    await expect(page.getByTestId("grid")).toBeVisible();

    // edit and save broken meter
    await page.getByTestId("grid").getByRole("button", { name: "edit" }).click();
    const meterModal = page.getByTestId("meter-modal");
    await expectModalVisible(meterModal);
    await expect(meterModal.getByLabel("Manufacturer")).toHaveValue("Demo meter");
    await meterModal.getByLabel("Power").fill("222");
    await meterModal.getByRole("button", { name: "Validate & save" }).click();
    await expectModalHidden(meterModal);
    await expect(page.getByTestId("grid")).toBeVisible();

    // verify restart
    await restart();
    await page.reload();
    await expect(page.getByTestId("fatal-error")).not.toBeVisible();
    await expect(page.getByTestId("grid")).toBeVisible();
    await page.getByTestId("grid").getByRole("button", { name: "edit" }).click();
    await expectModalVisible(meterModal);
    await expect(meterModal.getByLabel("Power")).toHaveValue("222");
    await meterModal.getByRole("button", { name: "Close" }).click();
    await expectModalHidden(meterModal);
  });
});
