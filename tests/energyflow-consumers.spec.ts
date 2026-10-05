import { test, expect } from "@playwright/test";
import { start, stop, baseUrl } from "./evcc";

const CONFIG = "energyflow-consumers.evcc.yaml";

test.use({ baseURL: baseUrl() });

test.beforeAll(async () => {
  await start(CONFIG);
});
test.afterAll(async () => {
  await stop();
});

test("consumers breakdown lists consumers and aux, excludes ext", async ({ page }) => {
  await page.goto("/");
  await page.getByTestId("energyflow").click();

  // expand the home consumers breakdown
  await page.getByTestId("energyflow-entry-home").click();

  const consumers = page.getByTestId("energyflow-entry-consumer");
  await expect(consumers).toHaveCount(2);
  await expect(consumers.filter({ hasText: "fridge" })).toBeVisible(); // consumer
  await expect(consumers.filter({ hasText: "heater" })).toBeVisible(); // aux
  await expect(consumers.filter({ hasText: "submeter" })).toBeHidden(); // ext excluded
});
