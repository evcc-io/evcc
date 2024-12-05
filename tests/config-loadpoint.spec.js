import { test, expect } from "@playwright/test";
import { start, stop, restart, baseUrl } from "./evcc";

const CONFIG_EMPTY = "config-empty.evcc.yaml";
const CONFIG_ONE_LP = "config-one-lp.evcc.yaml";

test.use({ baseURL: baseUrl() });

test.afterEach(async () => {
  await stop();
});

async function enableExperimental(page) {
  await page
    .getByTestId("generalconfig-experimental")
    .getByRole("button", { name: "edit" })
    .click();
  await page.getByLabel("Experimental 🧪").click();
  await page.getByRole("button", { name: "Close" }).click();
}

async function addDemoCharger(page) {
  const lpModal = page.getByTestId("loadpoint-modal");
  await lpModal.getByRole("button", { name: "Add charger" }).click();

  const modal = page.getByTestId("charger-modal");
  await modal.getByLabel("Manufacturer").selectOption("Demo charger");
  await modal.getByRole("button", { name: "Save" }).click();
  await expect(modal).not.toBeVisible();
}

async function addVehicle(page, title) {
  await page.getByRole("button", { name: "Add vehicle" }).click();
  const modal = page.getByTestId("vehicle-modal");
  await modal.getByLabel("Manufacturer").selectOption("Generic vehicle");
  await modal.getByLabel("Title").fill(title);
  await modal.getByRole("button", { name: "Validate & save" }).click();
  await expect(modal).not.toBeVisible();
}

async function newLoadpoint(page, title) {
  const lpModal = page.getByTestId("loadpoint-modal");
  await page.getByRole("button", { name: "Add charge point" }).click();
  await expect(lpModal).toBeVisible();
  await lpModal.getByLabel("Title").fill(title);
}

test.describe("loadpoint", async () => {
  test("create, update and delete", async ({ page }) => {
    await start(CONFIG_EMPTY);

    await page.goto("/#/config");
    await enableExperimental(page);

    const lpModal = page.getByTestId("loadpoint-modal");
    const chargerModal = page.getByTestId("charger-modal");

    await expect(page.getByTestId("loadpoint")).toHaveCount(0);

    // new loadpoint
    await newLoadpoint(page, "Solar Carport");
    await lpModal.getByRole("button", { name: "Add charger" }).click();

    // add charger
    await chargerModal.getByLabel("Manufacturer").selectOption("Demo charger");
    await chargerModal.getByLabel("Charge status").selectOption("C");
    await chargerModal.getByLabel("Power").fill("11000");
    await chargerModal.getByRole("button", { name: "Save" }).click();
    await expect(lpModal.getByLabel("Title")).toHaveValue("Solar Carport");

    // create loadpoint
    await lpModal.getByRole("button", { name: "Save" }).click();
    await expect(page.getByTestId("loadpoint")).toHaveCount(1);
    await expect(page.getByTestId("loadpoint")).toContainText("Solar Carport");
    await expect(page.getByTestId("loadpoint")).toContainText("charging");
    await expect(page.getByTestId("loadpoint")).toContainText("11.0 kW");

    // restart button appears
    const restartButton = await page
      .getByTestId("bottom-banner")
      .getByRole("button", { name: "Restart" });
    await expect(restartButton).toBeVisible();

    // restart
    await restart(CONFIG_EMPTY);
    await page.reload();
    await expect(page.getByTestId("loadpoint")).toHaveCount(1);
    await expect(page.getByTestId("loadpoint")).toContainText("Solar Carport");

    // update loadpoint title
    await page.getByTestId("loadpoint").getByRole("button", { name: "edit" }).click();
    await lpModal.getByLabel("Title").fill("Solar Carport 2");
    await lpModal.getByRole("button", { name: "Save" }).click();
    await expect(page.getByTestId("loadpoint")).toContainText("Solar Carport 2");

    // restart
    await restart(CONFIG_EMPTY);
    await page.reload();
    await expect(page.getByTestId("loadpoint")).toHaveCount(1);
    await expect(page.getByTestId("loadpoint")).toContainText("Solar Carport 2");

    // update loadpoint power
    await page.getByTestId("loadpoint").getByRole("button", { name: "edit" }).click();
    await lpModal.getByTestId("chargerPower-22kw").click();
    await lpModal.getByRole("button", { name: "Save" }).click();
    await expect(lpModal).not.toBeVisible();

    // restart
    await restart(CONFIG_EMPTY);
    await page.reload();
    await expect(page.getByTestId("loadpoint")).toHaveCount(1);
    await page.getByTestId("loadpoint").getByRole("button", { name: "edit" }).click();
    await expect(lpModal.getByTestId("chargerPower-22kw")).toHaveClass(/active/);
    await expect(lpModal.getByLabel("Title")).toHaveValue("Solar Carport 2");
    await lpModal.getByRole("button", { name: "Close" }).click();

    // delete loadpoint
    await page.getByTestId("loadpoint").getByRole("button", { name: "edit" }).click();
    await lpModal.getByRole("button", { name: "Delete" }).click();
    await expect(page.getByTestId("loadpoint")).toHaveCount(0);

    // restart
    await restart(CONFIG_EMPTY);
    await page.reload();
    await expect(page.getByTestId("loadpoint")).toHaveCount(0);
  });

  test("priority", async ({ page }) => {
    await start(CONFIG_ONE_LP);
    await page.goto("/#/config");
    await enableExperimental(page);

    const lpModal = page.getByTestId("loadpoint-modal");

    // loadpoint from yaml
    await expect(page.getByTestId("loadpoint")).toHaveCount(1);
    await expect(page.getByTestId("loadpoint")).toContainText("Carport");

    // add two more loadpoints
    for (const title of ["Garage", "Garden"]) {
      await newLoadpoint(page, title);
      await addDemoCharger(page);
      await lpModal.getByRole("button", { name: "Save" }).click();
      await expect(lpModal).not.toBeVisible();
    }

    // three loadpoints
    await expect(page.getByTestId("loadpoint")).toHaveCount(3);
    await expect(page.getByTestId("loadpoint").nth(0)).toContainText("Carport");
    await expect(page.getByTestId("loadpoint").nth(1)).toContainText("Garage");
    await expect(page.getByTestId("loadpoint").nth(2)).toContainText("Garden");

    // second loadpoint > priority 2
    await page.getByTestId("loadpoint").nth(1).getByRole("button", { name: "edit" }).click();
    await lpModal.getByTestId("loadpointParamPriority-2").click();
    await lpModal.getByRole("button", { name: "Save" }).click();
    await expect(lpModal).not.toBeVisible();

    // third loadpoint > priority 1
    await page.getByTestId("loadpoint").nth(2).getByRole("button", { name: "edit" }).click();
    await lpModal.getByTestId("loadpointParamPriority-1").click();
    await lpModal.getByRole("button", { name: "Save" }).click();
    await expect(lpModal).not.toBeVisible();
    // restart
    await restart(CONFIG_ONE_LP);
    await page.reload();

    // check priorities
    await page.getByTestId("loadpoint").nth(1).getByRole("button", { name: "edit" }).click();
    await expect(lpModal.getByTestId("loadpointParamPriority-2")).toHaveClass(/active/);
    await lpModal.getByRole("button", { name: "Close" }).click();

    await page.getByTestId("loadpoint").nth(2).getByRole("button", { name: "edit" }).click();
    await expect(lpModal.getByTestId("loadpointParamPriority-1")).toHaveClass(/active/);
    await lpModal.getByRole("button", { name: "Close" }).click();
  });

  test("vehicle", async ({ page }) => {
    await start(CONFIG_EMPTY);
    await page.goto("/#/config");
    await enableExperimental(page);

    const LP_1 = "Carport";
    const LP_2 = "Garage";
    const VEHICLE_1 = "Green Car";
    const VEHICLE_2 = "Yellow Van";

    const lpModal = page.getByTestId("loadpoint-modal");

    // add loadpoint > no vehicle option
    await newLoadpoint(page, LP_1);
    await addDemoCharger(page);
    await expect(lpModal.getByLabel("Default vehicle")).not.toBeVisible();
    await lpModal.getByRole("button", { name: "Save" }).click();
    await expect(lpModal).not.toBeVisible();

    // edit loadpoint
    await page.getByTestId("loadpoint").nth(0).getByRole("button", { name: "edit" }).click();
    await expect(lpModal).toContainText("No vehicles are configured.");
    await lpModal.getByRole("button", { name: "Close" }).click();

    // add vehicles
    for (const title of [VEHICLE_1, VEHICLE_2]) {
      await addVehicle(page, title);
    }

    // set vehicle as default for loadpoint 1
    await page.getByTestId("loadpoint").nth(0).getByRole("button", { name: "edit" }).click();
    await expect(lpModal).toBeVisible();
    await lpModal.getByLabel("Default vehicle").selectOption(VEHICLE_1);
    await lpModal.getByRole("button", { name: "Save" }).click();
    await expect(lpModal).not.toBeVisible();

    // add second loadpoint
    await newLoadpoint(page, LP_2);
    await addDemoCharger(page);
    await lpModal.getByLabel("Default vehicle").selectOption(VEHICLE_2);
    await lpModal.getByRole("button", { name: "Save" }).click();

    // restart
    await restart(CONFIG_EMPTY);
    await page.reload();

    // check loadpoint default vehicles
    for (const [index, vehicle] of [VEHICLE_1, VEHICLE_2].entries()) {
      await page.getByTestId("loadpoint").nth(index).getByRole("button", { name: "edit" }).click();
      await expect(lpModal.locator("#loadpointParamVehicle option:checked")).toHaveText(vehicle);
      await lpModal.getByRole("button", { name: "Close" }).click();
      await expect(lpModal).not.toBeVisible();
    }
  });
});
