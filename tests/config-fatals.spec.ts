import { test, expect } from "@playwright/test";
import { start, stop, restart, baseUrl } from "./evcc";
import { startSimulator, stopSimulator, simulatorHost } from "./simulator";
import {
  expectModalVisible,
  expectModalHidden,
  enableExperimental,
  addDemoCharger,
  newLoadpoint,
} from "./utils";

test.use({ baseURL: baseUrl() });

test.afterEach(async () => {
  await stop();
});

test.describe("fatal config handling", async () => {
  test("broken pv meter", async ({ page }) => {
    await startSimulator();
    await start();

    await page.goto("/#/config");
    await enableExperimental(page, false);

    // create meter
    await page.getByRole("button", { name: "Add solar or battery" }).click();
    const meterModal = page.getByTestId("meter-modal");
    await meterModal.getByRole("button", { name: "Add solar meter" }).click();
    await meterModal.getByLabel("Title").fill("North Roof");
    await meterModal.getByLabel("Manufacturer").selectOption("shelly-1pm");
    await meterModal.getByLabel("IP address or hostname").fill(simulatorHost());
    await meterModal.getByRole("button", { name: "Validate & save" }).click();
    await expectModalHidden(meterModal);
    await expect(page.getByTestId("pv")).toBeVisible();
    await expect(page.getByTestId("pv")).toContainText("North Roof");

    // break meter
    await stopSimulator();
    await restart();
    await page.reload();

    // remove meter
    await expect(page.getByTestId("fatal-error")).toBeVisible();
    await expect(page.getByTestId("pv")).toBeVisible();
    await page.getByTestId("pv").getByRole("button", { name: "edit" }).click();
    await expectModalVisible(meterModal);
    await meterModal.getByRole("button", { name: "Delete" }).click();
    await expectModalHidden(meterModal);
    await expect(page.getByTestId("pv")).toHaveCount(0);

    // restart and check again
    await restart();
    await page.reload();
    await expect(page.getByTestId("pv")).toHaveCount(0);
    await expect(page.getByTestId("fatal-error")).not.toBeVisible();
  });

  test("broken loadpoint meter", async ({ page }) => {
    await startSimulator();
    await start();

    await page.goto("/#/config");
    await enableExperimental(page, false);

    const lpModal = page.getByTestId("loadpoint-modal");

    // create loadpoint with demo charger and shelly meter that will break
    await newLoadpoint(page, "Test Carport");
    await addDemoCharger(page);

    // add shelly meter
    await lpModal.getByRole("button", { name: "Add dedicated energy meter" }).click();
    const meterModal = page.getByTestId("meter-modal");
    await expectModalVisible(meterModal);
    await meterModal.getByLabel("Manufacturer").selectOption("shelly-1pm");
    await meterModal.getByLabel("IP address or hostname").fill(simulatorHost());
    await meterModal.getByRole("button", { name: "Validate & save" }).click();
    await expectModalHidden(meterModal);

    await lpModal.getByRole("button", { name: "Save" }).click();
    await expectModalHidden(lpModal);
    await expect(page.getByTestId("loadpoint")).toContainText("Test Carport");
    await page.waitForLoadState("networkidle");

    // break meter
    await stopSimulator();
    await restart();
    await page.reload();

    // verify loadpoint still visible with error
    await expect(page.getByTestId("fatal-error")).toBeVisible();
    await expect(page.getByTestId("fatal-error")).toContainText(
      /meter: .+? cannot create meter .+?: cannot create meter type 'template': cannot create meter type 'shelly'/
    );
    await expect(page.getByTestId("fatal-error")).toContainText(
      /loadpoint: .+? missing charge meter instance/
    );
    await expect(page.getByTestId("loadpoint")).toBeVisible();

    // open modal and delete meter
    await page.getByTestId("loadpoint").getByRole("button", { name: "edit" }).click();
    await expectModalVisible(lpModal);
    await expect(lpModal.getByRole("textbox", { name: "Energy meter" })).toHaveClass(/is-invalid/);
    await lpModal.getByRole("textbox", { name: "Energy meter" }).click();
    await expectModalVisible(meterModal);
    await meterModal.getByRole("button", { name: "Delete" }).click();
    await expectModalHidden(meterModal);
    await lpModal.getByRole("button", { name: "Save" }).click();
    await expectModalHidden(lpModal);
    await page.waitForLoadState("networkidle");

    // restart and verify
    await restart();
    await page.reload();
    await expect(page.getByTestId("loadpoint")).toBeVisible();
    await expect(page.getByTestId("fatal-error")).not.toBeVisible(); // error should be gone
  });

  test("broken grid meter", async ({ page }) => {
    // setup test data for mock api
    await startSimulator();
    await start();

    await page.goto("/#/config");
    await enableExperimental(page, false);

    // create grid meter
    await page.getByRole("button", { name: "Add grid meter" }).click();
    const meterModal = page.getByTestId("meter-modal");
    await meterModal.getByLabel("Manufacturer").selectOption("shelly-1pm");
    await meterModal.getByLabel("IP address or hostname").fill(simulatorHost());
    await meterModal.getByRole("button", { name: "Validate & save" }).click();
    await expectModalHidden(meterModal);
    await expect(page.getByTestId("grid")).toBeVisible();

    // break meter
    await stopSimulator();
    await restart();
    await page.reload();

    // verify grid meter still visible with error
    await expect(page.getByTestId("fatal-error")).toBeVisible();
    await expect(page.getByTestId("grid")).toBeVisible();

    // open modal and delete meter
    await page.getByTestId("grid").getByRole("button", { name: "edit" }).click();
    await expectModalVisible(meterModal);
    await meterModal.getByRole("button", { name: "Delete" }).click();
    await expectModalHidden(meterModal);
    await expect(page.getByTestId("grid")).toHaveCount(0);

    // restart and verify
    await restart();
    await page.reload();
    await expect(page.getByTestId("grid")).toHaveCount(0);
    await expect(page.getByTestId("fatal-error")).not.toBeVisible(); // error should be gone
  });
});
