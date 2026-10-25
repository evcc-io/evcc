import { test, expect } from "@playwright/test";
import { start, stop, baseUrl } from "./evcc";
import { enableExperimental } from "./utils";

test.use({ baseURL: baseUrl() });

test.beforeAll(async () => {
  await start("forecast-planner.evcc.yaml");
});
test.afterAll(async () => {
  await stop();
});

const gridSlots = async (request: any): Promise<number> => {
  const res = await request.get("/api/state?jq=.forecast.grid|length");
  return Number(await res.text());
};

test.describe("planner extends grid", () => {
  test("switch", async ({ page, request }) => {
    await page.goto("/#/config");
    await enableExperimental(page);

    await page.goto("/#/forecast");
    const toggle = page.getByLabel("Extend with planner forecast");
    await expect(toggle).toBeVisible();
    await expect(toggle).not.toBeChecked();
    expect(await gridSlots(request)).toBeLessThanOrEqual(4 * 12);

    await toggle.click();
    await expect(toggle).toBeChecked();
    await expect.poll(() => gridSlots(request)).toBeGreaterThan(4 * 12);

    await toggle.click();
    await expect(toggle).not.toBeChecked();
    await expect.poll(() => gridSlots(request)).toBeLessThanOrEqual(4 * 12);
  });
});
