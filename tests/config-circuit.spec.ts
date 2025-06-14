import { test, expect } from "@playwright/test";
import { start, stop, restart, baseUrl } from "./evcc";
import {
  enableExperimental,
  expectModalVisible,
  expectModalHidden,
  editorClear,
  editorType,
} from "./utils";

const CONFIG_YAML = "config-circuit.evcc.yaml";
const CONFIG_EMPTY = "config-empty.evcc.yaml";

test.use({ baseURL: baseUrl() });

test.afterEach(async () => {
  await stop();
});

test.describe("circuit", async () => {
  test("from yaml", async ({ page }) => {
    await start(CONFIG_YAML);

    await page.goto("/#/config");
    await enableExperimental(page);

    await expect(page.getByTestId("loadpoint")).toHaveCount(1);
    await expect(page.getByTestId("loadpoint")).toContainText(["Power", "1.0 kW"].join(""));

    await expect(page.getByTestId("grid")).toHaveCount(1);
    await expect(page.getByTestId("grid")).toContainText(["Power", "2.1 kW"].join(""));
    await expect(page.getByTestId("grid")).toContainText(
      ["Current L1, L2, L3", "3.0 · 3.0 · 3.0 A"].join("")
    );

    await expect(page.getByTestId("circuits")).toHaveCount(1);
    await expect(page.getByTestId("circuits")).toContainText(["Power", "2.1 kW"].join(""));
    await expect(page.getByTestId("circuits")).toContainText(
      ["Current", "3.0 A / 16.0 A"].join("")
    );
  });

  test("via ui", async ({ page }) => {
    await start(CONFIG_EMPTY);

    await page.goto("/#/config");
    await enableExperimental(page);

    // add grid meter
    await page.getByRole("button", { name: "Add grid meter" }).click();
    const meterModal = page.getByTestId("meter-modal");
    await expectModalVisible(meterModal);
    await meterModal.getByLabel("Manufacturer").selectOption("Demo meter");
    await meterModal.getByLabel("Power").fill("2070");
    await page.getByRole("button", { name: "Show advanced settings" }).click();
    await meterModal.getByLabel("L1 current").fill("3");
    await meterModal.getByLabel("L2 current").fill("3");
    await meterModal.getByLabel("L3 current").fill("3");
    await meterModal.getByRole("button", { name: "Save" }).click();
    await expectModalHidden(meterModal);

    // add loadpoint and charger
    const lpModal = page.getByTestId("loadpoint-modal");
    await page.getByRole("button", { name: "Add charge point" }).click();
    await expectModalVisible(lpModal);
    await lpModal.getByLabel("Title").fill("Carport");

    // add charger
    await lpModal.getByRole("button", { name: "Add charger" }).click();
    const chargerModal = page.getByTestId("charger-modal");
    await expectModalVisible(chargerModal);
    await chargerModal.getByLabel("Manufacturer").selectOption("Demo charger");
    await chargerModal.getByLabel("Charge status").selectOption("C");
    await chargerModal.getByLabel("Power").fill("1000");
    await chargerModal.getByRole("button", { name: "Save" }).click();
    await expectModalHidden(chargerModal);
    await expectModalVisible(lpModal);

    await lpModal.getByRole("button", { name: "Save" }).click();
    await expectModalHidden(lpModal);

    // add circuit via ui as yaml input
    await page.getByTestId("circuits").getByRole("button", { name: "edit" }).click();
    const circuitsModal = page.getByTestId("circuits-modal");
    await expectModalVisible(circuitsModal);

    const editor = circuitsModal.getByTestId("yaml-editor");
    await editorClear(editor);
    await editorType(editor, [
      // prettier-ignore
      "- name: main",
      "  meter: db:1",
      "maxcurrent: 16",
      "Shift+Tab",
      "- name: house",
      "  title: House",
      "maxcurrent: 10",
      "parent: main",
      "Shift+Tab",
      "- name: garage",
      "  title: Garage",
      "maxcurrent: 8",
      "parent: main",
    ]);

    await circuitsModal.getByRole("button", { name: "Save" }).click();
    await expectModalHidden(circuitsModal);

    // restart
    const restartButton = await page
      .getByTestId("bottom-banner")
      .getByRole("button", { name: "Restart" });
    await expect(restartButton).toBeVisible();
    await restart(CONFIG_EMPTY);
    await page.reload();

    // assign loadpoint to circuit
    await page.getByTestId("loadpoint").getByRole("button", { name: "edit" }).click();
    await expectModalVisible(lpModal);
    await lpModal.getByLabel("Circuit").selectOption("Garage [garage]");
    await lpModal.getByLabel("Circuit").selectOption("House [house]");
    await lpModal.getByRole("button", { name: "Save" }).click();
    await expectModalHidden(lpModal);

    // save, restart and check values
    await expect(restartButton).toBeVisible();
    await restart(CONFIG_EMPTY);
    await page.reload();

    // verify the configuration matches the yaml test
    await expect(page.getByTestId("loadpoint")).toHaveCount(1);
    await expect(page.getByTestId("loadpoint")).toContainText(["Power", "1.0 kW"].join(""));

    await expect(page.getByTestId("grid")).toHaveCount(1);
    await expect(page.getByTestId("grid")).toContainText(["Power", "2.1 kW"].join(""));
    await expect(page.getByTestId("grid")).toContainText(
      ["Current L1, L2, L3", "3.0 · 3.0 · 3.0 A"].join("")
    );

    await expect(page.getByTestId("circuits")).toHaveCount(1);
    await expect(page.getByTestId("circuits")).toContainText(
      ["(main)", "Power", "2.1 kW", "Current", "3.0 A / 16.0 A"].join("")
    );
    await expect(page.getByTestId("circuits")).toContainText(
      ["House (house)", "Power", "1.0 kW", "Current", "6.0 A / 10.0 A"].join("")
    );
    await expect(page.getByTestId("circuits")).toContainText(
      ["Garage (garage)", "Power", "0.0 kW", "Current", "0.0 A / 8.0 A"].join("")
    );

    // assign to garage
    await page.getByTestId("loadpoint").getByRole("button", { name: "edit" }).click();
    await expectModalVisible(lpModal);
    await lpModal.getByLabel("Circuit").selectOption("Garage [garage]");
    await lpModal.getByRole("button", { name: "Save" }).click();
    await expectModalHidden(lpModal);

    // save, restart and check values
    await expect(restartButton).toBeVisible();
    await restart(CONFIG_EMPTY);
    await page.reload();

    // verify circuits
    await expect(page.getByTestId("circuits")).toHaveCount(1);
    await expect(page.getByTestId("circuits")).toContainText(
      ["(main)", "Power", "2.1 kW", "Current", "3.0 A / 16.0 A"].join("")
    );
    await expect(page.getByTestId("circuits")).toContainText(
      ["House (house)", "Power", "0.0 kW", "Current", "0.0 A / 10.0 A"].join("")
    );
    await expect(page.getByTestId("circuits")).toContainText(
      ["Garage (garage)", "Power", "1.0 kW", "Current", "6.0 A / 8.0 A"].join("")
    );
  });
});
