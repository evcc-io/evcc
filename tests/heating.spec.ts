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
  test("no Min+Solar mode for switch device", async ({ page }) => {
    const lp = page.getByTestId("loadpoint").first();
    const mode = lp.getByTestId("mode");
    await expect(mode.getByRole("button", { name: "Solar", exact: true })).toBeVisible();
    await expect(mode.getByRole("button", { name: "Min+Solar" })).toHaveCount(0);
  });

  test("min current and phases but no max current for switch device", async ({ page }) => {
    const lp = page.getByTestId("loadpoint").first();
    await lp.getByTestId("loadpoint-settings-button").last().click();
    const modal = page.getByTestId("loadpoint-settings-modal").first();
    await expectModalVisible(modal);
    // min current and phases feed the solar switch-on threshold and stay
    await expect(modal.getByText("Min. Current")).toBeVisible();
    await expect(modal.getByText("Phases")).toBeVisible();
    await expect(modal.getByText("Max. Current")).toHaveCount(0);
  });
});
