import { test, expect, type Locator } from "@playwright/test";
import { start, stop, restart, baseUrl } from "./evcc";

test.use({ baseURL: baseUrl() });

test.afterEach(async () => {
  await stop();
});

const expectActiveMode = async (loadpoint: Locator, name: string) => {
  const mode = loadpoint.getByTestId("mode");
  await expect(mode.getByRole("button", { name })).toContainClass("active");
};

test("default mode on boot", async ({ page }) => {
  await start(undefined, "default-mode.sql");
  await page.goto("/");
  const loadpoints = page.getByTestId("loadpoint");

  // configured default wins over last mode
  await expectActiveMode(loadpoints.nth(0), "Fast");

  // keep as is restores last mode
  await expectActiveMode(loadpoints.nth(1), "Fast");

  // heaters get their default too
  await expectActiveMode(loadpoints.nth(2), "Fast");

  // legacy minpv default becomes smart with always charge
  const legacy = loadpoints.nth(3);
  await expectActiveMode(legacy, "Smart");
  await legacy.getByRole("button", { name: "Always charge" }).click();
  await expect(legacy.getByRole("switch", { name: "Always charge" })).toBeChecked();

  // always charge was seeded once and survives the default being rewritten to smart
  await restart();
  await page.reload();
  await expectActiveMode(legacy, "Smart");
  await legacy.getByRole("button", { name: "Always charge" }).click();
  await expect(legacy.getByRole("switch", { name: "Always charge" })).toBeChecked();
});
