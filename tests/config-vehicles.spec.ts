import { test, expect } from "@playwright/test";
import { start, stop, restart, baseUrl } from "./evcc";
import {
  editorClear,
  editorPaste,
  enableExperimental,
  expectModalHidden,
  expectModalVisible,
} from "./utils";

const CONFIG_WITH_VEHICLE = "config-with-vehicle.evcc.yaml";

test.use({ baseURL: baseUrl() });
test.describe.configure({ mode: "parallel" });

test.afterEach(async () => {
  await stop();
});

const GENERIC_VEHICLE = "Generic vehicle (without API)";

test.describe("vehicles", async () => {
  test("create, edit and delete vehicles", async ({ page }) => {
    await start();

    await page.goto("/#/config");
    await enableExperimental(page, false);

    await expect(page.getByTestId("vehicle")).toHaveCount(0);
    const vehicleModal = page.getByTestId("vehicle-modal");

    // create #1
    await page.getByTestId("add-vehicle").click();
    await expectModalVisible(vehicleModal);
    await vehicleModal.getByLabel("Manufacturer").selectOption(GENERIC_VEHICLE);
    await vehicleModal.getByLabel("Title").fill("Green Car");
    await vehicleModal.getByRole("button", { name: "Validate & save" }).click();
    await expectModalHidden(vehicleModal);

    await expect(page.getByTestId("vehicle")).toHaveCount(1);

    // create #2
    await page.getByTestId("add-vehicle").click();
    await expectModalVisible(vehicleModal);
    await vehicleModal.getByLabel("Manufacturer").selectOption(GENERIC_VEHICLE);
    await vehicleModal.getByLabel("Title").fill("Yellow Van");
    await vehicleModal.getByRole("button", { name: "Validate & save" }).click();
    await expectModalHidden(vehicleModal);

    await expect(page.getByTestId("vehicle")).toHaveCount(2);
    await expect(page.getByTestId("vehicle").nth(0)).toHaveText(/Green Car/);
    await expect(page.getByTestId("vehicle").nth(1)).toHaveText(/Yellow Van/);

    // edit #1
    await page.getByTestId("vehicle").nth(0).getByRole("button", { name: "edit" }).click();
    await expectModalVisible(vehicleModal);
    await expect(vehicleModal.getByLabel("Title")).toHaveValue("Green Car");
    await vehicleModal.getByLabel("Title").fill("Fancy Car");
    await vehicleModal.getByRole("button", { name: "Validate & save" }).click();
    await expectModalHidden(vehicleModal);
    await expect(page.getByTestId("vehicle")).toHaveCount(2);
    await expect(page.getByTestId("vehicle").nth(0)).toHaveText(/Fancy Car/);

    // delete #1
    await page.getByTestId("vehicle").nth(0).getByRole("button", { name: "edit" }).click();
    await expectModalVisible(vehicleModal);
    await vehicleModal.getByRole("button", { name: "Delete" }).click();
    await expectModalHidden(vehicleModal);

    await expect(page.getByTestId("vehicle")).toHaveCount(1);
    await expect(page.getByTestId("vehicle").nth(0)).toHaveText(/Yellow Van/);

    // delete #2
    await page.getByTestId("vehicle").nth(0).getByRole("button", { name: "edit" }).click();
    await expectModalVisible(vehicleModal);
    await vehicleModal.getByRole("button", { name: "Delete" }).click();
    await expectModalHidden(vehicleModal);

    await expect(page.getByTestId("vehicle")).toHaveCount(0);
  });

  test("config should survive restart", async ({ page }) => {
    await start();

    await page.goto("/#/config");
    await enableExperimental(page);

    await expect(page.getByTestId("vehicle")).toHaveCount(0);
    const vehicleModal = page.getByTestId("vehicle-modal");

    // create #1 & #2
    await page.getByTestId("add-vehicle").click();
    await expectModalVisible(vehicleModal);
    await vehicleModal.getByLabel("Manufacturer").selectOption(GENERIC_VEHICLE);
    await vehicleModal.getByLabel("Title").fill("Green Car");
    await vehicleModal.getByRole("button", { name: "Validate & save" }).click();
    await expectModalHidden(vehicleModal);

    await page.getByTestId("add-vehicle").click();
    await expectModalVisible(vehicleModal);
    await vehicleModal.getByLabel("Manufacturer").selectOption(GENERIC_VEHICLE);
    await vehicleModal.getByLabel("Title").fill("Yellow Van");
    await vehicleModal.getByLabel("car").click();
    await vehicleModal.getByLabel("van").check();
    await vehicleModal.getByRole("button", { name: "Validate & save" }).click();
    await expectModalHidden(vehicleModal);

    await expect(page.getByTestId("vehicle")).toHaveCount(2);

    // restart evcc
    await restart();
    await page.reload();

    await expect(page.getByTestId("vehicle")).toHaveCount(2);
    await expect(page.getByTestId("vehicle").nth(0)).toHaveText(/Green Car/);
    await expect(page.getByTestId("vehicle").nth(1)).toHaveText(/Yellow Van/);
  });

  test("mixed config (yaml + db)", async ({ page }) => {
    await start(CONFIG_WITH_VEHICLE);

    await page.goto("/#/config");
    await enableExperimental(page, false);

    await expect(page.getByTestId("vehicle")).toHaveCount(1);
    const vehicleModal = page.getByTestId("vehicle-modal");

    // create #2
    await page.getByTestId("add-vehicle").click();
    await expectModalVisible(vehicleModal);
    await vehicleModal.getByLabel("Manufacturer").selectOption(GENERIC_VEHICLE);
    await vehicleModal.getByLabel("Title").fill("Green Car");
    await vehicleModal.getByRole("button", { name: "Validate & save" }).click();
    await expectModalHidden(vehicleModal);

    await expect(page.getByTestId("vehicle")).toHaveCount(2);
    await expect(page.getByTestId("vehicle").nth(0)).toHaveText(/YAML Bike/);
    await expect(page.getByTestId("vehicle").nth(1)).toHaveText(/Green Car/);
  });

  test("advanced fields", async ({ page }) => {
    await start();

    await page.goto("/#/config");
    await enableExperimental(page);

    await page.getByTestId("add-vehicle").click();
    const vehicleModal = page.getByTestId("vehicle-modal");

    // generic
    await expectModalVisible(vehicleModal);
    await vehicleModal.getByLabel("Manufacturer").selectOption(GENERIC_VEHICLE);
    await expect(vehicleModal.getByLabel("Title")).toBeVisible();
    await expect(vehicleModal.getByLabel("Car")).toBeVisible(); // icon
    await expect(vehicleModal.getByLabel("Battery capacity")).toBeVisible();

    await page.getByRole("button", { name: "Show advanced settings" }).click();
    await expect(vehicleModal.getByLabel("Default mode")).toBeVisible();
    await expect(vehicleModal.getByLabel("Maximum phases")).toBeVisible();
    await expect(vehicleModal.getByLabel("Minimum current")).toBeVisible();
    await expect(vehicleModal.getByLabel("Maximum current")).toBeVisible();
    await expect(vehicleModal.getByLabel("Priority")).toBeVisible();
    await expect(vehicleModal.getByLabel("RFID identifiers")).toBeVisible();

    await page.getByRole("button", { name: "Hide advanced settings" }).click();
    await expect(vehicleModal.getByLabel("Default mode")).not.toBeVisible();

    // polestar template
    await vehicleModal.getByLabel("Manufacturer").selectOption("Polestar");
    await expect(vehicleModal.getByLabel("Username")).toBeVisible();
    await expect(vehicleModal.getByLabel("Password")).toBeVisible();
    await expect(vehicleModal.getByLabel("Cache optional")).not.toBeVisible();
    await expect(vehicleModal.getByLabel("Default mode")).not.toBeVisible();

    await page.getByRole("button", { name: "Show advanced settings" }).click();
    await expect(vehicleModal.getByLabel("Cache optional")).toBeVisible();
    await expect(vehicleModal.getByLabel("Default mode")).toBeVisible();
  });

  test("save and restore rfid identifiers", async ({ page }) => {
    await start();

    await page.goto("/#/config");
    await enableExperimental(page);

    await page.getByTestId("add-vehicle").click();
    const vehicleModal = page.getByTestId("vehicle-modal");

    // generic
    await expectModalVisible(vehicleModal);
    await vehicleModal.getByLabel("Manufacturer").selectOption(GENERIC_VEHICLE);
    await vehicleModal.getByLabel("Title").fill("RFID Car");
    await page.getByRole("button", { name: "Show advanced settings" }).click();
    await vehicleModal.getByLabel("RFID identifiers").fill("aaa\nbbb \n ccc\n\nddd\n");
    await vehicleModal.getByRole("button", { name: "Validate & save" }).click();
    await expectModalHidden(vehicleModal);
    await expect(page.getByTestId("restart-needed")).toBeVisible();

    // restart evcc
    await restart();
    await page.reload();

    await expect(page.getByTestId("vehicle")).toHaveCount(1);
    await page.getByTestId("vehicle").getByRole("button", { name: "edit" }).click();
    await expectModalVisible(vehicleModal);
    await vehicleModal.getByRole("button", { name: "Show advanced settings" }).click();
    await expect(vehicleModal.getByLabel("RFID identifiers")).toHaveValue("aaa\nbbb\nccc\nddd");
    await vehicleModal.getByLabel("Close").click();
    await expectModalHidden(vehicleModal);
    await expect(page.getByTestId("fatal-error")).not.toBeVisible();
  });

  test("user-defined vehicle", async ({ page }) => {
    await start();

    await page.goto("/#/config");
    await enableExperimental(page, false);

    await page.getByTestId("add-vehicle").click();
    const modal = page.getByTestId("vehicle-modal");
    await expectModalVisible(modal);

    await modal.getByLabel("Manufacturer").selectOption("User-defined device");
    await page.waitForLoadState("networkidle");
    const editor = modal.getByTestId("yaml-editor");
    await expect(editor).toContainText("title: green Honda");

    await editorClear(editor);
    await editorPaste(
      editor,
      page,
      `title: blue Honda
capacity: 12.3
soc:
  source: const
  value: 42`
    );

    const restResult = modal.getByTestId("test-result");
    await expect(restResult).toContainText("Status: unknown");
    await restResult.getByRole("link", { name: "validate" }).click();
    await expect(restResult).toContainText("Status: successful");
    await expect(restResult).toContainText(["Capacity", "12.3 kWh"].join(""));
    await expect(restResult).toContainText(["Charge", "42.0%"].join(""));

    // create
    await modal.getByRole("button", { name: "Save" }).click();
    await expectModalHidden(modal);
    await expect(page.getByTestId("vehicle")).toHaveCount(1);

    // restart evcc
    await restart();
    await page.reload();

    await expect(page.getByTestId("vehicle")).toHaveCount(1);
    await expect(page.getByTestId("vehicle").nth(0)).toContainText("blue Honda");
    await page.getByTestId("vehicle").getByRole("button", { name: "edit" }).click();
    await expectModalVisible(modal);
    await expect(modal.getByLabel("Manufacturer")).toHaveValue("User-defined device");
    await page.waitForLoadState("networkidle");
    await expect(editor).toContainText("title: blue Honda");

    // update
    await editorClear(editor);
    await editorPaste(
      editor,
      page,
      `title: pink Honda
capacity: 23.4
soc:
  source: const
  value: 32`
    );
    await expect(restResult).toContainText("Status: unknown");
    await restResult.getByRole("link", { name: "validate" }).click();
    await expect(restResult).toContainText("Status: successful");
    await expect(restResult).toContainText(["Capacity", "23.4 kWh"].join(""));
    await expect(restResult).toContainText(["Charge", "32.0%"].join(""));
    await modal.getByRole("button", { name: "Save" }).click();
    await expectModalHidden(modal);
    await expect(page.getByTestId("vehicle")).toHaveCount(1);
    await expect(page.getByTestId("vehicle").nth(0)).toContainText("pink Honda");

    // delete
    await page.getByTestId("vehicle").getByRole("button", { name: "edit" }).click();
    await expectModalVisible(modal);
    await expect(editor).toContainText("title: pink Honda");
    await modal.getByRole("button", { name: "Delete" }).click();
    await expectModalHidden(modal);
    await expect(page.getByTestId("vehicle")).toHaveCount(0);

    // restart evcc
    await restart();
    await page.reload();

    await expect(page.getByTestId("vehicle")).toHaveCount(0);
  });
});
