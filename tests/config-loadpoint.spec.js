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

    // add loadpoint via UI
    await newLoadpoint(page, "Garage");
    await addDemoCharger(page);
    await lpModal.getByRole("button", { name: "Save" }).click();
    await expect(lpModal).not.toBeVisible();

    // two loadpoints
    await expect(page.getByTestId("loadpoint")).toHaveCount(2);
    await expect(page.getByTestId("loadpoint").nth(0)).toContainText("Carport");
    await expect(page.getByTestId("loadpoint").nth(1)).toContainText("Garage");

    // second loadpoint: increase priority
    await page.getByTestId("loadpoint").nth(1).getByRole("button", { name: "edit" }).click();
    await expect(lpModal).toBeVisible();
    await expect(lpModal.getByLabel("Priority")).toHaveValue("0 (default)");
    await lpModal.getByLabel("Priority").selectOption("1");
    await lpModal.getByRole("button", { name: "Save" }).click();
    await expect(lpModal).not.toBeVisible();

    // restart
    await restart(CONFIG_ONE_LP);
    await page.reload();

    // check priorities
    await page.getByTestId("loadpoint").nth(1).getByRole("button", { name: "edit" }).click();
    await expect(lpModal.getByLabel("Priority")).toHaveValue("1");
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

  test("keep mode", async ({ page }) => {
    await start(CONFIG_EMPTY);
    await page.goto("/#/config");
    await enableExperimental(page);

    // add grid meter
    await page.getByRole("button", { name: "Add grid meter" }).click();
    const meterModal = page.getByTestId("meter-modal");
    await meterModal.getByLabel("Manufacturer").selectOption("Demo meter");
    await meterModal.getByLabel("Power").fill("-1000");
    await meterModal.getByRole("button", { name: "Save" }).click();
    await expect(meterModal).not.toBeVisible();

    // add a loadpoint with dummy charger,
    const lpModal = page.getByTestId("loadpoint-modal");
    await newLoadpoint(page, "Carport");
    await addDemoCharger(page);
    await lpModal.getByLabel("Default mode").selectOption("---");
    await lpModal.getByRole("button", { name: "Save" }).click();
    await expect(lpModal).not.toBeVisible();
    await restart(CONFIG_EMPTY);

    // change on main ui
    await page.goto("/");
    await expect(page.getByRole("button", { name: "Off" })).toHaveClass(/active/);
    await page.getByRole("button", { name: "Solar", exact: true }).click();
    await restart(CONFIG_EMPTY);
    await page.reload();
    await expect(page.getByRole("button", { name: "Solar", exact: true })).toHaveClass(/active/);

    // change default mode in config to fast
    await page.goto("/#/config");
    // open first loadpoint
    await page.getByTestId("loadpoint").getByRole("button", { name: "edit" }).click();
    await expect(lpModal).toBeVisible();
    await lpModal.getByLabel("Default mode").selectOption("Fast");
    await lpModal.getByRole("button", { name: "Save" }).click();
    await expect(lpModal).not.toBeVisible();
    await restart(CONFIG_EMPTY);

    // check loadpoint mode
    await page.goto("/");
    await expect(page.getByRole("button", { name: "Fast" })).toHaveClass(/active/);
  });
});
