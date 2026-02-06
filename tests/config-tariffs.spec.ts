import { test, expect, type Page, type Locator } from "@playwright/test";
import { start, stop, restart, baseUrl } from "./evcc";
import { expectModalHidden, expectModalVisible, editorClear, editorPaste } from "./utils";

const CONFIG_WITH_TARIFFS = "config-with-tariffs.evcc.yaml";
const CONFIG_TARIFFS_LEGACY = "tariffs-legacy.sql";

async function deleteTariff(
  page: Page,
  modal: Locator,
  tariffLocator: Locator,
  nth?: number
): Promise<void> {
  const target = nth !== undefined ? tariffLocator.nth(nth) : tariffLocator;
  await target.getByRole("button", { name: "edit" }).click();
  await expectModalVisible(modal);
  await modal.getByRole("button", { name: "Delete" }).click();
  await expectModalHidden(modal);
}

test.use({ baseURL: baseUrl() });
test.describe.configure({ mode: "parallel" });

test.afterEach(async () => {
  await stop();
});

test.describe("tariffs", async () => {
  test("tariffs not configured", async ({ page }) => {
    await start();
    await page.goto("/#/config");

    // New configuration section should show with "Add Tariff" button
    await expect(page.getByRole("heading", { name: "Tariffs & Forecasts" })).toBeVisible();
    await expect(page.getByRole("button", { name: "Add Tariff" })).toBeVisible();

    // Old tariff card should not be shown
    await expect(page.getByTestId("tariffs-legacy")).not.toBeVisible();
  });

  test("tariffs from yaml ui (legacy)", async ({ page }) => {
    await start(undefined, CONFIG_TARIFFS_LEGACY);
    await page.goto("/#/config");

    await page.getByTestId("tariffs-legacy").getByRole("button", { name: "edit" }).click();
    const modal = page.getByTestId("tariffs-legacy-modal");
    await expectModalVisible(modal);
    await page.waitForLoadState("networkidle");

    // check for new configuration notice
    await expect(modal.getByRole("alert")).toContainText("New tariff configuration available");

    // default content
    const editor = modal.getByTestId("yaml-editor");
    await expect(editor).toContainText("currency: EUR");

    // clear and enter invalid yaml
    await editorClear(editor);
    await editorPaste(editor, page, "foo: bar");
    await page.getByRole("button", { name: "Save" }).click();
    await expect(modal.getByTestId("error")).toContainText("invalid keys: foo");

    // clear and enter valid yaml
    await editorClear(editor);
    await editorPaste(
      editor,
      page,
      `currency: CHF
grid:
  type: fixed
  price: 0.123`
    );

    await page.getByRole("button", { name: "Save" }).click();
    await expect(modal.getByTestId("error")).not.toBeVisible();

    // modal closes
    await expectModalHidden(modal);

    // restart button appears
    const restartButton = await page
      .getByTestId("bottom-banner")
      .getByRole("button", { name: "Restart" });
    await expect(restartButton).toBeVisible();

    await restart();

    // restart done
    await expect(restartButton).not.toBeVisible();

    await expect(page.getByTestId("generalconfig-currency")).toContainText("Currency CHF");
    await expect(page.getByTestId("tariffs-legacy")).toContainText(
      ["Grid price", "12.3 ct."].join("")
    );
  });

  test("tariffs from evcc.yaml", async ({ page }) => {
    await start(CONFIG_WITH_TARIFFS);
    await page.goto("/#/config");

    await expect(page.getByTestId("generalconfig-currency")).toContainText("Currency SEK");
    await expect(page.getByTestId("tariffs-legacy")).toBeVisible();
    await expect(page.getByTestId("tariffs-legacy")).toContainText(
      [
        "Tariffs & Forecasts",
        "Grid price",
        "30.0 öre",
        "Feed-in price",
        "-10.0 öre",
        "Grid CO₂",
        "300 g",
      ].join("")
    );
  });

  test("create, verify, delete all types", async ({ page }) => {
    await start();
    await page.goto("/#/config");

    const modal = page.getByTestId("tariff-modal");
    const tariffGrid = page.getByTestId("tariff-grid");
    const tariffFeedin = page.getByTestId("tariff-feedin");
    const tariffCo2 = page.getByTestId("tariff-co2");
    const tariffSolar = page.getByTestId("tariff-solar");
    const tariffPlanner = page.getByTestId("tariff-planner");
    const addTariff = page.getByRole("button", { name: "Add tariff" });
    const addForecast = page.getByRole("button", { name: "Add forecast" });

    // modal buttons
    const addGridImport = modal.getByRole("button", { name: "Add grid import tariff" });
    const addGridExport = modal.getByRole("button", { name: "Add grid export tariff" });
    const addCo2Forecast = modal.getByRole("button", { name: "Add CO₂ forecast" });
    const addSolarForecast = modal.getByRole("button", { name: "Add solar forecast" });
    const addPlannerForecast = modal.getByRole("button", { name: "Add planner forecast" });
    const save = modal.getByRole("button", { name: "Validate & save" });

    // initial state
    await expect(addTariff).toBeVisible();
    await expect(addForecast).toBeVisible();

    // create grid tariff
    await addTariff.click();
    await expectModalVisible(modal);
    await addGridImport.click();
    await modal.getByLabel("Provider").selectOption("Fixed Price");
    await modal.getByLabel("Price").fill("32.1");
    await save.click();
    await expectModalHidden(modal);
    await expect(tariffGrid).toContainText(["Price", "32.1 ct"].join(""));
    await expect(addTariff).toBeVisible();

    // create feedin tariff
    await addTariff.click();
    await expectModalVisible(modal);
    await addGridExport.click();
    await modal.getByLabel("Provider").selectOption("Fixed Price");
    await modal.getByLabel("Price").fill("8.7");
    await save.click();
    await expectModalHidden(modal);
    await expect(tariffFeedin).toContainText(["Price", "8.7 ct"].join(""));
    await expect(addTariff).not.toBeVisible();

    // create CO₂ forecast
    await addForecast.click();
    await expectModalVisible(modal);
    await addCo2Forecast.click();
    await modal.getByLabel("Provider").selectOption("Demo CO₂ Forecast");
    await save.click();
    await expectModalHidden(modal);
    await expect(tariffCo2).toBeVisible();

    // create solar forecast #1
    await addForecast.click();
    await expectModalVisible(modal);
    await addSolarForecast.click();
    await modal.getByLabel("Title").fill("Roof South");
    await modal.getByLabel("Provider").selectOption("Demo PV Forecast");
    await save.click();
    await expectModalHidden(modal);
    await expect(tariffSolar.nth(0)).toContainText("Roof South");

    // create solar forecast #2
    await addForecast.click();
    await expectModalVisible(modal);
    await addSolarForecast.click();
    await modal.getByLabel("Title").fill("Roof West");
    await modal.getByLabel("Provider").selectOption("Demo PV Forecast");
    await save.click();
    await expectModalHidden(modal);
    await expect(tariffSolar.nth(1)).toContainText("Roof West");

    // create planner forecast
    await addForecast.click();
    await expectModalVisible(modal);
    // co2 already exists, should not be offered
    await expect(addCo2Forecast).not.toBeVisible();
    await expect(addSolarForecast).toBeVisible();
    await expect(addPlannerForecast).toBeVisible();
    await addPlannerForecast.click();
    await modal.getByLabel("Provider").selectOption("Demo Market Price");
    await save.click();
    await expectModalHidden(modal);
    await expect(tariffPlanner).toContainText("Price");

    await addForecast.click();
    await expectModalVisible(modal);
    await expect(addPlannerForecast).not.toBeVisible();

    // restart and verify persistence
    await restart();
    await page.reload();
    await expect(tariffGrid).toBeVisible();
    await expect(tariffGrid).toContainText(["Price", "32.1 ct"].join(""));
    await expect(tariffFeedin).toBeVisible();
    await expect(tariffFeedin).toContainText(["Price", "8.7 ct"].join(""));
    await expect(tariffCo2).toBeVisible();
    await expect(tariffSolar).toHaveCount(2);
    await expect(tariffSolar.nth(0)).toContainText("Roof South");
    await expect(tariffSolar.nth(1)).toContainText("Roof West");
    await expect(tariffPlanner).toBeVisible();
    await expect(tariffPlanner).toContainText("Price");

    // delete all in reverse order
    await deleteTariff(page, modal, tariffPlanner);
    await expect(tariffPlanner).toHaveCount(0);
    await deleteTariff(page, modal, tariffSolar, 1);
    await expect(tariffSolar).toHaveCount(1);
    await deleteTariff(page, modal, tariffSolar, 0);
    await expect(tariffSolar).toHaveCount(0);
    await deleteTariff(page, modal, tariffCo2);
    await expect(tariffCo2).toHaveCount(0);
    await deleteTariff(page, modal, tariffFeedin);
    await expect(tariffFeedin).toHaveCount(0);
    await expect(addTariff).toBeVisible();
    await deleteTariff(page, modal, tariffGrid);
    await expect(tariffGrid).toHaveCount(0);

    // final state: both add buttons visible again
    await expect(addTariff).toBeVisible();
    await expect(addForecast).toBeVisible();
  });

  test("currency change", async ({ page }) => {
    await start();
    await page.goto("/#/config");

    const tariffModal = page.getByTestId("tariff-modal");
    const tariffGrid = page.getByTestId("tariff-grid");

    // create grid tariff (default EUR)
    await page.getByRole("button", { name: "Add tariff" }).click();
    await expectModalVisible(tariffModal);
    await tariffModal.getByRole("button", { name: "Add grid import tariff" }).click();
    await tariffModal.getByLabel("Provider").selectOption("Fixed Price");
    await tariffModal.getByLabel("Price").fill("32.1");
    await tariffModal.getByRole("button", { name: "Validate & save" }).click();
    await expectModalHidden(tariffModal);
    await expect(tariffGrid).toContainText(["Price", "32.1 ct"].join(""));

    // change currency to NOK
    await page.getByTestId("generalconfig-currency").getByRole("button", { name: "edit" }).click();
    const currencyModal = page.getByTestId("currency-modal");
    await expectModalVisible(currencyModal);
    await currencyModal.getByLabel("Currency").selectOption("NOK");
    await expect(
      currencyModal.getByText("Example: Your charging price was 12.2 øre/kWh. You saved kr 20.20.")
    ).toBeVisible();
    await currencyModal.getByRole("button", { name: "Save" }).click();
    await expectModalHidden(currencyModal);

    // verify
    await expect(tariffGrid).toContainText(["Price", "32.1 øre"].join(""));
  });

  test("time-based tariff (zones)", async ({ page }) => {
    await start();
    await page.goto("/#/config");

    const modal = page.getByTestId("tariff-modal");
    const tariffGrid = page.getByTestId("tariff-grid");
    const addTariff = page.getByRole("button", { name: "Add tariff" });
    const addGridImport = modal.getByRole("button", { name: "Add grid import tariff" });
    const save = modal.getByRole("button", { name: "Validate & save" });

    // select time-based tariff template
    await addTariff.click();
    await expectModalVisible(modal);
    await addGridImport.click();
    await modal.getByLabel("Provider").selectOption("Time-based Tariff");
    await modal.getByLabel("Default price").fill("20");

    const addZone = modal.getByRole("button", { name: "Add zone" });
    await expect(addZone).toBeVisible();

    // create intermediate zone with complex constraints
    await addZone.click();
    await modal.getByLabel("Price", { exact: true }).fill("0");
    await modal.getByRole("button", { name: "Months" }).click();
    await modal.getByLabel("Oct").check();
    await modal.getByLabel("Nov").check();
    await modal.getByLabel("Dec").check();
    await modal.getByRole("button", { name: "Months" }).click();
    await modal.getByRole("button", { name: "Weekdays" }).click();
    await modal.getByLabel("Sun").check();
    await modal.getByRole("button", { name: "Weekdays" }).click();
    await modal.getByLabel("From", { exact: true }).fill("01:00");
    await modal.getByLabel("To", { exact: true }).fill("03:00");
    await modal.getByRole("button", { name: "Save", exact: true }).click();

    // verify and delete
    const zones = modal.getByTestId("property-zone");
    await expect(zones).toHaveCount(1);
    await expect(zones.first()).toContainText(["0.0 ct", "Sun, Oct – Dec, 01:00 – 03:00"].join(""));
    await zones.first().getByRole("button", { name: "Remove zone" }).click();
    await expect(zones).toHaveCount(0);

    // create night rate zone
    await addZone.click();
    await modal.getByLabel("Price", { exact: true }).fill("10");
    await modal.getByLabel("From", { exact: true }).fill("00:00");
    await modal.getByLabel("To", { exact: true }).fill("06:00");
    await modal.getByRole("button", { name: "Save", exact: true }).click();
    await expect(zones.first()).toContainText(["10.0 ct", "00:00 – 06:00"].join(""));

    // create peak rate zone
    await addZone.click();
    await modal.getByLabel("Price", { exact: true }).fill("30");
    await modal.getByLabel("From", { exact: true }).fill("16:00");
    await modal.getByLabel("To", { exact: true }).fill("20:00");
    await modal.getByRole("button", { name: "Save", exact: true }).click();

    // verify zones
    await expect(zones).toHaveCount(2);
    await expect(zones.nth(0)).toContainText(["10.0 ct", "00:00 – 06:00"].join(""));
    await expect(zones.nth(1)).toContainText(["30.0 ct", "16:00 – 20:00"].join(""));

    // save and verify
    await save.click();
    await expectModalHidden(modal);
    await expect(tariffGrid).toBeVisible();
    await expect(tariffGrid).toContainText(["Forecast", "10.0 ct – 30.0 ct"].join(""));
  });
});
