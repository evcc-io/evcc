import { test, expect } from "@playwright/test";
import { start, stop, baseUrl, restart } from "./evcc";
import {
  editorClear,
  editorPaste,
  enableExperimental,
  expectModalHidden,
  expectModalVisible,
} from "./utils";

const CONFIG_MESSAGING_MIGRATE = "config-messaging-migrate.sql";

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

    await modal.getByRole("link", { name: "Services (0)" }).click();

    //  Pushover
    await modal.getByRole("button", { name: "Add messaging" }).click();
    await modal.getByRole("link", { name: "Pushover" }).click();

    const pushoverBox = modal.getByTestId("service-box-pushover");
    const pushoverToken = pushoverBox.getByLabel("Token");
    const pushoverRecipients = pushoverBox.getByLabel("Recipients");
    const pushoverDevices = pushoverBox.getByLabel("Device names");

    await pushoverToken.fill("pushoverToken");
    await pushoverRecipients.fill(["recipient1", "recipient2", "recipient3"].join("\n"));
    await pushoverDevices.fill(["device1", "device2", "device3"].join("\n"));

    //  Telegram
    await modal.getByRole("button", { name: "Add messaging" }).click();
    await modal.getByRole("link", { name: "Telegram" }).click();

    const telegramBox = modal.getByTestId("service-box-telegram");
    const telegramToken = telegramBox.getByLabel("Token");
    const telegramRecipients = telegramBox.getByLabel("Chat IDs");

    await telegramToken.fill("telegramToken");
    await telegramRecipients.fill(["chatid1", "chatid2", "chatid3"].join("\n"));

    //  Email
    await modal.getByRole("button", { name: "Add messaging" }).click();
    await modal.getByRole("link", { name: "Email" }).click();

    const emailBox = modal.getByTestId("service-box-email");
    const emailHost = emailBox.getByLabel("Host");
    const emailPort = emailBox.getByLabel("Port");
    const emailUser = emailBox.getByLabel("User");
    const emailPassword = emailBox.getByLabel("Password");
    const emailFrom = emailBox.getByLabel("From");
    const emailTo = emailBox.getByLabel("To");

    await emailHost.fill("emailserver.example.com");
    await emailPort.fill("587");
    await emailUser.fill("john.doe");
    await emailPassword.fill("secret123");
    await emailFrom.fill("john.doe@mail.com");
    await emailTo.fill("recipient@mail.com");

    //  Shout
    await modal.getByRole("button", { name: "Add messaging" }).click();
    await modal.getByRole("link", { name: "Shout" }).click();

    const shoutBox = modal.getByTestId("service-box-shout");
    const shoutUri = shoutBox.getByLabel("Uri");
    await shoutUri.fill("gotify://gotify.example.com:443/AzyoeNS.D4iJLVa/?priority=1");

    //  Ntfy
    await modal.getByRole("button", { name: "Add messaging" }).click();
    await modal.getByRole("link", { name: "Ntfy" }).click();

    const ntfyBox = modal.getByTestId("service-box-ntfy");
    const ntfyHost = ntfyBox.getByLabel("Host");
    const ntfyTopics = ntfyBox.getByLabel("Topics");
    const ntfyPriority = ntfyBox.getByLabel("Priority");
    const ntfyTagsAndEmojis = ntfyBox.getByLabel("Tags & emojis");

    await ntfyHost.fill("ntfy.sh");
    await ntfyTopics.fill(["evcc_alert", "evcc_pushmessage"].join("\n"));
    await ntfyPriority.selectOption({ label: "low" });
    await ntfyTagsAndEmojis.fill(["+1", "blue_car"].join("\n"));

    //  Custom
    await modal.getByRole("button", { name: "Add messaging" }).click();
    await modal.getByRole("link", { name: "Custom" }).click();

    const customBox = modal.getByTestId("service-box-custom");
    const customEncoding = customBox.getByLabel("Encoding");
    const customPlugin = customBox.getByTestId("yaml-editor");

    await customEncoding.selectOption({ label: "title" });
    await expect(customPlugin).toContainText(
      ["send:", "source: script", 'cmd: /usr/local/bin/evcc_message "{{.send}}"'].join("\n")
    );

    await editorClear(customPlugin);
    await editorPaste(
      customPlugin,
      page,
      ["send:", "source: script", 'cmd: /usr/local/bin/evcc "Title: {{.send}}"'].join("\n")
    );

    // validate connection
    await modal.getByRole("button", { name: "Save" }).click();
    await expectModalHidden(modal);
    await expect(messagingCard).toContainText(["Amount", "6"].join(""));

    // restart button appears
    const restartButton = await page
      .getByTestId("bottom-banner")
      .getByRole("button", { name: "Restart" });
    await expect(restartButton).toBeVisible();

    await restart();
    await page.reload();

    await messagingCard.getByRole("button", { name: "edit" }).click();

    // validate Pushover
    // validate Telegram
    // validate Email
    // validate Shout
    // validate Ntfy
    // validate Custom
  });

  test("messaging via db (yaml to json migration)", async ({ page }) => {
    await start(undefined, CONFIG_MESSAGING_MIGRATE);
    await page.goto("/#/config");
    await enableExperimental(page, false);

    const messagingCard = page.getByTestId("messaging");
    await expect(messagingCard).toContainText(["Amount", "6"].join(""));
  });
});
