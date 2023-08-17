const { test, expect } = require("@playwright/test");
const { start, stop, restart, cleanRestart } = require("./evcc");

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

    // create #1
    await page.getByRole("button", { name: "Add vehicle" }).click();
    await page.getByLabel("Manufacturer").selectOption("Generic vehicle");
    await page.getByLabel("Title").fill("Green Car");
    await page.getByRole("button", { name: "Validate & save" }).click();

    await expect(page.getByTestId("vehicle")).toHaveCount(1);

    // create #2
    await page.getByRole("button", { name: "Add vehicle" }).click();
    await page.getByLabel("Manufacturer").selectOption("Generic vehicle");
    await page.getByLabel("Title").fill("Yellow Van");
    await page.getByRole("button", { name: "Validate & save" }).click();

    await expect(page.getByTestId("vehicle")).toHaveCount(2);
    await expect(page.getByTestId("vehicle").nth(0)).toHaveText(/Green Car/);
    await expect(page.getByTestId("vehicle").nth(1)).toHaveText(/Yellow Van/);

    // edit #1
    await page.getByTestId("vehicle").nth(0).getByRole("button", { name: "edit" }).click();
    await expect(page.getByLabel("Title")).toHaveValue("Green Car");
    await page.getByLabel("Title").fill("Fancy Car");
    await page.getByRole("button", { name: "Validate & save" }).click();

    await expect(page.getByTestId("vehicle")).toHaveCount(2);
    await expect(page.getByTestId("vehicle").nth(0)).toHaveText(/Fancy Car/);

    // delete #1
    await page.getByTestId("vehicle").nth(0).getByRole("button", { name: "edit" }).click();
    await page.getByRole("button", { name: "Delete Vehicle" }).click();

    await expect(page.getByTestId("vehicle")).toHaveCount(1);
    await expect(page.getByTestId("vehicle").nth(0)).toHaveText(/Yellow Van/);

    // delete #2
    await page.getByTestId("vehicle").nth(0).getByRole("button", { name: "edit" }).click();
    await page.getByRole("button", { name: "Delete Vehicle" }).click();

    await expect(page.getByTestId("vehicle")).toHaveCount(0);
  });

  test("config should survive restart", async ({ page }) => {
    await page.goto("/#/config");

    await expect(page.getByTestId("vehicle")).toHaveCount(0);

    // create #1 & #2
    await page.getByRole("button", { name: "Add vehicle" }).click();
    await page.getByLabel("Manufacturer").selectOption("Generic vehicle");
    await page.getByLabel("Title").fill("Green Car");
    await page.getByRole("button", { name: "Validate & save" }).click();
    await page.getByRole("button", { name: "Add vehicle" }).click();
    await page.getByLabel("Manufacturer").selectOption("Generic vehicle");
    await page.getByLabel("Title").fill("Yellow Van");
    await page.getByLabel("car").click();
    await page.getByLabel("van").check();
    await page.getByRole("button", { name: "Validate & save" }).click();

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

    // create #1
    await page.getByRole("button", { name: "Add vehicle" }).click();
    await page.getByLabel("Manufacturer").selectOption("Generic vehicle");
    await page.getByLabel("Title").fill("Green Car");
    await page.getByRole("button", { name: "Validate & save" }).click();

    await expect(page.getByTestId("vehicle")).toHaveCount(2);
    await expect(page.getByTestId("vehicle").nth(0)).toHaveText(/YAML Bike/);
    await expect(page.getByTestId("vehicle").nth(1)).toHaveText(/Green Car/);
  });
});
