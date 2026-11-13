import { test, expect, type Locator } from "@playwright/test";
import { start, stop, baseUrl } from "./evcc";

test.use({ baseURL: baseUrl() });

test.beforeAll(async () => {
  await start("energy-loading.evcc.yaml", "energy.sql");
});
test.afterAll(async () => {
  await stop();
});

const height = async (card: Locator) => Math.round((await card.boundingBox())!.height);

const viewports = {
  phone: { width: 390, height: 800 },
  desktop: { width: 1280, height: 900 },
};

for (const [name, viewport] of Object.entries(viewports)) {
  test(`first load keeps its frame on ${name}`, async ({ page }) => {
    await page.setViewportSize(viewport);

    let release = () => {};
    const held = new Promise<void>((resolve) => (release = resolve));
    await page.route("**/api/history/**", async (route) => {
      await held;
      await route.continue();
    });

    await page.goto("/#/energy?year=2026&month=9&day=15");
    const progress = page.getByRole("progressbar", { name: "Loading" });
    const summary = page.getByRole("region", { name: "Summary" });
    const overview = page.getByRole("region", { name: "Overview" });
    const autarky = summary.getByTestId("energy-stat-autarky");

    // pending: progress line, labels in place, no values
    await expect(progress).toBeVisible();
    await expect(summary.getByText("Self-consumption")).toBeVisible();
    await expect(autarky).toContainText("Autarky");
    await expect(autarky).not.toContainText("%");
    await expect(overview.getByRole("heading", { name: "Overview" })).toBeVisible();
    const pending = [await height(summary), await height(overview)];

    release();

    // loaded: values in the same frame
    await expect(autarky).toContainText("60%");
    await expect(progress).toBeHidden();
    await expect(overview.getByText("Grid export")).toBeVisible();
    expect([await height(summary), await height(overview)]).toEqual(pending);
  });
}
