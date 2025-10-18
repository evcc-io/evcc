import { test, expect } from "@playwright/test";
import { start, stop, restart, baseUrl } from "./evcc";
import { expectModalVisible, expectModalHidden, enableExperimental } from "./utils";

test.use({ baseURL: baseUrl() });

test.afterEach(async () => {
  await stop();
});

const CONFIG = "basics.evcc.yaml";
const VALID_VENDOR_ID = "ABCD1234";
const VALID_DEVICE_ID = "1234567890AB";
const INVALID_VENDOR_ID = "INVALID";
const INVALID_DEVICE_ID = "NOTVALID";

test.describe("SHM", () => {
  test("configure SHM with validation and persistence", async ({ page }) => {
    await start(CONFIG);
    await page.goto("/#/config");
    await enableExperimental(page, false);

    const shmCard = page.getByTestId("shm");

    // configure SHM with IDs
    await shmCard.getByRole("button", { name: "edit" }).click();
    const modal = page.getByTestId("shm-modal");
    await expectModalVisible(modal);

    await modal.getByRole("button", { name: "Show advanced settings" }).click();

    // test vendor ID validation
    const vendor = modal.getByLabel("Vendor ID");
    await vendor.fill(INVALID_VENDOR_ID);
    await modal.getByRole("button", { name: "Save" }).click();
    expect(await vendor.evaluate((el: HTMLInputElement) => el.validity.valid)).toBe(false);
    await vendor.fill(VALID_VENDOR_ID);
    expect(await vendor.evaluate((el: HTMLInputElement) => el.validity.valid)).toBe(true);

    // test device ID validation
    const device = modal.getByLabel("Device ID");
    await device.fill(INVALID_DEVICE_ID);
    expect(await device.evaluate((el: HTMLInputElement) => el.validity.valid)).toBe(false);
    await device.fill(VALID_DEVICE_ID);
    expect(await device.evaluate((el: HTMLInputElement) => el.validity.valid)).toBe(true);

    await modal.getByRole("button", { name: "Save" }).click();
    await expectModalHidden(modal);

    // verify persistence after restart
    await restart(CONFIG);
    await page.goto("/#/config");

    await shmCard.getByRole("button", { name: "edit" }).click();
    await expectModalVisible(modal);
    await expect(vendor).toHaveValue(VALID_VENDOR_ID);
    await expect(device).toHaveValue(VALID_DEVICE_ID);

    // verify SEMP endpoint contains configured IDs
    const [sempPage] = await Promise.all([
      page.context().waitForEvent("page"),
      modal.getByTestId("semp-url").click(),
    ]);
    const xml = await sempPage.content();
    expect(xml).toContain(
      `<DeviceId>F-${VALID_VENDOR_ID}-${VALID_DEVICE_ID.toLowerCase()}-00</DeviceId>`
    );
    await sempPage.close();
  });
});
