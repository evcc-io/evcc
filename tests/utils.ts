import { expect, type Page, type Locator } from "@playwright/test";

export async function enableExperimental(page: Page, inline = true): Promise<void> {
  if (inline) {
    await page.getByRole("button", { name: "Enable Experimental Features" }).click();
  } else {
    await openTopNavigation(page);
    await page.getByTestId("topnavigation-settings").click();
    const modal = page.getByTestId("global-settings-modal");
    await expectModalVisible(modal);
    await modal.getByLabel("Experimental 🧪").click();
    await modal.getByRole("button", { name: "Close" }).click();
    await expectModalHidden(modal);
    await expect(page.locator(".modal-backdrop")).not.toBeVisible();
  }
}

export async function openTopNavigation(page: Page): Promise<void> {
  await expect(page.getByTestId("topnavigation-button")).toBeVisible();
  await page.getByTestId("topnavigation-button").click();
  await expectTopNavigationOpened(page);
}

export async function closeTopNavigation(page: Page): Promise<void> {
  await expectTopNavigationOpened(page);
  await page.getByTestId("topnavigation-button").click();
  await expectTopNavigationClosed(page);
}

export async function expectTopNavigationOpened(page: Page): Promise<void> {
  await expect(page.getByTestId("topnavigation-button")).toHaveAttribute("aria-expanded", "true");
  await expect(page.getByTestId("topnavigation-dropdown")).toBeVisible();
}

export async function expectTopNavigationClosed(page: Page): Promise<void> {
  await expect(page.getByTestId("topnavigation-button")).toHaveAttribute("aria-expanded", "false");
  await expect(page.getByTestId("topnavigation-dropdown")).not.toBeVisible();
}

export async function expectModalVisible(modal: Locator): Promise<void> {
  await expect(modal).toBeVisible();
  await expect(modal).toHaveAttribute("aria-hidden", "false");
}

export async function expectModalHidden(modal: Locator): Promise<void> {
  await expect(modal).not.toBeVisible();
  await expect(modal).toHaveAttribute("aria-hidden", "true");
}

export async function editorClear(editor: Locator, iterations = 10): Promise<void> {
  for (let i = 0; i < iterations; i++) {
    await editor.locator(".view-line").nth(0).click();
    await editor.page().keyboard.press("ControlOrMeta+KeyA", { delay: 10 });
    await editor.page().keyboard.press("Backspace", { delay: 10 });
  }
}

export async function editorPaste(editor: Locator, page: Page, text: string): Promise<void> {
  await editor.locator(".view-line").nth(0).click();
  await page.evaluate((text) => navigator.clipboard.writeText(text), text);
  await page.keyboard.press("ControlOrMeta+KeyV", { delay: 50 });
}
