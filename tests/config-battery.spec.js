import { test, expect } from "@playwright/test";
import { start, stop, restart, baseUrl } from "./evcc";
import { startSimulator, stopSimulator, simulatorUrl, simulatorHost } from "./simulator";
import { enableExperimental } from "./utils";

const CONFIG_GRID_ONLY = "config-grid-only.evcc.yaml";

test.use({ baseURL: baseUrl() });

test.beforeAll(async () => {
  await startSimulator();
  await start(CONFIG_GRID_ONLY);
});
test.afterAll(async () => {
  await stop();
  await stopSimulator();
});

test.describe("battery meter", async () => {
  test("create, edit and remove battery meter", async ({ page }) => {
    // setup test data for mock openems api
    await page.goto(simulatorUrl());
    await page.getByLabel("Battery Power").fill("-2500");
    await page.getByLabel("Battery SoC").fill("75");
    await page.getByRole("button", { name: "Apply changes" }).click();

    await page.goto("/#/config");
    await enableExperimental(page);

    await expect(page.getByTestId("battery")).toHaveCount(0);

    // create #1
    await page.getByRole("button", { name: "Add solar or battery" }).click();

    const meterModal = page.getByTestId("meter-modal");
    await meterModal.getByRole("button", { name: "Add battery meter" }).click();
    await meterModal.getByLabel("Manufacturer").selectOption("OpenEMS");
    await meterModal.getByLabel("IP address or hostname").fill(simulatorHost());
    await expect(meterModal.getByRole("button", { name: "Validate & save" })).toBeVisible();
    await meterModal.getByRole("link", { name: "validate" }).click();
    await expect(meterModal.getByTestId("device-tag-soc")).toContainText("75.0%");
    await expect(meterModal.getByTestId("device-tag-power")).toContainText("-2.5 kW");
    await meterModal.getByRole("button", { name: "Save" }).click();
    await expect(page.getByTestId("battery")).toBeVisible(1);
    await expect(page.getByTestId("battery")).toContainText("openems");

    // edit #1
    await page.getByTestId("battery").getByRole("button", { name: "edit" }).click();
    await meterModal.getByLabel("Battery capacity in kWh").fill("20");
    await meterModal.getByRole("button", { name: "Validate & save" }).click();

    const battery = page.getByTestId("battery");
    await expect(battery).toBeVisible(1);
    await expect(battery).toContainText("openems");
    await expect(battery.getByTestId("device-tag-soc")).toContainText("75.0%");
    await expect(battery.getByTestId("device-tag-power")).toContainText("-2.5 kW");
    await expect(battery.getByTestId("device-tag-capacity")).toContainText("20.0 kWh");

    // restart and check in main ui
    await restart(CONFIG_GRID_ONLY);
    await page.goto("/");
    await page.getByTestId("visualization").click();
    await expect(page.getByTestId("energyflow")).toContainText("Battery charging75%2.5 kW");

    // delete #1
    await page.goto("/#/config");
    await page.getByTestId("battery").getByRole("button", { name: "edit" }).click();
    await meterModal.getByRole("button", { name: "Delete" }).click();

    await expect(page.getByTestId("battery")).toHaveCount(0);
  });

  test("advanced fields", async ({ page }) => {
    await page.goto("/#/config");
    await enableExperimental(page);

    await page.getByRole("button", { name: "Add solar or battery" }).click();

    const meterModal = page.getByTestId("meter-modal");
    await meterModal.getByRole("button", { name: "Add battery meter" }).click();
    await meterModal.getByLabel("Manufacturer").selectOption("OpenEMS");
    await expect(meterModal.getByLabel("Password optional")).not.toBeVisible();
    await page.getByRole("button", { name: "Show advanced settings" }).click();
    await expect(meterModal.getByLabel("Password optional")).toBeVisible();
    await page.getByRole("button", { name: "Hide advanced settings" }).click();
    await expect(meterModal.getByLabel("Password optional")).not.toBeVisible();
  });
});
