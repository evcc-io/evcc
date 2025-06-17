import { expect, type Page, type Locator } from "@playwright/test";

export async function enableExperimental(page: Page): Promise<void> {
  await page.getByTestId("topnavigation-button").click();
  await page.getByTestId("topnavigation-settings").click();
  await page.getByLabel("Experimental 🧪").click();
  await page.getByRole("button", { name: "Close" }).click();
  await expect(page.locator(".modal-backdrop")).not.toBeVisible();
}

export async function expectModalVisible(modal: Locator): Promise<void> {
  await expect(modal).toBeVisible();
  await expect(modal).toHaveAttribute("aria-hidden", "false");
}

export async function expectModalHidden(modal: Locator): Promise<void> {
  await expect(modal).not.toBeVisible();
  await expect(modal).toHaveAttribute("aria-hidden", "true");
}

export async function editorClear(editor: Locator, iterations = 6): Promise<void> {
  await editor.locator(".view-line").nth(0).click();
  for (let i = 0; i < iterations; i++) {
    await editor.page().keyboard.press("ControlOrMeta+KeyA", { delay: 10 });
    await editor.page().keyboard.press("Backspace", { delay: 10 });
  }
}

export async function editorPaste(editor: Locator, page: Page, text: string): Promise<void> {
  await editor.locator(".view-line").nth(0).click();
  await page.evaluate((text) => navigator.clipboard.writeText(text), text);
  await page.keyboard.press("ControlOrMeta+KeyV", { delay: 50 });
}
