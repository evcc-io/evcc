import { test, expect } from "@playwright/test";
import { start, stop, baseUrl } from "./evcc";

test.use({ baseURL: baseUrl() });

test.describe("fatal", async () => {
  test("evcc yaml error", async ({ page }) => {
    const instance = await start("fatal-syntax.evcc.yaml");
    await page.goto("/");
    await expect(page.getByTestId("bottom-banner")).toBeVisible();
    await expect(page.getByTestId("bottom-banner")).toContainText("failed parsing config file");
    await expect(page.getByTestId("generalconfig-password")).not.toBeVisible();
    await stop(instance);
  });
  test("database error", async ({ page }) => {
    const instance = await start("fatal-db.evcc.yaml");
    await page.goto("/");
    await expect(page.getByTestId("bottom-banner")).toBeVisible();
    await expect(page.getByTestId("bottom-banner")).toContainText("invalid database");
    await expect(page.getByTestId("generalconfig-password")).not.toBeVisible();
    await stop(instance);
  });
});
