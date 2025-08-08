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
  // First wait for any potential API calls or operations to complete
  await modal.page().waitForLoadState("networkidle", { timeout: 5000 });
  
  // Wait for Bootstrap modal to complete hiding animation with extended timeout
  await expect(modal).not.toBeVisible({ timeout: 15000 });
  
  // Give additional time for aria-hidden attribute to be set
  await expect(modal).toHaveAttribute("aria-hidden", "true", { timeout: 5000 });
}

export async function waitForModalToHide(modal: Locator): Promise<void> {
  // More robust approach: wait for the modal to lose the 'show' class
  // which indicates Bootstrap has finished the hiding animation
  await modal.waitFor({ state: "hidden", timeout: 10000 });
  
  // Ensure the modal backdrop is also gone
  const page = modal.page();
  await expect(page.locator(".modal-backdrop")).not.toBeVisible({ timeout: 5000 });
}

export async function expectModalHiddenAfterSave(modal: Locator): Promise<void> {
  // Wait for save operation to complete by checking for network idle
  await modal.page().waitForLoadState("networkidle", { timeout: 10000 });
  
  // Wait for any loading states or spinners to disappear
  await modal.page().waitForTimeout(500);
  
  // Now wait for the modal to hide
  await expectModalHidden(modal);
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

export enum LoadpointType {
  Charging = "charging",
  Heating = "heating",
}

export async function addDemoCharger(
  page: Page,
  type: LoadpointType = LoadpointType.Charging
): Promise<void> {
  const lpModal = page.getByTestId("loadpoint-modal");
  await lpModal
    .getByRole("button", { name: type === LoadpointType.Heating ? "Add heater" : "Add charger" })
    .click();

  const modal = page.getByTestId("charger-modal");
  await expectModalVisible(modal);
  await modal
    .getByLabel("Manufacturer")
    .selectOption(type === LoadpointType.Heating ? "Demo heat pump" : "Demo charger");
  await modal.getByRole("button", { name: "Save" }).click();
  await expectModalHidden(modal);
  await expectModalVisible(lpModal);
}

export async function addDemoMeter(page: Page, power = "0"): Promise<void> {
  const lpModal = page.getByTestId("loadpoint-modal");
  await lpModal.getByRole("button", { name: "Add dedicated energy meter" }).click();

  const modal = page.getByTestId("meter-modal");
  await expectModalVisible(modal);
  await modal.getByLabel("Manufacturer").selectOption("Demo meter");
  await modal.getByLabel("Power").fill(power);
  await modal.getByRole("button", { name: "Save" }).click();
  await expectModalHidden(modal);
  await expectModalVisible(lpModal);
}

export async function addVehicle(page: Page, title: string): Promise<void> {
  await page.getByRole("button", { name: "Add vehicle" }).click();
  const modal = page.getByTestId("vehicle-modal");
  await expectModalVisible(modal);
  await modal.getByLabel("Manufacturer").selectOption("Generic vehicle (without API)");
  await modal.getByLabel("Title").fill(title);
  await modal.getByRole("button", { name: "Validate & save" }).click();
  await expectModalHidden(modal);
}

export async function newLoadpoint(
  page: Page,
  title: string,
  type: LoadpointType = LoadpointType.Charging
): Promise<void> {
  const lpModal = page.getByTestId("loadpoint-modal");
  await page.getByRole("button", { name: "Add charger or heater" }).click();
  await expectModalVisible(lpModal);
  await lpModal
    .getByRole("button", {
      name: type === LoadpointType.Heating ? "Add heating device" : "Add charging point",
    })
    .click();
  await lpModal.getByLabel("Title").fill(title);
}
