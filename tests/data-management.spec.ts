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
    await enableExperimental(page, false);
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
    await restart();
    await expect(page.getByTestId("offline-indicator")).toHaveAttribute("aria-hidden", "true");

    // verify sessions deleted
    await page.goto("/#/sessions?year=2023&month=5");
    await expect(page.getByTestId("sessions-entry")).toHaveCount(0);
  });
});
