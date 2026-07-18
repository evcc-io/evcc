import { test, expect, type Page } from "@playwright/test";
import { start, stop, baseUrl } from "./evcc";
import { expectModalVisible, expectModalHidden, editorClear, editorPaste } from "./utils";

test.use({ baseURL: baseUrl() });

test.afterEach(async () => {
  await stop();
});

const SCRIPT_YAML = `power:
  source: script
  cmd: echo 9999`;

const UPDATED_YAML = `power:
  source: script
  cmd: echo 8888`;

async function login(page: Page) {
  const loginModal = page.getByTestId("login-modal");
  await expectModalVisible(loginModal);
  await loginModal.getByLabel("Administrator Password").fill("secret");
  await loginModal.getByRole("button", { name: "Login" }).click();
  await expectModalHidden(loginModal);
}

async function addCustomGridMeter(page: Page, yaml: string) {
  await page.getByRole("button", { name: "Add grid meter" }).click();
  const modal = page.getByTestId("meter-modal");
  await expectModalVisible(modal);
  await modal.getByLabel("Manufacturer").selectOption("User-defined device");
  const editor = modal.getByTestId("yaml-editor");
  await expect(editor).toBeVisible();
  await editorClear(editor);
  await editorPaste(editor, page, yaml);
  return modal;
}

test.describe("script plugin requires admin password", async () => {
  test("caches password across validate and create", async ({ page }) => {
    await start(undefined, "password.sql", "");
    await page.goto("/#/config");
    await login(page);

    const meterModal = await addCustomGridMeter(page, SCRIPT_YAML);
    const prompt = meterModal.getByTestId("admin-password-prompt");
    const validate = meterModal.getByTestId("test-result").getByRole("link", { name: "validate" });

    // validate without password reveals the field
    await validate.click();
    await expect(prompt).toBeVisible();

    // editing the config hides the field again
    await editorClear(meterModal.getByTestId("yaml-editor"));
    await editorPaste(meterModal.getByTestId("yaml-editor"), page, SCRIPT_YAML);
    await expect(prompt).not.toBeVisible();

    // wrong password keeps the field with an invalid hint
    await validate.click();
    await expect(prompt).toBeVisible();
    await prompt.getByLabel("Administrator Password").fill("wrong");
    await validate.click();
    await expect(prompt.getByText("Invalid password. Please try again.")).toBeVisible();

    // correct password validates and hides the field
    await prompt.getByLabel("Administrator Password").fill("secret");
    await validate.click();
    await expect(meterModal.getByTestId("test-result")).toContainText("Status: successful");
    await expect(prompt).not.toBeVisible();

    // save reuses the cached password, no field reappears
    await meterModal.getByRole("button", { name: "Save" }).click();
    await expect(prompt).not.toBeVisible();
    await expectModalHidden(meterModal);
    await expect(page.getByTestId("grid")).toBeVisible();
  });

  test("re-prompts on update after reload", async ({ page }) => {
    await start(undefined, "password.sql", "");
    await page.goto("/#/config");
    await login(page);

    // create a script meter (enter the password once)
    const meterModal = await addCustomGridMeter(page, SCRIPT_YAML);
    const prompt = meterModal.getByTestId("admin-password-prompt");
    await meterModal.getByRole("button", { name: "Save" }).click();
    await expect(prompt).toBeVisible();
    await prompt.getByLabel("Administrator Password").fill("secret");
    await meterModal.getByRole("button", { name: "Save" }).click();
    await expectModalHidden(meterModal);
    await expect(page.getByTestId("grid")).toBeVisible();

    // reload drops the cached password
    await page.reload();

    // editing and saving the existing device prompts again
    await page.getByTestId("grid").getByRole("button", { name: "edit" }).click();
    await expectModalVisible(meterModal);
    await editorClear(meterModal.getByTestId("yaml-editor"));
    await editorPaste(meterModal.getByTestId("yaml-editor"), page, UPDATED_YAML);
    await meterModal.getByRole("button", { name: "Save" }).click();
    await expect(prompt).toBeVisible();
    await prompt.getByLabel("Administrator Password").fill("secret");
    await meterModal.getByRole("button", { name: "Save" }).click();
    await expectModalHidden(meterModal);
  });
});
