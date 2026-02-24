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
    await expect(ocppModal).toContainText("Detected station IDs");
    await expect(ocppModal).toContainText([OCPP_STATION_ID, "Unknown"].join(""));
    await expect(ocppModal).not.toContainText("No OCPP chargers detected.");
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
