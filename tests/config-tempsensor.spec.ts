import { test, expect, type Locator } from "@playwright/test";
import { start, stop, restart, baseUrl } from "./evcc";
import {
  expectModalVisible,
  expectModalHidden,
  editorClear,
  editorPaste,
  LoadpointType,
  addDemoCharger,
  newLoadpoint,
} from "./utils";

test.use({ baseURL: baseUrl() });
test.describe.configure({ mode: "parallel" });

test.afterEach(async () => {
  await stop();
});

const addTempSensor = async (lpModal: Locator, sensorModal: Locator) => {
  await lpModal.getByRole("link", { name: "Advanced configuration" }).click();
  await lpModal.getByRole("button", { name: "Add external temperature sensor" }).click();
  await expectModalVisible(sensorModal);
  await sensorModal.getByLabel("Manufacturer").selectOption("User-defined device");
  await sensorModal.getByRole("button", { name: "Validate & save" }).click();
  await expectModalHidden(sensorModal);
  await expectModalVisible(lpModal);
  await lpModal.getByRole("button", { name: "Save" }).click();
  await expectModalHidden(lpModal);
};

test.describe("temp sensor", async () => {
  test("create, update and delete", async ({ page }) => {
    await start();
    await page.goto("/#/config");

    const lpModal = page.getByTestId("loadpoint-modal");
    const sensorModal = page.getByTestId("tempsensor-modal");
    const lpEntry = page.getByTestId("loadpoint");

    // create
    await newLoadpoint(page, "Boiler", LoadpointType.Heating);
    await addDemoCharger(page, LoadpointType.Heating);
    await lpModal.getByRole("link", { name: "Advanced configuration" }).click();
    await lpModal.getByRole("button", { name: "Add external temperature sensor" }).click();
    await expectModalVisible(sensorModal);
    await expect(
      sensorModal.getByRole("heading", { name: "Add Temperature Sensor" })
    ).toBeVisible();
    await sensorModal.getByLabel("Manufacturer").selectOption("User-defined device");
    const testResult = sensorModal.getByTestId("test-result");
    await expect(testResult).toContainText("Status: unknown");
    await testResult.getByRole("link", { name: "validate" }).click();
    await expect(testResult).toContainText("Status: successful");
    await expect(testResult).toContainText(["Temperature", "45.0°C"].join(""));
    await sensorModal.getByRole("button", { name: "Save" }).click();
    await expectModalHidden(sensorModal);
    await expectModalVisible(lpModal);
    await expect(lpModal).toContainText("User-defined device");
    await lpModal.getByRole("button", { name: "Save" }).click();
    await expectModalHidden(lpModal);

    // sensor overrides temperature, heater keeps limit
    await restart();
    await page.goto("/");
    await expect(page.getByTestId("current-soc")).toContainText("45.0°C");
    await expect(page.getByTestId("vehicle-status-limit")).toContainText("80.0°C");

    // update
    await page.goto("/#/config");
    await expect(lpEntry).toContainText(["Temperature", "45.0°C"].join(""));
    await expect(lpEntry).toContainText(["Heater limit", "80.0°C"].join(""));
    await lpEntry.getByRole("button", { name: "edit" }).click();
    await expectModalVisible(lpModal);
    await lpModal.getByText("User-defined device").click();
    await expectModalVisible(sensorModal);
    await expect(
      sensorModal.getByRole("heading", { name: "Edit Temperature Sensor" })
    ).toBeVisible();
    const editor = sensorModal.getByTestId("yaml-editor");
    await editorClear(editor);
    await editorPaste(
      editor,
      page,
      `temp:
  source: const
  value: 30`
    );
    await sensorModal.getByRole("button", { name: "Validate & save" }).click();
    await expectModalHidden(sensorModal);
    await expectModalVisible(lpModal);
    await lpModal.getByRole("button", { name: "Close" }).click();
    await expectModalHidden(lpModal);

    await restart();
    await page.goto("/");
    await expect(page.getByTestId("current-soc")).toContainText("30.0°C");

    // delete
    await page.goto("/#/config");
    await expect(lpEntry).toContainText(["Temperature", "30.0°C"].join(""));
    await lpEntry.getByRole("button", { name: "edit" }).click();
    await expectModalVisible(lpModal);
    await lpModal.getByText("User-defined device").click();
    await expectModalVisible(sensorModal);
    await sensorModal.getByRole("button", { name: "Delete" }).click();
    await expectModalHidden(sensorModal);
    await expectModalVisible(lpModal);
    await expect(
      lpModal.getByRole("button", { name: "Add external temperature sensor" })
    ).toBeVisible();
    await lpModal.getByRole("button", { name: "Save" }).click();
    await expectModalHidden(lpModal);

    await restart();
    await page.goto("/");
    await expect(page.getByTestId("current-soc")).toContainText("50.0°C");
    await page.goto("/#/config");
    await expect(lpEntry).toContainText(["Temperature", "50.0°C"].join(""));
  });

  test("delete with loadpoint", async ({ page }) => {
    await start();
    await page.goto("/#/config");

    const lpModal = page.getByTestId("loadpoint-modal");
    const sensorModal = page.getByTestId("tempsensor-modal");

    await newLoadpoint(page, "Boiler", LoadpointType.Heating);
    await addDemoCharger(page, LoadpointType.Heating);
    await addTempSensor(lpModal, sensorModal);

    await restart();
    await page.reload();
    const sensors = await page.request.get("/api/config/devices/tempsensor");
    expect(await sensors.json()).toHaveLength(1);

    await page.getByTestId("loadpoint").getByRole("button", { name: "edit" }).click();
    await expectModalVisible(lpModal);
    await lpModal.getByRole("button", { name: "Delete" }).click();
    await expectModalHidden(lpModal);
    await expect(page.getByTestId("loadpoint")).toHaveCount(0);

    const remaining = await page.request.get("/api/config/devices/tempsensor");
    expect(await remaining.json()).toBeNull();
  });

  test("hidden for charging point", async ({ page }) => {
    await start();
    await page.goto("/#/config");

    const lpModal = page.getByTestId("loadpoint-modal");
    await newLoadpoint(page, "Carport");
    await addDemoCharger(page);
    await lpModal.getByRole("link", { name: "Advanced configuration" }).click();
    await expect(lpModal.getByLabel("Default mode")).toBeVisible();
    await expect(
      lpModal.getByRole("button", { name: "Add external temperature sensor" })
    ).toHaveCount(0);
  });
});
