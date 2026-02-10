import { test, expect } from "@playwright/test";
import { start, stop, restart, baseUrl } from "./evcc";
import { expectModalHidden, expectModalVisible, editorClear, editorPaste } from "./utils";

const CONFIG_WITH_MESSAGING = "config-messaging.evcc.yaml";
const CONFIG_MESSAGING_LEGACY = "messaging-legacy.sql";

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

  test("messaging from yaml ui (legacy)", async ({ page }) => {
    await start(undefined, CONFIG_MESSAGING_LEGACY);
    await page.goto("/#/config");

    const messagingCard = page.getByTestId("messaging");
    await expect(messagingCard).toBeVisible();
    await expect(messagingCard).toContainText(["Configured", "yes"].join(""));

    await messagingCard.getByRole("button", { name: "edit" }).click();
    const modal = page.getByTestId("messaging-legacy-modal");
    await expectModalVisible(modal);

    // check for new configuration notice
    await expect(modal.getByRole("alert")).toContainText("New messaging configuration available");

    const validYaml = `events:
  start:
    title: Charge started
    msg: Started charging`;

    // default content
    const editor = modal.getByTestId("yaml-editor");
    await expect(editor).toContainText(validYaml);

    // clear and enter invalid yaml
    await editorClear(editor);
    await editorPaste(editor, page, "foo: bar");
    await page.getByRole("button", { name: "Save" }).click();
    await expect(modal.getByTestId("error")).toContainText("invalid keys: foo");

    // clear and enter valid yaml
    await editorClear(editor);
    await editorPaste(editor, page, validYaml);

    await page.getByRole("button", { name: "Save" }).click();
    await expect(modal.getByTestId("error")).not.toBeVisible();

    // modal closes
    await expectModalHidden(modal);

    // restart button appears
    const restartButton = page
      .getByTestId("bottom-banner")
      .getByRole("button", { name: "Restart" });
    await expect(restartButton).toBeVisible();

    await restart();

    // restart done
    await expect(restartButton).not.toBeVisible();

    await expect(messagingCard).toBeVisible();
    await expect(messagingCard).toContainText(["Configured", "yes"].join(""));
  });

  test("messaging from evcc.yaml", async ({ page }) => {
    await start(CONFIG_WITH_MESSAGING);
    await page.goto("/#/config");

    const messagingCard = page.getByTestId("messaging");
    await expect(messagingCard).toBeVisible();
    await expect(messagingCard).toContainText(["Configured", "yes"].join(""));
  });
  test("messaging events via ui", async ({ page }) => {
    await start();
    await page.goto("/#/config");

    const messagingCard = page.getByTestId("messaging");
    await expect(messagingCard).toBeVisible();
    await expect(messagingCard).toContainText(["Configured", "no"].join(""));

    await messagingCard.getByRole("button", { name: "edit" }).click();
    const modal = page.getByTestId("messaging-modal");
    await expectModalVisible(modal);

    // validate start event
    const eventStart = modal.getByTestId("event-start");
    const switchInput = eventStart.getByRole("switch");
    const titleInput = eventStart.getByLabel("Title");
    const messageInput = eventStart.getByLabel("Message");

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
    await expect(messagingCard).toContainText(["Events", "1", "Messengers", "0"].join(""));

    // restart button appears
    const restartButton = page
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
  test("create, verify, delete custom service via ui", async ({ page }) => {
    await start();
    await page.goto("/#/config");

    const messagingCard = page.getByTestId("messaging");
    await expect(messagingCard).toBeVisible();
    await expect(messagingCard).toContainText(["Configured", "no"].join(""));

    await messagingCard.getByRole("button", { name: "edit" }).click();
    const messagingModal = page.getByTestId("messaging-modal");
    await expectModalVisible(messagingModal);

    await messagingModal.getByRole("link", { name: "Messengers (0)" }).click();
    await messagingModal.getByRole("button", { name: "Add messenger" }).click();

    const messengerModal = page.getByTestId("messenger-modal");
    await expectModalHidden(messagingModal);
    await expectModalVisible(messengerModal);

    await messengerModal.getByLabel("Messenger").selectOption("User-defined device");
    await messengerModal.getByRole("button", { name: "Validate & save" }).click();

    await expectModalHidden(messengerModal);
    await expectModalVisible(messagingModal);

    const messengerBox = messagingModal.getByTestId("messenger-box-0");
    await expect(messengerBox).toHaveText(["#1", "User-defined device"].join(""));

    await messagingModal.getByRole("button", { name: "Close" }).click();
    await expectModalHidden(messagingModal);
    await expect(messagingCard).toContainText(["Events", "0", "Messengers", "1"].join(""));

    await messagingCard.getByRole("button", { name: "edit" }).click();
    await messagingModal.getByRole("button", { name: "edit" }).click();
    await messengerModal.getByRole("button", { name: "Delete" }).click();
    await expect(messengerBox).not.toBeVisible();
    await messagingModal.getByRole("button", { name: "Close" }).click();

    await expectModalHidden(messengerModal);
    await expectModalHidden(messagingModal);

    await expect(messagingCard).toContainText(["Configured", "no"].join(""));
  });
});
