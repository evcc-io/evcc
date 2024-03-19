const { test, expect } = require("@playwright/test");
const { start, stop } = require("./evcc");

test.beforeEach(async ({ page }) => {
  await start("basics.evcc.yaml");
  await page.goto("/");
});

test.afterEach(async () => {
  await stop();
});

test.describe("password", async () => {
  test("set initial password", async ({ page }) => {
    const modal = page.getByTestId("password-modal");

    await expect(modal).toBeVisible();
    await expect(modal.getByRole("heading", { name: "Set Administrator Password" })).toBeVisible();

    // empty password
    await modal.getByRole("button", { name: "Set Password" }).click();
    await expect(modal.getByText("Password should not be empty")).toBeVisible();

    // invalid repeat
    await modal.getByLabel("Password", { exact: true }).fill("foo");
    await modal.getByLabel("Repeat password", { exact: true }).fill("bar");
    await modal.getByRole("button", { name: "Set Password" }).click();
    await expect(modal.getByText("Passwords do not match")).toBeVisible();

    // success
    await modal.getByLabel("Password", { exact: true }).fill("secret");
    await modal.getByLabel("Repeat password", { exact: true }).fill("secret");
    await modal.getByRole("button", { name: "Set Password" }).click();
    await expect(modal).not.toBeVisible();
  });

  test("login", async ({ page }) => {
    // set initial password
    const modal = page.getByTestId("password-modal");
    await modal.getByLabel("Password", { exact: true }).fill("secret");
    await modal.getByLabel("Repeat password", { exact: true }).fill("secret");
    await modal.getByRole("button", { name: "Set Password" }).click();

    // go to config
    await page.getByTestId("topnavigation-button").click();
    await page.getByRole("button", { name: "Settings" }).click();
    await page.getByLabel("Experimental 🧪").click();
    await page.getByRole("button", { name: "Close" }).click();
    await page.getByTestId("topnavigation-button").click();
    await page.getByRole("link", { name: "Configuration" }).click();

    // login modal
    const login = page.getByTestId("login-modal");
    await expect(login).toBeVisible();
    await expect(login.getByRole("heading", { name: "Authentication" })).toBeVisible();

    // enter wrong password
    await login.getByLabel("Password", { exact: true }).fill("wrong");
    await login.getByRole("button", { name: "Login" }).click();
    await expect(login.getByText("Login failed: Password is invalid.")).toBeVisible();

    // enter correct password
    await login.getByLabel("Password", { exact: true }).fill("secret");
    await login.getByRole("button", { name: "Login" }).click();
    await expect(login).not.toBeVisible();
    await expect(page.getByRole("heading", { name: "Configuration" })).toBeVisible();
  });
});
