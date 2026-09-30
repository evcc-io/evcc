import os from "os";
import path from "path";
import fs from "fs";
import { test, expect, type Page } from "@playwright/test";
import { start, stop, baseUrl } from "./evcc";
import {
  startSimulator,
  stopSimulator,
  simulatorHost,
  simulatorUrl,
  simulatorApply,
} from "./simulator";

test.use({ baseURL: baseUrl() });

// simulator config without a default vehicle, so connecting starts detection
function detectionConfig() {
  const content = fs.readFileSync("./tests/vehicle-detection.evcc.yaml", "utf8");
  const result = content.replace(/localhost:7072/g, simulatorHost());
  const resultPath = path.join(
    os.tmpdir(),
    `vehicle-detection-${simulatorHost().split(":")[1]}.evcc.generated.yaml`
  );
  fs.writeFileSync(resultPath, result);
  return resultPath;
}

async function loadpointState(page: Page) {
  const res = await page.request.get("/api/state?jq=" + encodeURIComponent(".loadpoints[0]"));
  return (await res.json()) as { vehicleName: string; vehicleDetectionActive: boolean };
}

test.beforeAll(async () => {
  await startSimulator();
});
test.afterAll(async () => {
  await stopSimulator();
});

test.beforeEach(async ({ page }) => {
  await start(detectionConfig());

  await page.goto(simulatorUrl());
  await page.getByTestId("loadpoint0").getByText("B (connected)").click();
  await simulatorApply(page);
});

test.afterEach(async () => {
  await stop();
});

test.describe("vehicle detection", async () => {
  for (const title of ["grüner Honda e", "blauer e-Golf"]) {
    test(`selecting ${title} during detection sticks`, async ({ page }) => {
      await page.goto("/");

      const vehicleTitle = page.getByTestId("vehicle-title");
      const name = page.getByTestId("vehicle-name");
      const spinner = vehicleTitle.locator(".spin");

      // detection running: guest vehicle, spinner
      await expect(name).toHaveText("Guest vehicle");
      await expect(spinner).toBeVisible();
      await expect.poll(async () => (await loadpointState(page)).vehicleDetectionActive).toBe(true);

      // user picks the vehicle
      await page.getByRole("combobox", { name: "Change vehicle" }).selectOption(title);
      await expect(name).toHaveText(title);
      await expect(spinner).toBeHidden();

      // selection must survive several update cycles
      await page.waitForTimeout(3000);
      await expect(name).toHaveText(title);
      await expect(spinner).toBeHidden();
      const state = await loadpointState(page);
      expect(state.vehicleDetectionActive).toBe(false);
      expect(state.vehicleName).toBe(title === "blauer e-Golf" ? "golf" : "honda");
    });
  }
});
