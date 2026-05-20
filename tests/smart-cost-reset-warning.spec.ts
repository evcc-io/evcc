import { test, expect } from "@playwright/test";
import { start, stop, baseUrl } from "./evcc";

test.use({ baseURL: baseUrl() });
test.describe.configure({ mode: "parallel" });

const CONFIG = "smart-cost-reset-warning.evcc.yaml";
const PRESET_LIMITS = "smart-cost-reset-warning.sql";

test.afterEach(async () => {
  await stop();
});

test.describe("smart cost reset warning", async () => {
  const limitWarningText = "However, there is still a limit of 12.0 ct/kWh.";

  test("loadpoint: stale limit can be removed without dynamic tariff", async ({ page }) => {
    await start(CONFIG, PRESET_LIMITS);

    await page.goto("/");
    const loadpoint = page.getByTestId("loadpoint").first();
    await loadpoint.getByTestId("loadpoint-settings-button").last().click();
    const modal = page.getByTestId("loadpoint-settings-modal").first();
    await expect(modal).toBeVisible();

    await expect(modal.getByText(limitWarningText)).toBeVisible();
    const removeLimit = modal.getByRole("link", { name: "Remove limit" });
    await expect(removeLimit).toBeVisible();
    await removeLimit.click();
    await expect(modal.getByText(limitWarningText)).not.toBeVisible();
  });

  test("battery: stale limit can be removed without dynamic tariff", async ({ page }) => {
    await start(CONFIG, PRESET_LIMITS);

    await page.goto("/#/battery");
    await expect(page.getByRole("heading", { name: "Grid charging" })).toBeVisible();
    await expect(page.getByText(limitWarningText)).toBeVisible();
    const removeLimit = page.getByRole("link", { name: "Remove limit" });
    await expect(removeLimit).toBeVisible();
    await removeLimit.click();
    await expect(page.getByText(limitWarningText)).not.toBeVisible();
  });
});
