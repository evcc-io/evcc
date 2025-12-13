import { test, expect } from "@playwright/test";
import { start, stop, baseUrl, restart } from "./evcc";
import { enableExperimental, expectModalHidden, expectModalVisible } from "./utils";

test.use({ baseURL: baseUrl() });

test.afterEach(async () => {
  await stop();
});

test.describe("messaging", async () => {
  test("messaging not configured", async ({ page }) => {
    await start();
    await page.goto("/#/config");
    await enableExperimental(page, false);

    const messagingCard = page.getByTestId("messaging");

    await expect(messagingCard).toBeVisible();
    await expect(messagingCard).toContainText(["Configured", "no"].join(""));
  });

  test("messaging events via ui", async ({ page }) => {
    await start();
    await page.goto("/#/config");
    await enableExperimental(page, false);

    const messagingCard = page.getByTestId("messaging");

    await messagingCard.getByRole("button", { name: "edit" }).click();
    const modal = page.getByTestId("messaging-modal");
    await expectModalVisible(modal);

    // validate start event
    const switchInput = modal.getByTestId("event-start-switch");
    const titleInput = modal.getByTestId("event-start-title").locator("input");
    const messageInput = modal.getByTestId("event-start-message");

    await expect(switchInput).not.toBeChecked();
    await expect(titleInput).toBeDisabled();
    await expect(titleInput).toHaveValue("Charge started");
    await expect(messageInput).toBeDisabled();
    await expect(messageInput).toHaveValue("Started charging in ${mode} mode.");

    await switchInput.check();
    await expect(titleInput).toBeEnabled();
    await expect(messageInput).toBeEnabled();

    await titleInput.fill("event-start-title");
    await messageInput.fill("event-start-message");

    // validate connection
    await modal.getByRole("button", { name: "Save" }).click();
    await expectModalHidden(modal);
    await expect(messagingCard).toContainText(["Configured", "no"].join(""));

    // restart button appears
    const restartButton = await page
      .getByTestId("bottom-banner")
      .getByRole("button", { name: "Restart" });
    await expect(restartButton).toBeVisible();

    await restart();
    await page.reload();

    await messagingCard.getByRole("button", { name: "edit" }).click();

    // validate start event
    await expect(switchInput).toBeChecked();
    await expect(titleInput).toBeEnabled();
    await expect(messageInput).toBeEnabled();
    await expect(titleInput).toHaveValue("event-start-title");
    await expect(messageInput).toHaveValue("event-start-message");
  });

  test("messaging services via ui", async ({ page }) => {
    await start();
    await page.goto("/#/config");
    await enableExperimental(page, false);

    const messagingCard = page.getByTestId("messaging");

    await messagingCard.getByRole("button", { name: "edit" }).click();
    const modal = page.getByTestId("messaging-modal");
    await expectModalVisible(modal);
  });
});
