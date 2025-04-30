import { test, expect } from "@playwright/test";
import { start, stop, restart, baseUrl } from "./evcc";
import { enableExperimental, expectModalHidden, expectModalVisible } from "./utils";

const CONFIG_GRID_ONLY = "config-grid-only.evcc.yaml";

test.use({ baseURL: baseUrl() });

test.beforeAll(async () => {
  await start(CONFIG_GRID_ONLY);
});
test.afterAll(async () => {
  await stop();
});

test.describe("aux meter", async () => {
  test("create and remove aux meter", async ({ page }) => {
    await page.goto("/#/config");
    await enableExperimental(page);

    await expect(page.getByTestId("aux")).toHaveCount(0);

    // create
    await page.getByRole("button", { name: "Add additional meter" }).click();

    const meterModal = page.getByTestId("meter-modal");
    await meterModal.getByRole("button", { name: "Add self-regulating consumer" }).click();
    await meterModal.getByLabel("Title").fill("Water heater");
    await meterModal.getByLabel("Manufacturer").selectOption("Demo meter");
    await meterModal.getByLabel("Power (W)").fill("1200");
    await meterModal.getByRole("button", { name: "Validate & save" }).click();
    await expectModalHidden(meterModal);

    // check
    await expect(page.getByTestId("aux")).toBeVisible(1);
    await expect(page.getByTestId("aux")).toContainText("Water heater");
    await expect(page.getByTestId("aux")).toContainText("1.2 kW");

    // restart and check again
    await restart(CONFIG_GRID_ONLY);
    await page.reload();

    // recheck
    await expect(page.getByTestId("aux")).toBeVisible(1);
    await expect(page.getByTestId("aux")).toContainText("Water heater");
    await expect(page.getByTestId("aux")).toContainText("1.2 kW");

    // delete
    await page.goto("/#/config");
    await page.getByTestId("aux").getByRole("button", { name: "edit" }).click();
    await expectModalVisible(meterModal);
    await meterModal.getByRole("button", { name: "Delete" }).click();
    await expect(page.getByTestId("aux")).toHaveCount(0);

    // restart and check again
    await restart(CONFIG_GRID_ONLY);
    await page.reload();
    await expect(page.getByTestId("aux")).toHaveCount(0);
    await expect(page.getByTestId("fatal-error")).not.toBeVisible();
  });
});
