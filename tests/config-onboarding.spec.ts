import { test, expect, type Page } from "@playwright/test";
import { start, stop, restart, baseUrl } from "./evcc";
import { expectModalHidden, expectModalVisible, finishLoadpoint, skipOnboarding } from "./utils";

test.use({ baseURL: baseUrl() });

const PASSWORD = "secret";

async function expectStep(page: Page, title: string) {
  const wizard = page.getByTestId("onboarding-modal");
  await expectModalVisible(wizard);
  await expect(wizard.getByRole("heading", { name: title })).toBeVisible();
  // modal title plus one step title, the previous step is gone once the slide finished
  await expect(wizard.getByRole("heading")).toHaveCount(1);
}

test.describe("wizard with password", async () => {
  test.beforeEach(async () => {
    await start(undefined, undefined, []);
  });
  test.afterEach(async () => {
    await stop();
  });

  test("full flow", async ({ page }) => {
    await page.goto("/");

    // redirected to config, welcome then password
    await expect(page).toHaveURL(/#\/config/);
    const wizard = page.getByTestId("onboarding-modal");
    await expectModalVisible(wizard);
    await page.keyboard.press("Escape");
    await expectModalVisible(wizard);
    await expect(wizard.getByRole("button", { name: "Skip setup" })).not.toBeVisible();
    await wizard.getByRole("button", { name: "Get started" }).click();
    await expectStep(page, "Set Administrator Password");
    await wizard.getByRole("button", { name: "Back" }).click();
    await expect(wizard.getByRole("button", { name: "Get started" })).toBeVisible();
    await wizard.getByRole("button", { name: "Get started" }).click();

    await wizard.getByLabel("New password").fill(PASSWORD);
    await wizard.getByLabel("Repeat password").fill("other");
    await wizard.getByRole("button", { name: "Create password" }).click();
    await expect(wizard.getByText("Passwords do not match")).toBeVisible();
    await wizard.getByLabel("Repeat password").fill(PASSWORD);
    await wizard.getByRole("button", { name: "Create password" }).click();

    // site
    await expectStep(page, "Your site");
    await expect(wizard).toContainText("Step 2 of 7");
    await wizard.getByLabel("Title").fill("Zuhause");
    await wizard.getByRole("button", { name: "Continue" }).click();

    // battery: none
    await expectStep(page, "Do you have a home battery?");
    await wizard.getByRole("button", { name: "No home battery" }).click();

    // solar: demo meter
    await expectStep(page, "Do you have a solar system?");
    await wizard.getByRole("button", { name: "Add solar system" }).click();
    const meterModal = page.getByTestId("meter-modal");
    await expectModalVisible(meterModal);
    await expectModalHidden(wizard);
    await meterModal.getByLabel("Title").fill("PV South");
    await meterModal.getByLabel("Manufacturer").selectOption("Demo meter");
    await meterModal.getByLabel("Power").fill("5000");
    await meterModal.getByRole("button", { name: "Save" }).click();
    await expectModalHidden(meterModal);
    await expect(wizard).toContainText("PV South");
    await expect(wizard).toContainText("5.0 kW");
    await wizard.getByRole("button", { name: "Continue" }).click();

    // grid: reuse hint inside the meter modal
    await expectStep(page, "Do you have a grid meter?");
    // skipping asks for confirmation
    await wizard.getByRole("button", { name: "No grid meter" }).click();
    await expect(wizard).toContainText("Adding one is recommended.");
    await expect(wizard.getByRole("button", { name: "Continue without" })).toBeVisible();
    await wizard.getByRole("button", { name: "Add grid meter" }).click();
    await expectModalVisible(meterModal);
    await expect(meterModal).toContainText(
      'Solar system (Demo meter) can also be used as "Grid meter".'
    );
    await meterModal.getByRole("button", { name: "Use its settings." }).click();
    await expect(meterModal.getByLabel("Power")).toHaveValue("5000");
    await meterModal.getByLabel("Power").fill("-2000");
    await meterModal.getByRole("button", { name: "Save" }).click();
    await expectModalHidden(meterModal);
    await expect(wizard).toContainText("-2.0 kW");
    await wizard.getByRole("button", { name: "Continue" }).click();

    // charging point with demo charger
    await expectStep(page, "What do you want to control?");
    await wizard.getByRole("button", { name: "Charging point", exact: true }).click();
    const lpModal = page.getByTestId("loadpoint-modal");
    await expectModalVisible(lpModal);
    await lpModal.getByLabel("Title").fill("Solar Carport");
    await lpModal.getByRole("button", { name: "Add charger" }).click();
    const chargerModal = page.getByTestId("charger-modal");
    await expectModalVisible(chargerModal);
    await chargerModal.getByLabel("Manufacturer").selectOption("Demo charger");
    await chargerModal.getByLabel("Charge status").selectOption("C");
    await chargerModal.getByLabel("Power").fill("3000");
    await chargerModal.getByRole("button", { name: "Save" }).click();
    await expectModalHidden(chargerModal);
    await expectModalVisible(lpModal);
    await finishLoadpoint(page);
    await expect(wizard).toContainText("Solar Carport");
    await wizard.getByRole("button", { name: "Continue" }).click();

    // done
    await expectStep(page, "You're ready");
    await wizard.getByRole("button", { name: "Restart" }).click();
    await restart(undefined, [], true);

    // wizard gone, checklist shows what was skipped
    await page.goto("/#/config");
    await expect(page.getByRole("heading", { name: "Configuration" })).toBeVisible();
    await expectModalHidden(wizard);
    const checklist = page.getByTestId("setup-checklist");
    await expect(checklist).toContainText("5 of 10 done");
    await expect(checklist.getByTestId("checklist-battery").getByRole("button")).toHaveCount(0);
    await expect(
      checklist.getByTestId("checklist-consumer").getByRole("button", { name: "Add" })
    ).toBeVisible();
    await expect(
      checklist.getByTestId("checklist-gridImport").getByRole("button", { name: "Add" })
    ).toBeVisible();
    await expect(
      checklist.getByTestId("checklist-gridExport").getByRole("button", { name: "Add" })
    ).toBeVisible();
    await expect(checklist.getByTestId("checklist-heater").getByRole("button")).toHaveCount(0);

    // energy flow on the main page
    await page.getByRole("link", { name: "Charge" }).click();
    await page.getByTestId("visualization").click();
    await expect(page.getByTestId("energyflow-entry-production")).toContainText("5.0 kW");
    await expect(page.getByTestId("energyflow-entry-loadpoints")).toContainText("3.0 kW");
    await expect(page.getByTestId("energyflow-entry-gridexport")).toContainText("2.0 kW");
    await expect(
      page.getByTestId("loadpoint").getByRole("heading", { name: "Solar Carport" })
    ).toBeVisible();
  });
});

test.describe("wizard", async () => {
  test.beforeEach(async () => {
    await start(undefined, undefined, ["--disable-auth"]);
  });
  test.afterEach(async () => {
    await stop();
  });

  test("reload keeps the step", async ({ page }) => {
    await page.goto("/#/config");
    const wizard = page.getByTestId("onboarding-modal");
    await expectModalVisible(wizard);
    await wizard.getByRole("button", { name: "Get started" }).click();
    await expectStep(page, "Set Administrator Password");
    await expect(wizard.getByRole("alert")).toContainText("Authentication is disabled");
    await wizard.getByRole("button", { name: "Continue" }).click();
    await expectStep(page, "Your site");
    await wizard.getByRole("button", { name: "Continue" }).click();
    await wizard.getByRole("button", { name: "No home battery" }).click();
    await expectStep(page, "Do you have a solar system?");

    await page.reload();
    await expectStep(page, "Do you have a solar system?");

    await wizard.getByRole("button", { name: "Add solar system" }).click();
    const meterModal = page.getByTestId("meter-modal");
    await expectModalVisible(meterModal);
    await page.reload();
    await expectModalVisible(meterModal);
    await expectModalHidden(wizard);
    await meterModal.getByRole("button", { name: "Close" }).click();
    await expectStep(page, "Do you have a solar system?");
  });

  test("sponsor modal inside charger branch", async ({ page }) => {
    await page.goto("/#/config?onboarding[type:control]");
    const wizard = page.getByTestId("onboarding-modal");
    await expectStep(page, "What do you want to control?");
    await wizard.getByRole("button", { name: "Charging point", exact: true }).click();

    const lpModal = page.getByTestId("loadpoint-modal");
    await expectModalVisible(lpModal);
    await lpModal.getByLabel("Title").fill("Peblar");
    await lpModal.getByRole("button", { name: "Add charger" }).click();
    const chargerModal = page.getByTestId("charger-modal");
    await expectModalVisible(chargerModal);
    await chargerModal.getByLabel("Manufacturer").selectOption({ label: "Peblar Business" });
    await expect(chargerModal.getByRole("button", { name: "Save" })).toBeDisabled();
    const testResult = chargerModal.getByTestId("test-result");
    await testResult.getByRole("link", { name: "Validate" }).click();
    await testResult.getByText("No sponsor token configured.").click();
    const sponsorModal = page.getByTestId("sponsor-modal");
    await expectModalVisible(sponsorModal);

    // unwind the stack
    await sponsorModal.getByRole("button", { name: "Close" }).click();
    await expectModalHidden(sponsorModal);
    await expectModalVisible(chargerModal);
    await chargerModal.getByRole("button", { name: "Cancel" }).click();
    await expectModalHidden(chargerModal);
    await expectModalVisible(lpModal);
    await lpModal.getByRole("button", { name: "Close" }).click();
    await expectModalHidden(lpModal);
    await expectStep(page, "What do you want to control?");
    await expect(wizard.getByRole("button", { name: "Charging point", exact: true })).toBeVisible();
  });

  test("skip, dismiss and persist", async ({ page }) => {
    await page.goto("/#/config");
    await skipOnboarding(page);

    const checklist = page.getByTestId("setup-checklist");
    await expect(checklist).toContainText("0 of 10 done");
    await checklist.getByTestId("checklist-loadpoint").getByRole("button", { name: "Add" }).click();
    const lpModal = page.getByTestId("loadpoint-modal");
    await expectModalVisible(lpModal);
    await lpModal.getByRole("button", { name: "Close" }).click();
    await expectModalHidden(lpModal);

    await checklist
      .getByTestId("checklist-battery")
      .getByRole("button", { name: "Dismiss" })
      .click();
    await expect(checklist.getByTestId("checklist-battery").getByRole("button")).toHaveCount(0);
    await expect(checklist).toContainText("1 of 10 done");

    // no solar system means no export tariff and no solar forecast
    await checklist.getByTestId("checklist-pv").getByRole("button", { name: "Dismiss" }).click();
    await expect(checklist.getByTestId("checklist-gridExport").getByRole("button")).toHaveCount(0);
    await expect(checklist.getByTestId("checklist-solar").getByRole("button")).toHaveCount(0);
    await expect(checklist).toContainText("4 of 10 done");

    // empty main page while nothing is configured
    await page.getByRole("link", { name: "Charge" }).click();
    await expect(page.getByTestId("setup-empty")).toBeVisible();
    await page.getByTestId("setup-empty").getByRole("link", { name: "Start setup" }).click();
    await expect(page.getByRole("heading", { name: "Configuration" })).toBeVisible();
    await expectModalHidden(page.getByTestId("onboarding-modal"));

    await restart();
    await page.goto("/#/config");
    await expectModalHidden(page.getByTestId("onboarding-modal"));
    await expect(checklist.getByTestId("checklist-battery").getByRole("button")).toHaveCount(0);
    await expect(checklist.getByTestId("checklist-solar").getByRole("button")).toHaveCount(0);
  });
});

test.describe("existing configuration", async () => {
  test.afterEach(async () => {
    await stop();
  });

  test("checklist without wizard", async ({ page }) => {
    await start("config-grid-only.evcc.yaml", undefined, ["--disable-auth"]);
    await page.goto("/#/config");
    await expect(page.getByRole("heading", { name: "Configuration" })).toBeVisible();
    await expectModalHidden(page.getByTestId("onboarding-modal"));
    const checklist = page.getByTestId("setup-checklist");
    await expect(checklist.getByTestId("checklist-grid").getByRole("button")).toHaveCount(0);
    await expect(checklist.getByTestId("checklist-heater").getByRole("button")).toHaveCount(0);
    await expect(checklist.getByTestId("checklist-pv")).toBeVisible();
  });

  test("disabled onboarding", async ({ page }) => {
    await start(undefined, undefined, ["--disable-auth", "--disable-onboarding"]);
    await page.goto("/#/config");
    await expect(page.getByRole("heading", { name: "Configuration" })).toBeVisible();
    await expectModalHidden(page.getByTestId("onboarding-modal"));
    await expect(page.getByTestId("setup-checklist")).not.toBeVisible();
  });

  test("meter only", async ({ page }) => {
    await start(undefined, undefined, ["--disable-auth"]);
    await page.goto("/#/config");
    await skipOnboarding(page);
    await page.getByRole("button", { name: "Add grid meter" }).click();
    const meterModal = page.getByTestId("meter-modal");
    await expectModalVisible(meterModal);
    await meterModal.getByLabel("Manufacturer").selectOption("Demo meter");
    await meterModal.getByLabel("Power").fill("1000");
    await meterModal.getByRole("button", { name: "Save" }).click();
    await expectModalHidden(meterModal);
    await restart();
    await expect(
      page.getByTestId("setup-checklist").getByTestId("checklist-grid").getByRole("button")
    ).toHaveCount(0);
    await page.getByRole("link", { name: "Charge" }).click();
    await expect(page.getByTestId("setup-empty")).not.toBeVisible();
    await expect(page.getByTestId("energyflow")).toBeVisible();
    await expect(page.getByTestId("energyflow-entry-gridimport")).toContainText("1.0 kW");
    await expect(page.getByTestId("loadpoints")).toBeEmpty();
  });
});
