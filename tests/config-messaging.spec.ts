import { test, expect } from "@playwright/test";
import { start, stop, baseUrl, restart } from "./evcc";
import { editorClear, editorPaste, expectModalHidden, expectModalVisible } from "./utils";

test.use({ baseURL: baseUrl() });

test.afterEach(async () => {
  await stop();
});

test.describe("messaging", async () => {
  test("not configured", async ({ page }) => {
    await start();
    await page.goto("/#/config");

    const messagingCard = page.getByTestId("messaging");

    await expect(messagingCard).toBeVisible();
    await expect(messagingCard).toContainText(["Configured", "no"].join(""));
  });
  test("configured: ui", async ({ page }) => {
    await start();
    await page.goto("/#/config");

    const messagingCard = page.getByTestId("messaging");
    const modal = page.getByTestId("messaging-modal");

    await messagingCard.getByRole("button", { name: "edit" }).click();
    await expectModalVisible(modal);

    const editor = modal.getByTestId("yaml-editor");
    const editorContent = [
      "events:",
      "  start:",
      "    title: Charge started",
      "    msg: Started charging",
    ].join("\n");

    await editorClear(editor);
    await editorPaste(editor, page, editorContent);

    await page.getByRole("button", { name: "Save" }).click();
    await expectModalHidden(modal);

    // restart button appears
    const restartButton = await page
      .getByTestId("bottom-banner")
      .getByRole("button", { name: "Restart" });
    await expect(restartButton).toBeVisible();

    await restart();
    await page.reload();

    await expect(messagingCard).toContainText(["Configured", "yes"].join(""));
    await messagingCard.getByRole("button", { name: "edit" }).click();
    await expectModalVisible(modal);
    await expect(modal).toContainText(editorContent);
  });
  test("configured: evcc.yaml", async ({ page }) => {
    await start("config-messaging.evcc.yaml");
    await page.goto("/#/config");

    const messagingCard = page.getByTestId("messaging");

    await expect(messagingCard).toBeVisible();
    await expect(messagingCard).toContainText(["Configured", "no"].join(""));
  });
});
