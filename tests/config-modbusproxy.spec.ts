import { test, expect } from "@playwright/test";
import { start, stop, baseUrl, restart } from "./evcc";
import { expectModalVisible, enableExperimental, expectModalHidden } from "./utils";

test.use({ baseURL: baseUrl() });
test.describe.configure({ mode: "parallel" });

const CONFIG_MODBUSPROXY_MIGRATE = "config-modbusproxy-migrate.sql";

test.afterEach(async () => {
  await stop();
});

test.describe("modbusproxy", async () => {
  test("modbusproxy not configured", async ({ page }) => {
    await start();
    await page.goto("/#/config");
    await enableExperimental(page, false);

    const modbusproxyCard = page.getByTestId("modbusproxy");

    await expect(modbusproxyCard).toBeVisible();
    await expect(modbusproxyCard).toContainText(["Configured", "no"].join(""));
  });

  test("modbusproxy via ui", async ({ page }) => {
    await start();
    await page.goto("/#/config");
    await enableExperimental(page, false);

    // add connection
    const modbusproxyCard = page.getByTestId("modbusproxy");

    await modbusproxyCard.getByRole("button", { name: "edit" }).click();
    const modal = await page.getByTestId("modbusproxy-modal");
    await expectModalVisible(modal);

    await expect(modal).toContainText("This feature requires a sponsor token.");

    await modal.getByRole("button", { name: "Add proxy connection" }).click();
    await expect(modal).toContainText("Connection #1");

    const evccBox = modal.getByTestId("evcc-box");
    const deviceBox = modal.getByTestId("device-box");

    const evccPort = evccBox.getByLabel("Port");
    await expect(evccPort).toHaveValue("1502");
    await evccPort.fill("501");

    await expect(evccBox.getByLabel("no")).toBeChecked();
    await evccBox.getByLabel("error").click();

    await expect(deviceBox.getByLabel("Network")).toBeChecked();
    await deviceBox.getByLabel("IP address or hostname").fill("127.0.0.1");

    const devicePort = deviceBox.getByLabel("Port");
    await expect(devicePort).toHaveValue("502");
    await devicePort.fill("602");

    await expect(deviceBox.getByText("TCP")).toBeChecked();
    await deviceBox.getByText("RTU").click();

    // validate connection
    await modal.getByRole("button", { name: "Save" }).click();
    await expectModalHidden(modal);

    await expect(modbusproxyCard).toContainText(["Amount", "1"].join(""));

    // restart button appears
    const restartButton = await page
      .getByTestId("bottom-banner")
      .getByRole("button", { name: "Restart" });
    await expect(restartButton).toBeVisible();

    await restart();
    await page.reload();

    await modbusproxyCard.getByRole("button", { name: "edit" }).click();

    await expect(evccBox.getByLabel("Port")).toHaveValue("501");
    await expect(evccBox.getByLabel("error")).toBeChecked();
    await expect(deviceBox.getByLabel("Network")).toBeChecked();
    await deviceBox.getByLabel("IP address or hostname").fill("127.0.0.1");
    await expect(deviceBox.getByLabel("Port")).toHaveValue("602");
    await expect(deviceBox.getByText("RTU")).toBeChecked();

    // remove connection
    await modal.getByRole("button", { name: "Remove" }).click();
    await expect(modal).not.toContainText("Connection #1");
  });

  test("modbusproxy via db (yaml to json migration)", async ({ page }) => {
    await start(undefined, CONFIG_MODBUSPROXY_MIGRATE);
    await page.goto("/#/config");
    await enableExperimental(page, false);

    const modbusproxyCard = page.getByTestId("modbusproxy");
    await expect(modbusproxyCard).toContainText(["Amount", "3"].join(""));

    await modbusproxyCard.getByRole("button", { name: "edit" }).click();
    const modal = await page.getByTestId("modbusproxy-modal");
    await expectModalVisible(modal);

    const connections = modal.getByTestId("modbusproxy-connection");
    await expect(connections).toHaveCount(3);

    // Connection #1: 192.0.2.2:502 (TCP)
    const connection1 = connections.nth(0);
    await expect(connection1).toContainText("Connection #1");
    await expect(connection1.getByTestId("evcc-box").getByLabel("Port")).toHaveValue("5021");
    const device1 = connection1.getByTestId("device-box");
    await expect(device1.getByLabel("Network")).toBeChecked();
    await expect(device1.getByLabel("IP address or hostname")).toHaveValue("192.0.2.2");
    await expect(device1.getByLabel("Port")).toHaveValue("502");
    await expect(device1.getByText("TCP")).toBeChecked();

    // Connection #2: /dev/ttyUSB0 9600 8N1
    const connection2 = connections.nth(1);
    await expect(connection2).toContainText("Connection #2");
    await expect(connection2.getByTestId("evcc-box")).toBeVisible();
    await expect(connection2.getByTestId("evcc-box").getByLabel("Port")).toBeVisible();
    await expect(connection2.getByTestId("evcc-box").getByLabel("Port")).toHaveValue("5022");
    const device2 = connection2.getByTestId("device-box");
    await expect(device2.getByLabel("RS485")).toBeChecked();
    await expect(device2.getByLabel("Device")).toHaveValue("/dev/ttyUSB0");
    await expect(device2.getByLabel("Baud rate")).toHaveValue("9600");
    await expect(device2.getByLabel("ComSet")).toHaveValue("8N1");

    // Connection #3: 192.0.2.3:502 (RTU)
    const connection3 = connections.nth(2);
    await expect(connection3).toContainText("Connection #3");
    await expect(connection3.getByTestId("evcc-box").getByLabel("Port")).toHaveValue("5023");
    const device3 = connection3.getByTestId("device-box");
    await expect(device3.getByLabel("Network")).toBeChecked();
    await expect(device3.getByLabel("IP address or hostname")).toHaveValue("192.0.2.3");
    await expect(device3.getByLabel("Port")).toHaveValue("502");
    await expect(device3.getByText("RTU")).toBeChecked();
  });
});
