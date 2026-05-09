import { test, expect } from "@playwright/test";
import { start, stop, baseUrl } from "./evcc";
import { startSimulator, stopSimulator, simulatorConfig } from "./simulator";
import { openMoreMenu } from "./utils";

const BASICS_CONFIG = "basics.evcc.yaml";

const UI_ROUTES = [
  { path: "/" },
  { path: "/#/sessions", title: "Sessions" },
  { path: "/#/config", title: "Configuration" },
];

test.use({ baseURL: baseUrl() });

test.describe("Basics", async () => {
  test.beforeAll(async () => {
    await start(BASICS_CONFIG);
  });

  const title = "Hello World";

  test.afterAll(async () => {
    await stop();
  });

  test("Menu options. No battery and grid.", async ({ page }) => {
    for (const route of UI_ROUTES) {
      await page.goto(route.path);

      await expect(page.getByRole("heading", { name: route.title || title })).toBeVisible();
      // no battery tab when battery is not configured
      await expect(page.getByRole("link", { name: "Battery" })).not.toBeVisible();
      const menu = await openMoreMenu(page);
      await expect(menu.getByRole("button", { name: "User Interface" })).toBeVisible();
      await expect(menu.getByRole("button", { name: "Need Help?" })).toBeVisible();
    }
  });

  test("Need help?", async ({ page }) => {
    await page.goto("/");

    await expect(page.getByRole("heading", { name: "Hello World" })).toBeVisible();
    const menu = await openMoreMenu(page);
    await menu.getByRole("button", { name: "Need Help?" }).click();

    await expect(page.getByRole("heading", { name: "Need help?" })).toBeVisible();
  });

  test("User Interface", async ({ page }) => {
    await page.goto("/");
    await expect(page.getByRole("heading", { name: "Hello World" })).toBeVisible();
    const menu = await openMoreMenu(page);
    await menu.getByRole("button", { name: "User Interface" }).click();

    await expect(page.getByRole("heading", { name: "User Interface" })).toBeVisible();
  });
});

test.describe("Advanced", async () => {
  test.beforeAll(async () => {
    await startSimulator();
    await start(simulatorConfig());
  });

  test.afterAll(async () => {
    await stop();
    await stopSimulator();
  });

  const title = "evcc Simulator";

  test("Menu options. All available.", async ({ page }) => {
    for (const route of UI_ROUTES) {
      await page.goto(route.path);

      await expect(page.getByRole("heading", { name: route.title || title })).toBeVisible();
      // battery tab visible when battery is configured
      await expect(page.getByRole("link", { name: "Battery" })).toBeVisible();
      const menu = await openMoreMenu(page);
      await expect(menu.getByRole("button", { name: "User Interface" })).toBeVisible();
      await expect(menu.getByRole("button", { name: "Need Help?" })).toBeVisible();
    }
  });

  test("Home Battery from bottom nav", async ({ page }) => {
    await page.goto("/");
    await expect(page.getByRole("heading", { name: title })).toBeVisible();
    await page.getByRole("link", { name: "Battery" }).click();

    await expect(page.getByRole("heading", { name: "Home Battery" })).toBeVisible();
  });

  test("Home Battery from energyflow", async ({ page }) => {
    await page.goto("/");
    await page.getByTestId("energyflow").click();
    await page
      .getByTestId("energyflow-entry-batterydischarge")
      .getByTestId("energyflow-entry-details")
      .click();

    await expect(page.getByRole("heading", { name: "Home Battery" })).toBeVisible();
  });
});
