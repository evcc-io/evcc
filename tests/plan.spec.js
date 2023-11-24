const { test, expect } = require("@playwright/test");
const { start, stop, restart } = require("./evcc");

const CONFIG = "plan.evcc.yaml";

test.beforeEach(async () => {
  await start(CONFIG);
});

test.afterEach(async () => {
  await stop();
});

test.describe("basic functionality", async () => {
  test("vehicle with soc and capacity, set and restart", async ({ page }) => {
    await page.goto("/");

    const lp1 = await page.getByTestId("loadpoint").first();

    await expect(lp1.getByTestId("plan-marker")).not.toBeVisible();
    await expect(lp1.getByText("Loadpoint", { exact: true })).toBeVisible();

    await lp1.getByTestId("limit-soc").getByRole("combobox").selectOption("90%");
    await lp1.getByTestId("charging-plan").getByRole("button", { name: "none" }).click();
    await page.getByRole("button", { name: "Set a charging plan" }).click();

    await page.getByTestId("plan-day").selectOption({ index: 1 });
    await page.getByTestId("plan-time").fill("09:30");
    await page.getByTestId("plan-soc").selectOption("80%");
    await page.getByRole("button", { name: "Close" }).click();

    await expect(lp1.getByTestId("plan-marker")).toBeVisible();
    await expect(lp1.getByTestId("charging-plan").getByRole("button")).toHaveText(
      "tomorrow 9:30 AM80%"
    );

    await restart(CONFIG);
    await page.reload();

    await expect(lp1.getByTestId("vehicle-status")).toContainText("Target charging starts at");
    await expect(lp1.getByTestId("plan-marker")).toBeVisible();
    await expect(lp1.getByTestId("charging-plan").getByRole("button")).toHaveText(
      "tomorrow 9:30 AM80%"
    );
    await lp1.getByTestId("charging-plan").getByRole("button").click();
    await expect(page.getByTestId("plan-soc")).toHaveValue("80");
  });
});

test.describe("guest vehicle", async () => {
  test("kWh based plan and limit", async ({ page }) => {
    await page.goto("/");

    const lp1 = await page.getByTestId("loadpoint").first();

    await lp1.getByTestId("change-vehicle").click();
    await lp1.getByRole("button", { name: "Guest vehicle" }).click();

    await lp1.getByTestId("limit-energy").getByRole("combobox").selectOption("50 kWh");

    await lp1.getByTestId("charging-plan").getByRole("button", { name: "none" }).click();
    await page.getByRole("button", { name: "Set a charging plan" }).click();
    await page.getByTestId("plan-energy").selectOption("25 kWh");
    await page.getByRole("button", { name: "Close" }).click();
    await expect(lp1.getByTestId("plan-marker")).toBeVisible();
  });
});
