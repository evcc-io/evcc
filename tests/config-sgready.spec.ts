import { test, expect, type Locator } from "@playwright/test";
import { start, stop, restart, baseUrl } from "./evcc";
import { startSimulator, stopSimulator, simulatorHost } from "./simulator";
import {
  expectModalHidden,
  expectModalVisible,
  newLoadpoint,
  finishLoadpoint,
  LoadpointType,
} from "./utils";

test.use({ baseURL: baseUrl() });
test.describe.configure({ mode: "parallel" });

test.afterEach(async () => {
  await stop();
});

async function saveRelay(relayModal: Locator, chargerModal: Locator) {
  await relayModal.getByRole("button", { name: "Validate & save" }).click();
  await expectModalHidden(relayModal);
  await expectModalVisible(chargerModal);
}

test.describe("sgready relays", async () => {
  test("create, edit, remove relay and delete heater", async ({ page }) => {
    await start();
    await page.goto("/#/config");

    const lpModal = page.getByTestId("loadpoint-modal");
    const chargerModal = page.getByTestId("charger-modal");
    const relayModal = page.getByTestId("subcharger-modal");
    const boost = chargerModal.getByTestId("device-ref-boost");
    const dim = chargerModal.getByTestId("device-ref-dim");

    // heater with sgready relay charger
    await newLoadpoint(page, "Wärmepumpe", LoadpointType.Heating);
    await lpModal.getByRole("button", { name: "Add heating device" }).click();
    await expectModalVisible(chargerModal);
    await chargerModal.getByLabel("Manufacturer").selectOption("SG Ready via relays");

    // boost relay is required
    await chargerModal.getByRole("button", { name: "Validate & save" }).click();
    await expect(boost).toContainText("Add switchable socket");
    await expectModalVisible(chargerModal);

    // add boost relay in nested modal
    await chargerModal.getByLabel("Boost relay").click();
    await expectModalVisible(relayModal);
    await expect(relayModal.getByRole("heading", { name: "Add Switchable Socket" })).toBeVisible();
    await relayModal.getByLabel("Manufacturer").selectOption("Demo switch");
    await saveRelay(relayModal, chargerModal);
    await expect(boost).toContainText("Demo switch");
    await expect(dim).toContainText("Add switchable socket");

    await chargerModal.getByRole("button", { name: "Validate & save" }).click();
    await expectModalHidden(chargerModal);
    await expectModalVisible(lpModal);
    await expect(lpModal.getByRole("heading", { name: "Heater created" })).toBeVisible();
    await finishLoadpoint(page);

    // survives restart
    await restart();
    await page.reload();
    const lpEntry = page.getByTestId("loadpoint");
    await expect(lpEntry).toContainText("Wärmepumpe");
    await expect(lpEntry).toContainText(["Status", "normal"].join(""));
    await expect(lpEntry).toContainText(["Boosted", "no"].join(""));
    await expect(lpEntry).toContainText(["Dimmed", "no"].join(""));
    await lpEntry.getByRole("button", { name: "edit" }).click();
    await expectModalVisible(lpModal);
    await lpModal.getByText("SG Ready via relays").click();
    await expectModalVisible(chargerModal);
    await expect(boost).toContainText("Demo switch");
    const testResult = chargerModal.getByTestId("test-result");
    await testResult.getByRole("link", { name: "validate" }).click();
    await expect(testResult).toContainText(["Status", "normal"].join(""));
    await expect(testResult).toContainText(["Boosted", "no"].join(""));
    await expect(testResult).toContainText(["Dimmed", "no"].join(""));

    // edit relay
    await boost.getByText("Demo switch").click();
    await expectModalVisible(relayModal);
    await expect(relayModal.getByRole("heading", { name: "Edit Switchable Socket" })).toBeVisible();
    await relayModal.getByLabel("Standby power").fill("20");
    await saveRelay(relayModal, chargerModal);

    // abandoned dim relay is removed again
    await chargerModal.getByLabel("Dim relay").click();
    await expectModalVisible(relayModal);
    await relayModal.getByLabel("Manufacturer").selectOption("Demo switch");
    await saveRelay(relayModal, chargerModal);
    await expect(dim).toContainText("Demo switch");
    await chargerModal.getByRole("button", { name: "Close" }).click();
    await expectModalHidden(chargerModal);
    await expectModalVisible(lpModal);
    await lpModal.getByText("SG Ready via relays").click();
    await expectModalVisible(chargerModal);
    await expect(dim).toContainText("Add switchable socket");

    // deleting the relay clears the reference
    await boost.getByText("Demo switch").click();
    await expectModalVisible(relayModal);
    await relayModal.getByRole("button", { name: "Delete" }).click();
    await expectModalHidden(relayModal);
    await expectModalVisible(chargerModal);
    await expect(boost).toContainText("Add switchable socket");

    // add a new relay and save
    await chargerModal.getByLabel("Boost relay").click();
    await expectModalVisible(relayModal);
    await relayModal.getByLabel("Manufacturer").selectOption("Demo switch");
    await saveRelay(relayModal, chargerModal);
    await chargerModal.getByRole("button", { name: "Validate & save" }).click();
    await expectModalHidden(chargerModal);
    await expectModalVisible(lpModal);

    // deleting the heater removes charger and relay
    await lpModal.getByRole("button", { name: "Delete" }).click();
    await expectModalHidden(lpModal);
    await expect(lpEntry).toHaveCount(0);
  });

  test("broken relay guides to the relay", async ({ page }) => {
    await startSimulator();
    await start();
    await page.goto("/#/config");

    const lpModal = page.getByTestId("loadpoint-modal");
    const chargerModal = page.getByTestId("charger-modal");
    const relayModal = page.getByTestId("subcharger-modal");
    const boost = chargerModal.getByTestId("device-ref-boost");
    const lpEntry = page.getByTestId("loadpoint");
    const fatal = page.getByTestId("fatal-error");

    // heater with shelly relay
    await newLoadpoint(page, "Wärmepumpe", LoadpointType.Heating);
    await lpModal.getByRole("button", { name: "Add heating device" }).click();
    await expectModalVisible(chargerModal);
    await chargerModal.getByLabel("Manufacturer").selectOption("SG Ready via relays");
    await chargerModal.getByLabel("Boost relay").click();
    await expectModalVisible(relayModal);
    await relayModal.getByLabel("Manufacturer").selectOption("Shelly 1");
    await relayModal.getByLabel("IP address or hostname").fill(simulatorHost());
    await saveRelay(relayModal, chargerModal);
    await expect(boost).toContainText("Shelly 1");
    await chargerModal.getByRole("button", { name: "Validate & save" }).click();
    await expectModalHidden(chargerModal);
    await finishLoadpoint(page);

    // break relay
    await stopSimulator();
    await restart();
    await page.reload();
    await expect(fatal).toBeVisible();
    await expect(fatal).toContainText("cannot create charger type 'shelly'");

    // follow the error to the relay and replace it
    await lpEntry.getByRole("button", { name: "edit" }).click();
    await expectModalVisible(lpModal);
    await lpModal.getByText("SG Ready via relays").click();
    await expectModalVisible(chargerModal);
    await boost.getByText("Shelly 1").click();
    await expectModalVisible(relayModal);
    await expect(relayModal.getByLabel("IP address or hostname")).toHaveValue(simulatorHost());
    await relayModal.getByRole("button", { name: "Delete" }).click();
    await expectModalHidden(relayModal);
    await expectModalVisible(chargerModal);
    await expect(boost).toContainText("Add switchable socket");
    await chargerModal.getByLabel("Boost relay").click();
    await expectModalVisible(relayModal);
    await relayModal.getByLabel("Manufacturer").selectOption("Demo switch");
    await saveRelay(relayModal, chargerModal);
    await chargerModal.getByRole("button", { name: "Validate & save" }).click();
    await expectModalHidden(chargerModal);

    await restart();
    await page.reload();
    await expect(fatal).not.toBeVisible();
    await expect(lpEntry).toContainText(["Status", "normal"].join(""));
  });
});
