import { test, expect } from "@playwright/test";
import { start, stop, baseUrl } from "./evcc";
import { enableExperimental } from "./utils";

const CONFIG_GRID_ONLY = "config-grid-only.evcc.yaml";

test.use({ baseURL: baseUrl() });

test.afterEach(async () => {
  await stop();
});

const SELECT_ALL = "ControlOrMeta+KeyA";

async function goToConfig(page) {
  await page.goto("/#/config");
  await enableExperimental(page);
}

test.describe("messaging", async () => {
  test("save a comment", async ({ page }) => {
    await start(CONFIG_GRID_ONLY);
    await goToConfig(page);

    await page.getByTestId("messaging").getByRole("button", { name: "edit" }).click();
    const modal = await page.getByTestId("messaging-modal");
    await expect(modal).toBeVisible();

    await modal.locator(".monaco-editor .view-line").nth(0).click();
    for (let i = 0; i < 4; i++) {
      await page.keyboard.press(SELECT_ALL, { delay: 10 });
      await page.keyboard.press("Backspace", { delay: 10 });
    }
    await page.keyboard.type("# hello world");
    await page.getByRole("button", { name: "Save" }).click();
    await expect(modal).not.toBeVisible();

    page.reload();

    await page.getByTestId("messaging").getByRole("button", { name: "edit" }).click();
    await expect(modal).toBeVisible();
    await expect(modal).toContainText("# hello world");
  });
});
