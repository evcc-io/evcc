import { test, expect } from "@playwright/test";
import { start, stop, baseUrl } from "./evcc";

test.use({ baseURL: baseUrl() });

test.beforeAll(async () => {
  await start("basics.evcc.yaml", "session-info-period.sql");
});
test.afterAll(async () => {
  await stop();
});

test("solar share", async ({ page }) => {
  await page.goto("/");
  await page.getByTestId("sessionInfoSelect").getByRole("combobox").selectOption("last24hSolar");
  await expect(page.getByTestId("sessionInfoLabel")).toContainText("Solar (24h)");
  // pv and battery of the total
  await expect(page.getByTestId("sessionInfoValue")).toContainText("50.0%");
});
