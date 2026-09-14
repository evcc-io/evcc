import { test, expect } from "@playwright/test";
import { start, stop, restart, baseUrl } from "./evcc";
import { expectModalHidden, expectModalVisible } from "./utils";

const CONFIG = "heating.evcc.yaml";

test.use({ baseURL: baseUrl() });

test.beforeAll(async () => {
  await start(CONFIG);
});
test.afterAll(async () => {
  await stop();
});

test.beforeEach(async ({ page }) => {
  await page.goto("/");
});

test.describe("loadpoint", async () => {
  test("initial values", async ({ page }) => {
    const lp = page.getByTestId("loadpoint").first();
    await expect(lp.getByTestId("current-soc")).toContainText("40.0°C");
    await expect(lp.getByTestId("limit-soc")).toContainText("100.0°C");
  });

  test("change limit in 1° steps", async ({ page }) => {
    const lp = page.getByTestId("loadpoint").first();
    await lp.getByTestId("limit-soc").getByRole("combobox").selectOption("69.0°C");
    await expect(lp.getByTestId("limit-soc")).toContainText("69.0°C");
    await page.reload();
    await expect(lp.getByTestId("limit-soc")).toContainText("69.0°C");
  });
});

test.describe("integrated device", async () => {
  test("switch device modes without always charge option", async ({ page }) => {
    const lp = page.getByTestId("loadpoint").first();
    const mode = lp.getByTestId("mode");
    await expect(mode.getByRole("button", { name: "Off", exact: true })).toBeVisible();
    await expect(mode.getByRole("button", { name: "Smart", exact: true })).toBeVisible();
    await expect(mode.getByRole("button", { name: "On", exact: true })).toBeVisible();
    await expect(mode.getByRole("button", { name: "Always charge" })).toHaveCount(0);
  });

  test("continuous device modes with always charge option", async ({ page }) => {
    const lp = page.getByTestId("loadpoint").nth(1);
    const mode = lp.getByTestId("mode");
    await expect(mode.getByRole("button", { name: "Normal", exact: true })).toBeVisible();
    await expect(mode.getByRole("button", { name: "Smart", exact: true })).toBeVisible();
    await expect(mode.getByRole("button", { name: "Boost", exact: true })).toBeVisible();
    await mode.getByRole("button", { name: "Always charge" }).click();
    await expect(mode.getByRole("switch", { name: "Always heat" })).toBeVisible();
  });

  test("min temperature", async ({ page }) => {
    const lp = page.getByTestId("loadpoint").first();
    await lp.getByTestId("loadpoint-settings-button").last().click();
    const modal = page.getByTestId("loadpoint-settings-modal").first();
    await expectModalVisible(modal);
    const minTemp = modal.getByLabel("Min. temperature");
    await expect(minTemp).toHaveValue("0");
    await minTemp.selectOption("50.0°C");
    await modal.getByRole("button", { name: "Close" }).click();
    await expectModalHidden(modal);

    // temp 40 below minimum 50 -> forced heating indicator
    await expect(lp.getByTestId("vehicle-status-minsoc")).toContainText("50.0°C");

    await page.reload();
    await expect(lp.getByTestId("vehicle-status-minsoc")).toContainText("50.0°C");
    await lp.getByTestId("loadpoint-settings-button").last().click();
    await expectModalVisible(modal);
    await expect(modal.getByLabel("Min. temperature")).toHaveValue("50");
    await modal.getByRole("button", { name: "Close" }).click();
    await expectModalHidden(modal);

    // value survives restart
    await restart(CONFIG);
    await page.reload();
    await expect(lp.getByTestId("vehicle-status-minsoc")).toContainText("50.0°C");
    await lp.getByTestId("loadpoint-settings-button").last().click();
    await expectModalVisible(modal);
    await expect(modal.getByLabel("Min. temperature")).toHaveValue("50");
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
