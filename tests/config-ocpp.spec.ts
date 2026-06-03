import { test, expect, type Page } from "@playwright/test";
import { start, stop, baseUrl } from "./evcc";
import { startSimulator, stopSimulator, simulatorUrl, simulatorHost } from "./simulator";
import { expectModalVisible } from "./utils";
import axios from "axios";

test.use({ baseURL: baseUrl() });

const OCPP_STATION_ID = "test-station-001";

// open the OCPP modal on the evcc config page
async function openOcppModal(page: Page) {
  await page.goto("/#/config");
  const ocppModal = page.getByTestId("ocpp-modal");
  await page.getByTestId("ocpp").getByRole("button", { name: "edit" }).click();
  await expectModalVisible(ocppModal);
  return ocppModal;
}

async function evccServerUrl(page: Page) {
  const ocppModal = await openOcppModal(page);
  return ocppModal.getByLabel("Server URL").inputValue();
}

// connect a charger to evcc via the simulator UI
async function connectCharger(page: Page, serverUrl: string) {
  await page.goto(simulatorUrl());
  const card = page.getByTestId("ocpp-add-client");
  await card.getByLabel("Server URL").fill(serverUrl);
  await card.getByLabel("Station ID").fill(OCPP_STATION_ID);
  await card.getByRole("button", { name: "Connect" }).click();
}

// enable/disable the mock upstream OCPP server via the simulator UI. the toggle
// button name reflects the live state, so getByRole auto-waits past the initial fetch.
async function setMockServer(
  page: Page,
  opts: { enabled: boolean; username?: string; password?: string }
) {
  await page.goto(simulatorUrl());
  const server = page.getByTestId("ocpp-server");
  if (opts.enabled) {
    if (opts.username) await server.getByLabel("Username").fill(opts.username);
    if (opts.password) await server.getByLabel("Password").fill(opts.password);
    await server.getByRole("button", { name: "Enable server" }).click();
    await expect(server.getByRole("button", { name: "Disable server" })).toBeVisible();
  } else {
    await server.getByRole("button", { name: "Disable server" }).click();
    await expect(server.getByRole("button", { name: "Enable server" })).toBeVisible();
  }
}

// open the forwarder editor for the given station (whatever the button state)
async function openForwarderEditor(page: Page, stationId: string) {
  const ocppModal = await openOcppModal(page);
  const station = ocppModal.getByTestId("ocpp-station").filter({ hasText: stationId });
  await station.getByRole("button").click();
  const forwarderModal = page.getByTestId("ocppforwarder-modal");
  await expectModalVisible(forwarderModal);
  return { ocppModal, station, forwarderModal };
}

// enable the mock upstream server, connect a charger, open the forwarder editor
async function startForwarding(page: Page, creds?: { username: string; password: string }) {
  const serverUrl = await evccServerUrl(page);
  await setMockServer(page, { enabled: true, ...creds });
  await connectCharger(page, serverUrl);
  return openForwarderEditor(page, OCPP_STATION_ID);
}

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
    await expect(station.getByRole("img", { name: "Unknown" })).toBeVisible();
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

test.describe("ocpp forwarder", () => {
  test("unreachable upstream shows error", async ({ page }) => {
    await connectCharger(page, await evccServerUrl(page));
    const { ocppModal, station, forwarderModal } = await openForwarderEditor(page, OCPP_STATION_ID);

    await forwarderModal.getByLabel("Upstream server URL").fill("ws://localhost:1/ocpp");
    await forwarderModal.getByRole("button", { name: "Save" }).click();

    // modal stays open; upstream is unreachable, so the error surfaces in place
    await expectModalVisible(forwarderModal);
    await expect(forwarderModal.getByTestId("ocppforwarder-error")).toBeVisible();

    // remove the rule from the still-open modal
    await forwarderModal.getByRole("button", { name: "Remove" }).click();
    await expectModalVisible(ocppModal);
    await expect(station.getByRole("button", { name: "No forwarding configured" })).toBeVisible();
  });

  test("connects and drops when upstream stops", async ({ page }) => {
    const { forwarderModal } = await startForwarding(page);

    await forwarderModal.getByLabel("Upstream server URL").fill(`ws://${simulatorHost()}`);
    await forwarderModal.getByRole("button", { name: "Save" }).click();
    await expect(forwarderModal.getByTestId("ocppforwarder-status")).toContainText("Connected");

    // simulator UI shows our station as the active upstream connection
    await page.goto(simulatorUrl());
    const lastStation = page.getByTestId("ocpp-server-last-station");
    await expect(lastStation).toContainText(OCPP_STATION_ID);
    await expect(lastStation).toContainText("active");

    // disabling the upstream server drops the forwarder connection
    await setMockServer(page, { enabled: false });
    const reopened = await openForwarderEditor(page, OCPP_STATION_ID);
    await expect(reopened.forwarderModal.getByTestId("ocppforwarder-status")).toContainText(
      "Not connected"
    );
  });

  test("basic auth", async ({ page }) => {
    const { forwarderModal } = await startForwarding(page, {
      username: "user",
      password: "secret",
    });

    await forwarderModal.getByLabel("Upstream server URL").fill(`ws://${simulatorHost()}`);
    await forwarderModal.getByLabel("Username").fill("user");
    await forwarderModal.getByLabel("Password").fill("secret");
    await forwarderModal.getByRole("button", { name: "Save" }).click();
    await expect(forwarderModal.getByTestId("ocppforwarder-status")).toContainText("Connected");

    // simulator UI shows our station as the active upstream connection
    await page.goto(simulatorUrl());
    await expect(page.getByTestId("ocpp-server-last-station")).toContainText(OCPP_STATION_ID);
  });

  test("param change reconnects", async ({ page }) => {
    const { forwarderModal } = await startForwarding(page, {
      username: "user",
      password: "secret",
    });
    const status = forwarderModal.getByTestId("ocppforwarder-status");

    // wrong password is rejected
    await forwarderModal.getByLabel("Upstream server URL").fill(`ws://${simulatorHost()}`);
    await forwarderModal.getByLabel("Username").fill("user");
    await forwarderModal.getByLabel("Password").fill("wrong");
    await forwarderModal.getByRole("button", { name: "Save" }).click();
    await expect(forwarderModal.getByTestId("ocppforwarder-error")).toBeVisible();
    await expect(status).toContainText("Not connected");

    // fixing the password re-establishes the connection
    await forwarderModal.getByLabel("Password").fill("secret");
    await forwarderModal.getByRole("button", { name: "Save" }).click();
    await expect(status).toContainText("Connected");
  });

  test("removing rule stops forwarding", async ({ page }) => {
    const { ocppModal, station, forwarderModal } = await startForwarding(page);

    await forwarderModal.getByLabel("Upstream server URL").fill(`ws://${simulatorHost()}`);
    await forwarderModal.getByRole("button", { name: "Save" }).click();
    await expect(forwarderModal.getByTestId("ocppforwarder-status")).toContainText("Connected");

    // removing the rule tears down forwarding
    await forwarderModal.getByRole("button", { name: "Remove" }).click();
    await expectModalVisible(ocppModal);
    await expect(station.getByRole("button", { name: "No forwarding configured" })).toBeVisible();

    // simulator UI no longer shows an active upstream connection
    await page.goto(simulatorUrl());
    await expect(page.getByTestId("ocpp-server-last-station")).not.toContainText("active");
  });
});
