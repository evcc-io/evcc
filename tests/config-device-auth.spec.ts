import { test, expect } from "@playwright/test";
import { start, stop, restart, baseUrl } from "./evcc";
import { enableExperimental, expectModalVisible, expectModalHidden } from "./utils";

test.use({ baseURL: baseUrl() });

const templateFlags = [
  "--disable-auth",
  "--template-type",
  "meter",
  "--template",
  "tests/config-device-auth-demo.tpl.yaml",
];

test.beforeEach(async () => {
  await start(undefined, undefined, templateFlags);
});
test.afterEach(async () => {
  await stop();
});

test.describe("config device auth", async () => {
  test("create grid meter with redirect auth", async ({ page }) => {
    await page.goto("/#/config");
    await enableExperimental(page, true);

    // verify no grid meter exists yet
    await expect(page.getByTestId("grid")).toHaveCount(0);
    await expect(page.getByRole("button", { name: "Add grid meter" })).toBeVisible();

    // create a grid meter with auth
    await page.getByRole("button", { name: "Add grid meter" }).click();
    const meterModal = page.getByTestId("meter-modal");
    await expectModalVisible(meterModal);
    await meterModal.getByLabel("Manufacturer").selectOption("Auth Demo Meter");

    // step 1: auth view
    await expect(meterModal.getByLabel("Region")).toBeVisible();
    await expect(meterModal.getByLabel("Authentication Method")).toBeVisible();
    await expect(meterModal.getByLabel("Power")).not.toBeVisible();
    await expect(meterModal.getByRole("button", { name: "Validate & save" })).not.toBeVisible();
    await expect(meterModal.getByRole("button", { name: "Save" })).not.toBeVisible();
    await meterModal.getByLabel("Region").selectOption("EU");
    await meterModal.getByLabel("Authentication Method").selectOption("redirect");
    await meterModal.getByRole("button", { name: "Prepare connection" }).click();
    await expect(meterModal.getByRole("link", { name: "Connect to localhost" })).toBeVisible();
    await expect(meterModal.getByLabel("Authentication Code")).not.toBeVisible();

    // we dont navigate to localhost, just trigger ui update because demo auth state is already established
    await page.evaluate(() => {
      document.dispatchEvent(new Event("visibilitychange"));
    });

    // step 2: show regular device form
    await expect(meterModal.getByLabel("Region")).toHaveValue("EU");
    await expect(meterModal.getByLabel("Authentication Method")).toHaveValue("redirect");
    await expect(meterModal.getByLabel("Power")).toBeVisible();
    await meterModal.getByLabel("Power").fill("5000");
    await expect(meterModal.getByRole("button", { name: "Validate & save" })).toBeVisible();
    await meterModal.getByRole("link", { name: "validate" }).click();
    await expect(meterModal.getByTestId("device-tag-power")).toContainText("5.0 kW");
    await meterModal.getByRole("button", { name: "Save" }).click();
    await expectModalHidden(meterModal);

    // verify meter creation
    await expect(page.getByTestId("grid")).toBeVisible();
    await expect(page.getByTestId("grid")).toContainText("Grid meter");
    await expect(page.getByTestId("grid")).toContainText(["Power", "5.0 kW"].join(""));

    // re-open meter for editing
    await page.getByTestId("grid").getByRole("button", { name: "edit" }).click();
    await expectModalVisible(meterModal);
    await expect(meterModal.getByLabel("Region")).toHaveValue("EU");
    await expect(meterModal.getByLabel("Authentication Method")).toHaveValue("redirect");
    await expect(meterModal.getByLabel("Power")).toHaveValue("5000");
    await expect(meterModal.getByRole("button", { name: "Prepare connection" })).not.toBeVisible();
    await expect(meterModal.getByRole("button", { name: "Validate & save" })).toBeVisible();
    await meterModal.getByRole("button", { name: "Close" }).click();
    await expectModalHidden(meterModal);

    // restart evcc (demo auth doesn't persist)
    await restart(undefined, templateFlags);
    await page.reload();

    // re-open meter for editing after restart, auth status as to be reestablished
    await page.getByTestId("grid").getByRole("button", { name: "edit" }).click();
    await expectModalVisible(meterModal);
    await expect(meterModal.getByLabel("Region")).toHaveValue("EU");
    await expect(meterModal.getByLabel("Authentication Method")).toHaveValue("redirect");
    await expect(meterModal.getByLabel("Power")).not.toBeVisible();
    // note: prepare connection step is auto-executed, since all required fields are already present
    await expect(meterModal.getByRole("link", { name: "Connect to localhost" })).toBeVisible();
    await expect(meterModal.getByRole("button", { name: "Validate & save" })).not.toBeVisible();
  });

  test("create grid meter with device-code auth", async ({ page }) => {
    await page.goto("/#/config");
    await enableExperimental(page, true);

    // create a grid meter with device-code auth
    await page.getByRole("button", { name: "Add grid meter" }).click();
    const meterModal = page.getByTestId("meter-modal");
    await expectModalVisible(meterModal);
    await meterModal.getByLabel("Manufacturer").selectOption("Auth Demo Meter");

    // select device-code method
    await meterModal.getByLabel("Region").selectOption("EU");
    await meterModal.getByLabel("Authentication Method").selectOption("device-code");
    await meterModal.getByRole("button", { name: "Prepare connection" }).click();

    // verify device code is displayed
    await expect(meterModal.getByLabel("Authentication Code")).toHaveValue("12AB345");
    await expect(meterModal).toContainText("Valid for");
    await expect(meterModal).toContainText("Copy this code");
    await expect(meterModal.getByRole("link", { name: "Connect to localhost" })).toBeVisible();
  });

  test("error region shows auth error", async ({ page }) => {
    await page.goto("/#/config");
    await enableExperimental(page, true);

    await page.getByRole("button", { name: "Add grid meter" }).click();
    const meterModal = page.getByTestId("meter-modal");
    await expectModalVisible(meterModal);
    await meterModal.getByLabel("Manufacturer").selectOption("Auth Demo Meter");
    await meterModal.getByLabel("Region").selectOption("ERROR");
    await meterModal.getByLabel("Authentication Method").selectOption("redirect");
    await meterModal.getByRole("button", { name: "Prepare connection" }).click();

    await expect(meterModal).toContainText("region not supported");
    await expect(meterModal.getByRole("button", { name: "Prepare connection" })).toBeVisible();
    await expect(meterModal.getByRole("link", { name: "Connect to localhost" })).not.toBeVisible();
    await expect(meterModal.getByLabel("Authentication Code")).not.toBeVisible();
    await expect(meterModal.getByLabel("Power")).not.toBeVisible();

    // clear error on input change
    await meterModal.getByLabel("Region").selectOption("EU");
    await expect(meterModal).not.toContainText("region not supported");
  });
});
