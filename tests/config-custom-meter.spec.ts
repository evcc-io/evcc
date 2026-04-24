import { test, expect } from "@playwright/test";
import { start, stop, restart, baseUrl } from "./evcc";
import { startSimulator, stopSimulator, simulatorHost } from "./simulator";
import { expectModalVisible, expectModalHidden, editorClear, editorPaste } from "./utils";

test.use({ baseURL: baseUrl() });

test.afterEach(async () => {
  await stop();
  await stopSimulator();
});

test.describe("custom meter type override", async () => {
  test("user-defined grid meter with explicit shelly type", async ({ page }) => {
    await startSimulator();
    await start();

    await page.goto("/#/config");

    // add grid meter as user-defined device with explicit type: shelly
    await page.getByRole("button", { name: "Add grid meter" }).click();
    const meterModal = page.getByTestId("meter-modal");
    await expectModalVisible(meterModal);
    await meterModal.getByLabel("Manufacturer").selectOption("User-defined device");
    await page.waitForLoadState("networkidle");

    const editor = meterModal.getByTestId("yaml-editor");
    await editorClear(editor, 20);
    await editorPaste(
      editor,
      page,
      `type: shelly
uri: http://${simulatorHost()}`
    );

    // validate
    const testResult = meterModal.getByTestId("test-result");
    await expect(testResult).toContainText("Status: unknown");
    await testResult.getByRole("link", { name: "validate" }).click();
    await expect(testResult).toContainText("Status: successful");
    await expect(testResult).toContainText(["Energy", "0.0 kWh"].join(""));

    // save
    await meterModal.getByRole("button", { name: "Save" }).click();
    await expectModalHidden(meterModal);
    await expect(page.getByTestId("grid")).toBeVisible();

    // restart and verify no fatal error
    await restart();
    await page.reload();
    await expect(page.getByTestId("fatal-error")).not.toBeVisible();
    await expect(page.getByTestId("grid")).toBeVisible();
    await expect(page.getByTestId("grid")).toContainText(["Energy", "0.0 kWh"].join(""));
  });
});
