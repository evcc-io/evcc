import { test, expect } from "@playwright/test";
import { start, stop, baseUrl } from "./evcc";

test.use({ baseURL: baseUrl() });

test.beforeEach(async ({ page }) => {
  await start("basics.evcc.yaml");
  await page.goto("/");
});

test.afterEach(async () => {
  await stop();
});

test("set initial password", async ({ page }) => {
  const modal = page.getByTestId("password-modal");

  await expect(modal).toBeVisible();
  await expect(modal.getByRole("heading", { name: "Set Administrator Password" })).toBeVisible();

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
  await expect(modal).not.toBeVisible();
});

test("login", async ({ page }) => {
  // set initial password
  const modal = page.getByTestId("password-modal");
  await modal.getByLabel("New password").fill("secret");
  await modal.getByLabel("Repeat password").fill("secret");
  await modal.getByRole("button", { name: "Create Password" }).click();

  // go to config
  await page.getByTestId("topnavigation-button").click();
  await page.getByRole("link", { name: "Configuration" }).click();

  // login modal
  const login = page.getByTestId("login-modal");
  await expect(login).toBeVisible();
  await expect(login.getByRole("heading", { name: "Authentication" })).toBeVisible();

  // enter wrong password
  await login.getByLabel("Password").fill("wrong");
  await login.getByRole("button", { name: "Login" }).click();
  await expect(login.getByText("Login failed: Password is invalid.")).toBeVisible();

  // enter correct password
  await login.getByLabel("Password").fill("secret");
  await login.getByRole("button", { name: "Login" }).click();
  await expect(login).not.toBeVisible();
  await expect(page.getByRole("heading", { name: "Configuration" })).toBeVisible();
});

test("http iframe hint", async ({ page }) => {
  // set initial password
  const modal = page.getByTestId("password-modal");
  await modal.getByLabel("New password").fill("secret");
  await modal.getByLabel("Repeat password").fill("secret");
  await modal.getByRole("button", { name: "Create Password" }).click();

  // go to config
  await page.getByTestId("topnavigation-button").click();
  await page.getByRole("link", { name: "Configuration" }).click();

  // login modal
  const login = page.getByTestId("login-modal");
  await expect(login).toBeVisible();
  await expect(login.getByRole("heading", { name: "Authentication" })).toBeVisible();

  // rewrite api call to simulate lost auth cookie
  await page.route("**/api/auth/status", (route) => {
    route.fulfill({ status: 200, body: "false" });
  });

  // enter correct password
  await login.getByLabel("Password").fill("secret");
  await login.getByRole("button", { name: "Login" }).click();

  // iframe hint visible (login-iframe-hint)
  await expect(login.getByTestId("login-iframe-hint")).toBeVisible();
});

test("update password", async ({ page }) => {
  const oldPassword = "secret";
  const newPassword = "newsecret";

  // set initial password
  const modal = page.getByTestId("password-modal");
  await modal.getByLabel("New password").fill(oldPassword);
  await modal.getByLabel("Repeat password").fill(oldPassword);
  await modal.getByRole("button", { name: "Create Password" }).click();

  // login modal
  page.goto("/#/config");
  const loginOld = page.getByTestId("login-modal");
  await loginOld.getByLabel("Password").fill(oldPassword);
  await loginOld.getByRole("button", { name: "Login" }).click();

  // update password
  await page.getByTestId("generalconfig-password").getByRole("button", { name: "edit" }).click();
  await expect(modal.getByRole("heading", { name: "Update Administrator Password" })).toBeVisible();
  await modal.getByLabel("Current password").fill(oldPassword);
  await modal.getByLabel("New password").fill(newPassword);
  await modal.getByLabel("Repeat password").fill(newPassword);
  await modal.getByRole("button", { name: "Update Password" }).click();
  await expect(
    modal.getByRole("heading", { name: "Update Administrator Password" })
  ).not.toBeVisible();

  // logout
  await page.getByTestId("topnavigation-button").click();
  await page.getByRole("button", { name: "Logout" }).click();

  // login modal
  await page.getByTestId("topnavigation-button").click();
  await expect(page.getByRole("button", { name: "Logout" })).not.toBeVisible();
  await page.getByRole("link", { name: "Configuration" }).click();
  const loginNew = page.getByTestId("login-modal");
  await loginNew.getByLabel("Password").fill(newPassword);
  await loginNew.getByRole("button", { name: "Login" }).click();
  await expect(page.getByRole("heading", { name: "Configuration" })).toBeVisible();
  await expect(loginNew).not.toBeVisible();

  // revert password
  await page.getByTestId("generalconfig-password").getByRole("button", { name: "edit" }).click();
  await modal.getByLabel("Current password").fill(newPassword);
  await modal.getByLabel("New password").fill(oldPassword);
  await modal.getByLabel("Repeat password").fill(oldPassword);
  await modal.getByRole("button", { name: "Update Password" }).click();
  await expect(page.getByTestId("password-modal")).not.toBeVisible();
});
