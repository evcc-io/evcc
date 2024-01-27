const { test, expect } = require("@playwright/test");
const { start, stop, restart, cleanRestart } = require("./evcc");
const { startSimulator, stopSimulator, SIMULATOR_URL, SIMULATOR_HOST } = require("./simulator");

const CONFIG_EMPTY = "config-empty.evcc.yaml";
const CONFIG_WITH_VEHICLE = "config-with-vehicle.evcc.yaml";

test.beforeAll(async () => {
  await start(CONFIG_EMPTY);
});
test.afterAll(async () => {
  await stop();
});

test.describe("basics", async () => {
  test("navigation to config", async ({ page }) => {
    await page.goto("/");
    await page.getByTestId("topnavigation-button").click();
    await page.getByRole("button", { name: "Settings" }).click();
    await page.getByLabel("Experimental 🧪").click();
    await page.getByRole("button", { name: "Close" }).click();
    await page.getByTestId("topnavigation-button").click();
    await page.getByRole("link", { name: "Configuration" }).click();
    await expect(page.getByRole("heading", { name: "Configuration" })).toBeVisible();
  });
  test("alert box should always be visible", async ({ page }) => {
    await page.goto("/#/config");
    await expect(page.getByRole("alert")).toBeVisible();
  });
});

test.describe("vehicles", async () => {
  test("create, edit and delete vehicles", async ({ page }) => {
    await page.goto("/#/config");

    await expect(page.getByTestId("vehicle")).toHaveCount(0);
    const vehicleModal = page.getByTestId("vehicle-modal");

    // create #1
    await page.getByTestId("add-vehicle").click();
    await vehicleModal.getByLabel("Manufacturer").selectOption("Generic vehicle");
    await vehicleModal.getByLabel("Title").fill("Green Car");
    await vehicleModal.getByRole("button", { name: "Validate & save" }).click();

    await expect(page.getByTestId("vehicle")).toHaveCount(1);

    // create #2
    await page.getByTestId("add-vehicle").click();
    await vehicleModal.getByLabel("Manufacturer").selectOption("Generic vehicle");
    await vehicleModal.getByLabel("Title").fill("Yellow Van");
    await vehicleModal.getByRole("button", { name: "Validate & save" }).click();

    await expect(page.getByTestId("vehicle")).toHaveCount(2);
    await expect(page.getByTestId("vehicle").nth(0)).toHaveText(/Green Car/);
    await expect(page.getByTestId("vehicle").nth(1)).toHaveText(/Yellow Van/);

    // edit #1
    await page.getByTestId("vehicle").nth(0).getByRole("button", { name: "edit" }).click();
    await expect(vehicleModal.getByLabel("Title")).toHaveValue("Green Car");
    await vehicleModal.getByLabel("Title").fill("Fancy Car");
    await vehicleModal.getByRole("button", { name: "Validate & save" }).click();

    await expect(page.getByTestId("vehicle")).toHaveCount(2);
    await expect(page.getByTestId("vehicle").nth(0)).toHaveText(/Fancy Car/);

    // delete #1
    await page.getByTestId("vehicle").nth(0).getByRole("button", { name: "edit" }).click();
    await vehicleModal.getByRole("button", { name: "Delete" }).click();

    await expect(page.getByTestId("vehicle")).toHaveCount(1);
    await expect(page.getByTestId("vehicle").nth(0)).toHaveText(/Yellow Van/);

    // delete #2
    await page.getByTestId("vehicle").nth(0).getByRole("button", { name: "edit" }).click();
    await vehicleModal.getByRole("button", { name: "Delete" }).click();

    await expect(page.getByTestId("vehicle")).toHaveCount(0);
  });

  test("config should survive restart", async ({ page }) => {
    await page.goto("/#/config");

    await expect(page.getByTestId("vehicle")).toHaveCount(0);
    const vehicleModal = page.getByTestId("vehicle-modal");

    // create #1 & #2
    await page.getByTestId("add-vehicle").click();
    await vehicleModal.getByLabel("Manufacturer").selectOption("Generic vehicle");
    await vehicleModal.getByLabel("Title").fill("Green Car");
    await vehicleModal.getByRole("button", { name: "Validate & save" }).click();

    await page.getByTestId("add-vehicle").click();
    await vehicleModal.getByLabel("Manufacturer").selectOption("Generic vehicle");
    await vehicleModal.getByLabel("Title").fill("Yellow Van");
    await vehicleModal.getByLabel("car").click();
    await vehicleModal.getByLabel("van").check();
    await vehicleModal.getByRole("button", { name: "Validate & save" }).click();

    await expect(page.getByTestId("vehicle")).toHaveCount(2);

    // restart evcc
    await restart(CONFIG_EMPTY);
    await page.reload();

    await expect(page.getByTestId("vehicle")).toHaveCount(2);
    await expect(page.getByTestId("vehicle").nth(0)).toHaveText(/Green Car/);
    await expect(page.getByTestId("vehicle").nth(1)).toHaveText(/Yellow Van/);
  });

  test("mixed config (yaml + db)", async ({ page }) => {
    await cleanRestart(CONFIG_WITH_VEHICLE);

    await page.goto("/#/config");

    await expect(page.getByTestId("vehicle")).toHaveCount(1);
    const vehicleModal = page.getByTestId("vehicle-modal");

    // create #1
    await page.getByTestId("add-vehicle").click();
    await vehicleModal.getByLabel("Manufacturer").selectOption("Generic vehicle");
    await vehicleModal.getByLabel("Title").fill("Green Car");
    await vehicleModal.getByRole("button", { name: "Validate & save" }).click();

    await expect(page.getByTestId("vehicle")).toHaveCount(2);
    await expect(page.getByTestId("vehicle").nth(0)).toHaveText(/YAML Bike/);
    await expect(page.getByTestId("vehicle").nth(1)).toHaveText(/Green Car/);
  });
});

test.describe("meters", async () => {
  test.beforeAll(async () => {
    await startSimulator();
  });
  test.afterAll(async () => {
    await stopSimulator();
  });

  test("create, edit and remove battery meter", async ({ page }) => {
    // setup test data for mock openems api
    await page.goto(SIMULATOR_URL);
    await page.getByLabel("Battery Power").fill("-2500");
    await page.getByLabel("Battery SoC").fill("75");
    await page.getByRole("button", { name: "Apply changes" }).click();

    await page.goto("/#/config");

    await expect(page.getByTestId("battery")).toHaveCount(0);

    // create #1
    await page.getByRole("button", { name: "Add solar or battery" }).click();

    const meterModal = page.getByTestId("meter-modal");
    await meterModal.getByRole("button", { name: "Add battery meter" }).click();
    await meterModal.getByLabel("Manufacturer").selectOption("OpenEMS");
    await meterModal.getByLabel("IP address or hostname").fill(SIMULATOR_HOST);
    await expect(meterModal.getByRole("button", { name: "Validate & save" })).toBeVisible();
    await meterModal.getByRole("link", { name: "validate" }).click();
    await expect(meterModal.getByText("SoC: 75.0%")).toBeVisible();
    await expect(meterModal.getByText("Power: -2.5 kW")).toBeVisible();
    await meterModal.getByRole("button", { name: "Save" }).click();
    await expect(page.getByTestId("battery")).toBeVisible(1);
    await expect(page.getByTestId("battery")).toContainText("openems");

    // edit #1
    await page.getByTestId("battery").getByRole("button", { name: "edit" }).click();
    await meterModal.getByLabel("Battery capacity in kWh").fill("20");
    await meterModal.getByRole("button", { name: "Validate & save" }).click();

    await expect(page.getByTestId("battery")).toBeVisible(1);
    await expect(page.getByTestId("battery")).toContainText("openems");
    await expect(page.getByTestId("battery").getByText("SoC: 75.0%")).toBeVisible();
    await expect(page.getByTestId("battery").getByText("Power: -2.5 kW")).toBeVisible();
    await expect(page.getByTestId("battery").getByText("Capacity: 20.0 kWh")).toBeVisible();

    // restart and check in main ui
    await restart(CONFIG_EMPTY);
    await page.goto("/");
    await page.getByTestId("visualization").click();
    await expect(page.getByTestId("energyflow")).toContainText("Battery charging75%2.5 kW");

    // delete #1
    await page.goto("/#/config");
    await page.getByTestId("battery").getByRole("button", { name: "edit" }).click();
    await meterModal.getByRole("button", { name: "Delete" }).click();

    await expect(page.getByTestId("battery")).toHaveCount(0);
  });
});

test.describe("site", async () => {
  test("change site title", async ({ page }) => {
    // initial value on main ui
    await page.goto("/");
    await expect(page.getByRole("heading", { name: "Hello World" })).toBeVisible();

    // change value in config
    await page.goto("/#/config");
    await page.getByLabel("Site title").fill("Whoops World");

    // reset form to initial value
    await page.getByRole("button", { name: "Cancel" }).click();
    await expect(page.getByLabel("Site title")).toHaveValue("Hello World");

    // change and save value
    await page.getByLabel("Site title").fill("Ahoy World");
    await page.getByRole("button", { name: "Save" }).click();

    // check changed value on main ui
    await page.getByTestId("home-link").click();
    await expect(page.getByRole("heading", { name: "Ahoy World" })).toBeVisible();
  });
});
