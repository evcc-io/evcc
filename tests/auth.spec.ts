import { test, expect } from "@playwright/test";
import { start, stop, baseUrl } from "./evcc";
import {
  expectModalHidden,
  expectModalVisible,
  login,
  openTopNavigation,
  expectTopNavigationClosed,
} from "./utils";
test.use({ baseURL: baseUrl() });

const BASIC = "basics.evcc.yaml";

test("set initial password", async ({ page }) => {
  await start(BASIC, null, "");
  await page.goto("/");

  const modal = page.getByTestId("password-setup-modal");

  await expectModalVisible(modal);
  await expect(modal.getByRole("heading", { name: "Set Administrator Password" })).toBeVisible();

  // should not be closable via ESC or outside click
  await page.keyboard.press("Escape");
  await expectModalVisible(modal);
  await page.click("body");
  await expectModalVisible(modal);

  // empty password
  await modal.getByRole("button", { name: "Create Password" }).click();
  await expect(modal.getByText("Password should not be empty")).toBeVisible();

  // invalid repeat
  await modal.getByLabel("New password").fill("foo");
  await modal.getByLabel("Repeat password").fill("bar");
  await modal.getByRole("button", { name: "Create Password" }).click();
  await expect(modal.getByText("Passwords do not match")).toBeVisible();

  // success
  await modal.getByLabel("New password").fill("secret");
  await modal.getByLabel("Repeat password").fill("secret");
  await modal.getByRole("button", { name: "Create Password" }).click();
  await expectModalHidden(modal);

  await stop();
});

test("login", async ({ page }) => {
  await start(BASIC, "password.sql", "");
  await page.goto("/");

  // login modal appears immediately when there is no auth cookie
  const loginModal = page.getByTestId("login-modal");
  await expectModalVisible(loginModal);
  await expect(loginModal.getByRole("heading", { name: "Authentication" })).toBeVisible();

  // enter wrong password
  await loginModal.getByLabel("Administrator Password").fill("wrong");
  await loginModal.getByRole("button", { name: "Login" }).click();
  await expect(loginModal.getByText("Password is invalid.")).toBeVisible();

  // enter correct password
  await loginModal.getByLabel("Administrator Password").fill("secret");
  await loginModal.getByRole("button", { name: "Login" }).click();
  await expectModalHidden(loginModal);

  // after login the main ui is accessible without another login prompt
  await openTopNavigation(page);
  await page.getByRole("link", { name: "Configuration" }).click();
  await expect(page.getByRole("heading", { name: "Configuration" })).toBeVisible();

  await stop();
});

test("http iframe hint", async ({ page }) => {
  await start(BASIC, "password.sql", "");
  await page.goto("/");

  // login modal appears immediately when there is no auth cookie
  const loginModal = page.getByTestId("login-modal");
  await expectModalVisible(loginModal);

  // rewrite api call to simulate lost auth cookie (cookie set by login but
  // not readable due to iframe/cross-origin restrictions)
  await page.route("**/api/auth/status", (route) => {
    route.fulfill({ status: 200, body: "false" });
  });

  // enter correct password
  await loginModal.getByLabel("Administrator Password").fill("secret");
  await loginModal.getByRole("button", { name: "Login" }).click();

  // iframe hint visible (login-iframe-hint)
  await expect(loginModal.getByTestId("login-iframe-hint")).toBeVisible();

  await stop();
});

test("update password", async ({ page }) => {
  await start(BASIC, "password.sql", "");

  const oldPassword = "secret";
  const newPassword = "newsecret";

  await page.goto("/#/config");
  await login(page, oldPassword);

  // update password
  await page.getByTestId("generalconfig-password").getByRole("button", { name: "edit" }).click();
  const modal = page.getByTestId("password-update-modal");
  await expectModalVisible(modal);
  await expect(modal.getByRole("heading", { name: "Update Administrator Password" })).toBeVisible();
  await modal.getByLabel("Current password").fill(oldPassword);
  await modal.getByLabel("New password").fill(newPassword);
  await modal.getByLabel("Repeat password").fill(newPassword);
  await modal.getByRole("button", { name: "Update Password" }).click();
  await expect(
    modal.getByRole("heading", { name: "Update Administrator Password" })
  ).not.toBeVisible();

  // logout
  await openTopNavigation(page);
  await page.getByRole("button", { name: "Logout" }).click();
  await expectTopNavigationClosed(page);

  // should be redirected to home page after logout
  await expect(page).toHaveURL("/#/");

  // login modal
  await openTopNavigation(page);
  await expect(page.getByRole("button", { name: "Logout" })).not.toBeVisible();
  await page.getByRole("link", { name: "Configuration" }).click();
  await expectTopNavigationClosed(page);
  await login(page, newPassword);
  await expect(page.getByRole("heading", { name: "Configuration" })).toBeVisible();

  // revert to old password
  await page.getByTestId("generalconfig-password").getByRole("button", { name: "edit" }).click();
  await expectModalVisible(modal);
  await modal.getByLabel("Current password").fill(newPassword);
  await modal.getByLabel("New password").fill(oldPassword);
  await modal.getByLabel("Repeat password").fill(oldPassword);
  await modal.getByRole("button", { name: "Update Password" }).click();
  await expect(
    modal.getByRole("heading", { name: "Update Administrator Password" })
  ).not.toBeVisible();

  await stop();
});

test("disable auth", async ({ page }) => {
  await start(BASIC, null, "--disable-auth");
  await page.goto("/");

  // no password modal
  const modal = page.getByTestId("password-setup-modal");
  await expectModalHidden(modal);

  // configuration page without login
  await openTopNavigation(page);
  await page.getByRole("link", { name: "Configuration" }).click();
  await expectTopNavigationClosed(page);
  await expect(page.getByRole("heading", { name: "Configuration" })).toBeVisible();

  await stop();
});
