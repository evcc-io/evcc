import { test, expect } from "@playwright/test";
import { start, stop, restart, baseUrl } from "./evcc";
import {
  expectModalVisible,
  expectModalHidden,
  editorClear,
  editorPaste,
  enableExperimental,
  LoadpointType,
  addDemoCharger,
  addDemoMeter,
  addVehicle,
  newLoadpoint,
} from "./utils";

const CONFIG_ONE_LP = "config-one-lp.evcc.yaml";

test.use({ baseURL: baseUrl() });
test.describe.configure({ mode: "parallel" });

test.afterEach(async () => {
  await stop();
});

test.describe("charging loadpoint", async () => {
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
    await expect(lpModal.getByRole("heading", { name: "Edit Charging Point" })).toBeVisible();
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
    await enableExperimental(page, false);

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
    await expectModalHidden(lpModal);
    await expect(page.getByTestId("loadpoint")).toHaveCount(2);

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
    await page.waitForLoadState("networkidle");
    await expect(page.getByRole("button", { name: "Solar", exact: true })).toHaveClass(/active/);

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
    await expectModalVisible(lpModal);
    await expect(lpModal.getByRole("heading", { name: "Edit Charger or Heater" })).toBeVisible();

    // restart without saving loadpoint
    await restart();
    await page.reload();

    // check loadpoint default vehicle
    await page.getByTestId("loadpoint").getByRole("button", { name: "edit" }).click();
    await expectModalVisible(lpModal);
    await lpModal.getByRole("button", { name: "Add charging point" }).click();
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
    await expectModalVisible(meterModal);
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
    await expect(lpModal.getByRole("button", { name: "Add dedicated energy meter" })).toBeVisible();
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
    await chargerModal.getByLabel("Manufacturer").selectOption("User-defined charger");
    await page.waitForLoadState("networkidle");
    const chargerEditor = chargerModal.getByTestId("yaml-editor");
    await expect(chargerEditor).toContainText(
      "status: # charger status (A: not connected, B: connected, C: charging)"
    );

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
    await lpModal.getByRole("button", { name: "Add dedicated energy meter" }).click();
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

    await expect(lpModal.getByLabel("Charger").first()).toHaveValue("User-defined charger [db:1]");
    await expect(lpModal.getByLabel("Energy meter").first()).toHaveValue(
      "User-defined device [db:2]"
    );
    await lpModal.getByLabel("Charger").first().click();
    await expectModalVisible(chargerModal);

    await expect(chargerModal.getByLabel("Manufacturer")).toHaveValue("User-defined charger");
    await page.waitForLoadState("networkidle");
    await expect(chargerEditor).toContainText("value: 'C'");
    await expect(chargerEditor).toContainText("value: 11000");
  });
});

test.describe("heating loadpoint", async () => {
  test("create, update and delete", async ({ page }) => {
    await start();

    await page.goto("/#/config");
    await enableExperimental(page);

    // add loadpoint
    await newLoadpoint(page, "Wärmepumpe", LoadpointType.Heating);
    const lpModal = page.getByTestId("loadpoint-modal");
    await lpModal.getByRole("button", { name: "Add heater" }).click();
    await addDemoCharger(page, LoadpointType.Heating);

    // check heading
    await expect(lpModal.getByRole("heading", { name: "Add Heating Device" })).toBeVisible();
    await lpModal.getByRole("button", { name: "Save" }).click();
    await expectModalHidden(lpModal);

    // restart edit
    await restart();
    await page.reload();
    const lpEntry = page.getByTestId("loadpoint");
    await expect(lpEntry.getByRole("img", { name: "heatpump" })).toBeVisible();
    await expect(lpEntry).toContainText("Wärmepumpe");
    await lpEntry.getByRole("button", { name: "edit" }).click();
    await expectModalVisible(lpModal);
    await expect(lpModal.getByRole("heading", { name: "Edit Heating Device" })).toBeVisible();
    await lpModal.getByRole("button", { name: "Save" }).click();
    await expectModalHidden(lpModal);

    // delete
    await lpEntry.getByRole("button", { name: "edit" }).click();
    await expectModalVisible(lpModal);
    await lpModal.getByRole("button", { name: "Delete" }).click();
    await expectModalHidden(lpModal);

    // restart delete
    await restart();
    await page.reload();

    // check loadpoint
    await expect(page.getByTestId("loadpoint")).not.toBeVisible();
  });

  // add user-defined heat pump
  test("user-defined heat pump", async ({ page }) => {
    await start();
    await page.goto("/#/config");
    await enableExperimental(page);

    // add loadpoint
    await newLoadpoint(page, "Wärmepumpe", LoadpointType.Heating);
    const lpModal = page.getByTestId("loadpoint-modal");
    await lpModal.getByRole("button", { name: "Add heater" }).click();

    // add user-defined heat pump
    const modal = page.getByTestId("charger-modal");
    await expectModalVisible(modal);
    await modal.getByLabel("Manufacturer").selectOption("User-defined heater");
    await modal.getByLabel("Manufacturer").selectOption("User-defined heat pump");
    await modal.getByLabel("Manufacturer").selectOption("User-defined heat pump (sg-ready, all)");
    await modal.getByLabel("Manufacturer").selectOption("User-defined heat pump (sg-ready, boost)");
    await modal.getByLabel("Manufacturer").selectOption("User-defined switch socket");
    await modal.getByLabel("Manufacturer").selectOption("User-defined heat pump");

    const editor = modal.getByTestId("yaml-editor");
    await editorClear(editor, 20);
    await editorPaste(
      editor,
      page,
      `setmaxpower:
  source: js
  script: console.log(setmaxpower); 
getmaxpower:
  source: const
  value: 2000
power:
  source: const
  value: 1000
energy:
  source: const
  value: 0.7
limittemp:
  source: const
  value: 50
temp:
  source: const
  value: 25
`
    );

    const restResult = modal.getByTestId("test-result");
    await expect(restResult).toContainText("Status: unknown");
    await restResult.getByRole("link", { name: "validate" }).click();
    await expect(restResult).toContainText("Status: successful");
    await expect(restResult).toContainText(["Power", "1.0 kW"].join(""));
    await expect(restResult).toContainText(["Energy", "0.7 kWh"].join(""));
    await expect(restResult).toContainText(["Temperature", "25.0°C"].join(""));
    await expect(restResult).toContainText(["Heater limit", "50.0°C"].join(""));

    await modal.getByRole("button", { name: "Save" }).click();
    await expectModalHidden(modal);
    await expectModalVisible(lpModal);

    // create
    await lpModal.getByRole("button", { name: "Save" }).click();
    await expectModalHidden(lpModal);

    await expect(page.getByTestId("loadpoint")).toHaveCount(1);
    const lpEntry = page.getByTestId("loadpoint").first();
    await expect(lpEntry.getByRole("img", { name: "heatpump" })).toBeVisible();
    await expect(lpEntry).toContainText("Wärmepumpe");
    await expect(lpEntry).toContainText(["Power", "1.0 kW"].join(""));
  });

  test("delete charger/heater reverts to loadpoint type select screen", async ({ page }) => {
    await start();
    await page.goto("/#/config");
    await enableExperimental(page);

    // add loadpoint
    await newLoadpoint(page, "Wärmepumpe", LoadpointType.Heating);
    await addDemoCharger(page, LoadpointType.Heating);
    const lpModal = page.getByTestId("loadpoint-modal");
    await lpModal.getByRole("button", { name: "Save" }).click();
    await expectModalHidden(lpModal);

    // delete heater
    await page.getByTestId("loadpoint").getByRole("button", { name: "edit" }).click();
    await expectModalVisible(lpModal);
    await lpModal.getByRole("textbox", { name: "Heater" }).click();
    const chargerModal = page.getByTestId("charger-modal");
    await expectModalVisible(chargerModal);
    await chargerModal.getByRole("button", { name: "Delete" }).click();
    await expectModalHidden(chargerModal);
    await expectModalVisible(lpModal);
    await expect(lpModal.getByRole("heading", { name: "Edit Charger or Heater" })).toBeVisible();
    await expect(lpModal.getByRole("button", { name: "Add charging point" })).toBeVisible();
    await expect(lpModal.getByRole("button", { name: "Add heating device" })).toBeVisible();
  });
});

test.describe("sponsor token", () => {
  test("create charger with missing sponsor token", async ({ page }) => {
    await start();
    await page.goto("/#/config");
    await enableExperimental(page);

    // add loadpoint with OCPP charger
    await newLoadpoint(page, "OCPP Test Charger");
    await page.getByTestId("loadpoint-modal").getByRole("button", { name: "Add charger" }).click();
    const chargerModal = page.getByTestId("charger-modal");
    await expectModalVisible(chargerModal);
    await chargerModal.getByLabel("Manufacturer").selectOption({ label: "OCPP 1.6J compatible" });

    // verify disabled save button
    await expect(chargerModal.getByRole("button", { name: "Save" })).toBeDisabled();

    // verify sponsor notice
    await expect(chargerModal).toContainText(
      "You must configure a sponsor token before you can create this device."
    );
    const testResult = chargerModal.getByTestId("test-result");
    await testResult.getByRole("link", { name: "Validate" }).click();
    const sponsorMessage = testResult.getByText("No sponsor token configured.");
    await expect(sponsorMessage).toBeVisible();

    // verify click on sponsor
    await sponsorMessage.click();
    const sponsorModal = page.getByTestId("sponsor-modal");
    await expectModalVisible(sponsorModal);
    await expect(sponsorModal.getByRole("heading")).toContainText("Sponsorship");
  });
});
