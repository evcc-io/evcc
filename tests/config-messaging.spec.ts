import { test, expect, type Locator } from "@playwright/test";
import { start, stop, baseUrl, restart } from "./evcc";
import { editorClear, editorPaste, expectModalHidden, expectModalVisible } from "./utils";

const CONFIG_MESSAGING_MIGRATE = "config-messaging-migrate.sql";

async function validateServices(modal: Locator) {
  // Validate Pushover
  const pushoverBox = modal.getByTestId("service-box-pushover");
  const pushoverToken = pushoverBox.getByLabel("Token");
  const pushoverRecipients = pushoverBox.getByLabel("Recipients");
  const pushoverDevices = pushoverBox.getByLabel("Device names");

  await expect(pushoverToken).toHaveValue("***");
  await expect(pushoverRecipients).toHaveValue(
    ["recipient1", "recipient2", "recipient3"].join("\n")
  );
  await expect(pushoverDevices).toHaveValue(["device1", "device2", "device3"].join("\n"));

  // Validate Telegram
  const telegramBox = modal.getByTestId("service-box-telegram");
  const telegramToken = telegramBox.getByLabel("Token");
  const telegramRecipients = telegramBox.getByLabel("Chat IDs");

  await expect(telegramToken).toHaveValue("***");
  await expect(telegramRecipients).toHaveValue(["12345", "-54321", "111"].join("\n"));

  // Validate Email
  const emailBox = modal.getByTestId("service-box-email");
  const emailHost = emailBox.getByLabel("Host");
  const emailPort = emailBox.getByLabel("Port");
  const emailUser = emailBox.getByLabel("User");
  const emailPassword = emailBox.getByLabel("Password");
  const emailFrom = emailBox.getByLabel("From");
  const emailTo = emailBox.getByLabel("To");

  await expect(emailHost).toHaveValue("emailserver.example.com");
  await expect(emailPort).toHaveValue("587");
  await expect(emailUser).toHaveValue("john.doe");
  await expect(emailPassword).toHaveValue("***");
  await expect(emailFrom).toHaveValue("john.doe@mail.com");
  await expect(emailTo).toHaveValue(["recipient1@mail.com", "recipient2@mail.com"].join("\n"));

  // Validate Shout
  const shoutBox = modal.getByTestId("service-box-shout");
  const shoutUri = shoutBox.getByLabel("Uri");

  await expect(shoutUri).toHaveValue("gotify://gotify.example.com:443/AzyoeNS.D4iJLVa/?priority=1");

  // Validate Ntfy
  const ntfyBox = modal.getByTestId("service-box-ntfy");
  const ntfyHost = ntfyBox.getByLabel("Host");
  const ntfyTopics = ntfyBox.getByLabel("Topics");
  const ntfyAccesstoken = ntfyBox.getByLabel("Access token");
  const ntfyPriority = ntfyBox.getByLabel("Priority");
  const ntfyTagsAndEmojis = ntfyBox.getByLabel("Tags & emojis");

  await expect(ntfyHost).toHaveValue("ntfy.sh");
  await expect(ntfyTopics).toHaveValue(["evcc_alert", "evcc_pushmessage"].join("\n"));
  await expect(ntfyAccesstoken).toHaveValue("***");
  await expect(ntfyPriority).toHaveValue("low");
  await expect(ntfyTagsAndEmojis).toHaveValue(["+1", "blue_car"].join("\n"));

  // Validate Custom
  const customBox = modal.getByTestId("service-box-custom");
  const customEncoding = customBox.getByLabel("Encoding");
  const customPlugin = customBox.getByTestId("yaml-editor");

  await expect(customEncoding).toHaveValue("title");
  await expect(customPlugin).toHaveText(
    ["123", 'cmd: /usr/local/bin/evcc "Title={{.send}}"', "source: script"].join("")
  );
}

test.use({ baseURL: baseUrl() });

test.afterEach(async () => {
  await stop();
});

test.describe("messaging", async () => {
  test("messaging not configured", async ({ page }) => {
    await start();
    await page.goto("/#/config");

    const messagingCard = page.getByTestId("messaging");

    await expect(messagingCard).toBeVisible();
    await expect(messagingCard).toContainText(["Configured", "no"].join(""));
  });

  test("messaging events via ui", async ({ page }) => {
    await start();
    await page.goto("/#/config");

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
    await modal.getByRole("button", { name: "Save", exact: true }).click();
    await expectModalHidden(modal);
    await expect(messagingCard).toContainText(["Events", "1", "Services", "0"].join(""));

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

    const messagingCard = page.getByTestId("messaging");

    await messagingCard.getByRole("button", { name: "edit" }).click();
    const modal = page.getByTestId("messaging-modal");
    await expectModalVisible(modal);

    await modal.getByRole("link", { name: "Services (0)" }).click();

    //  Pushover
    await modal.getByRole("button", { name: "Add service" }).click();
    await modal.getByRole("link", { name: "Pushover" }).click();

    const pushoverBox = modal.getByTestId("service-box-pushover");
    const pushoverToken = pushoverBox.getByLabel("Token");
    const pushoverRecipients = pushoverBox.getByLabel("Recipients");
    const pushoverDevices = pushoverBox.getByLabel("Device names");

    await pushoverToken.fill("pushoverToken");
    await pushoverRecipients.fill(["recipient1", "recipient2", "recipient3"].join("\n"));
    await pushoverDevices.fill(["device1", "device2", "device3"].join("\n"));

    //  Telegram
    await modal.getByRole("button", { name: "Add service" }).click();
    await modal.getByRole("link", { name: "Telegram" }).click();

    const telegramBox = modal.getByTestId("service-box-telegram");
    const telegramToken = telegramBox.getByLabel("Token");
    const telegramRecipients = telegramBox.getByLabel("Chat IDs");

    await telegramToken.fill("telegramToken");
    await telegramRecipients.fill(["12345", "-54321", "111"].join("\n"));

    //  Email
    await modal.getByRole("button", { name: "Add service" }).click();
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
    await emailTo.fill(["recipient1@mail.com", "recipient2@mail.com"].join("\n"));

    //  Shout
    await modal.getByRole("button", { name: "Add service" }).click();
    await modal.getByRole("link", { name: "Shout" }).click();

    const shoutBox = modal.getByTestId("service-box-shout");
    const shoutUri = shoutBox.getByLabel("Uri");
    await shoutUri.fill("gotify://gotify.example.com:443/AzyoeNS.D4iJLVa/?priority=1");

    //  Ntfy
    await modal.getByRole("button", { name: "Add service" }).click();
    await modal.getByRole("link", { name: "Ntfy" }).click();

    const ntfyBox = modal.getByTestId("service-box-ntfy");
    const ntfyHost = ntfyBox.getByLabel("Host");
    const ntfyTopics = ntfyBox.getByLabel("Topics");
    const ntfyAccesstoken = ntfyBox.getByLabel("Access token");
    const ntfyPriority = ntfyBox.getByLabel("Priority");
    const ntfyTagsAndEmojis = ntfyBox.getByLabel("Tags & emojis");

    await ntfyHost.fill("ntfy.sh");
    await ntfyTopics.fill(["evcc_alert", "evcc_pushmessage"].join("\n"));
    await ntfyAccesstoken.fill("accessToken");
    await ntfyPriority.selectOption({ label: "low" });
    await ntfyTagsAndEmojis.fill(["+1", "blue_car"].join("\n"));

    //  Custom
    await modal.getByRole("button", { name: "Add service" }).click();
    await modal.getByRole("link", { name: "Custom" }).click();

    const customBox = modal.getByTestId("service-box-custom");
    const customEncoding = customBox.getByLabel("Encoding");
    const customPlugin = customBox.getByTestId("yaml-editor");

    await customEncoding.selectOption({ label: "title" });
    await expect(customPlugin).toHaveText(
      ["123", "source: script", 'cmd: /usr/local/bin/evcc_message "{{.send}}"'].join("")
    );

    await editorClear(customPlugin);
    await editorPaste(
      customPlugin,
      page,
      ['cmd: /usr/local/bin/evcc "Title={{.send}}"', "source: script\n"].join("\n")
    );

    // validate connection
    await modal.getByRole("button", { name: "Save", exact: true }).click();
    await expectModalHidden(modal);
    await expect(messagingCard).toContainText(["Events", "0", "Services", "6"].join(""));

    // restart button appears
    const restartButton = await page
      .getByTestId("bottom-banner")
      .getByRole("button", { name: "Restart" });
    await expect(restartButton).toBeVisible();

    await restart();
    await page.reload();

    await messagingCard.getByRole("button", { name: "edit" }).click();
    await expectModalVisible(modal);

    await modal.getByRole("link", { name: "Services (6)" }).click();
    await validateServices(modal);
  });

  test("messaging via db (yaml to json migration)", async ({ page }) => {
    await start(undefined, CONFIG_MESSAGING_MIGRATE);
    await page.goto("/#/config");

    const messagingCard = page.getByTestId("messaging");
    await expect(messagingCard).toContainText(["Events", "3", "Services", "6"].join(""));

    await messagingCard.getByRole("button", { name: "edit" }).click();
    const modal = page.getByTestId("messaging-modal");
    await expectModalVisible(modal);

    // validate events
    for (const e of ["start", "stop", "connect"]) {
      await expect(modal.getByTestId(`event-${e}-switch`)).toBeChecked();
    }
    for (const e of ["disconnect", "guest", "soc", "asleep"]) {
      await expect(modal.getByTestId(`event-${e}-switch`)).not.toBeChecked();
    }

    // validate services
    await modal.getByRole("link", { name: "Services (6)" }).click();
    await validateServices(modal);
  });
});
