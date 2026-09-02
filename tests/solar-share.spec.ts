import { test, expect, type Page } from "@playwright/test";
import { start, stop, restart, baseUrl } from "./evcc";
import { expectModalVisible, expectModalHidden, newLoadpoint, addDemoCharger } from "./utils";

test.use({ baseURL: baseUrl() });
test.describe.configure({ mode: "parallel" });

test.afterEach(async () => {
  await stop();
});

async function openSettingsModal(page: Page) {
  await page.goto("/");
  await page.getByTestId("loadpoint-settings-button").nth(1).click();
  const modal = page.getByTestId("loadpoint-settings-modal");
  await expectModalVisible(modal);
  return modal;
}

test.describe("solar share", async () => {
  test("slider active without thresholds", async ({ page }) => {
    await start("basics.evcc.yaml");

    const modal = await openSettingsModal(page);
    const slider = modal.getByLabel("Solar Share");
    await expect(slider).toBeEnabled();
    await expect(slider).toHaveValue("100");
    await expect(modal.getByText("covers the minimum charging power")).toBeVisible();
  });

  test("slider disabled by yaml thresholds", async ({ page }) => {
    await start("solar-share.evcc.yaml");

    const modal = await openSettingsModal(page);
    await expect(modal.getByLabel("Solar Share")).toBeDisabled();
    await expect(modal.getByText("Remove the enable/disable thresholds")).toBeVisible();
    await expect(modal.getByRole("link", { name: "Edit configuration" })).not.toBeVisible();
  });

  test("slider disabled by configured thresholds, reset via config", async ({ page }) => {
    await start();

    // create loadpoint with custom thresholds
    await page.goto("/#/config");
    const lpModal = page.getByTestId("loadpoint-modal");
    await newLoadpoint(page, "Carport");
    await addDemoCharger(page);
    await lpModal.getByRole("link", { name: "Advanced configuration" }).click();
    await lpModal.getByTestId("loadpointSolarMode-custom").click();
    await lpModal.getByLabel("Enable grid power").fill("-500");
    await lpModal.getByLabel("Disable grid power").fill("300");
    await expect(lpModal.getByText("disabled while watt thresholds are set")).toBeVisible();
    await lpModal.getByRole("button", { name: "Save" }).click();
    await expectModalHidden(lpModal);
    await restart();

    // slider disabled, link to configuration
    const modal = await openSettingsModal(page);
    await expect(modal.getByLabel("Solar Share")).toBeDisabled();
    await modal.getByRole("link", { name: "Edit configuration" }).click();

    // remove thresholds via hint link
    await expectModalVisible(lpModal);
    await expect(lpModal.getByText("disabled while watt thresholds are set")).toBeVisible();
    await lpModal.getByRole("link", { name: "Remove thresholds" }).click();
    await expect(lpModal.getByLabel("Enable grid power")).toHaveValue("0");
    await expect(lpModal.getByLabel("Disable grid power")).toHaveValue("0");
    await expect(lpModal.getByText("disabled while watt thresholds are set")).not.toBeVisible();
    await lpModal.getByRole("button", { name: "Save" }).click();
    await expectModalHidden(lpModal);

    // slider active again
    const modal2 = await openSettingsModal(page);
    await expect(modal2.getByLabel("Solar Share")).toBeEnabled();
  });
});
