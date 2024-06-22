import { test, expect } from "@playwright/test";
import { start, stop, restart, cleanRestart, baseUrl } from "./evcc";

const CONFIG_EMPTY = "config-empty.evcc.yaml";
const CONFIG_WITH_VEHICLE = "config-with-vehicle.evcc.yaml";

test.use({ baseURL: baseUrl() });

test.beforeAll(async () => {
  await start(CONFIG_EMPTY, "password.sql");
});
test.afterAll(async () => {
  await stop();
});

async function login(page) {
  await page.locator("#loginPassword").fill("secret");
  await page.getByRole("button", { name: "Login" }).click();
}

async function enableExperimental(page) {
  await page
    .getByTestId("generalconfig-experimental")
    .getByRole("button", { name: "edit" })
    .click();
  await page.getByLabel("Experimental 🧪").click();
  await page.getByRole("button", { name: "Close" }).click();
}

test.describe("vehicles", async () => {
  test("create, edit and delete vehicles", async ({ page }) => {
    await page.goto("/#/config");
    await login(page);
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
    await page.goto("/#/config");
    await login(page);
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
    await restart(CONFIG_EMPTY);
    await page.reload();

    await expect(page.getByTestId("vehicle")).toHaveCount(2);
    await expect(page.getByTestId("vehicle").nth(0)).toHaveText(/Green Car/);
    await expect(page.getByTestId("vehicle").nth(1)).toHaveText(/Yellow Van/);
  });

  test("mixed config (yaml + db)", async ({ page }) => {
    await cleanRestart(CONFIG_WITH_VEHICLE, "password.sql");

    await page.goto("/#/config");
    await login(page);
    await enableExperimental(page);

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
