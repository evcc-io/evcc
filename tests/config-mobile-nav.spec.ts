import { test, expect, devices } from "@playwright/test";
import { start, stop, restart, baseUrl } from "./evcc";
import { expectModalHidden, expectModalVisible, newLoadpoint, addDemoCharger } from "./utils";

test.use({ baseURL: baseUrl(), viewport: devices["iPhone 12 Mini"].viewport });
test.describe.configure({ mode: "parallel" });

const desktop = devices["Desktop Chrome"].viewport;

test.describe("mobile section navigation", async () => {
  test.beforeAll(async () => {
    await start();
  });
  test.afterAll(async () => {
    await stop();
  });
  test("shows section list instead of long page", async ({ page }) => {
    await page.goto("/#/config");
    const nav = page.getByTestId("config-section-nav");
    await expect(nav).toBeVisible();
    await expect(nav.getByRole("button", { name: "Vehicles" })).toBeVisible();
    await expect(page.getByRole("heading", { name: "Vehicles" })).not.toBeVisible();
  });

  test("open and close section detail", async ({ page }) => {
    await page.goto("/#/config");
    await page.getByRole("button", { name: "Vehicles" }).click();

    const detail = page.getByTestId("section-detail-vehicles");
    await expect(detail).toBeVisible();
    await page.waitForURL("**/#/config#vehicles");
    await expect(detail.getByTestId("add-vehicle")).toBeVisible();

    await page.getByRole("button", { name: "Back" }).click();
    await expect(detail).not.toBeVisible();
    await page.waitForURL("**/#/config");
    await expect(page.getByTestId("config-section-nav")).toBeVisible();
  });

  test("browser back closes section detail", async ({ page }) => {
    await page.goto("/#/config");
    await page.getByRole("button", { name: "Grid" }).click();
    await expect(page.getByTestId("section-detail-grid")).toBeVisible();

    await page.goBack();
    await expect(page.getByTestId("section-detail-grid")).not.toBeVisible();
    await expect(page.getByTestId("config-section-nav")).toBeVisible();
  });

  test("deep link opens section detail directly", async ({ page }) => {
    await page.goto("/#/config#tariffs");
    const detail = page.getByTestId("section-detail-tariffs");
    await expect(detail).toBeVisible();

    // back button must not leave the app
    await page.getByRole("button", { name: "Back" }).click();
    await expect(detail).not.toBeVisible();
    await page.waitForURL("**/#/config");
  });

  test("modal keeps section detail open", async ({ page }) => {
    await page.goto("/#/config#vehicles");
    const detail = page.getByTestId("section-detail-vehicles");
    await expect(detail).toBeVisible();

    await detail.getByTestId("add-vehicle").click();
    const vehicleModal = page.getByTestId("vehicle-modal");
    await expectModalVisible(vehicleModal);
    expect(page.url()).toContain("vehicle");
    expect(page.url()).toContain("#vehicles");

    await vehicleModal.getByRole("button", { name: "Cancel" }).click();
    await expectModalHidden(vehicleModal);
    await expect(detail).toBeVisible();
  });

  test("modal deep link with section hash survives reload", async ({ page }) => {
    await page.goto("/#/config#integrations");
    const detail = page.getByTestId("section-detail-integrations");
    await expect(detail).toBeVisible();

    await detail.getByTestId("mqtt").getByRole("button", { name: "edit" }).click();
    const mqttModal = page.getByTestId("mqtt-modal");
    await expectModalVisible(mqttModal);

    await page.reload();
    await expectModalVisible(mqttModal);
    await mqttModal.getByRole("button", { name: "Cancel" }).click();
    await expectModalHidden(mqttModal);
    await expect(detail).toBeVisible();
  });

  test("desktop keeps long page and scrolls to anchor", async ({ page }) => {
    await page.setViewportSize(desktop);
    await page.goto("/#/config#tariffs");

    await expect(page.getByTestId("config-section-nav")).not.toBeVisible();
    await expect(page.getByTestId("section-detail-tariffs")).not.toBeVisible();
    const heading = page.getByRole("heading", { name: "Tariffs & forecasts" });
    await expect(heading).toBeInViewport();
  });

  test("resize switches between list and long page", async ({ page }) => {
    await page.goto("/#/config#grid");
    await expect(page.getByTestId("section-detail-grid")).toBeVisible();

    await page.setViewportSize(desktop);
    await expect(page.getByTestId("section-detail-grid")).not.toBeVisible();
    // level 2: the header title briefly shows "Grid" while its swap transition runs
    await expect(page.getByRole("heading", { name: "Grid", level: 2 })).toBeVisible();

    await page.setViewportSize(devices["iPhone 12 Mini"].viewport);
    await expect(page.getByTestId("section-detail-grid")).toBeVisible();
  });
});

test.describe("section indicators", async () => {
  test.afterEach(async () => {
    await stop();
  });

  test("device counts", async ({ page }) => {
    await start("config-with-vehicle.evcc.yaml");
    await page.goto("/#/config");

    const nav = page.getByTestId("config-section-nav");
    await expect(nav.getByRole("button", { name: "Charging points & heaters" })).toContainText("1");
    await expect(nav.getByRole("button", { name: "Vehicles" })).toContainText("1");
    await expect(nav.getByRole("button", { name: "Grid" })).toContainText("1");
  });

  test("error indicator on fatal device error", async ({ page }) => {
    await start("config-invalid-references-vehicle.evcc.yaml");
    await page.goto("/#/config#loadpoints");

    // loadpoint referencing a vehicle that disappears after restart
    await newLoadpoint(page, "Garage");
    await addDemoCharger(page);
    const lpModal = page.getByTestId("loadpoint-modal");
    await lpModal.getByRole("link", { name: "Advanced configuration" }).click();
    await lpModal.getByLabel("Default vehicle").selectOption("Legacy Vehicle");
    await lpModal.getByRole("button", { name: "Save" }).click();
    await expectModalHidden(lpModal);

    await restart();
    await page.goto("/#/config");

    await expect(page.getByTestId("fatal-error")).toBeVisible();
    const row = page.getByRole("button", { name: "Charging points & heaters" });
    await expect(row.getByTestId("section-error")).toBeVisible();
  });
});
