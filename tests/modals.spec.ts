import { test, expect } from "@playwright/test";
import { start, stop, baseUrl } from "./evcc";
import { startSimulator, stopSimulator, simulatorConfig } from "./simulator";

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
      await page.getByTestId("topnavigation-button").click();
      await expect(page.getByRole("button", { name: "User Interface" })).toBeVisible();
      await expect(page.getByRole("button", { name: "Home Battery" })).not.toBeVisible();
      await expect(page.getByRole("button", { name: "Need help?" })).toBeVisible();
    }
  });

  test("Need help?", async ({ page }) => {
    await page.goto("/");

    await expect(page.getByRole("heading", { name: "Hello World" })).toBeVisible();
    await page.getByTestId("topnavigation-button").click();
    await page.getByRole("button", { name: "Need help?" }).click();

    await expect(page.getByRole("heading", { name: "Need help?" })).toBeVisible();
  });

  test("User Interface", async ({ page }) => {
    await page.goto("/");
    await expect(page.getByRole("heading", { name: "Hello World" })).toBeVisible();
    await page.getByTestId("topnavigation-button").click();
    await page.getByRole("button", { name: "User Interface" }).click();

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
      await page.getByTestId("topnavigation-button").click();
      await expect(page.getByRole("button", { name: "User Interface" })).toBeVisible();
      await expect(page.getByRole("button", { name: "Home Battery" })).toBeVisible();
      await expect(page.getByRole("button", { name: "Need help?" })).toBeVisible();
    }
  });

  test("Home Battery from top navigation", async ({ page }) => {
    await page.goto("/");
    await expect(page.getByRole("heading", { name: title })).toBeVisible();
    await page.getByTestId("topnavigation-button").click();
    await page.getByRole("button", { name: "Home Battery" }).click();

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
