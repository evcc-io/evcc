import { test, expect } from "@playwright/test";
import { start, stop, baseUrl } from "./evcc";
import { expectModalHidden, expectModalVisible } from "./utils";

test.use({ baseURL: baseUrl() });
test.describe.configure({ mode: "parallel" });

test.beforeAll(async () => {
  await start();
});
test.afterAll(async () => {
  await stop();
});

test.describe("deep linking", async () => {
  test("browser back closes modal", async ({ page }) => {
    await page.goto("/#/config");

    // Open grid meter modal and verify URL param is added
    await page.getByTestId("add-grid").click();
    const meterModal = page.getByTestId("meter-modal");
    await expectModalVisible(meterModal);
    const urlWithModal = page.url();
    expect(urlWithModal).toContain("meter[type:grid]");

    // Use browser back to close modal and verify URL param is removed
    await page.goBack();
    await expectModalHidden(meterModal);
    const urlAfterBack = page.url();
    expect(urlAfterBack).not.toContain("meter[type:grid]");
    expect(urlAfterBack).toContain("/#/config");
  });

  test("direct navigation opens modal", async ({ page }) => {
    // Navigate directly with query param and verify modal opens without clicking
    await page.goto("/#/config?mqtt");
    const mqttModal = page.getByTestId("mqtt-modal");
    await expectModalVisible(mqttModal);

    // Close modal and verify URL is updated back to config page
    await mqttModal.getByRole("button", { name: "Cancel" }).click();
    await expectModalHidden(mqttModal);
    await page.waitForURL("**/#/config");
  });

  test("deep link persists after page reload", async ({ page }) => {
    await page.goto("/#/config");

    // Create and save an offline vehicle with title "Test Car"
    await page.getByTestId("add-vehicle").click();
    const vehicleModal = page.getByTestId("vehicle-modal");
    await expectModalVisible(vehicleModal);
    await vehicleModal.getByLabel("Title").fill("Test Car");
    await vehicleModal.getByRole("button", { name: "Save" }).click();
    await expectModalHidden(vehicleModal);

    // Edit vehicle, verify URL param persists after reload
    await page.getByTestId("vehicle").getByRole("button", { name: "edit" }).click();
    await expectModalVisible(vehicleModal);
    expect(page.url()).toContain("vehicle=1");
    await page.reload();
    await expectModalVisible(vehicleModal);
    await expect(vehicleModal.getByLabel("Title")).toHaveValue("Test Car");
  });
});
