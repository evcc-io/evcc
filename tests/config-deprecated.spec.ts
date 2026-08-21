import { test, expect } from "@playwright/test";
import { start, stop, restart, baseUrl } from "./evcc";
import { expectModalVisible, expectModalHidden } from "./utils";

test.use({ baseURL: baseUrl() });

const flags = ["--disable-auth", "--template-type", "meter", "--template"];
const templateFlags = [...flags, "tests/config-deprecated-false.tpl.yaml"];
const deprecatedFlags = [...flags, "tests/config-deprecated-true.tpl.yaml"];

test.beforeAll(async () => {
  await start(undefined, undefined, templateFlags);
});

test.afterAll(async () => {
  await stop();
});

test.describe("deprecated template", async () => {
  test("editable after deprecation", async ({ page }) => {
    await page.goto("/#/config");

    await page.getByRole("button", { name: "Add grid meter" }).click();
    const meterModal = page.getByTestId("meter-modal");
    await expectModalVisible(meterModal);
    await meterModal.getByLabel("Manufacturer").selectOption("Old Meter");
    await meterModal.getByLabel("Power").fill("5000");
    await meterModal.getByRole("button", { name: "Save" }).click();
    await expectModalHidden(meterModal);

    await expect(page.getByTestId("grid")).toBeVisible();
    await expect(page.getByTestId("grid")).toContainText("Grid meter");

    await restart(undefined, deprecatedFlags);
    await page.reload();

    await expect(page.getByTestId("grid")).toBeVisible();

    await page.getByTestId("grid").getByRole("button", { name: "edit" }).click();
    await expectModalVisible(meterModal);
    await expect(meterModal.getByLabel("Manufacturer")).toHaveValue("Old Meter");
    await expect(meterModal.getByLabel("Power")).toHaveValue("5000");
    await meterModal.getByRole("button", { name: "Save" }).click();
    await expectModalHidden(meterModal);

    await page.getByTestId("grid").getByRole("button", { name: "edit" }).click();
    await expectModalVisible(meterModal);
    await meterModal.getByRole("button", { name: "Delete" }).click();
    await expectModalHidden(meterModal);

    await expect(page.getByTestId("grid")).not.toBeVisible();

    await page.getByRole("button", { name: "Add grid meter" }).click();
    await expectModalVisible(meterModal);
    const manufacturerSelect = meterModal.getByLabel("Manufacturer");
    const options = await manufacturerSelect.locator("option").allTextContents();
    expect(options).not.toContain("Old Meter");
    expect(options).not.toContain("Deprecated Meter");
  });
});
