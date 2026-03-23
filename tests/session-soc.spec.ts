import { test, expect, type Page } from "@playwright/test";
import { start, stop, baseUrl } from "./evcc";
import { expectModalVisible } from "./utils";
import {
  startSimulator,
  stopSimulator,
  simulatorUrl,
  simulatorConfig,
  simulatorApply,
} from "./simulator";

const date = new Date();
const YEAR = date.getFullYear();
const MONTH = date.getMonth() + 1;

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

async function disconnect(page: Page) {
  await page.goto(simulatorUrl());
  await page.getByTestId("loadpoint0").getByText("A (disconnected)").click();
  await simulatorApply(page);
  await page.goto("/");
  await expect(page.getByTestId("vehicle-status")).toHaveText("Disconnected.");
}

async function verifySessionSoc(page: Page, startSoc: string, endSoc: string) {
  await page.goto(`/#/sessions?year=${YEAR}&month=${MONTH}`);
  await expect(page.getByTestId("sessions-entry")).toHaveCount(1);
  await page.getByTestId("sessions-entry").click();
  const modal = page.getByTestId("session-details");
  await expectModalVisible(modal);
  const socElement = modal.getByTestId("session-details-soc");
  await expect(socElement).toContainText(startSoc);
  await expect(socElement).toContainText(endSoc);
}

test("session records soc range", async ({ page }) => {
  // connect vehicle with 20% SoC
  await page.goto(simulatorUrl());
  await page.getByTestId("vehicle0").getByLabel("SoC").fill("20");
  await page.getByTestId("loadpoint0").getByLabel("Energy").fill("0");
  await page.getByTestId("loadpoint0").getByText("B (connected)").click();
  await simulatorApply(page);

  await page.goto("/");
  await expect(page.getByTestId("vehicle-status")).toHaveText("Connected.");
  await expect(page.getByTestId("current-soc")).toContainText("20");

  // start charging
  await page.goto(simulatorUrl());
  await page.getByTestId("loadpoint0").getByLabel("Power").fill("6000");
  await page.getByTestId("loadpoint0").getByText("C (charging)").click();
  await page.getByTestId("loadpoint0").getByText("Enabled").check();
  await simulatorApply(page);

  await page.goto("/");
  await expect(page.getByTestId("vehicle-status")).toContainText("Charging");

  // update SoC to 80% and energy to 10 kWh
  await page.goto(simulatorUrl());
  await page.getByTestId("loadpoint0").getByLabel("Energy").fill("10");
  await page.getByTestId("vehicle0").getByLabel("SoC").fill("80");
  await simulatorApply(page);

  await page.goto("/");
  await expect(page.getByTestId("current-soc")).toContainText("80");

  await disconnect(page);
  await verifySessionSoc(page, "20%", "80%");
});

test("session soc resets on vehicle switch", async ({ page }) => {
  // connect golf at 20% SoC
  await page.goto(simulatorUrl());
  await page.getByTestId("vehicle0").getByLabel("SoC").fill("20");
  await page.getByTestId("loadpoint0").getByLabel("Energy").fill("0");
  await page.getByTestId("loadpoint0").getByText("B (connected)").click();
  await simulatorApply(page);

  await page.goto("/");
  await expect(page.getByTestId("current-soc")).toContainText("20");

  // start charging
  await page.goto(simulatorUrl());
  await page.getByTestId("loadpoint0").getByLabel("Power").fill("6000");
  await page.getByTestId("loadpoint0").getByText("C (charging)").click();
  await page.getByTestId("loadpoint0").getByText("Enabled").check();
  await simulatorApply(page);

  await page.goto("/");
  await expect(page.getByTestId("vehicle-status")).toContainText("Charging");

  await page.goto(simulatorUrl());
  await page.getByTestId("loadpoint0").getByLabel("Energy").fill("1");
  await simulatorApply(page);

  // switch vehicle to tesla
  await page.goto("/");
  await page.getByTestId("change-vehicle").locator("select").selectOption("weißes Model 3");

  // set tesla SoC to 50%
  await page.goto(simulatorUrl());
  await page.getByTestId("vehicle1").getByLabel("SoC").fill("50");
  await simulatorApply(page);

  await page.goto("/");
  await expect(page.getByTestId("current-soc")).toContainText("50");

  // charge to 70%
  await page.goto(simulatorUrl());
  await page.getByTestId("loadpoint0").getByLabel("Energy").fill("10");
  await page.getByTestId("vehicle1").getByLabel("SoC").fill("70");
  await simulatorApply(page);

  await page.goto("/");
  await expect(page.getByTestId("current-soc")).toContainText("70");

  await disconnect(page);
  await verifySessionSoc(page, "50%", "70%");
});
