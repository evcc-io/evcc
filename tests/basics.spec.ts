import { test, expect } from "@playwright/test";
import { start, stop, baseUrl } from "./evcc";
import { enableExperimental } from "./utils";

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
    await page
      .getByTestId("sessionInfoSelect")
      .first()
      .getByRole("combobox")
      .selectOption({ label: "Solar" });
    await expect(page.getByTestId("sessionInfoLabel").first()).toContainText("Solar");
    // by click on value, wraps around
    await page.getByTestId("sessionInfoValue").first().click();
    await expect(page.getByTestId("sessionInfoLabel").first()).toContainText("Duration");
  });
  test("keep selection on reload", async ({ page }) => {
    await expect(page.getByTestId("sessionInfoLabel").first()).toContainText("Duration");
    await page
      .getByTestId("sessionInfoSelect")
      .first()
      .getByRole("combobox")
      .selectOption({ label: "Solar" });
    await expect(page.getByTestId("sessionInfoLabel").first()).toContainText("Solar");
    await page.reload();
    await expect(page.getByTestId("sessionInfoLabel").first()).toContainText("Solar");
  });
});

test.describe("loadpoint settings", async () => {
  test("phase selection", async ({ page }) => {
    const minCurrentSelected = page.getByLabel("Min. current").locator("option:checked");
    const maxCurrentSelected = page.getByLabel("Max. current").locator("option:checked");

    await page.getByTestId("loadpoint-settings-button").nth(1).click();
    await expect(page.getByLabel("auto-switching")).not.toBeVisible();
    await expect(page.getByLabel("3 phase")).toBeChecked();
    await expect(page.getByLabel("1 phase")).not.toBeChecked();
    await expect(maxCurrentSelected).toHaveText("16 A (11.0 kW) [default]");
    await expect(minCurrentSelected).toHaveText("6 A (4.1 kW) [default]");

    await page.getByLabel("1 phase").click();
    await expect(maxCurrentSelected).toHaveText("16 A (3.7 kW) [default]");
    await expect(minCurrentSelected).toHaveText("6 A (1.4 kW) [default]");
  });
});

test.describe("network requests", async () => {
  test("no failed requests", async ({ page }) => {
    await page.waitForLoadState("networkidle");

    const failedRequests: string[] = [];
    page.on("requestfailed", (request) => failedRequests.push(request.url()));

    await page.reload();
    await page.waitForLoadState("networkidle");

    expect(failedRequests).toHaveLength(0);
  });
});

test.describe("session info experimental", async () => {
  test("period groups", async ({ page }) => {
    await expect(page.getByTestId("sessionInfoLabel").first()).toContainText("Duration");
    await page.goto("/#/config");
    await enableExperimental(page);
    await page.goto("/");

    // session, 24h and 7d groups; the collapsed select hides its groups
    const select = page.getByTestId("sessionInfoSelect").first().getByRole("combobox");
    await expect(select.getByRole("group", { includeHidden: true })).toHaveCount(3);
    await expect(select.getByRole("option", { name: "Solar" })).toHaveCount(1);
    await expect(select.getByRole("option", { name: "Charged" })).toHaveCount(2);

    await select.selectOption("last7dEnergy");
    await expect(page.getByTestId("sessionInfoLabel").first()).toContainText("Charged (7d)");
    await page.reload();
    await expect(page.getByTestId("sessionInfoLabel").first()).toContainText("Charged (7d)");
  });
});
