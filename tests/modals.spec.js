import { test, expect } from "@playwright/test";
import { start, stop } from "./evcc";
import { startSimulator, stopSimulator } from "./simulator";

const BASICS_CONFIG = "basics.evcc.yaml";
const SIMULATOR_CONFIG = "simulator.evcc.yaml";

const UI_ROUTES = ["/", "/#/sessions", "/#/config"];

async function login() {
  // TODO: uncomment this once auth is released
  // await page.locator("#loginPassword").fill("secret");
  // await page.getByRole("button", { name: "Login" }).click();
}

test.describe("Basics", async () => {
  test.beforeAll(async () => {
    await start(BASICS_CONFIG, "password.sql");
  });

  test.afterAll(async () => {
    await stop();
  });

  test("Menu options. No battery and grid.", async ({ page }) => {
    for (const route of UI_ROUTES) {
      await page.goto(route);
      if (route === "/#/config") {
        await login(page);
      }

      await page.getByTestId("topnavigation-button").click();
      await expect(page.getByRole("button", { name: "General Settings" })).toBeVisible();
      await expect(page.getByRole("button", { name: "Battery Settings" })).not.toBeVisible();
      await expect(page.getByRole("button", { name: "Need help?" })).toBeVisible();
    }
  });

  test("Need help?", async ({ page }) => {
    await page.goto("/");

    await page.getByTestId("topnavigation-button").click();
    await page.getByRole("button", { name: "Need help?" }).click();

    await expect(page.getByRole("heading", { name: "Need help?" })).toBeVisible();
  });

  test("General Settings", async ({ page }) => {
    await page.goto("/");
    await page.getByTestId("topnavigation-button").click();
    await page.getByRole("button", { name: "General Settings" }).click();

    await expect(page.getByRole("heading", { name: "General Settings" })).toBeVisible();
  });
});

test.describe("Advanced", async () => {
  test.beforeAll(async () => {
    await start(SIMULATOR_CONFIG, "password.sql");
    await startSimulator();
  });

  test.afterAll(async () => {
    await stopSimulator();
    await stop();
  });

  test("Menu options. All available.", async ({ page }) => {
    for (const route of UI_ROUTES) {
      await page.goto(route);
      if (route === "/#/config") {
        await login(page);
      }

      await page.getByTestId("topnavigation-button").click();
      await expect(page.getByRole("button", { name: "General Settings" })).toBeVisible();
      await expect(page.getByRole("button", { name: "Battery Settings" })).toBeVisible();
      await expect(page.getByRole("button", { name: "Need help?" })).toBeVisible();
    }
  });

  test("Battery Settings from top navigation", async ({ page }) => {
    await page.goto("/");
    await page.getByTestId("topnavigation-button").click();
    await page.getByRole("button", { name: "Battery Settings" }).click();

    await expect(page.getByRole("heading", { name: "Battery Settings" })).toBeVisible();
  });

  test("Battery Settings from energyflow", async ({ page }) => {
    await page.goto("/");
    await page.getByTestId("energyflow").click();
    await page
      .getByTestId("energyflow-entry-batterydischarge")
      .getByTestId("energyflow-entry-details")
      .click();

    await expect(page.getByRole("heading", { name: "Battery Settings" })).toBeVisible();
  });
});
