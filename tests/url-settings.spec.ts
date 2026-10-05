import { test, expect } from "@playwright/test";
import { start, stop, baseUrl } from "./evcc";

test.use({ baseURL: baseUrl() });

test.beforeAll(async () => {
  await start();
});
test.afterAll(async () => {
  await stop();
});

test.describe("ui settings via url", async () => {
  test("apply, persist and strip valid params", async ({ page }) => {
    await page.goto("/?theme=dark&lang=de&unit=mi&format=12");

    await expect(page.locator("html")).toHaveAttribute("data-bs-theme", "dark");
    await expect(page.locator("html")).toHaveAttribute("lang", "de");

    const stored = await page.evaluate(() => ({
      theme: localStorage["settings_theme"],
      locale: localStorage["settings_locale"],
      unit: localStorage["settings_unit"],
      format: localStorage["settings_12h_format"],
    }));
    expect(stored).toEqual({ theme: "dark", locale: "de", unit: "mi", format: "true" });

    // params removed from address bar
    expect(new URL(page.url()).search).toBe("");
  });

  test("ignore invalid value and keep it in url", async ({ page }) => {
    await page.goto("/?theme=bogus");
    await expect(page.locator("html")).not.toHaveAttribute("data-bs-theme", "bogus");
    expect(new URL(page.url()).searchParams.get("theme")).toBe("bogus");
  });
});
