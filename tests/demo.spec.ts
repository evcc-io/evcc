import { test, expect } from "@playwright/test";
import { start, stop, baseUrl } from "./evcc";
import { expectModalHidden } from "./utils";
import { ChildProcess } from "child_process";

test.use({ baseURL: baseUrl() });

let instance: ChildProcess;

test.beforeAll(async () => {
  instance = await start(undefined, undefined, "--demo");
});
test.afterAll(async () => {
  // force quit by instance, shutdown endpoint disabled
  await stop(instance);
});

test.beforeEach(async ({ page }) => {
  await page.goto("/");
});

test.describe("demo mode", async () => {
  test("no admin password prompt", async ({ page }) => {
    await expectModalHidden(page.getByTestId("password-modal"));
  });

  test("site title", async ({ page }) => {
    await expect(page.getByRole("heading", { name: "Demo Mode" })).toBeVisible();
  });

  test("two loadpoints", async ({ page }) => {
    await expect(page.getByTestId("loadpoint")).toHaveCount(2);
    await expect(page.getByTestId("loadpoint").nth(0)).toContainText("Carport");
    await expect(page.getByTestId("loadpoint").nth(1)).toContainText("Garage");
  });

  test("auth is locked", async ({ page }) => {
    await page.getByTestId("topnavigation-button").click();
    await page.getByRole("link", { name: "Configuration" }).click();
    const loginModal = page.getByTestId("login-modal");
    await expect(loginModal).toBeVisible();
    await expect(loginModal).toContainText("Login is not supported in demo mode.");
  });
});
