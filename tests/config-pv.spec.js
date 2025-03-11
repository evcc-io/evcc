import { test, expect } from "@playwright/test";
import { start, stop, restart, baseUrl } from "./evcc";
import { enableExperimental } from "./utils";

const CONFIG_GRID_ONLY = "config-grid-only.evcc.yaml";

test.use({ baseURL: baseUrl() });

test.beforeAll(async () => {
  await start(CONFIG_GRID_ONLY);
});
test.afterAll(async () => {
  await stop();
});

test.describe("pv meter", async () => {
  test("create, edit and remove pv meter", async ({ page }) => {
    await page.goto("/#/config");
    await enableExperimental(page);

    await expect(page.getByTestId("pv")).toHaveCount(0);

    // create #1
    await page.getByRole("button", { name: "Add solar or battery" }).click();

    const meterModal = page.getByTestId("meter-modal");
    await meterModal.getByRole("button", { name: "Add solar meter" }).click();
    await meterModal.getByLabel("Title").fill("PV North");
    await meterModal.getByLabel("Manufacturer").selectOption("Demo meter");
    await meterModal.getByLabel("Power (W)").fill("5000");
    await expect(meterModal.getByRole("button", { name: "Validate & save" })).toBeVisible();
    await meterModal.getByRole("link", { name: "validate" }).click();
    await expect(meterModal.getByTestId("device-tag-power")).toContainText("5.0 kW");
    await meterModal.getByRole("button", { name: "Save" }).click();
    await expect(meterModal).not.toBeVisible();
    await expect(page.getByTestId("pv")).toBeVisible(1);
    await expect(page.getByTestId("pv")).toContainText("PV North");

    // edit #1
    await page.getByTestId("pv").getByRole("button", { name: "edit" }).click();
    await expect(meterModal).toBeVisible();
    await meterModal.getByLabel("Power (W)").fill("6000");
    await meterModal.getByRole("button", { name: "Validate & save" }).click();
    await expect(meterModal).not.toBeVisible();

    const pv = page.getByTestId("pv");
    await expect(pv).toBeVisible(1);
    await expect(pv).toContainText("PV North");
    await expect(pv.getByTestId("device-tag-power")).toContainText("6.0 kW");

    // restart and check in main ui
    await restart(CONFIG_GRID_ONLY);
    await page.goto("/");
    await page.getByTestId("visualization").click();
    await expect(page.getByTestId("energyflow")).toContainText("Production6.0 kW");

    // delete #1
    await page.goto("/#/config");
    await page.getByTestId("pv").getByRole("button", { name: "edit" }).click();
    await meterModal.getByRole("button", { name: "Delete" }).click();

    await expect(page.getByTestId("pv")).toHaveCount(0);

    // restart and check again
    await restart(CONFIG_GRID_ONLY);
    await page.reload();
    await expect(page.getByTestId("pv")).toHaveCount(0);
  });
});
