import { test, expect } from "@playwright/test";
import { start, stop, baseUrl } from "./evcc";

test.use({ baseURL: baseUrl() });

test.beforeAll(async () => {
  await start("smart-cost-only.evcc.yaml");
});
test.afterAll(async () => {
  await stop();
});

test.beforeEach(async ({ page }) => {
  await page.goto("/");
});

test.describe("main screen", async () => {
  test("smart mode", async ({ page }) => {
    await expect(page.getByRole("button", { name: "Off" })).toBeVisible();
    await expect(page.getByRole("button", { name: "Smart" })).toBeVisible();
    await expect(page.getByRole("button", { name: "Fast" })).toBeVisible();
  });

  test("no production and feedin", async ({ page }) => {
    await page.getByTestId("energyflow").click();
    await expect(page.getByTestId("energyflow-entry-gridimport")).toBeVisible();
    await expect(page.getByTestId("energyflow-entry-home")).not.toBeVisible();
    await expect(page.getByTestId("energyflow-entry-loadpoints")).toBeVisible();
    await expect(page.getByTestId("energyflow-entry-gridexport")).not.toBeVisible();
  });
});
