import { test, expect, type Locator } from "@playwright/test";
import { start, stop, baseUrl, restart } from "./evcc";
import { editorClear, editorPaste, expectModalHidden, expectModalVisible } from "./utils";

const CONFIG_MESSAGING_MIGRATE = "config-messaging-migrate.sql";

async function validateServices(modal: Locator) {
  // Validate Pushover
  const pushoverBox = modal.getByTestId("service-box-pushover");
  await expect(pushoverBox.getByLabel("Token")).toHaveValue("***");
  await expect(pushoverBox.getByLabel("Recipients")).toHaveValue(
    ["recipient1", "recipient2", "recipient3"].join("\n")
  );
  await expect(pushoverBox.getByLabel("Device names")).toHaveValue(
    ["device1", "device2", "device3"].join("\n")
  );

  // Validate Telegram
  const telegramBox = modal.getByTestId("service-box-telegram");
  await expect(telegramBox.getByLabel("Token")).toHaveValue("***");
  await expect(telegramBox.getByLabel("Chat IDs")).toHaveValue(
    ["12345", "-54321", "111"].join("\n")
  );

  // Validate Email
  const emailBox = modal.getByTestId("service-box-email");
  await expect(emailBox.getByLabel("Host")).toHaveValue("emailserver.example.com");
  await expect(emailBox.getByLabel("Port")).toHaveValue("587");
  await expect(emailBox.getByLabel("User")).toHaveValue("john.doe");
  await expect(emailBox.getByLabel("Password")).toHaveValue("***");
  await expect(emailBox.getByLabel("From")).toHaveValue("john.doe@mail.com");
  await expect(emailBox.getByLabel("To")).toHaveValue(
    ["recipient1@mail.com", "recipient2@mail.com"].join("\n")
  );

  // Validate Shout
  await expect(modal.getByTestId("service-box-shout").getByLabel("Uri")).toHaveValue(
    "gotify://gotify.example.com:443/AzyoeNS.D4iJLVa/?priority=1"
  );

  // Validate Ntfy
  const ntfyBox = modal.getByTestId("service-box-ntfy");
  await expect(ntfyBox.getByLabel("Host")).toHaveValue("ntfy.sh");
  await expect(ntfyBox.getByLabel("Topics")).toHaveValue(
    ["evcc_alert", "evcc_pushmessage"].join("\n")
  );
  await expect(ntfyBox.getByLabel("Access token")).toHaveValue("***");
  await expect(ntfyBox.getByLabel("Priority")).toHaveValue("low");
  await expect(ntfyBox.getByLabel("Tags & emojis")).toHaveValue(["+1", "blue_car"].join("\n"));

  // Validate Custom
  const customBox = modal.getByTestId("service-box-custom");
  await expect(customBox.getByLabel("Encoding")).toHaveValue("title");
  await expect(customBox.getByTestId("yaml-editor")).toHaveText(
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
    await pushoverBox.getByLabel("Token").fill("pushoverToken");
    await pushoverBox
      .getByLabel("Recipients")
      .fill(["recipient1", "recipient2", "recipient3"].join("\n"));
    await pushoverBox.getByLabel("Device names").fill(["device1", "device2", "device3"].join("\n"));

    //  Telegram
    await modal.getByRole("button", { name: "Add service" }).click();
    await modal.getByRole("link", { name: "Telegram" }).click();

    const telegramBox = modal.getByTestId("service-box-telegram");
    await telegramBox.getByLabel("Token").fill("telegramToken");
    await telegramBox.getByLabel("Chat IDs").fill(["12345", "-54321", "111"].join("\n"));

    //  Email
    await modal.getByRole("button", { name: "Add service" }).click();
    await modal.getByRole("link", { name: "Email" }).click();

    const emailBox = modal.getByTestId("service-box-email");
    await emailBox.getByLabel("Host").fill("emailserver.example.com");
    await emailBox.getByLabel("Port").fill("587");
    await emailBox.getByLabel("User").fill("john.doe");
    await emailBox.getByLabel("Password").fill("secret123");
    await emailBox.getByLabel("From").fill("john.doe@mail.com");
    await emailBox.getByLabel("To").fill(["recipient1@mail.com", "recipient2@mail.com"].join("\n"));

    //  Shout
    await modal.getByRole("button", { name: "Add service" }).click();
    await modal.getByRole("link", { name: "Shout" }).click();

    await modal
      .getByTestId("service-box-shout")
      .getByLabel("Uri")
      .fill("gotify://gotify.example.com:443/AzyoeNS.D4iJLVa/?priority=1");

    //  Ntfy
    await modal.getByRole("button", { name: "Add service" }).click();
    await modal.getByRole("link", { name: "Ntfy" }).click();

    const ntfyBox = modal.getByTestId("service-box-ntfy");
    await ntfyBox.getByLabel("Host").fill("ntfy.sh");
    await ntfyBox.getByLabel("Topics").fill(["evcc_alert", "evcc_pushmessage"].join("\n"));
    await ntfyBox.getByLabel("Access token").fill("accessToken");
    await ntfyBox.getByLabel("Priority").selectOption({ label: "low" });
    await ntfyBox.getByLabel("Tags & emojis").fill(["+1", "blue_car"].join("\n"));

    //  Custom
    await modal.getByRole("button", { name: "Add service" }).click();
    await modal.getByRole("link", { name: "Custom" }).click();

    const customBox = modal.getByTestId("service-box-custom");
    await customBox.getByLabel("Encoding").selectOption({ label: "title" });
    const customPlugin = customBox.getByTestId("yaml-editor");
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

    await expect(page.getByTestId("fatal-error")).toContainText(
      "cannot create messenger type 'telegram': invalid bot token"
    );

    const messagingCard = page.getByTestId("messaging");
    await expect(messagingCard).toHaveClass(/round-box--error/);
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

  test("add and remove service", async ({ page }) => {
    await start();
    await page.goto("/#/config");

    const messagingCard = page.getByTestId("messaging");
    await messagingCard.getByRole("button", { name: "edit" }).click();

    const modal = page.getByTestId("messaging-modal");
    await expectModalVisible(modal);
    await modal.getByRole("link", { name: "Services (0)" }).click();

    // add custom
    await modal.getByRole("button", { name: "Add service" }).click();
    await modal.getByRole("link", { name: "Custom" }).click();

    // validate connection
    await modal.getByRole("button", { name: "Save", exact: true }).click();
    await expectModalHidden(modal);

    // restart button appears
    const restartButton = await page
      .getByTestId("bottom-banner")
      .getByRole("button", { name: "Restart" });
    await expect(restartButton).toBeVisible();

    await restart();
    await page.reload();

    await messagingCard.getByRole("button", { name: "edit" }).click();
    await expectModalVisible(modal);

    await modal.getByRole("link", { name: "Services (1)" }).click();
    await modal.getByRole("button", { name: "Remove" }).click();

    // validate connection
    await modal.getByRole("button", { name: "Save", exact: true }).click();
    await expectModalHidden(modal);

    // restart button appears
    await expect(restartButton).toBeVisible();
    await restart();
    await page.reload();

    await messagingCard.getByRole("button", { name: "edit" }).click();
    await expectModalVisible(modal);

    // ensure no messaging service exists
    await expect(modal.getByRole("link", { name: "Services (0)" })).toBeVisible();
  });
});
