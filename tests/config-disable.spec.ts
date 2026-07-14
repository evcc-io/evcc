import { test, expect, type Page, type Locator } from "@playwright/test";
import { start, stop, restart, baseUrl } from "./evcc";
import { startSimulator, stopSimulator, simulatorHost } from "./simulator";
import {
  expectModalVisible,
  expectModalHidden,
  addDemoCharger,
  newLoadpoint,
  finishLoadpoint,
  openMoreMenu,
  editorClear,
  editorPaste,
} from "./utils";

test.use({ baseURL: baseUrl() });
test.describe.configure({ mode: "parallel" });

test.afterEach(async () => {
  await stop();
});

async function expectNoFatal(page: Page) {
  await expect(page.getByTestId("fatal-error")).not.toBeVisible();
}

function autoAcceptDialogs(page: Page) {
  page.on("dialog", (dialog) => dialog.accept());
}

function disabledBadge(card: Locator) {
  return card.getByRole("button", { name: "Enable" });
}

async function createLoadpoint(page: Page, title: string) {
  await newLoadpoint(page, title);
  await addDemoCharger(page);
  // charger save instant-creates the loadpoint
  await finishLoadpoint(page);
}

async function toggleLoadpointDisable(page: Page, index: number, action: "Disable" | "Enable") {
  const target = page.getByTestId("loadpoint").nth(index);
  const lpModal = page.getByTestId("loadpoint-modal");
  await target.getByRole("button", { name: "edit" }).click();
  await expectModalVisible(lpModal);
  await lpModal.getByRole("button", { name: action }).click();
  await expectModalHidden(lpModal);
}

test.describe("disable / enable", async () => {
  test("loadpoint", async ({ page }) => {
    autoAcceptDialogs(page);
    await start();
    await page.goto("/#/config");

    // two loadpoints so disabling one keeps the site valid
    await createLoadpoint(page, "Carport");
    // reload between creations to reset modal state
    await page.reload();
    await createLoadpoint(page, "Garage");

    const target = page.getByTestId("loadpoint").nth(0);

    await toggleLoadpointDisable(page, 0, "Disable");

    // card shows disabled state
    await expect(disabledBadge(target)).toBeVisible();

    // restart, no fatal
    await restart();
    await page.reload();
    await expectNoFatal(page);
    await expect(disabledBadge(target)).toBeVisible();

    // re-enable by clicking the disabled card
    await disabledBadge(target).click();
    await expect(disabledBadge(target)).toHaveCount(0);

    // restart, no fatal
    await restart();
    await page.reload();
    await expectNoFatal(page);
    await expect(disabledBadge(target)).toHaveCount(0);
  });

  test("grid meter", async ({ page }) => {
    autoAcceptDialogs(page);
    await start();
    await page.goto("/#/config");

    const meterModal = page.getByTestId("meter-modal");

    // add grid meter
    await page.getByRole("button", { name: "Add grid meter" }).click();
    await expectModalVisible(meterModal);
    await meterModal.getByLabel("Manufacturer").selectOption("Demo meter");
    await meterModal.getByLabel("Power").fill("0");
    await meterModal.getByRole("button", { name: "Save" }).click();
    await expectModalHidden(meterModal);

    const gridCard = page.getByTestId("grid");

    // open modal, disable
    await gridCard.getByRole("button", { name: "edit" }).click();
    await expectModalVisible(meterModal);
    await expect(meterModal.getByRole("button", { name: "Disable" })).toBeVisible();
    await meterModal.getByRole("button", { name: "Disable" }).click();
    await expectModalHidden(meterModal);

    // card shows disabled state
    await expect(disabledBadge(gridCard)).toBeVisible();

    // restart, no fatal
    await restart();
    await page.reload();
    await expectNoFatal(page);
    await expect(disabledBadge(gridCard)).toBeVisible();

    // re-enable by clicking the disabled card
    await disabledBadge(gridCard).click();
    await expect(disabledBadge(gridCard)).toHaveCount(0);

    // restart, no fatal
    await restart();
    await page.reload();
    await expectNoFatal(page);
    await expect(disabledBadge(gridCard)).toHaveCount(0);
  });

  test("pv meter", async ({ page }) => {
    autoAcceptDialogs(page);
    await start();
    await page.goto("/#/config");

    const meterModal = page.getByTestId("meter-modal");

    // add pv meter
    await page.getByRole("button", { name: "Add solar or battery" }).click();
    await expectModalVisible(meterModal);
    await meterModal.getByRole("button", { name: "Add solar meter" }).click();
    await meterModal.getByLabel("Manufacturer").selectOption("Demo meter");
    await meterModal.getByLabel("Title").fill("PV");
    await meterModal.getByLabel("Power").fill("0");
    await meterModal.getByRole("button", { name: "Save" }).click();
    await expectModalHidden(meterModal);

    const pvCard = page.getByTestId("pv");

    // open modal, disable
    await pvCard.getByRole("button", { name: "edit" }).click();
    await expectModalVisible(meterModal);
    await expect(meterModal.getByRole("button", { name: "Disable" })).toBeVisible();
    await meterModal.getByRole("button", { name: "Disable" }).click();
    await expectModalHidden(meterModal);

    // card shows disabled state
    await expect(disabledBadge(pvCard)).toBeVisible();

    // restart, no fatal
    await restart();
    await page.reload();
    await expectNoFatal(page);
    await expect(disabledBadge(pvCard)).toBeVisible();

    // re-enable by clicking the disabled card
    await disabledBadge(pvCard).click();
    await expect(disabledBadge(pvCard)).toHaveCount(0);

    // restart, no fatal
    await restart();
    await page.reload();
    await expectNoFatal(page);
    await expect(disabledBadge(pvCard)).toHaveCount(0);
  });

  test("grid tariff", async ({ page }) => {
    autoAcceptDialogs(page);
    await start();
    await page.goto("/#/config");

    const tariffModal = page.getByTestId("tariff-modal");
    const gridCard = page.getByTestId("tariff-grid");

    // add grid tariff
    await page.getByRole("button", { name: "Add tariff" }).click();
    await expectModalVisible(tariffModal);
    await tariffModal.getByRole("button", { name: "Add grid import tariff" }).click();
    await tariffModal.getByLabel("Provider").selectOption("Fixed Price");
    await tariffModal.getByLabel("Price").fill("32.1");
    await tariffModal.getByRole("button", { name: "Validate & save" }).click();
    await expectModalHidden(tariffModal);

    // open modal, disable
    await gridCard.getByRole("button", { name: "edit" }).click();
    await expectModalVisible(tariffModal);
    await tariffModal.getByRole("button", { name: "Disable" }).click();
    await expectModalHidden(tariffModal);

    // card shows disabled state
    await expect(disabledBadge(gridCard)).toBeVisible();

    // restart, tariff not instantiated
    await restart();
    await page.reload();
    await expectNoFatal(page);
    await expect(disabledBadge(gridCard)).toBeVisible();
    await expect(gridCard).not.toContainText("32.1");

    // re-enable by clicking the disabled card
    await disabledBadge(gridCard).click();
    await expect(disabledBadge(gridCard)).toHaveCount(0);

    // restart, tariff active again
    await restart();
    await page.reload();
    await expectNoFatal(page);
    await expect(disabledBadge(gridCard)).toHaveCount(0);
    await expect(gridCard).toContainText(["Price", "32.1 ct"].join(""));
  });

  test("user-defined vehicle", async ({ page }) => {
    autoAcceptDialogs(page);
    await start();
    await page.goto("/#/config");

    // add user-defined vehicle
    await page.getByTestId("add-vehicle").click();
    const modal = page.getByTestId("vehicle-modal");
    await expectModalVisible(modal);
    await modal.getByLabel("Manufacturer").selectOption("User-defined device");
    await page.waitForLoadState("networkidle");
    const editor = modal.getByTestId("yaml-editor");
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
    await modal.getByRole("button", { name: "Save" }).click();
    await expectModalHidden(modal);

    const vehicleCard = page.getByTestId("vehicle");
    await expect(vehicleCard).toHaveCount(1);

    // open modal, disable
    await vehicleCard.getByRole("button", { name: "edit" }).click();
    await expectModalVisible(modal);
    await modal.getByRole("button", { name: "Disable" }).click();
    await expectModalHidden(modal);

    // card shows disabled state
    await expect(disabledBadge(vehicleCard)).toBeVisible();

    // restart, no fatal
    await restart();
    await page.reload();
    await expectNoFatal(page);
    await expect(disabledBadge(vehicleCard)).toBeVisible();

    // re-enable by clicking the disabled card
    await disabledBadge(vehicleCard).click();
    await expect(disabledBadge(vehicleCard)).toHaveCount(0);

    // restart, yaml config intact
    await restart();
    await page.reload();
    await expectNoFatal(page);
    await expect(vehicleCard).toContainText("blue Honda");
  });
});

test.describe("disabled loadpoint behavior", async () => {
  test("broken charger boots after disabling its loadpoint", async ({ page }) => {
    autoAcceptDialogs(page);
    await startSimulator();
    await start();
    await page.goto("/#/config");

    // second loadpoint so the site stays valid
    await createLoadpoint(page, "Carport");
    await page.reload();

    // loadpoint with shelly charger against simulator
    await newLoadpoint(page, "Garage");
    const lpModal = page.getByTestId("loadpoint-modal");
    await lpModal.getByRole("button", { name: "Add charger" }).click();
    const chargerModal = page.getByTestId("charger-modal");
    await expectModalVisible(chargerModal);
    await chargerModal.getByLabel("Manufacturer").selectOption("Shelly 1");
    await chargerModal.getByLabel("IP address or hostname").fill(simulatorHost());
    await chargerModal.getByRole("button", { name: "Validate & save" }).click();
    await expectModalHidden(chargerModal);
    await expectModalVisible(lpModal);
    // charger save instant-creates the loadpoint
    await finishLoadpoint(page);
    await page.waitForLoadState("networkidle");

    // break charger
    await stopSimulator();
    await restart();
    await page.reload();
    await expect(page.getByTestId("fatal-error")).toBeVisible();

    // disable loadpoint with broken charger
    await toggleLoadpointDisable(page, 1, "Disable");
    await expect(disabledBadge(page.getByTestId("loadpoint").nth(1))).toBeVisible();

    // clean boot, charger no longer instantiated
    await restart();
    await page.reload();
    await expectNoFatal(page);
    await expect(disabledBadge(page.getByTestId("loadpoint").nth(1))).toBeVisible();

    // re-enable, fatal returns
    await disabledBadge(page.getByTestId("loadpoint").nth(1)).click();
    await restart();
    await page.reload();
    await expect(page.getByTestId("fatal-error")).toBeVisible();
  });

  test("disabling a loadpoint keeps api indexes stable", async ({ page }) => {
    autoAcceptDialogs(page);
    await start();
    await page.goto("/#/config");

    await createLoadpoint(page, "Carport");
    await page.reload();
    await createLoadpoint(page, "Garage");
    await page.reload();
    await createLoadpoint(page, "Süd");

    // disable middle loadpoint
    await toggleLoadpointDisable(page, 1, "Disable");
    await restart();

    // main ui hides disabled loadpoint
    await page.goto("/");
    await expect(page.getByTestId("loadpoint")).toHaveCount(2);
    await expect(page.getByRole("heading", { name: "Garage" })).not.toBeVisible();
    await expect(page.getByTestId("loadpoint").nth(0)).toContainText("Carport");
    await expect(page.getByTestId("loadpoint").nth(1)).toContainText("Süd");

    // state keeps disabled loadpoint at its position
    const state = await (await page.request.get("/api/state")).json();
    expect(state.loadpoints).toHaveLength(3);
    expect(state.loadpoints[1].disabled).toBe(true);
    expect(state.loadpoints[1].title).toBe("Garage");

    // api index of third loadpoint does not shift
    const res = await page.request.post("/api/loadpoints/3/mode/now");
    expect(res.status()).toBe(200);
    const updated = await (await page.request.get("/api/state")).json();
    expect(updated.loadpoints[2].mode).toBe("now");
    expect(updated.loadpoints[0].mode).not.toBe("now");

    // disabled loadpoint has no api routes
    const disabledRes = await page.request.post("/api/loadpoints/2/mode/now");
    expect(disabledRes.status()).toBe(404);

    // ui settings show disabled loadpoint as inactive row
    const menu = await openMoreMenu(page);
    await menu.getByRole("button", { name: "User Interface" }).click();
    const modal = page.getByTestId("global-settings-modal");
    await expectModalVisible(modal);
    const garageRow = modal.getByRole("listitem", { name: "Draggable: Garage" });
    await expect(garageRow).toContainText("disabled");
    await expect(modal.getByRole("switch", { name: "Hide Garage" })).toHaveCount(0);
    await expect(modal.getByRole("switch", { name: "Hide Carport" })).toBeEnabled();

    // last-visible guard ignores disabled loadpoint
    await modal.getByRole("switch", { name: "Hide Carport" }).click();
    await expect(modal.getByRole("switch", { name: "Hide Süd" })).toBeDisabled();
  });

  test("editing a disabled loadpoint", async ({ page }) => {
    autoAcceptDialogs(page);
    await start();
    await page.goto("/#/config");

    await createLoadpoint(page, "Carport");
    await page.reload();
    await createLoadpoint(page, "Garage");

    await toggleLoadpointDisable(page, 1, "Disable");

    const target = page.getByTestId("loadpoint").nth(1);
    const lpModal = page.getByTestId("loadpoint-modal");

    // update title while disabled
    await target.getByRole("button", { name: "edit" }).click();
    await expectModalVisible(lpModal);
    await lpModal.getByLabel("Title").fill("Garage 2");
    await lpModal.getByRole("button", { name: "Save" }).click();
    await expectModalHidden(lpModal);
    await expect(target).toContainText("Garage 2");

    // survives restart, still disabled
    await restart();
    await page.reload();
    await expect(target).toContainText("Garage 2");
    await expect(disabledBadge(target)).toBeVisible();

    // re-enable, title applies to live loadpoint
    await disabledBadge(target).click();
    await restart();
    await page.goto("/");
    await expect(page.getByRole("heading", { name: "Garage 2" })).toBeVisible();
  });
});
