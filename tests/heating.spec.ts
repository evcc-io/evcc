import { test, expect } from "@playwright/test";
import { start, stop, baseUrl } from "./evcc";
import { expectModalVisible } from "./utils";

test.use({ baseURL: baseUrl() });

test.beforeAll(async () => {
  await start("heating.evcc.yaml");
});
test.afterAll(async () => {
  await stop();
});

test.beforeEach(async ({ page }) => {
  await page.goto("/");
});

test.describe("loadpoint", async () => {
  test("initial values", async ({ page }) => {
    await expect(page.getByTestId("current-soc")).toContainText("55.0°C");
    await expect(page.getByTestId("limit-soc")).toContainText("100.0°C");
  });

  test("change limit in 1° steps", async ({ page }) => {
    await page.getByTestId("limit-soc").getByRole("combobox").selectOption("69.0°C");
    await expect(page.getByTestId("limit-soc")).toContainText("69.0°C");
    await page.reload();
    await expect(page.getByTestId("limit-soc")).toContainText("69.0°C");
  });
});

test.describe("integrated device", async () => {
  test("no charged energy and no Min+Solar mode", async ({ page }) => {
    const lp = page.getByTestId("loadpoint").first();
    // integrated device has no charging session, so charged energy is hidden
    await expect(lp.getByTestId("charged")).not.toBeVisible();
    // switch device has no current control, so Min+Solar mode is not offered
    const mode = lp.getByTestId("mode");
    await expect(mode.getByRole("button", { name: "Solar", exact: true })).toBeVisible();
    await expect(mode.getByRole("button", { name: "Min+Solar" })).toHaveCount(0);
  });

  test("no current settings in loadpoint settings", async ({ page }) => {
    const lp = page.getByTestId("loadpoint").first();
    await lp.getByTestId("loadpoint-settings-button").last().click();
    const modal = page.getByTestId("loadpoint-settings-modal").first();
    await expectModalVisible(modal);
    await expect(modal.getByText("Charging Current")).toHaveCount(0);
  });
});
