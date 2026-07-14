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

function changeVehicleSelect(page: Page) {
  return page.getByRole("combobox", { name: "Change vehicle" });
}

async function switchVehicle(page: Page, title: string) {
  await changeVehicleSelect(page).selectOption(title);
  await expect(page.getByTestId("vehicle-name")).toHaveText(title);
}

async function openVehicleSettings(page: Page) {
  await changeVehicleSelect(page).selectOption({ label: "→ Settings" });
  const modal = page.getByTestId("vehicle-settings-modal");
  await expectModalVisible(modal);
  return modal;
}

async function closeModal(page: Page) {
  const modal = page.getByTestId("vehicle-settings-modal");
  await modal.getByRole("button", { name: "Close" }).click();
  await expectModalHidden(modal);
}

test.describe("minSoc", async () => {
  test("apply and restart", async ({ page }) => {
    await page.goto("/");

    const modal = await openVehicleSettings(page);
    await expect(modal).toContainText("charged to x in solar mode");
    await modal.getByRole("combobox", { name: "Minimum charge" }).selectOption("20%");
    await expect(modal).toContainText("charged to 20% in solar mode");
    await closeModal(page);
    await page.waitForLoadState("networkidle");

    await restart(simulatorConfig());
    await page.reload();

    await openVehicleSettings(page);
    await expect(page.getByText("charged to 20% in solar mode")).toBeVisible();
  });

  test("show minsoc indicator when minsoc is active", async ({ page }) => {
    await page.goto("/");

    const modal = await openVehicleSettings(page);
    await modal.getByRole("combobox", { name: "Minimum charge" }).selectOption("50%");
    await closeModal(page);

    await expect(page.getByTestId("vehicle-status-minsoc")).toBeVisible();
    await expect(page.getByTestId("vehicle-status-minsoc")).toHaveText("50%");

    await page.getByTestId("vehicle-status-minsoc").click();
    await expectModalVisible(modal);
    await modal.getByRole("combobox", { name: "Minimum charge" }).selectOption("---");
    await closeModal(page);

    await expect(page.getByTestId("vehicle-status-minsoc")).not.toBeVisible();
  });
});

test.describe("limitSoc", async () => {
  test("apply and restart", async ({ page }) => {
    await page.goto("/");

    const modal = await openVehicleSettings(page);
    await modal.getByRole("combobox", { name: "Default limit" }).selectOption("80%");
    await closeModal(page);
    await expect(page.getByTestId("limit-soc-value")).toContainText("80%");
    await page.waitForLoadState("networkidle");
    await page.waitForTimeout(500); // bad practice but may help here :/

    await restart(simulatorConfig());
    await page.reload();

    await expect(page.getByTestId("limit-soc-value")).toContainText("80%");

    const modalAfterRestart = await openVehicleSettings(page);
    await expect(modalAfterRestart.getByRole("combobox", { name: "Default limit" })).toHaveValue(
      "80"
    );
  });
});

test.describe("minSoc and limitSoc", async () => {
  test("hidden for offline vehicles", async ({ page }) => {
    await page.goto("/");

    await switchVehicle(page, "grüner Honda e");

    const modal = await openVehicleSettings(page);
    await expect(modal.getByRole("combobox", { name: "Default mode" })).toBeVisible();
    await expect(modal.getByRole("combobox", { name: "Minimum charge" })).not.toBeVisible();
    await expect(modal.getByRole("combobox", { name: "Default limit" })).not.toBeVisible();
  });

  test("no settings entry for guest vehicles", async ({ page }) => {
    await page.goto("/");

    // switch to guest vehicle (no known vehicle)
    const select = changeVehicleSelect(page);
    await select.selectOption("Guest vehicle");

    await expect(select.locator("option", { hasText: "Settings" })).toHaveCount(0);
  });
});

test.describe("vehicle switch in modal", async () => {
  test("switch context and keep per-vehicle values", async ({ page }) => {
    await page.goto("/");

    const modal = await openVehicleSettings(page);
    const switcher = modal.getByRole("combobox", { name: "Change vehicle" });
    const defaultMode = modal.getByRole("combobox", { name: "Default mode" });

    await expect(switcher.locator("option", { hasText: "Guest vehicle" })).toHaveCount(0);
    await expect(switcher.locator("option", { hasText: "Settings" })).toHaveCount(0);

    await switcher.selectOption("grüner Honda e");
    await defaultMode.selectOption("Fast");

    await switcher.selectOption("blauer e-Golf");
    await expect(defaultMode).toHaveValue("");

    await switcher.selectOption("grüner Honda e");
    await expect(defaultMode).toHaveValue("now");
  });
});

test.describe("default mode", async () => {
  async function setDefaultMode(page: Page, option: string) {
    const modal = await openVehicleSettings(page);
    await modal.getByRole("combobox", { name: "Default mode" }).selectOption(option);
    await closeModal(page);
  }

  test("apply and restart", async ({ page }) => {
    await page.goto("/");

    await setDefaultMode(page, "Fast");
    await page.waitForLoadState("networkidle");

    await restart(simulatorConfig());
    await page.reload();

    const modal = await openVehicleSettings(page);
    await expect(modal.getByRole("combobox", { name: "Default mode" })).toHaveValue("now");
  });

  test("applied on vehicle switch", async ({ page }) => {
    await page.goto("/");
    const mode = page.getByTestId("loadpoint").first().getByTestId("mode");

    // per-vehicle default modes
    await setDefaultMode(page, "Solar");
    await switchVehicle(page, "grüner Honda e");
    await setDefaultMode(page, "Fast");

    // switching vehicles applies the vehicle's default mode
    await switchVehicle(page, "blauer e-Golf");
    await expect(mode.getByRole("button", { name: "Solar", exact: true })).toHaveClass(/active/);

    await switchVehicle(page, "grüner Honda e");
    await expect(mode.getByRole("button", { name: "Fast" })).toHaveClass(/active/);
  });
});
