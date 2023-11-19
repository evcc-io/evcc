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

test.describe("soc plan", async () => {
  test("apply and restart", async ({ page }) => {
    await page.goto("/");

    await expect(page.getByTestId("plan-marker")).not.toBeVisible();

    await page.getByTestId("limit-soc").getByRole("combobox").selectOption("90%");
    await page.getByTestId("charging-plan").getByRole("button", { name: "none" }).click();
    await page.getByRole("button", { name: "Set a charging plan" }).click();

    await page.getByTestId("plan-day").selectOption({ index: 1 });
    await page.getByTestId("plan-time").fill("09:30");
    await page.getByTestId("plan-soc").selectOption("80%");
    await page.getByRole("button", { name: "Close" }).click();

    await expect(page.getByTestId("plan-marker")).toBeVisible();
    await expect(page.getByTestId("charging-plan").getByRole("button")).toHaveText(
      "tomorrow 9:30 AM80%"
    );

    await restart(CONFIG);
    await page.reload();

    await expect(page.getByTestId("vehicle-status")).toContainText("Target charging starts at");
    await expect(page.getByTestId("plan-marker")).toBeVisible();
    await expect(page.getByTestId("charging-plan").getByRole("button")).toHaveText(
      "tomorrow 9:30 AM80%"
    );
    await page.getByTestId("charging-plan").getByRole("button").click();
    await expect(page.getByTestId("plan-soc")).toHaveValue("80");
  });
});
