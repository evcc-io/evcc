import { test, expect } from "@playwright/test";
import { start, stop, baseUrl } from "./evcc";
import { expectModalVisible } from "./utils";
import {
  startSimulator,
  stopSimulator,
  simulatorUrl,
  simulatorConfig,
  simulatorApply,
} from "./simulator";

test.use({ baseURL: baseUrl() });

test.beforeAll(async () => {
  await startSimulator();
});
test.afterAll(async () => {
  await stopSimulator();
});

test.beforeEach(async ({ page }) => {
  await start(simulatorConfig());

  // start disconnected
  await page.goto(simulatorUrl());
  await page.getByTestId("loadpoint0").getByText("A (disconnected)").click();
  await simulatorApply(page);
});

test.afterEach(async () => {
  await stop();
});

test("soc range", async ({ page }) => {
  const loadpoint = page.getByTestId("loadpoint0");
  const vehicle = page.getByTestId("vehicle0");

  // connect vehicle with 20% SoC
  await page.goto(simulatorUrl());
  await vehicle.getByLabel("SoC").fill("20");
  await loadpoint.getByLabel("Energy").fill("0");
  await loadpoint.getByText("B (connected)").click();
  await simulatorApply(page);

  await page.goto("/");
  await expect(page.getByTestId("vehicle-status")).toHaveText("Connected.");
  await expect(page.getByTestId("current-soc")).toContainText("20");

  // start charging
  await page.goto(simulatorUrl());
  await loadpoint.getByLabel("Power").fill("6000");
  await loadpoint.getByText("C (charging)").click();
  await loadpoint.getByText("Enabled").check();
  await simulatorApply(page);

  await page.goto("/");
  await expect(page.getByTestId("vehicle-status")).toContainText("Charging");

  // update SoC to 80% and energy to 10 kWh
  await page.goto(simulatorUrl());
  await loadpoint.getByLabel("Energy").fill("10");
  await vehicle.getByLabel("SoC").fill("80");
  await simulatorApply(page);

  await page.goto("/");
  await expect(page.getByTestId("current-soc")).toContainText("80");

  // disconnect
  await page.goto(simulatorUrl());
  await loadpoint.getByText("A (disconnected)").click();
  await simulatorApply(page);
  await page.goto("/");
  await expect(page.getByTestId("vehicle-status")).toHaveText("Disconnected.");

  // verify session soc range
  await page.goto("/#/sessions");
  await expect(page.getByTestId("sessions-entry")).toHaveCount(1);
  await page.getByTestId("sessions-entry").click();
  const modal = page.getByTestId("session-details");
  await expectModalVisible(modal);
  await expect(modal.getByTestId("session-details-soc")).toContainText("20 – 80%");
});
