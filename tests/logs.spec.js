import { test, expect } from "@playwright/test";
import { start, stop, baseUrl } from "./evcc";

test.use({ baseURL: baseUrl() });

test.beforeAll(async () => {
  await start("basics.evcc.yaml", "password.sql");
});
test.afterAll(async () => {
  await stop();
});

async function login(page) {
  await page.locator("#loginPassword").fill("secret");
  await page.getByRole("button", { name: "Login" }).click();
}

test.describe("opening logs", async () => {
  test("via config", async ({ page }) => {
    await page.goto("/");
    await page.getByTestId("topnavigation-button").click();
    await page.getByRole("link", { name: "Configuration" }).click();
    await login(page);
    await page.getByRole("link", { name: "Logs" }).click();
    await expect(page.getByRole("heading", { name: "Logs", exact: false })).toBeVisible();
  });
  test("via notifications", async ({ page }) => {
    await page.goto("/");
    await page.evaluate(() => window.app.raise({ message: "Fake Error" }));
    await page.getByTestId("notification-icon").click();
    await page.getByRole("link", { name: "View full logs" }).click();
    await login(page);
    await expect(page.getByRole("heading", { name: "Logs", exact: false })).toBeVisible();
  });
  test("via need help", async ({ page }) => {
    await page.goto("/");
    await page.getByTestId("topnavigation-button").click();
    await page.getByRole("button", { name: "Need Help?" }).click();
    await page.getByRole("link", { name: "View logs" }).click();
    await login(page);
    await expect(page.getByRole("heading", { name: "Logs", exact: false })).toBeVisible();
  });
});

test.describe("features", async () => {
  test("content", async ({ page }) => {
    await page.goto("/#/log");
    await login(page);
    await expect(page.getByTestId("log-content")).toContainText("starting ui and api at");
  });
});
