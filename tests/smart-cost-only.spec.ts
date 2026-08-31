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

test.describe("always charge", async () => {
  test("toggle states via dropdown", async ({ page }) => {
    const mode = page.getByTestId("mode");
    const chevron = mode.getByRole("button", { name: "Always charge" });
    await chevron.click();
    const dropdown = page.getByTestId("always-charge-dropdown");
    await expect(dropdown).toBeVisible();

    const toggle = dropdown.getByRole("switch");
    await expect(toggle).not.toBeChecked();
    await toggle.click();
    await expect(toggle).toBeChecked();

    await dropdown.getByRole("button", { name: "Only for this session" }).click();
    await expect(dropdown).toContainText("until end of session");

    await toggle.click();
    await expect(toggle).not.toBeChecked();

    await page.keyboard.press("Escape");
    await expect(dropdown).not.toBeVisible();
  });

  test("legacy minpv api maps to smart with always charge", async ({ page, request }) => {
    const res = await request.post(baseUrl() + "/api/loadpoints/1/mode/minpv");
    expect(res.ok()).toBeTruthy();

    const mode = page.getByTestId("mode");
    await expect(mode.getByRole("button", { name: "Smart" })).toHaveClass(/active/);
    await mode.getByRole("button", { name: "Always charge" }).click();
    const dropdown = page.getByTestId("always-charge-dropdown");
    await expect(dropdown.getByRole("switch")).toBeChecked();

    // legacy pv resets always charge
    await request.post(baseUrl() + "/api/loadpoints/1/mode/pv");
    await expect(dropdown.getByRole("switch")).not.toBeChecked();
  });
});
