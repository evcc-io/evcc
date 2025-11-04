import { test, expect } from "@playwright/test";
import { start, stop, baseUrl } from "./evcc";
import { enableExperimental, expectModalVisible, expectModalHidden } from "./utils";

test.use({ baseURL: baseUrl() });

test.afterEach(async () => {
  await stop();
});

test.describe("control settings (interval)", () => {
  test("interval is immediately visible after save without restart", async ({ page }) => {
    await start();
    await page.goto("/#/config");
    await enableExperimental(page);

    const entry = page.getByTestId("generalconfig-control");
    await expect(entry).toContainText("10 s");

    await entry.getByRole("button", { name: "Edit" }).click();

    const modal = page.getByTestId("control-modal");
    await expectModalVisible(modal);

    const intervalInput = modal.getByLabel("Interval");
    await expect(intervalInput).toHaveValue("10");
    await intervalInput.fill("20");

    await modal.getByRole("button", { name: "Save" }).click();

    await expectModalHidden(modal);

    await expect(page.getByTestId("restart-needed")).toBeVisible();

    await expect(entry).toContainText("20 s");
    await expect(entry).not.toContainText("10 s");

    await entry.getByRole("button", { name: "Edit" }).click();
    await expectModalVisible(modal);
    await expect(intervalInput).toHaveValue("20");
  });

  test("residual power is immediately visible after save", async ({ page }) => {
    await start();
    await page.goto("/#/config");
    await enableExperimental(page);

    const entry = page.getByTestId("generalconfig-control");
    await entry.getByRole("button", { name: "Edit" }).click();

    const modal = page.getByTestId("control-modal");
    await expectModalVisible(modal);

    const residualPowerInput = modal.getByLabel("Residual power");
    const initialValue = await residualPowerInput.inputValue();

    await residualPowerInput.fill("200");

    await modal.getByRole("button", { name: "Save" }).click();

    await expectModalHidden(modal);

    await entry.getByRole("button", { name: "Edit" }).click();
    await expectModalVisible(modal);
    await expect(residualPowerInput).toHaveValue("200");
    await expect(residualPowerInput).not.toHaveValue(initialValue);
  });
});
