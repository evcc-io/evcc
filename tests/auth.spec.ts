import { test, expect } from "@playwright/test";
import { start, stop, baseUrl } from "./evcc";
import {
  expectModalHidden,
  expectModalVisible,
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

  // go to config
  await openTopNavigation(page);
  await page.getByRole("link", { name: "Configuration" }).click();
  await expectTopNavigationClosed(page);

  // login modal
  const login = page.getByTestId("login-modal");
  await expectModalVisible(login);
  await expect(login.getByRole("heading", { name: "Authentication" })).toBeVisible();

  // enter wrong password
  await login.getByLabel("Administrator Password").fill("wrong");
  await login.getByRole("button", { name: "Login" }).click();
  await expect(login.getByText("Password is invalid.")).toBeVisible();

  // enter correct password
  await login.getByLabel("Administrator Password").fill("secret");
  await login.getByRole("button", { name: "Login" }).click();
  await expectModalHidden(login);
  await expect(page.getByRole("heading", { name: "Configuration" })).toBeVisible();

  await stop();
});

test("http iframe hint", async ({ page }) => {
  await start(BASIC, "password.sql", "");
  await page.goto("/");

  // go to config
  await openTopNavigation(page);
  await page.getByRole("link", { name: "Configuration" }).click();
  await expectTopNavigationClosed(page);

  // login modal
  const login = page.getByTestId("login-modal");
  await expectModalVisible(login);
  await expect(login.getByRole("heading", { name: "Authentication" })).toBeVisible();

  // rewrite api call to simulate lost auth cookie
  await page.route("**/api/auth/status", (route) => {
    route.fulfill({ status: 200, body: "false" });
  });

  // enter correct password
  await login.getByLabel("Administrator Password").fill("secret");
  await login.getByRole("button", { name: "Login" }).click();

  // iframe hint visible (login-iframe-hint)
  await expect(login.getByTestId("login-iframe-hint")).toBeVisible();

  await stop();
});

test("update password", async ({ page }) => {
  await start(BASIC, "password.sql", "");

  const oldPassword = "secret";
  const newPassword = "newsecret";

  // login modal
  await page.goto("/#/config");
  const loginModal = page.getByTestId("login-modal");
  await expectModalVisible(loginModal);
  await loginModal.getByLabel("Administrator Password").fill(oldPassword);
  await loginModal.getByRole("button", { name: "Login" }).click();
  await expectModalHidden(loginModal);

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

  // login modal
  await openTopNavigation(page);
  await expect(page.getByRole("button", { name: "Logout" })).not.toBeVisible();
  await page.getByRole("link", { name: "Configuration" }).click();
  await expectTopNavigationClosed(page);
  const loginNew = page.getByTestId("login-modal");
  await expectModalVisible(loginNew);
  await loginNew.getByLabel("Administrator Password").fill(newPassword);
  await loginNew.getByRole("button", { name: "Login" }).click();
  await expect(page.getByRole("heading", { name: "Configuration" })).toBeVisible();
  await expectModalHidden(loginNew);

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
