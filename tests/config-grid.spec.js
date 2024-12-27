import { test, expect } from "@playwright/test";
import { start, stop, restart, baseUrl } from "./evcc";
import { startSimulator, stopSimulator, simulatorUrl, simulatorHost } from "./simulator";
import { enableExperimental } from "./utils";

const CONFIG_EMPTY = "config-empty.evcc.yaml";

test.use({ baseURL: baseUrl() });

test.beforeAll(async () => {
  await startSimulator();
  await start(CONFIG_EMPTY);
});
test.afterAll(async () => {
  await stop();
  await stopSimulator();
});

test.describe("main screen", async () => {
  test("modes", async ({ page }) => {
    await page.goto("/");
    await expect(page.getByRole("button", { name: "Off" })).toBeVisible();
    await expect(page.getByRole("button", { name: "Fast" })).toBeVisible();
  });
});

test.describe("grid meter", async () => {
  test("create, edit and remove grid meter", async ({ page }) => {
    // setup test data for mock openems api
    await page.goto(simulatorUrl());
    await page.getByLabel("Grid Power").fill("5000");
    await page.getByRole("button", { name: "Apply changes" }).click();

    await page.goto("/#/config");
    await enableExperimental(page);

    await expect(page.getByTestId("grid")).toHaveCount(1);
    await expect(page.getByTestId("grid").getByTestId("device-tag-configured")).toContainText("no");

    // create #1
    await page.getByTestId("grid").getByRole("button", { name: "edit" }).click();

    const meterModal = page.getByTestId("meter-modal");
    await meterModal.getByLabel("Manufacturer").selectOption("OpenEMS");
    await meterModal.getByLabel("IP address or hostname").fill(simulatorHost());
    await expect(meterModal.getByRole("button", { name: "Validate & save" })).toBeVisible();
    await meterModal.getByRole("link", { name: "validate" }).click();
    await expect(meterModal.getByTestId("device-tag-power")).toContainText("5.0 kW");
    await meterModal.getByRole("button", { name: "Save" }).click();
    await expect(meterModal).not.toBeVisible();

    // restart
    await restart(CONFIG_EMPTY);
    await expect(page.getByTestId("grid").getByTestId("device-tag-power")).toContainText("5.0 kW");

    // check in main ui
    await page.goto("/");
    await page.getByTestId("visualization").click();
    await expect(page.getByTestId("energyflow")).toContainText(["Grid use", "5.0 kW"].join(""));

    // delete #1
    await page.goto("/#/config");
    await page.getByTestId("grid").getByRole("button", { name: "edit" }).click();
    await meterModal.getByRole("button", { name: "Delete" }).click();
    await expect(meterModal).not.toBeVisible();

    await expect(page.getByTestId("grid")).toHaveCount(1);
    await expect(page.getByTestId("grid").getByTestId("device-tag-configured")).toContainText("no");
  });
});
