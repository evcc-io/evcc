import { test, expect } from "@playwright/test";
import { start, stop, baseUrl, restart } from "./evcc";
import { expectModalVisible, enableExperimental, expectModalHidden } from "./utils";

test.use({ baseURL: baseUrl() });

test.beforeEach(async ({ page }) => {
  await start();
  await page.goto("/#/config");
  await enableExperimental(page, false);
});

test.afterEach(async () => {
  await stop();
});

test.describe("modbusproxy", async () => {
  test("modbusproxy not configured", async ({ page }) => {
    const modbusproxyCard = page.getByTestId("modbusproxy");

    await expect(modbusproxyCard).toBeVisible();
    await expect(modbusproxyCard).toContainText(["Configured", "no"].join(""));
  });

  test("modbusproxy via ui", async ({ page }) => {
    // add connection
    const modbusproxyCard = page.getByTestId("modbusproxy");

    await modbusproxyCard.getByRole("button", { name: "edit" }).click();
    const modal = await page.getByTestId("modbusproxy-modal");
    await expectModalVisible(modal);

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

    await expect(deviceBox.getByLabel("TCP")).toBeChecked();
    await deviceBox.locator('label[for="modbusRtu-0"]').click();

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

    await expect(evccBox.getByLabel("Port", { exact: true })).toHaveValue("501");
    await expect(evccBox.getByLabel("error")).toBeChecked();
    await expect(deviceBox.getByLabel("Network")).toBeChecked();
    await deviceBox.getByLabel("IP address or hostname").fill("127.0.0.1");
    await expect(deviceBox.getByLabel("Port", { exact: true })).toHaveValue("602");
    await expect(deviceBox.getByLabel("RTU")).toBeChecked();

    // remove connection
    await modal.getByRole("button", { name: "Remove" }).click();
    await expect(modal).not.toContainText("Connection #1");
  });
});
