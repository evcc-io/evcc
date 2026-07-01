import { test, expect, type Page } from "@playwright/test";
import { start, stop, restart, baseUrl } from "./evcc";
import { expectModalVisible, expectModalHidden } from "./utils";
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
  await page.getByTestId("vehicle0").getByLabel("SoC").fill("20");
  await page.getByTestId("loadpoint0").getByText("B (connected)").click();
  await simulatorApply(page);
});

test.afterEach(async () => {
  await stop();
});

test.describe("minSoc", async () => {
  test("apply and restart", async ({ page }) => {
    await page.goto("/");

    await page.getByTestId("charging-plan").getByRole("button", { name: "none" }).click();
    const modal = page.getByTestId("charging-plan-modal");
    await expectModalVisible(modal);
    await modal.getByRole("link", { name: "Arrival" }).click();

    await expect(modal).toContainText("charged to x in solar mode");
    await modal.getByRole("combobox", { name: "Min. charge %" }).selectOption("20%");
    await expect(modal).toContainText("charged to 20% in solar mode");
    await modal.getByRole("button", { name: "Close" }).click();
    await expectModalHidden(modal);
    await page.waitForLoadState("networkidle");

    await restart(simulatorConfig());
    await page.reload();

    await page.getByTestId("charging-plan").getByRole("button", { name: "none" }).click();
    await page.getByRole("link", { name: "Arrival" }).click();
    await expect(page.getByText("charged to 20% in solar mode")).toBeVisible();
  });

  test("show minsoc indicator when minsoc is active", async ({ page }) => {
    await page.goto("/");

    await expect(page.getByTestId("charging-plan")).toContainText("Plan");
    await page.getByTestId("charging-plan").getByRole("button", { name: "none" }).click();
    await page.getByRole("link", { name: "Arrival" }).click();
    await page.getByRole("combobox", { name: "Min. charge %" }).selectOption("50%");
    await page.getByRole("button", { name: "Close" }).click();

    await expect(page.getByTestId("vehicle-status-minsoc")).toBeVisible();
    await expect(page.getByTestId("vehicle-status-minsoc")).toHaveText("50%");

    await page.getByTestId("vehicle-status-minsoc").click();
    await page.getByRole("combobox", { name: "Min. charge %" }).selectOption("---");
    await page.getByRole("button", { name: "Close" }).click();

    await expect(page.getByTestId("vehicle-status-minsoc")).not.toBeVisible();
  });
});

test.describe("limitSoc", async () => {
  test("apply and restart", async ({ page }) => {
    await page.goto("/");

    await page.getByTestId("charging-plan").getByRole("button", { name: "none" }).click();
    const modal = page.getByTestId("charging-plan-modal");
    await expectModalVisible(modal);
    await modal.getByRole("link", { name: "Arrival" }).click();

    await modal.getByRole("combobox", { name: "Default limit" }).selectOption("80%");
    await modal.getByRole("button", { name: "Close" }).click();
    await expectModalHidden(modal);
    await expect(page.getByTestId("limit-soc-value")).toContainText("80%");
    await page.waitForLoadState("networkidle");
    await page.waitForTimeout(500); // bad practice but may help here :/

    await restart(simulatorConfig());
    await page.reload();

    await expect(page.getByTestId("limit-soc-value")).toContainText("80%");

    await page.getByTestId("charging-plan").getByRole("button", { name: "none" }).click();
    await page.getByRole("link", { name: "Arrival" }).click();
    await expect(page.getByRole("combobox", { name: "Default limit" })).toHaveValue("80");
  });
});

test.describe("minSoc and limitSoc", async () => {
  test("hidden for offline vehicles", async ({ page }) => {
    await page.goto("/");

    // switch to offline vehicle
    await page.getByTestId("change-vehicle").locator("select").selectOption("grüner Honda e");

    await page.getByTestId("charging-plan").getByRole("button", { name: "none" }).click();
    await page.getByRole("link", { name: "Arrival" }).click();
    await expect(page.getByRole("combobox", { name: "Min. charge %" })).not.toBeVisible();
    await expect(page.getByRole("combobox", { name: "Default limit" })).not.toBeVisible();
  });

  test("no tabs for guest vehicles", async ({ page }) => {
    await page.goto("/");

    // switch to guest vehicle (no known vehicle)
    await page.getByTestId("change-vehicle").locator("select").selectOption("Guest vehicle");

    await page.getByTestId("charging-plan").getByRole("button", { name: "none" }).click();
    const modal = page.getByTestId("charging-plan-modal");
    await expectModalVisible(modal);
    await expect(modal.getByRole("link", { name: "Departure" })).toHaveCount(0);
    await expect(modal.getByRole("link", { name: "Arrival" })).toHaveCount(0);
  });
});

test.describe("default mode", async () => {
  async function setDefaultMode(page: Page, option: string) {
    await page.getByTestId("charging-plan").getByRole("button", { name: "none" }).click();
    const modal = page.getByTestId("charging-plan-modal");
    await expectModalVisible(modal);
    await modal.getByRole("link", { name: "Arrival" }).click();
    await modal.getByRole("combobox", { name: "Default mode" }).selectOption(option);
    await modal.getByRole("button", { name: "Close" }).click();
    await expectModalHidden(modal);
  }

  test("apply and restart", async ({ page }) => {
    await page.goto("/");

    await setDefaultMode(page, "Fast");
    await page.waitForLoadState("networkidle");

    await restart(simulatorConfig());
    await page.reload();

    await page.getByTestId("charging-plan").getByRole("button", { name: "none" }).click();
    await page.getByRole("link", { name: "Arrival" }).click();
    await expect(page.getByRole("combobox", { name: "Default mode" })).toHaveValue("now");
  });

  test("applied on vehicle switch", async ({ page }) => {
    await page.goto("/");
    const mode = page.getByTestId("loadpoint").first().getByTestId("mode");
    const changeVehicle = page.getByTestId("change-vehicle").locator("select");

    // per-vehicle default modes
    await setDefaultMode(page, "Solar");
    await changeVehicle.selectOption("grüner Honda e");
    await setDefaultMode(page, "Fast");

    // switching vehicles applies the vehicle's default mode
    await changeVehicle.selectOption("blauer e-Golf");
    await expect(mode.getByRole("button", { name: "Solar", exact: true })).toHaveClass(/active/);

    await changeVehicle.selectOption("grüner Honda e");
    await expect(mode.getByRole("button", { name: "Fast" })).toHaveClass(/active/);
  });
});
