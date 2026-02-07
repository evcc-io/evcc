import { test, expect } from "@playwright/test";
import { start, stop, restart, baseUrl } from "./evcc";
import {
  expectModalVisible,
  expectModalHidden,
  editorClear,
  editorPaste,
  addDemoCharger,
  newLoadpoint,
} from "./utils";

test.use({ baseURL: baseUrl() });
test.describe.configure({ mode: "parallel" });

test.afterEach(async () => {
  await stop();
});

test.describe("invalid references", async () => {
  test("circuit", async ({ page }) => {
    await start();
    await page.goto("/#/config");

    // Create circuit via UI
    await page.getByTestId("circuits").getByRole("button", { name: "edit" }).click();
    const circuitsModal = page.getByTestId("circuits-modal");
    await expectModalVisible(circuitsModal);

    const circuitEditor = circuitsModal.getByTestId("yaml-editor");
    await editorClear(circuitEditor);
    await editorPaste(
      circuitEditor,
      page,
      `- name: main
  title: Main`
    );

    await circuitsModal.getByRole("button", { name: "Save" }).click();
    await expectModalHidden(circuitsModal);

    // Restart
    await restart();

    // Create loadpoint with demo charger
    const lpModal = page.getByTestId("loadpoint-modal");
    await newLoadpoint(page, "Test Carport");
    await addDemoCharger(page);

    // Wait for circuit field to be available and assign to circuit main
    await expect(lpModal.getByLabel("Circuit")).toBeVisible();
    await lpModal.getByLabel("Circuit").selectOption("Main [main]");
    await lpModal.getByRole("button", { name: "Save" }).click();
    await expectModalHidden(lpModal);

    // Edit circuit and rename "main" to "main2"
    await page.getByTestId("circuits").getByRole("button", { name: "edit" }).click();
    await expectModalVisible(circuitsModal);
    await editorClear(circuitEditor);
    await editorPaste(
      circuitEditor,
      page,
      `- name: main2
  title: Main`
    );
    await circuitsModal.getByRole("button", { name: "Save" }).click();
    await expectModalHidden(circuitsModal);

    // Save and restart
    await restart();

    // Check boot error
    await expect(page.getByTestId("fatal-error")).toBeVisible();
    await expect(page.getByTestId("fatal-error")).toContainText("circuit: not found: main");

    // Verify loadpoint tile has error class
    const loadpointTile = page.getByTestId("loadpoint");
    await expect(loadpointTile).toBeVisible();
    await expect(loadpointTile).toHaveClass(/round-box--error/);

    // Edit loadpoint
    await loadpointTile.getByRole("button", { name: "edit" }).click();
    await expectModalVisible(lpModal);

    // Verify circuit select is hidden
    await expect(lpModal.getByLabel("Circuit")).not.toBeVisible();

    // Verify invalid-reference-alert with correct text is visible
    const alert = lpModal.getByTestId("invalid-reference-alert");
    await expect(alert).toBeVisible();
    await expect(alert).toContainText("Circuit does not exist: main");

    // Click remove button
    await alert.getByRole("link", { name: "Remove" }).click();

    // Verify the circuit select is now available again
    await expect(lpModal.getByLabel("Circuit")).toBeVisible();
    await expect(alert).not.toBeVisible();

    // Save and restart
    await lpModal.getByRole("button", { name: "Save" }).click();
    await expectModalHidden(lpModal);

    await restart();

    // Verify no error
    await expect(page.getByTestId("fatal-error")).not.toBeVisible();
    await expect(loadpointTile).not.toHaveClass(/round-box--error/);
    await expect(loadpointTile).toContainText("Test Carport");
  });

  test("vehicle", async ({ page }) => {
    // Start with YAML file containing one vehicle
    await start("config-invalid-references-vehicle.evcc.yaml");
    await page.goto("/#/config");

    const lpModal = page.getByTestId("loadpoint-modal");

    // Create loadpoint with demo charger and assign vehicle
    await newLoadpoint(page, "Garage");
    await addDemoCharger(page);
    await expect(lpModal.getByLabel("Default vehicle")).toBeVisible();
    await lpModal.getByLabel("Default vehicle").selectOption("Legacy Vehicle");
    await lpModal.getByRole("button", { name: "Save" }).click();
    await expectModalHidden(lpModal);

    // Restart without YAML file (simulating user changed it)
    await restart();

    // Verify fatal error on boot
    await expect(page.getByTestId("fatal-error")).toBeVisible();
    await expect(page.getByTestId("fatal-error")).toContainText("vehicle: not found: car");

    // Verify loadpoint has error class
    const loadpointTile = page.getByTestId("loadpoint");
    await expect(loadpointTile).toBeVisible();
    await expect(loadpointTile).toHaveClass(/round-box--error/);

    // Open loadpoint modal and verify invalid reference alert
    await loadpointTile.getByRole("button", { name: "edit" }).click();
    await expectModalVisible(lpModal);

    const alert = lpModal.getByTestId("invalid-reference-alert");
    await expect(alert).toBeVisible();
    await expect(alert).toContainText("Vehicle does not exist: car");

    // Remove vehicle reference
    await alert.getByRole("link", { name: "Remove" }).click();
    await expect(alert).not.toBeVisible();

    // Verify "no vehicles" message is shown
    await expect(lpModal).toContainText("No vehicles are configured.");

    // Save and restart
    await lpModal.getByRole("button", { name: "Save" }).click();
    await expectModalHidden(lpModal);
    await restart();

    // Verify no fatal error and no error class
    await expect(page.getByTestId("fatal-error")).not.toBeVisible();
    await expect(loadpointTile).not.toHaveClass(/round-box--error/);
  });
});
