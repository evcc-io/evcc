import { test, expect, type Page, type Locator } from "@playwright/test";
import { start, stop, restart, baseUrl } from "./evcc";
import { expectModalVisible, expectModalHidden } from "./utils";

const templateFlags = [
  "--disable-auth",
  "--template-type",
  "meter",
  "--template",
  "tests/config-duration-demo.tpl.yaml",
];

test.use({ baseURL: baseUrl() });

test.afterEach(async () => {
  await stop();
});

async function openGridMeterModal(page: Page): Promise<Locator> {
  await page.goto("/#/config");
  await page.getByTestId("grid").getByRole("button", { name: "edit" }).click();
  const modal = page.getByTestId("meter-modal");
  await expectModalVisible(modal);
  return modal;
}

test.describe("duration fields", async () => {
  test("template defaults and roundtrip", async ({ page }) => {
    await start(undefined, undefined, templateFlags);
    await page.goto("/#/config");
    await page.getByRole("button", { name: "Add grid meter" }).click();
    const modal = page.getByTestId("meter-modal");
    await expectModalVisible(modal);
    await modal.getByLabel("Manufacturer").selectOption("Duration Demo Meter");

    await expect(modal.getByLabel("Short duration")).toHaveValue("30");
    await expect(modal.getByText("Example: 10", { exact: true })).toBeVisible();
    await expect(modal.getByText("seconds")).toBeVisible();
    await expect(modal.getByLabel("Long duration")).toHaveValue("3");
    await expect(modal.getByText("hours")).toBeVisible();

    await modal.getByLabel("Long duration").fill("6");
    await modal.getByRole("button", { name: "Save" }).click();
    await expectModalHidden(modal);

    await restart(undefined, templateFlags);
    await page.reload();

    // edited value (stored as nanoseconds) and untouched default (stored as string)
    const reopened = await openGridMeterModal(page);
    await expect(reopened.getByLabel("Long duration")).toHaveValue("6");
    await expect(reopened.getByLabel("Short duration")).toHaveValue("30");
  });

  test("existing values as string and nanoseconds", async ({ page }) => {
    await start(undefined, "config-duration-values.sql", templateFlags);

    const modal = await openGridMeterModal(page);
    await expect(modal.getByLabel("Long duration")).toHaveValue("12");
    await expect(modal.getByLabel("Short duration")).toHaveValue("15");

    await modal.getByRole("button", { name: "Save" }).click();
    await expectModalHidden(modal);

    await restart(undefined, templateFlags);
    await page.reload();

    const reopened = await openGridMeterModal(page);
    await expect(reopened.getByLabel("Long duration")).toHaveValue("12");
    await expect(reopened.getByLabel("Short duration")).toHaveValue("15");
  });
});
