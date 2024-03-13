const { test, expect } = require("@playwright/test");
const { start, stop } = require("./evcc");

test.beforeAll(async () => {
  await start("vehicle-error.evcc.yaml");
});
test.afterAll(async () => {
  await stop();
});

test.beforeEach(async ({ page }) => {
  await page.goto("/");
});

test.describe("vehicle startup error", async () => {
  test("broken vehicle: normal title and 'not reachable' icon", async ({ page }) => {
    await expect(page.getByTestId("vehicle-name")).toHaveText("Broken Tesla");
    await expect(page.getByTestId("vehicle-not-reachable-icon")).toBeVisible();
  });

  test("guest vehicle: normal title and no icon", async ({ page }) => {
    // switch to offline vehicle
    await page.getByRole("button", { name: "Broken Tesla" }).click();
    await page.getByRole("button", { name: "Guest vehicle" }).click();

    await expect(page.getByTestId("vehicle-name")).toHaveText("Guest vehicle");
    await expect(page.getByTestId("vehicle-not-reachable-icon")).not.toBeVisible();
  });
});
