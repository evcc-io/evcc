import { test, expect } from "@playwright/test";
import { start, stop, restart, baseUrl } from "./evcc";
import {
  enableExperimental,
  expectModalHidden,
  expectModalVisible,
  editorClear,
  editorType,
} from "./utils";

const CONFIG_GRID_ONLY = "config-grid-only.evcc.yaml";
const CONFIG_WITH_TARIFFS = "config-with-tariffs.evcc.yaml";

test.use({ baseURL: baseUrl() });
test.describe.configure({ mode: "parallel" });

test.afterEach(async () => {
  await stop();
});

async function goToConfig(page) {
  await page.goto("/#/config");
  await enableExperimental(page);
}

test.describe("tariffs", async () => {
  test("tariffs not configured", async ({ page }) => {
    await start(CONFIG_GRID_ONLY);
    await goToConfig(page);

    await expect(page.getByTestId("tariffs")).not.toBeVisible();
    await expect(page.getByTestId("add-tariffs")).toBeVisible();
  });

  test("tariffs via ui", async ({ page }) => {
    await start(CONFIG_GRID_ONLY);
    await goToConfig(page);

    await page.getByTestId("add-tariffs").click();
    const modal = await page.getByTestId("tariffs-modal");
    await expectModalVisible(modal);
    await page.waitForLoadState("networkidle");

    // default content
    const editor = modal.getByTestId("yaml-editor");
    await expect(editor).toContainText("#currency: EUR");

    // clear and enter invalid yaml
    await editorClear(editor);
    await editorType(editor, "foo: bar");
    await page.getByRole("button", { name: "Save" }).click();
    await expect(modal.getByTestId("error")).toContainText("invalid keys: foo");

    // clear and enter valid yaml
    await editorClear(editor);
    await editorType(editor, [
      // prettier-ignore
      "currency: CHF",
      "grid:",
      "  type: fixed",
      "price: 0.123",
    ]);

    await page.getByRole("button", { name: "Save" }).click();
    await expect(modal.getByTestId("error")).not.toBeVisible();

    // modal closes
    await expectModalHidden(modal);

    // restart button appears
    const restartButton = await page
      .getByTestId("bottom-banner")
      .getByRole("button", { name: "Restart" });
    await expect(restartButton).toBeVisible();

    await restart(CONFIG_GRID_ONLY);

    // restart done
    await expect(restartButton).not.toBeVisible();

    await expect(page.getByTestId("tariffs")).toContainText(
      ["Currency", "CHF", "Grid price", "12.3 rp"].join("")
    );
  });

  test("tariffs from evcc.yaml", async ({ page }) => {
    await start(CONFIG_WITH_TARIFFS);
    await goToConfig(page);

    await expect(page.getByTestId("tariffs")).toBeVisible();
    await expect(page.getByTestId("tariffs")).toContainText(
      [
        "Tariffs",
        "Currency",
        "SEK",
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
