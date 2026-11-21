import { test, expect } from "@playwright/test";
import { start, stop, baseUrl } from "./evcc";

test.use({ baseURL: baseUrl() });

test.afterEach(async () => {
  await stop();
});

test.describe("price lines", async () => {
  test("focus and release", async ({ page }) => {
    await start("forecast-prices.evcc.yaml");
    await page.goto("/#/forecast");

    const grid = page.getByRole("button", { name: /Grid import/ });
    const feedin = page.getByRole("button", { name: /Grid export/ });
    await expect(grid).toHaveAttribute("aria-pressed", "false");
    await expect(feedin).toHaveAttribute("aria-pressed", "false");

    await grid.click();
    await expect(grid).toHaveAttribute("aria-pressed", "true");
    await expect(feedin).toHaveAttribute("aria-pressed", "false");

    await feedin.click();
    await expect(feedin).toHaveAttribute("aria-pressed", "true");
    await expect(grid).toHaveAttribute("aria-pressed", "false");

    await feedin.click();
    await expect(feedin).toHaveAttribute("aria-pressed", "false");
    await expect(grid).toHaveAttribute("aria-pressed", "false");
  });

  test("survives a reload", async ({ page }) => {
    await start("forecast-prices.evcc.yaml");
    await page.goto("/#/forecast");

    await page.getByRole("button", { name: /Grid import/ }).click();
    await page.reload();

    await expect(page.getByRole("button", { name: /Grid import/ })).toHaveAttribute(
      "aria-pressed",
      "true"
    );
  });

  test("single tariff", async ({ page }) => {
    await start("forecast-grid-only.evcc.yaml");
    await page.goto("/#/forecast");

    const grid = page.getByRole("button", { name: /Grid import/ });
    await expect(grid).toHaveAttribute("aria-pressed", "false");
    await grid.click();
    await expect(grid).toHaveAttribute("aria-pressed", "true");
    await grid.click();
    await expect(grid).toHaveAttribute("aria-pressed", "false");
  });
});
