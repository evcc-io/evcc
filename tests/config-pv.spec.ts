import { test, expect } from "@playwright/test";
import { start, stop, restart, baseUrl } from "./evcc";
import { enableExperimental, expectModalHidden, expectModalVisible } from "./utils";

const CONFIG_GRID_ONLY = "config-grid-only.evcc.yaml";

test.use({ baseURL: baseUrl() });

test.beforeAll(async () => {
  await start(CONFIG_GRID_ONLY);
});
test.afterAll(async () => {
  await stop();
});

test.describe("pv meter", async () => {
  test("create, edit and remove pv meter", async ({ page }) => {
    await page.goto("/#/config");
    await enableExperimental(page, false);

    await expect(page.getByTestId("pv")).toHaveCount(0);

    // create #1
    await page.getByRole("button", { name: "Add solar or battery" }).click();

    const meterModal = page.getByTestId("meter-modal");
    await meterModal.getByRole("button", { name: "Add solar meter" }).click();
    await meterModal.getByLabel("Title").fill("PV North");
    await meterModal.getByLabel("Manufacturer").selectOption("Demo meter");
    await meterModal.getByLabel("Power optional").fill("5000");
    await page.getByRole("button", { name: "Show advanced settings" }).click();
    await expect(meterModal.getByLabel("Minimum charge")).not.toBeVisible(); // battery usage only
    await expect(meterModal.getByLabel("Maximum AC power of the hybrid inverter")).toBeVisible(); // pv usage only
    await expect(meterModal.getByRole("button", { name: "Validate & save" })).toBeVisible();
    await meterModal.getByRole("link", { name: "validate" }).click();
    await expect(meterModal.getByTestId("device-tag-power")).toContainText("5.0 kW");
    await meterModal.getByRole("button", { name: "Save" }).click();
    await expectModalHidden(meterModal);
    await expect(page.getByTestId("pv")).toBeVisible();
    await expect(page.getByTestId("pv")).toContainText("PV North");

    // edit #1
    await page.getByTestId("pv").getByRole("button", { name: "edit" }).click();
    await expectModalVisible(meterModal);
    await meterModal.getByLabel("Power optional").fill("6000");
    await meterModal.getByRole("button", { name: "Validate & save" }).click();
    await expectModalHidden(meterModal);

    const pv = page.getByTestId("pv");
    await expect(pv).toBeVisible();
    await expect(pv).toContainText("PV North");
    await expect(pv.getByTestId("device-tag-power")).toContainText("6.0 kW");

    // restart and check in main ui
    await restart(CONFIG_GRID_ONLY);
    await page.goto("/");
    await page.getByTestId("visualization").click();
    await expect(page.getByTestId("energyflow")).toContainText("Production6.0 kW");

    // delete #1
    await page.goto("/#/config");
    await page.getByTestId("pv").getByRole("button", { name: "edit" }).click();
    await expectModalVisible(meterModal);
    await meterModal.getByRole("button", { name: "Delete" }).click();
    await expectModalHidden(meterModal);
    await expect(page.getByTestId("pv")).toHaveCount(0);

    // restart and check again
    await restart(CONFIG_GRID_ONLY);
    await page.reload();
    await expect(page.getByTestId("pv")).toHaveCount(0);
  });

  test("create broken pv meter with validation failure", async ({ page }) => {
    await page.goto("/#/config");
    await enableExperimental(page, false);

    await expect(page.getByTestId("pv")).toHaveCount(0);

    // create broken meter
    await page.getByRole("button", { name: "Add solar or battery" }).click();

    const meterModal = page.getByTestId("meter-modal");
    await expectModalVisible(meterModal);
    await meterModal.getByRole("button", { name: "Add solar meter" }).click();
    await meterModal.getByLabel("Title").fill("Broken PV");
    await meterModal.getByLabel("Manufacturer").selectOption("SunSpec Inverter");
    await meterModal.getByLabel("IP address or hostname").fill("0.0.0.0");
    await meterModal.getByRole("button", { name: "Validate & save" }).click();

    // wait for validation to complete and check failure
    const testResult = meterModal.getByTestId("test-result");
    await expect(testResult).toContainText("Status: failed");
    await expect(testResult).toContainText("connection refused");

    // verify "Save anyway" button is now visible
    await expect(meterModal.getByRole("button", { name: "Save anyway" })).toBeVisible();

    // save anyway
    await meterModal.getByRole("button", { name: "Save anyway" }).click();
    await expectModalHidden(meterModal);

    // verify broken meter is visible in list
    await expect(page.getByTestId("pv")).toBeVisible();
    await expect(page.getByTestId("pv")).toContainText("Broken PV");
  });
});
