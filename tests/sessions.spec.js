const { test, expect } = require("@playwright/test");
const { execEvcc, stopEvcc } = require("./utils");

let server;

test.beforeAll(() => {
  server = execEvcc("basics.evcc.yaml", "basics.db.sql");
});

test.afterAll(() => {
  stopEvcc(server);
});

test.beforeEach(async ({ page }) => {
  await page.goto("/#/sessions");
});

test.describe("sessions", async () => {
  test("title", async ({ page }) => {
    await expect(page.getByRole("heading", { name: "Charging Sessions" })).toBeVisible();
  });
});
