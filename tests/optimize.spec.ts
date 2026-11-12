import { test, expect } from "@playwright/test";
import { start, stop, baseUrl } from "./evcc";
import { expectModalVisible, openMoreMenu } from "./utils";

const CONFIG = "basics.evcc.yaml";
const SQL = "optimize.sql";

test.use({ baseURL: baseUrl() });
test.describe.configure({ mode: "parallel" });

test.beforeEach(async () => {
  await start(CONFIG, SQL);
});
test.afterEach(async () => {
  await stop();
});

test.describe("optimize page", async () => {
  test("menu entry visible without optimizer result", async ({ page }) => {
    await page.goto("/");
    await openMoreMenu(page);
    await page.getByRole("link", { name: "Optimize" }).click();
    await expect(page.getByRole("heading", { name: "Optimize Debug" })).toBeVisible();
  });

  test("empty state while collecting data", async ({ page }) => {
    await page.goto("/#/optimize");
    await expect(page.getByTestId("optimize-empty")).toContainText("collecting data");
    await expect(page.getByText("Net grid cost")).not.toBeVisible();
    await page.getByRole("link", { name: "Check logs" }).click();
    await expect(page.getByTestId("log-search")).toHaveValue("optimizer");
  });

  test("disabled state", async ({ page }) => {
    await page.goto("/#/optimize");
    await expect(page.getByTestId("optimize-empty")).toBeVisible();

    await expect(await page.request.post("/api/config/optimizer/false")).toBeOK();
    await expect(page.getByText("The optimizer is disabled.")).toBeVisible();

    // settings open in place, the page stays reachable without its menu entry
    await page.getByRole("button", { name: "Open settings" }).click();
    await expectModalVisible(page.getByTestId("optimizer-modal"));
    await expect(page).toHaveURL("/#/optimize");
  });
});
