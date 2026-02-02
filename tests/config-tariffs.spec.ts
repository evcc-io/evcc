import { test, expect } from "@playwright/test";
import { start, stop, restart, baseUrl } from "./evcc";
import { expectModalHidden, expectModalVisible, editorClear, editorPaste } from "./utils";

const CONFIG_WITH_TARIFFS = "config-with-tariffs.evcc.yaml";
const CONFIG_TARIFFS_LEGACY = "tariffs-legacy.sql";

test.use({ baseURL: baseUrl() });
test.describe.configure({ mode: "parallel" });

test.afterEach(async () => {
  await stop();
});

test.describe("tariffs", async () => {
  test("tariffs not configured", async ({ page }) => {
    await start();
    await page.goto("/#/config");

    // New configuration section should show with "Add Tariff" button
    await expect(page.getByRole("heading", { name: "Tariffs & Forecasts" })).toBeVisible();
    await expect(page.getByRole("button", { name: "Add Tariff" })).toBeVisible();

    // Old tariff card should not be shown
    await expect(page.getByTestId("tariffs-legacy")).not.toBeVisible();
  });

  test("tariffs from yaml ui (legacy)", async ({ page }) => {
    await start(undefined, CONFIG_TARIFFS_LEGACY);
    await page.goto("/#/config");

    await page.getByTestId("tariffs-legacy").getByRole("button", { name: "edit" }).click();
    const modal = page.getByTestId("tariffs-legacy-modal");
    await expectModalVisible(modal);
    await page.waitForLoadState("networkidle");

    // check for new configuration notice
    await expect(modal.getByRole("alert")).toContainText("New tariff configuration available");

    // default content
    const editor = modal.getByTestId("yaml-editor");
    await expect(editor).toContainText("currency: EUR");

    // clear and enter invalid yaml
    await editorClear(editor);
    await editorPaste(editor, page, "foo: bar");
    await page.getByRole("button", { name: "Save" }).click();
    await expect(modal.getByTestId("error")).toContainText("invalid keys: foo");

    // clear and enter valid yaml
    await editorClear(editor);
    await editorPaste(
      editor,
      page,
      `currency: CHF
grid:
  type: fixed
  price: 0.123`
    );

    await page.getByRole("button", { name: "Save" }).click();
    await expect(modal.getByTestId("error")).not.toBeVisible();

    // modal closes
    await expectModalHidden(modal);

    // restart button appears
    const restartButton = await page
      .getByTestId("bottom-banner")
      .getByRole("button", { name: "Restart" });
    await expect(restartButton).toBeVisible();

    await restart();

    // restart done
    await expect(restartButton).not.toBeVisible();

    await expect(page.getByTestId("generalconfig-currency")).toContainText("Currency CHF");
    await expect(page.getByTestId("tariffs-legacy")).toContainText(
      ["Grid price", "12.3 ct."].join("")
    );
  });

  test("tariffs from evcc.yaml", async ({ page }) => {
    await start(CONFIG_WITH_TARIFFS);
    await page.goto("/#/config");

    await expect(page.getByTestId("generalconfig-currency")).toContainText("Currency SEK");
    await expect(page.getByTestId("tariffs-legacy")).toBeVisible();
    await expect(page.getByTestId("tariffs-legacy")).toContainText(
      [
        "Tariffs & Forecasts",
        "Grid price",
        "30.0 öre",
        "Feed-in price",
        "-10.0 öre",
        "Grid CO₂",
        "300 g",
      ].join("")
    );
  });
});
