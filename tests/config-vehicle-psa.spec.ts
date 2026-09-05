import { test, expect } from "@playwright/test";
import { start, stop, baseUrl } from "./evcc";
import { expectModalVisible } from "./utils";

test.use({ baseURL: baseUrl() });
test.beforeEach(async () => {
  await start(undefined, undefined, ["--disable-auth"]);
});
test.afterEach(async () => {
  await stop();
});

test.describe("PSA login", () => {
  for (const brand of ["Citroën", "DS", "Opel", "Peugeot"]) {
    test(brand, async ({ page }) => {
      await page.goto("/#/config");
      await page.getByTestId("add-vehicle").click();
      const modal = page.getByTestId("vehicle-modal");
      await expectModalVisible(modal);
      await modal.getByLabel("Manufacturer").selectOption(brand);
      await expect(modal.getByLabel("Country code")).toHaveValue("de");
      await modal.getByLabel("Country code").fill("gb");
      await modal.getByLabel("Username").fill("driver@example.com");
      await modal.getByRole("button", { name: "Prepare connection" }).click();

      const login = modal.getByRole("link", { name: "Open login page" });
      await expect(login).toBeVisible();
      const href = await login.getAttribute("href");
      expect(new URL(href!).searchParams.get("redirect_uri")).toMatch(/\/gb$/);
      await expect(
        modal.getByLabel("Paste the code or the address of the page you were redirected to.")
      ).toBeVisible();
      await expect(modal.getByRole("button", { name: "Validate & save" })).not.toBeVisible();
    });
  }
});
