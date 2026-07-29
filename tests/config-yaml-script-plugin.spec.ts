import { test, expect, type Page } from "@playwright/test";
import { start, stop, baseUrl } from "./evcc";
import { expectModalVisible, expectModalHidden, editorClear, editorPaste } from "./utils";

test.use({ baseURL: baseUrl() });

test.afterEach(async () => {
  await stop();
});

// circuits config is a yaml list, exercising the list-shaped critical-plugin detection
const SCRIPT_YAML = `- name: main
  getmaxcurrent:
    source: script
    cmd: echo 9999`;

const UPDATED_YAML = `- name: main
  getmaxcurrent:
    source: script
    cmd: echo 8888`;

async function login(page: Page) {
  const loginModal = page.getByTestId("login-modal");
  await expectModalVisible(loginModal);
  await loginModal.getByLabel("Administrator Password").fill("secret");
  await loginModal.getByRole("button", { name: "Login" }).click();
  await expectModalHidden(loginModal);
}

async function openCircuitsModal(page: Page, yaml: string) {
  await page.getByTestId("circuits").getByRole("button", { name: "edit" }).click();
  const modal = page.getByTestId("circuits-modal");
  await expectModalVisible(modal);
  const editor = modal.getByTestId("yaml-editor");
  await expect(editor).toBeVisible();
  await editorClear(editor);
  await editorPaste(editor, page, yaml);
  return modal;
}

test.describe("yaml config with script plugin requires admin password", async () => {
  test("caches password across save and reopen", async ({ page }) => {
    await start(undefined, "password.sql", "");
    await page.goto("/#/config");
    await login(page);

    const modal = await openCircuitsModal(page, SCRIPT_YAML);
    const prompt = modal.getByTestId("admin-password-prompt");
    const save = modal.getByRole("button", { name: "Save" });

    // save without password reveals the field, modal stays open
    await save.click();
    await expect(prompt).toBeVisible();
    await expectModalVisible(modal);

    // wrong password keeps the field with an invalid hint
    await prompt.getByLabel("Administrator Password").fill("wrong");
    await save.click();
    await expect(prompt.getByText("Invalid password. Please try again.")).toBeVisible();

    // correct password saves and closes the modal
    await prompt.getByLabel("Administrator Password").fill("secret");
    await save.click();
    await expectModalHidden(modal);

    // reopen and edit: cached password is reused, no prompt
    const reopened = await openCircuitsModal(page, UPDATED_YAML);
    await reopened.getByRole("button", { name: "Save" }).click();
    await expect(reopened.getByTestId("admin-password-prompt")).not.toBeVisible();
    await expectModalHidden(reopened);
  });
});
