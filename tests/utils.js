import { expect } from "@playwright/test";

export async function enableExperimental(page) {
  await page.getByTestId("topnavigation-button").click();
  await page.getByTestId("topnavigation-settings").click();
  await page.getByLabel("Experimental 🧪").click();
  await page.getByRole("button", { name: "Close" }).click();
  await expect(page.locator(".modal-backdrop")).not.toBeVisible();
}

export async function expectModalVisible(modal) {
  await expect(modal).toBeVisible();
  await expect(modal).toHaveAttribute("aria-hidden", "false");
}

export async function expectModalHidden(modal) {
  await expect(modal).not.toBeVisible();
  await expect(modal).toHaveAttribute("aria-hidden", "true");
}
