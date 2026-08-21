import { test, expect } from "@playwright/test";
import { start, stop, baseUrl } from "./evcc";
import { expectModalVisible } from "./utils";

test.use({ baseURL: baseUrl() });

const templateFlags = [
  "--disable-auth",
  "--template-type",
  "meter",
  "--template",
  "tests/config-required-field.tpl.yaml",
];

test.beforeEach(async () => {
  await start(undefined, undefined, templateFlags);
});
test.afterEach(async () => {
  await stop();
});

test.describe("required field validation", async () => {
  // regression test for evcc-io/evcc#29919: clearing a required field after a
  // failed connection test must not allow saving via the "Save anyway" button
  test("cannot save with empty required field after failed test", async ({ page }) => {
    await page.goto("/#/config");

    await page.getByRole("button", { name: "Add grid meter" }).click();
    const meterModal = page.getByTestId("meter-modal");
    await expectModalVisible(meterModal);
    await meterModal.getByLabel("Manufacturer").selectOption("Required Field Demo");

    // fill the required field so the connection test runs (and then fails)
    await meterModal.getByLabel("Secret").fill("some-secret");
    await meterModal.getByRole("button", { name: "Validate & save" }).click();

    // failed test exposes the "Save anyway" force-save button
    const forceSave = meterModal.getByRole("button", { name: "Save anyway" });
    await expect(forceSave).toBeVisible();

    // clearing the field invalidates the stale test result: the button must
    // revert to "Validate & save" instead of staying in force-save mode
    await meterModal.getByLabel("Secret").fill("");
    await expect(forceSave).not.toBeVisible();
    const validateSave = meterModal.getByRole("button", { name: "Validate & save" });
    await expect(validateSave).toBeVisible();

    // attempting to save with the empty required field is blocked by validation
    await validateSave.click();
    await expectModalVisible(meterModal);
    await expect(meterModal.getByLabel("Secret")).toHaveJSProperty("validity.valid", false);

    // nothing was persisted
    await expect(page.getByTestId("grid")).toHaveCount(0);
  });
});
