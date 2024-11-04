import { expect } from "@playwright/test";

export async function enableExperimental(page) {
  await page.getByTestId("topnavigation-button").click();
  await page.getByTestId("topnavigation-settings").click();
  await page.getByLabel("Experimental 🧪").click();
  await page.getByRole("button", { name: "Close" }).click();
  await expect(page.locator(".modal-backdrop")).not.toBeVisible();
}

export async function login(page) {
  await page.locator("#loginPassword").fill("secret");
  await page.getByRole("button", { name: "Login" }).click();
  await expect(page.locator("#loginPassword")).not.toBeVisible();
}
