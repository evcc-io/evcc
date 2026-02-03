import { test, expect } from "@playwright/test";
import { start, stop, baseUrl, restart } from "./evcc";
import { expectModalHidden, expectModalVisible } from "./utils";

test.use({ baseURL: baseUrl() });

test.afterEach(async () => {
  await stop();
});

test.describe("eebus", async () => {
  test("first start, auto-configured", async ({ page }) => {
    await start();
    await page.goto("/#/config");

    await page.getByTestId("eebus").getByRole("button", { name: "edit" }).click();
    const modal = page.getByTestId("eebus-modal");
    await expectModalVisible(modal);

    await expect(modal.getByLabel("SHIP-ID")).not.toBeEmpty();
    await expect(modal.getByLabel("SKI")).not.toBeEmpty();

    await page.getByRole("button", { name: "Show advanced settings" }).click();
    await expect(modal.getByLabel("Port")).toHaveValue("4712");
    await expect(modal.getByLabel("Interfaces")).toBeVisible();
    await expect(modal.getByLabel("Public certificate")).not.toBeEmpty();
    await expect(modal.getByLabel("Private key")).toHaveValue("***");
  });

  test("change values", async ({ page }) => {
    await start();
    await page.goto("/#/config");

    const eebusCard = page.getByTestId("eebus");

    await eebusCard.getByRole("button", { name: "edit" }).click();
    const modal = page.getByTestId("eebus-modal");
    await expectModalVisible(modal);

    const shipidReadonly = modal.locator("#eebus-shipid-display");
    const ski = modal.getByLabel("SKI");
    const shipidEditable = modal.locator("#eebus-shipid");
    const port = modal.getByLabel("Port");
    const interfaces = modal.getByLabel("Interfaces");
    const publicCertificate = modal.getByLabel("Public certificate");
    const privateKey = modal.getByLabel("Private key");

    await page.getByRole("button", { name: "Show advanced settings" }).click();
    await shipidEditable.fill("EVCC-1234");
    await port.fill("4321");
    await interfaces.fill("eth0");
    await publicCertificate.fill("public certificate");
    await privateKey.fill("private key");

    // validate connection
    await modal.getByRole("button", { name: "Save" }).click();
    await expectModalHidden(modal);

    // restart button appears
    const restartButton = page
      .getByTestId("bottom-banner")
      .getByRole("button", { name: "Restart" });
    await expect(restartButton).toBeVisible();

    await restart();
    await page.reload();

    await eebusCard.getByRole("button", { name: "edit" }).click();

    await page.getByRole("button", { name: "Show advanced settings" }).click();
    await expect(shipidReadonly).toHaveValue("EVCC-1234");
    await expect(ski).not.toBeVisible();
    await expect(shipidEditable).toHaveValue("EVCC-1234");
    await expect(port).toHaveValue("4321");
    await expect(interfaces).toHaveValue("eth0");
    await expect(publicCertificate).toHaveValue("public certificate");
    await expect(privateKey).toHaveValue("***");
  });
  test("delete: ski changes", async ({ page }) => {
    await start();
    await page.goto("/#/config");

    const eebusCard = page.getByTestId("eebus");

    await page.getByTestId("eebus").getByRole("button", { name: "edit" }).click();
    const modal = page.getByTestId("eebus-modal");
    await expectModalVisible(modal);

    // remember ski
    const ski = modal.getByLabel("SKI");
    const skiValue = await ski.inputValue();

    page.on("dialog", async (dialog) => await dialog.accept());
    await modal.getByRole("button", { name: "Remove" }).click();

    // restart button appears
    const restartButton = page
      .getByTestId("bottom-banner")
      .getByRole("button", { name: "Restart" });
    await expect(restartButton).toBeVisible();

    await restart();
    await page.reload();

    await eebusCard.getByRole("button", { name: "edit" }).click();

    await expect(ski).not.toHaveValue(skiValue);
  });

  test("configured in yaml", async ({ page }) => {
    await start("config-eebus.evcc.yaml");
    await page.goto("/#/config");

    await page.getByTestId("eebus").getByRole("button", { name: "edit" }).click();
    const modal = page.getByTestId("eebus-modal");
    await expectModalVisible(modal);

    // yaml configuration note is visible
    await expect(modal.getByText("Configured via evcc.yaml", { exact: false })).toBeVisible();

    // SHIP-ID readonly field is visible with configured value
    await expect(modal.getByLabel("SHIP-ID")).toBeVisible();
    await expect(modal.getByLabel("SHIP-ID")).toHaveValue("EVCC_HEMS_01");
    await expect(modal.getByLabel("SHIP-ID")).toHaveAttribute("readonly");

    // SKI readonly field is visible
    await expect(modal.getByLabel("SKI")).toBeVisible();
    await expect(modal.getByLabel("SKI")).toHaveAttribute("readonly");

    // no advanced settings button
    await expect(modal.getByRole("button", { name: "Show advanced settings" })).not.toBeVisible();

    // no action buttons
    await expect(modal.getByRole("button", { name: "Save" })).not.toBeVisible();
    await expect(modal.getByRole("button", { name: "Remove" })).not.toBeVisible();
  });
});
