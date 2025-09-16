import { test, expect } from "@playwright/test";
import { start, stop, baseUrl, restart } from "./evcc";
import {
  enableExperimental,
  openTopNavigation,
  expectModalVisible,
  expectModalHidden,
} from "./utils";
import fs from "fs";
import path from "path";

test.use({ baseURL: baseUrl() });

test.describe("reset", async () => {
  test("reset sessions", async ({ page }) => {
    await start(undefined, "sessions.sql");

    // check sessions
    await page.goto("/#/sessions?year=2023&month=5");
    await expect(page.getByTestId("sessions-entry")).toHaveCount(4);

    // open backup & restore modal
    await openTopNavigation(page);
    await page.getByRole("link", { name: "Configuration" }).click();
    await enableExperimental(page);
    await page.getByRole("button", { name: "Backup & Restore" }).click();
    const modal = page.getByTestId("backup-restore-modal");
    await expectModalVisible(modal);

    // reset
    await expect(modal.getByRole("button", { name: "Reset..." })).toBeDisabled();
    await modal.getByRole("checkbox", { name: "Charging sessions" }).check();
    await modal.getByRole("button", { name: "Reset..." }).click();
    const confirmModal = page.getByTestId("backup-restore-confirm-modal");
    await expectModalVisible(confirmModal);
    await expect(confirmModal.getByLabel("Administrator Password")).not.toBeVisible(); // disable auth mode
    await confirmModal.getByRole("button", { name: "Reset & restart" }).click();
    await expectModalHidden(confirmModal);
    await expectModalHidden(modal);
    await expect(page.getByTestId("offline-indicator")).toHaveAttribute("aria-hidden", "false");

    // manual restart
    await restart(undefined, undefined, true);
    await expect(page.getByTestId("offline-indicator")).toHaveAttribute("aria-hidden", "true");

    // verify sessions deleted
    await page.goto("/#/sessions?year=2023&month=5");
    await expect(page.getByTestId("sessions-entry")).toHaveCount(0);
    await stop();
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
    await page.getByRole("button", { name: "Backup & Restore" }).click();
    const modal = page.getByTestId("backup-restore-modal");
    await expectModalVisible(modal);
    await expect(modal.getByRole("button", { name: "Reset..." })).toBeDisabled();
    await modal.getByRole("checkbox", { name: "Settings" }).check();
    await modal.getByRole("button", { name: "Reset..." }).click();
    const confirmModal = page.getByTestId("backup-restore-confirm-modal");
    await expectModalVisible(confirmModal);
    await expect(confirmModal.getByLabel("Administrator Password")).not.toBeVisible(); // disable auth mode
    await confirmModal.getByRole("button", { name: "Reset & restart" }).click();
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
    await stop();
  });
});

test.describe("backup and restore", async () => {
  test("download backup and restore from file", async ({ page }) => {
    const initialTitle = "My Home Base";
    const changedTitle = "Changed Title";

    await start();
    await page.goto("/#/config");
    await enableExperimental(page);

    // set initial title
    await page.getByTestId("generalconfig-title").getByRole("button", { name: "edit" }).click();
    const titleModal = page.getByTestId("title-modal");
    await expectModalVisible(titleModal);
    await titleModal.getByLabel("Title").fill(initialTitle);
    await titleModal.getByRole("button", { name: "Save" }).click();
    await expectModalHidden(titleModal);

    // create grid meter
    await page.getByTestId("add-grid").click();
    const meterModal = page.getByTestId("meter-modal");
    await expectModalVisible(meterModal);
    await meterModal.getByLabel("Manufacturer").selectOption("Demo meter");
    await meterModal.getByLabel("Power").fill("2000");
    await meterModal.getByRole("button", { name: "Save" }).click();
    await expectModalHidden(meterModal);

    // verify initial state
    await expect(page.getByTestId("grid")).toBeVisible();
    await expect(page.getByTestId("generalconfig-title")).toContainText(initialTitle);

    // open backup & restore modal
    await page.getByRole("button", { name: "Backup & Restore" }).click();
    const backupModal = page.getByTestId("backup-restore-modal");
    await expectModalVisible(backupModal);

    // download backup
    const downloadPromise = page.waitForEvent("download");
    await backupModal.getByRole("button", { name: "Download backup..." }).click();

    // download backup confirm
    const backupConfirmModal = page.getByTestId("backup-restore-confirm-modal");
    await expectModalVisible(backupConfirmModal);
    await backupConfirmModal.getByRole("button", { name: "Download backup" }).click();
    await expectModalHidden(backupConfirmModal);
    const download = await downloadPromise;
    await expectModalVisible(backupModal);
    await backupModal.locator(".btn-close").click();
    await expectModalHidden(backupModal);

    // change title and delete meter
    await page.getByTestId("generalconfig-title").getByRole("button", { name: "edit" }).click();
    await expectModalVisible(titleModal);
    await titleModal.getByLabel("Title").fill(changedTitle);
    await titleModal.getByRole("button", { name: "Save" }).click();
    await expectModalHidden(titleModal);
    await page.getByTestId("grid").getByRole("button", { name: "edit" }).click();
    const gridModal = page.getByTestId("meter-modal");
    await expectModalVisible(gridModal);
    await gridModal.getByRole("button", { name: "Delete" }).click();
    await expectModalHidden(gridModal);

    // verify changes
    await expect(page.getByTestId("grid")).not.toBeVisible();
    await expect(page.getByTestId("generalconfig-title")).toContainText(changedTitle);

    // restore
    await page.getByRole("button", { name: "Backup & Restore" }).click();
    await expectModalVisible(backupModal);

    // prepare backup file
    const downloadPath = await download.path();
    const dbPath = path.join(path.dirname(downloadPath), "backup.db");
    fs.copyFileSync(downloadPath, dbPath);

    const fileChooserPromise = page.waitForEvent("filechooser");
    await backupModal.getByText("Browse").click();
    const fileChooser = await fileChooserPromise;
    await fileChooser.setFiles(dbPath);
    const restoreButton = backupModal.getByRole("button", { name: "Restore..." });
    await expect(restoreButton).toBeEnabled({ timeout: 3000 });
    await restoreButton.click();

    // confirm restore
    const restoreConfirmModal = page.getByTestId("backup-restore-confirm-modal");
    await expectModalVisible(restoreConfirmModal);
    await restoreConfirmModal.getByRole("button", { name: "Restore & restart" }).click();
    await expectModalHidden(restoreConfirmModal);

    // restart after restore
    await expect(page.getByTestId("offline-indicator")).toHaveAttribute("aria-hidden", "false");
    await restart(undefined, undefined, true);
    await expect(page.getByTestId("offline-indicator")).toHaveAttribute("aria-hidden", "true");
    await page.getByRole("link", { name: "Let's start configuration" }).click();

    // verify initial state
    await expect(page.getByTestId("grid")).toBeVisible();
    await expect(page.getByTestId("generalconfig-title")).toContainText(initialTitle);
    await stop();
  });

  test("backup with authentication", async ({ page }) => {
    // start with auth enabled
    await start(undefined, "password.sql", "");
    await page.goto("/#/config");

    // login to access config
    const loginModal = page.getByTestId("login-modal");
    await expectModalVisible(loginModal);
    await loginModal.getByLabel("Password").fill("secret");
    await loginModal.getByRole("button", { name: "Login" }).click();
    await expectModalHidden(loginModal);

    await enableExperimental(page);

    // open backup & restore modal
    await page.getByRole("button", { name: "Backup & Restore" }).click();
    const backupModal = page.getByTestId("backup-restore-modal");
    await expectModalVisible(backupModal);

    // try wrong password
    await backupModal.getByRole("button", { name: "Download backup..." }).click();
    const backupConfirmModal = page.getByTestId("backup-restore-confirm-modal");
    await expectModalVisible(backupConfirmModal);
    const passwordField = backupConfirmModal.getByLabel("Administrator Password");
    await expect(passwordField).toBeVisible();
    await passwordField.fill("wrongpassword");
    await backupConfirmModal.getByRole("button", { name: "Download backup" }).click();
    await expect(backupConfirmModal.getByText("Password is invalid.")).toBeVisible();
    await passwordField.clear();
    await passwordField.fill("secret");
    const downloadPromise = page.waitForEvent("download");
    await backupConfirmModal.getByRole("button", { name: "Download backup" }).click();
    await expectModalHidden(backupConfirmModal);

    // verify backup was downloaded successfully
    const download = await downloadPromise;
    await expect(download.suggestedFilename()).toContain("evcc-backup");
    await stop();
  });
});
