import { test, expect } from "@playwright/test";
import { start, stop, baseUrl } from "./evcc";
import { expectModalVisible } from "./utils";

const CONFIG_HOST_PATTERN = "config-host-pattern.evcc.yaml";

test.use({ baseURL: baseUrl() });

test.describe("host pattern validation", async () => {
  test.afterEach(async () => {
    await stop();
  });

  test("show error for meter with invalid host pattern on startup", async ({ page }) => {
    await start(CONFIG_HOST_PATTERN);
    await page.goto("/#/config");

    // verify fatal error is shown
    await expect(page.getByTestId("fatal-error")).toBeVisible();
    await expect(page.getByTestId("fatal-error")).toContainText("does not match required pattern");
  });

  test("reject URL with scheme in host field", async ({ page }) => {
    await start();
    await page.goto("/#/config");

    const modal = page.getByTestId("meter-modal");
    await page.getByRole("button", { name: "Add solar or battery" }).click();
    await expectModalVisible(modal);
    await modal.getByRole("button", { name: "Add solar meter" }).click();

    await modal.getByLabel("Title").fill("Test PV");
    await page.waitForLoadState("networkidle");
    await modal.getByLabel("Manufacturer").selectOption("Shelly Pro 3 EM");

    const hostInput = modal.getByLabel("IP address or hostname");
    await hostInput.fill("http://192.168.1.100");

    // Check browser invalid state
    const isValid = await hostInput.evaluate((el: HTMLInputElement) => el.checkValidity());
    expect(isValid).toBe(false);

    // Check validate status is still unknown (hasn't tried to validate yet)
    const testResult = modal.getByTestId("test-result");
    await expect(testResult).toContainText("Status: unknown");

    // Manually delete the pattern attribute to bypass client validation
    await hostInput.evaluate((el: HTMLInputElement) => el.removeAttribute("pattern"));
    await testResult.getByRole("link", { name: "validate" }).click();
    await expect(testResult).toContainText("Status: failed");
    await expect(testResult).toContainText("does not match required pattern");
  });
});
