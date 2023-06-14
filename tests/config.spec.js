const { test, expect } = require("@playwright/test");
const { start, stop } = require("./evcc");

test.beforeAll(async () => {
  await start("config.evcc.yaml");
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

    // create #1
    await page.getByRole("button", { name: "add vehicle" }).click();
    await page.getByLabel("Manufacturer").selectOption("Generisches Fahrzeug");
    await page.getByLabel("Title").fill("Green Car");
    await page.getByRole("button", { name: "Test" }).click();
    await page.getByRole("button", { name: "Create" }).click();

    await expect(page.getByTestId("vehicle")).toHaveCount(1);

    // create #2
    await page.getByRole("button", { name: "add vehicle" }).click();
    await page.getByLabel("Manufacturer").selectOption("Generisches Fahrzeug");
    await page.getByLabel("Title").fill("Yellow Van");
    await page.getByLabel("Icon").selectOption("van");
    await page.getByRole("button", { name: "Test" }).click();
    await page.getByRole("button", { name: "Create" }).click();

    await expect(page.getByTestId("vehicle")).toHaveCount(2);
    await expect(page.getByTestId("vehicle").nth(0)).toHaveText(/Green Car/);
    await expect(page.getByTestId("vehicle").nth(1)).toHaveText(/Yellow Van/);

    // edit #1
    await page.getByTestId("vehicle").nth(0).getByRole("button", { name: "edit" }).click();
    await expect(page.getByLabel("Title")).toHaveValue("Green Car");
    await expect(page.getByLabel("Icon")).toHaveValue("car");
    await page.getByLabel("Title").fill("Fancy Car");
    await page.getByRole("button", { name: "Test" }).click();
    await page.getByRole("button", { name: "Update" }).click();

    await expect(page.getByTestId("vehicle")).toHaveCount(2);
    await expect(page.getByTestId("vehicle").nth(0)).toHaveText(/Fancy Car/);

    // delete #1
    await page.getByTestId("vehicle").nth(0).getByRole("button", { name: "edit" }).click();
    await page.getByRole("button", { name: "Delete Vehicle" }).click();

    await expect(page.getByTestId("vehicle")).toHaveCount(1);
    await expect(page.getByTestId("vehicle").nth(0)).toHaveText(/Yellow Van/);
  });
});
