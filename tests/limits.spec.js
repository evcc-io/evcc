const { test, expect } = require("@playwright/test");
const { start, stop, restart } = require("./evcc");
const { startSimulator, stopSimulator, SIMULATOR_URL } = require("./simulator");

const CONFIG = "simulator.evcc.yaml";

test.beforeAll(async () => {
  await startSimulator();
});
test.afterAll(async () => {
  await stopSimulator();
});

test.beforeEach(async ({ page }) => {
  await start(CONFIG);

  await page.goto(SIMULATOR_URL);
  await page.getByLabel("Grid Power").fill("500");
  await page.getByTestId("vehicle0").getByLabel("SoC").fill("20");
  await page.getByTestId("loadpoint0").getByText("B (connected)").click();
  await page.getByRole("button", { name: "Apply changes" }).click();
});

test.afterEach(async () => {
  await stop();
});

test.describe("limitSoc", async () => {
  test("survives a reload", async ({ page }) => {
    await page.goto("/");

    await expect(page.getByTestId("limit-soc-value")).toHaveText("100%");
    await page.getByTestId("limit-soc").getByRole("combobox").selectOption("50%");
    await expect(page.getByTestId("limit-soc-value")).toHaveText("50%");

    await page.reload();

    await expect(page.getByTestId("limit-soc-value")).toHaveText("50%");
  });

  test("can be set even if vehicle isn't connected yet", async ({ page }) => {
    await page.goto(SIMULATOR_URL);
    await page.getByTestId("loadpoint0").getByText("A (disconnected)").click();
    await page.getByRole("button", { name: "Apply changes" }).click();

    await page.goto("/");
    await expect(page.getByTestId("vehicle-title")).toContainText("blauer e-Golf");
    await expect(page.getByTestId("vehicle-status")).toHaveText("Disconnected.");
    await expect(page.getByTestId("limit-soc-value")).toHaveText("100%");
    await page.getByTestId("limit-soc").getByRole("combobox").selectOption("50%");
    await expect(page.getByTestId("limit-soc-value")).toHaveText("50%");

    await page.goto(SIMULATOR_URL);
    await page.getByTestId("loadpoint0").getByText("B (connected)").click();
    await page.getByRole("button", { name: "Apply changes" }).click();

    await page.goto("/");
    await expect(page.getByTestId("limit-soc-value")).toHaveText("50%");
  });

  test("limit soc should be resetted when vehicle gets disconnected", async ({ page }) => {
    await page.goto("/");
    await expect(page.getByTestId("limit-soc-value")).toHaveText("100%");
    await page.getByTestId("limit-soc").getByRole("combobox").selectOption("50%");
    await expect(page.getByTestId("limit-soc-value")).toHaveText("50%");

    // disconnect
    await page.goto(SIMULATOR_URL);
    await page.getByTestId("loadpoint0").getByText("A (disconnected)").click();
    await page.getByRole("button", { name: "Apply changes" }).click();

    await page.goto("/");
    await expect(page.getByTestId("vehicle-status")).toHaveText("Disconnected.");
    await expect(page.getByTestId("limit-soc-value")).toHaveText("100%");

    // connect
    await page.goto(SIMULATOR_URL);
    await page.getByTestId("loadpoint0").getByText("B (connected)").click();
    await page.getByRole("button", { name: "Apply changes" }).click();

    await page.goto("/");
    await expect(page.getByTestId("vehicle-status")).toHaveText("Connected.");
    await expect(page.getByTestId("limit-soc-value")).toHaveText("100%");
  });
});

test.describe("limitEnergy", async () => {
  test("survives a reload", async ({ page }) => {
    await page.goto("/");

    await page.getByRole("button", { name: "blauer e-Golf" }).click();
    await page.getByRole("button", { name: "grüner Honda e" }).click();

    await expect(page.getByTestId("limit-energy-value")).toHaveText("none");
    await page.getByTestId("limit-energy").getByRole("combobox").selectOption("10 kWh (+35%)");
    await expect(page.getByTestId("limit-energy-value")).toHaveText("10 kWh");

    await page.reload();
    await expect(page.getByTestId("limit-energy-value")).toHaveText("10 kWh");
  });
  test("should not be reset on vehicle change", async ({ page }) => {
    await page.goto("/");

    await page.getByRole("button", { name: "blauer e-Golf" }).click();
    await page.getByRole("button", { name: "grüner Honda e" }).click();
    await page.getByTestId("limit-energy").getByRole("combobox").selectOption("10 kWh (+35%)");

    await page.getByRole("button", { name: "grüner Honda e" }).click();
    await page.getByRole("button", { name: "Guest vehicle" }).click();
    await expect(page.getByTestId("limit-energy-value")).toHaveText("10 kWh");
  });
});
