import { test, expect } from "@playwright/test";
import { start, stop, restart, baseUrl } from "./evcc";
import { enableExperimental, expectModalVisible, expectModalHidden } from "./utils";
import { simulatorUrl, startSimulator, stopSimulator } from "./simulator";

test.use({ baseURL: baseUrl() });

const templateFlags = [
  "--disable-auth",
  "--template-type",
  "meter",
  "--template",
  "tests/config-device-auth-demo.tpl.yaml",
];

test.beforeEach(async () => {
  await startSimulator();
  await start(undefined, undefined, templateFlags);
});
test.afterEach(async () => {
  await stop();
  await stopSimulator();
});

test.describe("config device auth", async () => {
  test("create grid meter with redirect auth", async ({ page }) => {
    await page.goto("/#/config");
    await enableExperimental(page, true);

    // verify no grid meter exists yet
    await expect(page.getByTestId("grid")).toHaveCount(0);
    await expect(page.getByRole("button", { name: "Add grid meter" })).toBeVisible();

    // create a grid meter with auth
    await page.getByRole("button", { name: "Add grid meter" }).click();
    const meterModal = page.getByTestId("meter-modal");
    await expectModalVisible(meterModal);
    await meterModal.getByLabel("Manufacturer").selectOption("Auth Demo Meter");

    // step 1: auth view
    await expect(meterModal.getByLabel("Server")).toBeVisible();
    await expect(meterModal.getByLabel("Redirect URI")).toBeVisible();
    await expect(meterModal.getByLabel("Authentication Method")).toBeVisible();
    await expect(meterModal.getByLabel("Power")).not.toBeVisible();
    await expect(meterModal.getByRole("button", { name: "Validate & save" })).not.toBeVisible();
    await expect(meterModal.getByRole("button", { name: "Save" })).not.toBeVisible();
    await meterModal.getByLabel("Server").fill(simulatorUrl());
    await meterModal.getByLabel("Redirect URI").fill(page.url());
    await meterModal.getByLabel("Authentication Method").selectOption("redirect");
    await meterModal.getByRole("button", { name: "Prepare connection" }).click();

    // Get the login link and remove target="_blank" to keep it in same page
    const loginLink = meterModal.getByRole("link", { name: /Connect/ });
    await expect(loginLink).toBeVisible();
    await loginLink.evaluate((el) => el.removeAttribute("target"));
    await loginLink.click();

    // Wait for navigation to mock login page
    await page.waitForLoadState("networkidle");

    // Click login button on mock page - use evaluate to ensure JS executes
    const loginButton = page.getByRole("button", { name: /Login Successfully/ });
    await expect(loginButton).toBeVisible();
    await loginButton.evaluate((btn: HTMLButtonElement) => btn.click());

    // Wait for redirect back to config page
    await page.waitForURL(/.*\/#\/config.*/);
    await page.waitForTimeout(1000);

    // After successful auth, reopen the meter modal to continue configuration
    await page.getByRole("button", { name: "Add grid meter" }).click();
    await expectModalVisible(meterModal);
    await meterModal.getByLabel("Manufacturer").selectOption("Auth Demo Meter");

    // step 2: show regular device form - auth is complete, fill in server details again
    await meterModal.getByLabel("Server").fill(simulatorUrl());
    await meterModal.getByLabel("Redirect URI").fill(page.url());
    await meterModal.getByLabel("Authentication Method").selectOption("redirect");

    // Even though auth is already done, still need to click prepare connection to proceed to device fields
    await meterModal.getByRole("button", { name: "Prepare connection" }).click();
    await page.waitForTimeout(500);

    // Now the Power field should be visible since auth is already complete
    await expect(meterModal.getByLabel("Power")).toBeVisible();
    await meterModal.getByLabel("Power").fill("5000");
    await expect(meterModal.getByRole("button", { name: "Validate & save" })).toBeVisible();
    await meterModal.getByRole("link", { name: "validate" }).click();
    await expect(meterModal.getByTestId("device-tag-power")).toContainText("5.0 kW");
    await meterModal.getByRole("button", { name: "Save" }).click();
    await expectModalHidden(meterModal);

    // verify meter creation
    await expect(page.getByTestId("grid")).toBeVisible();
    await expect(page.getByTestId("grid")).toContainText("Grid meter");
    await expect(page.getByTestId("grid")).toContainText(["Power", "5.0 kW"].join(""));

    // re-open meter for editing
    await page.getByTestId("grid").getByRole("button", { name: "edit" }).click();
    await expectModalVisible(meterModal);
    await expect(meterModal.getByLabel("Server")).toHaveValue(simulatorUrl());
    await expect(meterModal.getByLabel("Authentication Method")).toHaveValue("redirect");
    await expect(meterModal.getByLabel("Power")).toHaveValue("5000");
    await expect(meterModal.getByRole("button", { name: "Prepare connection" })).not.toBeVisible();
    await expect(meterModal.getByRole("button", { name: "Validate & save" })).toBeVisible();
    await meterModal.getByRole("button", { name: "Close" }).click();
    await expectModalHidden(meterModal);

    // restart evcc (demo auth doesn't persist)
    await restart(undefined, templateFlags);
    await page.reload();

    // re-open meter for editing after restart, auth status has to be reestablished
    await page.getByTestId("grid").getByRole("button", { name: "edit" }).click();
    await expectModalVisible(meterModal);
    await expect(meterModal.getByLabel("Server")).toHaveValue(simulatorUrl());
    await expect(meterModal.getByLabel("Authentication Method")).toHaveValue("redirect");
    await expect(meterModal.getByLabel("Power")).not.toBeVisible();
    // note: prepare connection step is auto-executed, since all required fields are already present
    await expect(meterModal.getByRole("link", { name: /Connect/ })).toBeVisible();
    await expect(meterModal.getByRole("button", { name: "Validate & save" })).not.toBeVisible();
  });

  test("create grid meter with device-code auth", async ({ page }) => {
    await page.goto("/#/config");
    await enableExperimental(page, true);

    // create a grid meter with device-code auth
    await page.getByRole("button", { name: "Add grid meter" }).click();
    const meterModal = page.getByTestId("meter-modal");
    await expectModalVisible(meterModal);
    await meterModal.getByLabel("Manufacturer").selectOption("Auth Demo Meter");

    // select device-code method
    await meterModal.getByLabel("Server").fill(simulatorUrl());
    await meterModal.getByLabel("Redirect URI").fill(page.url());
    await meterModal.getByLabel("Authentication Method").selectOption("device-code");
    await meterModal.getByRole("button", { name: "Prepare connection" }).click();

    // verify device code is displayed
    await expect(meterModal.getByLabel("Authentication Code")).toHaveValue("12AB345");
    await expect(meterModal).toContainText("Valid for");
    await expect(meterModal).toContainText("Copy this code");
    await expect(meterModal.getByRole("link", { name: /Connect/ })).toBeVisible();
  });

  test("error server shows auth error", async ({ page }) => {
    await page.goto("/#/config");
    await enableExperimental(page, true);

    await page.getByRole("button", { name: "Add grid meter" }).click();
    const meterModal = page.getByTestId("meter-modal");
    await expectModalVisible(meterModal);
    await meterModal.getByLabel("Manufacturer").selectOption("Auth Demo Meter");
    await meterModal.getByLabel("Server").fill("ERROR");
    await meterModal.getByLabel("Redirect URI").fill(page.url());
    await meterModal.getByLabel("Authentication Method").selectOption("redirect");
    await meterModal.getByRole("button", { name: "Prepare connection" }).click();

    await expect(meterModal).toContainText("server not supported");
    await expect(meterModal.getByRole("button", { name: "Prepare connection" })).toBeVisible();
    await expect(meterModal.getByRole("link", { name: /Connect/ })).not.toBeVisible();
    await expect(meterModal.getByLabel("Authentication Code")).not.toBeVisible();
    await expect(meterModal.getByLabel("Power")).not.toBeVisible();

    // clear error on input change
    await meterModal.getByLabel("Server").fill(simulatorUrl());
    await expect(meterModal).not.toContainText("server not supported");
  });
});
