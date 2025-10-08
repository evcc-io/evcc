import { test, expect } from "@playwright/test";
import { start, stop, restart, baseUrl } from "./evcc";
import {
  editorClear,
  editorPaste,
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

test.describe("ext meter", async () => {
  test("template-based ext meter", async ({ page }) => {
    await page.goto("/#/config");
    await enableExperimental(page, false);
    await expect(page.getByTestId("ext")).toHaveCount(0);

    await page.getByRole("button", { name: "Add additional meter" }).click();
    const meterModal = page.getByTestId("meter-modal");
    await expectModalVisible(meterModal);
    await meterModal.getByRole("button", { name: "Add regular consumer" }).click();

    await expect(meterModal.getByLabel("Usage")).toHaveValue("charge");

    await meterModal.getByLabel("Manufacturer").selectOption("cFos PowerBrain Meter");
    await page.waitForLoadState("networkidle");

    await meterModal.getByLabel("Usage").selectOption("battery");
    await page.waitForLoadState("networkidle");

    await meterModal.getByLabel("Manufacturer").selectOption("Demo battery");
    await page.waitForLoadState("networkidle");

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

  test("switch from template to custom ext meter", async ({ page }) => {
    await page.goto("/#/config");
    await enableExperimental(page, false);

    await page.getByRole("button", { name: "Add additional meter" }).click();
    const meterModal = page.getByTestId("meter-modal");
    await expectModalVisible(meterModal);
    await meterModal.getByRole("button", { name: "Add regular consumer" }).click();

    await meterModal.getByLabel("Title").fill("Custom ext meter");
    await meterModal.getByLabel("Usage").selectOption("battery");
    await meterModal.getByLabel("Manufacturer").selectOption("Demo battery");
    await meterModal.getByLabel("Charge").fill("50");

    await meterModal.getByLabel("Manufacturer").selectOption("User-defined device");
    await page.waitForLoadState("networkidle");

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

    await page.waitForLoadState("networkidle");
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
    await page.waitForLoadState("networkidle");
    await expect(editor).toContainText("value: 2000 # W");

    await meterModal.getByRole("button", { name: "Delete" }).click();
    await expectModalHidden(meterModal);
    await expect(page.getByTestId("ext")).toHaveCount(0);

    await restart(CONFIG_GRID_ONLY);
    await page.reload();
    await expect(page.getByTestId("ext")).toHaveCount(0);
  });
});
