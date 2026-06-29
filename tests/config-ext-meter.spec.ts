import { test, expect, type Page } from "@playwright/test";
import { start, stop, restart, baseUrl } from "./evcc";
import { editorClear, editorPaste, expectModalHidden, expectModalVisible } from "./utils";

const CONFIG_GRID_ONLY = "config-grid-only.evcc.yaml";
const CONFIG_BASICS = "basics.evcc.yaml";

test.use({ baseURL: baseUrl() });

async function createAdditionalMeter(page: Page, title: string, power: string) {
  const modal = page.getByTestId("meter-modal");

  await page.getByRole("button", { name: "Add additional meter" }).click();
  await expectModalVisible(modal);
  await modal.getByLabel("Title").fill(title);
  await modal.getByLabel("Manufacturer").selectOption("Demo meter");
  await modal.getByLabel("Power").fill(power);
  await modal.getByRole("button", { name: "Save" }).click();
  await expectModalHidden(modal);
}

test.describe("ext meter", async () => {
  test.beforeEach(async () => {
    await start(CONFIG_GRID_ONLY);
  });
  test.afterEach(async () => {
    await stop();
  });

  test("template-based ext meter", async ({ page }) => {
    await page.goto("/#/config");
    await expect(page.getByTestId("ext")).toHaveCount(0);

    // additional meter flow: opens ext modal directly, usage defaults to charge
    await page.getByRole("button", { name: "Add additional meter" }).click();
    const meterModal = page.getByTestId("meter-modal");
    await expectModalVisible(meterModal);

    await expect(meterModal.getByLabel("Usage")).toHaveValue("charge");

    await meterModal.getByLabel("Usage").selectOption("battery");
    await meterModal.getByLabel("Manufacturer").selectOption("Demo battery");
    await expect(meterModal.getByLabel("Charge")).toBeVisible();
    await meterModal.getByLabel("Title").fill("House battery");
    await meterModal.getByLabel("Charge").fill("75");

    const testResult = meterModal.getByTestId("test-result");
    await testResult.getByRole("link", { name: "validate" }).click();
    await expect(testResult).toContainText("Status: successful");
    await expect(testResult).toContainText("75.0%");

    await meterModal.getByRole("button", { name: "Save" }).click();
    await expectModalHidden(meterModal);

    await expect(page.getByTestId("ext")).toHaveCount(1);
    await expect(page.getByTestId("ext")).toContainText("House battery");

    await restart(CONFIG_GRID_ONLY);
    await page.reload();
    await expect(page.getByTestId("ext")).toHaveCount(1);

    await page.getByTestId("ext").getByRole("button", { name: "edit" }).click();
    await expectModalVisible(meterModal);
    await expect(meterModal.getByLabel("Title")).toHaveValue("House battery");
    await expect(meterModal.getByLabel("Usage")).toBeDisabled();
    await expect(meterModal.getByLabel("Manufacturer")).toBeDisabled();

    await meterModal.getByLabel("Charge").clear();
    await meterModal.getByLabel("Charge").fill("85");
    await testResult.getByRole("link", { name: "validate" }).click();
    await expect(testResult).toContainText("85.0%");
    await meterModal.getByRole("button", { name: "Save" }).click();
    await expectModalHidden(meterModal);

    await page.getByTestId("ext").getByRole("button", { name: "edit" }).click();
    await expectModalVisible(meterModal);
    await expect(meterModal.getByLabel("Charge")).toHaveValue("85");

    await meterModal.getByRole("button", { name: "Delete" }).click();
    await expectModalHidden(meterModal);
    await expect(page.getByTestId("ext")).toHaveCount(0);

    await restart(CONFIG_GRID_ONLY);
    await page.reload();
    await expect(page.getByTestId("ext")).toHaveCount(0);
    await expect(page.getByTestId("fatal-error")).not.toBeVisible();
  });

  test("convert charge ext meter to consumer", async ({ page }) => {
    await page.goto("/#/config");
    await expect(page.getByTestId("ext")).toHaveCount(0);
    await expect(page.getByTestId("consumer")).toHaveCount(0);

    // additional meter defaults to usage charge
    await createAdditionalMeter(page, "Fridge", "150");
    await expect(page.getByTestId("ext")).toHaveCount(1);

    const meterModal = page.getByTestId("meter-modal");
    await page.getByTestId("ext").getByRole("button", { name: "edit" }).click();
    await expectModalVisible(meterModal);

    page.once("dialog", (dialog) => dialog.accept());
    await meterModal.getByRole("button", { name: "Convert to consumer" }).click();
    await expectModalHidden(meterModal);

    // moved from additional meters into consumers
    await expect(page.getByTestId("ext")).toHaveCount(0);
    await expect(page.getByTestId("consumer")).toHaveCount(1);
    await expect(page.getByTestId("consumer")).toContainText("Fridge");

    // persists across restart (history reconciled on boot)
    await restart(CONFIG_GRID_ONLY);
    await page.reload();
    await expect(page.getByTestId("ext")).toHaveCount(0);
    await expect(page.getByTestId("consumer")).toHaveCount(1);
    await expect(page.getByTestId("consumer")).toContainText("Fridge");
    await expect(page.getByTestId("fatal-error")).not.toBeVisible();
  });

  test("convert option hidden for non-charge ext meter", async ({ page }) => {
    await page.goto("/#/config");

    await page.getByRole("button", { name: "Add additional meter" }).click();
    const meterModal = page.getByTestId("meter-modal");
    await expectModalVisible(meterModal);

    await meterModal.getByLabel("Usage").selectOption("battery");
    await meterModal.getByLabel("Manufacturer").selectOption("Demo battery");
    await meterModal.getByLabel("Title").fill("House battery");
    await meterModal.getByLabel("Charge").fill("75");
    await meterModal.getByRole("button", { name: "Save" }).click();
    await expectModalHidden(meterModal);

    await page.getByTestId("ext").getByRole("button", { name: "edit" }).click();
    await expectModalVisible(meterModal);
    await expect(meterModal.getByRole("button", { name: "Convert to consumer" })).toHaveCount(0);
  });

  test("switch from template to custom ext meter", async ({ page }) => {
    await page.goto("/#/config");

    await page.getByRole("button", { name: "Add additional meter" }).click();
    const meterModal = page.getByTestId("meter-modal");
    await expectModalVisible(meterModal);

    await meterModal.getByLabel("Title").fill("Custom ext meter");
    await meterModal.getByLabel("Usage").selectOption("pv");

    // switch to an in-beteen template to ensure we dont leak values
    await meterModal.getByLabel("Manufacturer").selectOption("SunSpec Hybrid Inverter");
    await expect(meterModal.getByLabel("IP address or hostname")).toBeVisible();

    await meterModal.getByLabel("Manufacturer").selectOption("User-defined device");

    const editor = meterModal.getByTestId("yaml-editor");
    await expect(editor).toContainText("power: # current power");

    const testResult = meterModal.getByTestId("test-result");
    await testResult.getByRole("link", { name: "validate" }).click();
    await expect(testResult).toContainText("Status: successful");
    await expect(testResult).toContainText("1.0 kW");

    await meterModal.getByRole("button", { name: "Save" }).click();
    await expectModalHidden(meterModal);

    await expect(page.getByTestId("ext")).toHaveCount(1);
    await expect(page.getByTestId("ext")).toContainText("Custom ext meter");

    await restart(CONFIG_GRID_ONLY);
    await page.reload();
    await expect(page.getByTestId("ext")).toHaveCount(1);

    await page.getByTestId("ext").getByRole("button", { name: "edit" }).click();
    await expectModalVisible(meterModal);
    await expect(meterModal.getByLabel("Title")).toHaveValue("Custom ext meter");
    await expect(meterModal.getByLabel("Usage")).toBeDisabled();
    await expect(meterModal.getByLabel("Manufacturer")).toHaveValue("User-defined device");

    await expect(editor).toContainText("value: 1000 # W");

    await editorClear(editor);
    await editorPaste(
      editor,
      page,
      `power:
  source: const
  value: 2000 # W`
    );

    await testResult.getByRole("link", { name: "validate" }).click();
    await expect(testResult).toContainText("Status: successful");
    await expect(testResult).toContainText("2.0 kW");

    await meterModal.getByRole("button", { name: "Save" }).click();
    await expectModalHidden(meterModal);

    await page.getByTestId("ext").getByRole("button", { name: "edit" }).click();
    await expectModalVisible(meterModal);
    await expect(editor).toContainText("value: 2000 # W");

    await meterModal.getByRole("button", { name: "Delete" }).click();
    await expectModalHidden(meterModal);
    await expect(page.getByTestId("ext")).toHaveCount(0);

    await restart(CONFIG_GRID_ONLY);
    await page.reload();
    await expect(page.getByTestId("ext")).toHaveCount(0);
  });
});

test.describe("ext meter order", async () => {
  test.beforeEach(async () => {
    await start(CONFIG_BASICS);
  });
  test.afterEach(async () => {
    await stop();
  });

  test("ensure order is preserved", async ({ page }) => {
    await page.goto("/#/config");
    await expect(page.getByTestId("ext")).toHaveCount(0);

    // Create meters
    await createAdditionalMeter(page, "Meter 1", "10");
    await createAdditionalMeter(page, "Meter 2", "20");
    await createAdditionalMeter(page, "Meter 3", "30");

    // Verify order in config UI
    const extMeters = page.getByTestId("ext");
    await expect(extMeters).toHaveCount(3);
    await expect(extMeters.nth(0)).toContainText("Meter 1");
    await expect(extMeters.nth(1)).toContainText("Meter 2");
    await expect(extMeters.nth(2)).toContainText("Meter 3");

    // Restart and check order is preserved in both UIs
    await restart(CONFIG_BASICS);

    // Check config UI, reload to reconnect websocket
    await page.reload();
    await expect(extMeters).toHaveCount(3);
    await expect(extMeters.nth(0)).toContainText("Meter 1");
    await expect(extMeters.nth(1)).toContainText("Meter 2");
    await expect(extMeters.nth(2)).toContainText("Meter 3");
  });
});
