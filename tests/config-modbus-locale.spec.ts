import { test, expect, type Page, type Locator } from "@playwright/test";
import { start, stop, baseUrl } from "./evcc";
import { expectModalVisible } from "./utils";

const CONFIG_MODBUS_FIELDS = "config-modbus-fields.sql";

// German browser language (comma as decimal separator), English UI
test.use({
  baseURL: baseUrl(),
  launchOptions: { args: ["--lang=de-DE"] },
  locale: "en-US",
});

test.beforeAll(async () => {
  await start(undefined, CONFIG_MODBUS_FIELDS);
});

test.afterAll(async () => {
  await stop();
});

async function openAdvancedSettings(page: Page): Promise<Locator> {
  await page.goto("/#/config");
  await page
    .getByTestId("pv")
    .filter({ hasText: "TCP Test" })
    .getByRole("button", { name: "edit" })
    .click();
  const modal = page.getByTestId("meter-modal");
  await expectModalVisible(modal);
  await modal.getByRole("button", { name: "Show advanced settings" }).click();
  return modal;
}

test.describe("duration fields with german decimal separator", async () => {
  test("accept comma input", async ({ page }) => {
    const modal = await openAdvancedSettings(page);
    const timeout = modal.getByLabel("Timeout");
    await timeout.pressSequentially("0,5");
    await expect(timeout).toHaveValue("0.5");
  });

  test("keep intermediate zeros while typing", async ({ page }) => {
    const modal = await openAdvancedSettings(page);
    const delay = modal.getByLabel("Delay");
    await delay.pressSequentially("0.004");
    await expect(delay).toHaveValue("0.004");
    await delay.blur();
    await expect(delay).toHaveValue("0.004");
  });
});
