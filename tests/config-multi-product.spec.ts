import { test, expect } from "@playwright/test";
import { start, stop, restart, baseUrl } from "./evcc";
import { expectModalVisible, expectModalHidden } from "./utils";

test.use({ baseURL: baseUrl() });

const flags = ["--disable-auth", "--template-type", "meter", "--template"];
const v1Flags = [...flags, "tests/config-multi-product-v1.tpl.yaml"];
const v2Flags = [...flags, "tests/config-multi-product-v2.tpl.yaml"];

test.beforeEach(async () => {
  await start(undefined, undefined, v1Flags);
});
test.afterEach(async () => {
  await stop();
});

test.describe("template with multiple products", async () => {
  test("keeps the selected product name", async ({ page }) => {
    await page.goto("/#/config");

    await page.getByRole("button", { name: "Add grid meter" }).click();
    const meterModal = page.getByTestId("meter-modal");
    await expectModalVisible(meterModal);

    // both products resolve to the same template, pick the second one (brand + description)
    await meterModal.getByLabel("Manufacturer").selectOption("Zeta Product B");
    await meterModal.getByLabel("Power").fill("5000");
    await meterModal.getByRole("button", { name: "Save" }).click();
    await expectModalHidden(meterModal);

    await page.getByTestId("grid").getByRole("button", { name: "edit" }).click();
    await expectModalVisible(meterModal);
    await expect(meterModal.getByLabel("Manufacturer")).toHaveValue("Zeta Product B");
    await meterModal.getByLabel("Close").click();
    await expectModalHidden(meterModal);

    // product renamed in template (software update): persisted name must survive
    await restart(undefined, v2Flags);
    await page.reload();

    await page.getByTestId("grid").getByRole("button", { name: "edit" }).click();
    await expectModalVisible(meterModal);
    await expect(meterModal.getByLabel("Manufacturer")).toHaveValue("Zeta Product B");
  });
});
