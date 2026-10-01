import { test, expect } from "@playwright/test";
import type { Page } from "@playwright/test";
import { start, stop, baseUrl } from "./evcc";
import { expectModalVisible, expectDatalistOptions } from "./utils";

test.use({ baseURL: baseUrl() });

const templateFlags = [
  "--disable-auth",
  "--template-type",
  "meter",
  "--template",
  "tests/config-param-service-demo.tpl.yaml",
];

test.beforeAll(async () => {
  await start(undefined, undefined, templateFlags);
});

test.afterAll(async () => {
  await stop();
});

async function openMeterModal(page: Page) {
  await page.goto("/#/config");
  await page.getByRole("button", { name: "Add grid meter" }).click();
  const meterModal = page.getByTestId("meter-modal");
  await expectModalVisible(meterModal);
  await meterModal.getByLabel("Manufacturer").selectOption("Service Demo Meter");
  return meterModal;
}

test.describe("config param service", async () => {
  test("autocomplete simple", async ({ page }) => {
    const meterModal = await openMeterModal(page);
    await expect(meterModal.getByLabel("Important value")).toHaveValue("demo-value");

    const otherValue = meterModal.getByLabel("Other value");
    await expectDatalistOptions(otherValue, ["demo-value"]);

    const country = meterModal.getByLabel("Country");
    await expectDatalistOptions(country, ["germany", "france", "spain"]);
  });

  test("autocomplete dependent", async ({ page }) => {
    const meterModal = await openMeterModal(page);
    await expect(meterModal.getByLabel("Important value")).toHaveValue("demo-value");

    const country = meterModal.getByLabel("Country");
    const city = meterModal.getByLabel("City");

    // initially empty
    await expectDatalistOptions(city, []);

    await country.fill("germany");
    await expectDatalistOptions(city, ["berlin", "munich", "hamburg"]);

    await country.fill("");
    await expectDatalistOptions(city, []);

    await country.fill("france");
    await expectDatalistOptions(city, ["paris", "lyon", "marseille"]);

    await country.fill("fantasy");
    await expectDatalistOptions(city, []);
  });

  test("auto-apply single service value", async ({ page }) => {
    const meterModal = await openMeterModal(page);

    // only required single-value field is auto-populated
    await expect(meterModal.getByLabel("Important value")).toHaveValue("demo-value");
    await expect(meterModal.getByLabel("Other value")).toHaveValue("");
    await expect(meterModal.getByLabel("Country")).toHaveValue("");
    await expect(meterModal.getByLabel("City")).toHaveValue("");
  });

  test("clear button", async ({ page }) => {
    const meterModal = await openMeterModal(page);
    const valueField = meterModal.getByLabel("Important value");
    await expect(valueField).toHaveValue("demo-value");

    const clearButton = valueField.locator("..").getByLabel("Clear");
    await expect(clearButton).toBeVisible();

    // click clear button and verify it disappears and field is cleared
    await clearButton.click();
    await expect(clearButton).not.toBeVisible();
    await expect(valueField).toHaveValue("");
  });
});
