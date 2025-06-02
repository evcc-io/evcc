import { test, expect } from "@playwright/test";
import { start, stop, restart, baseUrl } from "./evcc";
import { startSimulator, stopSimulator, simulatorHost } from "./simulator";
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
    await enableExperimental(page);

    await expect(page.getByTestId("pv")).toHaveCount(0);

    // create #1
    await page.getByRole("button", { name: "Add solar or battery" }).click();

    const meterModal = page.getByTestId("meter-modal");
    await meterModal.getByRole("button", { name: "Add solar meter" }).click();
    await meterModal.getByLabel("Title").fill("PV North");
    await meterModal.getByLabel("Manufacturer").selectOption("Demo meter");
    await meterModal.getByLabel("Power").fill("5000");
    await expect(meterModal.getByRole("button", { name: "Validate & save" })).toBeVisible();
    await meterModal.getByRole("link", { name: "validate" }).click();
    await expect(meterModal.getByTestId("device-tag-power")).toContainText("5.0 kW");
    await meterModal.getByRole("button", { name: "Save" }).click();
    await expectModalHidden(meterModal);
    await expect(page.getByTestId("pv")).toBeVisible(1);
    await expect(page.getByTestId("pv")).toContainText("PV North");

    // edit #1
    await page.getByTestId("pv").getByRole("button", { name: "edit" }).click();
    await expectModalVisible(meterModal);
    await meterModal.getByLabel("Power").fill("6000");
    await meterModal.getByRole("button", { name: "Validate & save" }).click();
    await expectModalHidden(meterModal);

    const pv = page.getByTestId("pv");
    await expect(pv).toBeVisible(1);
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

  test("remove broken pv meter", async ({ page }) => {
    // setup test data for mock openems api
    await startSimulator();

    await page.goto("/#/config");
    await enableExperimental(page);

    // create meter
    await page.getByRole("button", { name: "Add solar or battery" }).click();
    const meterModal = page.getByTestId("meter-modal");
    await meterModal.getByRole("button", { name: "Add solar meter" }).click();
    await meterModal.getByLabel("Title").fill("North Roof");
    await meterModal.getByLabel("Manufacturer").selectOption("shelly-1pm");
    await meterModal.getByLabel("IP address or hostname").fill(simulatorHost());
    await meterModal.getByRole("button", { name: "Validate & save" }).click();
    await expectModalHidden(meterModal);
    await expect(page.getByTestId("pv")).toBeVisible(1);
    await expect(page.getByTestId("pv")).toContainText("North Roof");

    // break meter
    await stopSimulator();
    await restart(CONFIG_GRID_ONLY);
    await page.reload();

    // remove meter
    await expect(page.getByTestId("fatal-error")).toBeVisible();
    await expect(page.getByTestId("pv")).toBeVisible(1);
    await page.getByTestId("pv").getByRole("button", { name: "edit" }).click();
    await expectModalVisible(meterModal);
    await meterModal.getByRole("button", { name: "Delete" }).click();
    await expectModalHidden(meterModal);
    await expect(page.getByTestId("pv")).toHaveCount(0);

    // restart and check again
    await restart(CONFIG_GRID_ONLY);
    await page.reload();
    await expect(page.getByTestId("pv")).toHaveCount(0);
    await expect(page.getByTestId("fatal-error")).not.toBeVisible();
  });
});
