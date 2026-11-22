import { test, expect } from "@playwright/test";
import { start, stop, restart, baseUrl } from "./evcc";
import {
  expectModalHidden,
  expectModalVisible,
  editorClear,
  editorPaste,
  newLoadpoint,
  finishLoadpoint,
  LoadpointType,
} from "./utils";

test.use({ baseURL: baseUrl() });

test.beforeAll(async () => {
  await start();
});
test.afterAll(async () => {
  await stop();
});

test.describe("demand predictor", async () => {
  test("device default, temperature forecast, persistence", async ({ page }) => {
    await page.goto("/#/config");

    const lpModal = page.getByTestId("loadpoint-modal");
    const chargerModal = page.getByTestId("charger-modal");
    const tariffModal = page.getByTestId("tariff-modal");
    const lpEntry = page.getByTestId("loadpoint");
    const predictor = lpModal.getByLabel("Profile");
    const selected = predictor.getByRole("option", { selected: true });
    const warning = lpModal.getByText("Requires a temperature forecast.");
    const addForecast = lpModal.getByRole("link", { name: "Add forecast" });
    const edit = lpEntry.getByRole("button", { name: "edit" });
    const save = lpModal.getByRole("button", { name: "Save" });

    // user-defined heater with weekday predictor
    await newLoadpoint(page, "Boiler", LoadpointType.Heating);
    await lpModal.getByRole("button", { name: "Add heating device" }).click();
    await expectModalVisible(chargerModal);
    await chargerModal.getByLabel("Manufacturer").selectOption("User-defined heating device");
    const editor = chargerModal.getByTestId("yaml-editor");
    await editorClear(editor);
    await editorPaste(
      editor,
      page,
      `status:
  source: const
  value: "C"
enabled:
  source: const
  value: true
enable:
  source: js
  script:
maxcurrent:
  source: js
  script:
features: [heating, integrateddevice]
predictor: [demandweekday]`
    );
    await chargerModal.getByRole("button", { name: "Save" }).click();
    await expectModalHidden(chargerModal);
    await expectModalVisible(lpModal);
    await finishLoadpoint(page);

    // device default preselected
    await edit.click();
    await expectModalVisible(lpModal);
    await expect(predictor).toHaveValue("");
    await expect(selected).toHaveText("Weekday");
    await expect(lpModal).toContainText("same weekday");

    // temperature needs a forecast
    await predictor.selectOption("Temperature");
    await expect(lpModal).toContainText("similar outdoor temperatures");
    await expect(warning).toBeVisible();

    // add forecast from the warning
    await addForecast.click();
    await expectModalVisible(tariffModal);
    await tariffModal.getByLabel("Provider").selectOption("Demo Temperature Forecast");
    await tariffModal.getByRole("button", { name: "Validate & save" }).click();
    await expectModalHidden(tariffModal);
    await expectModalVisible(lpModal);
    await expect(predictor).toHaveValue("temperature");
    await expect(warning).toHaveCount(0);
    await save.click();
    await expectModalHidden(lpModal);

    // survives restart
    await restart();
    await page.reload();
    await edit.click();
    await expectModalVisible(lpModal);
    await expect(predictor).toHaveValue("temperature");
    await expect(warning).toHaveCount(0);

    // back to device default
    await predictor.selectOption({ value: "" });
    await save.click();
    await expectModalHidden(lpModal);
    await edit.click();
    await expectModalVisible(lpModal);
    await expect(predictor).toHaveValue("");
    await expect(selected).toHaveText("Weekday");
  });
});
