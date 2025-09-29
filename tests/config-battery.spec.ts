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

test.describe("battery meter", async () => {
  test("create, edit and remove battery meter", async ({ page }) => {
    await page.goto("/#/config");
    await enableExperimental(page, false);

    await expect(page.getByTestId("battery")).toHaveCount(0);

    // create #1
    await page.getByRole("button", { name: "Add solar or battery" }).click();

    const meterModal = page.getByTestId("meter-modal");
    await meterModal.getByRole("button", { name: "Add battery meter" }).click();
    await meterModal.getByLabel("Title").fill("Demo Battery");
    await meterModal.getByLabel("Manufacturer").selectOption("Demo battery");
    await meterModal.getByLabel("Power").fill("4000");
    await meterModal.getByLabel("Charge").fill("80");
    await expect(meterModal.getByLabel("Maximum AC power of the hybrid inverter")).toHaveCount(0);
    await expect(meterModal.getByRole("button", { name: "Validate & save" })).toBeVisible();
    await meterModal.getByRole("link", { name: "validate" }).click();
    await expect(meterModal.getByTestId("device-tag-power")).toContainText("4.0 kW");
    await expect(meterModal.getByTestId("device-tag-soc")).toContainText("80.0%");
    await meterModal.getByRole("button", { name: "Save" }).click();
    await expectModalHidden(meterModal);
    await expect(page.getByTestId("battery")).toBeVisible();
    await expect(page.getByTestId("battery")).toContainText("Demo Battery");

    // edit #1
    await page.getByTestId("battery").getByRole("button", { name: "edit" }).click();
    await expectModalVisible(meterModal);
    await meterModal.getByLabel("Power").fill("5000");
    await meterModal.getByRole("button", { name: "Validate & save" }).click();
    await expectModalHidden(meterModal);

    const battery = page.getByTestId("battery");
    await expect(battery).toBeVisible();
    await expect(battery).toContainText("Demo Battery");
    await expect(battery.getByTestId("device-tag-power")).toContainText("5.0 kW");
    await expect(battery.getByTestId("device-tag-soc")).toContainText("80.0%");

    // restart and check in main ui
    await restart(CONFIG_GRID_ONLY);
    await page.goto("/");
    await page.getByTestId("visualization").click();
    await expect(page.getByTestId("energyflow")).toContainText("Battery discharging80%5.0 kW");

    // delete #1
    await page.goto("/#/config");
    await page.getByTestId("battery").getByRole("button", { name: "edit" }).click();
    await expectModalVisible(meterModal);
    await meterModal.getByRole("button", { name: "Delete" }).click();
    await expectModalHidden(meterModal);
    await expect(page.getByTestId("battery")).toHaveCount(0);
  });

  test("advanced fields", async ({ page }) => {
    await page.goto("/#/config");
    await enableExperimental(page, false);

    await page.getByRole("button", { name: "Add solar or battery" }).click();

    const meterModal = page.getByTestId("meter-modal");
    await meterModal.getByRole("button", { name: "Add battery meter" }).click();
    await meterModal.getByLabel("Title").fill("Demo Battery");
    await meterModal.getByLabel("Manufacturer").selectOption("Demo battery");

    await expect(meterModal.getByLabel("Meter reading")).not.toBeVisible();
    await page.getByRole("button", { name: "Show advanced settings" }).click();
    await expect(meterModal.getByLabel("Meter reading")).toBeVisible();
    await page.getByRole("button", { name: "Hide advanced settings" }).click();
    await expect(meterModal.getByLabel("Meter reading")).not.toBeVisible();
  });
});
