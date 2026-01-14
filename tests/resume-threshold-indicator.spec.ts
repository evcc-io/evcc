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

test.beforeEach(async ({ page }) => {
  await start(simulatorConfig());

  await page.goto(simulatorUrl());
  await page.getByLabel("Grid Power").fill("500");
  await page.getByTestId("vehicle0").getByLabel("SoC").fill("80");
  await page.getByTestId("loadpoint0").getByText("B (connected)").click();
  await simulatorApply(page);

  await page.goto("/");
  await setResumeThreshold(page, "5%");
  await setLimitSoc(page, "50%");
});

test.afterEach(async () => {
  await stop();
});

const setResumeThreshold = async (page: any, threshold: string) => {
  await page.getByTestId("charging-plan").getByRole("button", { name: "none" }).click();
  const modal = page.getByTestId("charging-plan-modal");
  await modal.getByRole("link", { name: "Arrival" }).click();
  await modal.getByRole("combobox", { name: "Resume threshold" }).selectOption(threshold);
  await modal.getByRole("button", { name: "Close" }).click();
};

const setLimitSoc = async (page: any, limit: string) => {
  await page.getByTestId("limit-soc").getByRole("combobox").selectOption(limit);
};

const expectMarkerVisible = async (page: any) => {
  await expect(
    page.locator(".resume-threshold-marker.resume-threshold-marker--visible")
  ).toBeVisible();
};

const expectMarkerNotVisible = async (page: any) => {
  await expect(
    page.locator(".resume-threshold-marker.resume-threshold-marker--visible")
  ).not.toBeVisible();
};

test.describe("resumeThreshold marker", async () => {
  test("visible when resumeThreshold is set", async ({ page }) => {
    // Set to Fast mode so resumeThreshold marker is visible
    await page.getByTestId("mode").first().getByRole("button", { name: "Fast" }).click();

    await expectMarkerVisible(page);
  });

  test("not visible in Off mode", async ({ page }) => {
    // Set to Off mode
    await page.getByTestId("mode").first().getByRole("button", { name: "Off" }).click();

    await expectMarkerNotVisible(page);
  });

  test("not visible in Solar mode", async ({ page }) => {
    // Set to Solar mode
    await page.getByTestId("mode").first().getByRole("button", { name: "Solar", exact: true }).click();

    await expectMarkerNotVisible(page);
  });

  test("visible in Min+Solar mode", async ({ page }) => {
    // Set to Min+Solar mode
    await page.getByTestId("mode").first().getByRole("button", { name: "Min+Solar" }).click();

    await expectMarkerVisible(page);
  });

  test("not visible when charging and limit > vehicle soc", async ({ page }) => {
    // Set to Fast mode
    await page.getByTestId("mode").first().getByRole("button", { name: "Fast" }).click();

    // Override limit to 90% (higher than vehicle soc of 80%)
    await setLimitSoc(page, "90%");

    // Set status to charging (C) in simulator
    await page.goto(simulatorUrl());
    await page.getByTestId("loadpoint0").getByText("C (charging)").click();
    await simulatorApply(page);

    await page.goto("/");
    await expectMarkerNotVisible(page);
  });

  test("not visible when active plan is set", async ({ page }) => {
    // Set to Fast mode
    await page.getByTestId("mode").first().getByRole("button", { name: "Fast" }).click();

    // Create and activate a plan with target time in 5 minutes
    await page.getByTestId("charging-plan").getByRole("button").click();
    const modal = page.getByTestId("charging-plan-modal");
    
    // Set date to today
    await modal.getByTestId("static-plan-day").selectOption({ index: 0 });
    
    // Set target time to now + 5 minutes
    const now = new Date();
    now.setMinutes(now.getMinutes() + 5);
    const hours = now.getHours().toString().padStart(2, "0");
    const minutes = now.getMinutes().toString().padStart(2, "0");
    await modal.getByTestId("static-plan-time").fill(`${hours}:${minutes}`);
    
    // Activate the plan
    await modal.getByTestId("static-plan-active").click();
    await modal.getByRole("button", { name: "Close" }).click();

    // Marker should not be visible when a plan is active
    await expectMarkerNotVisible(page);
  });


  test("not visible when resumeThreshold is set to ---", async ({ page }) => {
    // Set to Fast mode
    await page.getByTestId("mode").first().getByRole("button", { name: "Fast" }).click();

    // Set resume threshold to "---" (disabled)
    await setResumeThreshold(page, "---");

    // Marker should not be visible when resume threshold is set to ---
    await expectMarkerNotVisible(page);
  });
});
