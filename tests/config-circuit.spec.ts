import { test, expect, type Page } from "@playwright/test";
import { start, stop, restart, baseUrl } from "./evcc";
import { expectModalVisible, expectModalHidden } from "./utils";

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
    ].join("")
  );
}

test.describe("circuit", async () => {
  test("from yaml", async ({ page }) => {
    await start(CONFIG_YAML);
    await page.goto("/#/config");
    await validateCircuitsTags(page);
  });

  test("via ui", async ({ page }) => {
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
      "New circuits configuration available"
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
});
