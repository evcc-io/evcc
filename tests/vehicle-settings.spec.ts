import { test, expect, type Locator, type Page } from "@playwright/test";
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

async function switchVehicle(page: Page, title: string) {
  await page.getByRole("combobox", { name: "Change vehicle" }).selectOption(title);
  await expect(page.getByTestId("vehicle-name")).toHaveText(title);
}

async function openVehicleSettings(page: Page) {
  const moreTab = page.getByTestId("tab-more");
  await moreTab.click();
  await moreTab.getByRole("button", { name: "Vehicles" }).click();
  const modal = page.getByTestId("vehicle-settings-modal");
  await expectModalVisible(modal);
  return modal;
}

function vehicleRow(modal: Locator, title: string) {
  return modal.getByRole("group", { name: title });
}

async function closeModal(page: Page) {
  const modal = page.getByTestId("vehicle-settings-modal");
  await modal.getByRole("button", { name: "Close" }).click();
  await expectModalHidden(modal);
}

test.describe("overview", async () => {
  test("all configured vehicles listed", async ({ page }) => {
    await page.goto("/");

    const modal = await openVehicleSettings(page);
    await expect(modal.getByRole("group")).toHaveCount(2);
    await expect(vehicleRow(modal, "blauer e-Golf")).toBeVisible();
    await expect(vehicleRow(modal, "grüner Honda e")).toBeVisible();
    await expect(modal).not.toContainText("Guest vehicle");
  });

  test("connected indicator only on connected vehicle", async ({ page }) => {
    await page.goto("/");

    const modal = await openVehicleSettings(page);
    await expect(
      vehicleRow(modal, "blauer e-Golf").getByText("connected to Carport")
    ).toBeVisible();
    await expect(vehicleRow(modal, "grüner Honda e").getByText("not connected")).toBeVisible();
  });

  test("per-vehicle values are independent", async ({ page }) => {
    await page.goto("/");

    const modal = await openVehicleSettings(page);
    const golfMode = vehicleRow(modal, "blauer e-Golf").getByRole("combobox", {
      name: "Default mode",
    });
    const hondaMode = vehicleRow(modal, "grüner Honda e").getByRole("combobox", {
      name: "Default mode",
    });

    await hondaMode.selectOption("Fast");
    await expect(hondaMode).toHaveValue("now");
    await expect(golfMode).toHaveValue("");
  });
});

test.describe("minSoc", async () => {
  test("apply and restart", async ({ page }) => {
    await page.goto("/");

    const modal = await openVehicleSettings(page);
    const minSoc = vehicleRow(modal, "blauer e-Golf").getByRole("combobox", {
      name: "Minimum charge",
    });
    await minSoc.selectOption("20");
    await expect(minSoc).toHaveValue("20");
    await closeModal(page);
    await page.waitForLoadState("networkidle");

    await restart(simulatorConfig());
    await page.reload();

    const modalAfterRestart = await openVehicleSettings(page);
    await expect(
      vehicleRow(modalAfterRestart, "blauer e-Golf").getByRole("combobox", {
        name: "Minimum charge",
      })
    ).toHaveValue("20");
  });

  test("show minsoc indicator when minsoc is active", async ({ page }) => {
    await page.goto("/");

    const modal = await openVehicleSettings(page);
    const minSoc = vehicleRow(modal, "blauer e-Golf").getByRole("combobox", {
      name: "Minimum charge",
    });
    await minSoc.selectOption("50");
    await closeModal(page);

    await expect(page.getByTestId("vehicle-status-minsoc")).toBeVisible();
    await expect(page.getByTestId("vehicle-status-minsoc")).toHaveText("50%");

    await page.getByTestId("vehicle-status-minsoc").click();
    await expectModalVisible(modal);
    await minSoc.selectOption("0");
    await closeModal(page);

    await expect(page.getByTestId("vehicle-status-minsoc")).not.toBeVisible();
  });
});

test.describe("limitSoc", async () => {
  test("apply and restart", async ({ page }) => {
    await page.goto("/");

    const modal = await openVehicleSettings(page);
    await vehicleRow(modal, "blauer e-Golf")
      .getByRole("combobox", { name: "Default limit" })
      .selectOption("80");
    await closeModal(page);
    await expect(page.getByTestId("limit-soc-value")).toContainText("80%");
    await page.waitForLoadState("networkidle");

    await restart(simulatorConfig());
    await page.reload();

    await expect(page.getByTestId("limit-soc-value")).toContainText("80%");

    const modalAfterRestart = await openVehicleSettings(page);
    await expect(
      vehicleRow(modalAfterRestart, "blauer e-Golf").getByRole("combobox", {
        name: "Default limit",
      })
    ).toHaveValue("80");
  });
});

test.describe("minSoc and limitSoc", async () => {
  test("hidden for offline vehicles", async ({ page }) => {
    await page.goto("/");

    const modal = await openVehicleSettings(page);
    const honda = vehicleRow(modal, "grüner Honda e");
    await expect(honda.getByRole("combobox", { name: "Default mode" })).toBeVisible();
    await expect(honda.getByRole("combobox", { name: "Minimum charge" })).not.toBeVisible();
    await expect(honda.getByRole("combobox", { name: "Default limit" })).not.toBeVisible();
  });
});

test.describe("default mode", async () => {
  async function setDefaultMode(page: Page, title: string, option: string) {
    const modal = await openVehicleSettings(page);
    await vehicleRow(modal, title)
      .getByRole("combobox", { name: "Default mode" })
      .selectOption(option);
    await closeModal(page);
  }

  test("apply and restart", async ({ page }) => {
    await page.goto("/");

    await setDefaultMode(page, "blauer e-Golf", "Fast");
    await page.waitForLoadState("networkidle");

    await restart(simulatorConfig());
    await page.reload();

    const modal = await openVehicleSettings(page);
    await expect(
      vehicleRow(modal, "blauer e-Golf").getByRole("combobox", { name: "Default mode" })
    ).toHaveValue("now");
  });

  test("applied on vehicle switch", async ({ page }) => {
    await page.goto("/");
    const mode = page.getByTestId("loadpoint").first().getByTestId("mode");

    // per-vehicle default modes
    await setDefaultMode(page, "blauer e-Golf", "Solar");
    await setDefaultMode(page, "grüner Honda e", "Fast");

    // switching vehicles applies the vehicle's default mode
    await switchVehicle(page, "grüner Honda e");
    await expect(mode.getByRole("button", { name: "Fast" })).toHaveClass(/active/);

    await switchVehicle(page, "blauer e-Golf");
    await expect(mode.getByRole("button", { name: "Solar", exact: true })).toHaveClass(/active/);
  });
});

test.describe("menu entry", async () => {
  test("hidden when no vehicles are configured", async ({ page }) => {
    await stop();
    await start("basics.evcc.yaml");

    await page.goto("/");
    const moreTab = page.getByTestId("tab-more");
    await moreTab.click();
    await expect(moreTab.getByRole("button", { name: "User Interface" })).toBeVisible();
    await expect(moreTab.getByRole("button", { name: "Vehicles" })).not.toBeVisible();
  });
});
