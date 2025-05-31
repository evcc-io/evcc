import { test, expect } from "@playwright/test";
import { start, stop, restart, baseUrl } from "./evcc";
import {
  editorClear,
  editorType,
  enableExperimental,
  expectModalHidden,
  expectModalVisible,
} from "./utils";

const CONFIG_GRID_ONLY = "config-grid-only.evcc.yaml";

test.use({ baseURL: baseUrl() });

test.beforeEach(async () => {
  await start(CONFIG_GRID_ONLY);
});
test.afterEach(async () => {
  await stop();
});

test.describe("aux meter", async () => {
  test("create and remove aux meter", async ({ page }) => {
    await page.goto("/#/config");
    await enableExperimental(page);

    await expect(page.getByTestId("aux")).toHaveCount(0);

    // create
    await page.getByRole("button", { name: "Add additional meter" }).click();

    const meterModal = page.getByTestId("meter-modal");
    await meterModal.getByRole("button", { name: "Add self-regulating consumer" }).click();
    await meterModal.getByLabel("Title").fill("Water heater");
    await meterModal.getByLabel("Manufacturer").selectOption("Demo meter");
    await meterModal.getByLabel("Power").fill("1200");
    await meterModal.getByRole("button", { name: "Validate & save" }).click();
    await expectModalHidden(meterModal);

    // check
    await expect(page.getByTestId("aux")).toBeVisible(1);
    await expect(page.getByTestId("aux")).toContainText("Water heater");
    await expect(page.getByTestId("aux")).toContainText("1.2 kW");

    // restart and check again
    await restart(CONFIG_GRID_ONLY);
    await page.reload();

    // recheck
    await expect(page.getByTestId("aux")).toBeVisible(1);
    await expect(page.getByTestId("aux")).toContainText("Water heater");
    await expect(page.getByTestId("aux")).toContainText("1.2 kW");

    // delete
    await page.goto("/#/config");
    await page.getByTestId("aux").getByRole("button", { name: "edit" }).click();
    await expectModalVisible(meterModal);
    await meterModal.getByRole("button", { name: "Delete" }).click();
    await expect(page.getByTestId("aux")).toHaveCount(0);

    // restart and check again
    await restart(CONFIG_GRID_ONLY);
    await page.reload();
    await expect(page.getByTestId("aux")).toHaveCount(0);
    await expect(page.getByTestId("fatal-error")).not.toBeVisible();
  });

  test("user-defined meter", async ({ page }) => {
    await page.goto("/#/config");
    await enableExperimental(page);

    await page.getByRole("button", { name: "Add additional meter" }).click();
    const modal = page.getByTestId("meter-modal");
    await expectModalVisible(modal);
    await modal.getByRole("button", { name: "Add self-regulating consumer" }).click();

    await modal.getByLabel("Title").fill("Large heater");
    await modal.getByLabel("Manufacturer").selectOption("User-defined device");
    await page.waitForLoadState("networkidle");
    const editor = modal.getByTestId("yaml-editor");
    await expect(editor).toContainText("power: # current power");

    await editorClear(editor);
    await editorType(editor, [
      // prettier-ignore
      "power:",
      "  source: const",
      "value: 3000 # W",
      "Shift+Tab",
      "energy:",
      "  source: const",
      "value: 42.0 # kWh",
    ]);

    const restResult = modal.getByTestId("test-result");
    await expect(restResult).toContainText("Status: unknown");
    await restResult.getByRole("link", { name: "validate" }).click();
    await expect(restResult).toContainText("Status: successful");
    await expect(restResult).toContainText(["Power", "3.0 kW"].join(""));
    await expect(restResult).toContainText(["Energy", "42.0 kWh"].join(""));

    // create
    await modal.getByRole("button", { name: "Save" }).click();
    await expectModalHidden(modal);
    await expect(page.getByTestId("aux")).toHaveCount(1);
    await expect(page.getByTestId("aux")).toContainText("Large heater");

    // restart evcc
    await restart(CONFIG_GRID_ONLY);
    await page.reload();

    await expect(page.getByTestId("aux")).toHaveCount(1);
    await expect(page.getByTestId("aux")).toContainText("Large heater");

    await page.getByTestId("aux").getByRole("button", { name: "edit" }).click();
    await expectModalVisible(modal);
    await expect(modal.getByLabel("Manufacturer")).toHaveValue("User-defined device");
    await page.waitForLoadState("networkidle");
    await expect(editor).toContainText("value: 3000 # W");
    await expect(editor).toContainText("value: 42.0 # kWh");

    // update
    await modal.getByLabel("Title").fill("Small heater");
    await editorClear(editor);
    await editorType(editor, [
      // prettier-ignore
      "power:",
      "  source: const",
      "value: 300 # W",
      "Shift+Tab",
      "energy:",
      "  source: const",
      "value: 4.2 # kWh",
    ]);
    await expect(restResult).toContainText("Status: unknown");
    await restResult.getByRole("link", { name: "validate" }).click();
    await expect(restResult).toContainText("Status: successful");
    await expect(restResult).toContainText(["Power", "0.3 kW"].join(""));
    await expect(restResult).toContainText(["Energy", "4.2 kWh"].join(""));
    await modal.getByRole("button", { name: "Save" }).click();
    await expectModalHidden(modal);
    await expect(page.getByTestId("aux")).toHaveCount(1);
    await expect(page.getByTestId("aux")).toContainText("Small heater");

    // delete
    await page.getByTestId("aux").getByRole("button", { name: "edit" }).click();
    await expectModalVisible(modal);
    await expect(editor).toContainText("value: 300 # W");
    await expect(editor).toContainText("value: 4.2 # kWh");
    await modal.getByRole("button", { name: "Delete" }).click();
    await expectModalHidden(modal);
    await expect(page.getByTestId("aux")).toHaveCount(0);

    // restart evcc
    await restart(CONFIG_GRID_ONLY);
    await page.reload();

    await expect(page.getByTestId("aux")).toHaveCount(0);
  });

  test("user-defined meter with errors", async ({ page }) => {
    await page.goto("/#/config");
    await enableExperimental(page);

    await page.getByRole("button", { name: "Add additional meter" }).click();

    const modal = page.getByTestId("meter-modal");
    await expectModalVisible(modal);
    await modal.getByRole("button", { name: "Add self-regulating consumer" }).click();

    await modal.getByLabel("Title").fill("Large heater");
    await modal.getByLabel("Manufacturer").selectOption("User-defined device");
    await page.waitForLoadState("networkidle");
    const editor = modal.getByTestId("yaml-editor");

    // yaml syntax error
    await editorClear(editor);
    await editorType(editor, [
      // prettier-ignore
      "hello: world",
      "  foo: bar",
    ]);

    // no errors
    await expect(editor.locator(".line-numbers.error")).toHaveCount(0);
    const restResult = modal.getByTestId("test-result");
    await expect(restResult).toContainText("Status: unknown");
    await restResult.getByRole("link", { name: "validate" }).click();
    await expect(restResult).toContainText("Status: failed");
    await expect(restResult).toContainText(
      "yaml: line 2: mapping values are not allowed in this context"
    );
    await expect(editor.locator(".line-numbers.error")).toHaveCount(1);

    // invalid field error
    await editorClear(editor);
    await editorType(editor, [
      // prettier-ignore
      "apower:",
      "  source: const",
      "value: 3000 # W",
    ]);
    await expect(restResult).toContainText("Status: unknown");
    await restResult.getByRole("link", { name: "validate" }).click();
    await expect(restResult).toContainText("Status: failed");
    await expect(restResult).toContainText("has invalid keys: apower");
    await expect(editor.locator(".line-numbers.error")).toHaveCount(0);

    // unknown source error
    await editorClear(editor);
    await editorType(editor, [
      // prettier-ignore
      "power:",
      "  source: unknown",
      "value: 3000 # W",
    ]);
    await expect(restResult).toContainText("Status: unknown");
    await restResult.getByRole("link", { name: "validate" }).click();
    await expect(restResult).toContainText("Status: failed");
    await expect(restResult).toContainText("invalid plugin type: unknown");
    await expect(editor.locator(".line-numbers.error")).toHaveCount(0);

    // missing required field error
    await editorClear(editor);
    await editorType(editor, [
      // prettier-ignore
      "energy:",
      "  source: const",
      "value: 300 # kWh",
    ]);
    await expect(restResult).toContainText("Status: unknown");
    await restResult.getByRole("link", { name: "validate" }).click();
    await expect(restResult).toContainText("Status: failed");
    await expect(restResult).toContainText("power: missing plugin source");
    await expect(editor.locator(".line-numbers.error")).toHaveCount(0);
  });
});
