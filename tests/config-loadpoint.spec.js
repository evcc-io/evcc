import { test, expect } from "@playwright/test";
import { start, stop, restart, baseUrl } from "./evcc";

const CONFIG = "config-empty.evcc.yaml";

test.use({ baseURL: baseUrl() });

test.beforeAll(async () => {
  await start(CONFIG);
});
test.afterAll(async () => {
  await stop();
});

async function enableExperimental(page) {
  await page
    .getByTestId("generalconfig-experimental")
    .getByRole("button", { name: "edit" })
    .click();
  await page.getByLabel("Experimental 🧪").click();
  await page.getByRole("button", { name: "Close" }).click();
}

test.describe("loadpoint", async () => {
  test("create, update and delete", async ({ page }) => {
    await page.goto("/#/config");
    await enableExperimental(page);

    const lpModal = page.getByTestId("loadpoint-modal");
    const chargerModal = page.getByTestId("charger-modal");

    await expect(page.getByTestId("loadpoint")).toHaveCount(1);
    await page.getByRole("button", { name: "Add charge point" }).click();
    await expect(lpModal).toBeVisible();

    // new loadpoint
    await lpModal.getByLabel("Title").fill("Solar Carport");
    await lpModal.getByRole("button", { name: "Add charger" }).click();
    await expect(lpModal).not.toBeVisible();
    await expect(chargerModal).toBeVisible();

    // add charger
    await chargerModal.getByLabel("Manufacturer").selectOption("Demowallbox");
    await chargerModal.getByLabel("Charge status").selectOption("C");
    await chargerModal.getByLabel("Power").fill("11000");
    await chargerModal.getByRole("button", { name: "Save" }).click();
    await expect(chargerModal).not.toBeVisible();
    await expect(lpModal).toBeVisible();
    await expect(lpModal.getByLabel("Title")).toHaveValue("Solar Carport");

    // create loadpoint
    await lpModal.getByRole("button", { name: "Save" }).click();
    await expect(lpModal).not.toBeVisible();
    await expect(page.getByTestId("loadpoint")).toHaveCount(2);
    await expect(page.getByTestId("loadpoint").nth(1)).toContainText("Solar Carport");
    await expect(page.getByTestId("loadpoint").nth(1)).toContainText("charging");
    await expect(page.getByTestId("loadpoint").nth(1)).toContainText("11.0 kW");

    // restart button appears
    const restartButton = await page
      .getByTestId("bottom-banner")
      .getByRole("button", { name: "Restart" });
    await expect(restartButton).toBeVisible();

    // restart
    await restart(CONFIG);
    await page.reload();
    await expect(page.getByTestId("loadpoint")).toHaveCount(2);
    await expect(page.getByTestId("loadpoint").nth(1)).toContainText("Solar Carport");

    // update loadpoint title
    await page.getByTestId("loadpoint").nth(1).getByRole("button", { name: "edit" }).click();
    await expect(lpModal).toBeVisible();
    await lpModal.getByLabel("Title").fill("Solar Carport 2");
    await lpModal.getByRole("button", { name: "Save" }).click();
    await expect(lpModal).not.toBeVisible();
    await expect(page.getByTestId("loadpoint").nth(1)).toContainText("Solar Carport 2");

    // restart
    await restart(CONFIG);
    await page.reload();
    await expect(page.getByTestId("loadpoint")).toHaveCount(2);
    await expect(page.getByTestId("loadpoint").nth(1)).toContainText("Solar Carport 2");

    // update loadpoint priority
    await page.getByTestId("loadpoint").nth(1).getByRole("button", { name: "edit" }).click();
    await expect(lpModal).toBeVisible();
    await lpModal.getByTestId("loadpointParamPriority-1").click();
    await lpModal.getByRole("button", { name: "Save" }).click();
    await expect(lpModal).not.toBeVisible();

    // restart
    await restart(CONFIG);
    await page.reload();
    await expect(page.getByTestId("loadpoint")).toHaveCount(2);
    await page.getByTestId("loadpoint").nth(1).getByRole("button", { name: "edit" }).click();
    await expect(lpModal).toBeVisible();
    await expect(lpModal.getByTestId("loadpointParamPriority-1")).toHaveClass(/active/);
    await expect(lpModal.getByLabel("Title")).toHaveValue("Solar Carport 2");
    await lpModal.getByRole("button", { name: "Close" }).click();
    await expect(lpModal).not.toBeVisible();

    // delete loadpoint
    await page.getByTestId("loadpoint").nth(1).getByRole("button", { name: "edit" }).click();
    await expect(lpModal).toBeVisible();
    await lpModal.getByRole("button", { name: "Delete" }).click();
    await expect(lpModal).not.toBeVisible();
    await expect(page.getByTestId("loadpoint")).toHaveCount(1);

    // restart
    await restart(CONFIG);
    await page.reload();
    await expect(page.getByTestId("loadpoint")).toHaveCount(1);
  });
});
