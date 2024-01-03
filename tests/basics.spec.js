const { test, expect } = require("@playwright/test");
const { start, stop, restart } = require("./evcc");

const CONFIG = "basics.evcc.yaml";

test.beforeAll(async () => {
  await start(CONFIG);
});
test.afterAll(async () => {
  await stop();
});

test.beforeEach(async ({ page }) => {
  await page.goto("/");
});

test.describe("main screen", async () => {
  test("site title", async ({ page }) => {
    await expect(page.getByRole("heading", { name: "Hello World" })).toBeVisible();
  });

  test("visualization", async ({ page }) => {
    const locator = page.getByTestId("visualization");
    await expect(locator).toBeVisible();
    await expect(locator).toContainText("1.0 kW");
  });

  test("one loadpoint", async ({ page }) => {
    await expect(page.getByTestId("loadpoint")).toHaveCount(1);
  });

  test("loadpoint title", async ({ page }) => {
    await expect(page.getByRole("heading", { name: "Carport" })).toBeVisible();
  });

  test("guest vehicle", async ({ page }) => {
    await expect(page.getByRole("heading", { name: "Guest vehicle" })).toBeVisible();
  });
});

test.describe("session info", async () => {
  test("default", async ({ page }) => {
    await expect(page.getByTestId("sessionInfoLabel").first()).toContainText("Duration");
  });
  test("change value", async ({ page }) => {
    // by select
    await page.getByTestId("sessionInfoSelect").first().selectOption({ label: "Solar" });
    await expect(page.getByTestId("sessionInfoLabel").first()).toContainText("Solar");
    // by click on value
    await page.getByTestId("sessionInfoValue").first().click();
    await expect(page.getByTestId("sessionInfoLabel").first()).toContainText("Duration");
  });
  test("keep selection on reload", async ({ page }) => {
    await expect(page.getByTestId("sessionInfoLabel").first()).toContainText("Duration");
    await page.getByTestId("sessionInfoSelect").first().selectOption({ label: "Solar" });
    await expect(page.getByTestId("sessionInfoLabel").first()).toContainText("Solar");
    await page.reload();
    await expect(page.getByTestId("sessionInfoLabel").first()).toContainText("Solar");
  });
});

test.describe("language", async () => {
  test("change and persist", async ({ page }) => {
    // english (browser default)
    await page.goto("/");
    await expect(page.getByTestId("vehicle-status")).toHaveText("Charging…");

    // switch to german
    await page.getByTestId("topnavigation-button").click();
    await page.getByRole("button", { name: "Settings" }).click();
    await page.getByLabel("Language").selectOption({ label: "Deutsch" });
    await page.getByRole("button", { name: "Close" }).click();
    await expect(page.getByTestId("vehicle-status")).toHaveText("Ladevorgang aktiv …");

    // survive reload
    await page.reload();
    await expect(page.getByTestId("vehicle-status")).toHaveText("Ladevorgang aktiv …");

    // survive restart
    await restart(CONFIG);
    console.log("restarted>>>>");
    await page.goto("/");
    await expect(page.getByTestId("vehicle-status")).toHaveText("Ladevorgang aktiv …");

    // switch to auto
    await page.getByTestId("topnavigation-button").click();
    await page.getByRole("button", { name: "Einstellungen" }).click();
    await page.getByLabel("Sprache").selectOption({ label: "Automatisch" });
    await page.getByRole("button", { name: "Close" }).click();
    await expect(page.getByTestId("vehicle-status")).toHaveText("Charging…");
  });
});
