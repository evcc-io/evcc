import { test, expect } from "@playwright/test";
import { start, stop, baseUrl, restart } from "./evcc";
import {
  enableExperimental,
  openTopNavigation,
  expectModalVisible,
  expectModalHidden,
} from "./utils";

test.use({ baseURL: baseUrl() });

test.afterEach(async () => {
  await stop();
});

test.describe("reset", async () => {
  test("reset sessions", async ({ page }) => {
    await start(undefined, "sessions.sql");

    // check sessions
    await page.goto("/#/sessions?year=2023&month=5");
    await expect(page.getByTestId("sessions-entry")).toHaveCount(4);

    // open data management modal
    await openTopNavigation(page);
    await page.getByRole("link", { name: "Configuration" }).click();
    await enableExperimental(page);
    await page.getByRole("button", { name: "Data management" }).click();
    const modal = page.getByTestId("data-management-modal");
    await expectModalVisible(modal);

    // reset
    await expect(modal.getByRole("button", { name: "Reset..." })).toBeDisabled();
    await modal.getByRole("checkbox", { name: "Charging sessions" }).check();
    await modal.getByRole("button", { name: "Reset..." }).click();
    const confirmModal = page.getByTestId("data-management-confirm-modal");
    await expectModalVisible(confirmModal);
    await confirmModal.getByRole("button", { name: "Reset" }).click();
    await expectModalHidden(confirmModal);
    await expectModalHidden(modal);
    await expect(page.getByTestId("offline-indicator")).toHaveAttribute("aria-hidden", "false");

    // manual restart
    await restart(undefined, undefined, true);
    await expect(page.getByTestId("offline-indicator")).toHaveAttribute("aria-hidden", "true");

    // verify sessions deleted
    await page.goto("/#/sessions?year=2023&month=5");
    await expect(page.getByTestId("sessions-entry")).toHaveCount(0);
  });

  test("reset settings", async ({ page }) => {
    const title = "Home base";

    await start(undefined, "sessions.sql");

    // create grid meter and title via UI
    await page.goto("/#/config");
    await enableExperimental(page);
    await page.getByTestId("add-grid").click();
    const meterModal = page.getByTestId("meter-modal");
    await expectModalVisible(meterModal);
    await meterModal.getByLabel("Manufacturer").selectOption("Demo meter");
    await meterModal.getByLabel("Power").fill("2000");
    await meterModal.getByRole("button", { name: "Save" }).click();
    await expectModalHidden(meterModal);
    await page.getByTestId("generalconfig-title").getByRole("button", { name: "edit" }).click();
    const titleModal = page.getByTestId("title-modal");
    await expectModalVisible(titleModal);
    await titleModal.getByLabel("Title").fill(title);
    await titleModal.getByRole("button", { name: "Save" }).click();
    await expectModalHidden(titleModal);

    // restart to apply
    await restart();

    // verify changes are present
    await expect(page.getByTestId("offline-indicator")).toHaveAttribute("aria-hidden", "true");
    await expect(page.getByTestId("grid")).toBeVisible();
    await expect(page.getByTestId("generalconfig-title")).toContainText(title);

    // reset settings only
    await page.getByRole("button", { name: "Data management" }).click();
    const modal = page.getByTestId("data-management-modal");
    await expectModalVisible(modal);
    await expect(modal.getByRole("button", { name: "Reset..." })).toBeDisabled();
    await modal.getByRole("checkbox", { name: "Settings" }).check();
    await modal.getByRole("button", { name: "Reset..." }).click();
    const confirmModal = page.getByTestId("data-management-confirm-modal");
    await expectModalVisible(confirmModal);
    await confirmModal.getByRole("button", { name: "Reset" }).click();
    await expectModalHidden(confirmModal);
    await expectModalHidden(modal);
    await expect(page.getByTestId("offline-indicator")).toHaveAttribute("aria-hidden", "false");

    await restart(undefined, undefined, true);

    // verify welcome message
    await expect(page.getByTestId("offline-indicator")).toHaveAttribute("aria-hidden", "true");
    await expect(page.getByRole("heading", { name: "Hello aboard!" })).toBeVisible();

    // verify sessions
    await page.goto("/#/sessions?year=2023&month=5");
    await expect(page.getByTestId("sessions-entry")).toHaveCount(4);

    // verify deleted config and settings
    await page.goto("/#/config");
    await expect(page.getByTestId("grid")).not.toBeVisible();
    await expect(page.getByTestId("generalconfig-title")).not.toContainText(title);
  });
});
