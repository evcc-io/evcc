import { test, expect } from "@playwright/test";
import { start, stop, baseUrl } from "./evcc";

test.use({ baseURL: baseUrl() });

test.beforeAll(async () => {
  await start("basics.evcc.yaml");
});
test.afterAll(async () => {
  await stop();
});

test.beforeEach(async ({ page }) => {
  await page.goto("/");
});

test.describe("main screen", async () => {
  test("site title", async ({ page }) => {
    await expect(page.getByRole("heading", { name: "Hello World" })).toBeVisible();
  });

  test("visualization", async ({ page }) => {
    const locator = page.getByTestId("visualization");
    await expect(locator).toBeVisible();
    await expect(locator).toContainText("1.0 kW");
  });

  test("one loadpoint", async ({ page }) => {
    await expect(page.getByTestId("loadpoint")).toHaveCount(1);
  });

  test("loadpoint title", async ({ page }) => {
    await expect(page.getByRole("heading", { name: "Carport" })).toBeVisible();
  });

  test("guest vehicle", async ({ page }) => {
    await expect(page.getByRole("heading", { name: "Guest vehicle" })).toBeVisible();
  });
});

test.describe("session info", async () => {
  test("default", async ({ page }) => {
    await expect(page.getByTestId("sessionInfoLabel").first()).toContainText("Duration");
  });
  test("change value", async ({ page }) => {
    // by select
    await page.getByTestId("sessionInfoSelect").first().selectOption({ label: "Solar" });
    await expect(page.getByTestId("sessionInfoLabel").first()).toContainText("Solar");
    // by click on value
    await page.getByTestId("sessionInfoValue").first().click();
    await expect(page.getByTestId("sessionInfoLabel").first()).toContainText("Duration");
  });
  test("keep selection on reload", async ({ page }) => {
    await expect(page.getByTestId("sessionInfoLabel").first()).toContainText("Duration");
    await page.getByTestId("sessionInfoSelect").first().selectOption({ label: "Solar" });
    await expect(page.getByTestId("sessionInfoLabel").first()).toContainText("Solar");
    await page.reload();
    await expect(page.getByTestId("sessionInfoLabel").first()).toContainText("Solar");
  });
});

test.describe("loadpoint settings", async () => {
  test("phase selection", async ({ page }) => {
    await page.getByTestId("loadpoint-settings-button").nth(1).click();
    await expect(page.getByLabel("auto-switching")).not.toBeVisible();
    await expect(page.getByLabel("3 phase")).toBeChecked();
    await expect(page.getByLabel("1 phase")).not.toBeChecked();
    await expect(page.getByText("~ 11.0 kW")).toBeVisible();
    await expect(page.getByText("~ 4.1 kW")).toBeVisible();

    await page.getByLabel("1 phase").click();
    await expect(page.getByText("~ 3.7 kW")).toBeVisible();
    await expect(page.getByText("~ 1.4 kW")).toBeVisible();
  });
});
