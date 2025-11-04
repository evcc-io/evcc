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

    // Initially, control entry should show default interval (30s)
    const controlEntry = page.getByTestId("generalconfig-control");
    await expect(controlEntry).toContainText("30");

    // Open control modal
    await controlEntry.getByRole("button", { name: "Edit" }).click();

    const modal = page.locator("#controlModal");
    await expectModalVisible(modal);

    // Change interval to 60 seconds
    const intervalInput = modal.getByLabel("Interval");
    await expect(intervalInput).toHaveValue("30");
    await intervalInput.fill("60");

    // Save the changes
    await modal.getByRole("button", { name: "Save" }).click();

    // Modal should close
    await expectModalHidden(modal);

    // The control entry should now show the new interval value (60s) without restart
    await expect(controlEntry).toContainText("60");
    await expect(controlEntry).not.toContainText("30");

    // Verify by opening the modal again
    await controlEntry.getByRole("button", { name: "Edit" }).click();
    await expectModalVisible(modal);
    await expect(intervalInput).toHaveValue("60");
  });

  test("residual power is immediately visible after save", async ({ page }) => {
    await start();
    await page.goto("/#/config");
    await enableExperimental(page);

    // Open control modal
    const controlEntry = page.getByTestId("generalconfig-control");
    await controlEntry.getByRole("button", { name: "Edit" }).click();

    const modal = page.locator("#controlModal");
    await expectModalVisible(modal);

    // Get initial residual power value
    const residualPowerInput = modal.getByLabel("Residual power");
    const initialValue = await residualPowerInput.inputValue();

    // Change residual power to 200W
    await residualPowerInput.fill("200");

    // Save the changes
    await modal.getByRole("button", { name: "Save" }).click();

    // Modal should close
    await expectModalHidden(modal);

    // Verify by opening the modal again
    await controlEntry.getByRole("button", { name: "Edit" }).click();
    await expectModalVisible(modal);
    await expect(residualPowerInput).toHaveValue("200");
    await expect(residualPowerInput).not.toHaveValue(initialValue);
  });
});
