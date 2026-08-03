import { test, expect } from "@playwright/test";
import { start, stop, baseUrl } from "./evcc";
import { expectModalVisible } from "./utils";

// German browser language (comma as decimal separator), English UI
// macOS Chromium reads --lang, Linux reads LC_ALL/LANG
test.use({
  baseURL: baseUrl(),
  launchOptions: {
    args: ["--lang=de-DE"],
    env: { ...process.env, LC_ALL: "de_DE.UTF-8", LANG: "de_DE.UTF-8" },
  },
  locale: "en-US",
});

test.beforeAll(async () => {
  await start();
});

test.afterAll(async () => {
  await stop();
});

test("decimal input with german separator", async ({ page }) => {
  await page.goto("/#/config");
  await page.getByRole("button", { name: "Add solar or battery" }).click();
  const modal = page.getByTestId("meter-modal");
  await expectModalVisible(modal);
  await modal.getByRole("button", { name: "Add solar meter" }).click();
  await modal.getByLabel("Manufacturer").selectOption({ label: "SunSpec Inverter" });
  await modal.getByRole("button", { name: "Show advanced settings" }).click();

  // comma accepted as decimal separator
  const timeout = modal.getByLabel("Timeout");
  await timeout.pressSequentially("0,5");
  await expect(timeout).toHaveValue("0.5");

  // intermediate zeros survive typing
  const delay = modal.getByLabel("Delay");
  await delay.pressSequentially("0.004");
  await expect(delay).toHaveValue("0.004");
  await delay.blur();
  await expect(delay).toHaveValue("0.004");
});
