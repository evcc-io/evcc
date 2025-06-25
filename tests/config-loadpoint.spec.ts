import { test, expect, type Page } from "@playwright/test";
import { start, stop, restart, baseUrl } from "./evcc";
import { expectModalVisible, expectModalHidden, editorClear, editorPaste } from "./utils";

const CONFIG_ONE_LP = "config-one-lp.evcc.yaml";

test.use({ baseURL: baseUrl() });

test.afterEach(async () => {
  await stop();
});

async function enableExperimental(page: Page) {
  await page
    .getByTestId("generalconfig-experimental")
    .getByRole("button", { name: "edit" })
    .click();
  await page.getByLabel("Experimental 🧪").click();
  await page.getByRole("button", { name: "Close" }).click();
}

async function addDemoCharger(page: Page) {
  const lpModal = page.getByTestId("loadpoint-modal");
  await lpModal.getByRole("button", { name: "Add charger" }).click();

  const modal = page.getByTestId("charger-modal");
  await expectModalVisible(modal);
  await modal.getByLabel("Manufacturer").selectOption("Demo charger");
  await modal.getByRole("button", { name: "Save" }).click();
  await expectModalHidden(modal);
  await expectModalVisible(lpModal);
}

async function addDemoMeter(page: Page, power = "0") {
  const lpModal = page.getByTestId("loadpoint-modal");
  await lpModal.getByRole("button", { name: "Add dedicated charger meter" }).click();

  const modal = page.getByTestId("meter-modal");
  await expectModalVisible(modal);
  await modal.getByLabel("Manufacturer").selectOption("Demo meter");
  await modal.getByLabel("Power").fill(power);
  await modal.getByRole("button", { name: "Save" }).click();
  await expectModalHidden(modal);
  await expectModalVisible(lpModal);
}

async function addVehicle(page: Page, title: string) {
  await page.getByRole("button", { name: "Add vehicle" }).click();
  const modal = page.getByTestId("vehicle-modal");
  await expectModalVisible(modal);
  await modal.getByLabel("Manufacturer").selectOption("Generic vehicle (without API)");
  await modal.getByLabel("Title").fill(title);
  await modal.getByRole("button", { name: "Validate & save" }).click();
  await expectModalHidden(modal);
}

async function newLoadpoint(page: Page, title: string) {
  const lpModal = page.getByTestId("loadpoint-modal");
  await page.getByRole("button", { name: "Add charge point" }).click();
  await expectModalVisible(lpModal);
  await lpModal.getByLabel("Title").fill(title);
}

test.describe("loadpoint", async () => {
  test("create, update and delete", async ({ page }) => {
    await start();

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
    await restart();
    await page.reload();
    await expect(page.getByTestId("loadpoint")).toHaveCount(1);
    await expect(page.getByTestId("loadpoint")).toContainText("Solar Carport");

    // update loadpoint title
    await page.getByTestId("loadpoint").getByRole("button", { name: "edit" }).click();
    await expectModalVisible(lpModal);
    await lpModal.getByLabel("Title").fill("Solar Carport 2");
    await lpModal.getByRole("button", { name: "Save" }).click();
    await expect(page.getByTestId("loadpoint")).toContainText("Solar Carport 2");

    // restart
    await restart();
    await page.reload();
    await expect(page.getByTestId("loadpoint")).toHaveCount(1);
    await expect(page.getByTestId("loadpoint")).toContainText("Solar Carport 2");

    // update loadpoint power
    await page.getByTestId("loadpoint").getByRole("button", { name: "edit" }).click();
    await expectModalVisible(lpModal);
    await lpModal.getByTestId("chargerPower-22kw").click();

    // update charger mode
    await expect(lpModal.getByRole("textbox", { name: "Charger" })).toHaveValue(
      "Demo charger [db:1]"
    );
    await lpModal.getByRole("textbox", { name: "Charger" }).click();
    await chargerModal.getByLabel("Charge status").selectOption("A");
    await chargerModal.getByRole("button", { name: "Save" }).click();
    await expectModalHidden(chargerModal);

    await lpModal.getByRole("button", { name: "Save" }).click();
    await expectModalHidden(lpModal);

    // restart
    await restart();
    await page.reload();
    await expect(page.getByTestId("loadpoint")).toHaveCount(1);
    await expect(page.getByTestId("loadpoint")).toContainText("not connected");

    await page.getByTestId("loadpoint").getByRole("button", { name: "edit" }).click();
    await expectModalVisible(lpModal);
    await expect(lpModal.getByTestId("chargerPower-22kw")).toHaveClass(/active/);
    await expect(lpModal.getByLabel("Title")).toHaveValue("Solar Carport 2");
    await lpModal.getByRole("button", { name: "Close" }).click();

    // delete loadpoint
    await page.getByTestId("loadpoint").getByRole("button", { name: "edit" }).click();
    await expectModalVisible(lpModal);
    await lpModal.getByRole("button", { name: "Delete" }).click();
    await expect(page.getByTestId("loadpoint")).toHaveCount(0);

    // restart
    await restart();
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
    await expectModalHidden(lpModal);

    // two loadpoints
    await expect(page.getByTestId("loadpoint")).toHaveCount(2);
    await expect(page.getByTestId("loadpoint").nth(0)).toContainText("Carport");
    await expect(page.getByTestId("loadpoint").nth(1)).toContainText("Garage");

    // second loadpoint: increase priority
    await page.getByTestId("loadpoint").nth(1).getByRole("button", { name: "edit" }).click();
    await expectModalVisible(lpModal);
    await expect(lpModal.getByLabel("Priority")).toHaveValue("0 (default)");
    await lpModal.getByLabel("Priority").selectOption("1");
    await lpModal.getByRole("button", { name: "Save" }).click();
    await expectModalHidden(lpModal);

    // restart
    await restart(CONFIG_ONE_LP);
    await page.reload();

    // check priorities
    await page.getByTestId("loadpoint").nth(1).getByRole("button", { name: "edit" }).click();
    await expectModalVisible(lpModal);
    await expect(lpModal.getByLabel("Priority")).toHaveValue("1");

    // change back to 0
    await lpModal.getByLabel("Priority").selectOption("0 (default)");
    await lpModal.getByRole("button", { name: "Save" }).click();
    await expectModalHidden(lpModal);

    // restart
    await restart(CONFIG_ONE_LP);
    await page.reload();

    // check priorities
    await page.getByTestId("loadpoint").nth(1).getByRole("button", { name: "edit" }).click();
    await expectModalVisible(lpModal);
    await expect(lpModal.getByLabel("Priority")).toHaveValue("0 (default)");
  });

  test("vehicle", async ({ page }) => {
    await start();
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
    await expectModalHidden(lpModal);

    // edit loadpoint
    await page.getByTestId("loadpoint").nth(0).getByRole("button", { name: "edit" }).click();
    await expectModalVisible(lpModal);
    await expect(lpModal).toContainText("No vehicles are configured.");
    await lpModal.getByRole("button", { name: "Close" }).click();

    // add vehicles
    for (const title of [VEHICLE_1, VEHICLE_2]) {
      await addVehicle(page, title);
    }

    // set vehicle as default for loadpoint 1
    await page.getByTestId("loadpoint").nth(0).getByRole("button", { name: "edit" }).click();
    await expectModalVisible(lpModal);
    await expect(lpModal.getByTestId("loadpointPollMode-charging")).toHaveClass(/active/);
    await expect(lpModal.getByRole("checkbox", { name: "Interpolate charge level" })).toBeChecked();
    await lpModal.getByLabel("Default vehicle").selectOption(VEHICLE_1);
    await lpModal.getByRole("button", { name: "Save" }).click();
    await expectModalHidden(lpModal);

    // add second loadpoint
    await newLoadpoint(page, LP_2);
    await addDemoCharger(page);
    await lpModal.getByLabel("Default vehicle").selectOption(VEHICLE_2);
    await lpModal.getByRole("button", { name: "Save" }).click();

    // restart
    await restart();
    await page.reload();

    // check loadpoint default vehicles
    for (const [index, vehicle] of [VEHICLE_1, VEHICLE_2].entries()) {
      await page.getByTestId("loadpoint").nth(index).getByRole("button", { name: "edit" }).click();
      await expectModalVisible(lpModal);
      await expect(lpModal.locator("#loadpointParamVehicle option:checked")).toHaveText(vehicle);
      await lpModal.getByRole("button", { name: "Close" }).click();
      await expectModalHidden(lpModal);
    }
  });

  test("keep mode", async ({ page }) => {
    await start();
    await page.goto("/#/config");
    await enableExperimental(page);

    // add grid meter
    await page.getByRole("button", { name: "Add grid meter" }).click();
    const meterModal = page.getByTestId("meter-modal");
    await meterModal.getByLabel("Manufacturer").selectOption("Demo meter");
    await meterModal.getByLabel("Power").fill("-1000");
    await meterModal.getByRole("button", { name: "Save" }).click();
    await expectModalHidden(meterModal);

    // add a loadpoint with dummy charger,
    const lpModal = page.getByTestId("loadpoint-modal");
    await newLoadpoint(page, "Carport");
    await addDemoCharger(page);
    await lpModal.getByLabel("Default mode").selectOption("---");
    await lpModal.getByRole("button", { name: "Save" }).click();
    await expectModalHidden(lpModal);
    await restart();

    // change on main ui
    await page.goto("/");
    await expect(page.getByRole("button", { name: "Off" })).toHaveClass(/active/);
    await page.getByRole("button", { name: "Solar", exact: true }).click();
    await restart();
    await page.reload();
    await expect(page.getByRole("button", { name: "Solar", exact: true })).toHaveClass(/active/);

    // change default mode in config to fast
    await page.goto("/#/config");
    // open first loadpoint
    await page.getByTestId("loadpoint").getByRole("button", { name: "edit" }).click();
    await expectModalVisible(lpModal);
    await lpModal.getByLabel("Default mode").selectOption("Fast");
    await lpModal.getByRole("button", { name: "Save" }).click();
    await expectModalHidden(lpModal);
    await restart();

    // check loadpoint mode
    await page.goto("/");
    await expect(page.getByRole("button", { name: "Fast" })).toHaveClass(/active/);
  });

  test("delete vehicle references", async ({ page }) => {
    await start();
    await page.goto("/#/config");
    await enableExperimental(page);

    // add vehicle, add loadpoint, add charger
    await addVehicle(page, "Porsche");
    await addVehicle(page, "Tesla");
    await newLoadpoint(page, "Garage");
    await addDemoCharger(page);
    const lpModal = page.getByTestId("loadpoint-modal");
    await expectModalVisible(lpModal);
    await lpModal.getByLabel("Default vehicle").selectOption("Porsche");
    await lpModal.getByRole("button", { name: "Save" }).click();
    await expectModalHidden(lpModal);

    // delete vehicle
    await page.getByTestId("vehicle").nth(0).getByRole("button", { name: "edit" }).click();
    const vehicleModal = page.getByTestId("vehicle-modal");
    await expectModalVisible(vehicleModal);
    await vehicleModal.getByRole("button", { name: "Delete" }).click();
    await expectModalHidden(vehicleModal);

    // restart
    await restart();
    await page.reload();

    // check loadpoint default vehicle
    await page.getByTestId("loadpoint").getByRole("button", { name: "edit" }).click();
    await expectModalVisible(lpModal);
    await expect(lpModal.getByLabel("Default vehicle")).toHaveValue("");
  });

  test("delete charger references", async ({ page }) => {
    await start();
    await page.goto("/#/config");
    await enableExperimental(page);

    // add loadpoint, add charger
    await newLoadpoint(page, "Garage");
    await addDemoCharger(page);
    const lpModal = page.getByTestId("loadpoint-modal");
    await lpModal.getByRole("button", { name: "Save" }).click();
    await expectModalHidden(lpModal);

    // delete charger
    await page.getByTestId("loadpoint").getByRole("button", { name: "edit" }).click();
    await expectModalVisible(lpModal);
    await lpModal.getByRole("textbox", { name: "Charger" }).click();
    const chargerModal = page.getByTestId("charger-modal");
    await expectModalVisible(chargerModal);
    await chargerModal.getByRole("button", { name: "Delete" }).click();
    await expectModalHidden(chargerModal);

    // restart without saving loadpoint
    await restart();
    await page.reload();

    // check loadpoint default vehicle
    await page.getByTestId("loadpoint").getByRole("button", { name: "edit" }).click();
    await expectModalVisible(lpModal);
    await expect(lpModal.getByRole("textbox", { name: "Title" })).toHaveValue("Garage");
    await expect(lpModal).toContainText("Configuring a charger is required.");
  });

  test("delete meter references", async ({ page }) => {
    await start();
    await page.goto("/#/config");
    await enableExperimental(page);

    // add loadpoint, add charger
    await newLoadpoint(page, "Garage");
    await addDemoCharger(page);
    await addDemoMeter(page, "11000");
    const lpModal = page.getByTestId("loadpoint-modal");
    await lpModal.getByRole("button", { name: "Save" }).click();
    await expectModalHidden(lpModal);
    await expect(page.getByTestId("loadpoint")).toContainText("11.0 kW");

    // delete charger
    await page.getByTestId("loadpoint").getByRole("button", { name: "edit" }).click();
    await expectModalVisible(lpModal);
    await lpModal.getByRole("textbox", { name: "Meter" }).click();
    const meterModal = page.getByTestId("meter-modal");
    await meterModal.getByRole("button", { name: "Delete" }).click();
    await expectModalHidden(meterModal);

    // restart without saving loadpoint
    await restart();
    await page.reload();

    // check loadpoint default vehicle
    await expect(page.getByTestId("loadpoint")).not.toContainText("11.0 kW");
    await page.getByTestId("loadpoint").getByRole("button", { name: "edit" }).click();
    await expectModalVisible(lpModal);
    await expect(lpModal.getByRole("textbox", { name: "Title" })).toHaveValue("Garage");
    await expect(
      lpModal.getByRole("button", { name: "Add dedicated charger meter" })
    ).toBeVisible();
  });

  test("user-defined charger", async ({ page }) => {
    await start();
    await page.goto("/#/config");
    await enableExperimental(page);

    // add loadpoint
    await newLoadpoint(page, "Carport");
    const lpModal = page.getByTestId("loadpoint-modal");
    await lpModal.getByRole("button", { name: "Add charger" }).click();

    // add user-defined charger
    const chargerModal = page.getByTestId("charger-modal");
    await expectModalVisible(chargerModal);
    await chargerModal.getByLabel("Manufacturer").selectOption("User-defined device");
    await page.waitForLoadState("networkidle");
    const chargerEditor = chargerModal.getByTestId("yaml-editor");
    await expect(chargerEditor).toContainText("status: # charger status [A..F]");

    await editorClear(chargerEditor, 20);
    await editorPaste(
      chargerEditor,
      page,
      `status:
  source: const
  value: 'C'
enabled:
  source: const
  value: true
enable:
  source: js
  script: console.log(enable)
maxcurrent:
  source: js
  script: console.log(maxcurrent)
power:
  source: const
  value: 11000`
    );

    const chargerRestResult = chargerModal.getByTestId("test-result");
    await expect(chargerRestResult).toContainText("Status: unknown");
    await chargerRestResult.getByRole("link", { name: "validate" }).click();
    await expect(chargerRestResult).toContainText("Status: successful");
    await expect(chargerRestResult).toContainText(["Status", "charging"].join(""));
    await expect(chargerRestResult).toContainText(["Enabled", "yes"].join(""));
    await expect(chargerRestResult).toContainText(["Power", "11.0 kW"].join(""));
    await chargerModal.getByRole("button", { name: "Save" }).click();
    await expectModalHidden(chargerModal);
    await expectModalVisible(lpModal);

    // add user-defined meter
    await lpModal.getByRole("button", { name: "Add dedicated charger meter" }).click();
    const meterModal = page.getByTestId("meter-modal");
    await expectModalVisible(meterModal);
    await meterModal.getByLabel("Manufacturer").selectOption("User-defined device");
    await page.waitForLoadState("networkidle");
    const meterEditor = meterModal.getByTestId("yaml-editor");
    await editorClear(meterEditor, 20);
    await editorPaste(
      meterEditor,
      page,
      `power:
  source: const
  value: 5000`
    );

    const meterRestResult = meterModal.getByTestId("test-result");
    await expect(meterRestResult).toContainText("Status: unknown");
    await meterRestResult.getByRole("link", { name: "validate" }).click();
    await expect(meterRestResult).toContainText("Status: successful");
    await expect(meterRestResult).toContainText(["Power", "5.0 kW"].join(""));
    await meterModal.getByRole("button", { name: "Save" }).click();
    await expectModalHidden(meterModal);

    // create
    await lpModal.getByRole("button", { name: "Save" }).click();
    await expectModalHidden(lpModal);

    await expect(page.getByTestId("loadpoint")).toHaveCount(1);
    await expect(page.getByTestId("loadpoint")).toContainText("Carport");

    // restart evcc
    await restart();
    await page.reload();

    const lpEntry = page.getByTestId("loadpoint");
    await expect(lpEntry).toHaveCount(1);
    await expect(lpEntry).toContainText("Carport");
    await expect(lpEntry).toContainText(["Status", "charging"].join(""));
    await expect(lpEntry).toContainText(["Enabled", "yes"].join(""));
    await expect(lpEntry).toContainText(["Power", "5.0 kW"].join(""));

    await lpEntry.getByRole("button", { name: "edit" }).click();
    await expectModalVisible(lpModal);

    await expect(lpModal.getByLabel("Charger").first()).toHaveValue("User-defined device [db:1]");
    await expect(lpModal.getByLabel("Energy meter").first()).toHaveValue(
      "User-defined device [db:2]"
    );
    await lpModal.getByLabel("Charger").first().click();
    await expectModalVisible(chargerModal);

    await expect(chargerModal.getByLabel("Manufacturer")).toHaveValue("User-defined device");
    await page.waitForLoadState("networkidle");
    await expect(chargerEditor).toContainText("value: 'C'");
    await expect(chargerEditor).toContainText("value: 11000");
  });
});
