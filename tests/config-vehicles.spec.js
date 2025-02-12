import { test, expect } from "@playwright/test";
import { start, stop, restart, baseUrl } from "./evcc";
import { enableExperimental } from "./utils";

const CONFIG_GRID_ONLY = "config-grid-only.evcc.yaml";
const CONFIG_WITH_VEHICLE = "config-with-vehicle.evcc.yaml";

test.use({ baseURL: baseUrl() });
test.describe.configure({ mode: "parallel" });

test.afterEach(async () => {
  await stop();
});

test.describe("vehicles", async () => {
  test("create, edit and delete vehicles", async ({ page }) => {
    await start(CONFIG_GRID_ONLY);

    await page.goto("/#/config");
    await enableExperimental(page);

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
    await start(CONFIG_GRID_ONLY);

    await page.goto("/#/config");
    await enableExperimental(page);

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
    await restart(CONFIG_GRID_ONLY);
    await page.reload();

    await expect(page.getByTestId("vehicle")).toHaveCount(2);
    await expect(page.getByTestId("vehicle").nth(0)).toHaveText(/Green Car/);
    await expect(page.getByTestId("vehicle").nth(1)).toHaveText(/Yellow Van/);
  });

  test("mixed config (yaml + db)", async ({ page }) => {
    await start(CONFIG_WITH_VEHICLE);

    await page.goto("/#/config");
    await enableExperimental(page);

    await expect(page.getByTestId("vehicle")).toHaveCount(1);
    const vehicleModal = page.getByTestId("vehicle-modal");

    // create #2
    await page.getByTestId("add-vehicle").click();
    await vehicleModal.getByLabel("Manufacturer").selectOption("Generic vehicle");
    await vehicleModal.getByLabel("Title").fill("Green Car");
    await vehicleModal.getByRole("button", { name: "Validate & save" }).click();

    await expect(page.getByTestId("vehicle")).toHaveCount(2);
    await expect(page.getByTestId("vehicle").nth(0)).toHaveText(/YAML Bike/);
    await expect(page.getByTestId("vehicle").nth(1)).toHaveText(/Green Car/);
  });

  test("advanced fields", async ({ page }) => {
    await start(CONFIG_GRID_ONLY);

    await page.goto("/#/config");
    await enableExperimental(page);

    await page.getByTestId("add-vehicle").click();
    const vehicleModal = page.getByTestId("vehicle-modal");

    // generic
    await vehicleModal.getByLabel("Manufacturer").selectOption("Generic vehicle");
    await expect(vehicleModal.getByLabel("Title")).toBeVisible();
    await expect(vehicleModal.getByLabel("Car")).toBeVisible(); // icon
    await expect(vehicleModal.getByLabel("Battery capacity")).toBeVisible();

    await page.getByRole("button", { name: "Show advanced settings" }).click();
    await expect(vehicleModal.getByLabel("Default mode")).toBeVisible();
    await expect(vehicleModal.getByLabel("Maximum phases")).toBeVisible();
    await expect(vehicleModal.getByLabel("Minimum current")).toBeVisible();
    await expect(vehicleModal.getByLabel("Maximum current")).toBeVisible();
    await expect(vehicleModal.getByLabel("Priority")).toBeVisible();
    await expect(vehicleModal.getByLabel("RFID identifiers")).toBeVisible();

    await page.getByRole("button", { name: "Hide advanced settings" }).click();
    await expect(vehicleModal.getByLabel("Default mode")).not.toBeVisible();

    // polestar template
    await vehicleModal.getByLabel("Manufacturer").selectOption("Polestar");
    await expect(vehicleModal.getByLabel("Username")).toBeVisible();
    await expect(vehicleModal.getByLabel("Password")).toBeVisible();
    await expect(vehicleModal.getByLabel("Cache optional")).not.toBeVisible();
    await expect(vehicleModal.getByLabel("Default mode")).not.toBeVisible();

    await page.getByRole("button", { name: "Show advanced settings" }).click();
    await expect(vehicleModal.getByLabel("Cache optional")).toBeVisible();
    await expect(vehicleModal.getByLabel("Default mode")).toBeVisible();
  });

  test("save and restore rfid identifiers", async ({ page }) => {
    await start(CONFIG_GRID_ONLY);

    await page.goto("/#/config");
    await enableExperimental(page);

    await page.getByTestId("add-vehicle").click();
    const vehicleModal = page.getByTestId("vehicle-modal");

    // generic
    await vehicleModal.getByLabel("Manufacturer").selectOption("Generic vehicle");
    await vehicleModal.getByLabel("Title").fill("RFID Car");
    await page.getByRole("button", { name: "Show advanced settings" }).click();
    await vehicleModal.getByLabel("RFID identifiers").fill("aaa\nbbb \n ccc\n\nddd\n");
    await vehicleModal.getByRole("button", { name: "Validate & save" }).click();
    await expect(page.getByTestId("restart-needed")).toBeVisible();

    // restart evcc
    await restart(CONFIG_GRID_ONLY);
    await page.reload();

    await expect(page.getByTestId("vehicle")).toHaveCount(1);
    await page.getByTestId("vehicle").getByRole("button", { name: "edit" }).click();
    await vehicleModal.getByRole("button", { name: "Show advanced settings" }).click();
    await expect(vehicleModal.getByLabel("RFID identifiers")).toHaveValue("aaa\nbbb\nccc\nddd");
    await vehicleModal.getByLabel("Close").click();
    await expect(page.getByTestId("fatal-error")).not.toBeVisible();
  });
});
