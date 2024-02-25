const { test, expect } = require("@playwright/test");
const { start, stop } = require("./evcc");
const { startSimulator, stopSimulator, SIMULATOR_URL } = require("./simulator");

const CONFIG = "simulator.evcc.yaml";

test.beforeAll(async () => {
  await start(CONFIG);
  await startSimulator();
});
test.afterAll(async () => {
  await stopSimulator();
  await stop();
});

test.beforeEach(async ({ page }) => {
  await page.goto(SIMULATOR_URL);
  await page.getByLabel("PV Power").fill("6000");
  await page.getByTestId("loadpoint0").getByLabel("Power").fill("6000");
  await page.getByTestId("loadpoint0").getByText("C (charging)").click();
  await page.getByTestId("loadpoint0").getByText("Enabled").check();
  await page.getByRole("button", { name: "Apply changes" }).click();
});

test.describe("smart cost limit", async () => {
  test("no limit, normal charging", async ({ page }) => {
    await page.goto("/");
    await expect(page.getByTestId("vehicle-status")).toHaveText("Charging…");
  });
  test("price below limit", async ({ page }) => {
    await page.goto("/");
    await page.getByTestId("loadpoint-settings-button").nth(1).click();
    await expect(page.getByTestId("loadpoint-settings-modal")).toBeVisible();
    await page
      .getByTestId("loadpoint-settings-modal")
      .getByLabel("Price limit")
      .selectOption("≤ 50.0 ct/kWh");
    await page.getByTestId("loadpoint-settings-modal").getByLabel("Close").click();
    await expect(page.getByTestId("loadpoint-settings-modal")).not.toBeVisible();
    await expect(page.getByTestId("vehicle-status")).toHaveText(
      "Charging cheap energy: 40.0 ct (limit 50.0 ct)"
    );
  });
  test("price above limit", async ({ page }) => {
    await page.goto("/");
    await page.getByTestId("loadpoint-settings-button").nth(1).click();
    await expect(page.getByTestId("loadpoint-settings-modal")).toBeVisible();
    await page
      .getByTestId("loadpoint-settings-modal")
      .getByLabel("Price limit")
      .selectOption("≤ 10.0 ct/kWh");
    await page.getByTestId("loadpoint-settings-modal").getByLabel("Close").click();
    await expect(page.getByTestId("loadpoint-settings-modal")).not.toBeVisible();
    await expect(page.getByTestId("vehicle-status")).toHaveText("Charging…");
  });
});
