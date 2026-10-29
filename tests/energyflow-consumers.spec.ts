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

test("lists consumers and aux alphabetically, excludes ext, others = remainder", async ({ page }) => {
  await page.goto("/");
  await page.getByTestId("energyflow").click();

  // expand the home consumers breakdown
  await page.getByTestId("energyflow-entry-home").click();

  const consumers = page.getByTestId("energyflow-entry-consumer");
  await expect(consumers).toHaveCount(2);
  await expect(consumers.filter({ hasText: "fridge" })).toBeVisible(); // consumer
  await expect(consumers.filter({ hasText: "boiler" })).toBeVisible(); // aux
  await expect(consumers.filter({ hasText: "submeter" })).toBeHidden(); // ext excluded
  await expect(consumers).toContainText(["boiler", "fridge"]); // alphabetical, not config order

  // unmetered remainder: home 900 W minus fridge and boiler, ext not subtracted
  const others = page.getByTestId("energyflow-entry-consumer-other");
  await expect(others).toContainText("Others");
  await expect(others).toContainText("250 W");
});
