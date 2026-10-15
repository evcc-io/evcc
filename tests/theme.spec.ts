import { test, expect } from "@playwright/test";
import { start, stop, baseUrl } from "./evcc";
import { openMoreMenu, expectModalVisible } from "./utils";

test.use({ baseURL: baseUrl() });

test.beforeAll(async () => {
  await start("basics.evcc.yaml", null, ["--disable-auth", "--custom-theme", "dark"]);
});

test.afterAll(async () => {
  await stop();
});

test.describe("custom theme", async () => {
  test("starts dark, user choice wins and survives reload", async ({ page }) => {
    await page.goto("/");
    await expect(page.locator("html")).toHaveAttribute("data-bs-theme", "dark");

    const menu = await openMoreMenu(page);
    await menu.getByRole("button", { name: "User Interface" }).click();
    const modal = page.getByTestId("global-settings-modal");
    await expectModalVisible(modal);
    await expect(modal.getByRole("radio", { name: "dark" })).toHaveAttribute(
      "aria-checked",
      "true"
    );

    await modal.getByRole("radio", { name: "light" }).click();
    await expect(page.locator("html")).toHaveAttribute("data-bs-theme", "light");

    await page.reload();
    await expect(page.locator("html")).toHaveAttribute("data-bs-theme", "light");
  });
});
