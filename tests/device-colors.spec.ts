import { test, expect, type Locator, type Page } from "@playwright/test";
import { start, stop, baseUrl } from "./evcc";

test.use({ baseURL: baseUrl() });

// from assets/js/colors.ts palette
const SKY = "rgb(14, 165, 233)"; // #0EA5E9 — palette[0]
const PINK = "rgb(236, 72, 153)"; // #EC4899 — palette[1]
const CUSTOM_HEX = "FF0080";
const CUSTOM_RGB = "rgb(255, 0, 128)";

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

test("device colors: autoassign, override, persistence", async ({ page }) => {
  // ---------- Step 1 — Sessions, by-vehicle view ----------
  await page.goto("/#/sessions?year=2026&month=5");
  await expect(page.getByRole("heading", { name: "Charging Sessions" })).toBeVisible();
  await page.getByRole("button", { name: "Vehicle", exact: true }).click();
  await expectLegendColor(page, "Honda", SKY);

  // ---------- Step 2 — Sessions, by-loadpoint + override ----------
  await page.getByRole("button", { name: "Charging point", exact: true }).click();
  // initial autoassign — Garage (more energy) → palette[0], Carport → palette[1]
  await expectLegendColor(page, "Garage", SKY);
  await expectLegendColor(page, "Carport", PINK);

  // open picker on Garage badge; second palette swatch = PINK
  await legendBadge(page, "Garage").click();
  const popover = page.getByRole("dialog");
  await expect(popover).toBeVisible();
  // palette swatches have title=hex
  await popover.getByTitle("#EC4899").click();
  await page.keyboard.press("Escape");

  await expectLegendColor(page, "Garage", PINK);
  // non-collision: Carport must shift off PINK to the first free entry (SKY)
  await expectLegendColor(page, "Carport", SKY);

  // ---------- Step 3 — History view, same colors ----------
  await page.goto("/#/history?period=day&year=2026&month=5&day=15");
  const lpSection = page
    .locator("section")
    .filter({ has: page.getByRole("heading", { name: "Charging & Heating" }) });
  await expect(lpSection).toBeVisible({ timeout: 10000 });
  await expectLegendColor(lpSection, "Garage", PINK);
  await expectLegendColor(lpSection, "Carport", SKY);

  // ---------- Step 4 — Custom hex on ext meter, reload ----------
  const meterSection = page
    .locator("section")
    .filter({ has: page.getByRole("heading", { name: "Consumption" }) });
  await expect(meterSection).toBeVisible();
  await legendBadge(meterSection, "Dishwasher").click();
  const popover2 = page.getByRole("dialog");
  await expect(popover2).toBeVisible();
  // save-on-type fires when input matches hex regex
  await popover2.getByLabel("Hex color").fill(CUSTOM_HEX);
  await page.keyboard.press("Escape");

  await expectLegendColor(meterSection, "Dishwasher", CUSTOM_RGB);

  // reload: color persists via settings DB
  await page.reload();
  const meterSection2 = page
    .locator("section")
    .filter({ has: page.getByRole("heading", { name: "Consumption" }) });
  await expectLegendColor(meterSection2, "Dishwasher", CUSTOM_RGB);
});
