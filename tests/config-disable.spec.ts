import { test, expect, type Page, type Locator } from "@playwright/test";
import { start, stop, restart, baseUrl } from "./evcc";
import { expectModalVisible, expectModalHidden, addDemoCharger, newLoadpoint } from "./utils";

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
  const lpModal = page.getByTestId("loadpoint-modal");
  await lpModal.getByRole("button", { name: "Save" }).click();
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
    const lpModal = page.getByTestId("loadpoint-modal");

    // open modal, disable
    await target.getByRole("button", { name: "edit" }).click();
    await expectModalVisible(lpModal);
    await expect(lpModal.getByRole("button", { name: "Disable" })).toBeVisible();
    await lpModal.getByRole("button", { name: "Disable" }).click();
    await expectModalHidden(lpModal);

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
});
