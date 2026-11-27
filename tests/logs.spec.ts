import { test, expect } from "@playwright/test";
import { start, stop, baseUrl } from "./evcc";
import { openMoreMenu, enableAppContext, expectAppEvent } from "./utils";

test.use({ baseURL: baseUrl() });

test.beforeAll(async () => {
  await start("basics.evcc.yaml");
});
test.afterAll(async () => {
  await stop();
});

test.describe("opening logs", async () => {
  test("via config", async ({ page }) => {
    await page.goto("/");
    const menu = await openMoreMenu(page);
    await menu.getByRole("link", { name: "Configuration" }).click();
    await expect(page.getByRole("heading", { name: "Configuration" })).toBeVisible();
    await page.getByTestId("config-system").getByRole("link", { name: "Logs" }).click();
    await expect(page.getByRole("heading", { name: "Logs", exact: false })).toBeVisible();
  });
  test("via more menu", async ({ page }) => {
    await page.goto("/");
    const menu = await openMoreMenu(page);
    await menu.getByRole("link", { name: "Logs" }).click();
    await expect(page.getByRole("heading", { name: "Logs", exact: false })).toBeVisible();
  });
  test("via notifications", async ({ page }) => {
    await page.goto("/");
    await page.evaluate(() => window.app.raise({ message: "Fake Error" }));
    await page.getByTestId("notification-icon").click();
    await page.getByRole("link", { name: "View full logs" }).click();
    await expect(page.getByRole("heading", { name: "Logs", exact: false })).toBeVisible();
  });
  test("via notification area", async ({ page }) => {
    await page.goto("/");
    await page.evaluate(() => window.app.raise({ message: "Fake Error", area: "site" }));
    await page.getByTestId("notification-icon").click();
    await page.getByRole("link", { name: "site" }).click();
    await expect(page).toHaveURL(/areas=site/);
    await page.getByRole("button", { name: "site" }).click();
    await expect(page.getByRole("checkbox", { name: "site" })).toBeChecked();
    await expect(page.getByRole("heading", { name: "Logs", exact: false })).toBeVisible();
  });
  test("via need help", async ({ page }) => {
    await page.goto("/");
    const menu = await openMoreMenu(page);
    await menu.getByRole("button", { name: "Need Help?" }).click();
    await page.getByRole("link", { name: "View logs" }).click();
    await expect(page.getByRole("heading", { name: "Logs", exact: false })).toBeVisible();
  });
});

test.describe("notifications", async () => {
  test("dedupe by number", async ({ page }) => {
    await page.goto("/");
    await page.evaluate(() => {
      window.app.raise({ message: "timeout after 10.5s" });
      window.app.raise({ message: "other error" });
      window.app.raise({ message: "timeout after 12s" });
    });
    await page.getByTestId("notification-icon").click();
    const modal = page.getByRole("dialog");
    await expect(modal.getByText("timeout after 12s")).toBeVisible();
    await expect(modal.getByText("timeout after 10.5s")).toHaveCount(0);
    await expect(modal.getByText("other error")).toBeVisible();
    await expect(modal.getByText("2", { exact: true })).toBeVisible();
  });
});

test.describe("features", async () => {
  test("content", async ({ page }) => {
    // the page shows the latest 1000 lines only; at debug level the 0.1s
    // interval pushes the startup line out within seconds, info keeps it
    await page.goto("/#/log?level=info");
    await page.getByTestId("log-search").fill("UI local");
    await expect(page.getByTestId("log-content")).toContainText("UI local");
  });
});

test.describe("areas", async () => {
  test("loadpoint title", async ({ page }) => {
    await page.goto("/#/log?level=info");
    await page.getByRole("button", { name: "All areas" }).click();
    await page.getByRole("checkbox", { name: "lp-1 (Carport)" }).check();
    await expect(page).toHaveURL(/areas=lp-1/);
    await expect(page.getByRole("button", { name: "lp-1 (Carport)" })).toBeVisible();
  });
  test("unknown area from url", async ({ page }) => {
    await page.goto("/#/log?areas=foo");
    await page.getByRole("button", { name: "foo" }).click();
    await expect(page.getByRole("checkbox", { name: "foo" })).toBeChecked();
    await expect(page).toHaveURL(/areas=foo/);
  });
});

test.describe("log download in app context", async () => {
  test("dispatches download event", async ({ page }) => {
    await enableAppContext(page);
    await page.goto("/#/log");
    await page.getByRole("link", { name: "Download complete log" }).click();
    expect(await expectAppEvent(page)).toMatchObject({
      type: "download",
      url: expect.stringContaining("/api/system/log?level=debug&format=txt"),
    });
  });
});
