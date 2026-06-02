import { test, expect } from "@playwright/test";
import { start, stop, baseUrl } from "./evcc";
import { startSimulator, stopSimulator, simulatorUrl } from "./simulator";
import { expectModalVisible } from "./utils";
import axios from "axios";

test.use({ baseURL: baseUrl() });

const OCPP_STATION_ID = "test-station-001";

test.beforeEach(async () => {
  await startSimulator();
  await start();
});

test.afterEach(async () => {
  await stop();
  await stopSimulator();
});

test.describe("ocpp", () => {
  test("ocpp modal", async ({ page }) => {
    // Navigate to config page
    await page.goto("/#/config");

    // Open OCPP card
    const ocppCard = page.getByTestId("ocpp");
    await ocppCard.getByRole("button", { name: "edit" }).click();

    const ocppModal = page.getByTestId("ocpp-modal");
    await expectModalVisible(ocppModal);
    const serverUrl = await ocppModal.getByLabel("Server URL").inputValue();
    await expect(ocppModal).toContainText("No OCPP chargers detected.");

    // Connect OCPP client via simulator UI
    await page.goto(simulatorUrl());

    // Fill in OCPP connection details
    const addClientCard = page.getByTestId("ocpp-add-client");
    await addClientCard.getByLabel("Server URL").fill(serverUrl);
    await addClientCard.getByLabel("Station ID").fill(OCPP_STATION_ID);
    await addClientCard.getByRole("button", { name: "Connect" }).click();

    // Navigate back to evcc config page
    await page.goto("/#/config");
    await ocppCard.getByRole("button", { name: "edit" }).click();
    await expectModalVisible(ocppModal);
    await expect(ocppModal).toContainText("Station IDs");
    const station = ocppModal.getByTestId("ocpp-station");
    await expect(station).toContainText(OCPP_STATION_ID);
    await expect(station).toContainText("Unknown");
    await expect(ocppModal).not.toContainText("No OCPP chargers detected.");
  });

  test("ocpp forwarder", async ({ page }) => {
    await page.goto("/#/config");

    // open OCPP modal and read the server URL
    const ocppCard = page.getByTestId("ocpp");
    const ocppModal = page.getByTestId("ocpp-modal");
    await ocppCard.getByRole("button", { name: "edit" }).click();
    await expectModalVisible(ocppModal);
    const serverUrl = await ocppModal.getByLabel("Server URL").inputValue();

    // connect a station via the simulator
    await page.goto(simulatorUrl());
    const addClientCard = page.getByTestId("ocpp-add-client");
    await addClientCard.getByLabel("Server URL").fill(serverUrl);
    await addClientCard.getByLabel("Station ID").fill(OCPP_STATION_ID);
    await addClientCard.getByRole("button", { name: "Connect" }).click();

    // back in evcc: open the forwarder editor for the station
    await page.goto("/#/config");
    await ocppCard.getByRole("button", { name: "edit" }).click();
    await expectModalVisible(ocppModal);
    const station = ocppModal.getByTestId("ocpp-station").filter({ hasText: OCPP_STATION_ID });
    await station.getByRole("button", { name: "No forwarding configured" }).click();

    const forwarderModal = page.getByTestId("ocppforwarder-modal");
    await expectModalVisible(forwarderModal);
    await forwarderModal.getByLabel("Upstream server URL").fill("ws://localhost:1/ocpp");
    await forwarderModal.getByRole("button", { name: "Save" }).click();

    // upstream is unreachable, so forwarding shows an error
    await expectModalVisible(ocppModal);
    await expect(station.getByRole("button", { name: "Forwarding error" })).toBeVisible();

    // remove the rule again
    await station.getByRole("button", { name: "Forwarding error" }).click();
    await expectModalVisible(forwarderModal);
    await forwarderModal.getByRole("button", { name: "Remove" }).click();
    await expectModalVisible(ocppModal);
    await expect(station.getByRole("button", { name: "No forwarding configured" })).toBeVisible();
  });

  test("create ocpp charger", async ({ page }) => {
    await page.goto("/#/config");

    // Open loadpoint modal and select charging point type
    await page.getByRole("button", { name: "Add charger or heater" }).click();
    const lpModal = page.getByTestId("loadpoint-modal");
    await expectModalVisible(lpModal);
    await lpModal.getByRole("button", { name: "Charging point" }).click();
    await lpModal.getByLabel("Title").fill("OCPP Test Charger");

    // Open charger modal and select OCPP 1.6J
    await lpModal.getByRole("button", { name: "Add charger" }).click();
    const chargerModal = page.getByTestId("charger-modal");
    await expectModalVisible(chargerModal);
    await chargerModal.getByLabel("Manufacturer").selectOption("OCPP 1.6J compatible");

    // Verify waiting for connection state and extract server URL
    const serverUrl = await chargerModal.getByLabel("OCPP-Server URL").inputValue();
    const waitingButton = chargerModal.getByRole("button", { name: "Waiting for connection" });
    await expect(waitingButton).toBeVisible();

    // Connect OCPP client via simulator REST API
    await axios.post(`${simulatorUrl()}/api/ocpp/connect`, {
      stationId: OCPP_STATION_ID,
      serverUrl: serverUrl,
    });

    // Verify connection successful
    await expect(chargerModal).toContainText("Connected!");
    await expect(waitingButton).not.toBeVisible();

    // Proceed to validation step
    await chargerModal.getByRole("button", { name: "Next step" }).click();
    await expect(chargerModal.getByLabel("Station ID")).toHaveValue(OCPP_STATION_ID);

    // Validate and verify sponsor token error
    const testResult = chargerModal.getByTestId("test-result");
    await testResult.getByRole("link", { name: "Validate" }).click();
    await expect(testResult).toContainText("No sponsor token configured.");
  });
});
