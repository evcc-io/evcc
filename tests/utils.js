import { expect } from "@playwright/test";

export async function enableExperimental(page) {
  await page.getByTestId("topnavigation-button").click();
  await page.getByTestId("topnavigation-settings").click();
  await page.getByLabel("Experimental 🧪").click();
  await page.getByRole("button", { name: "Close" }).click();
  await expect(page.locator(".modal-backdrop")).not.toBeVisible();
}
