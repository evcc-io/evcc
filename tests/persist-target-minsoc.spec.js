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

test.describe("targetSoc", async () => {
  test("survives a restart", async ({ page }) => {
    await page.goto("/");

    await expect(page.getByTestId("target-soc-value")).toHaveText("100%");
    await page.getByTestId("target-soc").getByRole("combobox").selectOption("50%");
    await expect(page.getByTestId("target-soc-value")).toHaveText("50%");

    await restart(CONFIG);
    await page.reload();

    await expect(page.getByTestId("target-soc-value")).toHaveText("50%");
  });

  test("can be set even if vehicle isn't connected yet", async ({ page }) => {
    await page.goto(SIMULATOR_URL);
    await page.getByTestId("loadpoint0").getByText("A (disconnected)").click();
    await page.getByRole("button", { name: "Apply changes" }).click();

    await page.goto("/");
    await expect(page.getByTestId("vehicle-title")).toContainText("blauer e-Golf");
    await expect(page.getByTestId("vehicle-status")).toHaveText("Disconnected.");
    await expect(page.getByTestId("target-soc-value")).toHaveText("100%");
    await page.getByTestId("target-soc").getByRole("combobox").selectOption("50%");
    await expect(page.getByTestId("target-soc-value")).toHaveText("50%");

    await page.goto(SIMULATOR_URL);
    await page.getByTestId("loadpoint0").getByText("B (connected)").click();
    await page.getByRole("button", { name: "Apply changes" }).click();

    await page.goto("/");
    await expect(page.getByTestId("target-soc-value")).toHaveText("50%");
  });

  test("target soc should be preserved when vehicle gets disconnected", async ({ page }) => {
    await page.goto("/");
    await expect(page.getByTestId("target-soc-value")).toHaveText("100%");
    await page.getByTestId("target-soc").getByRole("combobox").selectOption("50%");
    await expect(page.getByTestId("target-soc-value")).toHaveText("50%");

    // disconnect
    await page.goto(SIMULATOR_URL);
    await page.getByTestId("loadpoint0").getByText("A (disconnected)").click();
    await page.getByRole("button", { name: "Apply changes" }).click();

    await page.goto("/");
    await expect(page.getByTestId("vehicle-status")).toHaveText("Disconnected.");
    await expect(page.getByTestId("target-soc-value")).toHaveText("50%");

    // connect
    await page.goto(SIMULATOR_URL);
    await page.getByTestId("loadpoint0").getByText("B (connected)").click();
    await page.getByRole("button", { name: "Apply changes" }).click();

    await page.goto("/");
    await expect(page.getByTestId("vehicle-status")).toHaveText("Connected.");
    await expect(page.getByTestId("target-soc-value")).toHaveText("50%");
  });
});

test.describe("targetEnergy", async () => {
  test("survives a restart", async ({ page }) => {
    await page.goto("/");

    await page.getByRole("button", { name: "blauer e-Golf" }).click();
    await page.getByRole("button", { name: "grüner Honda e" }).click();

    await expect(page.getByTestId("target-energy-value")).toHaveText("none");
    await page.getByTestId("target-energy").getByRole("combobox").selectOption("10 kWh (+35%)");
    await expect(page.getByTestId("target-energy-value")).toHaveText("10 kWh");

    await restart(CONFIG);
    await page.reload();

    await page.getByRole("button", { name: "blauer e-Golf" }).click();
    await page.getByRole("button", { name: "grüner Honda e" }).click();

    await expect(page.getByTestId("target-energy-value")).toHaveText("10 kWh");
  });
});

test.describe("minSoc", async () => {
  test("survives a restart", async ({ page }) => {
    await page.goto("/");

    await page.getByTestId("charging-plan").getByRole("button", { name: "none" }).click();
    await page.getByRole("link", { name: "Arrival" }).click();

    await expect(page.getByText("charged to x% in solar mode")).toBeVisible();
    await page.getByRole("combobox", { name: "Min. charge %" }).selectOption("20%");
    await expect(page.getByText("charged to 20% in solar mode")).toBeVisible();

    await restart(CONFIG);
    await page.reload();

    await page.getByTestId("charging-plan").getByRole("button", { name: "none" }).click();
    await page.getByRole("link", { name: "Arrival" }).click();
    await expect(page.getByText("charged to 20% in solar mode")).toBeVisible();
  });

  test("show minsoc instead of plan when minsoc is active", async ({ page }) => {
    await page.goto("/");

    await expect(page.getByTestId("charging-plan")).toContainText("Plan");
    await page.getByTestId("charging-plan").getByRole("button", { name: "none" }).click();
    await page.getByRole("link", { name: "Arrival" }).click();
    await page.getByRole("combobox", { name: "Min. charge %" }).selectOption("50%");
    await page.getByRole("button", { name: "Close" }).click();

    await expect(page.getByTestId("charging-plan")).toContainText("Min charge");
    await expect(page.getByTestId("vehicle-status")).toHaveText("Minimum charging to 50%.");
    await page.getByTestId("charging-plan").getByRole("button", { name: "50%" }).click();

    await page.getByRole("combobox", { name: "Min. charge %" }).selectOption("--");
    await page.getByRole("button", { name: "Close" }).click();
    await expect(page.getByTestId("charging-plan")).toContainText("Plan");
    await expect(page.getByTestId("charging-plan").getByRole("button")).toHaveText("none");
  });

  test("disabled for offline vehicles", async ({ page }) => {
    await page.goto("/");

    // switch to offline vehicle
    await page.getByRole("button", { name: "blauer e-Golf" }).click();
    await page.getByRole("button", { name: "grüner Honda e" }).click();

    await page.getByTestId("charging-plan").getByRole("button", { name: "none" }).click();
    await page.getByRole("link", { name: "Arrival" }).click();
    await expect(page.getByRole("combobox", { name: "Min. charge %" })).toBeDisabled();
  });

  test("disabled for guest vehicles", async ({ page }) => {
    await page.goto("/");

    // switch to offline vehicle
    await page.getByRole("button", { name: "blauer e-Golf" }).click();
    await page.getByRole("button", { name: "Guest vehicle" }).click();

    await page.getByTestId("charging-plan").getByRole("button", { name: "none" }).click();
    await page.getByRole("link", { name: "Arrival" }).click();
    await expect(page.getByRole("combobox", { name: "Min. charge %" })).toBeDisabled();
  });
});

test.describe("targetTime", async () => {
  test("survives a restart", async ({ page }) => {
    await page.goto("/");

    await page.getByTestId("target-soc").getByRole("combobox").selectOption("90%");
    await page.getByTestId("charging-plan").getByRole("button", { name: "none" }).click();

    await page.getByTestId("target-day").selectOption({ index: 1 });
    await page.getByTestId("target-time").fill("09:30");

    await page.getByRole("button", { name: "Activate" }).click();
    await expect(page.getByTestId("charging-plan").getByRole("button")).toHaveText(
      "tomorrow 9:30 AM"
    );

    await restart(CONFIG);
    await page.reload();

    await expect(page.getByTestId("vehicle-status")).toContainText("Target charging starts at");
    await expect(page.getByTestId("charging-plan").getByRole("button")).toHaveText(
      "tomorrow 9:30 AM"
    );
    await expect(page.getByTestId("target-soc-value")).toHaveText("90%");
  });
});
