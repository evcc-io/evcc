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
