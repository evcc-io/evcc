import { test, expect } from "@playwright/test";
import type { Page } from "@playwright/test";
import { start, stop, baseUrl } from "./evcc";
import { expectModalVisible } from "./utils";

test.use({ baseURL: baseUrl() });

const templateFlags = [
  "--disable-auth",
  "--template-type",
  "meter",
  "--template",
  "tests/config-param-service-modbus.tpl.yaml",
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
  await meterModal.getByLabel("Manufacturer").selectOption("Service Modbus Test Meter");
  return meterModal;
}

test.describe("modbus service expansion", async () => {
  test("tcp/ip and serial switching", async ({ page }) => {
    const meterModal = await openMeterModal(page);

    await meterModal.getByLabel("Register address").fill("100");
    await meterModal.getByLabel("IP address or hostname").fill("192.168.1.1");
    await meterModal.getByLabel("Port").fill("502");
    await expect(meterModal.getByLabel("Test value")).toHaveValue("100,id:2,tcp");

    await meterModal.getByLabel("Test value").clear();
    await meterModal.getByText("RS485").first().click();

    await meterModal.getByLabel("Modbus ID").fill("44");
    await meterModal.getByLabel("Device").fill("/dev/ttyUSB0");
    await meterModal.getByLabel("Baud rate").selectOption("9600");
    await meterModal.getByLabel("ComSet").selectOption("8N1");
    await expect(meterModal.getByLabel("Test value")).toHaveValue("100,id:44,serial");
  });

  test("no service call without connection params", async ({ page }) => {
    const meterModal = await openMeterModal(page);

    await meterModal.getByLabel("Register address").fill("100");

    await expect(meterModal.getByLabel("Test value")).toHaveValue("");
  });

  test("template default id and port are used", async ({ page }) => {
    const meterModal = await openMeterModal(page);

    await meterModal.getByLabel("Register address").fill("100");
    await meterModal.getByLabel("IP address or hostname").fill("192.168.1.1");
    await expect(meterModal.getByLabel("Test value")).toHaveValue("100,id:2,tcp");
  });

  test("template defaults are included in validation request", async ({ page }) => {
    const meterModal = await openMeterModal(page);

    // Fill required fields
    await meterModal.getByLabel("Register address").fill("100");
    await meterModal.getByLabel("IP address or hostname").fill("192.168.1.1");
    await meterModal.getByLabel("Test value").fill("test");

    // Intercept validation POST request
    const requestPromise = page.waitForRequest(
      (req) => req.url().includes("/api/config/test/meter") && req.method() === "POST"
    );
    await meterModal.getByRole("button", { name: "Save" }).click();
    const body = (await requestPromise).postDataJSON();

    // Template has id: 2 - verify it's included even though user didn't set it
    expect(body.id).toBe(2);
  });
});
