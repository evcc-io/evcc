import { test, expect, type Page, type Locator } from "@playwright/test";
import { start, stop, baseUrl } from "./evcc";
import { expectModalVisible, enableExperimental } from "./utils";

const CONFIG_MODBUS_FIELDS = "config-modbus-fields.sql";

test.use({ baseURL: baseUrl() });

test.beforeAll(async () => {
  await start(undefined, CONFIG_MODBUS_FIELDS);
});

test.afterAll(async () => {
  await stop();
});

async function openMeterModal(page: Page, title: string): Promise<Locator> {
  await page.goto("/#/config");
  await enableExperimental(page, true);
  await page
    .getByTestId("pv")
    .filter({ hasText: title })
    .getByRole("button", { name: "edit" })
    .click();
  const modal = page.getByTestId("meter-modal");
  await expectModalVisible(modal);
  return modal;
}

test.describe("modbus fields", async () => {
  test("tcpip", async ({ page }) => {
    const modal = await openMeterModal(page, "TCP Test");
    await expect(page.getByLabel("Network")).toBeChecked();
    await expect(page.getByLabel("TCP")).toBeChecked();
    await expect(modal.getByLabel("IP address or hostname")).toHaveValue("192.168.1.10");
    await expect(modal.getByLabel("Port", { exact: true })).toHaveValue("5020");
    await expect(modal.getByLabel("Modbus ID")).toHaveValue("10");
  });

  test("rs485tcpip", async ({ page }) => {
    const modal = await openMeterModal(page, "RTU/IP Test");
    await expect(page.getByLabel("Network")).toBeChecked();
    await expect(page.getByLabel("RTU")).toBeChecked();
    await expect(modal.getByLabel("IP address or hostname")).toHaveValue("192.168.1.20");
    await expect(modal.getByLabel("Port", { exact: true })).toHaveValue("8899");
    await expect(modal.getByLabel("Modbus ID")).toHaveValue("20");
  });

  test("rs485serial", async ({ page }) => {
    const modal = await openMeterModal(page, "Serial Test");
    await expect(page.getByLabel("Serial / USB")).toBeChecked();
    await expect(modal.getByLabel("Device name")).toHaveValue("/dev/ttyUSB5");
    await expect(modal.getByLabel("Baud rate")).toHaveValue("19200");
    await expect(modal.getByLabel("ComSet")).toHaveValue("8E1");
    await expect(modal.getByLabel("Modbus ID")).toHaveValue("30");
  });
});
