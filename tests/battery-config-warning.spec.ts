import { test, expect } from "@playwright/test";
import { start, stop, baseUrl } from "./evcc";

const CONFIG = "battery-config-warning.evcc.yaml";

test.use({ baseURL: baseUrl() });

test.beforeAll(async () => {
  await start(CONFIG);
});

test.afterAll(async () => {
  await stop();
});

test("shows invalid battery master on main screen", async ({ page }) => {
  await page.goto("/");

  const warning = page.getByTestId("battery-config-warnings");
  await expect(warning).toBeVisible();
  await expect(warning).toContainText('Battery battery: master reference "missing" is invalid');
  await expect(warning.getByRole("link", { name: "Review configuration" })).toBeVisible();

  await warning.getByRole("button", { name: "Close" }).click();
  await expect(warning).toHaveCount(0);
});
