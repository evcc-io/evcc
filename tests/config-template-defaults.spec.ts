import { test, expect } from "@playwright/test";
import { start, stop, restart, baseUrl } from "./evcc";
import { expectModalHidden, expectModalVisible } from "./utils";

const templateFlags = [
  "--disable-auth",
  "--template-type",
  "meter",
  "--template",
  "tests/config-template-defaults-demo.tpl.yaml",
];

test.use({ baseURL: baseUrl() });

test.beforeAll(async () => {
  await start(undefined, undefined, templateFlags);
});
test.afterAll(async () => {
  await stop();
});

test.describe("template defaults", async () => {
  test("cleared param falls back to default", async ({ page }) => {
    await page.goto("/#/config");

    const grid = page.getByTestId("grid");
    const modal = page.getByTestId("meter-modal");
    const power = modal.getByRole("spinbutton", { name: "Power" });
    const validate = modal.getByRole("link", { name: "validate" });
    const resultPower = modal.getByTestId("test-result").getByTestId("device-tag-power");

    await page.getByRole("button", { name: "Add grid meter" }).click();
    await expectModalVisible(modal);
    await modal.getByLabel("Manufacturer").selectOption("Template Defaults Demo Meter");

    // form shows the default, backend uses it
    await expect(power).toHaveValue("1000");
    await validate.click();
    await expect(resultPower).toContainText("1.0 kW");

    // explicit value
    await power.fill("2500");
    await power.blur();
    await validate.click();
    await expect(resultPower).toContainText("2.5 kW");

    // cleared value falls back to the default
    await power.fill("");
    await power.blur();
    await expect(power).toHaveValue("");
    await expect(power).toHaveAttribute("placeholder", "1000");
    await validate.click();
    await expect(resultPower).toContainText("1.0 kW");

    await modal.getByRole("button", { name: "Save" }).click();
    await expectModalHidden(modal);
    await expect(grid.getByTestId("device-tag-power")).toContainText("1.0 kW");

    // persisted empty value still renders with the default
    await restart(undefined, templateFlags);
    await page.reload();
    await expect(grid.getByTestId("device-tag-power")).toContainText("1.0 kW");

    await grid.getByRole("button", { name: "edit" }).click();
    await expectModalVisible(modal);
    await expect(power).toHaveValue("1000");
  });
});
