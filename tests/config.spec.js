import { test, expect } from "@playwright/test";
import { start, stop, baseUrl } from "./evcc";

const CONFIG_GRID_ONLY = "config-grid-only.evcc.yaml";

test.use({ baseURL: baseUrl() });

test.beforeAll(async () => {
  await start(CONFIG_GRID_ONLY, "password.sql");
});
test.afterAll(async () => {
  await stop();
});

async function login(page) {
  await page.locator("#loginPassword").fill("secret");
  await page.getByRole("button", { name: "Login" }).click();
}

async function enableExperimental(page) {
  await page
    .getByTestId("generalconfig-experimental")
    .getByRole("button", { name: "edit" })
    .click();
  await page.getByLabel("Experimental 🧪").click();
  await page.getByRole("button", { name: "Close" }).click();
}

test.describe("basics", async () => {
  test("navigation to config", async ({ page }) => {
    await page.goto("/");
    await page.getByTestId("topnavigation-button").click();
    await page.getByRole("link", { name: "Configuration" }).click();
    await login(page);
    await expect(page.getByRole("heading", { name: "Configuration" })).toBeVisible();
  });
  test.skip("alert box should always be visible", async ({ page }) => {
    await page.goto("/#/config");
    await login(page);
    await enableExperimental(page);
    await expect(page.getByRole("alert")).toBeVisible();
  });
});

test.describe("general", async () => {
  test("change site title", async ({ page }) => {
    // initial value on main ui
    await page.goto("/");
    await expect(page.getByRole("heading", { name: "Hello World" })).toBeVisible();

    // change value in config
    await page.goto("/#/config");
    await login(page);
    await enableExperimental(page);

    await expect(page.getByTestId("generalconfig-title")).toContainText("Hello World");
    await page.getByTestId("generalconfig-title").getByRole("button", { name: "edit" }).click();
    const modal = page.getByTestId("title-modal");
    await expect(modal).toBeVisible();
    await modal.getByLabel("Title").fill("Whoops World");

    // close modal and ignore entry on cancel
    await modal.getByRole("button", { name: "Cancel" }).click();
    await expect(modal).not.toBeVisible();
    await expect(page.getByTestId("generalconfig-title")).toContainText("Hello World");

    // change and save value
    await page.getByTestId("generalconfig-title").getByRole("button", { name: "edit" }).click();
    await modal.getByLabel("Title").fill("Ahoy World");
    await modal.getByRole("button", { name: "Save" }).click();
    await expect(modal).not.toBeVisible();
    await expect(page.getByTestId("generalconfig-title")).toContainText("Ahoy World");

    // check changed value on main ui
    await page.getByTestId("home-link").click();
    await expect(page.getByRole("heading", { name: "Ahoy World" })).toBeVisible();
  });
});
