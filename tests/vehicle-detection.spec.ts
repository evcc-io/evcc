import { test, expect } from "@playwright/test";
import { start, stop, baseUrl } from "./evcc";
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

test.beforeEach(async () => {
  await start(simulatorConfig("./tests/vehicle-detection.evcc.yaml"));
});

test.afterEach(async () => {
  await stop();
});

test("vehicle selected during detection stays selected", async ({ page }) => {
  // vehicle plugs in
  await page.goto(simulatorUrl());
  await page.getByTestId("loadpoint0").getByText("B (connected)").click();
  await simulatorApply(page);

  // detection running
  await page.goto("/");
  await expect(page.getByTestId("vehicle-name")).toHaveText("Guest vehicle");
  await expect(page.getByTestId("vehicle-detection-icon")).toBeVisible();

  // user knows which vehicle it is
  await page.getByRole("combobox", { name: "Change vehicle" }).selectOption("blauer e-Golf");
  await expect(page.getByTestId("vehicle-name")).toHaveText("blauer e-Golf");
  await expect(page.getByTestId("vehicle-detection-icon")).toBeHidden();

  // still the case after a reload
  await page.reload();
  await expect(page.getByTestId("vehicle-name")).toHaveText("blauer e-Golf");
  await expect(page.getByTestId("vehicle-detection-icon")).toBeHidden();
});
