import { test, expect } from "@playwright/test";
import { start, stop, baseUrl } from "./evcc";

test.use({ baseURL: baseUrl() });

test.beforeAll(async () => {
  await start("heating.evcc.yaml");
});
test.afterAll(async () => {
  await stop();
});

test.beforeEach(async ({ page }) => {
  await page.goto("/");
});

test.describe("loadpoint", async () => {
  test("initial values", async ({ page }) => {
    await expect(page.getByTestId("current-soc")).toContainText("55.0°C");
    await expect(page.getByTestId("limit-soc")).toContainText("100.0°C");
  });

  test("change limit in 1° steps", async ({ page }) => {
    await page.getByTestId("limit-soc").getByRole("combobox").selectOption("69.0°C");
    await expect(page.getByTestId("limit-soc")).toContainText("69.0°C");
    await page.reload();
    await expect(page.getByTestId("limit-soc")).toContainText("69.0°C");
  });
});
