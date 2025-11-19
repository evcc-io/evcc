import { test, expect } from "@playwright/test";
import { start, stop, baseUrl } from "./evcc";
import { startSimulator, stopSimulator, simulatorUrl } from "./simulator";
import { expectModalVisible, enableExperimental } from "./utils";

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
  test("ocpp config status", async ({ page }) => {
    // Navigate to config page
    await page.goto("/#/config");
    await enableExperimental(page, true);

    // Open OCPP card
    const ocppCard = page.getByTestId("ocpp");
    await expect(ocppCard).toContainText(["Configured", "no"].join(""));
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
    await expect(ocppCard).toContainText(["Connections", "0/0", "Detected", "1"].join(""));
    await ocppCard.getByRole("button", { name: "edit" }).click();
    await expectModalVisible(ocppModal);
    await expect(ocppModal).toContainText("Detected station IDs");
    await expect(ocppModal).toContainText([OCPP_STATION_ID, "Unknown"].join(""));
    await expect(ocppModal).not.toContainText("No OCPP chargers detected.");
  });
});
