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

test.describe("minSoc", async () => {
  test("apply and restart", async ({ page }) => {
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

    await page.getByRole("combobox", { name: "Min. charge %" }).selectOption("---");
    await page.getByRole("button", { name: "Close" }).click();
    await expect(page.getByTestId("charging-plan")).toContainText("Plan");
    await expect(page.getByTestId("charging-plan").getByRole("button")).toHaveText("none");
  });
});

test.describe("limitSoc", async () => {
  test("apply and restart", async ({ page }) => {
    await page.goto("/");

    await page.getByTestId("charging-plan").getByRole("button", { name: "none" }).click();
    await page.getByRole("link", { name: "Arrival" }).click();

    await page.getByRole("combobox", { name: "Default limit" }).selectOption("80%");
    await page.getByRole("button", { name: "Close" }).click();
    await expect(page.getByTestId("limit-soc-value")).toContainText("80%");

    await restart(CONFIG);
    await page.reload();

    await expect(page.getByTestId("limit-soc-value")).toContainText("80%");

    await page.getByTestId("charging-plan").getByRole("button", { name: "none" }).click();
    await page.getByRole("link", { name: "Arrival" }).click();
    await expect(page.getByRole("combobox", { name: "Default limit" })).toHaveValue("80");
  });
});

test.describe("minSoc and limitSoc", async () => {
  test("disabled for offline vehicles", async ({ page }) => {
    await page.goto("/");

    // switch to offline vehicle
    await page.getByRole("button", { name: "blauer e-Golf" }).click();
    await page.getByRole("button", { name: "grüner Honda e" }).click();

    await page.getByTestId("charging-plan").getByRole("button", { name: "none" }).click();
    await page.getByRole("link", { name: "Arrival" }).click();
    await expect(page.getByRole("combobox", { name: "Min. charge %" })).toBeDisabled();
    await expect(page.getByRole("combobox", { name: "Default limit" })).toBeDisabled();
  });

  test("disabled for guest vehicles", async ({ page }) => {
    await page.goto("/");

    // switch to offline vehicle
    await page.getByRole("button", { name: "blauer e-Golf" }).click();
    await page.getByRole("button", { name: "Guest vehicle" }).click();

    await page.getByTestId("charging-plan").getByRole("button", { name: "none" }).click();
    await page.getByRole("link", { name: "Arrival" }).click();
    await expect(page.getByRole("combobox", { name: "Min. charge %" })).toBeDisabled();
    await expect(page.getByRole("combobox", { name: "Default limit" })).toBeDisabled();
  });
});
