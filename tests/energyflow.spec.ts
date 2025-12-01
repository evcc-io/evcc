import { test, expect } from "@playwright/test";
import { start, stop, baseUrl } from "./evcc";

const CONFIG = "config-with-tariffs.evcc.yaml";

test.use({ baseURL: baseUrl() });

test.beforeAll(async () => {
  await start(CONFIG);
});

test.afterAll(async () => {
  await stop();
});

test.beforeEach(async ({ page }) => {
  await page.goto("/");
});

test("tariff toggle", async ({ page }) => {
  const energyflow = page.getByTestId("energyflow");
  await energyflow.click();

  const details = page
    .getByTestId("energyflow-entry-gridimport")
    .getByTestId("energyflow-entry-details");

  // check price
  await expect(details).toHaveText("30.0 öre");

  // toggle to co2
  await details.click();
  await expect(details).toHaveText("300 g");

  // reload page and verify persistence
  await page.reload();
  await expect(details).toHaveText("300 g");

  // toggle back to price
  await details.click();
  await expect(details).toHaveText("30.0 öre");
});
