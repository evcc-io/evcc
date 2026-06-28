import { test, expect, type Locator, type Page } from "@playwright/test";
import { start, stop, baseUrl } from "./evcc";

test.use({ baseURL: baseUrl() });

// from assets/js/colors.ts palette
const BLUE = "rgb(96, 165, 250)"; // #60A5FA — palette[0]
const AMBER = "rgb(251, 191, 36)"; // #FBBF24 — palette[1]
const CUSTOM_HEX = "FF0080";
const CUSTOM_RGB = "rgb(255, 0, 128)";
const CUSTOM_HEX2 = "00C8FF";
const CUSTOM_RGB2 = "rgb(0, 200, 255)";

test.beforeAll(async () => {
  await start("device-colors.evcc.yaml", "device-colors.sql");
});
test.afterAll(async () => {
  await stop();
});

function legendBadge(scope: Locator | Page, label: string): Locator {
  // first match — multiple charts can show the same legend
  return scope.getByLabel(label, { exact: true }).first();
}

async function expectLegendColor(scope: Locator | Page, label: string, rgb: string): Promise<void> {
  const dot = legendBadge(scope, label).locator("span").first();
  await expect(dot).toHaveCSS("background-color", rgb);
}

function chartSection(page: Page, heading: string): Locator {
  return page.locator("section").filter({ has: page.getByRole("heading", { name: heading }) });
}

test("device colors: autoassign, override, persistence", async ({ page }) => {
  // ---------- Step 1 — Sessions, by-vehicle view ----------
  await page.goto("/#/sessions?year=2026&month=5");
  await expect(page.getByRole("heading", { name: "Charging Sessions" })).toBeVisible();
  await page.getByRole("button", { name: "Vehicle", exact: true }).click();
  await expectLegendColor(page, "Honda", BLUE);

  // ---------- Step 2 — Sessions, by-loadpoint + override ----------
  await page.getByRole("button", { name: "Charging point", exact: true }).click();
  // initial autoassign — Garage (more energy) → palette[0], Carport → palette[1]
  await expectLegendColor(page, "Garage", BLUE);
  await expectLegendColor(page, "Carport", AMBER);

  // open picker on Garage badge; second palette swatch = AMBER
  await legendBadge(page, "Garage").click();
  const popover = page.getByRole("dialog");
  await expect(popover).toBeVisible();
  // palette swatches have title=hex
  await popover.getByTitle("#FBBF24").click();
  await page.keyboard.press("Escape");

  await expectLegendColor(page, "Garage", AMBER);
  // non-collision: Carport must shift off AMBER to the first free entry (BLUE)
  await expectLegendColor(page, "Carport", BLUE);

  // ---------- Step 3 — History view, same colors ----------
  await page.goto("/#/history?period=day&year=2026&month=5&day=15");
  const lpSection = page
    .locator("section")
    .filter({ has: page.getByRole("heading", { name: "Charging & Heating" }) });
  await expect(lpSection).toBeVisible({ timeout: 10000 });
  await expectLegendColor(lpSection, "Garage", AMBER);
  await expectLegendColor(lpSection, "Carport", BLUE);

  // ---------- Step 4 — Custom hex on consumer meter (Consumption chart) ----------
  const consumerSection = chartSection(page, "Consumption");
  await expect(consumerSection).toBeVisible();
  await legendBadge(consumerSection, "Dishwasher").click();
  const popover2 = page.getByRole("dialog");
  await expect(popover2).toBeVisible();
  // save-on-type fires when input matches hex regex
  await popover2.getByLabel("Hex color").fill(CUSTOM_HEX);
  await page.keyboard.press("Escape");

  await expectLegendColor(consumerSection, "Dishwasher", CUSTOM_RGB);

  // ---------- Step 5 — Custom hex on ext meter (Additional meters chart) ----------
  const meterSection = chartSection(page, "Additional meters");
  await expect(meterSection).toBeVisible();
  await legendBadge(meterSection, "Alternative grid").click();
  const popover3 = page.getByRole("dialog");
  await expect(popover3).toBeVisible();
  await popover3.getByLabel("Hex color").fill(CUSTOM_HEX2);
  await page.keyboard.press("Escape");

  await expectLegendColor(meterSection, "Alternative grid", CUSTOM_RGB2);

  // ---------- Step 6 — Reload: both colors persist via settings DB ----------
  await page.reload();
  await expectLegendColor(chartSection(page, "Consumption"), "Dishwasher", CUSTOM_RGB);
  await expectLegendColor(chartSection(page, "Additional meters"), "Alternative grid", CUSTOM_RGB2);
});
