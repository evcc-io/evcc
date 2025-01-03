import { test, expect } from "@playwright/test";
import { start, stop, restart, baseUrl } from "./evcc";
import { enableExperimental } from "./utils";

const CONFIG = "config-empty.evcc.yaml";

test.use({ baseURL: baseUrl() });

test.beforeAll(async () => {
  await start(CONFIG, null, "");
});
test.afterAll(async () => {
  await stop();
});

const PASSWORD = "secret";

test.describe("onboarding", async () => {
  test("password, loadpoint, grid, pv", async ({ page }) => {
    await page.goto("/");

    // set admin password
    const admin = page.getByTestId("password-modal");
    await expect(admin).toBeVisible();
    await admin.getByLabel("New password").fill(PASSWORD);
    await admin.getByLabel("Repeat password").fill(PASSWORD);
    await admin.getByRole("button", { name: "Create Password" }).click();
    await expect(admin).not.toBeVisible();

    // onboarding
    await expect(page.locator("body")).toContainText("Hello aboard!");
    await page.getByRole("link", { name: "Let's start configuration" }).click();

    // login
    const login = page.getByTestId("login-modal");
    await expect(login).toBeVisible();
    await login.getByLabel("Password").fill(PASSWORD);
    await login.getByRole("button", { name: "Login" }).click();
    await expect(login).not.toBeVisible();

    // config page
    await expect(page.getByRole("heading", { name: "Configuration" })).toBeVisible();
    await enableExperimental(page);

    // create loadpoint with charger
    await page.getByTestId("add-loadpoint").click();
    const lpModal = page.getByTestId("loadpoint-modal");
    await lpModal.getByLabel("Title").fill("Solar Carport");
    await lpModal.getByRole("button", { name: "Add charger" }).click();

    const chargerModal = page.getByTestId("charger-modal");
    await chargerModal.getByLabel("Manufacturer").selectOption("Demo charger");
    await chargerModal.getByLabel("Charge status").selectOption("C");
    await chargerModal.getByLabel("Power").fill("3000");
    await chargerModal.getByRole("button", { name: "Save" }).click();
    await expect(chargerModal).not.toBeVisible();
    await lpModal.getByRole("button", { name: "Save" }).click();
    await expect(lpModal).not.toBeVisible();

    // create grid meter
    await page.getByRole("button", { name: "Add grid meter" }).click();
    const gridModal = page.getByTestId("meter-modal");
    await expect(gridModal).toBeVisible();
    await gridModal.getByLabel("Manufacturer").selectOption("Demo meter");
    await gridModal.getByLabel("Power").fill("-2000");
    await gridModal.getByRole("button", { name: "Save" }).click();
    await expect(gridModal).not.toBeVisible();

    // create pv meter
    await page.getByRole("button", { name: "Add solar or battery" }).click();
    const pvModal = page.getByTestId("meter-modal");
    await expect(gridModal).toBeVisible();
    await pvModal.getByRole("button", { name: "Add solar meter" }).click();
    await pvModal.getByLabel("Manufacturer").selectOption("Demo meter");
    await pvModal.getByLabel("Power").fill("5000");
    await pvModal.getByRole("button", { name: "Save" }).click();
    await expect(pvModal).not.toBeVisible();

    // restart
    const restartButton = await page
      .getByTestId("bottom-banner")
      .getByRole("button", { name: "Restart" });
    await expect(restartButton).toBeVisible();
    await restart(CONFIG);
    await expect(restartButton).not.toBeVisible();

    // navigate to main screen
    await page.getByTestId("home-link").click();

    // verify configuration
    await page.getByTestId("visualization").click();
    await expect(page.getByTestId("energyflow-entry-production")).toContainText("5.0 kW");
    await expect(page.getByTestId("energyflow-entry-loadpoints")).toContainText("3.0 kW");
    await expect(page.getByTestId("energyflow-entry-gridexport")).toContainText("2.0 kW");

    const loadpoint = page.getByTestId("loadpoint");
    await expect(loadpoint.getByRole("heading", { name: "Solar Carport" })).toBeVisible();
    await expect(loadpoint).toContainText("3.0 kW");
    await expect(loadpoint).toContainText("Charging…");
  });
});
