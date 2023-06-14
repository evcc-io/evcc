const { test, expect } = require("@playwright/test");
const { start, stop } = require("./evcc");

test.beforeAll(async () => {
  await start("basics.evcc.yaml");
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
    await expect(locator).toContainText("1,000 W");
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
