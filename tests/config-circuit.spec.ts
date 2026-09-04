import { test, expect, type Page } from "@playwright/test";
import { start, stop, restart, baseUrl } from "./evcc";
import { expectModalVisible, expectModalHidden, editorClear, editorPaste } from "./utils";

const CONFIG_YAML = "config-circuit.evcc.yaml";
const CONFIG_CIRCUITS_LEGACY = "config-circuits.sql";

test.use({ baseURL: baseUrl() });

test.afterEach(async () => {
  await stop();
});

async function validateCircuitsTags(page: Page) {
  const loadpoints = page.getByTestId("loadpoint");

  await expect(loadpoints).toHaveCount(2);
  await expect(loadpoints.nth(0)).toContainText("Power1.0 kW");
  await expect(loadpoints.nth(1)).toContainText("Power1.0 kW");

  await expect(page.getByTestId("circuits")).toHaveCount(1);
  await expect(page.getByTestId("circuits")).toContainText(
    [
      "Main",
      "2.0/10.0 kW",
      "12/16 A",
      "kW",
      "A",
      "Carport 1",
      "2.0 kW",
      " ",
      "Carport 2",
      "2.0 kW",
      " ",
      "Child",
      "0/10 A",
      "A",
    ].join(""),
  );
}

test.describe("circuit", async () => {
  test("from yaml", async ({ page }) => {
    await start(CONFIG_YAML);
    await page.goto("/#/config");
    await validateCircuitsTags(page);
  });

  test("via legacy ui", async ({ page }) => {
    await start(undefined, CONFIG_CIRCUITS_LEGACY);
    await page.goto("/#/config");

    const card = page.getByTestId("circuits");
    await expect(card).toBeVisible();
    await expect(card).toContainText(["Configured", "no"].join(""));

    await card.getByRole("button", { name: "edit" }).click();
    const circuitsModal = page.getByTestId("circuits-legacy-modal");
    await expectModalVisible(circuitsModal);

    // check for new configuration notice
    await expect(circuitsModal.getByRole("alert")).toContainText(
      "New circuits configuration available",
    );
    await circuitsModal.getByRole("button", { name: "Cancel" }).click();
    await expectModalHidden(circuitsModal);

    // add missing configuration via ui to be able to validate circuit references
    for (const [loadpointName, circuitName] of [
      ["Carport 1", "[main]"],
      ["Carport 2", "[main]"],
    ]) {
      // add loadpoint
      const lpModal = page.getByTestId("loadpoint-modal");
      await page.getByRole("button", { name: "Add charging point or heater" }).click();
      await expectModalVisible(lpModal);
      await lpModal.getByRole("button", { name: "Add charging point" }).click();
      await lpModal.getByLabel("Title").fill(loadpointName);

      // add charger
      await lpModal.getByRole("button", { name: "Add charger" }).click();
      const chargerModal = page.getByTestId("charger-modal");
      await expectModalVisible(chargerModal);
      await chargerModal.getByLabel("Manufacturer").selectOption("Demo charger");
      await chargerModal.getByLabel("Charge status").selectOption("C");
      await chargerModal.getByLabel("Power").fill("1000");
      await chargerModal.getByRole("radio", { name: "Enabled: Yes" }).click();
      await chargerModal.getByRole("button", { name: "Save" }).click();
      await expectModalHidden(chargerModal);
      await expectModalVisible(lpModal);

      await lpModal.getByRole("link", { name: "Advanced configuration" }).click();
      await expect(lpModal.getByLabel("Circuit")).toBeVisible();

      // assign loadpoint to circuit
      await lpModal.getByLabel("Circuit").selectOption(circuitName);
      await lpModal.getByRole("button", { name: "Save" }).click();
      await expectModalHidden(lpModal);
    }

    // restart button appears
    const restartButton = page
      .getByTestId("bottom-banner")
      .getByRole("button", { name: "Restart" });
    await expect(restartButton).toBeVisible();

    // restart
    await restart();
    await page.reload();

    await validateCircuitsTags(page);
  });

  test("via config ui", async ({ page }) => {
    await start();
    await page.goto("/#/config");

    const meterModal = page.getByTestId("meter-modal");
    await page.getByRole("button", { name: "Add additional meter" }).click();
    await expectModalVisible(meterModal);
    await meterModal.getByLabel("Title").fill("Circuit meter");
    await meterModal.getByLabel("Manufacturer").selectOption("Demo meter");
    await meterModal.getByLabel("Power").fill("1000");
    await meterModal.getByRole("button", { name: "Save" }).click();
    await expectModalHidden(meterModal);

    const card = page.getByTestId("circuits");
    await card.getByRole("button", { name: "edit" }).click();

    const circuitsModal = page.getByTestId("circuits-modal");
    await expectModalVisible(circuitsModal);
    await circuitsModal.getByRole("button", { name: "Add main circuit" }).click();

    const circuitModal = page.getByTestId("circuit-modal");
    await expectModalVisible(circuitModal);
    await circuitModal.getByLabel("Title").fill("Main");
    await circuitModal
      .getByLabel("Circuit", { exact: true })
      .selectOption({ label: "Static circuit" });
    await circuitModal.getByLabel("Maximum power").fill("10000");
    await circuitModal.getByLabel("Meter Selection").selectOption({ label: "Dedicated meter" });
    await circuitModal.getByTestId("circuit-meter-change").click();

    const changeMeterModal = page.getByTestId("changeMeter-modal");
    await expectModalVisible(changeMeterModal);
    const meterOption = changeMeterModal.getByRole("option", { name: /Circuit meter/ });
    const meterValue = await meterOption.getAttribute("value");
    await changeMeterModal.getByRole("combobox").selectOption(meterValue!);
    await changeMeterModal.getByRole("button", { name: "Save" }).click();
    await expectModalHidden(changeMeterModal);

    await circuitModal.getByRole("button", { name: "Save" }).click();
    await expectModalHidden(circuitModal);

    const mainCircuit = circuitsModal.getByTestId("circuit-node").filter({ hasText: "Main" });
    await expect(mainCircuit).toBeVisible();
    await expect(mainCircuit).toContainText("Circuit meter");
    await mainCircuit.getByTestId("circuit-add-sub").click();
    await expectModalVisible(circuitModal);
    await expect(circuitModal.getByLabel("Parent circuit")).toHaveValue("Main");
    await circuitModal.getByLabel("Title").fill("Child");
    await circuitModal
      .getByLabel("Circuit", { exact: true })
      .selectOption({ label: "Static circuit" });
    await circuitModal.getByLabel("Maximum current").fill("10");
    await circuitModal.getByRole("button", { name: "Save" }).click();
    await expectModalHidden(circuitModal);

    await circuitsModal.getByRole("button", { name: "Close" }).last().click();
    await expectModalHidden(circuitsModal);

    for (const [loadpointName, circuitName] of [
      ["Carport 1", "Main"],
      ["Carport 2", "Main"],
    ]) {
      const lpModal = page.getByTestId("loadpoint-modal");
      await page.getByRole("button", { name: "Add charging point or heater" }).click();
      await expectModalVisible(lpModal);
      await lpModal.getByRole("button", { name: "Add charging point" }).click();
      await lpModal.getByLabel("Title").fill(loadpointName);

      await lpModal.getByRole("button", { name: "Add charger" }).click();
      const chargerModal = page.getByTestId("charger-modal");
      await expectModalVisible(chargerModal);
      await chargerModal.getByLabel("Manufacturer").selectOption("Demo charger");
      await chargerModal.getByLabel("Charge status").selectOption("C");
      await chargerModal.getByLabel("Power").fill("1000");
      await chargerModal.getByRole("radio", { name: "Enabled: Yes" }).click();
      await chargerModal.getByRole("button", { name: "Save" }).click();
      await expectModalHidden(chargerModal);
      await expectModalVisible(lpModal);

      await lpModal.getByRole("link", { name: "Advanced configuration" }).click();
      await expect(lpModal.getByLabel("Circuit")).toBeVisible();
      const circuitOption = lpModal.getByRole("option", { name: new RegExp(circuitName) });
      const circuitValue = await circuitOption.getAttribute("value");
      await lpModal.getByLabel("Circuit").selectOption(circuitValue!);
      await lpModal.getByRole("button", { name: "Save" }).click();
      await expectModalHidden(lpModal);
    }

    const restartButton = page
      .getByTestId("bottom-banner")
      .getByRole("button", { name: "Restart" });
    await expect(restartButton).toBeVisible();
    await restart();
    await page.reload();

    await expect(page.getByTestId("loadpoint")).toHaveCount(2);
    await expect(page.getByTestId("circuits")).toContainText("Main");
  });
});

test.describe("circuit test result", async () => {
  test("shows configured limits", async ({ page }) => {
    await start();
    await page.goto("/#/config");

    await page.getByTestId("circuits").getByRole("button", { name: "edit" }).click();
    const circuitsModal = page.getByTestId("circuits-modal");
    await expectModalVisible(circuitsModal);
    await circuitsModal.getByRole("button", { name: "Add main circuit" }).click();

    const circuitModal = page.getByTestId("circuit-modal");
    await expectModalVisible(circuitModal);
    await circuitModal.getByLabel("Title").fill("House");
    await circuitModal.getByLabel("Maximum current").fill("16");
    await circuitModal.getByLabel("Maximum power").fill("10000");
    await circuitModal.getByRole("link", { name: "validate" }).click();

    const testResult = circuitModal.getByTestId("test-result");
    await expect(testResult).toContainText("Status: successful");
    await expect(testResult).toContainText("Max. current16.0 A");
    await expect(testResult).toContainText("Max. power10.0 kW");
  });

  test("user-defined yaml sample", async ({ page }) => {
    await start();
    await page.goto("/#/config");

    await page.getByTestId("circuits").getByRole("button", { name: "edit" }).click();
    const circuitsModal = page.getByTestId("circuits-modal");
    await expectModalVisible(circuitsModal);
    await circuitsModal.getByRole("button", { name: "Add main circuit" }).click();

    const circuitModal = page.getByTestId("circuit-modal");
    await expectModalVisible(circuitModal);
    await circuitModal.getByLabel("Title").fill("House");
    await circuitModal
      .getByLabel("Circuit", { exact: true })
      .selectOption({ label: "User-defined circuit" });
    await expect(circuitModal.getByTestId("yaml-editor")).toContainText("maxcurrent: 63");
    await circuitModal.getByRole("link", { name: "validate" }).click();

    const testResult = circuitModal.getByTestId("test-result");
    await expect(testResult).toContainText("Status: successful");
    await expect(testResult).toContainText("Max. current63.0 A");
    await expect(testResult).toContainText("Max. power30.0 kW");
    await circuitModal.getByRole("button", { name: "Save" }).click();
    await expectModalHidden(circuitModal);

    // user-defined sub circuit keeps parent reference next to yaml, yaml parent is overruled
    await circuitsModal.getByRole("button", { name: "Add sub-circuit" }).click();
    await expectModalVisible(circuitModal);
    await expect(circuitModal.getByLabel("Parent circuit")).toHaveValue("House");
    await circuitModal.getByLabel("Title").fill("Garage");
    await circuitModal
      .getByLabel("Circuit", { exact: true })
      .selectOption({ label: "User-defined circuit" });
    const editor = circuitModal.getByTestId("yaml-editor");
    await expect(editor).toContainText("maxcurrent: 63");
    await editorClear(editor);
    await editorPaste(editor, page, "maxcurrent: 16\nparent: db:99");
    await circuitModal.getByRole("link", { name: "validate" }).click();
    await expect(testResult).toContainText("Status: successful");
    await expect(testResult).toContainText("Max. current16.0 A");
    await circuitModal.getByRole("button", { name: "Save" }).click();
    await expectModalHidden(circuitModal);
    await expect(circuitsModal).toContainText(["House", "Garage"].join(""));

    // reopen: parent survives the round trip
    await circuitsModal.getByRole("button", { name: "edit" }).nth(1).click();
    await expectModalVisible(circuitModal);
    await expect(circuitModal.getByLabel("Title")).toHaveValue("Garage");
    await expect(circuitModal.getByLabel("Parent circuit")).toHaveValue("House");
  });
});
